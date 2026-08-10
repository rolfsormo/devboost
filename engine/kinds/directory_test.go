package kinds

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirExistsDiffPendingWhenAbsent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "target")
	op, err := DirExists{Path: dir}.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op for an absent directory")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected %s to be a directory after execute", dir)
	}
}

func TestDirExistsDiffNilWhenPresent(t *testing.T) {
	dir := t.TempDir()
	op, err := DirExists{Path: dir}.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op for an existing directory, got %+v", op)
	}
}

func TestDirExistsErrorsWhenPathIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A file exists at the path but isn't a directory: Diff should report
	// it as pending (create-directory would fail at Execute, which is the
	// correct place for that failure to surface — Diff itself shouldn't
	// silently treat "a file is here" as "the directory exists").
	op, err := DirExists{Path: file}.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when a non-directory occupies the path")
	}
	if err := op.Execute(); err == nil {
		t.Fatal("expected Execute to fail when a file occupies the target path")
	}
}
