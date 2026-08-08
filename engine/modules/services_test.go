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
