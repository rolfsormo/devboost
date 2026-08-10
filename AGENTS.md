# AGENTS.md - Development Guide for AI Agents and Contributors

This document provides guidelines for AI agents and human contributors working on devboost. The goal is to maintain **super high quality** code that follows Go and 2026 best practices, prioritizes **ease of use**, and makes it **trivially easy** to add new modules.

## Core Principles

### 1. User Experience First

- **Zero prompts**: Users should never be asked questions during execution. All decisions should be made automatically with sensible defaults.
- **Fully configurable**: Everything must be configurable via `~/.devboost.yaml`, but defaults should work perfectly out of the box.
- **Idempotent**: Safe to run multiple times. This is structural, not per-module discipline — a resource's `Diff()` returning `nil` once converged *is* "nothing to do," not something each module has to remember to check.
- **Non-destructive**: Never modify user's existing files directly. Use managed blocks/includes and back up before any mutating write.

### 2. Code Quality Standards

- **Go idioms**:
  - Prefer the standard library over reinventing helpers (`filepath.Base`, not a hand-rolled equivalent)
  - Wrap errors with context: `fmt.Errorf("resource %s: %w", id, err)`
  - `go vet ./...` and `gofmt` must pass cleanly
  - Keep functions focused (single responsibility)

- **Error handling**:
  - Always check and propagate errors
  - Provide helpful error messages with context — an unregistered `CommandGuarded` ID fails loudly, never a silent no-op
  - Never silently fail (unless explicitly handling an expected, documented case — e.g. a file that doesn't exist yet)

- **Readability**:
  - Clear function names (`Corepack`, not `applyCorepack`)
  - Comments explain *why*, not *what* — well-named identifiers already say what
  - Since most of this code is read (and often written) by both humans and coding agents, clarity beats cleverness

### 3. Module Development

**Adding a new module should be trivial.** A module is a plain function
returning a list of typed `engine.Resource` values — see
[ARCHITECTURE.md](ARCHITECTURE.md) for the full resource model
(`ResourceKind`, `PendingOp`, `ComputeDiff`, `DependsOn`).

1. Create `engine/modules/foo.go`:

```go
package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Foo ports <whatever prior art this replaces, or notes it's new>,
// gated on foo.enable.
//
// <Rationale: why this tool, why this default — see the "Research
// before defaulting" section below and the module-author skill.>
func Foo(cfg *config.Config) []engine.Resource {
	if cfg.Get("foo.enable", "true") != "true" {
		return nil
	}
	path := cfg.Get("foo.path", "~/.foorc")
	return []engine.Resource{
		{ID: "foo_config", Kind: kinds.File{Path: path, Content: renderFooConfig(cfg)}},
	}
}
```

2. Add it to `engine/modules/registry.go`'s `All` slice:

```go
{Name: "foo", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Foo(cfg) }},
```

3. Write `foo_test.go` covering the enable-gate and the default resource shape.

4. `go build ./... && go test ./...`

**That's it!** The module is now part of `plan`, `apply`, and `doctor` — with no separate plan/apply logic to keep in sync, since `engine.ComputeDiff` is the only place "what needs to change" is computed.

### Module Best Practices

- **Always check the enable flag first**: `if cfg.Get("module.enable", "true") != "true" { return nil }`
- **Prefer an existing kind** (`engine/kinds/`) over writing new diff/apply logic — `File`, `BlockInFile`, `LineInFile`, `GitConfig`, `Package`, `DirExists`, `GitClone` cover most cases
- **`CommandGuarded` is the deliberate escape hatch, not a shortcut** — use it only when no typed kind fits, and only by registering a real Go implementation via `kinds.RegisterCommand` (see `engine/kinds/commandguarded.go`). A module still only ever declares data (`{ID, Params, Wants}`) at the call site, never imperative logic.
- **Provide defaults**: All config values should have sensible defaults
  - **Research before defaulting**: Before choosing a default value, always research developer community preferences and best practices — check adoption/reputation (GitHub stars, maintenance activity), prefer a tool's own first-party docs over general blog consensus when they conflict, and check whether the choice is still current (tools get deprecated/unbundled/superseded — verify, don't assume)
  - **Justify defaults**: Defaults should be chosen based on what works best for developers, not just what the tool's own default is
  - **Document reasoning, honestly**: Write the rationale as a doc comment directly above the module's constructor, and say plainly whether it's well-documented consensus, a reasonable-but-undocumented preference, or (if research reveals it) now questionable — see `.agents/skills/devboost-module-author/SKILL.md` for the full process, and `engine/modules/corepack.go`/`engine/modules/direnv.go` for real examples of a default that got fixed, not just caveated, once research showed it should change
- **Idempotent operations**: a resource kind's `Diff()` must return `nil` once system state matches desired state — this is what makes `apply` safe to re-run, not a per-module convention to remember
- **Explicit dependencies**: declare `DependsOn` whenever one resource's correctness depends on another having already run (e.g. two resources writing to the same file) — never rely on registration order

### 4. Commit Message Style (CBEAMS)

