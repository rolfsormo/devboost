# Direnv module

db_module_direnv_register() {
    db_register_module "direnv" \
        "db_module_direnv_plan" \
        "db_module_direnv_apply"
}

db_module_direnv_plan() {
    local enable=$(db_yaml_get '.direnv.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    if [[ ! -f "$rc_path" ]]; then
        db_log_info "Would create: $rc_path"
    fi
}

db_module_direnv_apply() {
    local enable=$(db_yaml_get '.direnv.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local rc_path=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    local content=$(db_yaml_get '.direnv.content' '')
    
    # Use default if content is empty
    if [[ -z "$content" ]]; then
        content="use_mise() { eval \"\$(mise activate direnv)\"; }"
    fi
    
    db_write_file "$rc_path" "$content"
}

