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
// is cloned to the configured path. Config is read lazily via a
// constructor, not a package-level var, since reading ~/.devboost.yaml at
// import time would make every consumer of this package touch the
// filesystem just by importing it.
func Znap() []engine.Resource {
	return []engine.Resource{
		{
			ID: "znap_install",
			Kind: kinds.GitClone{
				URL:  config.Get("zsh.znap_git", "https://github.com/marlonrichert/zsh-snap.git"),
				Dest: config.Get("zsh.znap_path", "~/.zsh-snap"),
			},
		},
	}
}
