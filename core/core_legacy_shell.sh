# Generic helpers for detecting and safely retiring legacy shell tooling
# that devboost's own modules have superseded (e.g. a pre-existing zinit
# setup duplicating znap's plugins, or asdf duplicating mise).
#
# Design (see AGENTS.md's non-destructive principle, applied to files
# devboost does not own):
#   - Redundant lines in a user-owned file (like ~/.zshrc) are never
#     deleted. They're commented out in place with a marker that both a
#     human and this tooling can recognize:
#       # devboost:disabled:<migration_id> <original line>
#     A human can undo this by hand (delete the marker prefix). If they
#     do, later runs must treat that as an explicit override and leave
#     the line alone.
#   - Redundant directories (like a pre-existing ~/.oh-my-zsh-style
#     install root) are moved aside into ~/.devboost/backups/, never
#     rm -rf'd.
#   - Every edit also gets a full pre/post file snapshot in
#     ~/.devboost/backups/ as an audit trail, independent of the marker.
#   - `devboost clean` (db_run_clean) is the opt-in, separate step that
#     actually deletes marked lines. It re-derives what to clean by
#     grepping the live file each run — no reliance on in-process state
#     from a prior apply — so it's idempotent and order-independent.

_DB_LEGACY_MARKER_PREFIX="# devboost:disabled:"

_db_legacy_marker_for() {
    local migration_id="$1"
    echo "${_DB_LEGACY_MARKER_PREFIX}${migration_id} "
}

# Hand-rolled snapshot with a caller-chosen suffix (db_backup_file's
# signature is fixed at one arg with no suffix hook).
_db_legacy_snapshot() {
    local file="$1" migration_id="$2" phase="$3"
    [[ -f "$file" ]] || return 0

    local backup_dir="${DB_BACKUP_DIR:-$HOME/.devboost/backups}"
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local dest="${backup_dir}/$(basename "$file").${phase}-${migration_id}-${timestamp}"

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would snapshot: $file -> $dest"
        return 0
    fi

    db_ensure_dir "$backup_dir"
    cp "$file" "$dest"
    db_log_verbose "Snapshotted ($phase): $file -> $dest"
}

# True (0) if `file` currently contains a line matching `grep_pattern`
# that is NOT marked disabled for `migration_id`, AND that same line was
# previously marked (i.e. present in marked form in an earlier post-*
# snapshot for this migration_id). This is the "user peeled the marker
# off by hand" signal — later applies must not re-disable it.
_db_legacy_line_restored() {
    local file="$1" grep_pattern="$2" migration_id="$3"
    [[ -f "$file" ]] || return 1

    local marker
    marker=$(_db_legacy_marker_for "$migration_id")

    # Is there a live, unmarked line matching the pattern?
    grep -Eq "$grep_pattern" "$file" 2>/dev/null || return 1
    grep -E "$grep_pattern" "$file" 2>/dev/null | grep -qv "^${marker}" || {
        # every matching live line is already marked -> nothing "restored"
        return 1
    }

    # Was it ever marked before, per our own snapshots? Check the most
    # recent post-<migration_id> snapshot of this file for the marked form.
    local backup_dir="${DB_BACKUP_DIR:-$HOME/.devboost/backups}"
    local base
    base=$(basename "$file")
    local latest_snapshot
    latest_snapshot=$(ls -t "${backup_dir}/${base}.post-${migration_id}-"* 2>/dev/null | head -1) || true
    [[ -n "$latest_snapshot" ]] || return 1

    grep -Fq "${marker}" "$latest_snapshot" 2>/dev/null
}

