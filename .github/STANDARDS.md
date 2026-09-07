# Repository standards baseline

Adapted from ReAgent's shared baseline for this Go SSH application.

CI covers main/CI branch pushes, pull requests, manual and reusable calls.
Validation has read-only permissions, obsolete-run cancellation, a timeout and
Go module/build caches. Mise selects Go 1.25.12 to match go.mod; GOTOOLCHAIN=local
prevents implicit toolchain changes and GOFLAGS=-mod=readonly preserves modules.

Run `mise install`, `mise run install`, then `mise run check`. Tasks reuse the
Makefile: gofmt validation, race-enabled tests (including existing server tests),
go vet, native build and CLI help smoke test. The tested binary and checksum are
uploaded as CI artifacts. No deployment or release automation is added.
Future CD must require validation, reuse tested artifacts, protect credentials
and verify published outputs.

README updates remain explicitly deferred to the later pass. That pass must
cover purpose, setup, quick start, usage/configuration, development checks,
license and accurate CI/release links. Validate workflow syntax and require
local checks and GitHub CI before integration.
