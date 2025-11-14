# Starship prompt module

db_module_starship_register() {
    db_register_module "starship" \
        "db_module_starship_plan" \
        "db_module_starship_apply"
}

db_module_starship_plan() {
    local enable=$(db_yaml_get '.prompt.enable_starship' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(db_yaml_get '.prompt.starship_config' "$HOME/.config/starship.toml")
    if [[ ! -f "$starship_config" ]]; then
        db_log_info "Would create starship config: $starship_config"
    fi
}

db_render_starship_config() {
    cat << 'EOF'
add_newline = false
command_timeout = 700

[character]
success_symbol = "[❯](bold green)"
error_symbol   = "[❯](bold red)"

[directory]
truncation_length = 3
style = "bold blue"

[git_branch]
symbol = ""
style = "bold yellow"

[git_status]
style = "bold red"
format = '([\[$all_status\]]($style))'

[nodejs]
symbol = ""
style = "green"

[python]
symbol = ""
style = "yellow"

[rust]
symbol = ""
style = "red"

[package]
disabled = true
EOF
}

db_module_starship_apply() {
    local enable=$(db_yaml_get '.prompt.enable_starship' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    local starship_config=$(db_yaml_get '.prompt.starship_config' "$HOME/.config/starship.toml")
    local config_dir=$(dirname "$starship_config")
    
    db_ensure_dir "$config_dir"
    
    local content=$(db_render_starship_config)
    db_write_file "$starship_config" "$content"
}

