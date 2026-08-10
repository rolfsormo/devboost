package kinds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveBlockNoOpWhenFileAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	if err := RemoveBlock(path, start, end); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveBlockNoOpWhenMarkerAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	original := "just some content\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveBlock(path, start, end); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Fatalf("expected file untouched, got %q", data)
	}
}

func TestRemoveBlockStripsMarkedSection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "f")
	content := "before\n" + start + "\nmanaged content\n" + end + "\nafter\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveBlock(path, start, end); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Contains(got, start) || strings.Contains(got, "managed content") {
		t.Fatalf("expected marked block removed, got %q", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("expected surrounding content preserved, got %q", got)
	}
}
