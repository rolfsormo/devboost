package kinds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLineInFileDiffNilWhenAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "missing")
	l := LineInFile{Path: path, Pattern: "zinit", MigrationID: "test"}
	op, err := l.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op for a missing file, got %+v", op)
	}
}

func TestLineInFileDiffPendingWhenUnmarkedMatchExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path, "zinit light zsh-users/zsh-autosuggestions", "eval \"$(mise activate zsh)\"")

	l := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	op, err := l.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op for an unmarked matching line")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, MarkerFor("zinit-znap-dup")+"zinit light") {
		t.Fatalf("expected disabled line to carry the marker, got %q", got)
	}
	if !strings.Contains(got, "mise activate zsh") {
		t.Fatalf("expected unrelated line to remain untouched, got %q", got)
	}
}

func TestLineInFileDiffNilOnceMarked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path, "zinit light zsh-users/zsh-autosuggestions")

	l := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	op, err := l.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = l.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op once marked, got %+v", op)
	}
}

// TestLineInFileRespectsManualRestore is the load-bearing test for the
// mechanism's whole point: a user peeling the marker off a previously
// disabled line by hand is an explicit override that later runs must not
// undo.
func TestLineInFileRespectsManualRestore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path, "zinit light zsh-users/zsh-autosuggestions")

	l := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	op, err := l.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	// User manually removes the marker prefix, restoring the line.
	data, _ := os.ReadFile(path)
	restored := strings.ReplaceAll(string(data), MarkerFor("zinit-znap-dup"), "")
	if err := os.WriteFile(path, []byte(restored), 0o644); err != nil {
		t.Fatal(err)
	}

	op, err = l.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected the manual restore to be respected (no re-disable), got %+v", op)
	}
}

func TestLineInFileLeavesNonMatchingLinesAlone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path,
		"zinit light zsh-users/zsh-autosuggestions",
		"zinit light zsh-users/zsh-completions",
	)

	// Only the autosuggestions line duplicates znap; completions doesn't.
	l := LineInFile{Path: path, Pattern: "zsh-autosuggestions", MigrationID: "zinit-znap-dup"}
	op, err := l.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Contains(got, MarkerFor("zinit-znap-dup")+"zinit light zsh-users/zsh-completions") {
		t.Fatalf("expected the non-matching completions line to stay unmarked, got %q", got)
	}
	if !strings.Contains(got, "\nzinit light zsh-users/zsh-completions") && !strings.HasPrefix(got, "zinit light zsh-users/zsh-completions") {
		t.Fatalf("expected the non-matching line to remain present, got %q", got)
	}
}

func TestLineInFileUndoRestoresExactOriginalLine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	original := `zinit light zsh-users/zsh-autosuggestions # trailing comment, tabs	and spaces`
	writeLines(t, path, original, "eval \"$(mise activate zsh)\"")

	l := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	op, err := l.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	undoOp, err := l.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if undoOp == nil {
		t.Fatal("expected a pending undo op for a marked line")
	}
	if err := undoOp.Execute(); err != nil {
		t.Fatalf("undo execute: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, original) {
		t.Fatalf("expected the original line restored exactly, got %q", got)
	}
	if strings.Contains(got, MarkerPrefix) {
		t.Fatalf("expected no marker remaining after undo, got %q", got)
	}
	if !strings.Contains(got, "mise activate zsh") {
		t.Fatalf("expected unrelated line to remain untouched, got %q", got)
	}
}

func TestLineInFileUndoNilWhenNothingMarked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path, "eval \"$(mise activate zsh)\"")

	l := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	op, err := l.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending undo op when nothing is marked, got %+v", op)
	}
}

func TestLineInFileUndoNilWhenFileMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	l := LineInFile{Path: filepath.Join(home, "missing"), Pattern: "zinit", MigrationID: "test"}
	op, err := l.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending undo op for a missing file, got %+v", op)
	}
}

func TestLineInFileUndoOnlyRestoresOwnMigration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	writeLines(t, path, "zinit light zsh-users/zsh-autosuggestions", `. "/x/nvm.sh"`)

	zinit := LineInFile{Path: path, Pattern: "zinit light", MigrationID: "zinit-znap-dup"}
	if op, err := zinit.Diff(); err != nil || op == nil {
		t.Fatalf("setup zinit diff: op=%+v err=%v", op, err)
	} else if err := op.Execute(); err != nil {
		t.Fatalf("setup zinit execute: %v", err)
	}

	nvm := LineInFile{Path: path, Pattern: `nvm\.sh`, MigrationID: "nvm-mise-dup"}
	if op, err := nvm.Diff(); err != nil || op == nil {
		t.Fatalf("setup nvm diff: op=%+v err=%v", op, err)
	} else if err := op.Execute(); err != nil {
		t.Fatalf("setup nvm execute: %v", err)
	}

	// Undoing zinit's migration must not touch nvm's marked line.
	undoOp, err := zinit.Undo()
	if err != nil || undoOp == nil {
		t.Fatalf("op=%+v err=%v", undoOp, err)
	}
	if err := undoOp.Execute(); err != nil {
		t.Fatalf("undo execute: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Contains(got, MarkerFor("zinit-znap-dup")) {
		t.Fatalf("expected zinit's marker gone after its own undo, got %q", got)
	}
	if !strings.Contains(got, MarkerFor("nvm-mise-dup")) {
		t.Fatalf("expected nvm's marker to remain untouched, got %q", got)
	}
}
