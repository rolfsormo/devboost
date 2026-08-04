# oh-my-zsh removal + migration helper
#
# This command replicates oh-my-zsh's own tools/uninstall.sh (removes
# ~/.oh-my-zsh, renames the current ~/.zshrc to
# ~/.zshrc.omz-uninstalled-<timestamp>, and restores ~/.zshrc.pre-oh-my-zsh
# if that pre-install snapshot exists) and then recovers any customization
# the user made *after* installing oh-my-zsh (aliases, PATH tweaks, exports)
# that would otherwise be stranded in the timestamped backup.
#
# Because the restore step always makes ~/.zshrc identical to the pre-install
# base (when a base exists), recovery reduces to: lines present in the
# timestamped backup but absent from the base are the user's additions —
# append them, after stripping oh-my-zsh's own template lines (ZSH_THEME,
# plugins=, source $ZSH/oh-my-zsh.sh, etc., which differ from the base too
# but aren't user content). There's no real 3-way conflict to resolve here
# since "current" and "base" always match going in.
#
# This is its own explicit subcommand — separate from apply/plan/doctor/
# uninstall — precisely because it removes a directory and rewrites .zshrc;
# it never runs as a side effect of anything else, and requires --yes.

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

# Replicates oh-my-zsh's tools/uninstall.sh: remove ~/.oh-my-zsh, rename the
# current ~/.zshrc to a timestamped backup, and restore ~/.zshrc.pre-oh-my-zsh
# if that pre-install snapshot exists.
_db_omz_uninstall() {
    local omz_dir="$HOME/.oh-my-zsh"
    local zshrc="$HOME/.zshrc"
    local base="$HOME/.zshrc.pre-oh-my-zsh"

    if [[ ! -d "$omz_dir" ]]; then
        db_log_verbose "No ~/.oh-my-zsh found — already removed or never installed."
        return 0
    fi

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would remove: $omz_dir"
        [[ -f "$zshrc" ]] && db_log_info "Would rename $zshrc to a timestamped .omz-uninstalled-* backup"
        [[ -f "$base" ]] && db_log_info "Would restore $base to $zshrc"
        return 0
    fi

    rm -rf "$omz_dir"
    db_log_success "Removed: $omz_dir"

    if [[ -f "$zshrc" ]]; then
        local saved="$HOME/.zshrc.omz-uninstalled-$(date +%Y-%m-%d_%H-%M-%S)"
        mv "$zshrc" "$saved"
        db_log_info "Renamed $zshrc to: $saved"
    fi

    if [[ -f "$base" ]]; then
        mv "$base" "$zshrc"
        db_log_success "Restored pre-oh-my-zsh config to: $zshrc"
    else
        db_log_info "No ~/.zshrc.pre-oh-my-zsh found — nothing to restore."
    fi
}

# Writes to stdout the lines in $other that are not present anywhere in $base,
# after stripping oh-my-zsh template lines from $other. Order-preserving,
# duplicate-preserving (does not dedupe repeated lines within $other itself).
_db_omz_lines_only_in() {
    local other="$1" base="$2"
    local stripped_other
    stripped_other=$(mktemp)
    _db_omz_strip_template "$other" > "$stripped_other"

    if [[ -n "$base" ]] && [[ -f "$base" ]]; then
        grep -vFxf "$base" "$stripped_other" 2>/dev/null || true
    else
        cat "$stripped_other"
    fi
    rm -f "$stripped_other"
}

db_run_migrate_from_oh_my_zsh() {
    local zshrc="$HOME/.zshrc"
    local base="$HOME/.zshrc.pre-oh-my-zsh"

    if [[ "${DB_DRY_RUN:-false}" != "true" ]] && [[ "${DB_OMZ_YES:-false}" != "true" ]]; then
        db_log_error "This removes ~/.oh-my-zsh and rewrites ~/.zshrc — pass --yes to confirm."
        db_log_info "Preview first with: devboost migrate-from-oh-my-zsh --dry-run"
        return 1
    fi

    # _db_omz_uninstall moves $base to $zshrc (replicating oh-my-zsh's own
    # uninstaller), consuming it — so snapshot its content first, since we
    # still need it below to tell the base's own lines apart from the user's
    # genuine post-install additions in the timestamped backup.
    local base_snapshot=""
    if [[ -f "$base" ]] && [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
        base_snapshot=$(mktemp)
        cp "$base" "$base_snapshot"
    fi

    _db_omz_uninstall

    local uninstalled
    uninstalled=$(_db_omz_find_uninstalled_backup)

    if [[ -z "$uninstalled" ]]; then
        [[ -n "$base_snapshot" ]] && rm -f "$base_snapshot"
        if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
            db_log_info "No existing ~/.zshrc.omz-uninstalled-* backup yet — nothing further to recover in a dry-run."
            return 0
        fi
        db_log_error "No ~/.zshrc.omz-uninstalled-* backup found — nothing to recover."
        return 1
    fi
    db_log_info "Found uninstall backup: $uninstalled"

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        if [[ -f "$base" ]]; then
            db_log_info "Would recover your additions from '$(basename "$uninstalled")' not already in '$(basename "$base")'"
        else
            db_log_info "Would recover your additions from '$(basename "$uninstalled")' (no pre-install base to compare against)"
        fi
        db_log_info "Would append them to: $zshrc"
        return 0
    fi

    local additions
    additions=$(_db_omz_lines_only_in "$uninstalled" "$base_snapshot")
    rm -f "$base_snapshot"

    if [[ -z "$additions" ]]; then
        db_log_success "No post-install customizations found beyond oh-my-zsh's own template — nothing to recover."
        db_log_info "Review $zshrc, then run 'devboost apply' to add devboost's include block."
        return 0
    fi

    db_backup_file "$zshrc"
    {
        if [[ -f "$zshrc" ]]; then
            cat "$zshrc"
            echo ""
        fi
        echo "$additions"
    } > "${zshrc}.devboost-omz-tmp"
    mv "${zshrc}.devboost-omz-tmp" "$zshrc"

    db_log_success "Recovered your customizations into: $zshrc"
    db_log_info "Review the result, then run 'devboost apply' to add devboost's include block."
    return 0
}
