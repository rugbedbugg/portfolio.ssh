# portfolio.ssh

`portfolio.ssh` is a read-only personal dossier served through an interactive SSH terminal. Visitors connect with a standard SSH client and browse projects, research, dispatches, and contact links in a responsive CGA-style interface. The service exposes portfolio navigation only; it is not a shell account.

## Preview

```text
╔══════════════════════════════════════════════════════════════════╗
║  OXIDE // DOSSIER                                               ║
║                                                                  ║
║  ABOUT   > CASE FILES   RESEARCH   DISPATCHES   CONTACT          ║
║                                                                  ║
║  ReAgent                         Python · Planning · Scoring       ║
║  Trionda-Trifecta-26             Python · ML · Simulation        ║
║  ResonanceID-cli                 Rust · DSP · SQLite              ║
║  HTTP-SVR-200-OK                 x86_64 · Linux · Networking      ║
║                                                                  ║
║  j/k move   enter open   esc back   : command   q exit           ║
╚══════════════════════════════════════════════════════════════════╝
```

The exact layout adapts to the terminal width and height.

## Prerequisites

- Go 1.25.12 or newer
- GNU Make for the convenience targets
- OpenSSH `ssh` for connecting
- OpenSSH `ssh-keygen` for host-key creation and the Windows smoke test
- PowerShell 5.1 or newer to run `scripts/smoke.ps1`

Clone the repository, then run the complete local check:

```powershell
make check
```

This runs the Go tests, `go vet`, and a build. On Windows, build the filename used by the smoke test and run the end-to-end check with:

```powershell
go build -o bin/portfolio-ssh.exe ./cmd/portfolio-ssh
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke.ps1
```

The smoke test creates an isolated temporary directory, generates a disposable Ed25519 host key, starts the server on an unused loopback port, connects with `ssh -tt`, verifies the `OXIDE` and `CASE FILES` output, and removes its temporary files. Windows OpenSSH can negotiate a remote PTY yet produce no capturable TUI content when both input and output are redirected. If the script reports that limitation, its automated startup and SSH-negotiation checks still passed; connect from an interactive terminal, enter `:projects`, confirm `OXIDE` and `CASE FILES` are visible, then enter `:exit`.

## Run locally

Generate a persistent development host key once. Do not add the private key to version control.

```powershell
New-Item -ItemType Directory -Force .ssh | Out-Null
ssh-keygen -t ed25519 -f .ssh/portfolio_ed25519
```

Press `Enter` at both passphrase prompts to leave the host key unencrypted. The service starts unattended and cannot unlock a passphrase-protected host key.

Start the service with its defaults:

```powershell
make run
```

It listens on TCP port `23234` on all interfaces and reads `.ssh/portfolio_ed25519`. Connect from another terminal:

```powershell
ssh -p 23234 portfolio@localhost
```

SSH host identity is tied to the host key. Verify a new fingerprint through a trusted channel before accepting it, and remove a stale `known_hosts` entry only after confirming that the server key was intentionally rotated.

## Navigate

In navigation mode:

| Key | Action |
| --- | --- |
| `j` / down arrow | Move to the next item |
| `k` / up arrow | Move to the previous item |
| `Enter` | Open the selected section or record |
| `Esc` | Return to the section index |
| `:` | Focus the command prompt |
| `q` | Exit the session |

At the command prompt, use `Tab` for completion, up/down arrows for command history, `Esc` to leave the prompt, and `Enter` to submit.

## Commands

| Command | Action |
| --- | --- |
| `help` | Show the command summary |
| `about` | Open the profile |
| `projects` or `ls` | List project case files |
| `project <id>` | Open one project |
| `research` | List publications |
| `dispatches` | List field notes |
| `contact` | List contact links |
| `open <id>` | Open one contact record |
| `clear` | Clear status and command history |
| `exit` or `quit` | End the SSH session |

Commands run inside the portfolio interface. They are never passed to an operating-system shell.

## Configuration

Command-line flags override environment variables. Durations use Go syntax such as `30s`, `10m`, or `1h`.

