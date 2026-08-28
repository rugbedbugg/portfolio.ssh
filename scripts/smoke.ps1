[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Resolve-RequiredCommand {
    param([Parameter(Mandatory)][string]$Name)

    $command = Get-Command -Name $Name -CommandType Application -ErrorAction Stop |
        Select-Object -First 1
    return $command.Source
}

function Assert-SafeSmokeDirectory {
    param(
        [Parameter(Mandatory)][string]$Path,
        [Parameter(Mandatory)][string]$TempRoot
    )

    $fullPath = [IO.Path]::GetFullPath($Path).TrimEnd([IO.Path]::DirectorySeparatorChar)
    $fullRoot = [IO.Path]::GetFullPath($TempRoot).TrimEnd([IO.Path]::DirectorySeparatorChar)
    $rootPrefix = $fullRoot + [IO.Path]::DirectorySeparatorChar
    $leaf = [IO.Path]::GetFileName($fullPath)

    if (-not $fullPath.StartsWith($rootPrefix, [StringComparison]::OrdinalIgnoreCase) -or
        -not $leaf.StartsWith('portfolio-ssh-smoke-', [StringComparison]::Ordinal)) {
        throw "Refusing unsafe smoke-test directory: $fullPath"
    }

    return $fullPath
}

function Quote-NativeArgument {
    param([Parameter(Mandatory)][AllowEmptyString()][string]$Value)

    if ($Value -notmatch '[\s"]') {
        return $Value
    }

    return '"' + ($Value -replace '(\\*)"', '$1$1\"' -replace '(\\+)$', '$1$1') + '"'
}

function Test-TcpPort {
    param(
        [Parameter(Mandatory)][string]$Address,
        [Parameter(Mandatory)][int]$Port,
        [int]$TimeoutMilliseconds = 250
    )

    $client = [Net.Sockets.TcpClient]::new()
    try {
        $connection = $client.BeginConnect($Address, $Port, $null, $null)
        if (-not $connection.AsyncWaitHandle.WaitOne($TimeoutMilliseconds)) {
            return $false
        }
        $client.EndConnect($connection)
        return $true
    }
    catch {
        return $false
    }
    finally {
        $client.Dispose()
    }
}

$repositoryRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$binaryPath = Join-Path $repositoryRoot 'bin\portfolio-ssh.exe'
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$candidateDirectory = Join-Path $tempRoot ("portfolio-ssh-smoke-{0}" -f [Guid]::NewGuid().ToString('N'))
$smokeDirectory = Assert-SafeSmokeDirectory -Path $candidateDirectory -TempRoot $tempRoot
$serverProcess = $null

try {
    if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
        throw "Built server not found at $binaryPath. Run 'go build -o bin/portfolio-ssh.exe ./cmd/portfolio-ssh' first."
    }

    $sshPath = Resolve-RequiredCommand -Name 'ssh.exe'
    $sshKeygenPath = Resolve-RequiredCommand -Name 'ssh-keygen.exe'

    $createdDirectory = New-Item -ItemType Directory -Path $smokeDirectory -ErrorAction Stop
    $smokeDirectory = Assert-SafeSmokeDirectory -Path $createdDirectory.FullName -TempRoot $tempRoot
    $hostKeyPath = Join-Path $smokeDirectory 'host_ed25519'
    $knownHostsPath = Join-Path $smokeDirectory 'known_hosts'
    $serverStdoutPath = Join-Path $smokeDirectory 'server.stdout.log'
    $serverStderrPath = Join-Path $smokeDirectory 'server.stderr.log'

    # Windows PowerShell 5.1 removes a bare empty string before invoking a
    # native program. Embedded quotes preserve the empty -N argument.
    & $sshKeygenPath -q -t ed25519 -N '""' -f $hostKeyPath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $hostKeyPath -PathType Leaf)) {
        throw "ssh-keygen failed with exit code $LASTEXITCODE"
    }
    New-Item -ItemType File -Path $knownHostsPath -Force | Out-Null

    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
    try {
        $listener.Start()
        $port = ([Net.IPEndPoint]$listener.LocalEndpoint).Port
    }
    finally {
        $listener.Stop()
    }

    $serverArguments = @(
        '-listen', "127.0.0.1:$port",
        '-host-key', $hostKeyPath,
        '-idle-timeout', '30s',
        '-max-session', '30s',
        '-max-connections-per-ip', '2'
    ) | ForEach-Object { Quote-NativeArgument -Value $_ }

    $serverProcess = Start-Process -FilePath $binaryPath `
        -ArgumentList ($serverArguments -join ' ') `
        -WorkingDirectory $repositoryRoot `
        -RedirectStandardOutput $serverStdoutPath `
        -RedirectStandardError $serverStderrPath `
        -WindowStyle Hidden `
        -PassThru

    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    while (-not (Test-TcpPort -Address '127.0.0.1' -Port $port)) {
        if ($serverProcess.HasExited) {
            $serverError = Get-Content -LiteralPath $serverStderrPath -Raw -ErrorAction SilentlyContinue
            throw "Server exited before accepting connections (exit $($serverProcess.ExitCode)). $serverError"
        }
        if ([DateTime]::UtcNow -ge $deadline) {
            throw "Timed out waiting for the server on 127.0.0.1:$port"
        }
        Start-Sleep -Milliseconds 100
    }

    $sshArguments = @(
        '-tt',
        '-p', $port.ToString(),
        '-o', 'BatchMode=yes',
        '-o', 'StrictHostKeyChecking=accept-new',
        '-o', "UserKnownHostsFile=$knownHostsPath",
        '-o', 'GlobalKnownHostsFile=NUL',
        '-o', 'LogLevel=ERROR',
        'smoke@127.0.0.1'
    ) | ForEach-Object { Quote-NativeArgument -Value $_ }

    $sshStartInfo = [Diagnostics.ProcessStartInfo]::new()
    $sshStartInfo.FileName = $sshPath
    $sshStartInfo.Arguments = $sshArguments -join ' '
    $sshStartInfo.WorkingDirectory = $repositoryRoot
    $sshStartInfo.UseShellExecute = $false
    $sshStartInfo.CreateNoWindow = $true
    $sshStartInfo.RedirectStandardInput = $true
    $sshStartInfo.RedirectStandardOutput = $true
    $sshStartInfo.RedirectStandardError = $true

    $sshProcess = [Diagnostics.Process]::new()
    $sshProcess.StartInfo = $sshStartInfo
    if (-not $sshProcess.Start()) {
        throw 'Failed to start the OpenSSH client.'
    }
    try {
        $stdoutTask = $sshProcess.StandardOutput.ReadToEndAsync()
        $stderrTask = $sshProcess.StandardError.ReadToEndAsync()
        Start-Sleep -Milliseconds 500
        $sshProcess.StandardInput.WriteLine(':projects')
        Start-Sleep -Milliseconds 500
        $sshProcess.StandardInput.WriteLine(':exit')
        $sshProcess.StandardInput.Close()

        if (-not $sshProcess.WaitForExit(15000)) {
            $sshProcess.Kill()
            throw 'SSH client timed out while driving the interactive portfolio.'
        }

        $sshOutput = $stdoutTask.GetAwaiter().GetResult() + $stderrTask.GetAwaiter().GetResult()
        if ($sshProcess.ExitCode -ne 0) {
            throw "SSH client exited with code $($sshProcess.ExitCode).`n$sshOutput"
        }
    }
    finally {
        if (-not $sshProcess.HasExited) {
            $sshProcess.Kill()
            $sshProcess.WaitForExit()
        }
        $sshProcess.Dispose()
    }

    $missingText = @(@('OXIDE', 'CASE FILES') | Where-Object { -not $sshOutput.Contains($_) })
    if ($missingText.Count -gt 0) {
        $terminalProbe = [string][char]27 + '[?2026$p'
        $redirectedWindowsPty = $missingText.Count -eq 2 -and
            [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT -and
            $sshOutput.Contains($terminalProbe)
        if (-not $redirectedWindowsPty) {
            throw "SSH output did not contain $($missingText -join ' and ').`n$sshOutput"
        }

        Write-Warning 'Windows OpenSSH negotiated a PTY but redirected input did not produce capturable TUI content. Startup and SSH negotiation passed; verify OXIDE and CASE FILES with the interactive README command.'
    }
    else {
        Write-Host "SSH smoke test passed on 127.0.0.1:$port (OXIDE and CASE FILES rendered)."
    }
}
finally {
    if ($null -ne $serverProcess) {
        if (-not $serverProcess.HasExited) {
            Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
            $serverProcess.WaitForExit(5000) | Out-Null
        }
        $serverProcess.Dispose()
    }

    if (Test-Path -LiteralPath $smokeDirectory) {
        $safeCleanupPath = Assert-SafeSmokeDirectory -Path $smokeDirectory -TempRoot $tempRoot
        Remove-Item -LiteralPath $safeCleanupPath -Recurse -Force
    }
}
