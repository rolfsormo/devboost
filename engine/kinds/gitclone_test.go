package kinds

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newLocalRepo creates a tiny local git repo to clone from, so the test
// doesn't depend on network access.
func newLocalRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "f")
	run("commit", "-q", "-m", "init")
	return dir
}

func TestGitCloneDiffPendingWhenAbsent(t *testing.T) {
	src := newLocalRepo(t)
	dest := filepath.Join(t.TempDir(), "dest")

	op, err := GitClone{URL: src, Dest: dest}.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when dest doesn't exist")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "f")); err != nil {
		t.Fatalf("expected cloned file to exist: %v", err)
	}
}

func TestGitCloneDiffNilWhenAlreadyCloned(t *testing.T) {
	src := newLocalRepo(t)
	dest := filepath.Join(t.TempDir(), "dest")

	op, err := GitClone{URL: src, Dest: dest}.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: expected a pending op, got op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = GitClone{URL: src, Dest: dest}.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op once already cloned, got %+v", op)
	}
}
