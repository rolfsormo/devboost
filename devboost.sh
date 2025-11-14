#!/usr/bin/env bash
# devboost - Bootstrap a modern dev environment
# Idempotent, config-driven, non-destructive

set -euo pipefail

# Version
VERSION="1.0.0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Defaults
CONFIG_FILE="${HOME}/.devboost.yaml"
DRY_RUN=false
VERBOSE=false
SUBCOMMAND="apply"
STATE_FILE="${HOME}/.devboost.state.json"
BACKUP_DIR="${HOME}/.devboost/backups"

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[verbose]${NC} $*"
    fi
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            apply|plan|doctor|uninstall)
                SUBCOMMAND="$1"
                shift
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            --version)
                echo "devboost $VERSION"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
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

# OS detection
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ -f /etc/debian_version ]]; then
        echo "debian"
    elif [[ -f /etc/fedora-release ]]; then
        echo "fedora"
    elif [[ -f /etc/arch-release ]]; then
        echo "arch"
    else
        echo "unknown"
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Ensure directory exists
ensure_dir() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would create directory: $1"
        return 0
    fi
    mkdir -p "$1"
}

# Backup file
backup_file() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        return 0
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${BACKUP_DIR}/${timestamp}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would backup: $file -> ${backup_path}/$(basename "$file")"
        return 0
    fi
    
    ensure_dir "$backup_path"
    cp "$file" "${backup_path}/$(basename "$file")"
    log_verbose "Backed up: $file"
}

# Parse YAML (simplified - handles basic nested structures)
parse_yaml() {
    local file="$1"
    local prefix="$2"
    local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
    
    sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p" "$file" |
    awk -F$fs '{
        indent = length($1)/2;
        vname[indent] = $2;
        for (i in vname) {if (i > indent) {delete vname[i]}}
        if (length($3) > 0) {
            vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])(".")}
            printf("%s%s%s=\"%s\"\n", "'"$prefix"'", vn, $2, $3);
        }
    }'
}

