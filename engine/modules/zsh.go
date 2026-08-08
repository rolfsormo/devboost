package modules

import (
	"os"
	"path/filepath"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Zsh ports modules/module_zsh.sh: renders .zshrc.devboost from config,
// injects the devboost include block into ~/.zshrc (skipping injection
// with a warning if an unmarked line already sources .zshrc.devboost —
// see zshIncludeBlock), and writes atuin's config.toml when atuin history
// is enabled. Gated overall on zsh.enable.
func Zsh(cfg *config.Config) []engine.Resource {
	if cfg.Get("zsh.enable", "true") != "true" {
		return nil
	}

	includeFile := cfg.Get("zsh.include_file", "~/.zshrc.devboost")
	// ~/.zshrc's path is hardcoded in the bash version too (never
	// config-driven, unlike include_file) — it's the one file a zsh
	// shell always sources by name, not something devboost lets a user
	// relocate.
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	zshrc := filepath.Join(home, ".zshrc")

	resources := []engine.Resource{
		{
			ID:   "zshrc_devboost",
			Kind: kinds.File{Path: includeFile, Content: renderZshDevboost(cfg)},
		},
		{
			ID:        "zshrc_include_block",
			Kind:      zshIncludeBlock{zshrcPath: zshrc},
			DependsOn: []string{"zshrc_devboost"},
		},
	}

	if cfg.Get("zsh.history.use_atuin", "true") == "true" {
		// Hardcoded in the bash version too — not a config.Get default
		// standing in for a real knob, atuin's config path was never
		// user-configurable there either.
		atuinConfig := filepath.Join(home, ".config", "atuin", "config.toml")
		resources = append(resources, engine.Resource{
			ID:   "atuin_config",
			Kind: kinds.File{Path: atuinConfig, Content: renderAtuinConfig(cfg)},
		})
	}

	return resources
}
