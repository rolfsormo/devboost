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

# --- Test 1: refuses without --yes, makes no changes ---
TEST_HOME=$(mktemp -d)
mkdir -p "$TEST_HOME/.oh-my-zsh"
cat > "$TEST_HOME/.zshrc" << 'EOF'
export ZSH="$HOME/.oh-my-zsh"
source $ZSH/oh-my-zsh.sh
export MY_VAR="test"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh 2>&1) && exit_code=0 || exit_code=$?

test_assert_eq \
    "No --yes: exits non-zero" \
    "1" \
    "$exit_code"

test_assert_contains \
    "No --yes: explains the flag is required" \
    "$output" \
    "--yes"

test_assert \
    "No --yes: ~/.oh-my-zsh left untouched" \
    "[[ -d $TEST_HOME/.oh-my-zsh ]]"

rm -rf "$TEST_HOME"

# --- Test 2: no ~/.oh-my-zsh and no backup present -> graceful error ---
TEST_HOME=$(mktemp -d)

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --yes 2>&1) && exit_code=0 || exit_code=$?

test_assert_eq \
    "Nothing to do: exits non-zero" \
    "1" \
    "$exit_code"

test_assert_contains \
    "Nothing to do: explains nothing was found" \
    "$output" \
    "nothing to recover"

rm -rf "$TEST_HOME"

# --- Test 3: full flow with pre-install base -> uninstall + recover additions ---
TEST_HOME=$(mktemp -d)
mkdir -p "$TEST_HOME/.oh-my-zsh"

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
alias ll="ls -la"
EOF

cat > "$TEST_HOME/.zshrc" << 'EOF'
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

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --yes 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Full flow: exits zero" \
    "0" \
    "$exit_code"

test_assert \
    "Full flow: removes ~/.oh-my-zsh" \
    "[[ ! -d $TEST_HOME/.oh-my-zsh ]]"

test_assert \
    "Full flow: consumes .zshrc.pre-oh-my-zsh" \
    "[[ ! -f $TEST_HOME/.zshrc.pre-oh-my-zsh ]]"

test_assert_contains \
    "Full flow: keeps base content" \
    "$result" \
    'alias ll="ls -la"'

test_assert_contains \
    "Full flow: recovers user's post-install additions" \
    "$result" \
    'export MY_CUSTOM_VAR="hello"'

test_assert_contains \
    "Full flow: recovers user's post-install alias" \
    "$result" \
    'alias gs="git status"'

test_assert_not_contains \
    "Full flow: strips oh-my-zsh ZSH_THEME line" \
    "$result" \
    "ZSH_THEME"

test_assert_not_contains \
    "Full flow: strips oh-my-zsh source line" \
    "$result" \
    "source \$ZSH/oh-my-zsh.sh"

test_assert_not_contains \
    "Full flow: strips oh-my-zsh plugins= line" \
    "$result" \
    "plugins=("

rm -rf "$TEST_HOME"

# --- Test 4: no pre-install base, and uninstall leaves no .zshrc to restore ---
# (uninstall_oh_my_zsh renames .zshrc away with nothing to put back) ->
# recover directly from the backup.
TEST_HOME=$(mktemp -d)
mkdir -p "$TEST_HOME/.oh-my-zsh"

cat > "$TEST_HOME/.zshrc" << 'EOF'
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="agnoster"
plugins=(git)
source $ZSH/oh-my-zsh.sh

export EDITOR="nvim"
alias gs="git status"
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --yes 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc" 2>/dev/null || echo "")

test_assert_eq \
    "No base, no restore target: exits zero" \
    "0" \
    "$exit_code"

test_assert_contains \
    "No base, no restore target: recovers customizations" \
    "$result" \
    'export EDITOR="nvim"'

test_assert_not_contains \
    "No base, no restore target: strips oh-my-zsh boilerplate" \
    "$result" \
    "ZSH_THEME"

rm -rf "$TEST_HOME"

# --- Test 5: nothing but oh-my-zsh's own template -> no false "recovered" claim ---
TEST_HOME=$(mktemp -d)
mkdir -p "$TEST_HOME/.oh-my-zsh"

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
EOF

cat > "$TEST_HOME/.zshrc" << 'EOF'
export PATH="$HOME/bin:$PATH"

export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="agnoster"
plugins=(git)
source $ZSH/oh-my-zsh.sh
EOF

output=$(HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --yes 2>&1) && exit_code=0 || exit_code=$?
result=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Nothing beyond template: exits zero" \
    "0" \
    "$exit_code"

test_assert_contains \
    "Nothing beyond template: says nothing to recover" \
    "$output" \
    "nothing to recover"

test_assert_eq \
    "Nothing beyond template: .zshrc is just the restored base" \
    'export PATH="$HOME/bin:$PATH"' \
    "$result"

rm -rf "$TEST_HOME"

# --- Test 6: --dry-run makes no changes, even without --yes ---
TEST_HOME=$(mktemp -d)
mkdir -p "$TEST_HOME/.oh-my-zsh"

cat > "$TEST_HOME/.zshrc.pre-oh-my-zsh" << 'EOF'
export PATH="$HOME/bin:$PATH"
EOF
cat > "$TEST_HOME/.zshrc" << 'EOF'
export PATH="$HOME/bin:$PATH"
export ZSH="$HOME/.oh-my-zsh"
source $ZSH/oh-my-zsh.sh
export MY_VAR="test"
EOF

before_zshrc=$(cat "$TEST_HOME/.zshrc")
HOME="$TEST_HOME" "$PROJECT_ROOT/devboost.sh" migrate-from-oh-my-zsh --dry-run >/dev/null 2>&1 || true
after_zshrc=$(cat "$TEST_HOME/.zshrc")

test_assert_eq \
    "Dry-run: .zshrc left untouched" \
    "$before_zshrc" \
    "$after_zshrc"

test_assert \
    "Dry-run: ~/.oh-my-zsh left untouched" \
    "[[ -d $TEST_HOME/.oh-my-zsh ]]"

test_assert \
    "Dry-run: .zshrc.pre-oh-my-zsh left untouched" \
    "[[ -f $TEST_HOME/.zshrc.pre-oh-my-zsh ]]"

rm -rf "$TEST_HOME"

test_suite_end
