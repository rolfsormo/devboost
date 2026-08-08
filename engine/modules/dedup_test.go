package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestZinitZnapDedupNoResourceWhenNoMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"), "eval \"$(mise activate zsh)\"\n")

	cfg := loadFixtureConfig(t, "")
	if got := ZinitZnapDedup(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when no zinit duplication present, got %v", got)
	}
}

func TestZinitZnapDedupDetectsDuplicate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"),
		"zinit light zdharma-continuum/fast-syntax-highlighting\n"+
			"zinit light zsh-users/zsh-autosuggestions\n"+
			"zinit light zsh-users/zsh-completions\n")

	cfg := loadFixtureConfig(t, "")
	got := ZinitZnapDedup(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}

	op, err := got[0].Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	got2 := string(data)
	if !containsAll(got2,
		kinds.MarkerFor("zinit-znap-dup")+"zinit light zdharma-continuum/fast-syntax-highlighting",
		kinds.MarkerFor("zinit-znap-dup")+"zinit light zsh-users/zsh-autosuggestions",
	) {
		t.Fatalf("expected both duplicate lines disabled, got %q", got2)
	}
	if containsAll(got2, kinds.MarkerFor("zinit-znap-dup")+"zinit light zsh-users/zsh-completions") {
		t.Fatalf("expected the non-duplicate completions line to stay untouched, got %q", got2)
	}
}

func TestAsdfMiseDedupDetectsDuplicate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"),
		". /opt/homebrew/opt/asdf/libexec/asdf.sh\n"+
			"eval \"$(mise activate zsh)\"\n")

	cfg := loadFixtureConfig(t, "")
	got := AsdfMiseDedup(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
}

func TestAsdfMiseDedupNoResourceWhenNoMatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"), "eval \"$(mise activate zsh)\"\n")

	cfg := loadFixtureConfig(t, "")
	if got := AsdfMiseDedup(cfg); len(got) != 0 {
		t.Fatalf("expected no resources, got %v", got)
	}
}

func TestNvmMiseDedupTargetsZprofileNotZshrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// nvm's dedup targets ~/.zprofile — put the redundant line ONLY there,
	// confirming this module doesn't (incorrectly) look at .zshrc instead.
	writeFile(t, filepath.Join(home, ".zprofile"),
		`export NVM_DIR="$HOME/.nvm"`+"\n"+
			`[ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"`+"\n")

	cfg := loadFixtureConfig(t, "")
	got := NvmMiseDedup(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}

	op, err := got[0].Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(home, ".zprofile"))
	got2 := string(data)
	if !containsAll(got2, "NVM_DIR") {
		t.Fatalf("expected NVM_DIR export to remain (harmless, not the expensive part), got %q", got2)
	}
	if !containsAll(got2, kinds.MarkerFor("nvm-mise-dup")) {
		t.Fatalf("expected the nvm source line to be disabled, got %q", got2)
	}
}

func TestDedupModulesDisabledByLegacyShellEnable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"), "zinit light zsh-users/zsh-autosuggestions\n")
	writeFile(t, filepath.Join(home, ".zprofile"), `[ -s "/x/nvm.sh" ] && \. "/x/nvm.sh"`+"\n")

	cfg := loadFixtureConfig(t, "legacy_shell:\n  enable: false\n")
	if got := ZinitZnapDedup(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when legacy_shell disabled, got %v", got)
	}
	if got := NvmMiseDedup(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when legacy_shell disabled, got %v", got)
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}
