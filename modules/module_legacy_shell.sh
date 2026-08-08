# Legacy shell tooling module
#
# Detects and safely retires shell tooling that duplicates what devboost's
# own modules already provide, when found in a pre-existing (non-devboost
# managed) ~/.zshrc or ~/.zprofile:
#   - zinit loading the same plugins znap (module_znap.sh) already loads
#     (zsh-autosuggestions, and a fast-syntax-highlighting/syntax-highlighting
#     fork of the same feature)
#   - asdf, alongside devboost's own mise (module_mise.sh)
#   - nvm's shell hook (in ~/.zprofile, login-shell-only — measured at
#     ~850-900ms per login shell via zprof, the dominant real-world startup
#     cost found on the investigation machine), alongside devboost's own mise
#
# Redundant lines are commented out in place (see core/core_legacy_shell.sh)
# rather than deleted, so the user can review/undo by hand. Actual deletion
# of marked lines happens only via the separate `devboost clean` command.

DB_LEGACY_ZINIT_ZNAP_ID="zinit-znap-dup"
DB_LEGACY_ASDF_MISE_ID="asdf-mise-dup"
DB_LEGACY_NVM_MISE_ID="nvm-mise-dup"

# Matches zinit lines loading plugins znap's default config already loads.
DB_LEGACY_ZINIT_DUP_PATTERN='^[[:space:]]*zinit (light|load)[^#]*(zsh-users/zsh-autosuggestions|zdharma-continuum/fast-syntax-highlighting|zsh-users/zsh-syntax-highlighting)'

# Matches the line sourcing asdf's shell integration.
DB_LEGACY_ASDF_SOURCE_PATTERN='(^|[[:space:]])\. .*/asdf\.sh([[:space:]]|$)'

# Matches lines sourcing nvm's shell hook or its bash-completion shim
# (e.g. `[ -s ".../nvm.sh" ] && \. ".../nvm.sh"`). Deliberately does not
# match a plain `export NVM_DIR=...` line — that's a harmless variable,
# not the expensive part. Matches on the quoted path alone (not the
# leading `\.`/`source` token) since backslash-escaped literals in this
# pattern are interpreted differently by grep -E vs awk's ERE dialect
# when passed through a shell variable — see core_legacy_shell.sh's
# _db_legacy_disable_lines, which runs the same pattern string through
# both. Keep future patterns here grep/awk-dialect-safe for that reason.
DB_LEGACY_NVM_SOURCE_PATTERN='"[^"]*/nvm(\.sh|/etc/bash_completion\.d/nvm)"'

db_module_legacy_shell_register() {
    db_register_module "legacy_shell" \
        "db_module_legacy_shell_plan" \
        "db_module_legacy_shell_apply" \
        "db_module_legacy_shell_doctor"
}

_db_legacy_shell_zshrc() {
    db_yaml_get '.legacy_shell.zshrc' "$HOME/.zshrc"
}

_db_legacy_shell_zprofile() {
    db_yaml_get '.legacy_shell.zprofile' "$HOME/.zprofile"
}

# True if `file` contains a live (unmarked) line matching `pattern` —
# i.e. one devboost hasn't already disabled for `migration_id`. A marker
# is prepended to, not a replacement of, the original line, so the
# pattern still matches inside an already-disabled line; excluding lines
# that start with the marker is what keeps doctor/plan from re-reporting
# something already fixed as still needing action.
_db_legacy_shell_unmarked_match_present() {
    local file="$1" pattern="$2" migration_id="$3"
    [[ -f "$file" ]] || return 1
    local marker="# devboost:disabled:${migration_id} "
    grep -E "$pattern" "$file" 2>/dev/null | grep -qv "^${marker}"
}

# True if zinit is loading a plugin that duplicates one of znap's.
_db_legacy_shell_zinit_dup_present() {
    _db_legacy_shell_unmarked_match_present "$1" "$DB_LEGACY_ZINIT_DUP_PATTERN" "$DB_LEGACY_ZINIT_ZNAP_ID"
}

