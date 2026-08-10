package main

import "testing"

func TestParseArgsDefaultsToApply(t *testing.T) {
	f := parseArgs(nil)
	if f.cmd != "apply" {
		t.Fatalf("got cmd %q, want %q", f.cmd, "apply")
	}
}

func TestParseArgsRecognizesUndo(t *testing.T) {
	f := parseArgs([]string{"undo"})
	if f.cmd != "undo" {
		t.Fatalf("got cmd %q, want %q", f.cmd, "undo")
	}
}

func TestParseArgsNoOptimizations(t *testing.T) {
	f := parseArgs([]string{"apply", "--no-optimizations"})
	if f.cmd != "apply" {
		t.Fatalf("got cmd %q, want %q", f.cmd, "apply")
	}
	if !f.noOptimizations {
		t.Fatal("expected noOptimizations to be true")
	}
}

func TestParseArgsNoOptimizationsDefaultsFalse(t *testing.T) {
	f := parseArgs([]string{"apply"})
	if f.noOptimizations {
		t.Fatal("expected noOptimizations to default to false")
	}
}

func TestParseArgsUndoWithDryRun(t *testing.T) {
	f := parseArgs([]string{"undo", "--dry-run"})
	if f.cmd != "undo" {
		t.Fatalf("got cmd %q, want %q", f.cmd, "undo")
	}
	if !f.dryRun {
		t.Fatal("expected dryRun to be true")
	}
}

func TestParseArgsConfigFlag(t *testing.T) {
	f := parseArgs([]string{"apply", "--config", "/tmp/custom.yaml"})
	if f.configPath != "/tmp/custom.yaml" {
		t.Fatalf("got configPath %q, want %q", f.configPath, "/tmp/custom.yaml")
	}
}

func TestParseArgsUndoWithForce(t *testing.T) {
	f := parseArgs([]string{"undo", "--force"})
	if f.cmd != "undo" {
		t.Fatalf("got cmd %q, want %q", f.cmd, "undo")
	}
	if !f.force {
		t.Fatal("expected force to be true")
	}
}

func TestParseArgsForceDefaultsFalse(t *testing.T) {
	f := parseArgs([]string{"undo"})
	if f.force {
		t.Fatal("expected force to default to false")
	}
}
