package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".devboost.yaml")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestLoadMissingFileYieldsEmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "default" {
		t.Fatalf("got %q, want default", got)
	}
}

func TestLoadMalformedFileErrors(t *testing.T) {
	path := writeFixture(t, "zsh:\n  - this is not valid: [")
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for malformed YAML")
	}
}

func TestGetReadsNestedKey(t *testing.T) {
	path := writeFixture(t, "zsh:\n  znap_path: /custom/path\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "/custom/path" {
		t.Fatalf("got %q, want /custom/path", got)
	}
}

func TestGetFallsBackToDefaultWhenKeyAbsent(t *testing.T) {
	path := writeFixture(t, "zsh:\n  other_key: value\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "default" {
		t.Fatalf("got %q, want default", got)
	}
}

func TestGetFallsBackToDefaultWhenIntermediateSegmentAbsent(t *testing.T) {
	path := writeFixture(t, "other:\n  key: value\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "default" {
		t.Fatalf("got %q, want default", got)
	}
}

func TestGetFallsBackToDefaultWhenValueIsNotAString(t *testing.T) {
	path := writeFixture(t, "zsh:\n  znap_path:\n    nested: true\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "default" {
		t.Fatalf("got %q, want default", got)
	}
}

// TestGetExpandsHomeInDefault is a regression test for a real bug found
// during the znap spike: expansion only ran on values actually read from
// the file, not on the default — so a default like "~/.zsh-snap" with no
// config file present was passed through to git clone literally,
// including the tilde, which git happily "cloned into" as a real
// directory named "~".
func TestGetExpandsHomeInDefault(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}
	got := cfg.Get("zsh.znap_path", "~/.zsh-snap")
	want := filepath.Join(home, ".zsh-snap")
	if got != want {
		t.Fatalf("got %q, want %q (default was not tilde-expanded)", got, want)
	}
}

func TestGetExpandsHomeInConfiguredValue(t *testing.T) {
	path := writeFixture(t, "zsh:\n  znap_path: \"~/custom-znap\"\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}
	got := cfg.Get("zsh.znap_path", "default")
	want := filepath.Join(home, "custom-znap")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetLeavesNonTildeValuesUntouched(t *testing.T) {
	path := writeFixture(t, "zsh:\n  znap_path: /absolute/path\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "/absolute/path" {
		t.Fatalf("got %q, want /absolute/path", got)
	}
}
