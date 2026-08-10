# devboost 🚀

**A curated, researched shell/dev-tool setup for macOS and Linux, installed with one command.**

devboost installs and configures a specific, opinionated set of modern CLI tools — zsh + starship, ripgrep/fd/bat/eza, mise for toolchains, tmux with session persistence — and wires them together correctly (aliases, PATH, shell init order). Every default is picked from real research into what the developer community actually converges on, documented in the code, not just assumed. It edits your shell config to make that happen, but only inside clearly-marked regions it owns, with a real `undo`.

```bash
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/install.sh | sh -s -- apply
```

Already have zinit, asdf, nvm, or oh-my-zsh set up the way you like? Add
`--no-optimizations` to install just the tools and leave your existing shell
setup alone — see [Quick Start](#-quick-start).

> **✨ What you get:** A beautiful shell (zsh + starship), smart navigation (zoxide, fzf), powerful search (ripgrep, fd), modern replacements (bat, eza, dust, duf, procs), seamless toolchain management (mise), and a fully configured tmux setup — all in under 5 minutes.

---

## 🎯 Why devboost?

Setting up a development environment is tedious. You spend hours installing tools, configuring shells, tweaking prompts, and setting up aliases. devboost does all of this **automatically** with **sensible defaults** that work out of the box.

**Key principles:**
- ✅ **Reversible**: Edits are confined to clearly-marked managed regions (an include line in `.zshrc`, a marker block in `.tmux.conf`), everything is backed up first, and nothing of yours is ever deleted
- ✅ **Idempotent**: Safe to run multiple times — only applies what's needed
- ✅ **Opinionated**: Curated selection of best-in-class tools
- ✅ **Zero prompts**: Everything works automatically with smart defaults
- ✅ **Cross-platform**: Works on macOS and Linux (Ubuntu, Debian, Fedora, Arch)

---

## 🚀 Quick Start

### Install & Run (Recommended)

```bash
# Download and run in one command
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/install.sh | sh -s -- apply

# Or, to skip touching any existing shell tooling (zinit, asdf, nvm, oh-my-zsh):
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/install.sh | sh -s -- apply --no-optimizations
```

**That's it!** Your development environment is being set up. Grab a coffee ☕ — this takes a few minutes.

`install.sh` is a small, pure-POSIX-shell bootstrap dispatcher (the same
pattern rustup uses): it detects your OS/architecture, downloads the
matching prebuilt `devboost` binary from the
[latest release](https://github.com/rolfsormo/devboost/releases/latest),
and execs it with whatever arguments you passed. No application logic
lives in the shell script itself, which is what makes it small enough to
actually read before running.

### Alternative: Review First (More Secure)

```bash
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/install.sh -o /tmp/install.sh
less /tmp/install.sh          # review it — it's short
sh /tmp/install.sh apply
```

### Building from Source

```bash
git clone https://github.com/rolfsormo/devboost.git
cd devboost
go build -o devboost ./cmd/devboost
./devboost apply
```

Copy the resulting `devboost` binary to somewhere on your `PATH` (e.g.
`~/bin/devboost`) to run it from anywhere afterward.

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
- **[fast-syntax-highlighting](https://github.com/zdharma-continuum/fast-syntax-highlighting)** - Real-time syntax highlighting
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
- **[tmux-logging](https://github.com/tmux-plugins/tmux-logging)** - Logging capabilities (opt-in — set `tmux.plugins.logging.enable: true`, off by default)
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
- **`undo`** - Reverse a prior [startup optimization](#-startup-optimizations) (zinit/asdf/nvm/oh-my-zsh)
- **`uninstall`** - Remove devboost-managed files
- **`clean`** - Permanently remove devboost-disabled optimization lines and archived directories

### Options

- `--config FILE` - Custom config file (default: `~/.devboost.yaml`)
- `--dry-run` - Show what would be done without making changes
- `--no-optimizations` - Skip [startup optimizations](#-startup-optimizations) for this run; same as `optimize.enable: false` in config
- `--force` - Let `undo` proceed even though something it would restore has changed since it last converged (`undo` refuses by default — see below)
- `--help, -h` - Show help message
- `--version` - Show version

### Examples

```bash
# Preview changes
devboost plan

# Set up your environment
devboost apply

# Set up your environment, but leave existing shell tooling untouched
devboost apply --no-optimizations

# Check system health
devboost doctor

# Use custom config
devboost apply --config ~/my-config.yaml

# Reverse a startup optimization (undoes zinit/asdf/nvm dedup and/or
# an oh-my-zsh migration, whichever devboost actually converged)
devboost undo --dry-run                     # preview first
devboost undo                               # then actually run it

# Permanently remove lines devboost previously disabled (see below)
devboost clean --dry-run                    # preview first
devboost clean                              # then actually run it
```

---

## 🧹 Startup Optimizations

A machine that's already had shell tooling installed by hand often ends up
running two things that do the same job — one from that earlier setup, one
from devboost — on every single shell start. devboost detects the overlap
and disables the redundant half, so you only pay the startup cost once.

| Found in your existing setup | Duplicates | Startup cost if left running |
|---|---|---|
| `zinit` loading `zsh-autosuggestions`/syntax highlighting | devboost's `znap` | wasted plugin load |
| `asdf` sourced in `.zshrc` | devboost's `mise` | wasted version-manager init |
| `nvm`'s shell hook sourced in `.zprofile` | devboost's `mise` | ~850–900ms per login shell — measured via `zprof` on a real machine, the single largest contributor found |
| `oh-my-zsh` (framework: its own plugin manager, prompt, curated plugins) | devboost's `znap` + `starship` + curated plugin set | slower startup, and can cause conflicting keybindings/completions |

All four are converged automatically by a plain `devboost apply` — no
confirmation prompt, no separate command, the same zero-prompt behavior as
every other resource devboost manages. Reversibility is what makes that a
reasonable default, not a pre-execution gate:

- **zinit, asdf, and nvm**: `apply` comments out just the redundant lines in
  place (prefixed `# devboost:disabled:...`) rather than deleting them.
- **oh-my-zsh**: `apply` replicates oh-my-zsh's own uninstaller — removes
  `~/.oh-my-zsh` (archived, not deleted), renames your current `.zshrc` to a
  timestamped `~/.zshrc.omz-uninstalled-*` backup, and restores
  `~/.zshrc.pre-oh-my-zsh` if that pre-install snapshot exists — then
  recovers anything you added to `.zshrc` *after* installing oh-my-zsh
  (aliases, `PATH` changes, etc.), which the plain uninstaller alone would
  otherwise strand in that backup. oh-my-zsh's own template lines
  (`ZSH_THEME`, `plugins=(...)`, `source $ZSH/oh-my-zsh.sh`, etc.) are
  stripped out first, so only your genuine additions get appended back.

Every one of these is backed up first (see [File Layout](#file-layout)), and
every one of these is undoable:

```bash
devboost undo --dry-run   # preview what would be restored
devboost undo             # restore whatever apply actually converged
```

`undo` reverses exactly what happened — it restores commented-out lines to
their original text, and for oh-my-zsh, moves the archived `~/.oh-my-zsh`
back into place and restores `.zshrc` from its pre-migration backup. Each
restored backup is renamed (suffixed `-reverted`, kept on disk rather than
deleted) so running `undo` again afterward correctly reports nothing left to
restore, instead of redoing the same restore. If something `undo` would
restore has changed since it last converged — e.g. you recreated
`~/.oh-my-zsh` by hand after the migration already ran — `undo` refuses and
tells you what changed, since its backups may no longer describe the current
state accurately; pass `--force` if you want it to proceed anyway.

To skip this detection entirely, up front, use `--no-optimizations` (see
[Quick Start](#-quick-start)) or set `optimize.enable: false` in your config
— the two are equivalent. `devboost clean` is a separate, one-way step: it
permanently deletes lines `apply` previously commented out, instead of
restoring them.

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
    python: "3.14"
```

See [`.devboost.yaml.example`](.devboost.yaml.example) for all available options.

---

## 🛡️ Safety & Philosophy

Setup touches a few existing files — an include line in `.zshrc`, a
marker block in `.tmux.conf`, redundant lines from other shell tooling
commented out in place, and (if present) oh-my-zsh archived rather than
deleted — but always this way:

- ✅ **Backed up first** — first-touch backups in `~/.devboost/backups/` before any existing file is touched
- ✅ **Edits confined to managed regions** — everything outside the include line/marker block is left alone; lines are never deleted, only commented out with a `# devboost:disabled:...` marker
- ✅ **Reversible** — `devboost undo` reverses any [startup optimization](#-startup-optimizations) apply converged
- ✅ **Idempotent** — safe to run multiple times
- ✅ **Preview mode** — use `plan` to see what would change
- ✅ **Easy removal** — `uninstall` removes all managed files and blocks

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

devboost is a Go binary with a normal Go test suite (`go test ./...`),
including an end-to-end test that builds the real binary and runs
`plan`/`apply`/`doctor` against a sandboxed `HOME`. See
[tests/README.md](tests/README.md) for details.

---

## 📋 Requirements

To install via `install.sh` (recommended):
- curl or wget (to fetch the prebuilt binary)
- git (devboost itself shells out to it for clones/config)
- sudo (for package installation on Linux)

To build from source instead:
- git
- Go (see `go.mod` for the minimum version)

No bash version requirement, no YAML-parser dependency — devboost is a
single self-contained binary with Go's YAML support built in.

### Supported Operating Systems

- **macOS** - Uses Homebrew (installed automatically if missing)
- **Debian/Ubuntu** - Uses apt
- **Fedora** - Uses dnf
- **Arch Linux** - Uses pacman

---

## 🐛 Troubleshooting

### Package Installation Failures

If a package fails to install, the full error output is shown so you can troubleshoot. Some packages may not be available in all package managers — install missing packages manually, or adjust your config's `packages.base` list.

### Tmux Plugins Not Installing

By default, devboost installs plugins automatically via the TPM CLI after writing the config. If you set `system.auto_install_plugins: false`, run `prefix + I` inside a running tmux session instead.

### Zoxide Errors

If you see `command not found: __zoxide_pwd`, ensure zoxide is installed and run `devboost apply` again to regenerate the config.

### Already Using oh-my-zsh, zinit, asdf, or nvm?

See [Startup Optimizations](#-startup-optimizations) — devboost detects
overlap with each of these and converges away from it automatically as
part of `apply`, reversibly (`devboost undo`). Use `--no-optimizations`
to skip this detection entirely instead.

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
- **MAJOR** (1.1.0 → 2.0.0): Breaking changes, review [CHANGELOG.md](CHANGELOG.md)

Check `devboost --version` and the changelog before upgrading across a
MAJOR version. (Automatic in-tool warnings for an outdated config version
aren't implemented yet — see the changelog manually for now.)

---

**Ready to boost your development environment?** 🚀

```bash
curl -fsSL https://raw.githubusercontent.com/rolfsormo/devboost/main/install.sh | sh -s -- apply
```
