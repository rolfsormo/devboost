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
//
// Why direnv at all: it's the established, general-purpose tool for
// per-directory environment loading (15k+ GitHub stars, actively
// maintained) — broader in scope than asdf's own .tool-versions
// auto-activation, which only handles tool versions, not arbitrary env
// vars. No competing tool has comparable mindshare here.
//
// Why this exact use_mise wiring is a compromise, not a first-party
// recommendation: mise's own docs (mise.jdx.dev/direnv.html) state "the
// official stance is you should not use direnv with mise," that `use
// mise` is deprecated, and that direnv-compatibility bugs won't be
// fixed upstream. mise's own preferred integration today is `mise
// activate zsh` alone, replacing direnv's job for toolchain activation
// entirely. devboost still defaults to direnv+use_mise because it's a
// real, working, first-party-documented (if discouraged) path, and many
// users arrive already using direnv for non-toolchain env vars, so
// dropping it isn't free either. If mise's compatibility stance
// hardens further, the better long-term default is likely `mise
// activate` alone with this module optional/off by default. Challenge
// this default if you're revisiting it — see engine/modules/mise.go for
// the mise side of this integration.
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
