# Tmux configuration module

db_module_tmux_register() {
    db_register_module "tmux" \
        "db_module_tmux_plan" \
        "db_module_tmux_apply"
}

db_module_tmux_plan() {
    local enable=$(db_yaml_get '.tmux.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    local conf_file=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    
    if [[ ! -d "$tpm_path" ]]; then
        db_log_info "Would install TPM to: $tpm_path"
    fi
    
    if [[ ! -f "$conf_file" ]] || ! grep -q "# >>> devboost tmux start" "$conf_file" 2>/dev/null; then
        db_log_info "Would inject tmux config block into: $conf_file"
    fi
}

db_render_tmux_block() {
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    local base_index=$(db_yaml_get '.tmux.settings.base_index' '1')
    local pane_base_index=$(db_yaml_get '.tmux.settings.pane_base_index' '1')
    local mouse=$(db_yaml_get '.tmux.settings.mouse' 'true')
    local history_limit=$(db_yaml_get '.tmux.settings.history_limit' '50000')
    local escape_time=$(db_yaml_get '.tmux.settings.escape_time' '0')
    local focus_events=$(db_yaml_get '.tmux.settings.focus_events' 'true')
    local continuum_restore=$(db_yaml_get '.tmux.settings.continuum_restore' 'true')
    local resurrect_capture=$(db_yaml_get '.tmux.settings.resurrect_capture_pane_contents' 'true')
    
    local mouse_val="on"
    [[ "$mouse" != "true" ]] && mouse_val="off"
    local focus_val="on"
    [[ "$focus_events" != "true" ]] && focus_val="off"
    local continuum_val="on"
    [[ "$continuum_restore" != "true" ]] && continuum_val="off"
    local resurrect_val="on"
    [[ "$resurrect_capture" != "true" ]] && resurrect_val="off"
    
    cat << EOF
# >>> devboost tmux start
set -g base-index ${base_index}
setw -g pane-base-index ${pane_base_index}
set -g mouse ${mouse_val}
set -g history-limit ${history_limit}
set -s escape-time ${escape_time}
set -g focus-events ${focus_val}
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @plugin 'tmux-plugins/tmux-yank'
set -g @plugin 'tmux-plugins/tmux-logging'
set -g @continuum-restore '${continuum_val}'
set -g @resurrect-capture-pane-contents '${resurrect_val}'
run '${tpm_path}/tpm'
# <<< devboost tmux end
EOF
}

db_module_tmux_apply() {
    local enable=$(db_yaml_get '.tmux.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local conf_file=$(db_yaml_get '.tmux.conf_file' "$HOME/.tmux.conf")
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    
    # Install TPM
    if [[ ! -d "$tpm_path" ]]; then
        db_log_info "Installing TPM..."
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would clone TPM to: $tpm_path"
        else
            db_ensure_dir "$(dirname "$tpm_path")"
            git clone https://github.com/tmux-plugins/tpm "$tpm_path" || {
                db_log_error "Failed to install TPM"
                return 1
            }
            db_log_success "Installed TPM"
        fi
    fi
    
    # Generate and inject tmux config block
    local block=$(db_render_tmux_block)
    db_upsert_block "$conf_file" "# >>> devboost tmux start" "# <<< devboost tmux end" "$block"
    
    # Install/update plugins (only if not dry-run and tmux is available)
    if [[ "${DB_DRY_RUN:-false}" != "true" ]] && db_command_exists tmux; then
        local tmux_control=$(db_yaml_get '.system.tmux_control_mode' 'true')
        if [[ "$tmux_control" == "true" ]]; then
            # Use CLI mode
            db_log_info "Installing tmux plugins via CLI..."
            "$tpm_path/bindings/install_plugins" &>/dev/null || true
            "$tpm_path/bindings/update_plugins" all &>/dev/null || true
        else
            # Normal mode - plugins will install on next tmux session
            db_log_info "Tmux plugins will install on next tmux session (run 'prefix + I' in tmux)"
        fi
    fi
}

