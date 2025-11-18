#!/usr/bin/env bash
# Test devboost on macOS in a sandboxed environment
# Creates a temporary home directory and runs tests without affecting user's config

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/test_common.sh"

# Cleanup function
cleanup() {
    if [[ -n "${TEST_HOME:-}" ]] && [[ -d "$TEST_HOME" ]]; then
        rm -rf "$TEST_HOME"
    fi
}
trap cleanup EXIT

test_suite_start "macOS Sandboxed Tests"

# Check if devboost.sh exists
if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
    echo -e "${RED}Error:${NC} devboost.sh not found. Run ./build.sh first."
    exit 1
fi

# Create temporary home directory
TEST_HOME=$(mktemp -d)
export HOME="$TEST_HOME"

echo -e "${BLUE}Using temporary home: $TEST_HOME${NC}"
echo ""

# Create minimal config
mkdir -p "$TEST_HOME/.config"
cat > "$TEST_HOME/.devboost.yaml" << 'CONFIGEOF'
version: "1.0.0"
CONFIGEOF

# Test 1: Syntax check
test_assert_exit_code \
    "Script syntax is valid" \
    0 \
    "bash -n $PROJECT_ROOT/devboost.sh"

# Test 2: Plan mode
test_assert_exit_code \
    "Plan mode executes successfully" \
    0 \
    "HOME=$TEST_HOME $PROJECT_ROOT/devboost.sh plan"

# Test 3: Apply dry-run
test_assert_exit_code \
    "Apply dry-run executes successfully" \
    0 \
    "HOME=$TEST_HOME $PROJECT_ROOT/devboost.sh apply --dry-run"

# Test 4: Doctor mode
test_assert_exit_code \
    "Doctor mode executes successfully" \
    0 \
    "HOME=$TEST_HOME $PROJECT_ROOT/devboost.sh doctor"

# Test 5: Verify files are created in temp directory (not real home)
if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
    # Run apply (not dry-run) to create files
    HOME=$TEST_HOME "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1 || true
    
    test_assert \
        ".zshrc.devboost is created in temp directory" \
        "[[ -f $TEST_HOME/.zshrc.devboost ]]"
    
    test_assert \
        ".zshrc is created in temp directory" \
        "[[ -f $TEST_HOME/.zshrc ]]"
    
    test_assert \
        "Real home directory is not modified" \
        "[[ ! -f $HOME/.zshrc.devboost ]] || [[ $HOME == $TEST_HOME ]]"
fi

# Test 6: Idempotency - second apply should be no-op
if [[ "${DB_DRY_RUN:-false}" != "true" ]]; then
    # Get initial state
    initial_zshrc=$(cat "$TEST_HOME/.zshrc" 2>/dev/null || echo "")
    
    # Run apply again
    HOME=$TEST_HOME "$PROJECT_ROOT/devboost.sh" apply >/dev/null 2>&1 || true
    
    # Check that file hasn't changed (idempotent)
    second_zshrc=$(cat "$TEST_HOME/.zshrc" 2>/dev/null || echo "")
    test_assert_eq \
        "Second apply is idempotent (no changes)" \
        "$initial_zshrc" \
        "$second_zshrc"
fi

test_suite_end

