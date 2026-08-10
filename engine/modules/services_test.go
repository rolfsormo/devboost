package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestServicesDisabledWhenAtuinDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "zsh:\n  history:\n    use_atuin: false\n")
	if got := Services(cfg, kinds.OSDarwin); len(got) != 0 {
		t.Fatalf("expected no resources when atuin disabled, got %v", got)
	}
}

func TestServicesNoResourceOnLinux(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	if got := Services(cfg, kinds.OSLinuxUbuntu); len(got) != 0 {
		t.Fatalf("expected no resources on Linux (informational only), got %v", got)
	}
}

// TestServicesAlwaysDeclaresResourceOnDarwinWhenEnabled is the same
// regression shape as mise_test.go's equivalent: Services() used to
// gate itself off with exec.LookPath("atuin") at construction time,
// which could never see atuin installed by a different resource (Pkg's
// brew install) earlier in the same apply run. Ordering is now handled
// structurally via NeedsProvider (see the test below), not by gating
// construction on current PATH state.
func TestServicesAlwaysDeclaresResourceOnDarwinWhenEnabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Services(cfg, kinds.OSDarwin)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource regardless of whether atuin is currently on PATH, got %d", len(got))
	}
}

// TestServicesDependsOnAtuinProvider mirrors mise_test.go's
// TestMiseDependsOnMiseProvider: atuin_service must have a real,
// engine-enforced dependency on whichever resource actually installs
// atuin (pkg.go's base_packages on Darwin, the only platform this
// resource exists on), not rely on registration order.
func TestServicesDependsOnAtuinProvider(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Services(cfg, kinds.OSDarwin)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	needs := got[0].NeedsProvider
	if len(needs) != 1 || needs[0] != "atuin" {
		t.Fatalf("expected atuin_service to NeedsProvider [\"atuin\"], got %v", needs)
	}
}
