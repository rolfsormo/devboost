# devboost Architecture

## Overview

devboost is a Go program built as a **Terraform-inspired typed-resource
engine**: every module describes desired state as a list of typed
`engine.Resource` values, and one shared function (`engine.ComputeDiff`)
computes what's out of sync. `plan` and `apply` both call that same
function — `plan` prints the result and stops, `apply` prints and then
executes it — so there is no separate hand-written plan narration to
drift out of sync with what apply actually does.

This replaced an earlier bash implementation (still visible in git
history) where every module hand-wrote a `plan` function and an `apply`
function separately, describing the same change twice in two different
places. That duplication was a real, hit-in-production bug class — see
the [CHANGELOG](CHANGELOG.md) for the double-sourcing bug this surfaced.
`ComputeDiff` closes that bug class structurally: there is only one place
"what needs to change" is computed, for both commands.

## Directory Structure

```
devboost/
  cmd/devboost/           # CLI entry point (main.go)

  engine/                 # Core engine — no knowledge of specific modules
    resource.go           # Resource, ResourceKind, PendingOp, ComputeDiff, DiffAndExecute
    plan.go                # Plan(): diff, print, stop
    apply.go                # Apply(): diff-and-execute one resource at a time, in dependency order
    doctor.go                # Doctor(): diffs the combined graph once, groups results by module
    diagnostic.go             # Diagnostic/DiagnosticFunc: read-only findings with nothing to converge

    kinds/                  # Resource kind implementations — the "providers"
      directory.go           # DirExists
      gitclone.go              # GitClone
      file.go                   # File (full-content, backed up before overwrite)
      gitconfig.go               # GitConfig (shells out to `git config`)
      blockinfile.go               # BlockInFile / RemoveBlock (managed block between markers)
      lineinfile.go                  # LineInFile (the dedup modules' comment-out mechanism)
      package.go                      # Package (per-OS package manager abstraction)
      commandguarded.go                 # CommandGuarded (the one deliberate escape hatch)
      backup.go                          # Shared backup/snapshot/archive helpers
      os.go                                # OS detection

    modules/                 # Module ports — the actual opinionated defaults
      registry.go              # Module registry: All, AllResources
      znap.go, starship.go, tmux.go, mise.go, pkg.go, git.go, corepack.go,
      direnv.go, services.go, security.go, zsh*.go, *_dedup.go, uninstall.go,
      migrateohmyzsh.go, clean.go

  config/                  # ~/.devboost.yaml reader (config.Config)
```

## The Resource Model

```go
// A resource kind knows how to diff itself against live system state.
type ResourceKind interface {
    Diff() (*PendingOp, error)
}

// What a module declares: an ID, a kind, and optional dependencies.
type Resource struct {
    ID        string
    Kind      ResourceKind
    DependsOn []string
}

// A pending change — never authored directly, always computed by Diff().
type PendingOp struct {
    ResourceID  string
    Description string
    Execute     func() error
}
```

`ComputeDiff` topologically sorts resources by `DependsOn` and calls
`Diff()` on each once — used by `plan` and `doctor`, where nothing has
converged yet so a single batch pass is safe. `DiffAndExecute` interleaves
diff-then-immediately-execute per resource in dependency order — required
by `apply`, because a dependent resource must see the *real* post-execution
state of what it depends on, not a stale batch diff.

## Module Interface

A module is just a plain Go function returning a resource list — no
interface to implement, no registration boilerplate beyond adding one
line to the registry:

```go
// engine/modules/foo.go
package modules

func Foo(cfg *config.Config) []engine.Resource {
    if cfg.Get("foo.enable", "true") != "true" {
        return nil
    }
    return []engine.Resource{
        {ID: "foo_config", Kind: kinds.File{Path: cfg.Get("foo.path", "~/.foorc"), Content: renderFooConfig(cfg)}},
    }
}
```

```go
// engine/modules/registry.go
var All = []Module{
    // ...
    {Name: "foo", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Foo(cfg) }},
}
```

That's it — `plan`, `apply`, and `doctor` all pick it up automatically
through `AllResources`/`Doctor`, with zero separate plan/apply logic to
keep in sync.

## Why Go Struct Literals, Not YAML, for Resource Declarations

Module resource declarations are plain Go struct literals — not a DSL,
not YAML. This was a deliberate choice made once Go was already settled
on as the implementation language (the *user-facing* `~/.devboost.yaml`
config file is unrelated and stays YAML — see `config/config.go`). The
reasoning: most of devboost's code will be read and modified by coding
agents more than by hand, so "not fluent in Go" carries little weight —
what matters is readability and correctness, and Go struct literals get
full compiler and type checking that a YAML-based DSL would need to
reinvent. See git history for the fuller discussion (Docker Compose was
considered as a counter-example favoring YAML, but ultimately devboost's
"a module should be trivial to add, optimized on boilerplate" goal was
better served by plain typed Go).

