# Security hygiene module
#
# Goals:
#   - Surface when tools are significantly out of date (LLMs regularly find 0-days in
#     widely-used packages; supply chain poisoning targets old pinned deps)
#   - Encourage a "stable but not bleeding-edge" update cadence
#   - Warn when config uses 'latest' for toolchains whose supply chains are higher risk
#   - Provide a convenient update alias so users can act on warnings without hunting commands
#
# Non-goals:
#   - Auto-updating anything without explicit user action
#   - Blocking apply/plan on security findings (advisory only)

db_module_security_register() {
    db_register_module "security" \
        "db_module_security_plan" \
        "db_module_security_apply" \
        "db_module_security_doctor"
}

db_module_security_plan() {
    local enable=$(db_yaml_get '.security.enable' 'true')
    [[ "$enable" != "true" ]] && return 0
    db_log_info "Would add update-check alias and security guidance to shell config"
}

db_module_security_apply() {
    local enable=$(db_yaml_get '.security.enable' 'true')
    [[ "$enable" != "true" ]] && return 0

    local include_file=$(db_yaml_get '.zsh.include_file' "$HOME/.zshrc.devboost")

    # Inject a db_security block with a 'devboost-check' alias that reminds the user
    # to run update checks. This is additive — it only sets an alias and a nag function.
    local block
    block=$(cat << 'EOF'
# >>> devboost security start
# Run 'devboost-check' at any time to see a summary of outdated tools.
devboost-check() {
    echo "=== devboost security check ==="
    local issues=0

    # Homebrew outdated
    if command -v brew &>/dev/null; then
        local outdated
        outdated=$(brew outdated --quiet 2>/dev/null | head -20)
        if [[ -n "$outdated" ]]; then
            echo "[brew] Outdated packages (run: brew upgrade):"
            echo "$outdated" | sed 's/^/  /'
            issues=$((issues + 1))
        else
            echo "[brew] All packages up to date"
        fi
    fi

    # mise outdated toolchains
    if command -v mise &>/dev/null; then
        local mise_outdated
        mise_outdated=$(mise outdated 2>/dev/null | grep -v "^Tool" | grep -v "^$" | head -10)
        if [[ -n "$mise_outdated" ]]; then
            echo "[mise] Outdated toolchains (run: mise upgrade):"
            echo "$mise_outdated" | sed 's/^/  /'
            issues=$((issues + 1))
        else
            echo "[mise] All toolchains up to date"
        fi
    fi

    # npm audit (only if a package.json is in the current directory)
    if command -v npm &>/dev/null && [[ -f "$(pwd)/package.json" ]]; then
        echo "[npm] Running audit in $(pwd)..."
        npm audit --audit-level=high 2>/dev/null || true
    fi

    if [[ $issues -eq 0 ]]; then
        echo "All checks passed."
    else
        echo ""
        echo "Tip: keep tools at stable LTS, not bleeding edge — run update checks weekly."
    fi
}
# <<< devboost security end
EOF
)

    if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
        db_log_info "Would inject security alias block into: $include_file"
        return 0
    fi

    db_upsert_block "$include_file" \
        "# >>> devboost security start" \
        "# <<< devboost security end" \
        "$block"

    db_log_success "Security: devboost-check alias available in new shells"
}

db_module_security_doctor() {
    local enable=$(db_yaml_get '.security.enable' 'true')
    [[ "$enable" != "true" ]] && return 0

    local found_issues=0

    # Warn if any mise toolchain is pinned to 'latest' — these pull unverified new releases
    # immediately. 'lts' and 'stable' are safer because they track a vetted release stream.
    for key in node python go rust deno; do
        local ver=$(db_yaml_get ".toolchains.globals.${key}" '')
        if [[ "$ver" == "latest" ]] && [[ "$key" != "deno" ]]; then
            # Deno 'latest' is its conventional stable channel — skip that warning
            db_log_warn "security: toolchain '$key' is pinned to 'latest' — prefer a specific version or 'lts'/'stable' to avoid pulling unreviewed releases"
            found_issues=$((found_issues + 1))
        fi
    done

    # Check if TPM was cloned over HTTPS (not HTTP)
    local tpm_path=$(db_yaml_get '.tmux.tpm_path' "$HOME/.tmux/plugins/tpm")
    if [[ -d "$tpm_path/.git" ]]; then
        local tpm_remote
        tpm_remote=$(git -C "$tpm_path" remote get-url origin 2>/dev/null || echo '')
        if [[ "$tpm_remote" == http://* ]]; then
            db_log_warn "security: TPM remote uses plain HTTP — re-clone over HTTPS"
            found_issues=$((found_issues + 1))
        fi
    fi

    # Check Homebrew/apt for significantly outdated packages (more than ~6 months behind)
    # We don't have reliable version-age data here, so we check if the package manager
    # itself hasn't been updated in a long time by inspecting its metadata freshness.
    if command -v brew &>/dev/null; then
        # If the Homebrew index is >7 days old without an update, warn
        local brew_repo
        brew_repo="$(brew --repository 2>/dev/null)/Library/Taps/homebrew/homebrew-core"
        if [[ -d "$brew_repo/.git" ]]; then
            local last_fetch
            last_fetch=$(git -C "$brew_repo" log -1 --format="%ct" 2>/dev/null || echo 0)
            local now
            now=$(date +%s)
            local age_days=$(( (now - last_fetch) / 86400 ))
            if [[ $age_days -gt 7 ]]; then
                db_log_warn "security: Homebrew index is ${age_days} days old — run 'brew update' to get security patches"
                found_issues=$((found_issues + 1))
            fi
        fi
    fi

    if [[ $found_issues -eq 0 ]]; then
        db_log_success "security: No obvious issues found"
    else
        db_log_warn "security: Run 'devboost-check' (after applying) for a live update summary"
    fi
}
