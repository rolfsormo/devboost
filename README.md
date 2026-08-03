# devboost 🚀

**One command to transform your workstation into a modern, opinionated development environment.**

Transform your macOS or Linux machine into a productivity powerhouse with a single command. devboost installs and configures the best-in-class tools for modern development, all while preserving your existing customizations.

```bash
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh | bash -s -- apply
```

> **✨ What you get:** A beautiful shell (zsh + starship), smart navigation (zoxide, fzf), powerful search (ripgrep, fd), modern replacements (bat, eza, dust, duf, procs), seamless toolchain management (mise), and a fully configured tmux setup — all in under 5 minutes.

---

## 🎯 Why devboost?

Setting up a development environment is tedious. You spend hours installing tools, configuring shells, tweaking prompts, and setting up aliases. devboost does all of this **automatically** with **sensible defaults** that work out of the box.

**Key principles:**
- ✅ **Non-destructive**: Never touches your existing configs — uses managed include files
- ✅ **Idempotent**: Safe to run multiple times — only applies what's needed
- ✅ **Opinionated**: Curated selection of best-in-class tools
- ✅ **Zero prompts**: Everything works automatically with smart defaults
- ✅ **Cross-platform**: Works on macOS and Linux (Ubuntu, Debian, Fedora, Arch)

---

## 🚀 Quick Start

### Install & Run (Recommended)

```bash
# Download and run in one command
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh | bash -s -- apply
```

**That's it!** Your development environment is being set up. Grab a coffee ☕ — this takes a few minutes.

### Alternative: Review First (More Secure)

```bash
# Download the script
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh -o /tmp/devboost.sh

# Review it (recommended)
less /tmp/devboost.sh

# Run it
bash /tmp/devboost.sh apply
```

### Install to PATH

```bash
# Download to a permanent location
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh -o ~/bin/devboost
chmod +x ~/bin/devboost

# Ensure ~/bin is in your PATH
export PATH="$HOME/bin:$PATH"

# Now run from anywhere
devboost apply
```

---

## 📦 What Gets Installed

devboost installs and configures a curated set of modern development tools. Here's everything that's included with links to their sources:

### 🐚 Shell & Navigation

