package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// asdfSourcePattern matches the line sourcing asdf's shell integration —
// ports DB_LEGACY_ASDF_SOURCE_PATTERN exactly.
const asdfSourcePattern = `(^|[[:space:]])\. .*/asdf\.sh([[:space:]]|$)`

// AsdfMiseDedup ports the asdf-mise-dup half of
// modules/module_legacy_shell.sh: asdf sourced alongside devboost's own
// mise, disabled in place, never deleted.
func AsdfMiseDedup(cfg *config.Config) []engine.Resource {
	if cfg.Get("legacy_shell.enable", "true") != "true" {
		return nil
	}
	zshrc := legacyShellZshrc(cfg)
	if !fileHasMatch(zshrc, asdfSourcePattern) {
		return nil
	}
	return []engine.Resource{
		{
			ID: "asdf_mise_dedup",
			Kind: kinds.LineInFile{
				Path:        zshrc,
				Pattern:     asdfSourcePattern,
				MigrationID: "asdf-mise-dup",
			},
		},
	}
}
