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
                        brew install "$pkg" || db_log_warn "Failed to install $pkg"
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
                        sudo apt-get install -y "$pkg" || db_log_warn "Failed to install $pkg"
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
                        sudo dnf install -y "$pkg" || db_log_warn "Failed to install $pkg"
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
                        sudo pacman -S --noconfirm "$pkg" || db_log_warn "Failed to install $pkg"
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

