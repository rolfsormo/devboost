# Changelog

All notable changes to devboost will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with OS/tooling-specific adjustments.

## [Unreleased]

### Added
- Comprehensive test suite for macOS and Linux
- Automated Podman installation and setup (macOS via Homebrew, Linux via system package managers)
- Docker/Podman runtime detection with automatic fallback
- Arch Linux ARM64 detection and graceful skipping
- Test scripts for all supported distributions (Ubuntu, Debian, Fedora, Arch)
- Sandboxed macOS testing environment

### Changed
- Linux test script now supports both Docker and Podman
- Test infrastructure automatically installs Podman if neither Docker nor Podman is available
- Improved error messages and platform detection
- **README.md completely rewritten** - More welcoming, better organized, includes complete tool list with links

### Fixed
- YAML parsing null value handling (prevents "unbound variable" errors)
- Test script compatibility with both Docker and Podman
- Package installation output now suppressed (cleaner output, errors still shown on failure)
- Missing packages for aliases (dust, duf, procs) now installed by default

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

[Unreleased]: https://github.com/yourusername/devboost/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/yourusername/devboost/releases/tag/v1.0.0

