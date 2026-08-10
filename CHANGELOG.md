# Changelog

All notable changes to devboost will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with OS/tooling-specific adjustments.

## [Unreleased]

### Changed
- **`legacy_shell` config key renamed to `optimize`.** "Legacy" never said what it was legacy *compared to*; `optimize` matches how the feature is actually described (and the README's own "Startup Optimizations" section). `legacy_shell.enable`/`.zshrc`/`.zprofile` become `optimize.enable`/`.zshrc`/`.zprofile`. `kinds.MarkerPrefix` (`# devboost:disabled:...`) is unchanged — it's a stable on-disk format independent of the config key name.
- **oh-my-zsh migration is now a normal `apply` resource, not a separate `--yes`-gated subcommand.** `migrate-from-oh-my-zsh` and `--yes` are removed. `omz_migration` joins the three existing dedup modules (zinit/asdf/nvm) as a fourth resource under `optimize.enable`, converged automatically by a plain `devboost apply` — no confirmation prompt, matching the zero-prompt behavior every other devboost resource already has. A confirm-then-rerun `--yes` gate is itself a soft prompt spread across two invocations; rustup's `-y`, Homebrew's `NONINTERACTIVE=1`, and oh-my-zsh's own installer's `--unattended` all confirm the real pattern is "ask only when run interactively by a human," not "skip and tell them to rerun with a flag."

### Added
- **`devboost undo`**: reverses whatever `apply` actually converged, for any resource whose kind implements the new `engine.Undoer` interface — currently the four optimize resources. Restores commented-out lines to their exact original text, and for oh-my-zsh, moves the archived `~/.oh-my-zsh` back into place and restores `.zshrc` from its pre-migration backup. Consumed backups are renamed (`-reverted` suffix, kept on disk rather than deleted) so a repeated `undo` correctly reports nothing left to restore instead of re-restoring the same backup. Refuses to run (unless `--force`) when something it would restore has changed since it last converged — e.g. `~/.oh-my-zsh` was recreated by hand after a migration already ran — since its backups may no longer describe the current state accurately.
- **`--no-optimizations`**: a per-invocation shortcut for `optimize.enable: false`, via a new `config.Config.Set` (the write-side mirror of `Get`'s dotted-key lookup) — so it's indistinguishable from the user having written the equivalent config key.
- **`--force`**: lets `undo` proceed despite the drift-refusal check above.
- `kinds.CommandGuarded` gained optional `UndoConverge`/`UndoSatisfied` on `GuardedCommand`, and `kinds.LineInFile` gained `Undo()` — both implement the new `engine.Undoer` interface.

## [2.0.0] - 2026-08-08

### Changed
- **Full rewrite from bash to Go.** Replaced the hand-written bash `plan`/`apply` function pairs per module (which had already drifted out of sync in production — see the `1.3.0` double-sourcing fix) with a Terraform-inspired typed-resource engine: modules declare desired state as `engine.Resource` values, and one shared function (`engine.ComputeDiff`) computes what's out of sync for both `plan` and `apply`. See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design. The CLI entry point is now a single Go binary (`devboost`) instead of a concatenated `devboost.sh`; `install.sh` is a new pure-POSIX-shell bootstrap dispatcher (rustup-style) that fetches and execs the binary.
- **All prior functionality ported**: `apply`, `plan`, `doctor` (now grouped tool-first by module), `uninstall`, `clean`, oh-my-zsh migration (later folded into `apply` itself — see `[Unreleased]`), and every module (znap, zsh, starship, tmux, mise, pkg, git/delta, corepack, direnv, services/atuin, security, and the zinit/asdf/nvm dedup modules).
- **direnv**: no longer writes a `use_mise` `.direnvrc` helper by default. mise's own docs call that integration pattern deprecated; `mise activate zsh` (already run globally) fully replaces it via mise's own directory-change hook. direnv stays installed for plain per-directory env vars; `direnv.content` still lets a user opt into managed `.direnvrc` content.
- **corepack**: now installs itself via `npm install -g corepack` when missing, instead of silently treating absence as "nothing to do." Node 25+ no longer bundles corepack (Node's own TSC decision), so absence is now the expected case on a current toolchain — this matches the exact replacement workflow Node's TSC decision names explicitly.

### Added
- `devboost clean --dry-run` support (previously apply-only in the bash version's port).
- Every module that picks a specific tool now documents why, as a doc comment above its constructor — adoption/reputation research, first-party guidance, and an honest confidence level (well-documented consensus vs. taste vs. now-questionable). See `.agents/skills/devboost-module-author/SKILL.md` for the process used to write these.
- Explicit `DependsOn` on resources — a real dependency graph with topological sort and cycle detection, replacing the bash version's implicit, hand-maintained module registration order.
- **Real Debian/Ubuntu and Fedora support for tools apt/dnf don't package.** Found via the first genuine end-to-end `apply` run this project has ever done on fresh Linux (previously only `--dry-run` was tested): lazygit, mise, atuin, starship, dust, and procs (varies by distro) aren't packaged at all on stock apt/dnf — the bash tool had this exact same gap, silently. `kinds.VendorInstall` (fetch-and-run the tool's own official installer) and `kinds.GitHubReleaseInstall` (download+extract the latest GitHub release for tools with no installer script at all — lazygit, procs) now converge these into `~/.local/bin`, added to `PATH` by the zsh module's own rendered config. See `docs/` and `ARCHITECTURE.md`'s Resource Kinds section.

### Fixed
- **A failed resource no longer aborts the whole `apply`.** `engine.DiffAndExecute` previously stopped the entire run on any resource's `Execute` error — found as a real, current-state blocker via the Linux testing above (one apt package genuinely unavailable silently prevented zsh/tmux/git config and everything else from converging). Failed resources and everything transitively depending on them are now skipped individually; everything else still converges. `Package.Execute` similarly no longer stops at the first failed package — it attempts all of them and reports every failure together.
- **`Mise()`/`Services()` no longer gate themselves off at module-construction time.** Both used to decide "does this resource even exist" via `exec.LookPath` before any resource had executed — meaning neither could ever see a tool a *different* resource installs earlier in the same `apply` run (e.g. mise's own toolchain-converge step silently never ran on a machine where mise itself needed installing first). Availability is now checked fresh inside each resource's own `Satisfied`/`Converge`.
- **Managed-tool invocations resolve their real path, not a bare command name.** devboost's own process `PATH` never includes `~/.local/bin` (only a future shell gets that, via the zsh module) — `exec.Command("mise", ...)` from inside devboost silently failed "not found" even moments after mise had genuinely just been installed. Fixed via `kinds.ResolveBinary`.
- `corepack` now explicitly `DependsOn`s `mise_toolchains` — it needs mise's Node toolchain to exist first, previously an undeclared/accidental ordering assumption.
- **`mise_toolchains`/`atuin_service` now have a real, engine-enforced dependency on whichever resource actually installs mise/atuin**, instead of relying on module registration order or a live `exec.LookPath` check inside `Satisfied`/`Converge`. `Resource` gained `Provides`/`NeedsProvider`: a resource declares a capability tag it satisfies (e.g. `base_packages` or `vendor_install_mise` both `Provides: []string{"mise"}`, whichever is real on this platform) and a consumer declares `NeedsProvider: []string{"mise"}` without knowing which concrete resource that resolves to. The engine resolves this to a real dependency edge at `topoSort` time; an unresolvable or ambiguous tag is a hard error. Consolidated `kinds.ResolveBinary`'s PATH-then-`~/.local/bin` resolution into a new `kinds.Command` helper, replacing a copy of the same fix previously duplicated in `mise.go` and `corepack.go`.
- **Anonymous `git clone` no longer hangs waiting on a Keychain prompt.** A successful anonymous HTTPS clone still invoked the configured `credential.helper` afterwards to store credentials — on macOS with Homebrew's default git config, this blocked on a GUI prompt that never comes in a headless `apply` run. Found via direct process-tree inspection of a real hung `apply`, both in `kinds.GitClone` directly and inside tmux's TPM plugin-install scripts (which shell out to `git clone` internally). devboost only ever clones public repos anonymously, so there's never a credential to store — the helper is now disabled per-invocation.

### Removed
- The bash implementation (`core/`, `modules/`, `build.sh`, `devboost.sh`, `devboost.sh.in`) and its bash-specific test suite.
- `system.package_manager`, `zsh.plugin_manager`, and `packages.optional` config keys, which were bash-only no-ops not carried forward as real config surface in the Go engine (znap's plugin set is still the actual behavior — see `.devboost.yaml.example`). `tmux.plugins` as a user-supplied plugin *list* was also dropped (TPM's plugin set is fixed in code, not user-configurable), but `tmux.plugins.logging.enable` was added back as a real, live key — see the tool-choice review below.

### Tool-choice review (2026-08-08)
- First pass of the periodic adversarial tool-choice review — see [docs/tool-choice-review-2026-08.md](docs/tool-choice-review-2026-08.md). Findings implemented directly: swapped `zsh-users/zsh-syntax-highlighting` → `zdharma-continuum/fast-syntax-highlighting`; gated `tmux-logging` behind `tmux.plugins.logging.enable` (default `false`, was previously bundled unconditionally); reverted atuin's `filter_mode` to its own upstream default (`global`) instead of devboost's unjustified `directory` override, adding `filter_mode_shell_up_key_binding: directory` separately for quick up-arrow recall; relabeled a few doc comments (starship's `command_timeout`/`add_newline`, `procs`) to state plainly where they're a deliberate deviation or genuine taste call rather than settled consensus.

### Known gaps carried forward from the bash version (not yet re-implemented)
- Automatic in-tool warning when a user's config predates the current MAJOR version (see [AGENTS.md](AGENTS.md#5-versioning-strategy)).
- Release builds are cut by hand for now (no GitHub Actions release pipeline yet) — see [v2.0.0](https://github.com/rolfsormo/devboost/releases/tag/v2.0.0).
- No GitHub Actions CI runs on this repo — deliberate, not a gap: local development cost is effectively free, GitHub Actions spend is not. All testing (`go build`/`go vet`/`go test ./...`, `tests/test-install.sh`, and real Docker container runs for Linux) is run locally before pushing.

### Known external gap (not devboost's to fix)
- mise has its own bug resolving `deno@lts` on Linux (malformed release URL — confirmed reproducible on a real Ubuntu 24.04 container, see [issue #15](https://github.com/rolfsormo/devboost/issues/15)). node/python/go/rust all install correctly via the same mechanism; only deno is affected. `Mise()`'s `Converge` already degrades gracefully (warns, doesn't fail the whole run) when this happens.

## [1.4.0] - 2026-08-08

### Added
- `legacy_shell` module: detects shell tooling in a pre-existing `~/.zshrc`/`~/.zprofile` that duplicates what devboost already manages — a leftover `zinit` setup loading the same plugins as devboost's `znap` (`zsh-autosuggestions`, a syntax-highlighting fork), `asdf` sourced alongside devboost's `mise`, or `nvm`'s shell hook (measured at ~850-900ms per login shell) also alongside `mise` — surfaced via `doctor`, and disabled by `apply`/`plan`. Redundant lines are commented out in place with a `# devboost:disabled:<id>` marker rather than deleted, so they can be reviewed or restored by hand at any time; if a user removes the marker themselves, later runs respect that as an explicit override and leave the line alone. Full pre/post snapshots of every edited file are kept in `~/.devboost/backups/` as an audit trail independent of the marker.
- `devboost clean` command: permanently removes lines previously marked `# devboost:disabled:...`. Idempotent and order-independent — it re-derives what to remove by scanning the live file each run, so it works correctly regardless of when or whether `apply` last ran. Respects `--dry-run`.

### Fixed
- `zsh` module no longer calls `compinit` itself in the generated `.zshrc.devboost`. It ran *before* znap was sourced, but znap redefines `compinit`/`compdef` as no-ops and runs its own deferred, `precmd`-hook-based compinit into a separate dumpfile the moment it loads — so devboost's own call was a wasted full completion rebuild every shell start, immediately superseded by znap's. Removing it, combined with the `legacy_shell` fixes above, took a real-machine's measured login-shell startup from ~1.44s to ~285-305ms (~80% reduction).

### Motivation
Investigating real-world zsh startup lag surfaced this exact conflict on a live machine: a pre-existing, non-devboost `zinit` setup, `asdf`, and `nvm` were all running fully redundant plugin/version-manager initialization on every shell start, before devboost's own managed config even loaded — compounded by devboost's own generated config doing a second, wasted completion rebuild on top.

## [1.3.0] - 2026-08-04

### Added
- `zsh` module and `security` doctor check now detect a pre-existing unmarked line in `~/.zshrc` that already sources `~/.zshrc.devboost` (e.g. left over from a prior manual edit or `devboost migrate-from-oh-my-zsh` recovery). `apply`/`plan` skip injecting devboost's own marked include block in that case instead of double-sourcing it — sourcing `.zshrc.devboost` twice silently doubled the cost of everything in it (znap, atuin, mise, aliases, etc.) on every shell start.

### Changed
- `devboost migrate-from-oh-my-zsh` now removes `~/.oh-my-zsh` itself (replicating oh-my-zsh's own `tools/uninstall.sh`) instead of requiring the user to run `uninstall_oh_my_zsh` first — one command instead of two. Since this is destructive, it now requires an explicit `--yes` flag to actually run (dry-run still needs no flag).
- Simplified customization recovery from a `git merge-file` 3-way merge to a plain diff-and-append: since the command always restores `.zshrc` from the pre-install base itself first, "current" and "base" are identical going into any merge, so a real conflict was structurally impossible — the git dependency and merge-conflict handling for this path were dead code and have been removed.

## [1.2.0] - 2026-08-03

### Added
- `devboost migrate-from-oh-my-zsh` command: after running oh-my-zsh's own `uninstall_oh_my_zsh` (which only restores the pre-install `.zshrc` and can lose later customizations), 3-way merges the timestamped `.zshrc.omz-uninstalled-*` backup back onto the restored `.zshrc` via `git merge-file`, stripping oh-my-zsh's own template lines first. Non-conflicting additions merge automatically; ambiguous ones get `<<<<<<<` conflict markers. Supports `--dry-run`.
- Security/doctor check: warns if `~/.oh-my-zsh` is installed alongside devboost (redundant with znap+starship, can slow shell startup); README documents manual removal
- Security hygiene module (`security`): `devboost-check` alias summarises outdated brew/mise packages; doctor warns on stale Homebrew index, `latest`-pinned toolchains, and HTTP TPM remotes
- Editor/terminal integration docs in README: Ghostty, Zed, VS Code auto-attach to named tmux session
- Comprehensive test suite for macOS and Linux
- Automated Podman installation and setup (macOS via Homebrew, Linux via system package managers)
- Docker/Podman runtime detection with automatic fallback
- Arch Linux ARM64 detection and graceful skipping
- Test scripts for all supported distributions (Ubuntu, Debian, Fedora, Arch)
- Sandboxed macOS testing environment

### Changed
- Default Python toolchain version: 3.12 → 3.14 (current stable)
- Default Go toolchain version: 1.23 → 1.26 (current stable)
- Default Deno toolchain channel: `latest` → `lts` (Deno now ships a maintained LTS channel; matches devboost's stable-not-bleeding-edge philosophy). Removed the `latest`-pin doctor warning's deno exemption, since deno no longer defaults to `latest`.
- Ubuntu Docker test image: 22.04 → 26.04 (current LTS)
- Debian Docker test image: bookworm → trixie (current stable)
- `system.tmux_control_mode` renamed to `system.auto_install_plugins` (same behaviour, clearer name)
- Linux test script now supports both Docker and Podman
- Test infrastructure automatically installs Podman if neither Docker nor Podman is available
- Improved error messages and platform detection
- **README.md completely rewritten** - More welcoming, better organized, includes complete tool list with links

### Fixed
- YAML parsing null value handling (prevents "unbound variable" errors)
- Test script compatibility with both Docker and Podman
- Package installation output now suppressed (cleaner output, errors still shown on failure)
- Missing packages for aliases (dust, duf, procs) now installed by default
- Test suite no longer triggers macOS Keychain GUI prompts during `git clone` (znap/TPM/tmux plugins) — `run-tests.sh` now neutralizes `credential.helper` for test git processes
- `tests/README.md` referenced a stale Ubuntu 22.04 image that no longer matched the Dockerfile
- README incorrectly stated `bash 4.0+` as a requirement, contradicting the project's own bash 3.2 compatibility support (macOS ships 3.2 by default; the test suite specifically covers it)

## [1.0.0] - 2025-01-XX

### Added
- Core framework with module registry system
- Package installation module (macOS, Debian/Ubuntu, Fedora, Arch)
- Znap plugin manager module
- Zsh configuration module with managed `.zshrc.devboost`
- Starship prompt configuration
- Tmux configuration with TPM and plugins
- Mise toolchain management
- Direnv integration
- Git delta configuration
- Services module (atuin daemon)
- Plan mode (dry-run)
- Doctor command for diagnostics
- Uninstall functionality
- Backup system for modified files
- YAML config parsing (yq/Python fallback)
- Cross-OS support

### Changed
- N/A (initial release)

### Fixed
- N/A (initial release)

### Security
- N/A (initial release)

---

## Version Format

- **[MAJOR.MINOR.PATCH]**: Version number
- **Breaking changes**: Require user action (config changes, etc.)
- **New features**: Backwards compatible additions
- **Bug fixes**: Backwards compatible fixes

## Upgrade Guide

### From X.Y.Z to X+1.0.0 (MAJOR upgrade)
- Review breaking changes below
- Update your `~/.devboost.yaml` if needed
- Run `devboost plan` to see what will change
- Run `devboost apply` to upgrade

### From X.Y.Z to X.Y+1.0 (MINOR upgrade)
- Safe to upgrade
- New features available
- Existing config continues to work

### From X.Y.Z to X.Y.Z+1 (PATCH upgrade)
- Safe to upgrade
- Bug fixes and improvements
- No config changes needed

---

[Unreleased]: https://github.com/yourusername/devboost/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/yourusername/devboost/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/yourusername/devboost/compare/v1.0.0...v1.2.0
[1.0.0]: https://github.com/yourusername/devboost/releases/tag/v1.0.0

