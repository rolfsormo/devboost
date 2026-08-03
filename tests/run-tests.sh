#!/usr/bin/env bash
# Run all devboost tests
# Usage: ./tests/run-tests.sh [test-name]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# Prevent git from prompting for credentials or touching macOS Keychain during tests.
# GIT_TERMINAL_PROMPT stops terminal prompts; it does NOT stop the macOS Keychain GUI
# dialog, which comes from credential.helper=osxkeychain in system-level gitconfig
# (e.g. Homebrew's /opt/homebrew/etc/gitconfig) and fires even for anonymous HTTPS
# clones. GIT_ASKPASS=true (a no-op that "succeeds" with no output) plus explicitly
# unsetting credential.helper for these processes avoids it.
export GIT_TERMINAL_PROMPT=0
export GIT_ASKPASS=true
export GIT_CONFIG_COUNT=1
export GIT_CONFIG_KEY_0=credential.helper
export GIT_CONFIG_VALUE_0=

# Colors
if [[ -t 1 ]]; then
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    BLUE=''
    NC=''
fi

# Build script first
echo -e "${BLUE}Building devboost.sh...${NC}"
./build.sh

echo ""
echo -e "${BLUE}Running tests...${NC}"
echo ""

# Run specific test or all tests
if [[ $# -gt 0 ]]; then
    test_name="$1"
    if [[ -f "$SCRIPT_DIR/$test_name" ]]; then
        "$SCRIPT_DIR/$test_name"
    else
        echo "Test not found: $test_name"
        echo "Available tests:"
        ls -1 "$SCRIPT_DIR"/test-*.sh 2>/dev/null | xargs -n1 basename || true
        exit 1
    fi
else
    # Run all tests
    failed=0
    
    for test in "$SCRIPT_DIR"/test-*.sh; do
        if [[ -f "$test" && -x "$test" ]]; then
            if ! "$test"; then
                failed=1
            fi
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        echo ""
        echo -e "${BLUE}All tests passed!${NC}"
        exit 0
    else
        echo ""
        echo -e "${BLUE}Some tests failed${NC}"
        exit 1
    fi
fi

