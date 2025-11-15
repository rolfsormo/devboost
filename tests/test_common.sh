#!/usr/bin/env bash
# Common test framework for devboost tests
# Provides assertion functions and test utilities

set -euo pipefail

# Test state
TEST_COUNT=0
TEST_PASSED=0
TEST_FAILED=0
TEST_FAILURES=()

# Colors for output
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Test assertion functions
test_assert() {
    local description="$1"
    local condition="$2"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    if eval "$condition"; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description")
        echo -e "${RED}✗${NC} $description"
        return 1
    fi
}

test_assert_eq() {
    local description="$1"
    local expected="$2"
    local actual="$3"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    if [[ "$expected" == "$actual" ]]; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description (expected: '$expected', got: '$actual')")
        echo -e "${RED}✗${NC} $description"
        echo -e "  ${RED}Expected:${NC} '$expected'"
        echo -e "  ${RED}Got:${NC} '$actual'"
        return 1
    fi
}

test_assert_ne() {
    local description="$1"
    local expected="$2"
    local actual="$3"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    if [[ "$expected" != "$actual" ]]; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description (expected not equal, but both are: '$actual')")
        echo -e "${RED}✗${NC} $description"
        return 1
    fi
}

test_assert_contains() {
    local description="$1"
    local haystack="$2"
    local needle="$3"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    if [[ "$haystack" == *"$needle"* ]]; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description (expected to find '$needle' in output)")
        echo -e "${RED}✗${NC} $description"
        echo -e "  ${RED}Looking for:${NC} '$needle'"
        echo -e "  ${RED}In:${NC} '$haystack'"
        return 1
    fi
}

test_assert_not_contains() {
    local description="$1"
    local haystack="$2"
    local needle="$3"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    if [[ "$haystack" != *"$needle"* ]]; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description (expected not to find '$needle' in output)")
        echo -e "${RED}✗${NC} $description"
        echo -e "  ${RED}Found (but shouldn't):${NC} '$needle'"
        return 1
    fi
}

test_assert_exit_code() {
    local description="$1"
    local expected_code="$2"
    shift 2
    local command="$*"
    
    TEST_COUNT=$((TEST_COUNT + 1))
    
    set +e
    eval "$command" >/dev/null 2>&1
    local actual_code=$?
    set -e
    
    if [[ $actual_code -eq $expected_code ]]; then
        TEST_PASSED=$((TEST_PASSED + 1))
        echo -e "${GREEN}✓${NC} $description"
        return 0
    else
        TEST_FAILED=$((TEST_FAILED + 1))
        TEST_FAILURES+=("$description (expected exit code $expected_code, got $actual_code)")
        echo -e "${RED}✗${NC} $description"
        echo -e "  ${RED}Expected exit code:${NC} $expected_code"
        echo -e "  ${RED}Got exit code:${NC} $actual_code"
        return 1
    fi
}

# Test suite functions
test_suite_start() {
    local suite_name="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Running: $suite_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

test_suite_end() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Results${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "Total:  $TEST_COUNT"
    echo -e "${GREEN}Passed: $TEST_PASSED${NC}"
    if [[ $TEST_FAILED -gt 0 ]]; then
        echo -e "${RED}Failed: $TEST_FAILED${NC}"
        echo ""
        echo -e "${RED}Failures:${NC}"
        for failure in "${TEST_FAILURES[@]}"; do
            echo -e "  ${RED}•${NC} $failure"
        done
        return 1
    else
        echo -e "${GREEN}Failed: $TEST_FAILED${NC}"
        return 0
    fi
}

# Utility functions
test_get_bash_version() {
    local bash_path="${1:-bash}"
    if command -v "$bash_path" >/dev/null 2>&1; then
        "$bash_path" --version | head -n1 | sed -E 's/.*version ([0-9]+\.[0-9]+).*/\1/'
    else
        echo "0.0"
    fi
}

test_is_bash3() {
    local bash_path="${1:-bash}"
    local version=$(test_get_bash_version "$bash_path")
    local major=$(echo "$version" | cut -d. -f1)
    local minor=$(echo "$version" | cut -d. -f2)
    
    if [[ $major -lt 3 ]] || [[ $major -eq 3 && $minor -lt 2 ]]; then
        return 1  # Too old
    elif [[ $major -eq 3 ]]; then
        return 0  # Bash 3.x
    else
        return 1  # Bash 4+
    fi
}

test_find_bash3() {
    # Try to find bash 3.x for testing
    # On macOS, system bash is often 3.2
    # On Linux, we might need to check /bin/bash
    
    local candidates=(
        "/bin/bash"           # System bash on Linux/macOS
        "/usr/bin/bash"       # Alternative location
        "bash"                # PATH bash
    )
    
    for bash_path in "${candidates[@]}"; do
        if test_is_bash3 "$bash_path"; then
            echo "$bash_path"
            return 0
        fi
    done
    
    return 1
}

