package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestPkgUsesDefaultsWhenUnconfigured(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Pkg(cfg, kinds.OSDarwin)
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
	got := Pkg(cfg, kinds.OSDarwin)
	p, ok := got[0].Kind.(kinds.Package)
	if !ok {
		t.Fatalf("expected a kinds.Package resource, got %T", got[0].Kind)
	}
	if len(p.Names) != 2 || p.Names[0] != "zsh" || p.Names[1] != "tmux" {
		t.Fatalf("got %v, want [zsh tmux]", p.Names)
	}
}

// TestPkgExcludesGapPackagesOnUbuntu is a regression test for a
// real bug found via a live Ubuntu container run: several default
// packages (lazygit, mise, atuin, starship, dust) aren't in Ubuntu's
// apt repos at all, and the apt provider would fail on them every time.
// Pkg must exclude them from the apt-managed list on OSLinuxUbuntu so
// LinuxVendorInstalls (registered separately) can converge them
// instead.
func TestPkgExcludesGapPackagesOnUbuntu(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Pkg(cfg, kinds.OSLinuxUbuntu)
	p, ok := got[0].Kind.(kinds.Package)
	if !ok {
		t.Fatalf("expected a kinds.Package resource, got %T", got[0].Kind)
	}
	for _, gapName := range []string{"lazygit", "mise", "atuin", "starship", "dust"} {
		for _, name := range p.Names {
			if name == gapName {
				t.Fatalf("expected %s excluded from apt-managed packages on Ubuntu, got %v", gapName, p.Names)
			}
		}
	}
	// Packages that ARE in apt must still be present.
	found := false
	for _, name := range p.Names {
		if name == "ripgrep" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ripgrep (a real apt package) still present, got %v", p.Names)
	}
}

func TestPkgKeepsGapPackagesOnMacOS(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Pkg(cfg, kinds.OSDarwin)
	p := got[0].Kind.(kinds.Package)
	found := false
	for _, name := range p.Names {
		if name == "lazygit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected lazygit still in the brew-managed list on macOS (no gap there), got %v", p.Names)
	}
}
