#!/usr/bin/env bash
# Test that devboost.sh actually runs with bash 3.x
# Creates a temporary environment and runs the script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/test_common.sh"

test_suite_start "Bash 3.x Runtime Tests"

# Find bash 3.x
if ! bash_path=$(test_find_bash3); then
    echo -e "${YELLOW}⚠${NC}  Bash 3.x not found on this system"
    echo -e "   Skipping runtime tests"
    echo -e "   To test on bash 3.x, install bash 3.2 or use a system with it"
    test_suite_end
    exit 0
fi

bash_version=$(test_get_bash_version "$bash_path")
echo -e "${BLUE}Using bash $bash_version at: $bash_path${NC}"
echo ""

# Check if script exists
if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
    echo -e "${RED}✗${NC}  devboost.sh not found"
    echo -e "   Run ./build.sh first to build the script"
    test_suite_end
    exit 1
fi

# Create temporary test environment
TEST_HOME=$(mktemp -d)
TEST_CONFIG="$TEST_HOME/.devboost.yaml"

# Cleanup function
cleanup() {
    rm -rf "$TEST_HOME"
}
trap cleanup EXIT

# Create minimal config
cat > "$TEST_CONFIG" << 'EOF'
version: "1.0.0"
EOF

# Test 1: Script can be sourced/executed without syntax errors
test_assert_exit_code \
    "Script syntax is valid for bash $bash_version" \
    0 \
    "$bash_path -n $PROJECT_ROOT/devboost.sh"

# Test 2: Script can run 'plan' command
test_assert_exit_code \
    "Script can execute 'plan' command with bash $bash_version" \
    0 \
    "HOME=$TEST_HOME $bash_path $PROJECT_ROOT/devboost.sh plan"

# Test 3: Script can run 'doctor' command
test_assert_exit_code \
    "Script can execute 'doctor' command with bash $bash_version" \
    0 \
    "HOME=$TEST_HOME $bash_path $PROJECT_ROOT/devboost.sh doctor"

# Test 4: Script can run 'apply --dry-run' command
test_assert_exit_code \
    "Script can execute 'apply --dry-run' command with bash $bash_version" \
    0 \
    "HOME=$TEST_HOME $bash_path $PROJECT_ROOT/devboost.sh apply --dry-run"

# Test 5: Verify no 'declare -A' errors appear
output=$(HOME=$TEST_HOME $bash_path "$PROJECT_ROOT/devboost.sh" plan 2>&1 || true)
test_assert_not_contains \
    "No 'declare: -A: invalid option' errors" \
    "$output" \
    "declare: -A: invalid option"

# Test 6: Verify no bash 4+ feature errors
test_assert_not_contains \
    "No bash 4+ feature errors" \
    "$output" \
    "invalid option"

test_suite_end

