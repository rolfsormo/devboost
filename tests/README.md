# Testing devboost

Most of devboost's test coverage is plain Go tests, run the normal way:

```bash
go test ./...
```

This covers the engine (`engine/`), every resource kind (`engine/kinds/`),
every module (`engine/modules/`), and the CLI (`cmd/devboost/`) — including
an end-to-end integration test (`TestSandboxedApplyPlanDoctorIdempotent`)
that builds the real binary, points `HOME` at a fresh temp directory, and
runs `plan`/`apply`/`doctor` against real tools (Homebrew, git, etc.). That
test is slow and touches real package managers, so it's skipped under
`-short`:

```bash
go test ./... -short   # fast: skips the slow end-to-end test
go test ./...          # full: includes it
```

## tests/test-install.sh

The one thing that isn't a Go test: `install.sh`, the pure-POSIX-shell
bootstrap dispatcher (the `curl | sh` entry point), which by design has no
Go involved until it execs the downloaded binary. `tests/test-install.sh`
builds the real `devboost` binary, serves it from a local HTTP server, and
runs the real `install.sh` against it — confirming platform detection
(including the Rosetta 2 case on Apple Silicon) and the fetch-verify-exec
flow work end to end, without needing a real GitHub release.

```bash
./tests/test-install.sh
```

`tests/test_common.sh` provides the shell assertion helpers
(`test_assert`, `test_assert_eq`, `test_assert_contains`, etc.) this script
uses.

## Adding tests

- New engine/kind/module/CLI behavior: a normal Go test alongside the code
  it covers (`foo.go` → `foo_test.go`), following the existing package
  conventions.
- New `install.sh` behavior: extend `tests/test-install.sh` directly.