## Resource Kinds ("Providers")

Each kind in `engine/kinds/` is a small, reusable, parametrized primitive
— the equivalent of a Terraform provider resource type. Shelling out
inside a kind's `Diff`/apply is fine, and often correct, when the target
tool's own CLI is the best available diff/apply primitive (`GitConfig`
shells to `git config`, `Package` shells to `brew`/`apt`/`dnf`/`pacman`) —
never as a shortcut to skip writing a real diff.

`CommandGuarded` is the one deliberate escape hatch, for state that
genuinely doesn't fit any typed kind (see `engine/kinds/commandguarded.go`
for the full reasoning). A module still only ever declares data — `{ID,
Params, Wants}` — never imperative logic at the declaration site. An
unregistered `ID` fails loudly (not a silent no-op): adding a new use
requires writing a real Go implementation in `kinds`, registered via
`RegisterCommand`, the same amount of real work as adding a proper kind.
This is intentional friction — it must never be the easy path when a
typed kind is achievable instead.

## Dependencies

Resources declare explicit `DependsOn` when one resource's correctness
depends on another having already run — e.g. `security`'s managed block
depends on `zsh`'s full-file render, because writing the block first and
then having zsh's `File` resource overwrite the whole file would silently
destroy it. `engine.ComputeDiff`/`DiffAndExecute` topologically sort on
this, with cycle and unknown-dependency detection. This replaced the bash
version's implicit, hand-maintained module registration order.

## doctor: Tool-First Grouping

`doctor` diffs every module's combined resource graph in one pass (not
per-module in isolation — an earlier version did that and broke the
moment a cross-module `DependsOn` was added, see the module's tests for
the regression coverage), then groups the results back by owning module
for display. This keeps output readable as dedup checks and diagnostics
accumulate — ten tools with several checks each still reads as ten
grouped sections, not a flat list of every individual check.

## Module Rationale Documentation

Every module that picks a specific tool documents *why* directly above
its constructor function — adoption/reputation research, first-party
guidance where it conflicts with common practice, and an honest
confidence level (well-documented consensus vs. taste vs. now-questionable).
See `.agents/skills/devboost-module-author/SKILL.md` for the process, and
any module file (e.g. `engine/modules/git.go`, `engine/modules/corepack.go`)
for real examples — including examples of a default that research
revealed was worth actually fixing, not just caveat-ing.

## Adding a New Module

1. Create `engine/modules/foo.go` with a `Foo(cfg *config.Config) []engine.Resource` function.
2. Research the tool choice (see the module-author skill) and write the rationale as a doc comment above `Foo`.
3. Add a test file `foo_test.go` covering the enable-gate and default-resource-shape.
4. Register it in `engine/modules/registry.go`'s `All` slice.
5. Run `go build ./... && go test ./...`.
6. Add any new config keys to `.devboost.yaml.example`.

## Design Principles

1. **Idempotency**: structural, not per-module discipline — `ComputeDiff` returning no pending ops *is* "nothing to do."
2. **Non-destructive**: never overwrite user files directly — managed blocks/includes, backups before every mutating write.
3. **Config-driven**: sensible defaults in code, overrides in `~/.devboost.yaml`.
4. **Explicit dependencies**: `DependsOn`, not registration order, decides execution order.
5. **One diff function**: `plan` and `apply` can never narrate a different change than what actually happens.

## Bootstrap Distribution

`install.sh` (repo root) is a small, pure-POSIX-shell dispatcher modeled
on rustup's `rustup-init.sh`: detect OS/arch (including the Rosetta 2
edge case on Apple Silicon), download the matching prebuilt binary from
the [latest release](https://github.com/rolfsormo/devboost/releases/latest),
exec it. No application logic lives there — everything real is in the Go
binary. Cross-compilation is currently local-only (`GOOS`/`GOARCH` builds
run by hand for each release, no GitHub Actions release pipeline yet).

## Future Enhancements

- Automate cross-platform release builds (currently cut by hand — see Bootstrap Distribution above).
- In-tool config-schema-version warning (bash version had this; not yet ported — see [CHANGELOG.md](CHANGELOG.md)).
- Async/prefetched `apt-get update` once the CLI orchestrates multiple resources concurrently.
- Periodic adversarial re-review of each module's tool choice against current ecosystem state (see the module-author skill).
