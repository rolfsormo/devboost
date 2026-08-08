package modules

import "testing"

func TestCorepackDisabledWhenMiseDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "toolchains:\n  enable_mise: false\n")
	if got := Corepack(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when mise is disabled, got %v", got)
	}
}

func TestCorepackEnabledByDefault(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Corepack(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
}
