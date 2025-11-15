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

# Helper function to map package names (bash 3.x compatible)
_db_pkg_map() {
    local os="$1" pkg="$2"
    case "$os" in
        darwin)
            case "$pkg" in
                zsh) echo "zsh" ;;
                zoxide) echo "zoxide" ;;
                fzf) echo "fzf" ;;
                ripgrep) echo "ripgrep" ;;
                fd) echo "fd" ;;
                bat) echo "bat" ;;
                eza) echo "eza" ;;
                jq) echo "jq" ;;
                yq) echo "yq" ;;
                git-delta) echo "git-delta" ;;
                lazygit) echo "lazygit" ;;
                direnv) echo "direnv" ;;
                mise) echo "mise" ;;
                atuin) echo "atuin" ;;
                starship) echo "starship" ;;
                tmux) echo "tmux" ;;
                dust) echo "dust" ;;
                duf) echo "duf" ;;
                procs) echo "procs" ;;
                *) echo "$pkg" ;;
            esac
            ;;
        linux-ubuntu)
            case "$pkg" in
                zsh) echo "zsh" ;;
                zoxide) echo "zoxide" ;;
                fzf) echo "fzf" ;;
                ripgrep) echo "ripgrep" ;;
                fd) echo "fd-find" ;;
                bat) echo "bat" ;;
                eza) echo "eza" ;;
                jq) echo "jq" ;;
                yq) echo "yq" ;;
                git-delta) echo "git-delta" ;;
                lazygit) echo "lazygit" ;;
                direnv) echo "direnv" ;;
                mise) echo "mise" ;;
                atuin) echo "atuin" ;;
                starship) echo "starship" ;;
                tmux) echo "tmux" ;;
                dust) echo "dust" ;;
                duf) echo "duf" ;;
                procs) echo "procs" ;;
                *) echo "$pkg" ;;
            esac
            ;;
        linux-fedora)
            case "$pkg" in
                zsh) echo "zsh" ;;
                zoxide) echo "zoxide" ;;
                fzf) echo "fzf" ;;
                ripgrep) echo "ripgrep" ;;
                fd) echo "fd-find" ;;
                bat) echo "bat" ;;
                eza) echo "eza" ;;
                jq) echo "jq" ;;
                yq) echo "yq" ;;
                git-delta) echo "git-delta" ;;
                lazygit) echo "lazygit" ;;
                direnv) echo "direnv" ;;
                mise) echo "mise" ;;
                atuin) echo "atuin" ;;
                starship) echo "starship" ;;
                tmux) echo "tmux" ;;
                dust) echo "dust" ;;
                duf) echo "duf" ;;
                procs) echo "procs" ;;
                *) echo "$pkg" ;;
            esac
            ;;
        linux-arch)
            case "$pkg" in
                zsh) echo "zsh" ;;
                zoxide) echo "zoxide" ;;
                fzf) echo "fzf" ;;
                ripgrep) echo "ripgrep" ;;
                fd) echo "fd" ;;
                bat) echo "bat" ;;
                eza) echo "eza" ;;
                jq) echo "jq" ;;
                yq) echo "yq" ;;
                git-delta) echo "git-delta" ;;
                lazygit) echo "lazygit" ;;
                direnv) echo "direnv" ;;
                mise) echo "mise" ;;
                atuin) echo "atuin" ;;
                starship) echo "starship" ;;
                tmux) echo "tmux" ;;
                dust) echo "dust" ;;
                duf) echo "duf" ;;
                procs) echo "procs" ;;
                *) echo "$pkg" ;;
            esac
            ;;
        *)
            echo "$pkg"
            ;;
    esac
}

db_module_pkg_apply() {
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
    for pkg in "${base_pkgs[@]}"; do
        mapped_pkgs+=("$(_db_pkg_map "$DB_OS" "$pkg")")
    done
    
    db_install_packages "${mapped_pkgs[@]}"
}

