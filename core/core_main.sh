# Main entry point and CLI

DB_VERSION="1.0.0"
DB_SUBCOMMAND="apply"
DB_DRY_RUN=false
DB_VERBOSE=false
DB_BACKUP_DIR="${HOME}/.devboost/backups"
DB_STATE_FILE="${HOME}/.devboost.state.json"

db_parse_flags() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            apply|plan|doctor|uninstall)
                DB_SUBCOMMAND="$1"
                shift
                ;;
            --config)
                DB_CONFIG_PATH="$2"
                shift 2
                ;;
            --dry-run)
                DB_DRY_RUN=true
                shift
                ;;
            --verbose|-v)
                DB_VERBOSE=true
                shift
                ;;
            --help|-h)
                db_show_help
                exit 0
                ;;
            --version)
                echo "devboost $DB_VERSION"
                exit 0
                ;;
            *)
                db_log_error "Unknown option: $1"
                db_show_help
                exit 1
                ;;
        esac
    done
}

db_show_help() {
    cat << EOF
devboost - Bootstrap a modern dev environment

Usage: devboost [COMMAND] [OPTIONS]

Commands:
  apply      Converge machine to config (default)
  plan       Show actions without changing anything
  doctor     Check prerequisites, PATHs, shells, conflicting files
  uninstall  Remove managed files/blocks (leaves user custom files untouched)

Options:
  --config FILE    Config file path (default: ~/.devboost.yaml)
  --dry-run        Show what would be done without making changes
  --verbose, -v    Enable verbose output
  --help, -h       Show this help message
  --version        Show version

EOF
}

coreMain() {
    # Parse command first (before flags)
    local cmd="apply"
    if [[ $# -gt 0 ]] && [[ "$1" =~ ^(apply|plan|doctor|uninstall)$ ]]; then
        cmd="$1"
        shift
    fi
    
    db_parse_flags "$@"
    
    # Override subcommand if it was set via flags (shouldn't happen, but just in case)
    DB_SUBCOMMAND="$cmd"
    
    db_log_info "devboost $DB_VERSION - $DB_SUBCOMMAND"
    
    # Check config version compatibility
    local config_version=$(db_yaml_get '.version' '')
    if [[ -n "$config_version" ]]; then
        local config_major=$(echo "$config_version" | cut -d. -f1)
        local script_major=$(echo "$DB_VERSION" | cut -d. -f1)
        
        if [[ "$config_major" -lt "$script_major" ]]; then
            db_log_warn "Config file version ($config_version) is older than script version ($DB_VERSION)"
            db_log_warn "Please review CHANGELOG.md for breaking changes"
        fi
    fi
    
    # Ensure backup directory exists
    db_ensure_dir "$DB_BACKUP_DIR"
    
    # Detect OS
    db_detect_os
    
    # Load modules (already sourced, just register)
    db_load_modules
    
    case "$DB_SUBCOMMAND" in
        apply)
            db_run_apply
            db_log_success "Configuration applied!"
            ;;
        plan)
            DB_DRY_RUN=true
            db_run_plan
            db_log_info "Plan complete"
            ;;
        doctor)
            db_run_doctor
            # Also run general diagnostics
            db_log_info "OS: $DB_OS"
            db_log_info "Current shell: $SHELL"
            db_command_exists zsh && db_log_success "zsh: found" || db_log_error "zsh: not found"
            db_command_exists git && db_log_success "git: found" || db_log_error "git: not found"
            db_command_exists curl && db_log_success "curl: found" || db_log_error "curl: not found"
            ;;
        uninstall)
            db_run_uninstall
            ;;
        *)
            db_die "Unknown command: $DB_SUBCOMMAND"
            ;;
    esac
}

db_run_uninstall() {
    db_log_warn "Uninstalling devboost managed files..."
    
    # Remove .zshrc.devboost
    local include_file=$(db_yaml_get '.zsh.include_file' "$HOME/.zshrc.devboost")
    if [[ -f "$include_file" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $include_file"
        else
            rm -f "$include_file"
            db_log_success "Removed: $include_file"
        fi
    fi
    
    # Remove devboost block from .zshrc
    local zshrc="${HOME}/.zshrc"
    if [[ -f "$zshrc" ]] && grep -q "# >>> devboost include start" "$zshrc" 2>/dev/null; then
        db_remove_block "$zshrc" "# >>> devboost include start" "# <<< devboost include end"
    fi
    
    # Remove devboost block from .tmux.conf
    local tmux_conf=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    if [[ -f "$tmux_conf" ]] && grep -q "# >>> devboost tmux start" "$tmux_conf" 2>/dev/null; then
        db_remove_block "$tmux_conf" "# >>> devboost tmux start" "# <<< devboost tmux end"
    fi
    
    # Remove direnvrc
    local direnvrc=$(db_yaml_get '.direnv.rc_path' "$HOME/.direnvrc")
    if [[ -f "$direnvrc" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $direnvrc"
        else
            db_backup_file "$direnvrc"
            rm -f "$direnvrc"
            db_log_success "Removed: $direnvrc"
        fi
    fi
    
    # Remove state file
    if [[ -f "$DB_STATE_FILE" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would remove: $DB_STATE_FILE"
        else
            rm -f "$DB_STATE_FILE"
            db_log_success "Removed: $DB_STATE_FILE"
        fi
    fi
    
    db_log_info "Uninstall complete. Backups are preserved in: $DB_BACKUP_DIR"
    db_log_info "Note: Packages, znap, TPM, and mise toolchains are not removed."
}