# Comments out every live, unmarked line in `file` matching
# `grep_pattern` (extended regex) with the devboost:disabled marker for
# `migration_id`. Idempotent: already-marked lines are left untouched.
# No-op if the user has manually restored a previously-marked line
# (see _db_legacy_line_restored) — respects their override.
_db_legacy_disable_lines() {
    local file="$1" grep_pattern="$2" migration_id="$3"
    [[ -f "$file" ]] || return 0

    local marker
    marker=$(_db_legacy_marker_for "$migration_id")

    # Anything to do at all? Only unmarked lines matching the pattern.
    local to_disable
    to_disable=$(grep -E "$grep_pattern" "$file" 2>/dev/null | grep -v "^${marker}") || true
    if [[ -z "$to_disable" ]]; then
        db_log_verbose "No redundant lines to disable for $migration_id in $file"
        return 0
    fi

    if _db_legacy_line_restored "$file" "$grep_pattern" "$migration_id"; then
        db_log_verbose "Skipping $migration_id in $file — user restored a previously-disabled line"
        return 0
    fi

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        while IFS= read -r line; do
            db_log_info "Would disable ($migration_id) in $(basename "$file"): $line"
        done <<< "$to_disable"
        return 0
    fi

    _db_legacy_snapshot "$file" "$migration_id" "pre"

    local temp_file
    temp_file=$(mktemp)
    awk -v pattern="$grep_pattern" -v marker="$marker" '
        $0 ~ pattern && index($0, marker) != 1 { print marker $0; next }
        { print }
    ' "$file" > "$temp_file"

    # grep -E and awk's ERE dialect can disagree on backslash-escaped
    # literals in a pattern passed through a shell variable (confirmed
    # with nvm's source-line pattern). If grep found lines to disable
    # but awk's rewrite is identical to the original, the pattern is
    # dialect-mismatched — fail loudly instead of silently claiming
    # success while leaving the file untouched.
    if diff -q "$file" "$temp_file" >/dev/null 2>&1; then
        rm -f "$temp_file"
        db_log_error "legacy_shell ($migration_id): grep matched lines in $file but awk's rewrite changed nothing — pattern is grep/awk-dialect-mismatched, not applied"
        return 1
    fi

    mv "$temp_file" "$file"

    db_log_success "Disabled redundant lines ($migration_id) in: $file"

    _db_legacy_snapshot "$file" "$migration_id" "post"
}

# Moves `dir` aside into ~/.devboost/backups/ instead of deleting it.
# Idempotent: no-op if `dir` no longer exists (already archived).
_db_legacy_archive_dir() {
    local dir="$1" migration_id="$2"
    [[ -d "$dir" ]] || return 0

    local backup_dir="${DB_BACKUP_DIR:-$HOME/.devboost/backups}"
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local dest="${backup_dir}/$(basename "$dir")-${migration_id}-${timestamp}"

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would archive: $dir -> $dest"
        return 0
    fi

    db_ensure_dir "$backup_dir"
    mv "$dir" "$dest"
    db_log_success "Archived: $dir -> $dest"
}

# Strips every devboost:disabled-marked line (any migration_id) from
# `file`, restoring nothing — this is real deletion, the opt-in step.
# Idempotent: no-op if no marked lines are present.
_db_legacy_clean_file() {
    local file="$1"
    [[ -f "$file" ]] || return 0

    grep -qF "$_DB_LEGACY_MARKER_PREFIX" "$file" 2>/dev/null || return 0

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        local count
        count=$(grep -cF "$_DB_LEGACY_MARKER_PREFIX" "$file" 2>/dev/null) || count=0
        db_log_info "Would remove $count devboost-disabled line(s) from: $file"
        return 0
    fi

    db_backup_file "$file"
    local temp_file
    temp_file=$(mktemp)
    grep -vF "$_DB_LEGACY_MARKER_PREFIX" "$file" > "$temp_file" || true
    mv "$temp_file" "$file"
    db_log_success "Removed devboost-disabled lines from: $file"
}

# List of files any legacy-shell migration might mark. Extend here if a
# future migration_id targets a different file.
_db_legacy_managed_files() {
    echo "$HOME/.zshrc"
    echo "$HOME/.zprofile"
}

db_run_clean() {
    db_log_info "Cleaning devboost-disabled lines..."
    local f
    while IFS= read -r f; do
        _db_legacy_clean_file "$f"
    done < <(_db_legacy_managed_files)
    db_log_info "Archived directories remain under: ${DB_BACKUP_DIR:-$HOME/.devboost/backups} (remove that folder to purge everything)"
}
