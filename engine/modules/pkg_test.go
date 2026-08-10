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

// TestPkgProvidesMatchesItsActualPackageList is a regression test for
// the NeedsProvider mechanism's correctness: base_packages.Provides
// must reflect exactly what this resource actually installs on THIS
// platform, so other modules' NeedsProvider: []string{"mise"} (etc.)
// resolves correctly regardless of which packages got excluded by the
// gap filter. On Ubuntu, mise is excluded (see the gap test above), so
// it must NOT appear in Provides there either — only on platforms where
// base_packages genuinely installs it.
func TestPkgProvidesMatchesItsActualPackageList(t *testing.T) {
	cfg := loadFixtureConfig(t, "")

	darwin := Pkg(cfg, kinds.OSDarwin)[0]
	if !containsString(darwin.Provides, "mise") {
		t.Fatalf("expected base_packages to Provide mise on Darwin, got %v", darwin.Provides)
	}

	ubuntu := Pkg(cfg, kinds.OSLinuxUbuntu)[0]
	if containsString(ubuntu.Provides, "mise") {
		t.Fatalf("expected base_packages to NOT Provide mise on Ubuntu (it's a real apt gap, converged elsewhere), got %v", ubuntu.Provides)
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
