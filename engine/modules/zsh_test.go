package modules

import "testing"

func TestZshDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "zsh:\n  enable: false\n")
	if got := Zsh(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestZshEnabledByDefaultProducesThreeResources(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Zsh(cfg)
	// zshrc.devboost, include block, atuin config (use_atuin defaults true)
	if len(got) != 3 {
		t.Fatalf("expected 3 resources, got %d: %v", len(got), got)
	}
}

func TestZshOmitsAtuinConfigWhenDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "zsh:\n  history:\n    use_atuin: false\n")
	got := Zsh(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 resources (no atuin config), got %d: %v", len(got), got)
	}
	for _, r := range got {
		if r.ID == "atuin_config" {
			t.Fatal("expected no atuin_config resource when use_atuin is false")
		}
	}
}

func TestZshIncludeBlockDependsOnDevboostFile(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Zsh(cfg)
	for _, r := range got {
		if r.ID == "zshrc_include_block" {
			if len(r.DependsOn) != 1 || r.DependsOn[0] != "zshrc_devboost" {
				t.Fatalf("expected dependency on zshrc_devboost, got %v", r.DependsOn)
			}
			return
		}
	}
	t.Fatal("expected a zshrc_include_block resource")
}
