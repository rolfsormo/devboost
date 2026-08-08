package modules

import "testing"

func TestGitDeltaDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "git:\n  delta:\n    enable: false\n")
	if got := Git(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestGitDeltaEnabledByDefault(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Git(cfg)
	if len(got) != 4 {
		t.Fatalf("expected 4 git config resources, got %d", len(got))
	}
}
