package kinds

import (
	"os"
	"os/exec"
	"testing"
)

// newTestRepo creates a bare local git repo dir and returns a function
// that runs a command with its cwd set there — tests use --local scope
// against this repo, never --global, so they can't touch the real
// developer's git config.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	return dir
}

// runIn shells to git config with --local against dir, matching what
// GitConfig itself does but from a fixed working directory (GitConfig has
// no explicit dir/cwd field — see the note on TestGitConfigDiff below for
// how the tests work around that).
func gitConfigLocal(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"config"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

func TestGitConfigDiffPendingWhenUnset(t *testing.T) {
	dir := newTestRepo(t)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	g := GitConfig{Key: "core.pager", Value: "delta", Scope: "--local"}
	op, err := g.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op for an unset key")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, err := gitConfigLocal(t, dir, "--local", "--get", "core.pager")
	if err != nil {
		t.Fatalf("expected core.pager to be set: %v", err)
	}
	if got != "delta\n" {
		t.Fatalf("got %q, want %q", got, "delta\n")
	}
}

func TestGitConfigDiffNilWhenAlreadyCorrect(t *testing.T) {
	dir := newTestRepo(t)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	g := GitConfig{Key: "core.pager", Value: "delta", Scope: "--local"}
	op, err := g.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = g.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op once already correct, got %+v", op)
	}
}

func TestGitConfigDiffPendingWhenValueDiffers(t *testing.T) {
	dir := newTestRepo(t)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	first := GitConfig{Key: "core.pager", Value: "less", Scope: "--local"}
	op, _ := first.Diff()
	if op != nil {
		_ = op.Execute()
	}

	changed := GitConfig{Key: "core.pager", Value: "delta", Scope: "--local"}
	op, err = changed.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when the configured value differs")
	}
}
