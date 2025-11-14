# devboost Architecture

## Overview

devboost is built as a **modular Bash framework** that concatenates into a single distributable script. This design provides:

- **Single-command UX**: One `devboost.sh` file to distribute
- **Easy extensibility**: Add modules by creating new files
- **Maintainability**: Clear separation of concerns
- **Zero runtime dependencies**: Pure Bash + `yq` (or Python fallback)

## Directory Structure

```
devboost/
  devboost.sh.in       # Entry point (minimal)
  build.sh            # Build script (concatenates everything)
  
  core/                # Framework components
    core_main.sh       # CLI, argument parsing, main execution
    core_log.sh        # Logging functions (db_log_*)
    core_os.sh         # OS detection, package manager abstraction
    core_yaml.sh       # YAML config parsing (via yq)
    core_files.sh      # File operations (backup, write, block management)
    core_modules.sh    # Module registry system
    
  modules/             # Feature modules
    module_pkg.sh      # Package installation
    module_znap.sh     # Znap plugin manager
    module_zsh.sh      # Zsh configuration
    module_starship.sh # Starship prompt
    module_tmux.sh     # Tmux configuration
    module_mise.sh     # Mise toolchains
    module_direnv.sh   # Direnv setup
    module_git.sh       # Git delta config
    module_services.sh  # Service management (atuin, etc.)
    
  templates/           # Template files (future use)
  dist/                # Build output (gitignored)
```

## Module Interface

Every module implements a simple interface:

```bash
# modules/module_foo.sh

db_module_foo_register() {
    db_register_module "foo" \
        "db_module_foo_plan" \
        "db_module_foo_apply" \
        "db_module_foo_doctor"  # optional
}

db_module_foo_plan() {
    # Read config with db_yaml_get
    # Output what would change
    db_log_info "Would do X, Y, Z"
}

db_module_foo_apply() {
    # Do the actual work, idempotently
    # Use core helpers: db_write_file, db_upsert_block, etc.
}

db_module_foo_doctor() {
    # Optional: diagnostics for this module
}
```

## Core Framework

### Module Registry

The registry (`core_modules.sh`) maintains:

- `DB_MODULE_NAMES`: Array of module names
- `DB_MODULE_PLAN_FUNC`: Map of name → plan function
- `DB_MODULE_APPLY_FUNC`: Map of name → apply function
- `DB_MODULE_DOCTOR_FUNC`: Map of name → doctor function (optional)

Modules register themselves during `db_load_modules()`.

### Config System

Uses `yq` (preferred) or Python3 with PyYAML (fallback) to parse `~/.devboost.yaml`:

```bash
# Get config value with default
local value=$(db_yaml_get '.zsh.enable' 'true')

# Get list (space-separated)
local pkgs=$(db_yaml_get_list '.packages.base[]')
```

### File Operations

Core file helpers ensure safety:

- `db_write_file`: Writes file with backup
- `db_upsert_block`: Replaces block between markers in existing file
- `db_remove_block`: Removes block between markers
- `db_backup_file`: Creates timestamped backup

### OS Abstraction

`core_os.sh` provides:

- `db_detect_os`: Sets `DB_OS` (darwin, linux-ubuntu, linux-fedora, linux-arch)
- `db_install_packages`: Installs packages via appropriate package manager
- `db_command_exists`: Checks if command is available

## Build Process

`build.sh` concatenates files in order:

1. Entry point (`devboost.sh.in`)
2. Core framework (in dependency order)
3. All modules
4. Module registration code
5. Main execution wrapper

Result: Single `devboost.sh` file (~1280 lines) that's self-contained.

## Adding a New Module

1. Create `modules/module_neovim.sh`:

```bash
db_module_neovim_register() {
    db_register_module "neovim" \
        "db_module_neovim_plan" \
        "db_module_neovim_apply"
}

db_module_neovim_plan() {
    db_log_info "Would configure Neovim"
}

db_module_neovim_apply() {
    local config_dir="$HOME/.config/nvim"
    db_ensure_dir "$config_dir"
    # ... configure neovim
}
```

2. Add registration call to `build.sh`:

```bash
cat modules/module_neovim.sh
echo ""
# ... in registration section:
db_module_neovim_register
```

3. Rebuild: `./build.sh`

That's it! The module is now part of the system.

## Design Principles

1. **Idempotency**: All operations are safe to re-run
2. **Non-destructive**: Never overwrite user files directly
3. **Config-driven**: Defaults in code, overrides in YAML
4. **Modular**: Each module is independent
5. **Extensible**: Adding features = adding modules

## Future Enhancements

- Template system for complex configs
- Module dependencies (e.g., zsh depends on znap)
- Parallel module execution
- Better diff/plan output
- Module-specific uninstall hooks

