#!/usr/bin/env bash
# Tests install.sh — the rustup-style bootstrap dispatcher — against a
# locally served binary, matching what a real fetch+exec would do
# (no GitHub Actions/real release URL needed, per the local-only
# cross-compilation decision).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

source "$SCRIPT_DIR/test_common.sh"

test_suite_start "install.sh bootstrap dispatcher"

if [[ ! -f "$PROJECT_ROOT/install.sh" ]]; then
    echo -e "${RED}Error:${NC} install.sh not found."
    exit 1
fi

test_assert_exit_code \
    "install.sh has valid POSIX sh syntax" \
    0 \
    "sh -n $PROJECT_ROOT/install.sh"

# --- Real end-to-end test: build the actual binary, serve it locally,
# run the real script against it, confirm it detects this platform and
# execs the binary correctly. ---
TMPDIR_TEST=$(mktemp -d)
cleanup() {
    [[ -n "${SERVER_PID:-}" ]] && kill "$SERVER_PID" 2>/dev/null || true
    rm -rf "$TMPDIR_TEST"
}
trap cleanup EXIT

OS=$(uname -s)
ARCH=$(uname -m)
case "$OS" in
    Darwin) PLATFORM_OS=darwin ;;
    Linux) PLATFORM_OS=linux ;;
    *) PLATFORM_OS="" ;;
esac
case "$ARCH" in
    x86_64|amd64) PLATFORM_ARCH=amd64 ;;
    arm64|aarch64) PLATFORM_ARCH=arm64 ;;
    *) PLATFORM_ARCH="" ;;
esac

if [[ -n "$PLATFORM_OS" && -n "$PLATFORM_ARCH" ]]; then
    (cd "$PROJECT_ROOT" && go build -o "$TMPDIR_TEST/devboost-${PLATFORM_OS}-${PLATFORM_ARCH}" ./cmd/devboost)

    PORT=8143
    (cd "$TMPDIR_TEST" && python3 -m http.server "$PORT" >/dev/null 2>&1) &
    SERVER_PID=$!

    # Poll for the server to actually accept connections instead of a
    # fixed sleep — a fixed 1s guess raced http.server's real bind time
    # on slower/loaded runners (observed failing on GitHub Actions'
    # macOS runner, which is meaningfully slower than local hardware).
    for _ in $(seq 1 50); do
        curl -s -o /dev/null "http://localhost:$PORT/" && break
        sleep 0.1
    done

    output=$(DEVBOOST_INSTALL_BASE_URL="http://localhost:$PORT" "$PROJECT_ROOT/install.sh" --version 2>&1) && exit_code=0 || exit_code=$?

    test_assert_eq \
        "install.sh fetch+exec against a local binary exits zero" \
        "0" \
        "$exit_code"

    test_assert_contains \
        "install.sh correctly detects this platform" \
        "$output" \
        "${PLATFORM_OS}-${PLATFORM_ARCH}"

    test_assert_contains \
        "install.sh execs the real binary (--version output present)" \
        "$output" \
        "devboost 2.0.0"

    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
else
    echo -e "${YELLOW}Skipping real fetch+exec test — unrecognized platform for this test harness${NC}"
fi

# --- Failure path: no server listening, should error clearly, not hang ---
output=$(DEVBOOST_INSTALL_BASE_URL="http://localhost:1" "$PROJECT_ROOT/install.sh" --version 2>&1) && exit_code=0 || exit_code=$?

test_assert_ne \
    "install.sh exits non-zero when the download fails" \
    "0" \
    "$exit_code"

test_suite_end