# Load config with defaults
load_config() {
    local config_file="$1"
    
    # Default config
    declare -gA CONFIG
    CONFIG[system.tmux_control_mode]="true"
    CONFIG[system.package_manager]="auto"
    CONFIG[zsh.enable]="true"
    CONFIG[zsh.plugin_manager]="znap"
    CONFIG[zsh.znap_git]="https://github.com/marlonrichert/zsh-snap.git"
    CONFIG[zsh.znap_path]="${HOME}/.zsh-snap"
    CONFIG[zsh.include_file]="${HOME}/.zshrc.devboost"
    CONFIG[zsh.history.use_atuin]="true"
    CONFIG[zsh.fzf.enable]="true"
    CONFIG[zsh.fzf.default_command_files]="fd --type f --hidden --follow --exclude .git"
    CONFIG[zsh.fzf.default_command_dirs]="fd --type d --hidden --follow --exclude .git"
    CONFIG[zsh.aliases.enable]="true"
    CONFIG[prompt.enable_starship]="true"
    CONFIG[prompt.starship_config]="${HOME}/.config/starship.toml"
    CONFIG[tmux.enable]="true"
    CONFIG[tmux.tpm_path]="${HOME}/.tmux/plugins/tpm"
    CONFIG[tmux.conf_file]="${HOME}/.tmux.conf"
    CONFIG[tmux.settings.base_index]="1"
    CONFIG[tmux.settings.pane_base_index]="1"
    CONFIG[tmux.settings.mouse]="true"
    CONFIG[tmux.settings.history_limit]="50000"
    CONFIG[tmux.settings.escape_time]="0"
    CONFIG[tmux.settings.focus_events]="true"
    CONFIG[tmux.settings.continuum_restore]="true"
    CONFIG[tmux.settings.resurrect_capture_pane_contents]="true"
    CONFIG[toolchains.enable_mise]="true"
    CONFIG[toolchains.globals.node]="lts"
    CONFIG[toolchains.globals.python]="3.12"
    CONFIG[toolchains.globals.go]="1.23"
    CONFIG[toolchains.globals.rust]="stable"
    CONFIG[toolchains.globals.deno]="latest"
    CONFIG[direnv.enable]="true"
    CONFIG[direnv.rc_path]="${HOME}/.direnvrc"
    CONFIG[git.delta.enable]="true"
    CONFIG[git.delta.line_numbers]="true"
    CONFIG[git.delta.navigate]="true"
    CONFIG[aesthetics.lsc_colours]="ExFxCxDxBxegedabagacad"
    CONFIG[aesthetics.clicolor]="true"
    
    # Default packages
    CONFIG[packages.base]="zsh zoxide fzf ripgrep fd bat eza jq yq git-delta lazygit direnv mise atuin starship tmux"
    
    # Load from file if exists
    if [[ -f "$config_file" ]]; then
        log_verbose "Loading config from: $config_file"
        
        # Try using yq if available (preferred)
        if command_exists yq; then
            # Use yq to flatten YAML to dot-notation
            while IFS='=' read -r key value; do
                # Remove quotes from value
                value="${value#\"}"
                value="${value%\"}"
                value="${value#\'}"
                value="${value%\'}"
                if [[ -n "$key" && -n "$value" ]]; then
                    CONFIG["$key"]="$value"
                fi
            done < <(yq eval '.. | select(type == "!!str" or type == "!!int" or type == "!!bool") | (path | join(".")) + "=" + .' "$config_file" 2>/dev/null | sed 's/^"//;s/"$//')
        # Fallback to Python if available
        elif command_exists python3 && python3 -c "import yaml" 2>/dev/null; then
            local temp_script=$(mktemp)
            cat > "$temp_script" << 'PYEOF'
import yaml
import sys

try:
    with open(sys.argv[1], 'r') as f:
        data = yaml.safe_load(f) or {}
    
    def flatten(d, parent_key='', sep='.'):
        items = []
        for k, v in d.items():
            new_key = f"{parent_key}{sep}{k}" if parent_key else k
            if isinstance(v, dict):
                items.extend(flatten(v, new_key, sep=sep).items())
            elif isinstance(v, list):
                items.append((new_key, ' '.join(str(x) for x in v)))
            else:
                items.append((new_key, str(v)))
        return dict(items)
    
    flat = flatten(data)
    for k, v in flat.items():
        print(f"{k}={v}")
except Exception:
    sys.exit(1)
PYEOF
            while IFS='=' read -r key value; do
                if [[ -n "$key" && -n "$value" ]]; then
                    CONFIG["$key"]="$value"
                fi
            done < <(python3 "$temp_script" "$config_file" 2>/dev/null)
            rm -f "$temp_script"
        else
            # Basic YAML parsing fallback
            log_warn "yq or python3 not found, using basic YAML parsing"
            local current_section=""
            while IFS= read -r line || [[ -n "$line" ]]; do
                # Skip comments and empty lines
                [[ "$line" =~ ^[[:space:]]*# ]] && continue
                [[ -z "${line// }" ]] && continue
                
                # Detect section headers
                if [[ "$line" =~ ^([a-zA-Z_]+):[[:space:]]*$ ]]; then
                    current_section="${BASH_REMATCH[1]}"
                    continue
                fi
                
                # Parse key: value pairs
                if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*(.+)$ ]]; then
                    # List item
                    local item="${BASH_REMATCH[1]}"
                    item="${item#\"}"
                    item="${item%\"}"
                    if [[ -n "$current_section" ]]; then
                        local existing="${CONFIG[packages.${current_section}]:-}"
                        CONFIG[packages.${current_section}]="${existing} ${item}"
                    fi
                elif [[ "$line" =~ ^[[:space:]]*([^:]+):[[:space:]]*(.+)$ ]]; then
                    local key="${BASH_REMATCH[1]// /}"
                    local value="${BASH_REMATCH[2]// /}"
                    value="${value#\"}"
                    value="${value%\"}"
                    value="${value#\'}"
                    value="${value%\'}"
                    
                    if [[ -n "$current_section" ]]; then
                        CONFIG["${current_section}.${key}"]="$value"
                    else
                        CONFIG["$key"]="$value"
                    fi
                fi
            done < "$config_file"
        fi
    else
        log_info "Config file not found, using defaults: $config_file"
    fi
}

