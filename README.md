# devboost

A single command that bootstraps a workstation (macOS + Linux) into a modern dev environment: zsh + starship + znap + fzf/zoxide/atuin + mise + direnv + tmux(+TPM)/resurrect/continuum/yank/logging — without clobbering user customizations.

## Quick Install & Run

### Option 1: Direct Download and Run (Recommended)

```bash
# Download and run in one command
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh | bash -s -- apply
```

**Note**: This downloads and executes the script directly. For better security, use Option 2 to review the script first.

### Option 2: Download First, Then Run (More Secure)

```bash
# Download the script
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh -o /tmp/devboost.sh

# Review it (optional but recommended)
less /tmp/devboost.sh

# Run it
bash /tmp/devboost.sh apply
```

### Option 3: Install to PATH

```bash
# Download to a permanent location
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh -o ~/bin/devboost

# Make it executable
chmod +x ~/bin/devboost

# Ensure ~/bin is in your PATH (add to ~/.zshrc if needed)
export PATH="$HOME/bin:$PATH"

# Now run from anywhere
devboost apply
```

### Option 4: Build from Source

```bash
# Clone the repository
git clone https://github.com/rolfsormo/devboost.git
cd devboost

# Build the single-file script
./build.sh

# Run it
./devboost.sh apply
```

**Note**: This project has been tested on macOS and Linux (Ubuntu, Debian, Fedora). The test suite automatically uses Docker or Podman (installing Podman if needed). Arch Linux tests are skipped on ARM64 systems due to image availability limitations.

## Features

- **Single command** to install/upgrade: `devboost` (or `sh devboost.sh`)
- **Idempotent**: safe to re-run; only apply drift
- **Modular**: each area is a module with clear inputs/outputs
- **Config-driven**: `~/.devboost.yaml` (defaults applied if missing)
- **Non-destructive**: keep user customizations intact. We inject managed blocks into separate include files and source them
- **Cross-OS**: macOS (Homebrew), Debian/Ubuntu (apt), Fedora (dnf), Arch (pacman)
- **Extendable**: simple module registry pattern; future plugins are just new modules
- **Zero prompts**: everything works out of the box with sensible defaults

## Architecture

devboost uses a **modular architecture**:

- **Core framework** (`core/`): Logging, OS detection, YAML parsing, file operations, module registry
- **Modules** (`modules/`): One file per feature area (pkg, znap, zsh, starship, tmux, mise, direnv, git, services)
- **Build system**: Concatenates all modules into a single `devboost.sh` for distribution

This makes it easy to:
- Add new modules (just create `modules/module_foo.sh` and register it)
- Maintain and test individual components
- Keep the codebase readable and organized

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

## Quick Start

### Option 1: Download and Run (Recommended)

```bash
# Download and run the pre-built script
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh | bash
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/devboost.git
cd devboost

# Build the single-file script
./build.sh

# Run it
./devboost.sh apply
```

### Option 3: Install to PATH

```bash
# After building or downloading
cp devboost.sh ~/bin/devboost
chmod +x ~/bin/devboost

# Now you can run from anywhere
devboost apply
```

## Usage

```bash
devboost [COMMAND] [OPTIONS]

# default subcommand = apply
```

### Commands

- `apply`: Converge machine to config (default)
- `plan`: Show actions without changing anything
- `doctor`: Check prerequisites, PATHs, shells, conflicting files
- `uninstall`: Remove managed files/blocks (leaves user custom files untouched)

### Options

- `--config FILE`: Config file path (default: `~/.devboost.yaml`)
- `--dry-run`: Show what would be done without making changes
- `--verbose, -v`: Enable verbose output
- `--help, -h`: Show help message
- `--version`: Show version

### Examples

```bash
# See what would change
devboost plan

# Apply configuration
devboost apply

# Apply with verbose output
devboost apply --verbose

# Use custom config file
devboost apply --config ~/my-devboost.yaml

# Check system health
devboost doctor

# Remove devboost managed files
devboost uninstall
```

## Configuration

Create `~/.devboost.yaml` to customize your setup. See `.devboost.yaml.example` for a complete example with all available options.

### Minimal Config

```yaml
# ~/.devboost.yaml
# Everything has sensible defaults, so this file is optional!

zsh:
  aliases:
    enable: true

tmux:
  enable: true
  settings:
    mouse: true
```

### Full Config Example

