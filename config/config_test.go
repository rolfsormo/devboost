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

func TestGetFallsBackToDefaultWhenValueIsAMap(t *testing.T) {
	path := writeFixture(t, "zsh:\n  znap_path:\n    nested: true\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("zsh.znap_path", "default"); got != "default" {
		t.Fatalf("got %q, want default", got)
	}
}

// TestGetStringifiesBoolean is a regression test: a real YAML boolean
// (enable: false, not "enable: \"false\"") was silently ignored by an
// earlier version of Get — it only recognized string-typed values, so a
// module reading .git.delta.enable would see the default ("true") even
// though the user explicitly wrote false. yq (the bash tool's reader)
// renders YAML booleans as plain "true"/"false" text, so Get must match
// that, not require users to quote their booleans.
func TestGetStringifiesBoolean(t *testing.T) {
	path := writeFixture(t, "git:\n  delta:\n    enable: false\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("git.delta.enable", "true"); got != "false" {
		t.Fatalf("got %q, want %q", got, "false")
	}
}

func TestGetStringifiesInteger(t *testing.T) {
	path := writeFixture(t, "tmux:\n  settings:\n    base_index: 1\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("tmux.settings.base_index", "0"); got != "1" {
		t.Fatalf("got %q, want %q", got, "1")
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

func TestGetListReadsStringItems(t *testing.T) {
	path := writeFixture(t, "packages:\n  base:\n    - zsh\n    - tmux\n    - fzf\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.GetList("packages.base")
	want := []string{"zsh", "tmux", "fzf"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestGetListNilWhenAbsent(t *testing.T) {
	path := writeFixture(t, "packages:\n  other: value\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetList("packages.base"); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestGetListNilWhenNotAList(t *testing.T) {
	path := writeFixture(t, "packages:\n  base: not-a-list\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GetList("packages.base"); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestSetThenGetRoundTrips(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("optimize.enable", "false")
	if got := cfg.Get("optimize.enable", "true"); got != "false" {
		t.Fatalf("got %q, want %q", got, "false")
	}
}

func TestSetCreatesIntermediateMaps(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// No "optimize" key exists at all yet — Set must create it rather
	// than panic or silently no-op.
	cfg.Set("optimize.enable", "false")
	if got := cfg.Get("optimize.enable", "true"); got != "false" {
		t.Fatalf("got %q, want %q", got, "false")
	}
}

func TestSetOverridesExistingKeyFromFile(t *testing.T) {
	path := writeFixture(t, "optimize:\n  enable: true\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("optimize.enable", "MISSING"); got != "true" {
		t.Fatalf("got %q, want %q before Set", got, "true")
	}
	cfg.Set("optimize.enable", "false")
	if got := cfg.Get("optimize.enable", "MISSING"); got != "false" {
		t.Fatalf("got %q, want %q after Set", got, "false")
	}
}
