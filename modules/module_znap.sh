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

