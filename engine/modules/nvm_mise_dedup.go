package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// nvmSourcePattern matches lines sourcing nvm's shell hook or its
// bash-completion shim — ports DB_LEGACY_NVM_SOURCE_PATTERN exactly,
// deliberately matching on the quoted path alone (not a leading
// backslash/source token) since that's what kept the original bash
// pattern grep/awk-dialect-safe; no such constraint applies in Go (RE2 is
// the only engine here), but there's no reason to diverge from a pattern
// that already works correctly.
const nvmSourcePattern = `"[^"]*/nvm(\.sh|/etc/bash_completion\.d/nvm)"`

// NvmMiseDedup ports the nvm-mise-dup half of
// modules/module_legacy_shell.sh: nvm's shell hook (measured at
// ~850-900ms per login shell — the dominant real-world contributor to
// the startup-lag investigation this whole optimization mechanism came
// from) sourced in ~/.zprofile alongside devboost's own mise, disabled
// in place. See zinit_znap_dedup.go for the shared rationale behind the
// optimization modules.
func NvmMiseDedup(cfg *config.Config) []engine.Resource {
	if cfg.Get("optimize.enable", "true") != "true" {
		return nil
	}
	zprofile := cfg.Get("optimize.zprofile", "~/.zprofile")
	if !fileHasMatch(zprofile, nvmSourcePattern) {
		return nil
	}
	return []engine.Resource{
		{
			ID: "nvm_mise_dedup",
			Kind: kinds.LineInFile{
				Path:        zprofile,
				Pattern:     nvmSourcePattern,
				MigrationID: "nvm-mise-dup",
			},
		},
	}
}
