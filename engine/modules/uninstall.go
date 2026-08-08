package modules

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Uninstall ports core_main.sh's db_run_uninstall: removes
// devboost-managed files and the marked blocks it injected into files it
// doesn't own, leaving the user's own content in those files untouched.
// Deliberately does NOT remove packages, znap, TPM, or mise toolchains —
// same scope the bash version documented (uninstalling devboost's own
// config surface, not undoing everything it ever installed on the
// system).
func Uninstall(cfg *config.Config) error {
	includeFile := cfg.Get("zsh.include_file", "~/.zshrc.devboost")
	if err := removeFile(includeFile); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	zshrc := filepath.Join(home, ".zshrc")
	if err := kinds.RemoveBlock(zshrc, zshIncludeStart, zshIncludeEnd); err != nil {
		return fmt.Errorf("remove devboost block from %s: %w", zshrc, err)
	}

	tmuxConf := cfg.Get("tmux.conf_file", "~/.tmux.conf")
	if err := kinds.RemoveBlock(tmuxConf, tmuxStartMarker, tmuxEndMarker); err != nil {
		return fmt.Errorf("remove devboost block from %s: %w", tmuxConf, err)
	}

	direnvrc := cfg.Get("direnv.rc_path", "~/.direnvrc")
	if err := removeFile(direnvrc); err != nil {
		return err
	}

	stateFile := filepath.Join(home, ".devboost.state.json")
	if err := removeFile(stateFile); err != nil {
		return err
	}

	return nil
}

// removeFile deletes path if it exists, backing it up first — matches
// the bash version's db_backup_file-then-rm pattern for uninstall's
// direct file removals (as opposed to the block removals above, which
// RemoveBlock already backs up internally).
func removeFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := kinds.BackupFile(path); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}
