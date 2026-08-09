# Changelog

All notable changes to devboost will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with OS/tooling-specific adjustments.

## [Unreleased]

## [2.0.0] - 2026-08-08

### Changed
- **Full rewrite from bash to Go.** Replaced the hand-written bash `plan`/`apply` function pairs per module (which had already drifted out of sync in production — see the `1.3.0` double-sourcing fix) with a Terraform-inspired typed-resource engine: modules declare desired state as `engine.Resource` values, and one shared function (`engine.ComputeDiff`) computes what's out of sync for both `plan` and `apply`. See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design. The CLI entry point is now a single Go binary (`devboost`) instead of a concatenated `devboost.sh`; `install.sh` is a new pure-POSIX-shell bootstrap dispatcher (rustup-style) that fetches and execs the binary.
- **All prior functionality ported**: `apply`, `plan`, `doctor` (now grouped tool-first by module), `uninstall`, `clean`, `migrate-from-oh-my-zsh`, and every module (znap, zsh, starship, tmux, mise, pkg, git/delta, corepack, direnv, services/atuin, security, and the zinit/asdf/nvm dedup modules).
- **direnv**: no longer writes a `use_mise` `.direnvrc` helper by default. mise's own docs call that integration pattern deprecated; `mise activate zsh` (already run globally) fully replaces it via mise's own directory-change hook. direnv stays installed for plain per-directory env vars; `direnv.content` still lets a user opt into managed `.direnvrc` content.
- **corepack**: now installs itself via `npm install -g corepack` when missing, instead of silently treating absence as "nothing to do." Node 25+ no longer bundles corepack (Node's own TSC decision), so absence is now the expected case on a current toolchain — this matches the exact replacement workflow Node's TSC decision names explicitly.

### Added
- `devboost clean --dry-run` support (previously apply-only in the bash version's port).
- Every module that picks a specific tool now documents why, as a doc comment above its constructor — adoption/reputation research, first-party guidance, and an honest confidence level (well-documented consensus vs. taste vs. now-questionable). See `.agents/skills/devboost-module-author/SKILL.md` for the process used to write these.
- Explicit `DependsOn` on resources — a real dependency graph with topological sort and cycle detection, replacing the bash version's implicit, hand-maintained module registration order.

### Removed
- The bash implementation (`core/`, `modules/`, `build.sh`, `devboost.sh`, `devboost.sh.in`) and its bash-specific test suite.
- `system.package_manager`, `zsh.plugin_manager`, `packages.optional`, and `tmux.plugins` config keys, which were either bash-only no-ops or not carried forward as real config surface in the Go engine (znap and TPM's own plugin set are still the actual behavior — see `.devboost.yaml.example`).

### Known gaps carried forward from the bash version (not yet re-implemented)
- Automatic in-tool warning when a user's config predates the current MAJOR version (see [AGENTS.md](AGENTS.md#5-versioning-strategy)).
- Release builds are still cut by hand (no GitHub Actions release pipeline yet) — a [v2.0.0 pre-release](https://github.com/rolfsormo/devboost/releases/tag/v2.0.0) exists with real binaries and a verified `install.sh` flow, but stays pre-release until this merges to `main` (see [README.md](README.md)).

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

