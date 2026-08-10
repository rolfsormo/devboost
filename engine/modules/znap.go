package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Znap ports modules/module_znap.sh: ensure znap (the zsh plugin manager)
// is cloned to the configured path. Every module constructor in this
// package takes the already-loaded *config.Config rather than reading
// ~/.devboost.yaml itself — the CLI entry point loads it once, and
// passing it explicitly keeps modules testable against an arbitrary
// fixture config instead of real files/env vars.
//
// Why znap over the other zsh plugin managers (zinit, antidote,
// sheldon, oh-my-zsh's own bundler): znap's pitch is minimal overhead —
// no framework, lazy-loads plugins, and its git-clone-based install
// (what this module does) has no separate binary/build step. This is
// the same lever this whole investigation pulled on for real: znap
// alongside zinit was one of the two concrete redundant-tooling issues
// found and fixed (see zinit_znap_dedup.go) — the choice of znap
// specifically isn't just taste, it's tied to the measured startup-lag
// work. That said, this is exactly the kind of pick worth an honest
// adversarial re-check (issue #9): issue #6 already proposes
// reconsidering znap in favor of zinit, and that discussion should
// settle on real current data, not the state of things when devboost
// first picked znap.
func Znap(cfg *config.Config) []engine.Resource {
	return []engine.Resource{
		{
			ID: "znap_install",
			Kind: kinds.GitClone{
				URL:  cfg.Get("zsh.znap_git", "https://github.com/marlonrichert/zsh-snap.git"),
				Dest: cfg.Get("zsh.znap_path", "~/.zsh-snap"),
			},
		},
	}
}
