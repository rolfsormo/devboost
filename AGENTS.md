# AGENTS.md - Development Guide for AI Agents and Contributors

This document provides guidelines for AI agents and human contributors working on devboost. The goal is to maintain **super high quality** code that follows 2025 best practices, prioritizes **ease of use**, and makes it **trivially easy** to add new modules.

## Core Principles

### 1. User Experience First

- **Zero prompts**: Users should never be asked questions during execution. All decisions should be made automatically with sensible defaults.
- **Fully configurable**: Everything must be configurable via `~/.devboost.yaml`, but defaults should work perfectly out of the box.
- **Idempotent**: Safe to run multiple times. Always check state before making changes.
- **Non-destructive**: Never modify user's existing files directly. Use managed blocks/includes.

### 2. Code Quality Standards (2025)

- **Bash best practices**:
  - Use `set -euo pipefail` at the top of all scripts
  - Quote all variables: `"$var"` not `$var`
  - Use `[[ ]]` for conditionals (not `[ ]`)
  - Prefer `command -v` over `which`
  - Use `readonly` for constants
  - Avoid `eval` unless absolutely necessary (and document why)

- **Error handling**:
  - Always check return codes
  - Provide helpful error messages with context
  - Use `db_log_error` for errors, `db_log_warn` for warnings
  - Never silently fail (unless explicitly handling expected failures)

- **Performance**:
  - Minimize external command calls
  - Cache results when appropriate (OS detection, config parsing)
  - Use efficient string operations
  - Avoid unnecessary subshells

- **Readability**:
  - Clear function names: `db_module_foo_apply` not `apply_foo`
  - Consistent naming: `db_*` prefix for all functions
  - Comments explain *why*, not *what*
  - Keep functions focused (single responsibility)

### 3. Module Development

**Adding a new module should be trivial:**

1. Create `modules/module_foo.sh`:

```bash
# Foo module

db_module_foo_register() {
    db_register_module "foo" \
        "db_module_foo_plan" \
        "db_module_foo_apply" \
        "db_module_foo_doctor"  # optional
}

db_module_foo_plan() {
    local enable=$(db_yaml_get '.foo.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    # Check what would change
    if [[ ! -f "$HOME/.foo/config" ]]; then
        db_log_info "Would create: $HOME/.foo/config"
    fi
}

db_module_foo_apply() {
    local enable=$(db_yaml_get '.foo.enable' 'true')
    if [[ "$enable" != "true" ]]; then
        return 0
    fi
    
    # Do the work idempotently
    db_ensure_dir "$HOME/.foo"
    local content=$(db_render_foo_config)
    db_write_file "$HOME/.foo/config" "$content"
}

# Optional: diagnostics
db_module_foo_doctor() {
    db_command_exists foo && db_log_success "foo: found" || db_log_error "foo: not found"
}
```

2. Add to `build.sh`:

```bash
cat modules/module_foo.sh
echo ""
# ... in registration section:
db_module_foo_register
```

3. Rebuild: `./build.sh`

**That's it!** The module is now part of the system.

### Module Best Practices

- **Always check enable flag**: `local enable=$(db_yaml_get '.module.enable' 'true')`
- **Use core helpers**: `db_write_file`, `db_upsert_block`, `db_backup_file`, `db_ensure_dir`
- **Respect dry-run**: Check `DB_DRY_RUN` before making changes
- **Provide defaults**: All config values should have sensible defaults
- **Idempotent operations**: Check if something exists before creating it
- **Use config system**: `db_yaml_get` for all configuration
- **Log appropriately**: Use `db_log_info`, `db_log_success`, `db_log_warn`, `db_log_error`

### 4. Commit Message Style (CBEAMS)