- **[zsh](https://www.zsh.org/)** - Powerful shell with extensive customization
- **[zoxide](https://github.com/ajeetdsouza/zoxide)** - Smarter `cd` command that learns your habits
- **[fzf](https://github.com/junegunn/fzf)** - Fuzzy finder for files, history, and more
- **[atuin](https://github.com/ellie/atuin)** - Magical shell history with sync and search

**Shell Configuration:**
- **[znap](https://github.com/marlonrichert/zsh-snap)** - Fast zsh plugin manager
- **[zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions)** - Suggests commands as you type
- **[zsh-syntax-highlighting](https://github.com/zsh-users/zsh-syntax-highlighting)** - Real-time syntax highlighting
- **[starship](https://github.com/starship/starship)** - Minimal, fast, customizable prompt
- **Smart aliases** - `ls` → `eza`, `cat` → `bat`, `grep` → `rg`, `find` → `fd`, `du` → `dust`, `df` → `duf`, `ps` → `procs`

### 🔍 Search & File Operations

- **[ripgrep](https://github.com/BurntSushi/ripgrep)** (`rg`) - Blazing fast text search
- **[fd](https://github.com/sharkdp/fd)** - Simple, fast alternative to `find`

### 🎨 Modern Replacements

- **[bat](https://github.com/sharkdp/bat)** - `cat` with syntax highlighting and Git integration
- **[eza](https://github.com/eza-community/eza)** - Modern `ls` with colors, Git status, and more
- **[dust](https://github.com/bootandy/dust)** - More intuitive `du` with visual tree
- **[duf](https://github.com/muesli/duf)** - Better `df` with colors and formatting
- **[procs](https://github.com/dalance/procs)** - Modern `ps` with colors and tree view

### 🛠️ Utilities

- **[jq](https://github.com/jqlang/jq)** - Command-line JSON processor
- **[yq](https://github.com/mikefarah/yq)** - YAML processor (jq for YAML)
- **[git-delta](https://github.com/dandavison/delta)** - Syntax-highlighted pager for Git
- **[lazygit](https://github.com/jesseduffield/lazygit)** - Simple terminal UI for Git

### 🌍 Environment Management

- **[mise](https://github.com/jdx/mise)** - Fast toolchain manager (replaces asdf/nvm/pyenv)
- **[direnv](https://github.com/direnv/direnv)** - Load and unload environment variables per directory
- **[tmux](https://github.com/tmux/tmux)** - Terminal multiplexer with session persistence

**Toolchains (via mise):**
- **Node.js** (LTS)
- **Python** (3.14)
- **Go** (1.26)
- **Rust** (stable)
- **Deno** (LTS)

### 🎭 Tmux Configuration

- **[TPM](https://github.com/tmux-plugins/tpm)** - Tmux Plugin Manager
- **[tmux-resurrect](https://github.com/tmux-plugins/tmux-resurrect)** - Restore tmux sessions after restart
- **[tmux-continuum](https://github.com/tmux-plugins/tmux-continuum)** - Automatic session saving
- **[tmux-yank](https://github.com/tmux-plugins/tmux-yank)** - Copy to system clipboard
- **[tmux-logging](https://github.com/tmux-plugins/tmux-logging)** - Logging capabilities
- Sensible defaults (mouse support, large history, etc.)

### 🔗 Editor & Terminal Integration

Wire your terminal emulator and code editor to the devboost tmux session so every window auto-attaches and persists across restarts.

The `-A` flag means "attach if the session exists, create it otherwise" — so every new window lands in the same session automatically.

**Ghostty** (`~/.config/ghostty/config`):
```
command = /opt/homebrew/bin/tmux new-session -A -s main
```

**Zed** (`~/.config/zed/settings.json`):
```json
{
  "terminal": {
    "shell": {
      "program": "/opt/homebrew/bin/tmux",
      "args": ["new-session", "-A", "-s", "main"]
    }
  }
}
```

**VS Code** (`settings.json`):
```json
{
  "terminal.integrated.defaultProfile.osx": "tmux",
  "terminal.integrated.profiles.osx": {
    "tmux": {
      "path": "/opt/homebrew/bin/tmux",
      "args": ["new-session", "-A", "-s", "main"]
    }
  }
}
```

**Tip — grouped sessions:** If you want Zed/VS Code terminals to share windows but track focus independently (no mirroring), use `-t main` instead:
```json
"args": ["new-session", "-t", "main"]
```
This creates a grouped session — you'll see `main` and `main-1` in `tmux ls`, but all windows are shared.

On Linux, replace `/opt/homebrew/bin/tmux` with the output of `which tmux`.

---

## 💻 Usage

```bash
devboost [COMMAND] [OPTIONS]
```

### Commands

- **`apply`** - Set up your environment (default)
- **`plan`** - Preview what would change (dry-run)
- **`doctor`** - Check system health and prerequisites
- **`uninstall`** - Remove devboost-managed files

### Options

- `--config FILE` - Custom config file (default: `~/.devboost.yaml`)
- `--dry-run` - Show what would be done without making changes
- `--verbose, -v` - Enable verbose output
- `--help, -h` - Show help message
- `--version` - Show version

### Examples

```bash
# Preview changes
devboost plan

# Set up your environment
devboost apply

# Check system health
devboost doctor

# Use custom config
devboost apply --config ~/my-config.yaml
```

---

## ⚙️ Configuration

Everything works out of the box with sensible defaults. Customize by creating `~/.devboost.yaml`:

```yaml
# ~/.devboost.yaml
# Everything has defaults, so this file is optional!

zsh:
  aliases:
    enable: true

tmux:
  enable: true
  settings:
    mouse: true

toolchains:
  enable_mise: true
  globals:
    node: "lts"
    python: "3.13"
```

See [`.devboost.yaml.example`](.devboost.yaml.example) for all available options.

---

## 🛡️ Safety & Philosophy

devboost is designed to be **completely non-destructive**:

- ✅ **Never modifies your files directly** — uses managed include files
- ✅ **Automatic backups** — first-touch backups in `~/.devboost/backups/`
- ✅ **Idempotent** — safe to run multiple times
- ✅ **Preview mode** — use `plan` to see what would change
- ✅ **Easy removal** — `uninstall` removes all managed files

### File Layout

```
~/.zshrc                    → Contains include block (you control this)
~/.zshrc.devboost          → Fully managed by devboost
~/.config/starship.toml    → Managed (backed up first)
~/.tmux.conf               → Contains managed block between markers
~/.devboost/backups/       → Automatic backups
```

---

## 📸 Screenshots

> **Note**: Screenshots coming soon! We'd love contributions showing devboost in action.

Want to contribute a screenshot? Show off:
- Your terminal with starship prompt
- Aliases in action (`ls`, `cat`, etc.)
- tmux session with multiple panes
- Toolchain management with mise

---

## 🧪 Testing

This project has been tested on:
- ✅ macOS (with Podman for Linux testing)
- ✅ Ubuntu/Debian (via Docker/Podman)
- ✅ Fedora (via Docker/Podman)
- ⚠️ Arch Linux (skipped on ARM64 due to image limitations)

The test suite automatically uses Docker or Podman (installing Podman if needed). See [tests/README.md](tests/README.md) for details.

---

## 📋 Requirements

- bash 3.2+ (macOS system bash works out of the box)
- git
- curl
- sudo (for package installation on Linux)
- yq or python3 with PyYAML (optional — falls back to basic parser)

### Supported Operating Systems

- **macOS** - Uses Homebrew (installed automatically if missing)
- **Debian/Ubuntu** - Uses apt
- **Fedora** - Uses dnf
- **Arch Linux** - Uses pacman

---

## 🐛 Troubleshooting

### Package Installation Failures

Package installation output is suppressed for cleaner logs. If a package fails to install, the full error output will be displayed to help you troubleshoot.

Some packages may not be available in all package managers. You can install missing packages manually or add them to your config's `packages.optional` list.

### YAML Parsing Issues

Install `yq` for better YAML parsing:
- macOS: `brew install yq`
- Linux: See [yq installation](https://github.com/mikefarah/yq#install)

Python 3 with PyYAML works as a fallback.

### Tmux Plugins Not Installing

By default, devboost installs plugins automatically via the TPM CLI after writing the config. If you set `system.auto_install_plugins: false`, run `prefix + I` inside a running tmux session instead.

### Zoxide Errors

If you see `command not found: __zoxide_pwd`, ensure zoxide is installed and run `devboost apply` again to regenerate the config.

---

## 🤝 Contributing

Contributions are very welcome! This project has been tested on macOS and Linux. Pull requests are encouraged for:

- Testing on additional Linux distributions
- Bug fixes and improvements
- New modules
- Documentation improvements
- Screenshots and visual examples

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 📚 Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical design documentation
- [AGENTS.md](AGENTS.md) - Development guidelines
- [CHANGELOG.md](CHANGELOG.md) - Version history

---

## 🎯 Versioning

devboost follows **Semantic Versioning**:

- **PATCH** (1.1.0 → 1.1.1): Bug fixes, safe to upgrade
- **MINOR** (1.1.0 → 1.2.0): New features, safe to upgrade
- **MAJOR** (1.1.0 → 2.0.0): Breaking changes, review changelog

The script will warn if your config file is from an older MAJOR version.

---

**Ready to boost your development environment?** 🚀

```bash
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/devboost.sh | bash -s -- apply
```