```yaml
# ~/.devboost.yaml

system:
  tmux_control_mode: true
  package_manager: auto

packages:
  base:
    - zsh
    - zoxide
    - fzf
    - ripgrep
    - fd
    - bat
    - eza
    - jq
    - yq
    - git-delta
    - lazygit
    - direnv
    - mise
    - atuin
    - starship
    - tmux

zsh:
  enable: true
  history:
    use_atuin: true
  fzf:
    enable: true
  aliases:
    enable: true

prompt:
  enable_starship: true

tmux:
  enable: true
  settings:
    base_index: 1
    mouse: true

toolchains:
  enable_mise: true
  globals:
    node: "lts"
    python: "3.12"
    go: "1.23"
    rust: "stable"
    deno: "latest"
```

## What Gets Installed

### Packages
- **Shell**: zsh
- **Navigation**: zoxide, fzf
- **Search**: ripgrep, fd
- **Viewers**: bat, eza
- **Utilities**: jq, yq, git-delta, lazygit, dust, duf, procs
- **Environment**: direnv, mise, atuin, starship
- **Terminal**: tmux

### Shell Configuration
- zsh with znap plugin manager
- zsh-autosuggestions and zsh-syntax-highlighting plugins
- Starship prompt (minimalist theme)
- Modern aliases (ls → eza, cat → bat, etc.)
- zoxide, fzf, and atuin integration

### Tmux Configuration
- TPM (Tmux Plugin Manager)
- Plugins: resurrect, continuum, yank, logging
- Sensible defaults (mouse support, large history, etc.)

### Toolchains (via mise)
- Node.js (LTS)
- Python 3.12
- Go 1.23
- Rust (stable)
- Deno (latest)

### Other
- direnv with mise integration
- git-delta for better diffs
- Atuin for shell history sync

## File Layout

devboost **never modifies your custom files directly**. Instead, it creates managed companion files:

- `~/.zshrc` → Contains include block that sources `~/.zshrc.devboost`
- `~/.zshrc.devboost` → Fully managed by devboost (replaced on apply)
- `~/.config/starship.toml` → Managed (backed up first)
- `~/.tmux.conf` → Contains managed block between markers
- `~/.devboost.state.json` → State tracking for idempotency
- `~/.devboost/backups/` → Backups of modified files

## Safety

- All writes are confined to **managed include files** or **managed blocks** between markers
- First-touch backups in `~/.devboost/backups/<timestamp>/`
- `plan` mode shows the diff of would-be changes
- `state.json` records versions/commit of templates for drift detection
- Idempotent: safe to run multiple times

## Requirements

- bash 4.0+
- git
- curl
- sudo (for package installation on Linux)
- yq or python3 with PyYAML (for YAML parsing, optional - falls back to basic parser)

## Supported Operating Systems

- **macOS**: Uses Homebrew (installed automatically if missing)
- **Debian/Ubuntu**: Uses apt
- **Fedora**: Uses dnf
- **Arch Linux**: Uses pacman

## Versioning

devboost follows **Semantic Versioning** with OS/tooling-specific adjustments:

- **MAJOR**: Breaking changes (config schema changes, removed features)
- **MINOR**: New features, backwards compatible
- **PATCH**: Bug fixes, backwards compatible

**Upgrade Safety:**
- PATCH and MINOR versions: Always safe to upgrade
- MAJOR versions: Review changelog for breaking changes

The script will warn if your config file is from an older MAJOR version, but won't block execution.

## Troubleshooting

### YAML Parsing Issues

If you encounter YAML parsing errors, install `yq`:
- macOS: `brew install yq`
- Linux: See [yq installation](https://github.com/mikefarah/yq#install)

Alternatively, Python 3 with PyYAML works as a fallback.

### Package Installation Failures

Some packages may not be available in all package managers. The script will warn but continue. You can install missing packages manually or add them to your config's `packages.optional` list.

### Tmux Plugins Not Installing

If using iTerm2 control mode, plugins install automatically via CLI. In normal tmux mode, run `prefix + I` in a tmux session to install plugins.

### Zoxide Errors

If you see `command not found: __zoxide_pwd`, ensure zoxide is installed and the `.zshrc.devboost` file was generated correctly. Run `devboost apply` again to regenerate.

## Contributing

Contributions are very welcome! This project has been tested on macOS and Linux (Ubuntu, Debian, Fedora). Pull requests are encouraged for:

- Testing on additional Linux distributions
- Bug fixes and improvements
- New modules
- Documentation improvements

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines, including:
- How to get started
- Testing requirements (all code must be tested!)
- Code quality standards
- How to add new modules (it's super easy!)
- Commit message style (CBEAMS)

See [AGENTS.md](AGENTS.md) for development guidelines and best practices.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed list of changes.
