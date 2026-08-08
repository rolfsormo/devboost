package modules

import (
	"path/filepath"
	"testing"

	"github.com/rolfsormo/devboost/config"
)

func loadFixtureConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".devboost.yaml")
	if yaml != "" {
		writeFile(t, path, yaml)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestStarshipDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "prompt:\n  enable_starship: false\n")
	if got := Starship(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestStarshipEnabledByDefault(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Starship(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
}
