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

