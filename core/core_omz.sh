# oh-my-zsh migration helper
#
# oh-my-zsh's own `uninstall_oh_my_zsh` removes ~/.oh-my-zsh and restores
# ~/.zshrc from ~/.zshrc.pre-oh-my-zsh (the pristine pre-install snapshot),
# renaming the file it replaces to ~/.zshrc.omz-uninstalled-<timestamp>.
# Any customization the user made *after* installing oh-my-zsh (aliases,
# PATH tweaks, exports) lives only in that timestamped file and is lost
# unless recovered.
#
# This command 3-way-merges those customizations back onto the restored
# .zshrc using `git merge-file`, after first stripping oh-my-zsh's own
# template lines (which differ from the pre-install base but are not user
# content) from the "other" side. Non-conflicting additions are merged
# automatically; anything ambiguous is left with <<<<<<< conflict markers
# for the user to resolve by hand. Nothing is auto-applied to the real
# ~/.zshrc without the user reviewing the result first.

# Lines matching these patterns are oh-my-zsh's own template scaffolding,
# not user content, even though their values are user-customized (e.g.
# ZSH_THEME). Matched against templates/zshrc.zsh-template upstream.
_db_omz_is_template_line() {
    local line="$1"
    case "$line" in
        '#'*) return 0 ;;
        '') return 0 ;;
        'export ZSH='*) return 0 ;;
        'ZSH_THEME='*) return 0 ;;
        'ZSH_THEME_RANDOM_CANDIDATES='*) return 0 ;;
        'CASE_SENSITIVE='*) return 0 ;;
        'HYPHEN_INSENSITIVE='*) return 0 ;;
        'DISABLE_MAGIC_FUNCTIONS='*) return 0 ;;
        'DISABLE_LS_COLORS='*) return 0 ;;
        'DISABLE_AUTO_TITLE='*) return 0 ;;
        'ENABLE_CORRECTION='*) return 0 ;;
        'COMPLETION_WAITING_DOTS='*) return 0 ;;
        'DISABLE_UNTRACKED_FILES_DIRTY='*) return 0 ;;
        'HIST_STAMPS='*) return 0 ;;
        'ZSH_CUSTOM='*) return 0 ;;
        'zstyle '*':omz:'*) return 0 ;;
        'plugins=('*) return 0 ;;
        'source $ZSH/oh-my-zsh.sh'*) return 0 ;;
        *) return 1 ;;
    esac
}

# Strip oh-my-zsh template lines from a file, writing the remainder to stdout.
_db_omz_strip_template() {
    local file="$1"
    local line
    while IFS= read -r line || [[ -n "$line" ]]; do
        _db_omz_is_template_line "$line" || echo "$line"
    done < "$file"
}

# Finds the most recently modified ~/.zshrc.omz-uninstalled-* backup, if any.
_db_omz_find_uninstalled_backup() {
    local candidate
    candidate=$(ls -t "$HOME"/.zshrc.omz-uninstalled-* 2>/dev/null | head -1) || true
    echo "$candidate"
}

db_run_migrate_from_oh_my_zsh() {
    local zshrc="$HOME/.zshrc"
    local base="$HOME/.zshrc.pre-oh-my-zsh"
    local uninstalled
    uninstalled=$(_db_omz_find_uninstalled_backup)

    if [[ -z "$uninstalled" ]]; then
        db_log_error "No ~/.zshrc.omz-uninstalled-* backup found."
        db_log_info "Run this after 'uninstall_oh_my_zsh' inside an oh-my-zsh shell — it creates that backup."
        return 1
    fi
    db_log_info "Found uninstall backup: $uninstalled"

    if [[ ! -f "$base" ]]; then
        db_log_warn "No ~/.zshrc.pre-oh-my-zsh found — oh-my-zsh's installer didn't save a pre-install snapshot."
        db_log_warn "Nothing to merge against. Review manually instead:"
        db_log_warn "  diff '$zshrc' '$uninstalled'"
        return 1
    fi

    if [[ ! -f "$zshrc" ]]; then
        db_log_error "No ~/.zshrc found to merge into."
        return 1
    fi

    if ! db_command_exists git; then
        db_log_error "git is required for this command (uses 'git merge-file')."
        return 1
    fi

    db_log_info "Stripping oh-my-zsh template lines from the uninstalled backup..."
    local stripped_other
    stripped_other=$(mktemp)
    _db_omz_strip_template "$uninstalled" > "$stripped_other"

    local stripped_base
    stripped_base=$(mktemp)
    _db_omz_strip_template "$base" > "$stripped_base"

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would 3-way merge: base=$(basename "$base") current=$(basename "$zshrc") other=$(basename "$uninstalled")"
        db_log_info "Would write result to: $zshrc (backup taken first)"
        rm -f "$stripped_other" "$stripped_base"
        return 0
    fi

    db_backup_file "$zshrc"

    local merged
    merged=$(mktemp)
    cp "$zshrc" "$merged"

    local merge_status=0
    git merge-file -L "current .zshrc" -L "pre-oh-my-zsh base" -L "your customizations" \
        "$merged" "$stripped_base" "$stripped_other" || merge_status=$?

    mv "$merged" "$zshrc"
    rm -f "$stripped_other" "$stripped_base"

    if [[ "$merge_status" -eq 0 ]]; then
        db_log_success "Merged customizations back into: $zshrc (no conflicts)"
        db_log_info "Review the result, then run 'devboost apply' to add devboost's include block."
        return 0
    fi

    db_log_warn "Merged with conflicts — resolve the <<<<<<< markers in: $zshrc"
    db_log_info "Your original (pre-merge) file was backed up to: $DB_BACKUP_DIR"
    return 1
}
