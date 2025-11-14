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

