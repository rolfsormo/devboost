#!/usr/bin/env bash
# Test devboost's legacy_shell module (zinit/znap, asdf/mise dedup) and the
# `clean` command, in a sandboxed environment.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$SCRIPT_DIR/test_common.sh"

cleanup() {
    if [[ -n "${TEST_HOME:-}" ]] && [[ -d "$TEST_HOME" ]]; then
        rm -rf "$TEST_HOME"
    fi
}
trap cleanup EXIT

test_suite_start "Legacy Shell Tooling + Clean Tests"

if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
    echo -e "${RED}Error:${NC} devboost.sh not found. Run ./build.sh first."
    exit 1
fi

# --- Test 1: clean .zshrc (no zinit, no asdf) -> apply is a no-op ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
export EDITOR="nvim"
eval "$(mise activate zsh)"
EOF

before=$(cat "$TEST_HOME/.zshrc")
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1 || true
after=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "Clean zshrc: no zinit/znap line disabled" \
    "$after" \
    'export EDITOR="nvim"'

test_assert_not_contains \
    "Clean zshrc: no devboost:disabled marker introduced" \
    "$after" \
    "devboost:disabled"

rm -rf "$TEST_HOME"

# --- Test 2: zinit+znap duplicate -> apply disables redundant zinit lines, idempotent ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
source "$HOME/.local/share/zinit/zinit.git/zinit.zsh"
zinit light zdharma-continuum/fast-syntax-highlighting
zinit light zsh-users/zsh-autosuggestions
zinit light zsh-users/zsh-completions
EOF

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "zinit dup: disables fast-syntax-highlighting line" \
    "$result" \
    "# devboost:disabled:zinit-znap-dup zinit light zdharma-continuum/fast-syntax-highlighting"

test_assert_contains \
    "zinit dup: disables zsh-autosuggestions line" \
    "$result" \
    "# devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions"

test_assert_contains \
    "zinit dup: leaves non-duplicate zsh-completions line active" \
    "$result" \
    "zinit light zsh-users/zsh-completions"

test_assert_not_contains \
    "zinit dup: zsh-completions line itself not marked" \
    "$result" \
    "devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-completions"

result_after_first=$result
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result_second=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "zinit dup: second apply is idempotent (no double-marking)" \
    "$result_after_first" \
    "$result_second"

rm -rf "$TEST_HOME"

# --- Test 3: asdf+mise duplicate -> apply disables asdf source line, idempotent ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
. /opt/homebrew/opt/asdf/libexec/asdf.sh
eval "$(mise activate zsh)"
EOF

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "asdf dup: disables asdf source line" \
    "$result" \
    "# devboost:disabled:asdf-mise-dup . /opt/homebrew/opt/asdf/libexec/asdf.sh"

test_assert_contains \
    "asdf dup: leaves mise line active" \
    "$result" \
    'eval "$(mise activate zsh)"'

result_after_first=$result
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result_second=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "asdf dup: second apply is idempotent" \
    "$result_after_first" \
    "$result_second"

rm -rf "$TEST_HOME"

