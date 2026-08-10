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

// TestSecurityDiagnosticsNoRepoManagedWarningsWhenClean checks only the
// diagnostics SecurityDiagnostics actually controls from a clean HOME:
// no toolchain-pinned-to-latest warning, no oh-my-zsh warning, no
// double-sourced-zshrc warning. It deliberately does NOT assert "zero
// diagnostics total" — two of SecurityDiagnostics' checks (Homebrew
// index staleness, TPM remote protocol) read real, uncontrolled
// external state (the actual machine's Homebrew cache age, actual TPM
// clone if present) that a temp HOME can't isolate. Asserting a single
// "clean" diagnostic here previously broke on CI runners with a
// genuinely stale Homebrew index — a real, accurate finding the test
// wrongly treated as a failure.
func TestSecurityDiagnosticsNoRepoManagedWarningsWhenClean(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	diags, err := SecurityDiagnostics(cfg)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, d := range diags {
		if d.Warn && (strings.Contains(d.Message, "'node'") || strings.Contains(d.Message, "oh-my-zsh") || strings.Contains(d.Message, "double-sourced")) {
			t.Fatalf("expected no repo-managed-state warnings from a clean HOME, got %+v", d)
		}
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