Follow the [CBEAMS commit message style](https://chris.beams.io/posts/git-commit/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, missing semicolons, etc.
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(zsh): add support for custom znap path

Allow users to configure znap installation path via
.zsh.znap_path in config file. Defaults to ~/.zsh-snap
if not specified.

Closes #42
```

```
fix(starship): correct git_status format syntax

The format string was using invalid variable concatenation.
Changed to use $all_status only, which is the correct
starship syntax.

Fixes #38
```

```
docs(readme): add installation instructions

Adds quick start section with curl-based installation
and build-from-source instructions.
```

**Rules:**
1. Separate subject from body with a blank line
2. Limit subject line to 50 characters
3. Capitalize subject line
4. Do not end subject line with a period
5. Use imperative mood ("add" not "adds" or "added")
6. Wrap body at 72 characters
7. Use body to explain *what* and *why* vs. *how*

### 5. Versioning Strategy

devboost follows **Semantic Versioning** with OS/tooling-specific adjustments:

**Format:** `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes that require user action
  - Config file schema changes (old configs won't work)
  - Removal of features
  - Changes to default behavior that users rely on
  - **Users should be aware**: If they're on an older MAJOR version, they may need to update their config

- **MINOR**: New features, backwards compatible
  - New modules
  - New config options (with defaults)
  - Enhancements to existing modules
  - **Users can upgrade safely**: Old configs still work

- **PATCH**: Bug fixes, backwards compatible
  - Fixes to existing functionality
  - Performance improvements
  - Documentation updates
  - **Users should upgrade**: Fixes issues they may be experiencing

**Version Compatibility:**

- Users can always upgrade PATCH versions safely
- Users can upgrade MINOR versions safely (new features available)
- Users upgrading MAJOR versions should review changelog for breaking changes
- The script should detect if user's config is from an older MAJOR version and warn (but not block)

**Version Detection:**

```bash
# In core_main.sh or similar
DB_VERSION="1.0.0"
DB_CONFIG_VERSION=$(db_yaml_get '.version' '')

if [[ -n "$DB_CONFIG_VERSION" ]]; then
    local config_major=$(echo "$DB_CONFIG_VERSION" | cut -d. -f1)
    local script_major=$(echo "$DB_VERSION" | cut -d. -f1)
    
    if [[ "$config_major" -lt "$script_major" ]]; then
        db_log_warn "Config file version ($DB_CONFIG_VERSION) is older than script version ($DB_VERSION)"
        db_log_warn "Please review CHANGELOG.md for breaking changes"
    fi
fi
```

**Versioning Best Practices:**

- Update version in `core/core_main.sh` (`DB_VERSION`)
- Tag releases: `git tag -a v1.0.0 -m "Release 1.0.0"`
- Maintain `CHANGELOG.md` with:
  - Breaking changes (MAJOR)
  - New features (MINOR)
  - Bug fixes (PATCH)

### 6. Testing Requirements

**All tests must pass before a change can be considered done.**

Before submitting changes, you **must** run and pass all applicable tests:

1. **Build test**: `./build.sh` must succeed
2. **Syntax check**: `bash -n dist/devboost.sh` must pass
3. **Plan test**: `./dist/devboost.sh plan` should show expected changes
4. **Idempotency test**: Run `apply` twice, second run should be no-op
5. **Config test**: Test with minimal config and full config
6. **Platform tests**: 
   - **macOS**: `./tests/test-macos.sh` must pass
   - **Linux**: `./tests/test-linux.sh all` must pass (if Docker is available)
   - If you can't test on a platform, note it in your PR and ask for help

**Test Execution:**
```bash
# Build first
./build.sh

# Test on macOS (sandboxed, safe)
./tests/test-macos.sh

# Test on all Linux distributions (requires Docker)
./tests/test-linux.sh all

# Or test individual distributions
./tests/test-linux.sh ubuntu
./tests/test-linux.sh debian
./tests/test-linux.sh fedora
./tests/test-linux.sh arch
```

**Failure is not an option**: If tests fail, the change is not complete. Fix the issues or document why the failure is acceptable (with maintainer approval).

### 7. Documentation Requirements

- **Code comments**: Explain *why*, not *what*
- **Function docs**: Brief comment above each exported function
- **Config docs**: Document all config options in `.devboost.yaml.example`
- **README**: Keep up to date with new features
- **CHANGELOG**: Document all user-facing changes

### 8. Security Considerations

- **Never execute user input**: All user input should be validated
- **Use absolute paths**: When possible, use absolute paths for security
- **Sanitize paths**: Validate file paths before operations
- **Backup before modify**: Always backup files before modifying
- **Principle of least privilege**: Don't require sudo unless necessary

### 9. Cross-Platform Compatibility

- **OS detection**: Use `db_detect_os` and check `DB_OS`
- **Package managers**: Use `db_install_packages` abstraction
- **Path differences**: Handle macOS (`/opt/homebrew`) vs Linux paths
- **Test on both**: When possible, test on macOS and Linux

### 10. Performance Guidelines

- **Minimize external calls**: Cache results when appropriate
- **Batch operations**: Group similar operations together
- **Lazy evaluation**: Only do work when needed
- **Efficient checks**: Use `command -v` not `which`, `test -f` not `ls`

## Quick Reference

### Adding a Module Checklist

- [ ] Create `modules/module_foo.sh`
- [ ] Implement `db_module_foo_register()`
- [ ] Implement `db_module_foo_plan()` (check enable flag)
- [ ] Implement `db_module_foo_apply()` (idempotent, use helpers)
- [ ] Add to `build.sh` (file inclusion + registration)
- [ ] Add config options to `.devboost.yaml.example`
- [ ] Test: `./build.sh && ./dist/devboost.sh plan`
- [ ] Test: `./dist/devboost.sh apply` (idempotent)
- [ ] Update README if user-facing
- [ ] Update CHANGELOG
- [ ] Commit with CBEAMS style

### Common Patterns

**Check if enabled:**
```bash
local enable=$(db_yaml_get '.module.enable' 'true')
if [[ "$enable" != "true" ]]; then
    return 0
fi
```

**Write file with backup:**
```bash
local content="..."
db_write_file "$HOME/.file" "$content"
```

**Inject block into existing file:**
```bash
local block="..."
db_upsert_block "$HOME/.file" "# start marker" "# end marker" "$block"
```

**Check command exists:**
```bash
if ! db_command_exists tool; then
    db_log_warn "tool not found, skipping"
    return 0
fi
```

**Respect dry-run:**
```bash
if [[ "${DB_DRY_RUN:-false}" == "true" ]]; then
    db_log_info "Would do X"
    return 0
fi
# Do actual work
```

## Questions?

If you're unsure about implementation details:
1. Check existing modules for patterns
2. Review core framework functions in `core/`
3. Test your changes thoroughly
4. Ask for review if needed

Remember: **Ease of use > Performance > Cleverness**