# Get config value
get_config() {
    local key="$1"
    echo "${CONFIG[$key]:-}"
}

# State management
read_state() {
    if [[ -f "$STATE_FILE" ]]; then
        log_verbose "Reading state from: $STATE_FILE"
        # Simple JSON reading (would use jq in production)
        return 0
    fi
    return 1
}

write_state() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would write state to: $STATE_FILE"
        return 0
    fi
    ensure_dir "$(dirname "$STATE_FILE")"
    echo "{}" > "$STATE_FILE"
    log_verbose "Wrote state to: $STATE_FILE"
}

# Module: Package Manager
module_pkg_plan() {
    local os=$(detect_os)
    log_info "Planning package installations for OS: $os"
    
    # This is a simplified version - would check each package
    return 0
}

module_pkg_apply() {
    local os=$(detect_os)
    log_info "Installing packages for OS: $os"
    
    case "$os" in
        macos)
            if ! command_exists brew; then
                log_info "Installing Homebrew..."
                if [[ "$DRY_RUN" != "true" ]]; then
                    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
                else
                    log_info "Would install Homebrew"
                fi
            fi
            
            local packages=$(get_config "packages.base")
            for pkg in $packages; do
                if ! brew list "$pkg" &>/dev/null; then
                    log_info "Installing: $pkg"
                    if [[ "$DRY_RUN" != "true" ]]; then
                        brew install "$pkg" || log_warn "Failed to install $pkg"
                    else
                        log_info "Would install: $pkg"
                    fi
                else
                    log_verbose "Already installed: $pkg"
                fi
            done
            ;;
        debian)
            if [[ "$DRY_RUN" != "true" ]]; then
                sudo apt-get update -qq
            fi
            
            # Package mapping for Debian/Ubuntu
            declare -A pkg_map=(
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
            )
            
            local packages=$(get_config "packages.base")
            for pkg in $packages; do
                local apt_name="${pkg_map[$pkg]:-$pkg}"
                if ! dpkg -l | grep -q "^ii[[:space:]]*${apt_name}[[:space:]]"; then
                    log_info "Installing: $apt_name"
                    if [[ "$DRY_RUN" != "true" ]]; then
                        sudo apt-get install -y "$apt_name" || log_warn "Failed to install $apt_name"
                    else
                        log_info "Would install: $apt_name"
                    fi
                else
                    log_verbose "Already installed: $apt_name"
                fi
            done
            ;;
        fedora)
            # Package mapping for Fedora
            declare -A pkg_map=(
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
            )
            
            local packages=$(get_config "packages.base")
            for pkg in $packages; do
                local dnf_name="${pkg_map[$pkg]:-$pkg}"
                if ! rpm -q "$dnf_name" &>/dev/null; then
                    log_info "Installing: $dnf_name"
                    if [[ "$DRY_RUN" != "true" ]]; then
                        sudo dnf install -y "$dnf_name" || log_warn "Failed to install $dnf_name"
                    else
                        log_info "Would install: $dnf_name"
                    fi
                else
                    log_verbose "Already installed: $dnf_name"
                fi
            done
            ;;
        arch)
            # Package mapping for Arch
            declare -A pkg_map=(
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
            )
            
            local packages=$(get_config "packages.base")
            for pkg in $packages; do
                local pacman_name="${pkg_map[$pkg]:-$pkg}"
                if ! pacman -Qi "$pacman_name" &>/dev/null; then
                    log_info "Installing: $pacman_name"
                    if [[ "$DRY_RUN" != "true" ]]; then
                        sudo pacman -S --noconfirm "$pacman_name" || log_warn "Failed to install $pacman_name"
                    else
                        log_info "Would install: $pacman_name"
                    fi
                else
                    log_verbose "Already installed: $pacman_name"
                fi
            done
            ;;
    esac
}