# True if asdf is sourced (redundant with mise).
_db_legacy_shell_asdf_present() {
    _db_legacy_shell_unmarked_match_present "$1" "$DB_LEGACY_ASDF_SOURCE_PATTERN" "$DB_LEGACY_ASDF_MISE_ID"
}

# True if nvm's shell hook is sourced (redundant with mise).
_db_legacy_shell_nvm_present() {
    _db_legacy_shell_unmarked_match_present "$1" "$DB_LEGACY_NVM_SOURCE_PATTERN" "$DB_LEGACY_NVM_MISE_ID"
}

db_module_legacy_shell_doctor() {
    local enable
    enable=$(db_yaml_get '.legacy_shell.enable' 'true')
    [[ "$enable" == "true" ]] || return 0

    local zshrc
    zshrc=$(_db_legacy_shell_zshrc)

    if _db_legacy_shell_zinit_dup_present "$zshrc"; then
        db_log_warn "legacy_shell: zinit is loading plugin(s) that duplicate znap's — run 'devboost apply' to disable them"
    else
        db_log_success "legacy_shell: no zinit/znap plugin duplication found"
    fi

    if _db_legacy_shell_asdf_present "$zshrc"; then
        db_log_warn "legacy_shell: asdf is active alongside mise — run 'devboost apply' to disable it"
    else
        db_log_success "legacy_shell: no asdf/mise duplication found"
    fi

    local zprofile
    zprofile=$(_db_legacy_shell_zprofile)

    if _db_legacy_shell_nvm_present "$zprofile"; then
        db_log_warn "legacy_shell: nvm's shell hook is active alongside mise (login-shell startup cost) — run 'devboost apply' to disable it"
    else
        db_log_success "legacy_shell: no nvm/mise duplication found"
    fi
}

db_module_legacy_shell_plan() {
    local enable
    enable=$(db_yaml_get '.legacy_shell.enable' 'true')
    [[ "$enable" == "true" ]] || return 0

    local zshrc
    zshrc=$(_db_legacy_shell_zshrc)

    if _db_legacy_shell_zinit_dup_present "$zshrc"; then
        db_log_info "Would disable redundant zinit plugin line(s) in: $zshrc"
    fi
    if _db_legacy_shell_asdf_present "$zshrc"; then
        db_log_info "Would disable redundant asdf source line in: $zshrc"
    fi

    local zprofile
    zprofile=$(_db_legacy_shell_zprofile)
    if _db_legacy_shell_nvm_present "$zprofile"; then
        db_log_info "Would disable redundant nvm source line(s) in: $zprofile"
    fi
}

db_module_legacy_shell_apply() {
    local enable
    enable=$(db_yaml_get '.legacy_shell.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi

    local zshrc
    zshrc=$(_db_legacy_shell_zshrc)
    [[ -f "$zshrc" ]] || return 0

    if _db_legacy_shell_zinit_dup_present "$zshrc"; then
        _db_legacy_disable_lines "$zshrc" "$DB_LEGACY_ZINIT_DUP_PATTERN" "$DB_LEGACY_ZINIT_ZNAP_ID"
    fi

    if _db_legacy_shell_asdf_present "$zshrc"; then
        _db_legacy_disable_lines "$zshrc" "$DB_LEGACY_ASDF_SOURCE_PATTERN" "$DB_LEGACY_ASDF_MISE_ID"
    fi

    local zprofile
    zprofile=$(_db_legacy_shell_zprofile)
    if [[ -f "$zprofile" ]] && _db_legacy_shell_nvm_present "$zprofile"; then
        _db_legacy_disable_lines "$zprofile" "$DB_LEGACY_NVM_SOURCE_PATTERN" "$DB_LEGACY_NVM_MISE_ID"
    fi
}
