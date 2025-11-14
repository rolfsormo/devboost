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

