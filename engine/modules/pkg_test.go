package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestPkgUsesDefaultsWhenUnconfigured(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Pkg(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	p, ok := got[0].Kind.(kinds.Package)
	if !ok {
		t.Fatalf("expected a kinds.Package resource, got %T", got[0].Kind)
	}
	if len(p.Names) == 0 {
		t.Fatal("expected default package list to be non-empty")
	}
}

func TestPkgUsesConfiguredList(t *testing.T) {
	cfg := loadFixtureConfig(t, "packages:\n  base:\n    - zsh\n    - tmux\n")
	got := Pkg(cfg)
	p, ok := got[0].Kind.(kinds.Package)
	if !ok {
		t.Fatalf("expected a kinds.Package resource, got %T", got[0].Kind)
	}
	if len(p.Names) != 2 || p.Names[0] != "zsh" || p.Names[1] != "tmux" {
		t.Fatalf("got %v, want [zsh tmux]", p.Names)
	}
}
