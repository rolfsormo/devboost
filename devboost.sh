#!/usr/bin/env bash
# devboost - Bootstrap a modern dev environment
# Idempotent, config-driven, non-destructive
# This file is the entry point - core and modules are concatenated during build

set -euo pipefail

# Version is set in core_main.sh


# === Core Framework ===
# Core logging functions

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

db_log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

db_log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

db_log_warn() {
    echo -e "${YELLOW}⚠${NC} $*"
}

db_log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

db_log_verbose() {
    if [[ "${DB_VERBOSE:-false}" == "true" ]]; then
        echo -e "${BLUE}[verbose]${NC} $*"
    fi
}

db_die() {
    db_log_error "$@"
    exit 1
}


# OS detection and package manager abstraction

db_detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        DB_OS="darwin"
    elif [[ -f /etc/debian_version ]]; then
        DB_OS="linux-ubuntu"
    elif [[ -f /etc/fedora-release ]]; then
        DB_OS="linux-fedora"
    elif [[ -f /etc/arch-release ]]; then
        DB_OS="linux-arch"
    else
        DB_OS="other"
    fi
    db_log_verbose "Detected OS: $DB_OS"
}

db_install_packages() {
    local pkgs=("$@")
    if [[ ${#pkgs[@]} -eq 0 ]]; then
        return 0
    fi
    
    case "$DB_OS" in
        darwin)
            if ! command -v brew >/dev/null 2>&1; then
                db_log_info "Installing Homebrew..."
                if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
                else
                    db_log_info "Would install Homebrew"
                fi
            fi
            for pkg in "${pkgs[@]}"; do
                if ! brew list "$pkg" &>/dev/null; then
                    db_log_info "Installing: $pkg"
                    if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                        local output
                        output=$(brew install "$pkg" 2>&1) || {
                            db_log_error "Failed to install $pkg"
                            echo "$output" >&2
                            return 1
                        }
                    else
                        db_log_info "Would install: $pkg"
                    fi
                else
                    db_log_verbose "Already installed: $pkg"
                fi
            done
            ;;
        linux-ubuntu)
            if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                sudo apt-get update -qq
            fi
            for pkg in "${pkgs[@]}"; do
                if ! dpkg -l | grep -q "^ii[[:space:]]*${pkg}[[:space:]]"; then
                    db_log_info "Installing: $pkg"
                    if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                        local output
                        output=$(sudo apt-get install -y "$pkg" 2>&1) || {
                            db_log_error "Failed to install $pkg"
                            echo "$output" >&2
                            return 1
                        }
                    else
                        db_log_info "Would install: $pkg"
                    fi
                else
                    db_log_verbose "Already installed: $pkg"
                fi
            done
            ;;
        linux-fedora)
            for pkg in "${pkgs[@]}"; do
                if ! rpm -q "$pkg" &>/dev/null; then
                    db_log_info "Installing: $pkg"
                    if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                        local output
                        output=$(sudo dnf install -y "$pkg" 2>&1) || {
                            db_log_error "Failed to install $pkg"
                            echo "$output" >&2
                            return 1
                        }
                    else
                        db_log_info "Would install: $pkg"
                    fi
                else
                    db_log_verbose "Already installed: $pkg"
                fi
            done
            ;;
        linux-arch)
            for pkg in "${pkgs[@]}"; do
                if ! pacman -Qi "$pkg" &>/dev/null; then
                    db_log_info "Installing: $pkg"
                    if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
                        local output
                        output=$(sudo pacman -S --noconfirm "$pkg" 2>&1) || {
                            db_log_error "Failed to install $pkg"
                            echo "$output" >&2
                            return 1
                        }
                    else
                        db_log_info "Would install: $pkg"
                    fi
                else
                    db_log_verbose "Already installed: $pkg"
                fi
            done
            ;;
        *)
            db_log_error "Unsupported OS: $DB_OS"
            return 1
            ;;
    esac
}

db_command_exists() {
    command -v "$1" >/dev/null 2>&1
}


# YAML config handling via yq

DB_CONFIG_PATH="${DB_CONFIG_PATH:-$HOME/.devboost.yaml}"

