package modules

import (
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestTmuxDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "tmux:\n  enable: false\n")
	if got := Tmux(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestTmuxEnabledByDefaultProducesThreeResources(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Tmux(cfg)
	// tpm clone, config block, plugin install (auto_install_plugins defaults true)
	if len(got) != 3 {
		t.Fatalf("expected 3 resources, got %d: %v", len(got), got)
	}
}

func TestTmuxOmitsPluginInstallWhenAutoInstallDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "system:\n  auto_install_plugins: false\n")
	got := Tmux(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 resources (no plugin install step), got %d: %v", len(got), got)
	}
	for _, r := range got {
		if r.ID == "tmux_plugins" {
			t.Fatal("expected no tmux_plugins resource when auto_install_plugins is false")
		}
	}
}

func TestTmuxConfigBlockDependsOnTPM(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Tmux(cfg)
	for _, r := range got {
		if r.ID == "tmux_config_block" {
			if len(r.DependsOn) != 1 || r.DependsOn[0] != "tmux_tpm" {
				t.Fatalf("expected tmux_config_block to depend on tmux_tpm, got %v", r.DependsOn)
			}
			return
		}
	}
	t.Fatal("expected a tmux_config_block resource")
}

func TestRenderTmuxBlockUsesConfiguredSettings(t *testing.T) {
	cfg := loadFixtureConfig(t, "tmux:\n  settings:\n    base_index: 0\n    mouse: false\n")
	block := renderTmuxBlock(cfg, "/tpm/path")
	if !strings.Contains(block, "base-index 0") {
		t.Fatalf("expected configured base_index in block, got %q", block)
	}
	if !strings.Contains(block, "mouse off") {
		t.Fatalf("expected mouse off in block, got %q", block)
	}
	if !strings.Contains(block, "run '/tpm/path/tpm'") {
		t.Fatalf("expected tpm path wired into run line, got %q", block)
	}
}

func TestRenderTmuxBlockDefaults(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	block := renderTmuxBlock(cfg, "/tpm/path")
	if !strings.Contains(block, "base-index 1") || !strings.Contains(block, "mouse on") {
		t.Fatalf("expected default settings, got %q", block)
	}
}

func TestTmuxPluginsResourceCarriesTPMPathAsParams(t *testing.T) {
	cfg := loadFixtureConfig(t, "tmux:\n  tpm_path: /custom/tpm\n")
	got := Tmux(cfg)
	for _, r := range got {
		if r.ID == "tmux_plugins" {
			c, ok := r.Kind.(kinds.CommandGuarded)
			if !ok {
				t.Fatalf("expected kinds.CommandGuarded, got %T", r.Kind)
			}
			if c.Params != "/custom/tpm" {
				t.Fatalf("got Params %v, want /custom/tpm", c.Params)
			}
			return
		}
	}
	t.Fatal("expected a tmux_plugins resource")
}
