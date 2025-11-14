# File manipulation helpers

db_ensure_dir() {
    local dir="$1"
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would create directory: $dir"
        return 0
    fi
    mkdir -p "$dir"
}

db_backup_file() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        return 0
    fi
    
    local backup_dir="${DB_BACKUP_DIR:-$HOME/.devboost/backups}"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="${backup_dir}/${timestamp}"
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would backup: $file -> ${backup_path}/$(basename "$file")"
        return 0
    fi
    
    db_ensure_dir "$backup_path"
    cp "$file" "${backup_path}/$(basename "$file")"
    db_log_verbose "Backed up: $file"
}

db_write_file() {
    local file="$1"
    local content="$2"
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would write: $file"
        echo "$content"
    else
        db_backup_file "$file"
        echo "$content" > "$file"
        db_log_success "Wrote: $file"
    fi
}

db_upsert_block() {
    local file="$1"
    local start_marker="$2"
    local end_marker="$3"
    local new_content="$4"
    
    if [[ ! -f "$file" ]]; then
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would create: $file"
        else
            echo "$new_content" > "$file"
            db_log_success "Created: $file"
        fi
        return 0
    fi
    
    if grep -q "$start_marker" "$file" 2>/dev/null; then
        # Replace existing block
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would replace block in: $file"
        else
            db_backup_file "$file"
            local temp_file=$(mktemp)
            local block_file=$(mktemp)
            echo "$new_content" > "$block_file"
            awk -v start="$start_marker" -v end="$end_marker" -v block_file="$block_file" '
                $0 ~ start { 
                    in_block=1
                    while ((getline line < block_file) > 0) {
                        print line
                    }
                    close(block_file)
                    next 
                }
                $0 ~ end { 
                    in_block=0
                    next 
                }
                !in_block { print }
            ' "$file" > "$temp_file"
            mv "$temp_file" "$file"
            rm -f "$block_file"
            db_log_success "Updated block in: $file"
        fi
    else
        # Append block
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "Would append block to: $file"
        else
            db_backup_file "$file"
            echo "" >> "$file"
            echo "$new_content" >> "$file"
            db_log_success "Injected block into: $file"
        fi
    fi
}

db_remove_block() {
    local file="$1"
    local start_marker="$2"
    local end_marker="$3"
    
    if [[ ! -f "$file" ]] || ! grep -q "$start_marker" "$file" 2>/dev/null; then
        return 0
    fi
    
    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would remove block from: $file"
    else
        db_backup_file "$file"
        local temp_file=$(mktemp)
        awk -v start="$start_marker" -v end="$end_marker" '
            $0 ~ start { in_block=1; next }
            $0 ~ end { in_block=0; next }
            !in_block { print }
        ' "$file" > "$temp_file"
        mv "$temp_file" "$file"
        db_log_success "Removed block from: $file"
    fi
}