db_yaml_get() {
    local path="$1" default="${2-}"
    if [[ -f "$DB_CONFIG_PATH" ]]; then
        if db_command_exists yq; then
            local result
            result=$(yq "$path" "$DB_CONFIG_PATH" 2>/dev/null || echo "$default")
            # Handle null values from yq
            if [[ "$result" == "null" ]] || [[ -z "$result" ]]; then
                echo "$default"
            else
                echo "$result"
            fi
        elif db_command_exists python3 && python3 -c "import yaml" 2>/dev/null; then
            python3 -c "
import yaml
import sys
try:
    with open('$DB_CONFIG_PATH', 'r') as f:
        data = yaml.safe_load(f) or {}
    def get_nested(d, keys):
        for k in keys.split('.'):
            if isinstance(d, dict) and k in d:
                d = d[k]
            else:
                return None
        return d
    result = get_nested(data, '$path')
    print(result if result is not None else '$default')
except:
    print('$default')
"
        else
            echo "$default"
        fi
    else
        echo "$default"
    fi
}

db_yaml_get_list() {
    local path="$1"
    if [[ -f "$DB_CONFIG_PATH" ]]; then
        if db_command_exists yq; then
            yq -e "$path[]" "$DB_CONFIG_PATH" 2>/dev/null | tr '\n' ' ' || echo ""
        else
            echo ""
        fi
    else
        echo ""
    fi
}


# File manipulation helpers

db_ensure_dir() {
    local dir="$1"
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would create directory: $dir"
        return 0
    fi
    mkdir -p "$dir"
}

db_backup_file() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        return 0
    fi
    
    local backup_dir="${DB_BACKUP_DIR:-$HOME/.devboost/backups}"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${backup_dir}/${timestamp}"
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would backup: $file -> ${backup_path}/$(basename "$file")"
        return 0
    fi
    
    db_ensure_dir "$backup_path"
    cp "$file" "${backup_path}/$(basename "$file")"
    db_log_verbose "Backed up: $file"
}

db_write_file() {
    local file="$1"
    local content="$2"
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would write: $file"
        echo "$content"
    else
        db_backup_file "$file"
        echo "$content" > "$file"
        db_log_success "Wrote: $file"
    fi
}

db_upsert_block() {
    local file="$1"
    local start_marker="$2"
    local end_marker="$3"
    local new_content="$4"
    
    if [[ ! -f "$file" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would create: $file"
        else
            echo "$new_content" > "$file"
            db_log_success "Created: $file"
        fi
        return 0
    fi
    
    if grep -q "$start_marker" "$file" 2>/dev/null; then
        # Replace existing block
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would replace block in: $file"
        else
            db_backup_file "$file"
            local temp_file=$(mktemp)
            local block_file=$(mktemp)
            echo "$new_content" > "$block_file"
            awk -v start="$start_marker" -v end="$end_marker" -v block_file="$block_file" '
                $0 ~ start { 
                    in_block=1
                    while ((getline line < block_file) > 0) {
                        print line
                    }
                    close(block_file)
                    next 
                }
                $0 ~ end { 
                    in_block=0
                    next 
                }
                !in_block { print }
            ' "$file" > "$temp_file"
            mv "$temp_file" "$file"
            rm -f "$block_file"
            db_log_success "Updated block in: $file"
        fi
    else
        # Append block
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would append block to: $file"
        else
            db_backup_file "$file"
            echo "" >> "$file"
            echo "$new_content" >> "$file"
            db_log_success "Injected block into: $file"
        fi
    fi
}

db_remove_block() {
    local file="$1"
    local start_marker="$2"
    local end_marker="$3"
    
    if [[ ! -f "$file" ]] || ! grep -q "$start_marker" "$file" 2>/dev/null; then
        return 0
    fi
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would remove block from: $file"
    else
        db_backup_file "$file"
        local temp_file=$(mktemp)
        awk -v start="$start_marker" -v end="$end_marker" '
            $0 ~ start { in_block=1; next }
            $0 ~ end { in_block=0; next }
            !in_block { print }
        ' "$file" > "$temp_file"
        mv "$temp_file" "$file"
        db_log_success "Removed block from: $file"
    fi
}


# Module registry system

DB_MODULE_NAMES=()
declare -A DB_MODULE_PLAN_FUNC
declare -A DB_MODULE_APPLY_FUNC
declare -A DB_MODULE_DOCTOR_FUNC

db_register_module() {
    local name="$1" plan="$2" apply="$3" doctor="${4:-}"
    DB_MODULE_NAMES+=("$name")
    DB_MODULE_PLAN_FUNC["$name"]="$plan"
    DB_MODULE_APPLY_FUNC["$name"]="$apply"
    if [[ -n "$doctor" ]]; then
        DB_MODULE_DOCTOR_FUNC["$name"]="$doctor"
    fi
    db_log_verbose "Registered module: $name"
}

# db_load_modules() is defined after all modules are loaded (in build output)

db_run_plan() {
    db_log_info "Planning changes..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Planning module: $m"
        "${DB_MODULE_PLAN_FUNC[$m]}" || true
    done
}

db_run_apply() {
    db_log_info "Applying configuration..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Applying module: $m"
        "${DB_MODULE_APPLY_FUNC[$m]}" || true
    done
}

db_run_doctor() {
    db_log_info "Running diagnostics..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        if [[ -n "${DB_MODULE_DOCTOR_FUNC[$m]:-}" ]]; then
            db_log_verbose "Checking module: $m"
            "${DB_MODULE_DOCTOR_FUNC[$m]}" || true
        fi
    done
}


# Main entry point and CLI

DB_VERSION="1.1.2"
DB_SUBCOMMAND="apply"
DB_DRY_RUN=false
DB_VERBOSE=false
DB_BACKUP_DIR="${HOME}/.devboost/backups"
DB_STATE_FILE="${HOME}/.devboost.state.json"

db_parse_flags() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            apply|plan|doctor|uninstall)
                DB_SUBCOMMAND="$1"
                shift
                ;;
            --config)
                DB_CONFIG_PATH="$2"
                shift 2
                ;;
            --dry-run)
                DB_DRY_RUN=true
                shift
                ;;
            --verbose|-v)
                DB_VERBOSE=true
                shift
                ;;
            --help|-h)
                db_show_help
                exit 0
                ;;
            --version)
                echo "devboost $DB_VERSION"
                exit 0
                ;;
            *)
                db_log_error "Unknown option: $1"
                db_show_help
                exit 1
                ;;
        esac
    done
}

