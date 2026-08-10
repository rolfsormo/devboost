package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZshIncludeBlockCreatesFileWhenAbsent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), zshIncludeStart) {
		t.Fatalf("expected include block in created file, got %q", data)
	}
}

func TestZshIncludeBlockAppendsWhenAbsentFromExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(path, []byte("existing stuff\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "existing stuff") || !strings.Contains(got, zshIncludeStart) {
		t.Fatalf("expected both existing content and include block, got %q", got)
	}
}

func TestZshIncludeBlockNilWhenAlreadyPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = z.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op once present, got %+v", op)
	}
}

// TestZshIncludeBlockWarnsInsteadOfDoubleSourcing is the load-bearing
// test for this module's whole reason to exist as a bespoke kind: an
// unmarked pre-existing line already sourcing .zshrc.devboost (e.g. left
// over from a manual edit or oh-my-zsh migration recovery) must not get
// a second, marked block appended on top — that would source
// .zshrc.devboost twice on every shell start, doubling the cost of
// everything in it. This was a real production bug fixed earlier in the
// bash tool; the Go port must not regress it.
func TestZshIncludeBlockWarnsInsteadOfDoubleSourcing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	unmarkedLine := `[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"` + "\n"
	if err := os.WriteFile(path, []byte(unmarkedLine), 0o644); err != nil {
		t.Fatal(err)
	}

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op describing the conflict, not nil")
	}
	if !strings.Contains(op.Description, "WARNING") {
		t.Fatalf("expected a warning description, got %q", op.Description)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute (acknowledge-only) should not error: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if got != unmarkedLine {
		t.Fatalf("expected the file to be left byte-for-byte unchanged, got %q", got)
	}
	if strings.Contains(got, zshIncludeStart) {
		t.Fatalf("expected no marked block to be injected (would double-source), got %q", got)
	}
}

// TestZshIncludeBlockRegexMatchesEvenInLooseComments documents an actual
// weakness inherited from the bash version's own detection regex
// ((^|[^#].*)\.zshrc\.devboost): [^#] only requires the character
// immediately before the match to not be '#', so a comment like
// "# see .zshrc.devboost for details" (space before the match, not '#')
// still matches and triggers the same warn-and-skip path, even though
// it's just a comment, not a real unmarked source line. This is existing
// bash behavior (confirmed by running the actual grep -Eq command), not
// something introduced by the port — ported faithfully rather than
// silently "fixed," since unlike the unambiguous node_modules bug, this
// one is a real product-behavior question (how strict should comment
// detection be?) worth a deliberate decision, not a silent unilateral
// change during a port.
func TestZshIncludeBlockRegexMatchesEvenInLooseComments(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	commentOnly := "# see .zshrc.devboost for details\n"
	if err := os.WriteFile(path, []byte(commentOnly), 0o644); err != nil {
		t.Fatal(err)
	}

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if !strings.Contains(op.Description, "WARNING") {
		t.Fatalf("expected the (imperfect, faithfully-ported) regex to still flag this as a false positive, got %q", op.Description)
	}
}

func TestZshIncludeBlockDoesNotFlagDirectlyCommentedOutLine(t *testing.T) {
	// The one comment shape the bash regex DOES correctly exclude: '#'
	// immediately adjacent to the match, no characters in between.
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".zshrc")
	commentOnly := "#.zshrc.devboost\n"
	if err := os.WriteFile(path, []byte(commentOnly), 0o644); err != nil {
		t.Fatal(err)
	}

	z := zshIncludeBlock{zshrcPath: path}
	op, err := z.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if strings.Contains(op.Description, "WARNING") {
		t.Fatalf("expected normal append for a directly-adjacent-# comment, got %q", op.Description)
	}
}