# --- Test 3b: nvm+mise duplicate (in ~/.zprofile) -> apply disables nvm source lines, idempotent ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
eval "$(mise activate zsh)"
EOF
cat > "$TEST_HOME/.zprofile" << 'EOF'
eval "$(/opt/homebrew/bin/brew shellenv)"
export NVM_DIR="$HOME/.nvm"
[ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
[ -s "/opt/homebrew/opt/nvm/etc/bash_completion.d/nvm" ] && \. "/opt/homebrew/opt/nvm/etc/bash_completion.d/nvm"
EOF

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result=$(cat "$TEST_HOME/.zprofile")

test_assert_contains \
    "nvm dup: disables nvm.sh source line" \
    "$result" \
    '# devboost:disabled:nvm-mise-dup [ -s "/opt/homebrew/opt/nvm/nvm.sh" ]'

test_assert_contains \
    "nvm dup: disables nvm bash_completion source line" \
    "$result" \
    '# devboost:disabled:nvm-mise-dup [ -s "/opt/homebrew/opt/nvm/etc/bash_completion.d/nvm" ]'

test_assert_contains \
    "nvm dup: leaves NVM_DIR export untouched (harmless, not the expensive part)" \
    "$result" \
    'export NVM_DIR="$HOME/.nvm"'

test_assert_not_contains \
    "nvm dup: NVM_DIR line itself not marked" \
    "$result" \
    'devboost:disabled:nvm-mise-dup export NVM_DIR'

result_after_first=$result
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result_second=$(cat "$TEST_HOME/.zprofile")

test_assert_eq \
    "nvm dup: second apply is idempotent" \
    "$result_after_first" \
    "$result_second"

rm -rf "$TEST_HOME"

# --- Test 4: user manually restores a disabled line -> next apply respects it ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
zinit light zsh-users/zsh-autosuggestions
. /opt/homebrew/opt/asdf/libexec/asdf.sh
eval "$(mise activate zsh)"
EOF

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1

# User removes the marker prefix by hand, restoring the zinit line.
sed_inplace() {
    if sed --version >/dev/null 2>&1; then
        sed -i "$@"
    else
        sed -i '' "$@"
    fi
}
sed_inplace 's/^# devboost:disabled:zinit-znap-dup //' "$TEST_HOME/.zshrc"

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
result=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "Manual restore: zinit line stays restored (not re-disabled)" \
    "$result" \
    "zinit light zsh-users/zsh-autosuggestions"

test_assert_not_contains \
    "Manual restore: no re-applied zinit marker" \
    "$result" \
    "devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions"

test_assert_contains \
    "Manual restore: unrelated asdf line still disabled" \
    "$result" \
    "# devboost:disabled:asdf-mise-dup . /opt/homebrew/opt/asdf/libexec/asdf.sh"

rm -rf "$TEST_HOME"

# --- Test 5: clean strips marked lines; idempotent; works without prior apply in-process ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
# devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions
# devboost:disabled:asdf-mise-dup . /opt/homebrew/opt/asdf/libexec/asdf.sh
eval "$(mise activate zsh)"
EOF
cat > "$TEST_HOME/.zprofile" << 'EOF'
export NVM_DIR="$HOME/.nvm"
# devboost:disabled:nvm-mise-dup [ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" clean 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")
result_zprofile=$(cat "$TEST_HOME/.zprofile")

test_assert_eq \
    "clean: exits zero" \
    "0" \
    "$exit_code"

test_assert_not_contains \
    "clean: removes zinit marker line entirely" \
    "$result" \
    "devboost:disabled:zinit-znap-dup"

test_assert_not_contains \
    "clean: removes asdf marker line entirely" \
    "$result" \
    "devboost:disabled:asdf-mise-dup"

test_assert_contains \
    "clean: leaves unrelated content untouched" \
    "$result" \
    'eval "$(mise activate zsh)"'

test_assert_not_contains \
    "clean: removes nvm marker line from .zprofile too" \
    "$result_zprofile" \
    "devboost:disabled:nvm-mise-dup"

test_assert_contains \
    "clean: leaves .zprofile's NVM_DIR untouched" \
    "$result_zprofile" \
    'export NVM_DIR="$HOME/.nvm"'

result_after_first=$result
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" clean >/dev/null 2>&1
result_second=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "clean: idempotent on second run" \
    "$result_after_first" \
    "$result_second"

rm -rf "$TEST_HOME"

# --- Test 6: --dry-run makes no changes for apply and clean ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
zinit light zsh-users/zsh-autosuggestions
. /opt/homebrew/opt/asdf/libexec/asdf.sh
EOF

before=$(cat "$TEST_HOME/.zshrc")
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply --dry-run >/dev/null 2>&1 || true
after_apply=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Dry-run apply: .zshrc left untouched" \
    "$before" \
    "$after_apply"

# Now actually mark it, then verify dry-run clean doesn't touch it.
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1
marked=$(cat "$TEST_HOME/.zshrc")
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" clean --dry-run >/dev/null 2>&1 || true
after_clean=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Dry-run clean: .zshrc left untouched" \
    "$marked" \
    "$after_clean"

rm -rf "$TEST_HOME"

# --- Test 7: doctor reports conflicts without modifying any file ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
zinit light zsh-users/zsh-autosuggestions
. /opt/homebrew/opt/asdf/libexec/asdf.sh
EOF
cat > "$TEST_HOME/.zprofile" << 'EOF'
[ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
EOF

before=$(cat "$TEST_HOME/.zshrc")
before_zprofile=$(cat "$TEST_HOME/.zprofile")
output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" doctor 2>&1) || true
after=$(cat "$TEST_HOME/.zshrc")
after_zprofile=$(cat "$TEST_HOME/.zprofile")

test_assert_contains \
    "doctor: reports zinit/znap duplication" \
    "$output" \
    "legacy_shell: zinit is loading plugin(s) that duplicate znap's"

test_assert_contains \
    "doctor: reports asdf/mise duplication" \
    "$output" \
    "legacy_shell: asdf is active alongside mise"

test_assert_contains \
    "doctor: reports nvm/mise duplication" \
    "$output" \
    "legacy_shell: nvm's shell hook is active alongside mise"

test_assert_eq \
    "doctor: .zshrc left untouched" \
    "$before" \
    "$after"

test_assert_eq \
    "doctor: .zprofile left untouched" \
    "$before_zprofile" \
    "$after_zprofile"

rm -rf "$TEST_HOME"

# --- Test 8: already-disabled lines must not be re-reported as still needing action ---
# Regression test: the marker prepends to, not replaces, the original line, so a
# naive "does this pattern appear anywhere in the file" check still matches inside
# an already-disabled line. doctor/plan must exclude marked lines specifically, not
# just detect the pattern's presence. (Found for real: after `apply` disabled the
# asdf line on the investigation machine, `doctor` kept warning "asdf is active"
# forever afterward.)
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
# devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions
# devboost:disabled:asdf-mise-dup . /opt/homebrew/opt/asdf/libexec/asdf.sh
eval "$(mise activate zsh)"
EOF
cat > "$TEST_HOME/.zprofile" << 'EOF'
export NVM_DIR="$HOME/.nvm"
# devboost:disabled:nvm-mise-dup [ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
EOF

doctor_output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" doctor 2>&1) || true
plan_output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" plan 2>&1) || true

test_assert_contains \
    "Already-disabled: doctor reports zinit/znap clean, not still-warning" \
    "$doctor_output" \
    "legacy_shell: no zinit/znap plugin duplication found"

test_assert_contains \
    "Already-disabled: doctor reports asdf/mise clean, not still-warning" \
    "$doctor_output" \
    "legacy_shell: no asdf/mise duplication found"

test_assert_contains \
    "Already-disabled: doctor reports nvm/mise clean, not still-warning" \
    "$doctor_output" \
    "legacy_shell: no nvm/mise duplication found"

test_assert_not_contains \
    "Already-disabled: plan says nothing more needs disabling" \
    "$plan_output" \
    "Would disable"

rm -rf "$TEST_HOME"

test_suite_end