db_show_help() {
    cat << EOF
devboost - Bootstrap a modern dev environment

Usage: devboost [COMMAND] [OPTIONS]

Commands:
  apply      Converge machine to config (default)
  plan       Show actions without changing anything
  doctor     Check prerequisites, PATHs, shells, conflicting files
  uninstall  Remove managed files/blocks (leaves user custom files untouched)

Options:
  --config FILE    Config file path (default: ~/.devboost.yaml)
  --dry-run        Show what would be done without making changes
  --verbose, -v    Enable verbose output
  --help, -h       Show this help message
  --version        Show version

EOF
}

coreMain() {
    # Parse command first (before flags)
    local cmd="apply"
    if [[ $# -gt 0 ]] && [[ "$1" =~ ^(apply|plan|doctor|uninstall)$ ]]; then
        cmd="$1"
        shift
    fi
    
    db_parse_flags "$@"
    
    # Override subcommand if it was set via flags (shouldn't happen, but just in case)
    DB_SUBCOMMAND="$cmd"
    
    db_log_info "devboost $DB_VERSION - $DB_SUBCOMMAND"
    
    # Check config version compatibility
    local config_version=$(db_yaml_get '.version' '')
    if [[ -n "$config_version" ]]; then
        local config_major=$(echo "$config_version" | cut -d. -f1)
        local script_major=$(echo "$DB_VERSION" | cut -d. -f1)
        
        if [[ "$config_major" -lt "$script_major" ]]; then
            db_log_warn "Config file version ($config_version) is older than script version ($DB_VERSION)"
            db_log_warn "Please review CHANGELOG.md for breaking changes"
        fi
    fi
    
    # Ensure backup directory exists
    db_ensure_dir "$DB_BACKUP_DIR"
    
    # Detect OS
    db_detect_os
    
    # Load modules (already sourced, just register)
    db_load_modules
    
    case "$DB_SUBCOMMAND" in
        apply)
            db_run_apply
            db_log_success "Configuration applied!"
            ;;
        plan)
            DB_DRY_RUN=true
            db_run_plan
            db_log_info "Plan complete"
            ;;
        doctor)
            db_run_doctor
            # Also run general diagnostics
            db_log_info "OS: $DB_OS"
            db_log_info "Current shell: $SHELL"
            db_command_exists zsh && db_log_success "zsh: found" || db_log_error "zsh: not found"
            db_command_exists git && db_log_success "git: found" || db_log_error "git: not found"
            db_command_exists curl && db_log_success "curl: found" || db_log_error "curl: not found"
            ;;
        uninstall)
            db_run_uninstall
            ;;
        *)
            db_die "Unknown command: $DB_SUBCOMMAND"
            ;;
    esac
}

