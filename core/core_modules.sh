# Module registry system
# Uses bash 3.x compatible approach (no associative arrays)

DB_MODULE_NAMES=()

# Helper functions for bash 3.x compatibility (simulating associative arrays)
_db_module_set() {
    local var="$1" key="$2" value="$3"
    # Sanitize key to be a valid variable name
    key=$(echo "$key" | tr -cd '[:alnum:]_')
    eval "${var}_${key}=\"\$value\""
}

_db_module_get() {
    local var="$1" key="$2"
    # Sanitize key to be a valid variable name
    key=$(echo "$key" | tr -cd '[:alnum:]_')
    eval "echo \"\${${var}_${key}:-}\""
}

db_register_module() {
    local name="$1" plan="$2" apply="$3" doctor="${4:-}"
    DB_MODULE_NAMES+=("$name")
    _db_module_set "DB_MODULE_PLAN_FUNC" "$name" "$plan"
    _db_module_set "DB_MODULE_APPLY_FUNC" "$name" "$apply"
    if [[ -n "$doctor" ]]; then
        _db_module_set "DB_MODULE_DOCTOR_FUNC" "$name" "$doctor"
    fi
    db_log_verbose "Registered module: $name"
}

# db_load_modules() is defined after all modules are loaded (in build output)

db_run_plan() {
    db_log_info "Planning changes..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Planning module: $m"
        local func=$(_db_module_get "DB_MODULE_PLAN_FUNC" "$m")
        [[ -n "$func" ]] && "$func" || true
    done
}

db_run_apply() {
    db_log_info "Applying configuration..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Applying module: $m"
        local func=$(_db_module_get "DB_MODULE_APPLY_FUNC" "$m")
        [[ -n "$func" ]] && "$func" || true
    done
}

db_run_doctor() {
    db_log_info "Running diagnostics..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        local func=$(_db_module_get "DB_MODULE_DOCTOR_FUNC" "$m")
        if [[ -n "$func" ]]; then
            db_log_verbose "Checking module: $m"
            "$func" || true
        fi
    done
}

