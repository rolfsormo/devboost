package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateFromOhMyZshNoYesRefuses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := MigrateFromOhMyZsh(false, false)
	if err == nil {
		t.Fatal("expected an error without --yes")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected the error to mention --yes, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".oh-my-zsh")); statErr != nil {
		t.Fatal("expected ~/.oh-my-zsh left untouched without --yes")
	}
}

func TestMigrateFromOhMyZshNothingToDo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No ~/.oh-my-zsh and no backup present at all.
	err := MigrateFromOhMyZsh(false, true)
	if err == nil {
		t.Fatal("expected an error when there's nothing to recover")
	}
	if !strings.Contains(err.Error(), "nothing to recover") {
		t.Fatalf("expected a 'nothing to recover' error, got: %v", err)
	}
}

func TestMigrateFromOhMyZshFullFlowWithPreInstallBase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".zshrc.pre-oh-my-zsh"),
		"export PATH=\"$HOME/bin:$PATH\"\nalias ll=\"ls -la\"\n")
	writeFile(t, filepath.Join(home, ".zshrc"),
		"export PATH=\"$HOME/bin:$PATH\"\n"+
			"alias ll=\"ls -la\"\n\n"+
			"# Path to your Oh My Zsh installation.\n"+
			"export ZSH=\"$HOME/.oh-my-zsh\"\n"+
			"ZSH_THEME=\"agnoster\"\n"+
			"plugins=(git docker kubectl)\n"+
			"source $ZSH/oh-my-zsh.sh\n\n"+
			"export EDITOR=\"nvim\"\n"+
			"alias gs=\"git status\"\n"+
			"export MY_CUSTOM_VAR=\"hello\"\n")

	if err := MigrateFromOhMyZsh(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh")); !os.IsNotExist(err) {
		t.Fatal("expected ~/.oh-my-zsh removed from its original location")
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc.pre-oh-my-zsh")); !os.IsNotExist(err) {
		t.Fatal("expected .zshrc.pre-oh-my-zsh consumed")
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	for _, want := range []string{
		`alias ll="ls -la"`,
		`export MY_CUSTOM_VAR="hello"`,
		`alias gs="git status"`,
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected result to contain %q, got %q", want, result)
		}
	}
	for _, unwanted := range []string{"ZSH_THEME", "source $ZSH/oh-my-zsh.sh", "plugins=("} {
		if strings.Contains(result, unwanted) {
			t.Fatalf("expected oh-my-zsh template line %q stripped, got %q", unwanted, result)
		}
	}
}

func TestMigrateFromOhMyZshNoBaseRecoversFromBackupDirectly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".zshrc"),
		"export ZSH=\"$HOME/.oh-my-zsh\"\n"+
			"ZSH_THEME=\"agnoster\"\n"+
			"plugins=(git)\n"+
			"source $ZSH/oh-my-zsh.sh\n\n"+
			"export EDITOR=\"nvim\"\n"+
			"alias gs=\"git status\"\n")

	if err := MigrateFromOhMyZsh(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)
	if !strings.Contains(result, `export EDITOR="nvim"`) {
		t.Fatalf("expected customizations recovered, got %q", result)
	}
	if strings.Contains(result, "ZSH_THEME") {
		t.Fatalf("expected oh-my-zsh boilerplate stripped, got %q", result)
	}
}

func TestMigrateFromOhMyZshNothingBeyondTemplate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".zshrc.pre-oh-my-zsh"), `export PATH="$HOME/bin:$PATH"`+"\n")
	writeFile(t, filepath.Join(home, ".zshrc"),
		`export PATH="$HOME/bin:$PATH"`+"\n\n"+
			"export ZSH=\"$HOME/.oh-my-zsh\"\n"+
			"ZSH_THEME=\"agnoster\"\n"+
			"plugins=(git)\n"+
			"source $ZSH/oh-my-zsh.sh\n")

	if err := MigrateFromOhMyZsh(false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	result := strings.TrimRight(string(data), "\n")
	if result != `export PATH="$HOME/bin:$PATH"` {
		t.Fatalf("expected .zshrc to be just the restored base, got %q", result)
	}
}

func TestMigrateFromOhMyZshDryRunMakesNoChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".zshrc.pre-oh-my-zsh"), `export PATH="$HOME/bin:$PATH"`+"\n")
	before := `export PATH="$HOME/bin:$PATH"` + "\n" +
		"export ZSH=\"$HOME/.oh-my-zsh\"\n" +
		"source $ZSH/oh-my-zsh.sh\n" +
		`export MY_VAR="test"` + "\n"
	writeFile(t, filepath.Join(home, ".zshrc"), before)

	if err := MigrateFromOhMyZsh(true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("expected .zshrc untouched by dry-run, got %q", after)
	}
	if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh")); err != nil {
		t.Fatal("expected ~/.oh-my-zsh untouched by dry-run")
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc.pre-oh-my-zsh")); err != nil {
		t.Fatal("expected .zshrc.pre-oh-my-zsh untouched by dry-run")
	}
}
