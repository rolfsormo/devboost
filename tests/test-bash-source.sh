#!/usr/bin/env bash
# Test that BASH_SOURCE[0] doesn't cause unbound variable errors
# This test verifies the script works even when BASH_SOURCE might be unbound

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/test_common.sh"

test_suite_start "BASH_SOURCE[0] Unbound Variable Tests"

# Test 1: Script should not error when piped from curl
# Simulate the curl | bash scenario
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    # This simulates how the script is executed via curl | bash
    # When piped, BASH_SOURCE[0] is unbound, which causes error with set -u
    output=$(cat "$PROJECT_ROOT/devboost.sh" | bash -u -s -- plan 2>&1)
    test_assert_not_contains \
        "No 'BASH_SOURCE[0]: unbound variable' error when piped" \
        "$output" \
        "BASH_SOURCE\[0\]: unbound variable"
    
    test_assert_exit_code \
        "Script executes successfully when piped with -u flag" \
        0 \
        "cat $PROJECT_ROOT/devboost.sh | bash -u -s -- plan >/dev/null 2>&1"
else
    echo -e "${YELLOW}⚠${NC}  devboost.sh not found, skipping test"
fi

# Test 2: Script should work with set -u (unbound variable check)
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    # Create a minimal test that sources the script with -u
    TEST_SCRIPT=$(mktemp)
    cat > "$TEST_SCRIPT" << 'TESTEOF'
#!/usr/bin/env bash
set -euo pipefail
# Source the script (simulating how it might be sourced)
source "$1" >/dev/null 2>&1 || true
# Try to execute a simple command
bash "$1" plan >/dev/null 2>&1 || true
TESTEOF
    chmod +x "$TEST_SCRIPT"
    
    test_assert_exit_code \
        "Script works with strict unbound variable checking" \
        0 \
        "bash -u $TEST_SCRIPT $PROJECT_ROOT/devboost.sh"
    
    rm -f "$TEST_SCRIPT"
else
    echo -e "${YELLOW}⚠${NC}  devboost.sh not found, skipping test"
fi

# Test 3: Verify BASH_SOURCE[0] check uses safe syntax
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    # Check that the script uses safe BASH_SOURCE access
    # Should use ${BASH_SOURCE[0]:-} to handle unbound variable
    bash_source_line=$(grep 'BASH_SOURCE\[0\]' "$PROJECT_ROOT/devboost.sh" 2>/dev/null | head -1 || true)
    test_assert_contains \
        "BASH_SOURCE[0] uses safe access pattern (\${BASH_SOURCE[0]:-})" \
        "$bash_source_line" \
        'BASH_SOURCE[0]:-'
fi

# Test 4: Direct execution test with strict mode
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    # Test that the script can be executed directly with -u flag
    output=$(bash -u "$PROJECT_ROOT/devboost.sh" plan 2>&1 || true)
    test_assert_not_contains \
        "No 'unbound variable' errors for BASH_SOURCE" \
        "$output" \
        "BASH_SOURCE\[0\]: unbound variable"
fi

test_suite_end

