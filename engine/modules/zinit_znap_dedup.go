package modules

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// zinitDupPattern matches zinit lines loading plugins znap's default
// config already loads — ports module_legacy_shell.sh's
// DB_LEGACY_ZINIT_DUP_PATTERN exactly.
const zinitDupPattern = `^[[:space:]]*zinit (light|load)[^#]*(zsh-users/zsh-autosuggestions|zdharma-continuum/fast-syntax-highlighting|zsh-users/zsh-syntax-highlighting)`

// ZinitZnapDedup ports the zinit-znap-dup half of modules/module_legacy_shell.sh:
// detects a pre-existing zinit setup loading plugins znap already
// provides, and disables the redundant lines in place (never deletes —
// see LineInFile). This is one of the four optimization modules grouped
// under the optimize config key (see omz_migration.go for the fourth),
// split out per the architecture doc's tool-first grouping decision.
//
// Why these optimization modules exist at all: running a tool alongside
// its devboost-managed replacement isn't just wasted disk — each one
// does real work on every login shell (sourcing shell hooks, scanning
// version files, re-registering completions), duplicating what the
// replacement already does. This investigation measured nvm's shell
// hook specifically at ~850-900ms per login shell on a real machine via
// zprof — the single largest contributor to shell startup lag found,
// and reason enough on its own to disable it by default (see
// nvm_mise_dedup.go). Because a broken shell is worse than a slow one,
// the three line-level modules (this one, asdf, nvm) never delete a
// line: they comment it out in place via LineInFile, leaving a clear,
// reversible trace, and they're drift-aware — if a user manually
// restores the line, the next run detects that and leaves it alone
// rather than fighting the user's explicit choice. See also: issue #8,
// on eventually treating this as a first-class config-porting migration
// rather than a silent comment-out, so a user's zinit-side
// customizations aren't just dropped when devboost picks a different
// tool.
func ZinitZnapDedup(cfg *config.Config) []engine.Resource {
	if cfg.Get("optimize.enable", "true") != "true" {
		return nil
	}
	zshrc := optimizeZshrc(cfg)
	if !fileHasMatch(zshrc, zinitDupPattern) {
		return nil
	}
	return []engine.Resource{
		{
			ID: "zinit_znap_dedup",
			Kind: kinds.LineInFile{
				Path:        zshrc,
				Pattern:     zinitDupPattern,
				MigrationID: "zinit-znap-dup",
			},
		},
	}
}

// optimizeZshrc ports _db_legacy_shell_zshrc's default — hardcoded to
// ~/.zshrc in the bash version, never actually config-driven despite the
// db_yaml_get call (no module ever set .legacy_shell.zshrc, now
// optimize.zshrc).
func optimizeZshrc(cfg *config.Config) string {
	return cfg.Get("optimize.zshrc", "~/.zshrc")
}

// fileHasMatch is a cheap presence pre-check so a module can decide
// whether to declare a LineInFile resource at all — mirrors the bash
// module's own _db_legacy_shell_*_present gating before calling
// _db_legacy_disable_lines. Not strictly required (LineInFile.Diff
// already returns nil if nothing matches), but keeps doctor/plan from
// declaring a resource whose kind will just immediately no-op, which
// matters once doctor groups output by module (task #15) — a module
// with zero declared resources reads as "nothing to check here," not
// "checked and found nothing," a real distinction once there's tool-first
// grouping to report against.
func fileHasMatch(path, pattern string) bool {
	data, err := os.ReadFile(expandHomeForLegacyCheck(path))
	if err != nil {
		return false
	}
	return regexp.MustCompile(pattern).Match(data)
}

func expandHomeForLegacyCheck(path string) string {
	// LineInFile/config.Get already expand ~ before this is called in
	// practice (legacyShellZshrc goes through cfg.Get), but guard anyway
	// since fileHasMatch could be called with a raw path in tests.
	if len(path) >= 2 && path[0] == '~' && path[1] == '/' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
