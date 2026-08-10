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

// TestCorepackDependsOnMiseToolchains is a regression test for a real
// bug found via a live Ubuntu container run: corepack needs mise's node
// toolchain to have already converged (corepack ships bundled with
// Node, or needs npm — which comes from that same toolchain — to
// self-install on Node 25+), but nothing declared that dependency, so
// corepack could run before mise's toolchain-converge step and fail
// with "npm not found."
func TestCorepackDependsOnMiseToolchains(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Corepack(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	deps := got[0].DependsOn
	if len(deps) != 1 || deps[0] != "mise_toolchains" {
		t.Fatalf("expected corepack to DependsOn mise_toolchains, got %v", deps)
	}
}
