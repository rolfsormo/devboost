#!/usr/bin/env bash
# Test devboost's handling of a pre-existing unmarked .zshrc.devboost source line,
# to avoid double-sourcing it (which doubles shell startup cost).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$SCRIPT_DIR/test_common.sh"

# This suite runs real 'apply' invocations (znap/TPM clones over HTTPS) — neutralize
# git's credential helper so a flaky/odd response never pops the macOS Keychain GUI
# dialog and hangs the test. See tests/run-tests.sh for the same guard.
export GIT_TERMINAL_PROMPT=0
export GIT_ASKPASS=true
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=credential.helper
export GIT_CONFIG_VALUE_0=

cleanup() {
    if [[ -n "${TEST_HOME:-}" ]] && [[ -d "$TEST_HOME" ]]; then
        rm -rf "$TEST_HOME"
    fi
}
trap cleanup EXIT

test_suite_start "Zshrc Double-Source Detection Tests"

if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
    echo -e "${RED}Error:${NC} devboost.sh not found. Run ./build.sh first."
    exit 1
fi

# --- Test 1: pre-existing unmarked source line -> apply skips, warns, no duplicate ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
export FOO=bar
[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")
source_count=$(grep -c '\.zshrc\.devboost' "$TEST_HOME/.zshrc")

test_assert_eq \
    "Apply with unmarked line: exits zero" \
    "0" \
    "$exit_code"

test_assert_contains \
    "Apply with unmarked line: warns about it" \
    "$output" \
    "double-sourcing"

test_assert_eq \
    "Apply with unmarked line: does not duplicate the source line" \
    "1" \
    "$source_count"

test_assert_not_contains \
    "Apply with unmarked line: does not add its own marked block" \
    "$result" \
    "devboost include start"

rm -rf "$TEST_HOME"

# --- Test 2: plan mode reports the same thing, makes no changes ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
export FOO=bar
[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"
EOF
before=$(cat "$TEST_HOME/.zshrc")

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" plan 2>&1) && exit_code=0 || exit_code=$?
after=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "Plan with unmarked line: warns about it" \
    "$output" \
    "double-sourcing"

test_assert_eq \
    "Plan with unmarked line: makes no changes" \
    "$before" \
    "$after"

rm -rf "$TEST_HOME"

# --- Test 3: normal case (no pre-existing line) still injects the marked block ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
export FOO=bar
EOF

HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1 || true
result=$(cat "$TEST_HOME/.zshrc")

test_assert_contains \
    "Normal case: injects marked include block" \
    "$result" \
    "devboost include start"

rm -rf "$TEST_HOME"

# --- Test 4: doctor warns if .zshrc ends up double-sourced anyway ---
TEST_HOME=$(mktemp -d)
cat > "$TEST_HOME/.zshrc" << 'EOF'
[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"
[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" doctor 2>&1) && exit_code=0 || exit_code=$?

test_assert_contains \
    "Doctor: warns about double-sourcing" \
    "$output" \
    "double-sourced"

rm -rf "$TEST_HOME"

test_suite_end
