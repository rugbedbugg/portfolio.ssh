# Contributing to portfolio.ssh

portfolio.ssh is a Go SSH/TUI application. Preserve its per-session isolation, security boundaries, and keyboard-oriented interface.

## Build and test

The Go version is declared in `go.mod` and resolved by mise. Run the repository's complete check target:

```sh
make check
```

This runs `go test ./...`, `go vet ./...`, and the application build. Use `make run` for a local server. Windows end-to-end smoke-test commands and their redirected-PTY limitation are documented in `README.md`.

## Change guidelines

- Keep commands, session state, rendering, content, and transport concerns within their existing packages.
- Add tests for content and session behavior; do not rely only on terminal screenshots.
- Update `internal/content/content.go` and its matching tests together when changing portfolio records.
- Preserve host-key handling, connection limits, timeouts, and unprivileged deployment assumptions.
- Follow the repository's imperative commit convention when preparing commits.

## Pull requests

Include `make check` output, describe interactive verification, and call out changes to SSH negotiation, configuration, or deployment behavior.