Follow the [CBEAMS commit message style](https://chris.beams.io/posts/git-commit/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code meaning change
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(mise): add support for a custom deno version pin

Allow toolchains.globals.deno to be set to a specific version
instead of only "lts". Defaults to "lts" if not specified.

Closes #42
```

```
fix(git): correct delta.line-numbers config key

The key was being read as git.delta.lineNumbers, which never
matched a real config key — changed to git.delta.line_numbers
to match the documented example.

Fixes #38
```

```
docs(readme): add build-from-source instructions

Adds a Quick Start section for building the binary directly,
since no release binaries are published yet.
```

**Rules:**
1. Separate subject from body with a blank line
2. Limit subject line to 50 characters
3. Capitalize subject line
4. Do not end subject line with a period
5. Use imperative mood ("add" not "adds" or "added")
6. Wrap body at 72 characters
7. Use body to explain *what* and *why* vs. *how*

### 5. Versioning Strategy

devboost follows **Semantic Versioning**:

**Format:** `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes that require user action
  - Config file schema changes (old configs won't work)
  - Removal of features
  - Changes to default behavior that users rely on
  - **Users should be aware**: If they're on an older MAJOR version, they may need to update their config

- **MINOR**: New features, backwards compatible
  - New modules
  - New config options (with defaults)
  - Enhancements to existing modules
  - **Users can upgrade safely**: Old configs still work

- **PATCH**: Bug fixes, backwards compatible
  - Fixes to existing functionality
  - Performance improvements
  - Documentation updates
  - **Users should upgrade**: Fixes issues they may be experiencing

**Version Compatibility:**

- Users can always upgrade PATCH versions safely
- Users can upgrade MINOR versions safely (new features available)
- Users upgrading MAJOR versions should review [CHANGELOG.md](CHANGELOG.md) for breaking changes
- **Not yet implemented**: automatic in-tool detection/warning that a user's config predates the current MAJOR version (the earlier bash implementation had this; it hasn't been ported — see [ARCHITECTURE.md](ARCHITECTURE.md#future-enhancements)). Until it exists, check the changelog manually before a MAJOR upgrade.

**Versioning Best Practices:**

- Update `version` in `cmd/devboost/main.go`
- Tag releases: `git tag -a v2.0.0 -m "Release 2.0.0"`
- Maintain `CHANGELOG.md` with:
  - Breaking changes (MAJOR)
  - New features (MINOR)
  - Bug fixes (PATCH)

**REQUIRED Before Pushing to origin/main:**

1. **Bump version** if needed (MAJOR for breaking changes, MINOR for new features, PATCH for fixes)
2. **Build**: `go build ./...` must succeed
3. **Vet**: `go vet ./...` must pass
4. **Test**: `go test ./...` must pass (this includes the slow real end-to-end apply test — don't skip it before a push, only during rapid local iteration)
5. **Commit**: the version bump in `cmd/devboost/main.go`

**Version Bump Guidelines:**
- **PATCH** (2.0.0 → 2.0.1): Bug fixes, documentation updates, internal improvements
- **MINOR** (2.0.0 → 2.1.0): New features, new modules, new config options (backwards compatible)
- **MAJOR** (2.0.0 → 3.0.0): Breaking changes, config schema changes, removed features

**Never push to main without:**
- ✅ Version bumped (if changes warrant it)
- ✅ `go build ./...` succeeds
- ✅ `go vet ./...` passes
- ✅ `go test ./...` passes (including any new tests for the feature)
- ✅ Changes documented (README, CHANGELOG, module rationale doc comment, or ARCHITECTURE.md as appropriate)

**Complete Workflow for Changes:**
1. Create a feature branch: `git checkout -b feat/<short-description>`
2. Make your changes in small, logical commits (see Commit Message Style)
3. Write/update tests for the changes
4. Run all tests and ensure they pass: `go test ./...`
5. Build and verify: `go build ./... && go vet ./...`
6. Update documentation (README, CHANGELOG, module rationale, etc.)
7. Bump version if needed (in `cmd/devboost/main.go`)
8. Push the branch and open a PR against `main`
9. PRs must pass all checks before merging — never push directly to `main`

**Why PRs matter for a public repo:**
- They create a reviewable record of intent and rationale
- They keep `main` always in a releasable state
- They allow external contributors to follow the same path as maintainers
- Squash or rebase before merging to keep `git log` readable

### 6. Testing Requirements

**All tests must pass before a change can be considered done.**

Before submitting changes, you **must** run and pass all applicable tests:

1. **Build**: `go build ./...` must succeed
2. **Vet**: `go vet ./...` must pass
3. **Unit tests**: `go test ./... -short` must pass (fast — skips the slow real end-to-end apply)
4. **Full tests**: `go test ./...` must pass before pushing (includes `TestSandboxedApplyPlanDoctorIdempotent`, which builds the real binary and runs `plan`/`apply --dry-run`/`doctor`/`apply` against real tools in a sandboxed `HOME` — slow, touches real Homebrew/git, but is what actually catches "the pieces work in isolation but not wired together")
5. **Plan test**: `go run ./cmd/devboost plan` should show expected changes
6. **Idempotency test**: run `apply` twice against a fresh temp `HOME` — the second run should make no further `.zshrc`/config changes
7. **Config test**: test with a minimal config and a full config (see `.devboost.yaml.example`)
8. **Platform tests**: the full Go test suite already covers macOS/Linux where the code is platform-dependent (see `runtime.GOOS` checks in the relevant tests) — if you can't test on a platform locally, note it in your PR

**Test Execution:**
```bash
go build ./...
go vet ./...
go test ./... -short   # fast iteration
go test ./...          # full, including the slow end-to-end test — required before push
```

**Failure is not an option**: If tests fail, the change is not complete. Fix the issues or document why the failure is acceptable (with maintainer approval).

### 7. Documentation Requirements

- **Code comments**: Explain *why*, not *what*
- **Module rationale**: Every module that picks a specific tool documents why, directly above its constructor function — see `.agents/skills/devboost-module-author/SKILL.md`
- **Config docs**: Document all config options in `.devboost.yaml.example`, and verify it actually loads (`config.Load(".devboost.yaml.example")`) rather than trusting it by inspection
- **README**: Keep up to date with new features
- **CHANGELOG**: Document all user-facing changes
- **ARCHITECTURE.md**: Keep up to date if the engine or module structure changes

### 8. Security Considerations

- **Never execute unsanitized user input**
- **Use absolute paths** where possible
- **Sanitize/validate file paths** before operations
- **Backup before modify**: always back up a file before overwriting it (see `kinds.BackupFile`)
- **Principle of least privilege**: don't require sudo unless necessary
- **Never fabricate credential-fetching commands against real hosts during testing** — if a keychain/credential-helper interaction needs to be tested, use a dummy host, never a real one

### 9. Cross-Platform Compatibility

- **OS detection**: `kinds.DetectOS()` returns a typed `kinds.OS` (`OSDarwin`, `OSLinuxUbuntu`, `OSLinuxFedora`, `OSLinuxArch`, `OSOther`)
- **Package managers**: `kinds.Package` abstracts brew/apt/dnf/pacman — add new package-name mappings there, not in individual modules
- **Path differences**: handle macOS (`/opt/homebrew`) vs Linux paths explicitly where it matters (e.g. tmux integration docs)
- **Test on both**: when possible, test on macOS and Linux locally before pushing — no GitHub Actions CI runs automatically (deliberate: local development cost is effectively free, GitHub Actions spend is not, so it stays opt-in rather than running on every push)

### 10. Performance Guidelines

- **Minimize external calls**: cache results when appropriate (e.g. `sync.Once` for a one-time `apt-get update`, not concurrency — just deduplication)
- **Batch where the engine already supports it**: `ComputeDiff` computes the whole graph once for `plan`/`doctor`; `apply`'s `DiffAndExecute` still runs one resource at a time in dependency order, which is required (a dependent resource must see real post-execution state), not accidental overhead

## Quick Reference

### Adding a Module Checklist

- [ ] Create `engine/modules/foo.go`
- [ ] Implement `Foo(cfg *config.Config) []engine.Resource` (check the enable flag first)
- [ ] Research the tool choice; write the rationale as a doc comment above `Foo`
- [ ] Register in `engine/modules/registry.go`'s `All` slice
- [ ] Write `foo_test.go` (enable-gate, default resource shape, any `DependsOn` interactions)
- [ ] Add config options to `.devboost.yaml.example`
- [ ] `go build ./... && go test ./...`
- [ ] Update README if user-facing
- [ ] Update CHANGELOG
- [ ] Commit with CBEAMS style

### Common Patterns

**Check if enabled:**
```go
if cfg.Get("module.enable", "true") != "true" {
	return nil
}
```

**Write a fully-managed file (with automatic backup before overwrite):**
```go
{ID: "foo_config", Kind: kinds.File{Path: path, Content: content}}
```

**Inject/update a block into an existing (user-owned) file:**
```go
{ID: "foo_block", Kind: kinds.BlockInFile{Path: path, StartMarker: startMarker, EndMarker: endMarker, Content: block}}
```

**Declare a dependency on another resource:**
```go
{ID: "foo_block", Kind: kinds.BlockInFile{...}, DependsOn: []string{"other_resource_id"}}
```

**Escape hatch for state no typed kind covers** (register the real implementation once, in an `init()`, in the file that owns the use case):
```go
func init() {
	kinds.RegisterCommand("foo_converged", kinds.GuardedCommand{
		Satisfied: func(params any) (bool, error) { /* ... */ },
		Converge:  func(params any) error { /* ... */ },
	})
}
// declaration site stays pure data:
{ID: "foo", Kind: kinds.CommandGuarded{ID: "foo_converged", Wants: "foo converged"}}
```

## Questions?

If you're unsure about implementation details:
1. Check existing modules for patterns (`engine/modules/*.go`)
2. Review [ARCHITECTURE.md](ARCHITECTURE.md) for the engine/resource model
3. Check `.agents/skills/devboost-module-author/SKILL.md` for the research-and-document process
4. Test your changes thoroughly
5. Ask for review if needed

Remember: **Ease of use > Performance > Cleverness**
