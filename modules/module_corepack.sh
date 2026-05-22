# Corepack module — enables pnpm/yarn shims bundled with Node.js

db_module_corepack_register() {
    db_register_module "corepack" \
        "db_module_corepack_plan" \
        "db_module_corepack_apply"
}

db_module_corepack_plan() {
    local enable=$(db_yaml_get '.toolchains.enable_mise' 'true')
    [[ "$enable" == "true" ]] || return 0
    db_log_info "Would run: corepack enable"
}

db_module_corepack_apply() {
    local enable=$(db_yaml_get '.toolchains.enable_mise' 'true')
    [[ "$enable" == "true" ]] || return 0

    if ! db_command_exists corepack; then
        db_log_warn "corepack not found (expected inside Node.js install), skipping"
        return 0
    fi

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would run: corepack enable"
        return 0
    fi

    corepack enable 2>/dev/null && db_log_success "corepack: pnpm/yarn shims enabled" \
        || db_log_warn "corepack enable failed — pnpm/yarn shims may be missing"
}
