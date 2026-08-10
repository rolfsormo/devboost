package kinds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const start = "# >>> devboost test start"
const end = "# <<< devboost test end"

func TestBlockInFileDiffCreatesFileWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "f")
	b := BlockInFile{Path: path, StartMarker: start, EndMarker: end, Content: "hello"}
	op, err := b.Diff()
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
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("expected content to contain block content, got %q", data)
	}
}

func TestBlockInFileDiffAppendsWhenMarkerAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("existing content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := BlockInFile{Path: path, StartMarker: start, EndMarker: end, Content: "new"}
	op, err := b.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "existing content") || !strings.Contains(string(data), "new") {
		t.Fatalf("expected both existing content and new block, got %q", data)
	}
}

func TestBlockInFileDiffReplacesExistingBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	initial := "before\n" + start + "\nold content\n" + end + "\nafter\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}
	b := BlockInFile{Path: path, StartMarker: start, EndMarker: end, Content: "new content"}
	op, err := b.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Contains(got, "old content") {
		t.Fatalf("expected old content to be replaced, got %q", got)
	}
	if !strings.Contains(got, "new content") || !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("expected new content preserved alongside before/after, got %q", got)
	}
}

func TestBlockInFileDiffNilWhenAlreadyCorrect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	b := BlockInFile{Path: path, StartMarker: start, EndMarker: end, Content: "hello"}
	op, err := b.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = b.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op once already correct, got %+v", op)
	}
}

func TestBlockInFileDiffIdempotentAfterAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := BlockInFile{Path: path, StartMarker: start, EndMarker: end, Content: "block"}
	op, err := b.Diff()
	if err != nil || op == nil {
		t.Fatalf("setup: op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("setup execute: %v", err)
	}

	op, err = b.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected idempotent no-op on second diff, got %+v", op)
	}
}
