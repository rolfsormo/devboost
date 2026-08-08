package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanRemovesMarkedLines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	writeFile(t, zshrc,
		"my custom alias\n"+
			"# devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions\n"+
			"eval \"$(mise activate zsh)\"\n")

	cfg := loadFixtureConfig(t, "")
	if err := Clean(cfg, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Contains(got, "devboost:disabled") {
		t.Fatalf("expected marked line removed, got %q", got)
	}
	if !strings.Contains(got, "my custom alias") || !strings.Contains(got, "mise activate zsh") {
		t.Fatalf("expected other lines preserved, got %q", got)
	}
}

func TestCleanIsNoOpWhenNothingMarked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	writeFile(t, zshrc, "my custom alias\n")

	cfg := loadFixtureConfig(t, "")
	if err := Clean(cfg, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "my custom alias\n" {
		t.Fatalf("expected file untouched, got %q", string(data))
	}
}

func TestCleanIsNoOpOnMissingFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")
	if err := Clean(cfg, false); err != nil {
		t.Fatalf("expected clean on a home with no managed files to be a no-op, got: %v", err)
	}
}

func TestCleanDryRunMakesNoChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	content := "# devboost:disabled:zinit-znap-dup zinit light zsh-users/zsh-autosuggestions\n"
	writeFile(t, zshrc, content)

	cfg := loadFixtureConfig(t, "")
	if err := Clean(cfg, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Fatalf("expected dry-run to make no changes, got %q", string(data))
	}
}

func TestCleanBacksUpBeforeRemoving(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	writeFile(t, zshrc, "# devboost:disabled:zinit-znap-dup zinit light foo\n")

	cfg := loadFixtureConfig(t, "")
	if err := Clean(cfg, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupDir := filepath.Join(home, ".devboost", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("expected a backup dir to exist: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one backup to have been made before cleaning")
	}
}
