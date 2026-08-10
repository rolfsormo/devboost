package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallRemovesZshrcDevboost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	includeFile := filepath.Join(home, ".zshrc.devboost")
	writeFile(t, includeFile, "generated content\n")

	cfg := loadFixtureConfig(t, "")
	if err := Uninstall(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(includeFile); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", includeFile, err)
	}
}

func TestUninstallRemovesDevboostBlockFromZshrcKeepingUserContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	writeFile(t, zshrc,
		"my custom alias\n"+
			zshIncludeStart+"\n"+
			`[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"`+"\n"+
			zshIncludeEnd+"\n"+
			"my other custom line\n")

	cfg := loadFixtureConfig(t, "")
	if err := Uninstall(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("expected .zshrc to still exist (user content, not devboost's), got: %v", err)
	}
	got := string(data)
	if strings.Contains(got, zshIncludeStart) {
		t.Fatalf("expected devboost's block removed, got %q", got)
	}
	if !strings.Contains(got, "my custom alias") || !strings.Contains(got, "my other custom line") {
		t.Fatalf("expected user's own content preserved, got %q", got)
	}
}

func TestUninstallRemovesDirenvrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	direnvrc := filepath.Join(home, ".direnvrc")
	writeFile(t, direnvrc, "use_mise() { eval \"$(mise activate direnv)\"; }\n")

	cfg := loadFixtureConfig(t, "")
	if err := Uninstall(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(direnvrc); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed", direnvrc)
	}
}

func TestUninstallIsNoOpOnAlreadyCleanHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")
	// Nothing devboost-managed exists at all — should not error.
	if err := Uninstall(cfg); err != nil {
		t.Fatalf("expected uninstall on a clean home to be a no-op, got: %v", err)
	}
}

func TestUninstallBacksUpBeforeRemoving(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	includeFile := filepath.Join(home, ".zshrc.devboost")
	writeFile(t, includeFile, "generated content\n")

	cfg := loadFixtureConfig(t, "")
	if err := Uninstall(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupRoot := filepath.Join(home, ".devboost", "backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected at least one backup under %s, err=%v", backupRoot, err)
	}
}
