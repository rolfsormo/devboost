package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// TestAllResourcesResolveWithDefaultConfig is the real integration check
// for the whole registry: every module's DependsOn references must
// resolve when all modules run together (the actual apply/plan
// scenario), not just in isolated pairwise tests. This is exactly the
// kind of cross-module coupling (zsh/security's file-write race) that
// only surfaces when the full set runs together — catches a bad
// DependsOn ID, a missing module in the registry, or a real dependency
// cycle before it reaches a real machine.
func TestAllResourcesResolveWithDefaultConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	resources := AllResources(cfg, kinds.OSDarwin)
	if len(resources) == 0 {
		t.Fatal("expected at least some resources with default config")
	}

	// ComputeDiff runs topoSort internally — an unresolvable DependsOn or
	// a cycle surfaces here as an error, not a panic or silent wrong order.
	if _, err := engine.ComputeDiff(resources); err != nil {
		t.Fatalf("registry produced resources that don't resolve: %v", err)
	}
}

func TestAllResourcesNoDuplicateIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	resources := AllResources(cfg, kinds.OSDarwin)
	seen := make(map[string]bool, len(resources))
	for _, r := range resources {
		if seen[r.ID] {
			t.Fatalf("duplicate resource ID %q across modules — topoSort would reject this at runtime", r.ID)
		}
		seen[r.ID] = true
	}
}
