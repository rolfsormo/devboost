#!/usr/bin/env bash
# Test bash 3.x compatibility
# Verifies that the script doesn't use bash 4+ features like associative arrays

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/test_common.sh"

test_suite_start "Bash 3.x Compatibility Tests"

# Test 1: Check that no associative arrays are declared
test_assert_not_contains \
    "No 'declare -A' in source files" \
    "$(grep -r "declare -A" "$PROJECT_ROOT/core" "$PROJECT_ROOT/modules" 2>/dev/null || true)" \
    "declare -A"

# Test 2: Check that no associative arrays are in built script
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    test_assert_not_contains \
        "No 'declare -A' in built script" \
        "$(grep "declare -A" "$PROJECT_ROOT/devboost.sh" 2>/dev/null || true)" \
        "declare -A"
else
    echo -e "${YELLOW}⚠${NC}  devboost.sh not found, skipping built script check"
    echo -e "   Run ./build.sh first to build the script"
fi

# Test 3: Check for other bash 4+ features
test_assert_not_contains \
    "No 'mapfile' or 'readarray' (bash 4+)" \
    "$(grep -r "mapfile\|readarray" "$PROJECT_ROOT/core" "$PROJECT_ROOT/modules" 2>/dev/null || true)" \
    "mapfile\|readarray"

# Test 4: Check for bash 4+ parameter expansion features
# ${var,,} and ${var^^} are bash 4+
# Note: We check for the literal patterns, not variable expansion
test_assert_not_contains \
    "No bash 4+ parameter expansion (case conversion)" \
    "$(grep -rE '\$\{[^}]*,,|\$\{[^}]*\^\^' "$PROJECT_ROOT/core" "$PROJECT_ROOT/modules" 2>/dev/null || true)" \
    ",,"

# Test 5: Syntax check with bash 3.x if available
if bash_path=$(test_find_bash3); then
    bash_version=$(test_get_bash_version "$bash_path")
    echo ""
    echo -e "${BLUE}Testing with bash $bash_version at: $bash_path${NC}"
    
    if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
        test_assert_exit_code \
            "Syntax check passes with bash $bash_version" \
            0 \
            "$bash_path -n $PROJECT_ROOT/devboost.sh"
    else
        echo -e "${YELLOW}⚠${NC}  devboost.sh not found, skipping syntax check"
    fi
else
    echo ""
    echo -e "${YELLOW}⚠${NC}  Bash 3.x not found, skipping syntax check"
    echo -e "   This is okay - the test verifies no bash 4+ features are used"
fi

# Test 6: Verify helper functions exist (bash 3.x compatibility layer)
if [[ -f "$PROJECT_ROOT/devboost.sh" ]]; then
    test_assert_contains \
        "Helper function _db_module_set exists (bash 3.x compatibility)" \
        "$(grep "_db_module_set" "$PROJECT_ROOT/devboost.sh" 2>/dev/null || true)" \
        "_db_module_set"
    
    test_assert_contains \
        "Helper function _db_module_get exists (bash 3.x compatibility)" \
        "$(grep "_db_module_get" "$PROJECT_ROOT/devboost.sh" 2>/dev/null || true)" \
        "_db_module_get"
    
    test_assert_contains \
        "Helper function _db_pkg_map exists (bash 3.x compatibility)" \
        "$(grep "_db_pkg_map" "$PROJECT_ROOT/devboost.sh" 2>/dev/null || true)" \
        "_db_pkg_map"
fi

test_suite_end

