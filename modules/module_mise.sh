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

# Returns a space-separated list of globally installed npm packages (excluding npm itself).
_db_mise_npm_globals() {
    local node_bin
    node_bin=$(command -v node 2>/dev/null) || return 0
    local npm_bin
    npm_bin=$(dirname "$node_bin")/npm
    [[ -x "$npm_bin" ]] || return 0
    "$npm_bin" list -g --depth=0 --parseable 2>/dev/null \
        | awk -F/ 'NF>1 && $NF!="npm" && $NF!="corepack" {print $NF}'
}

# Returns the currently active mise-managed node version, or empty string.
_db_mise_current_node_version() {
    mise current node 2>/dev/null | tr -d '[:space:]' || true
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
    local python=$(db_yaml_get '.toolchains.globals.python' '3.14')
    local go=$(db_yaml_get '.toolchains.globals.go' '1.26')
    local rust=$(db_yaml_get '.toolchains.globals.rust' 'stable')
    local deno=$(db_yaml_get '.toolchains.globals.deno' 'lts')

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would run: mise use -g node@${node} python@${python} go@${go} rust@${rust} deno@${deno}"
        db_log_info "Would run: mise install"
        return 0
    fi

    # Capture current node version and its global npm packages before any upgrade.
    local prev_node_version
    prev_node_version=$(_db_mise_current_node_version)
    local npm_globals=()
    if [[ -n "$prev_node_version" ]] && db_command_exists node; then
        while IFS= read -r pkg; do
            [[ -n "$pkg" ]] && npm_globals+=("$pkg")
        done < <(_db_mise_npm_globals)
    fi

    mise use -g "node@${node}" "python@${python}" "go@${go}" "rust@${rust}" "deno@${deno}" 2>/dev/null || true
    mise install 2>/dev/null || db_log_warn "Some toolchains may not be available"
    db_log_success "Configured mise toolchains"

    # After upgrade, offer to migrate npm globals to the new node version.
    if [[ ${#npm_globals[@]} -gt 0 ]]; then
        local new_node_version
        new_node_version=$(_db_mise_current_node_version)
        if [[ "$prev_node_version" != "$new_node_version" ]]; then
            db_log_warn "Node upgraded: ${prev_node_version} → ${new_node_version}"
            db_log_warn "The following global npm packages were present in the old version:"
            for pkg in "${npm_globals[@]}"; do
                db_log_warn "  - $pkg"
            done
            db_log_warn "Note: any version pins or custom configuration for these packages will NOT be migrated."
            if db_confirm "Reinstall these packages into node@${new_node_version}?"; then
                local npm_bin
                npm_bin=$(dirname "$(command -v node)")/npm
                for pkg in "${npm_globals[@]}"; do
                    db_log_info "Installing $pkg..."
                    "$npm_bin" install -g "$pkg" 2>/dev/null \
                        && db_log_success "  ✓ $pkg" \
                        || db_log_warn "  ✗ $pkg (failed — install manually if needed)"
                done
            fi
        fi
    fi
}