| Environment variable | Flag | Default | Purpose |
| --- | --- | --- | --- |
| `PORTFOLIO_SSH_LISTEN` | `-listen` | `:23234` | TCP listen address |
| `PORTFOLIO_SSH_HOST_KEY` | `-host-key` | `.ssh/portfolio_ed25519` | Ed25519 private host-key path |
| `PORTFOLIO_SSH_IDLE_TIMEOUT` | `-idle-timeout` | `10m` | Disconnect an idle client |
| `PORTFOLIO_SSH_MAX_SESSION` | `-max-session` | `1h` | Maximum session lifetime |
| `PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP` | `-max-connections-per-ip` | `5` | Simultaneous sessions allowed per source IP |

For example:

```powershell
$env:PORTFOLIO_SSH_LISTEN = '127.0.0.1:23234'
$env:PORTFOLIO_SSH_HOST_KEY = 'D:\secrets\portfolio_ed25519'
$env:PORTFOLIO_SSH_IDLE_TIMEOUT = '5m'
$env:PORTFOLIO_SSH_MAX_SESSION = '30m'
$env:PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP = '3'
.\bin\portfolio-ssh.exe
```

## Deploy with systemd

Build for Linux, create a dedicated account with no login shell, and keep the service on an unprivileged high port:

```sh
go build -trimpath -o portfolio-ssh ./cmd/portfolio-ssh
sudo useradd --system --home-dir /var/lib/portfolio-ssh --create-home --shell /usr/sbin/nologin portfolio-ssh
sudo install -o root -g root -m 0755 portfolio-ssh /usr/local/bin/portfolio-ssh
sudo install -d -o portfolio-ssh -g portfolio-ssh -m 0700 /var/lib/portfolio-ssh/.ssh
sudo -u portfolio-ssh ssh-keygen -q -t ed25519 -N '' -f /var/lib/portfolio-ssh/.ssh/portfolio_ed25519
```

Create `/etc/systemd/system/portfolio-ssh.service`:

```ini
[Unit]
Description=Interactive SSH portfolio
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=portfolio-ssh
Group=portfolio-ssh
WorkingDirectory=/var/lib/portfolio-ssh
Environment=PORTFOLIO_SSH_LISTEN=:23234
Environment=PORTFOLIO_SSH_HOST_KEY=/var/lib/portfolio-ssh/.ssh/portfolio_ed25519
Environment=PORTFOLIO_SSH_IDLE_TIMEOUT=10m
Environment=PORTFOLIO_SSH_MAX_SESSION=1h
Environment=PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP=5
ExecStart=/usr/local/bin/portfolio-ssh
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/portfolio-ssh

[Install]
WantedBy=multi-user.target
```

Then enable it:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now portfolio-ssh.service
sudo systemctl status portfolio-ssh.service
```

Point a DNS `A` record (and `AAAA` when IPv6 is configured) at the server. Permit the chosen TCP port through the host and provider firewalls, and publish the port in the visitor command, for example `ssh -p 23234 portfolio@example.com`. An HTTP reverse proxy cannot proxy SSH; use direct TCP exposure or a layer-4 TCP proxy. Port `22` normally requires elevated binding privileges and may conflict with the host's administrative SSH daemon, so the example keeps `23234` and the non-root service account.

## Security boundaries

- The endpoint is intentionally public: it does not check visitor credentials or grant a user account. A client must request an interactive PTY; non-PTY and executable-command requests are rejected.
- Only SSH `session` channels and the fixed portfolio command language are enabled. Subsystems, local/reverse forwarding, and agent forwarding are disabled.
- Idle timeouts, maximum session durations, and per-source-IP connection limits bound basic resource use. They are not a substitute for network-level rate limiting or denial-of-service protection.
- The host key is the server's long-lived identity and must remain private, readable only by the service account, backed up securely, and rotated deliberately.
- Portfolio URLs are displayed as content. The service does not execute them or run visitor-supplied shell commands.
- Run the binary as a dedicated non-root user, expose only the selected TCP port, keep dependencies and the operating system patched, and monitor service logs.

## Update portfolio content

Edit the typed records in `internal/content/content.go`, update the matching tests in `internal/content/content_test.go`, then run `make check`. Rebuild and restart the deployed service to publish the new immutable content.

## Commit convention

Use a bracketed category followed by a concise imperative summary:

```text
[Feature]: Add a portfolio section
[Fix]: Reject an invalid configuration
[Docs]: Clarify the deployment guide
[Test]: Cover command completion
```
