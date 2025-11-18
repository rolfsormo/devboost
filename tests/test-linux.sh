#!/usr/bin/env bash
# Test devboost on Linux distributions using Docker/Podman
# Usage: ./tests/test-linux.sh [ubuntu|debian|fedora|arch|all]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source test framework
source "$SCRIPT_DIR/test_common.sh"

# Detect container runtime
_detect_container_runtime() {
    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        echo "docker"
    elif command -v podman >/dev/null 2>&1 && podman info >/dev/null 2>&1; then
        echo "podman"
    else
        echo ""
    fi
}

# Check if running on ARM64
_is_arm64() {
    if [[ "$(uname -m)" == "arm64" ]] || [[ "$(uname -m)" == "aarch64" ]]; then
        return 0
    else
        return 1
    fi
}

# Test a single distribution
_test_distro() {
    local distro="$1"
    local runtime="$2"
    
    # Skip Arch on ARM64
    if [[ "$distro" == "arch" ]] && _is_arm64; then
        echo -e "${YELLOW}⚠${NC}  Skipping Arch Linux on ARM64 (official image limitation)"
        return 0
    fi
    
    echo ""
    echo -e "${BLUE}Testing on $distro using $runtime...${NC}"
    
    local dockerfile="$SCRIPT_DIR/docker/Dockerfile.$distro"
    if [[ ! -f "$dockerfile" ]]; then
        test_assert "Dockerfile exists for $distro" "false"
        return 1
    fi
    
    # Build image
    local image_name="devboost-test-$distro"
    echo "Building image: $image_name"
    if ! "$runtime" build -f "$dockerfile" -t "$image_name" "$PROJECT_ROOT" >/dev/null 2>&1; then
        test_assert "Image build succeeds for $distro" "false"
        return 1
    fi
    
    # Copy devboost.sh into container and run tests
    local test_script=$(cat << 'TESTEOF'
#!/usr/bin/env bash
set -euo pipefail

cd /home/testuser

# Copy devboost.sh
cp /tmp/devboost.sh ./devboost.sh
chmod +x ./devboost.sh

# Create minimal config
mkdir -p ~/.config
cat > ~/.devboost.yaml << 'CONFIGEOF'
version: "1.0.0"
CONFIGEOF

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Syntax check
if bash -n devboost.sh 2>&1; then
    echo "✓ Syntax check passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "✗ Syntax check failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

# Test 2: Plan mode
if ./devboost.sh plan >/dev/null 2>&1; then
    echo "✓ Plan mode passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "✗ Plan mode failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

# Test 3: Apply dry-run
if ./devboost.sh apply --dry-run >/dev/null 2>&1; then
    echo "✓ Apply dry-run passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "✗ Apply dry-run failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

# Test 4: Doctor mode
if ./devboost.sh doctor >/dev/null 2>&1; then
    echo "✓ Doctor mode passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo "✗ Doctor mode failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

echo ""
echo "All tests passed: $TESTS_PASSED/$((TESTS_PASSED + TESTS_FAILED))"
exit 0
TESTEOF
)
    
    # Run container with test script
    local output
    local exit_code=0
    if output=$("$runtime" run --rm \
        -v "$PROJECT_ROOT/devboost.sh:/tmp/devboost.sh:ro" \
        "$image_name" \
        bash -c "$test_script" 2>&1); then
        echo "$output"
        test_assert "$distro: All container tests passed" "true"
    else
        echo "$output"
        test_assert "$distro: Container tests passed" "false"
        exit_code=1
    fi
    
    # Cleanup image (optional, but good practice)
    "$runtime" rmi "$image_name" >/dev/null 2>&1 || true
    
    return $exit_code
}

# Main test function
main() {
    local distro="${1:-all}"
    local runtime=$(_detect_container_runtime)
    
    if [[ -z "$runtime" ]]; then
        echo -e "${RED}Error:${NC} Neither Docker nor Podman is available"
        echo "Please install Docker or Podman to run Linux tests"
        exit 1
    fi
    
    echo -e "${BLUE}Using container runtime: $runtime${NC}"
    
    # Check if devboost.sh exists
    if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
        echo -e "${RED}Error:${NC} devboost.sh not found. Run ./build.sh first."
        exit 1
    fi
    
    test_suite_start "Linux Distribution Tests"
    
    local distros=()
    if [[ "$distro" == "all" ]]; then
        distros=(ubuntu debian fedora arch)
    else
        distros=("$distro")
    fi
    
    local failed=0
    for d in "${distros[@]}"; do
        if ! _test_distro "$d" "$runtime"; then
            failed=1
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        test_suite_end
        return 0
    else
        echo ""
        echo -e "${RED}Some distribution tests failed${NC}"
        test_suite_end
        return 1
    fi
}

# Run main if script is executed directly
if [[ "${BASH_SOURCE[0]:-}" == "${0}" ]]; then
    main "$@"
fi

