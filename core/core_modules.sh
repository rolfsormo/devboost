# Module registry system

DB_MODULE_NAMES=()
declare -A DB_MODULE_PLAN_FUNC
declare -A DB_MODULE_APPLY_FUNC
declare -A DB_MODULE_DOCTOR_FUNC

db_register_module() {
    local name="$1" plan="$2" apply="$3" doctor="${4:-}"
    DB_MODULE_NAMES+=("$name")
    DB_MODULE_PLAN_FUNC["$name"]="$plan"
    DB_MODULE_APPLY_FUNC["$name"]="$apply"
    if [[ -n "$doctor" ]]; then
        DB_MODULE_DOCTOR_FUNC["$name"]="$doctor"
    fi
    db_log_verbose "Registered module: $name"
}

# db_load_modules() is defined after all modules are loaded (in build output)

db_run_plan() {
    db_log_info "Planning changes..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Planning module: $m"
        "${DB_MODULE_PLAN_FUNC[$m]}" || true
    done
}

db_run_apply() {
    db_log_info "Applying configuration..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        db_log_verbose "Applying module: $m"
        "${DB_MODULE_APPLY_FUNC[$m]}" || true
    done
}

db_run_doctor() {
    db_log_info "Running diagnostics..."
    for m in "${DB_MODULE_NAMES[@]}"; do
        if [[ -n "${DB_MODULE_DOCTOR_FUNC[$m]:-}" ]]; then
            db_log_verbose "Checking module: $m"
            "${DB_MODULE_DOCTOR_FUNC[$m]}" || true
        fi
    done
}

