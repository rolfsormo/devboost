package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

func omzResource(t *testing.T) engine.Resource {
	t.Helper()
	cfg := loadFixtureConfig(t, "")
	got := OhMyZshMigration(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	return got[0]
}

func TestOhMyZshMigrationDisabledByOptimizeEnable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := loadFixtureConfig(t, "optimize:\n  enable: false\n")
	if got := OhMyZshMigration(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when optimize disabled, got %v", got)
	}
}

func TestOhMyZshMigrationSatisfiedWhenAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No ~/.oh-my-zsh at all.
	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op when ~/.oh-my-zsh is absent, got %+v", op)
	}
}

func TestOhMyZshMigrationPendingWhenPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when ~/.oh-my-zsh is present")
	}
}

func TestOhMyZshMigrationConvergesAutomatically(t *testing.T) {
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

	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	// No confirmation gate, no --yes: Execute converges directly, same
	// as any other resource's Diff/Execute — this is the core behavior
	// change from the old migrate-from-oh-my-zsh --yes-gated subcommand.
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh")); !os.IsNotExist(err) {
		t.Fatal("expected ~/.oh-my-zsh removed from its original location")
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

	// And now Satisfied should report true — no more pending op.
	op2, err := r.Kind.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op2 != nil {
		t.Fatalf("expected no pending op after convergence, got %+v", op2)
	}
}

func TestOhMyZshMigrationNothingBeyondTemplate(t *testing.T) {
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

	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
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

func TestOhMyZshMigrationUndoRestoresArchiveAndZshrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".oh-my-zsh", "oh-my-zsh.sh"), "# marker file\n")
	writeFile(t, filepath.Join(home, ".zshrc.pre-oh-my-zsh"),
		"export PATH=\"$HOME/bin:$PATH\"\nalias ll=\"ls -la\"\n")
	originalZshrc := "export PATH=\"$HOME/bin:$PATH\"\n" +
		"alias ll=\"ls -la\"\n\n" +
		"export ZSH=\"$HOME/.oh-my-zsh\"\n" +
		"ZSH_THEME=\"agnoster\"\n" +
		"plugins=(git)\n" +
		"source $ZSH/oh-my-zsh.sh\n\n" +
		"export MY_CUSTOM_VAR=\"hello\"\n"
	writeFile(t, filepath.Join(home, ".zshrc"), originalZshrc)

	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	undoer, ok := r.Kind.(engine.Undoer)
	if !ok {
		t.Fatal("expected omz_migration's Kind to implement engine.Undoer")
	}
	undoOp, err := undoer.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if undoOp == nil {
		t.Fatal("expected a pending undo op after a real migration")
	}
	if err := undoOp.Execute(); err != nil {
		t.Fatalf("undo execute: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh", "oh-my-zsh.sh")); err != nil {
		t.Fatalf("expected ~/.oh-my-zsh restored with its contents intact: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	// Undo restores .zshrc from the backup taken right before the
	// recovered-additions rewrite — i.e. the post-uninstallOhMyZsh,
	// pre-append state, which is .zshrc.pre-oh-my-zsh's content (since
	// that's what uninstallOhMyZsh restores into .zshrc before the
	// recovery rewrite runs).
	if !strings.Contains(string(data), `alias ll="ls -la"`) {
		t.Fatalf("expected restored .zshrc to contain pre-oh-my-zsh content, got %q", data)
	}
}

// TestOhMyZshMigrationUndoIsNotRepeatable is a regression test for the
// backup-renaming fix: a second `devboost undo` after the archived
// ~/.oh-my-zsh and .zshrc backup have already been consumed must report
// nothing pending, not re-restore (or error trying to restore) the same
// already-moved material. The archived directory renaming itself is a
// no-op the second time around (os.Rename already moved it out of the
// backup root entirely), but the .zshrc backup must be explicitly
// marked reverted, or findLatestZshrcBackup would keep finding and
// re-restoring the same snapshot on every subsequent undo.
func TestOhMyZshMigrationUndoIsNotRepeatable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".oh-my-zsh", "oh-my-zsh.sh"), "# marker file\n")
	writeFile(t, filepath.Join(home, ".zshrc.pre-oh-my-zsh"),
		"export PATH=\"$HOME/bin:$PATH\"\nalias ll=\"ls -la\"\n")
	writeFile(t, filepath.Join(home, ".zshrc"),
		"export PATH=\"$HOME/bin:$PATH\"\n"+
			"alias ll=\"ls -la\"\n\n"+
			"export ZSH=\"$HOME/.oh-my-zsh\"\n"+
			"source $ZSH/oh-my-zsh.sh\n\n"+
			"export MY_CUSTOM_VAR=\"hello\"\n")

	r := omzResource(t)
	op, err := r.Kind.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	undoer := r.Kind.(engine.Undoer)
	firstUndo, err := undoer.Undo()
	if err != nil || firstUndo == nil {
		t.Fatalf("first undo: op=%+v err=%v", firstUndo, err)
	}
	if err := firstUndo.Execute(); err != nil {
		t.Fatalf("first undo execute: %v", err)
	}

	secondUndo, err := undoer.Undo()
	if err != nil {
		t.Fatalf("unexpected error on second undo: %v", err)
	}
	if secondUndo != nil {
		t.Fatalf("expected no pending op on a second undo, got %+v", secondUndo)
	}

	// The consumed .zshrc backup should still exist on disk, just
	// renamed — never deleted, same as every other backup in this
	// codebase.
	backupRoot := filepath.Join(home, ".devboost", "backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatal(err)
	}
	foundReverted := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), revertedSuffix) {
			foundReverted = true
		}
	}
	if !foundReverted {
		t.Fatalf("expected a %s-suffixed backup directory to remain on disk, got entries: %v", revertedSuffix, entries)
	}
}

func TestOhMyZshMigrationUndoNilWhenNeverConverged(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := omzResource(t)
	undoer, ok := r.Kind.(engine.Undoer)
	if !ok {
		t.Fatal("expected omz_migration's Kind to implement engine.Undoer")
	}
	op, err := undoer.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending undo op when nothing was ever migrated, got %+v", op)
	}
}

func TestOhMyZshMigrationRegisteredCommand(t *testing.T) {
	// Sanity check that the CommandGuarded ID used by OhMyZshMigration
	// is actually registered — an unregistered ID fails loudly per
	// kinds.CommandGuarded's contract, so this would fail Diff() if the
	// init() registration and the ID string in OhMyZshMigration ever
	// drifted apart.
	c := kinds.CommandGuarded{ID: "omz_migration_converged", Wants: "x"}
	home := t.TempDir()
	os_ := os.Getenv("HOME")
	defer os.Setenv("HOME", os_)
	os.Setenv("HOME", home)
	if _, err := c.Diff(); err != nil {
		t.Fatalf("expected omz_migration_converged to be registered: %v", err)
	}
}