# Module: Znap
module_znap_plan() {
    local znap_path=$(get_config "zsh.znap_path")
    if [[ ! -d "$znap_path" ]]; then
        log_info "Would install znap to: $znap_path"
        return 1
    fi
    return 0
}

module_znap_apply() {
    local znap_path=$(get_config "zsh.znap_path")
    local znap_git=$(get_config "zsh.znap_git")
    
    if [[ -d "$znap_path" ]]; then
        log_verbose "Znap already installed at: $znap_path"
        return 0
    fi
    
    log_info "Installing znap..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would clone znap to: $znap_path"
        return 0
    fi
    
    ensure_dir "$(dirname "$znap_path")"
    git clone --depth 1 "$znap_git" "$znap_path" || {
        log_error "Failed to install znap"
        return 1
    }
    log_success "Installed znap"
}

# Module: Zsh
module_zsh_plan() {
    local include_file=$(get_config "zsh.include_file")
    local zshrc="${HOME}/.zshrc"
    
    local changes=0
    
    # Check if include file needs to be created/updated
    if [[ ! -f "$include_file" ]]; then
        log_info "Would create: $include_file"
        changes=$((changes + 1))
    fi
    
    # Check if .zshrc needs include block
    if [[ -f "$zshrc" ]]; then
        if ! grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
            log_info "Would inject devboost include block into: $zshrc"
            changes=$((changes + 1))
        fi
    else
        log_info "Would create: $zshrc with include block"
        changes=$((changes + 1))
    fi
    
    return $changes
}

