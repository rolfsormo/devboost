package kinds

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBinaryAvailableTrueForRealPathBinary(t *testing.T) {
	// "sh" is guaranteed present on any machine capable of running these
	// tests at all — a real, unmocked check of the PATH-lookup branch.
	available, err := BinaryAvailable("sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Fatal("expected sh to be found on PATH")
	}
}

func TestBinaryAvailableFalseWhenNotFoundAnywhere(t *testing.T) {
	available, err := BinaryAvailable("definitely-not-a-real-binary-devboost-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if available {
		t.Fatal("expected false for a binary that doesn't exist anywhere")
	}
}

func TestBinaryAvailableTrueWhenOnlyInManagedBinDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "devboost-test-managed-binary")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	available, err := BinaryAvailable("devboost-test-managed-binary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Fatal("expected true — the binary is in ManagedBinDir, even though it's not on this process's PATH")
	}
}
