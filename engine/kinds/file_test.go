package kinds

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileDiffPendingWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "f")
	f := File{Path: path, Content: "hello\n", NoBackup: true}
	op, err := f.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op for an absent file")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "hello\n" {
		t.Fatalf("got %q, err %v; want %q", data, err, "hello\n")
	}
}

func TestFileDiffNilWhenContentMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := File{Path: path, Content: "same\n"}
	op, err := f.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op when content matches, got %+v", op)
	}
}

func TestFileDiffPendingWhenContentDiffers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := File{Path: path, Content: "new\n", NoBackup: true}
	op, err := f.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when content differs")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new\n" {
		t.Fatalf("got %q, want %q", data, "new\n")
	}
}

func TestFileExecuteBacksUpByDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, "target")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := File{Path: path, Content: "new\n"} // NoBackup left false
	op, err := f.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	dir, err := DefaultBackupDir()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected at least one backup entry under %s, err=%v", dir, err)
	}
}