db_run_uninstall() {
    db_log_warn "Uninstalling devboost managed files..."
    
    # Remove .zshrc.devboost
    local include_file=$(db_yaml_get '.zsh.include_file' "$HOME/.zshrc.devboost")
    if [[ -f "$include_file" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $include_file"
        else
            rm -f "$include_file"
            db_log_success "Removed: $include_file"
        fi
    fi
    
    # Remove devboost block from .zshrc
    local zshrc="${HOME}/.zshrc"
    if [[ -f "$zshrc" ]] && grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
        db_remove_block "$zshrc" "# >>> devboost include start" "# <<< devboost include end"
    fi
    
    # Remove devboost block from .tmux.conf
    local tmux_conf=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    if [[ -f "$tmux_conf" ]] && grep -q "# >>> devboost tmux start" "$tmux_conf" 2>/dev/null; then
        db_remove_block "$tmux_conf" "# >>> devboost tmux start" "# <<< devboost tmux end"
    fi
    
    # Remove direnvrc
    local direnvrc=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    if [[ -f "$direnvrc" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $direnvrc"
        else
            db_backup_file "$direnvrc"
            rm -f "$direnvrc"
            db_log_success "Removed: $direnvrc"
        fi
    fi
    
    # Remove state file
    if [[ -f "$DB_STATE_FILE" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $DB_STATE_FILE"
        else
            rm -f "$DB_STATE_FILE"
            db_log_success "Removed: $DB_STATE_FILE"
        fi
    fi
    
    db_log_info "Uninstall complete. Backups are preserved in: $DB_BACKUP_DIR"
    db_log_info "Note: Packages, znap, TPM, and mise toolchains are not removed."
}


# === Modules ===
# Package installation module

db_module_pkg_register() {
    db_register_module "pkg" \
        "db_module_pkg_plan" \
        "db_module_pkg_apply"
}

db_module_pkg_plan() {
    local base_pkgs=$(db_yaml_get_list '.packages.base[]')
    if [[ -n "$base_pkgs" ]]; then
        db_log_info "Would install packages: $base_pkgs"
    fi
}

db_module_pkg_apply() {
    # Map package names to OS-specific names
    declare -A pkg_map_darwin=(
        [zsh]="zsh"
        [zoxide]="zoxide"
        [fzf]="fzf"
        [ripgrep]="ripgrep"
        [fd]="fd"
        [bat]="bat"
        [eza]="eza"
        [jq]="jq"
        [yq]="yq"
        [git-delta]="git-delta"
        [lazygit]="lazygit"
        [direnv]="direnv"
        [mise]="mise"
        [atuin]="atuin"
        [starship]="starship"
        [tmux]="tmux"
        [dust]="dust"
        [duf]="duf"
        [procs]="procs"
    )
    
    declare -A pkg_map_ubuntu=(
        [zsh]="zsh"
        [zoxide]="zoxide"
        [fzf]="fzf"
        [ripgrep]="ripgrep"
        [fd]="fd-find"
        [bat]="bat"
        [eza]="eza"
        [jq]="jq"
        [yq]="yq"
        [git-delta]="git-delta"
        [lazygit]="lazygit"
        [direnv]="direnv"
        [mise]="mise"
        [atuin]="atuin"
        [starship]="starship"
        [tmux]="tmux"
        [dust]="dust"
        [duf]="duf"
        [procs]="procs"
    )
    
    declare -A pkg_map_fedora=(
        [zsh]="zsh"
        [zoxide]="zoxide"
        [fzf]="fzf"
        [ripgrep]="ripgrep"
        [fd]="fd-find"
        [bat]="bat"
        [eza]="eza"
        [jq]="jq"
        [yq]="yq"
        [git-delta]="git-delta"
        [lazygit]="lazygit"
        [direnv]="direnv"
        [mise]="mise"
        [atuin]="atuin"
        [starship]="starship"
        [tmux]="tmux"
        [dust]="dust"
        [duf]="duf"
        [procs]="procs"
    )
    
    declare -A pkg_map_arch=(
        [zsh]="zsh"
        [zoxide]="zoxide"
        [fzf]="fzf"
        [ripgrep]="ripgrep"
        [fd]="fd"
        [bat]="bat"
        [eza]="eza"
        [jq]="jq"
        [yq]="yq"
        [git-delta]="git-delta"
        [lazygit]="lazygit"
        [direnv]="direnv"
        [mise]="mise"
        [atuin]="atuin"
        [starship]="starship"
        [tmux]="tmux"
        [dust]="dust"
        [duf]="duf"
        [procs]="procs"
    )
    
    # Get base packages from config
    local base_pkgs_str=$(db_yaml_get_list '.packages.base[]')
    if [[ -z "$base_pkgs_str" ]]; then
        # Default packages
        base_pkgs_str="zsh zoxide fzf ripgrep fd bat eza jq yq git-delta lazygit direnv mise atuin starship tmux dust duf procs"
    fi
    
    # Convert to array
    read -ra base_pkgs <<< "$base_pkgs_str"
    
    # Map packages based on OS
    local mapped_pkgs=()
    case "$DB_OS" in
        darwin)
            for pkg in "${base_pkgs[@]}"; do
                mapped_pkgs+=("${pkg_map_darwin[$pkg]:-$pkg}")
            done
            ;;
        linux-ubuntu)
            for pkg in "${base_pkgs[@]}"; do
                mapped_pkgs+=("${pkg_map_ubuntu[$pkg]:-$pkg}")
            done
            ;;
        linux-fedora)
            for pkg in "${base_pkgs[@]}"; do
                mapped_pkgs+=("${pkg_map_fedora[$pkg]:-$pkg}")
            done
            ;;
        linux-arch)
            for pkg in "${base_pkgs[@]}"; do
                mapped_pkgs+=("${pkg_map_arch[$pkg]:-$pkg}")
            done
            ;;
    esac
    
    db_install_packages "${mapped_pkgs[@]}"
}


# Znap (zsh plugin manager) module

db_module_znap_register() {
    db_register_module "znap" \
        "db_module_znap_plan" \
        "db_module_znap_apply"
}

db_module_znap_plan() {
    local znap_path=$(db_yaml_get '.zsh.znap_path' "$HOME/.zsh-snap")
    if [[ ! -d "$znap_path" ]]; then
        db_log_info "Would install znap to: $znap_path"
    fi
}

db_module_znap_apply() {
    local znap_path=$(db_yaml_get '.zsh.znap_path' "$HOME/.zsh-snap")
    local znap_git=$(db_yaml_get '.zsh.znap_git' "https://github.com/marlonrichert/zsh-snap.git")
    
    if [[ -d "$znap_path" ]]; then
        db_log_verbose "Znap already installed at: $znap_path"
        return 0
    fi
    
    db_log_info "Installing znap..."
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would clone znap to: $znap_path"
        return 0
    fi
    
    db_ensure_dir "$(dirname "$znap_path")"
    git clone --depth 1 "$znap_git" "$znap_path" || {
        db_log_error "Failed to install znap"
        return 1
    }
    db_log_success "Installed znap"
}


# Zsh configuration module

db_module_zsh_register() {
    db_register_module "zsh" \
        "db_module_zsh_plan" \
        "db_module_zsh_apply"
}

db_module_zsh_plan() {
    local enable=$(db_yaml_get '.zsh.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local include_file=$(db_yaml_get '.zsh.include_file' "$HOME/.zshrc.devboost")
    local zshrc="${HOME}/.zshrc"
    
    if [[ ! -f "$include_file" ]]; then
        db_log_info "Would create: $include_file"
    fi
    
    if [[ ! -f "$zshrc" ]] || ! grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
        db_log_info "Would inject devboost include block into: $zshrc"
    fi
}

db_render_zsh_devboost() {
    local znap_path=$(db_yaml_get '.zsh.znap_path' "$HOME/.zsh-snap")
    local enable_starship=$(db_yaml_get '.prompt.enable_starship' 'true')
    local starship_config=$(db_yaml_get '.prompt.starship_config' "$HOME/.config/starship.toml")
    local use_atuin=$(db_yaml_get '.zsh.history.use_atuin' 'true')
    local fzf_enable=$(db_yaml_get '.zsh.fzf.enable' 'true')
    local fzf_files=$(db_yaml_get '.zsh.fzf.default_command_files' 'fd --type f --hidden --follow --exclude .git')
    local fzf_dirs=$(db_yaml_get '.zsh.fzf.default_command_dirs' 'fd --type d --hidden --follow --exclude .git')
    local enable_mise=$(db_yaml_get '.toolchains.enable_mise' 'true')
    local enable_direnv=$(db_yaml_get '.direnv.enable' 'true')
    local clicolor=$(db_yaml_get '.aesthetics.clicolor' 'true')
    local lsc_colours=$(db_yaml_get '.aesthetics.lsc_colours' 'ExFxCxDxBxegedabagacad')
    local aliases_enable=$(db_yaml_get '.zsh.aliases.enable' 'true')
    
    cat << EOF
# Generated by devboost - DO NOT EDIT MANUALLY
export EDITOR="nvim"
export LANG="en_US.UTF-8"

setopt HIST_IGNORE_ALL_DUPS HIST_REDUCE_BLANKS SHARE_HISTORY INC_APPEND_HISTORY
autoload -Uz compinit && compinit -u
setopt AUTO_CD NO_BEEP

# znap
source "${znap_path}/znap.zsh"

# prompt
EOF
    
    if [[ "$enable_starship" == "true" ]]; then
        echo "export STARSHIP_CONFIG=\"${starship_config}\""
        echo 'eval "$(starship init zsh)"'
    fi
    
    cat << 'EOF'

# plugins
znap source zsh-users/zsh-autosuggestions
znap source zsh-users/zsh-syntax-highlighting

# nav/search/history
eval "$(zoxide init zsh)"
EOF
    
    if [[ "$use_atuin" == "true" ]]; then
        echo 'eval "$(atuin init zsh)"'
    fi
    
    if [[ "$fzf_enable" == "true" ]]; then
        echo 'eval "$(fzf --zsh 2>/dev/null || /opt/homebrew/bin/fzf --zsh 2>/dev/null || true)"'
        echo "export FZF_DEFAULT_COMMAND='${fzf_files}'"
        echo 'export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"'
        echo "export FZF_ALT_C_COMMAND='${fzf_dirs}'"
    fi
    
    cat << EOF

# toolchains & per-project env
EOF
    
    if [[ "$enable_mise" == "true" ]]; then
        echo 'eval "$(mise activate zsh)"'
    fi
    
    if [[ "$enable_direnv" == "true" ]]; then
        echo 'eval "$(direnv hook zsh)"'
    fi
    
    cat << EOF

# aesthetics
EOF
    
    if [[ "$clicolor" == "true" ]]; then
        echo 'export CLICOLOR=1'
    fi
    
    echo "export LSCOLORS=\"${lsc_colours}\""
    
    cat << EOF

# aliases
EOF
    
    if [[ "$aliases_enable" == "true" ]]; then
        cat << EOF
alias ls='eza -alg --git --group --time-style=relative'
alias cat='bat -pp'
alias grep='rg'
alias find='fd'
alias du='dust'
alias df='duf'
alias ps='procs'
alias lg='lazygit'
alias tm='tmux attach -t main || tmux new -s main'
alias please='sudo $(fc -ln -1)'
EOF
    fi
}

db_module_zsh_apply() {
    local enable=$(db_yaml_get '.zsh.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local include_file=$(db_yaml_get '.zsh.include_file' "$HOME/.zshrc.devboost")
    local zshrc="${HOME}/.zshrc"
    
    # Generate and write .zshrc.devboost
    local content=$(db_render_zsh_devboost)
    db_write_file "$include_file" "$content"
    
    # Inject include block into .zshrc
    local include_block="# >>> devboost include start
[ -f \"\$HOME/.zshrc.devboost\" ] && source \"\$HOME/.zshrc.devboost\"
# <<< devboost include end
"
    
    if [[ ! -f "$zshrc" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would create: $zshrc"
        else
            echo "$include_block" > "$zshrc"
            db_log_success "Created: $zshrc"
        fi
    elif ! grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would append include block to: $zshrc"
        else
            db_backup_file "$zshrc"
            echo "" >> "$zshrc"
            echo "$include_block" >> "$zshrc"
            db_log_success "Injected include block into: $zshrc"
        fi
    else
        db_log_verbose "Include block already present in: $zshrc"
    fi
}


# Starship prompt module

db_module_starship_register() {
    db_register_module "starship" \
        "db_module_starship_plan" \
        "db_module_starship_apply"
}

db_module_starship_plan() {
    local enable=$(db_yaml_get '.prompt.enable_starship' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(db_yaml_get '.prompt.starship_config' "$HOME/.config/starship.toml")
    if [[ ! -f "$starship_config" ]]; then
        db_log_info "Would create starship config: $starship_config"
    fi
}

db_render_starship_config() {
    cat << 'EOF'
add_newline = false
command_timeout = 700

[character]
success_symbol = "[❯](bold green)"
error_symbol   = "[❯](bold red)"

[directory]
truncation_length = 3
style = "bold blue"

[git_branch]
symbol = ""
style = "bold yellow"

[git_status]
style = "bold red"
format = '([\[$all_status\]]($style))'

[nodejs]
symbol = ""
style = "green"

[python]
symbol = ""
style = "yellow"

[rust]
symbol = ""
style = "red"

[package]
disabled = true
EOF
}

db_module_starship_apply() {
    local enable=$(db_yaml_get '.prompt.enable_starship' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(db_yaml_get '.prompt.starship_config' "$HOME/.config/starship.toml")
    local config_dir=$(dirname "$starship_config")
    
    db_ensure_dir "$config_dir"
    
    local content=$(db_render_starship_config)
    db_write_file "$starship_config" "$content"
}


# Tmux configuration module

db_module_tmux_register() {
    db_register_module "tmux" \
        "db_module_tmux_plan" \
        "db_module_tmux_apply"
}

db_module_tmux_plan() {
    local enable=$(db_yaml_get '.tmux.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    local conf_file=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    
    if [[ ! -d "$tpm_path" ]]; then
        db_log_info "Would install TPM to: $tpm_path"
    fi
    
    if [[ ! -f "$conf_file" ]] || ! grep -q "# >>> devboost tmux start" "$conf_file" 2>/dev/null; then
        db_log_info "Would inject tmux config block into: $conf_file"
    fi
}

db_render_tmux_block() {
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    local base_index=$(db_yaml_get '.tmux.settings.base_index' '1')
    local pane_base_index=$(db_yaml_get '.tmux.settings.pane_base_index' '1')
    local mouse=$(db_yaml_get '.tmux.settings.mouse' 'true')
    local history_limit=$(db_yaml_get '.tmux.settings.history_limit' '50000')
    local escape_time=$(db_yaml_get '.tmux.settings.escape_time' '0')
    local focus_events=$(db_yaml_get '.tmux.settings.focus_events' 'true')
    local continuum_restore=$(db_yaml_get '.tmux.settings.continuum_restore' 'true')
    local resurrect_capture=$(db_yaml_get '.tmux.settings.resurrect_capture_pane_contents' 'true')
    
    local mouse_val="on"
    [[ "$mouse" != "true" ]] && mouse_val="off"
    local focus_val="on"
    [[ "$focus_events" != "true" ]] && focus_val="off"
    local continuum_val="on"
    [[ "$continuum_restore" != "true" ]] && continuum_val="off"
    local resurrect_val="on"
    [[ "$resurrect_capture" != "true" ]] && resurrect_val="off"
    
    cat << EOF
# >>> devboost tmux start
set -g base-index ${base_index}
setw -g pane-base-index ${pane_base_index}
set -g mouse ${mouse_val}
set -g history-limit ${history_limit}
set -s escape-time ${escape_time}
set -g focus-events ${focus_val}
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @plugin 'tmux-plugins/tmux-yank'
set -g @plugin 'tmux-plugins/tmux-logging'
set -g @continuum-restore '${continuum_val}'
set -g @resurrect-capture-pane-contents '${resurrect_val}'
run '${tpm_path}/tpm'
# <<< devboost tmux end
EOF
}

db_module_tmux_apply() {
    local enable=$(db_yaml_get '.tmux.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local conf_file=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    
    # Install TPM
    if [[ ! -d "$tpm_path" ]]; then
        db_log_info "Installing TPM..."
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would clone TPM to: $tpm_path"
        else
            db_ensure_dir "$(dirname "$tpm_path")"
            git clone https://github.com/tmux-plugins/tpm "$tpm_path" || {
                db_log_error "Failed to install TPM"
                return 1
            }
            db_log_success "Installed TPM"
        fi
    fi
    
    # Generate and inject tmux config block
    local block=$(db_render_tmux_block)
    db_upsert_block "$conf_file" "# >>> devboost tmux start" "# <<< devboost tmux end" "$block"
    
    # Install/update plugins (only if not dry-run and tmux is available)
    if [[ "${DB_DRY_RUN:-false}" != "true" ]] && db_command_exists tmux; then
        local tmux_control=$(db_yaml_get '.system.tmux_control_mode' 'true')
        if [[ "$tmux_control" == "true" ]]; then
            # Use CLI mode
            db_log_info "Installing tmux plugins via CLI..."
            "$tpm_path/bindings/install_plugins" &>/dev/null || true
            "$tpm_path/bindings/update_plugins" all &>/dev/null || true
        else
            # Normal mode - plugins will install on next tmux session
            db_log_info "Tmux plugins will install on next tmux session (run 'prefix + I' in tmux)"
        fi
    fi
}


# Mise (toolchain manager) module

db_module_mise_register() {
    db_register_module "mise" \
        "db_module_mise_plan" \
        "db_module_mise_apply"
}

db_module_mise_plan() {
    local enable=$(db_yaml_get '.toolchains.enable_mise' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    db_log_info "Would configure mise toolchains"
}

db_module_mise_apply() {
    local enable=$(db_yaml_get '.toolchains.enable_mise' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    if ! db_command_exists mise; then
        db_log_warn "mise not found, skipping toolchain setup"
        return 0
    fi
    
    db_log_info "Configuring mise toolchains..."
    
    local node=$(db_yaml_get '.toolchains.globals.node' 'lts')
    local python=$(db_yaml_get '.toolchains.globals.python' '3.12')
    local go=$(db_yaml_get '.toolchains.globals.go' '1.23')
    local rust=$(db_yaml_get '.toolchains.globals.rust' 'stable')
    local deno=$(db_yaml_get '.toolchains.globals.deno' 'latest')
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would run: mise use -g node@${node} python@${python} go@${go} rust@${rust} deno@${deno}"
        db_log_info "Would run: mise install"
    else
        mise use -g "node@${node}" "python@${python}" "go@${go}" "rust@${rust}" "deno@${deno}" 2>/dev/null || true
        mise install 2>/dev/null || db_log_warn "Some toolchains may not be available"
        db_log_success "Configured mise toolchains"
    fi
}


# Direnv module

db_module_direnv_register() {
    db_register_module "direnv" \
        "db_module_direnv_plan" \
        "db_module_direnv_apply"
}

db_module_direnv_plan() {
    local enable=$(db_yaml_get '.direnv.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    if [[ ! -f "$rc_path" ]]; then
        db_log_info "Would create: $rc_path"
    fi
}

db_module_direnv_apply() {
    local enable=$(db_yaml_get '.direnv.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    local content=$(db_yaml_get '.direnv.content' '')
    
    # Use default if content is empty
    if [[ -z "$content" ]]; then
        content="use_mise() { eval \"\$(mise activate direnv)\"; }"
    fi
    
    db_write_file "$rc_path" "$content"
}


# Git delta configuration module

db_module_git_register() {
    db_register_module "git" \
        "db_module_git_plan" \
        "db_module_git_apply"
}

db_module_git_plan() {
    local enable=$(db_yaml_get '.git.delta.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    db_log_info "Would configure git delta"
}

db_module_git_apply() {
    local enable=$(db_yaml_get '.git.delta.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    if ! db_command_exists git; then
        db_log_warn "git not found, skipping delta config"
        return 0
    fi
    
    db_log_info "Configuring git delta..."
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would set git config delta settings"
    else
        git config --global core.pager delta || true
        git config --global interactive.diffFilter 'delta --color-only' || true
        git config --global delta.navigate "$(db_yaml_get '.git.delta.navigate' 'true')" || true
        git config --global delta.line-numbers "$(db_yaml_get '.git.delta.line_numbers' 'true')" || true
        db_log_success "Configured git delta"
    fi
}


# Services module (atuin daemon, etc.)

db_module_services_register() {
    db_register_module "services" \
        "db_module_services_plan" \
        "db_module_services_apply"
}

db_module_services_plan() {
    local use_atuin=$(db_yaml_get '.zsh.history.use_atuin' 'true')
    if [[ "$use_atuin" != "true" ]]; then
        return 0
    fi
    
    db_log_info "Would start atuin daemon"
}

db_module_services_apply() {
    local use_atuin=$(db_yaml_get '.zsh.history.use_atuin' 'true')
    if [[ "$use_atuin" != "true" ]]; then
        return 0
    fi
    
    if ! db_command_exists atuin; then
        db_log_warn "atuin not found, skipping service setup"
        return 0
    fi
    
    if [[ "$DB_OS" == "darwin" ]]; then
        if db_command_exists brew; then
            if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
                db_log_info "Would start atuin service via brew"
            else
                brew services start atuin 2>/dev/null || db_log_verbose "Atuin service may already be running"
            fi
        fi
    else
        # Linux - suggest systemd or manual start
        db_log_info "On Linux, ensure atuin daemon is running (systemd user service or manual start)"
    fi
}


# === Module Registration ===
db_load_modules() {
    # Register all modules
    db_module_pkg_register
    db_module_znap_register
    db_module_zsh_register
    db_module_starship_register
    db_module_tmux_register
    db_module_mise_register
    db_module_direnv_register
    db_module_git_register
    db_module_services_register
    
    db_log_verbose "Loaded ${#DB_MODULE_NAMES[@]} modules"
}

# === Main Execution ===
# Run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    coreMain "$@"
fi
