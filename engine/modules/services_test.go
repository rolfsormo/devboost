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
// brew install) earlier in the same apply run. Availability is now
// checked live inside the registered command's Satisfied.
func TestServicesAlwaysDeclaresResourceOnDarwinWhenEnabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Services(cfg, kinds.OSDarwin)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource regardless of whether atuin is currently on PATH, got %d", len(got))
	}
}
