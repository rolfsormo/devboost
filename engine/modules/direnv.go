package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

const direnvDefaultContent = `use_mise() { eval "$(mise activate direnv)"; }`

// Direnv ports modules/module_direnv.sh: write the devboost-managed
// .direnvrc, gated on direnv.enable. Content is configurable via
// direnv.content, defaulting to a use_mise helper.
func Direnv(cfg *config.Config) []engine.Resource {
	if cfg.Get("direnv.enable", "true") != "true" {
		return nil
	}
	path := cfg.Get("direnv.rc_path", "~/.direnvrc")
	content := cfg.Get("direnv.content", direnvDefaultContent)
	return []engine.Resource{
		{
			ID:   "direnvrc",
			Kind: kinds.File{Path: path, Content: content},
		},
	}
}
