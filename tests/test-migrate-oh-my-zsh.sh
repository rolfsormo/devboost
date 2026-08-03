#!/usr/bin/env bash
# Test devboost's migrate-from-oh-my-zsh command in a sandboxed environment

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

test_suite_start "Migrate-from-oh-my-zsh Tests"

if [[ ! -f "$PROJECT_ROOT/devboost.sh" ]]; then
    echo -e "${RED}Error:${NC} devboost.sh not found. Run ./build.sh first."
    exit 1
fi

# --- Test 1: no backup present -> graceful error, nothing written ---
TEST_HOME=$(mktemp -d)

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh 2>&1) && exit_code=0 || exit_code=$?

test_assert_eq \
    "No backup: exits non-zero" \
    "1" \
    "$exit_code"

test_assert_contains \
    "No backup: explains what to run first" \
    "$output" \
    "uninstall_oh_my_zsh"

rm -rf "$TEST_HOME"

# --- Test 2: clean merge, no conflicts ---
TEST_HOME=$(mktemp -d)

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
alias ll="ls -la"
EOF

cp "$TEST_HOME/.zshrc.pre-oh-my-zsh" "$TEST_HOME/.zshrc"

cat > "$TEST_HOME/.zshrc.omz-uninstalled-20260101_120000" << 'EOF'
export PATH="$HOME/bin:$PATH"
alias ll="ls -la"

# Path to your Oh My Zsh installation.
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="agnoster"
plugins=(git docker kubectl)
source $ZSH/oh-my-zsh.sh

export EDITOR="nvim"
alias gs="git status"
export MY_CUSTOM_VAR="hello"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Clean merge: exits zero" \
    "0" \
    "$exit_code"

test_assert_contains \
    "Clean merge: keeps base content" \
    "$result" \
    'alias ll="ls -la"'

test_assert_contains \
    "Clean merge: recovers user's post-install additions" \
    "$result" \
    'export MY_CUSTOM_VAR="hello"'

test_assert_contains \
    "Clean merge: recovers user's post-install alias" \
    "$result" \
    'alias gs="git status"'

test_assert_not_contains \
    "Clean merge: strips oh-my-zsh ZSH_THEME line" \
    "$result" \
    "ZSH_THEME"

test_assert_not_contains \
    "Clean merge: strips oh-my-zsh source line" \
    "$result" \
    "source \$ZSH/oh-my-zsh.sh"

test_assert_not_contains \
    "Clean merge: strips oh-my-zsh plugins= line" \
    "$result" \
    "plugins=("

rm -rf "$TEST_HOME"

# --- Test 3: genuine conflict -> conflict markers, non-zero exit ---
TEST_HOME=$(mktemp -d)

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
alias ll="ls -la"
EOF

# Current .zshrc diverges from base on the same line the "other" side also changed
cat > "$TEST_HOME/.zshrc" << 'EOF'
export PATH="$HOME/bin:/usr/local/go/bin:$PATH"
alias ll="ls -la"
EOF

cat > "$TEST_HOME/.zshrc.omz-uninstalled-20260101_120000" << 'EOF'
export PATH="$HOME/bin:$PATH:/opt/custom/bin"
alias ll="ls -la"

export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="agnoster"
plugins=(git)
source $ZSH/oh-my-zsh.sh

alias gs="git status"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Conflict: exits non-zero" \
    "1" \
    "$exit_code"

test_assert_contains \
    "Conflict: leaves conflict markers in .zshrc" \
    "$result" \
    "<<<<<<<"

test_assert_contains \
    "Conflict: still recovers non-conflicting addition" \
    "$result" \
    'alias gs="git status"'

rm -rf "$TEST_HOME"

# --- Test 4: no pre-oh-my-zsh base -> graceful error, nothing overwritten ---
TEST_HOME=$(mktemp -d)

cat > "$TEST_HOME/.zshrc" << 'EOF'
export PATH="$HOME/bin:$PATH"
EOF

cat > "$TEST_HOME/.zshrc.omz-uninstalled-20260101_120000" << 'EOF'
export PATH="$HOME/bin:$PATH"
export ZSH="$HOME/.oh-my-zsh"
source $ZSH/oh-my-zsh.sh
EOF

before=$(cat "$TEST_HOME/.zshrc")
output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh 2>&1) && exit_code=0 || exit_code=$?
after=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "No base backup: exits non-zero" \
    "1" \
    "$exit_code"

test_assert_eq \
    "No base backup: .zshrc left untouched" \
    "$before" \
    "$after"

rm -rf "$TEST_HOME"

# --- Test 5: --dry-run makes no changes ---
TEST_HOME=$(mktemp -d)

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
EOF
cp "$TEST_HOME/.zshrc.pre-oh-my-zsh" "$TEST_HOME/.zshrc"
cat > "$TEST_HOME/.zshrc.omz-uninstalled-20260101_120000" << 'EOF'
export PATH="$HOME/bin:$PATH"
export ZSH="$HOME/.oh-my-zsh"
source $ZSH/oh-my-zsh.sh
export MY_VAR="test"
EOF

before=$(cat "$TEST_HOME/.zshrc")
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --dry-run >/dev/null 2>&1 || true
after=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Dry-run: .zshrc left untouched" \
    "$before" \
    "$after"

rm -rf "$TEST_HOME"

test_suite_end
