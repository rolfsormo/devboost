// Package modules holds the Go-engine ports of devboost's bash modules.
// This package coexists with the original bash modules/*.sh tree during
// the v2 migration; it does not replace or modify any bash file.
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
