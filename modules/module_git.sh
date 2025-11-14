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