module_zsh_apply() {
    local include_file=$(get_config "zsh.include_file")
    local zshrc="${HOME}/.zshrc"
    local znap_path=$(get_config "zsh.znap_path")
    
    # Generate .zshrc.devboost content
    local content="# Generated by devboost - DO NOT EDIT MANUALLY
export EDITOR=\"nvim\"
export LANG=\"en_US.UTF-8\"

setopt HIST_IGNORE_ALL_DUPS HIST_REDUCE_BLANKS SHARE_HISTORY INC_APPEND_HISTORY
autoload -Uz compinit && compinit -u
setopt AUTO_CD NO_BEEP

# znap
source \"${znap_path}/znap.zsh\"

# prompt
"
    
    if [[ "$(get_config "prompt.enable_starship")" == "true" ]]; then
        local starship_config=$(get_config "prompt.starship_config")
        content+="export STARSHIP_CONFIG=\"${starship_config}\"
eval \"\$(starship init zsh)\"
"
    fi
    
    content+="
# plugins
znap source zsh-users/zsh-autosuggestions
znap source zsh-users/zsh-syntax-highlighting

# nav/search/history
eval \"\$(zoxide init zsh)\"
"
    
    if [[ "$(get_config "zsh.history.use_atuin")" == "true" ]]; then
        content+="eval \"\$(atuin init zsh)\"
"
    fi
    
    if [[ "$(get_config "zsh.fzf.enable")" == "true" ]]; then
        content+="eval \"\$(fzf --zsh 2>/dev/null || /opt/homebrew/bin/fzf --zsh 2>/dev/null || true)\"
"
        local fzf_files=$(get_config "zsh.fzf.default_command_files")
        local fzf_dirs=$(get_config "zsh.fzf.default_command_dirs")
        content+="export FZF_DEFAULT_COMMAND='${fzf_files}'
export FZF_CTRL_T_COMMAND=\"\$FZF_DEFAULT_COMMAND\"
export FZF_ALT_C_COMMAND='${fzf_dirs}'
"
    fi
    
    content+="
# toolchains & per-project env
"
    
    if [[ "$(get_config "toolchains.enable_mise")" == "true" ]]; then
        content+="eval \"\$(mise activate zsh)\"
"
    fi
    
    if [[ "$(get_config "direnv.enable")" == "true" ]]; then
        content+="eval \"\$(direnv hook zsh)\"
"
    fi
    
    content+="
# aesthetics
"
    
    if [[ "$(get_config "aesthetics.clicolor")" == "true" ]]; then
        content+="export CLICOLOR=1
"
    fi
    
    local lsc_colours=$(get_config "aesthetics.lsc_colours")
    content+="export LSCOLORS=\"${lsc_colours}\"

# aliases
"
    
    if [[ "$(get_config "zsh.aliases.enable")" == "true" ]]; then
        content+="alias ls='eza -alg --git --group --time-style=relative'
alias cat='bat -pp'
alias grep='rg'
alias find='fd'
alias du='dust'
alias df='duf'
alias ps='procs'
alias lg='lazygit'
alias tm='tmux attach -t main || tmux new -s main'
alias please='sudo \$(fc -ln -1)'
"
    fi
    
    # Write include file
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would write: $include_file"
        echo "$content"
    else
        backup_file "$include_file"
        echo "$content" > "$include_file"
        log_success "Wrote: $include_file"
    fi
    
    # Inject include block into .zshrc
    local include_block="# >>> devboost include start
[ -f \"\$HOME/.zshrc.devboost\" ] && source \"\$HOME/.zshrc.devboost\"
# <<< devboost include end
"
    
    if [[ ! -f "$zshrc" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would create: $zshrc"
        else
            echo "$include_block" > "$zshrc"
            log_success "Created: $zshrc"
        fi
    elif ! grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would append include block to: $zshrc"
        else
            backup_file "$zshrc"
            echo "" >> "$zshrc"
            echo "$include_block" >> "$zshrc"
            log_success "Injected include block into: $zshrc"
        fi
    else
        log_verbose "Include block already present in: $zshrc"
    fi
}

# Module: Starship
module_starship_plan() {
    if [[ "$(get_config "prompt.enable_starship")" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(get_config "prompt.starship_config")
    if [[ ! -f "$starship_config" ]]; then
        log_info "Would create starship config: $starship_config"
        return 1
    fi
    return 0
}

module_starship_apply() {
    if [[ "$(get_config "prompt.enable_starship")" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(get_config "prompt.starship_config")
    local config_dir=$(dirname "$starship_config")
    
    ensure_dir "$config_dir"
    
    local content="add_newline = false
command_timeout = 700

[character]
success_symbol = \"[❯](bold green)\"
error_symbol   = \"[❯](bold red)\"

[directory]
truncation_length = 3
style = \"bold blue\"

[git_branch]
symbol = \"\"
style = \"bold yellow\"

[git_status]
style = \"bold red\"
format = '([[\$all_status\$ahead_behind]](\$style) )'

[nodejs]
symbol = \"\"
style = \"green\"

[python]
symbol = \"\"
style = \"yellow\"

[rust]
symbol = \"\"
style = \"red\"

[package]
disabled = true
"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would write starship config: $starship_config"
        echo "$content"
    else
        backup_file "$starship_config"
        echo "$content" > "$starship_config"
        log_success "Wrote starship config: $starship_config"
    fi
}

# Module: Tmux
module_tmux_plan() {
    if [[ "$(get_config "tmux.enable")" != "true" ]]; then
        return 0
    fi
    
    local conf_file=$(get_config "tmux.conf_file")
    local tpm_path=$(get_config "tmux.tpm_path")
    
    local changes=0
    
    if [[ ! -d "$tpm_path" ]]; then
        log_info "Would install TPM to: $tpm_path"
        changes=$((changes + 1))
    fi
    
    if [[ ! -f "$conf_file" ]] || ! grep -q "# >>> devboost tmux start" "$conf_file" 2>/dev/null; then
        log_info "Would inject tmux config block into: $conf_file"
        changes=$((changes + 1))
    fi
    
    return $changes
}

module_tmux_apply() {
    if [[ "$(get_config "tmux.enable")" != "true" ]]; then
        return 0
    fi
    
    local conf_file=$(get_config "tmux.conf_file")
    local tpm_path=$(get_config "tmux.tpm_path")
    
    # Install TPM
    if [[ ! -d "$tpm_path" ]]; then
        log_info "Installing TPM..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would clone TPM to: $tpm_path"
        else
            ensure_dir "$(dirname "$tpm_path")"
            git clone https://github.com/tmux-plugins/tpm "$tpm_path" || {
                log_error "Failed to install TPM"
                return 1
            }
            log_success "Installed TPM"
        fi
    fi
    
    # Generate tmux config block
    local base_index=$(get_config "tmux.settings.base_index")
    local pane_base_index=$(get_config "tmux.settings.pane_base_index")
    local mouse=$(get_config "tmux.settings.mouse")
    local history_limit=$(get_config "tmux.settings.history_limit")
    local escape_time=$(get_config "tmux.settings.escape_time")
    local focus_events=$(get_config "tmux.settings.focus_events")
    local continuum_restore=$(get_config "tmux.settings.continuum_restore")
    local resurrect_capture=$(get_config "tmux.settings.resurrect_capture_pane_contents")
    
    local mouse_val="on"
    [[ "$mouse" != "true" ]] && mouse_val="off"
    local focus_val="on"
    [[ "$focus_events" != "true" ]] && focus_val="off"
    local continuum_val="on"
    [[ "$continuum_restore" != "true" ]] && continuum_val="off"
    local resurrect_val="on"
    [[ "$resurrect_capture" != "true" ]] && resurrect_val="off"
    
    local block="# >>> devboost tmux start
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
"
    
    # Inject or replace block in .tmux.conf
    if [[ ! -f "$conf_file" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "Would create: $conf_file"
        else
            echo "$block" > "$conf_file"
            log_success "Created: $conf_file"
        fi
    else
        if grep -q "# >>> devboost tmux start" "$conf_file" 2>/dev/null; then
            # Replace existing block
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would replace devboost block in: $conf_file"
            else
                backup_file "$conf_file"
                # Use awk to replace between markers
                local temp_file=$(mktemp)
                local block_file=$(mktemp)
                echo "$block" > "$block_file"
                awk -v block_file="$block_file" '
                    /# >>> devboost tmux start/ { 
                        in_block=1
                        while ((getline line < block_file) > 0) {
                            print line
                        }
                        close(block_file)
                        next 
                    }
                    /# <<< devboost tmux end/ { 
                        in_block=0
                        next 
                    }
                    !in_block { print }
                ' "$conf_file" > "$temp_file"
                mv "$temp_file" "$conf_file"
                rm -f "$block_file"
                log_success "Updated devboost block in: $conf_file"
            fi
        else
            # Append block
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would append devboost block to: $conf_file"
            else
                backup_file "$conf_file"
                echo "" >> "$conf_file"
                echo "$block" >> "$conf_file"
                log_success "Injected devboost block into: $conf_file"
            fi
        fi
    fi
    
    # Install/update plugins (only if not dry-run and tmux is available)
    if [[ "$DRY_RUN" != "true" ]] && command_exists tmux; then
        local tmux_control=$(get_config "system.tmux_control_mode")
        if [[ "$tmux_control" == "true" ]]; then
            # Use CLI mode
            log_info "Installing tmux plugins via CLI..."
            "$tpm_path/bindings/install_plugins" &>/dev/null || true
            "$tpm_path/bindings/update_plugins" all &>/dev/null || true
        else
            # Normal mode - plugins will install on next tmux session
            log_info "Tmux plugins will install on next tmux session (run 'prefix + I' in tmux)"
        fi
    fi
}

# Module: Mise
module_mise_plan() {
    if [[ "$(get_config "toolchains.enable_mise")" != "true" ]]; then
        return 0
    fi
    
    log_info "Would configure mise toolchains"
    return 1
}

module_mise_apply() {
    if [[ "$(get_config "toolchains.enable_mise")" != "true" ]]; then
        return 0
    fi
    
    if ! command_exists mise; then
        log_warn "mise not found, skipping toolchain setup"
        return 0
    fi
    
    log_info "Configuring mise toolchains..."
    
    local node=$(get_config "toolchains.globals.node")
    local python=$(get_config "toolchains.globals.python")
    local go=$(get_config "toolchains.globals.go")
    local rust=$(get_config "toolchains.globals.rust")
    local deno=$(get_config "toolchains.globals.deno")
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would run: mise use -g node@${node} python@${python} go@${go} rust@${rust} deno@${deno}"
        log_info "Would run: mise install"
    else
        mise use -g "node@${node}" "python@${python}" "go@${go}" "rust@${rust}" "deno@${deno}" 2>/dev/null || true
        mise install 2>/dev/null || log_warn "Some toolchains may not be available"
        log_success "Configured mise toolchains"
    fi
}

# Module: Direnv
module_direnv_plan() {
    if [[ "$(get_config "direnv.enable")" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(get_config "direnv.rc_path")
    if [[ ! -f "$rc_path" ]]; then
        log_info "Would create: $rc_path"
        return 1
    fi
    return 0
}

module_direnv_apply() {
    if [[ "$(get_config "direnv.enable")" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(get_config "direnv.rc_path")
    local content=$(get_config "direnv.content")
    
    # Use default if content is empty
    if [[ -z "$content" ]]; then
        content="use_mise() { eval \"\$(mise activate direnv)\"; }"
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would write: $rc_path"
        echo "$content"
    else
        backup_file "$rc_path"
        echo "$content" > "$rc_path"
        log_success "Wrote: $rc_path"
    fi
}

# Module: Git
module_git_plan() {
    if [[ "$(get_config "git.delta.enable")" != "true" ]]; then
        return 0
    fi
    
    log_info "Would configure git delta"
    return 1
}

module_git_apply() {
    if [[ "$(get_config "git.delta.enable")" != "true" ]]; then
        return 0
    fi
    
    if ! command_exists git; then
        log_warn "git not found, skipping delta config"
        return 0
    fi
    
    log_info "Configuring git delta..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would set git config delta settings"
    else
        git config --global core.pager delta || true
        git config --global interactive.diffFilter 'delta --color-only' || true
        git config --global delta.navigate "$(get_config "git.delta.navigate")" || true
        git config --global delta.line-numbers "$(get_config "git.delta.line_numbers")" || true
        log_success "Configured git delta"
    fi
}

# Module: Services
module_services_plan() {
    if [[ "$(get_config "zsh.history.use_atuin")" != "true" ]]; then
        return 0
    fi
    
    log_info "Would start atuin daemon"
    return 1
}

module_services_apply() {
    if [[ "$(get_config "zsh.history.use_atuin")" != "true" ]]; then
        return 0
    fi
    
    if ! command_exists atuin; then
        log_warn "atuin not found, skipping service setup"
        return 0
    fi
    
    local os=$(detect_os)
    
    if [[ "$os" == "macos" ]]; then
        if command_exists brew; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would start atuin service via brew"
            else
                brew services start atuin 2>/dev/null || log_verbose "Atuin service may already be running"
            fi
        fi
    else
        # Linux - suggest systemd or manual start
        log_info "On Linux, ensure atuin daemon is running (systemd user service or manual start)"
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    log_info "devboost $VERSION - $SUBCOMMAND"
    
    # Load config
    load_config "$CONFIG_FILE"
    
    # Ensure backup directory exists
    ensure_dir "$BACKUP_DIR"
    
    case "$SUBCOMMAND" in
        apply)
            log_info "Applying configuration..."
            module_pkg_apply
            module_znap_apply
            module_zsh_apply
            module_starship_apply
            module_tmux_apply
            module_mise_apply
            module_direnv_apply
            module_git_apply
            module_services_apply
            write_state
            log_success "Configuration applied!"
            ;;
        plan)
            DRY_RUN=true
            log_info "Planning changes (dry-run)..."
            module_pkg_plan
            module_znap_plan
            module_zsh_plan
            module_starship_plan
            module_tmux_plan
            module_mise_plan
            module_direnv_plan
            module_git_plan
            module_services_plan
            log_info "Plan complete"
            ;;
        doctor)
            log_info "Running diagnostics..."
            local os=$(detect_os)
            log_info "OS: $os"
            
            # Check prerequisites
            log_info "Checking prerequisites..."
            command_exists zsh && log_success "zsh: found" || log_error "zsh: not found"
            command_exists git && log_success "git: found" || log_error "git: not found"
            command_exists curl && log_success "curl: found" || log_error "curl: not found"
            
            if [[ "$os" == "macos" ]]; then
                command_exists brew && log_success "brew: found" || log_warn "brew: not found (will be installed)"
            fi
            
            # Check shell
            log_info "Current shell: $SHELL"
            if [[ "$SHELL" != *"zsh"* ]]; then
                log_warn "Not using zsh as default shell"
            fi
            
            # Check PATH
            log_info "Checking PATH..."
            echo "$PATH" | tr ':' '\n' | while read -r p; do
                [[ -d "$p" ]] && log_verbose "PATH: $p (exists)" || log_warn "PATH: $p (missing)"
            done
            
            # Check for conflicts
            log_info "Checking for conflicts..."
            [[ -f "${HOME}/.zshrc" ]] && log_info "Found: ~/.zshrc"
            [[ -f "${HOME}/.tmux.conf" ]] && log_info "Found: ~/.tmux.conf"
            ;;
        uninstall)
            log_warn "Uninstalling devboost managed files..."
            
            # Remove .zshrc.devboost
            local include_file=$(get_config "zsh.include_file")
            if [[ -f "$include_file" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "Would remove: $include_file"
                else
                    rm -f "$include_file"
                    log_success "Removed: $include_file"
                fi
            fi
            
            # Remove devboost block from .zshrc
            local zshrc="${HOME}/.zshrc"
            if [[ -f "$zshrc" ]] && grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "Would remove devboost block from: $zshrc"
                else
                    backup_file "$zshrc"
                    local temp_file=$(mktemp)
                    awk '
                        /# >>> devboost include start/ { in_block=1; next }
                        /# <<< devboost include end/ { in_block=0; next }
                        !in_block { print }
                    ' "$zshrc" > "$temp_file"
                    mv "$temp_file" "$zshrc"
                    log_success "Removed devboost block from: $zshrc"
                fi
            fi
            
            # Remove devboost block from .tmux.conf
            local tmux_conf=$(get_config "tmux.conf_file")
            if [[ -f "$tmux_conf" ]] && grep -q "# >>> devboost tmux start" "$tmux_conf" 2>/dev/null; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "Would remove devboost block from: $tmux_conf"
                else
                    backup_file "$tmux_conf"
                    local temp_file=$(mktemp)
                    awk '
                        /# >>> devboost tmux start/ { in_block=1; next }
                        /# <<< devboost tmux end/ { in_block=0; next }
                        !in_block { print }
                    ' "$tmux_conf" > "$temp_file"
                    mv "$temp_file" "$tmux_conf"
                    log_success "Removed devboost block from: $tmux_conf"
                fi
            fi
            
            # Remove starship config (optional - ask user?)
            local starship_config=$(get_config "prompt.starship_config")
            if [[ -f "$starship_config" ]]; then
                log_info "Starship config found: $starship_config (not removing - may contain user customizations)"
            fi
            
            # Remove direnvrc (optional)
            local direnvrc=$(get_config "direnv.rc_path")
            if [[ -f "$direnvrc" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "Would remove: $direnvrc"
                else
                    backup_file "$direnvrc"
                    rm -f "$direnvrc"
                    log_success "Removed: $direnvrc"
                fi
            fi
            
            # Remove state file
            if [[ -f "$STATE_FILE" ]]; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "Would remove: $STATE_FILE"
                else
                    rm -f "$STATE_FILE"
                    log_success "Removed: $STATE_FILE"
                fi
            fi
            
            log_info "Uninstall complete. Backups are preserved in: $BACKUP_DIR"
            log_info "Note: Packages, znap, TPM, and mise toolchains are not removed."
            ;;
        *)
            log_error "Unknown subcommand: $SUBCOMMAND"
            show_help
            exit 1
            ;;
    esac
}

# Run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi

