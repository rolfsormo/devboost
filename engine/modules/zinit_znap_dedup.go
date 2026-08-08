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
// see LineInFile). This is one of the three per-tool dedup modules split
// out of the original single legacy_shell module, per the architecture
// doc's tool-first grouping decision.
func ZinitZnapDedup(cfg *config.Config) []engine.Resource {
	if cfg.Get("legacy_shell.enable", "true") != "true" {
		return nil
	}
	zshrc := legacyShellZshrc(cfg)
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

// legacyShellZshrc ports _db_legacy_shell_zshrc's default — hardcoded to
// ~/.zshrc in the bash version, never actually config-driven despite the
// db_yaml_get call (no module ever set .legacy_shell.zshrc).
func legacyShellZshrc(cfg *config.Config) string {
	return cfg.Get("legacy_shell.zshrc", "~/.zshrc")
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
