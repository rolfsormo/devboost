package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestSecurityDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "security:\n  enable: false\n")
	if got := Security(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

// TestSecurityDegradesGracefullyWhenZshDisabled is a regression test for
// the security_check_alias resource's DependsOn on zsh's
// zshrc_devboost — that dependency only resolves if both modules' resources
// are combined in the same list AND zshrc_devboost actually exists in it.
// zsh.enable: false is a valid, sensible config combination (security
// alone doesn't need zsh's plugin manager), so Security must not declare
// a resource whose dependency can never be satisfied.
func TestSecurityDegradesGracefullyWhenZshDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "zsh:\n  enable: false\n")
	if got := Security(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when zsh is disabled (its DependsOn target wouldn't exist), got %v", got)
	}
}

func TestSecurityEnabledByDefaultInjectsAliasBlock(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Security(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	b, ok := got[0].Kind.(kinds.BlockInFile)
	if !ok {
		t.Fatalf("expected kinds.BlockInFile, got %T", got[0].Kind)
	}
	if !strings.Contains(b.Content, "devboost-check()") {
		t.Fatalf("expected devboost-check function in block content, got %q", b.Content)
	}
}

func TestSecurityDiagnosticsNoneWhenClean(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(diags) != 1 || diags[0].Warn {
		t.Fatalf("expected a single non-warning 'no issues' diagnostic, got %+v", diags)
	}
}

func TestSecurityDiagnosticsWarnsOnLatestPin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "toolchains:\n  globals:\n    node: latest\n")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, d := range diags {
		if d.Warn && strings.Contains(d.Message, "'node'") && strings.Contains(d.Message, "latest") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning about node pinned to latest, got %+v", diags)
	}
}

func TestSecurityDiagnosticsWarnsOnOhMyZshPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := loadFixtureConfig(t, "")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, d := range diags {
		if d.Warn && strings.Contains(d.Message, "oh-my-zsh") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an oh-my-zsh warning, got %+v", diags)
	}
}

func TestSecurityDiagnosticsWarnsOnDoubleSourcedZshrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".zshrc"),
		"source ~/.zshrc.devboost\nsource ~/.zshrc.devboost\n")
	cfg := loadFixtureConfig(t, "")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, d := range diags {
		if d.Warn && strings.Contains(d.Message, "double-sourced") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a double-sourced warning, got %+v", diags)
	}
}

func TestSecurityDiagnosticsDisabledReturnsNil(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "security:\n  enable: false\n")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diags != nil {
		t.Fatalf("expected nil diagnostics when disabled, got %+v", diags)
	}
}
