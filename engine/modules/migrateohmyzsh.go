// Package modules also houses the oh-my-zsh migration, ported from
// core/core_omz.sh. It's registered as a normal CommandGuarded resource
// (see the init() below and OhMyZshMigration) — one of the four
// optimization modules grouped under the optimize config key, alongside
// zinit/asdf/nvm dedup (see zinit_znap_dedup.go for the shared
// rationale). It used to be its own special-cased CLI subcommand
// (migrate-from-oh-my-zsh) gated behind a bespoke --yes flag; that's
// gone now — apply converges this automatically, exactly like its three
// siblings, and devboost undo is the way back for all four. See
// engine.Undoer's doc comment and this file's UndoConverge registration
// for how.
package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// omzTemplateLinePrefixes and omzTemplateLineExact port
// _db_omz_is_template_line: lines matching these are oh-my-zsh's own
// template scaffolding, not user content, even when their values are
// user-customized (e.g. ZSH_THEME).
var omzTemplateLinePrefixes = []string{
	"#",
	"export ZSH=",
	"ZSH_THEME=",
	"ZSH_THEME_RANDOM_CANDIDATES=",
	"CASE_SENSITIVE=",
	"HYPHEN_INSENSITIVE=",
	"DISABLE_MAGIC_FUNCTIONS=",
	"DISABLE_LS_COLORS=",
	"DISABLE_AUTO_TITLE=",
	"ENABLE_CORRECTION=",
	"COMPLETION_WAITING_DOTS=",
	"DISABLE_UNTRACKED_FILES_DIRTY=",
	"HIST_STAMPS=",
	"ZSH_CUSTOM=",
	"plugins=(",
	"source $ZSH/oh-my-zsh.sh",
}

// splitLinesTrimmed splits on newlines, dropping a trailing empty
// "line" a file ending in \n would otherwise produce — same correctness
// concern kinds.splitLines already handles for LineInFile, duplicated
// here in plain form rather than exporting kinds' version across a
// package boundary for something this small.
func splitLinesTrimmed(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func isOmzTemplateLine(line string) bool {
	if line == "" {
		return true
	}
	for _, prefix := range omzTemplateLinePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	// zstyle '*:omz:*' lines — ports the bash case pattern
	// 'zstyle '*':omz:'*'.
	if strings.HasPrefix(line, "zstyle ") && strings.Contains(line, ":omz:") {
		return true
	}
	return false
}

// stripOmzTemplate ports _db_omz_strip_template.
func stripOmzTemplate(content string) string {
	lines := splitLinesTrimmed(content)
	var out []string
	for _, line := range lines {
		if !isOmzTemplateLine(line) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// omzArchiveLabel is ArchiveDir's label for ~/.oh-my-zsh, factored out
// so Undo can glob for the same prefix it was archived under.
const omzArchiveLabel = "omz-migrate"

// revertedSuffix marks a backup as already consumed by a prior undo —
// appended to the backup's own path when undo restores from it, so a
// second `devboost undo` doesn't find and re-restore the same one (the
// glob in both find functions below excludes anything already carrying
// this suffix). Kept on disk rather than deleted, same as every other
// backup — reviewable if someone needs to see what happened, and its
// new name says plainly that it's already been acted on.
const revertedSuffix = "-reverted"

// findLatestArchivedOmzDir finds the most recently archived, not yet
// reverted, ~/.oh-my-zsh (see ArchiveDir/omzArchiveLabel), or "" if none
// exists — the undo-side mirror of findUninstalledBackup below.
func findLatestArchivedOmzDir(home string) (string, error) {
	backupRoot, err := kinds.DefaultBackupDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(backupRoot)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	prefix := ".oh-my-zsh-" + omzArchiveLabel + "-"
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || strings.HasSuffix(e.Name(), revertedSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = filepath.Join(backupRoot, e.Name())
		}
	}
	return latest, nil
}

// findLatestZshrcBackup finds the most recent, not yet reverted,
// BackupFile snapshot of ~/.zshrc (a timestamped subdirectory of the
// backup root containing a .zshrc file — see kinds.BackupFile), or "" if
// none exists. This is the snapshot Converge takes immediately before
// appending recovered customizations, so it's what Undo restores from.
func findLatestZshrcBackup(home string) (string, error) {
	backupRoot, err := kinds.DefaultBackupDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(backupRoot)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if !e.IsDir() || strings.HasSuffix(e.Name(), revertedSuffix) {
			continue
		}
		candidate := filepath.Join(backupRoot, e.Name(), ".zshrc")
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = candidate
		}
	}
	return latest, nil
}

// findUninstalledBackup ports _db_omz_find_uninstalled_backup: the most
// recently modified ~/.zshrc.omz-uninstalled-* file, if any.
func findUninstalledBackup(home string) (string, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return "", err
	}
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), ".zshrc.omz-uninstalled-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = filepath.Join(home, e.Name())
		}
	}
	return latest, nil
}

// uninstallOhMyZsh ports _db_omz_uninstall: replicates oh-my-zsh's own
// tools/uninstall.sh. ~/.oh-my-zsh is moved aside (not deleted outright
// — this Go port is more conservative than the bash version, which does
// rm -rf; moving aside costs nothing and gives an extra recovery path if
// something in the migration goes wrong), the current .zshrc is renamed
// to a timestamped backup, and .zshrc.pre-oh-my-zsh is restored to
// .zshrc if it exists.
func uninstallOhMyZsh(home string) error {
	omzDir := filepath.Join(home, ".oh-my-zsh")
	zshrc := filepath.Join(home, ".zshrc")
	base := filepath.Join(home, ".zshrc.pre-oh-my-zsh")

	if _, err := os.Stat(omzDir); os.IsNotExist(err) {
		return nil
	}

	if err := kinds.ArchiveDir(omzDir, omzArchiveLabel); err != nil {
		return fmt.Errorf("archive %s: %w", omzDir, err)
	}
	fmt.Printf("Removed: %s\n", omzDir)

	if _, err := os.Stat(zshrc); err == nil {
		saved := filepath.Join(home, ".zshrc.omz-uninstalled-"+time.Now().Format("2006-01-02_15-04-05"))
		if err := os.Rename(zshrc, saved); err != nil {
			return fmt.Errorf("rename %s: %w", zshrc, err)
		}
		fmt.Printf("Renamed %s to: %s\n", zshrc, saved)
	}

	if _, err := os.Stat(base); err == nil {
		if err := os.Rename(base, zshrc); err != nil {
			return fmt.Errorf("restore %s: %w", base, err)
		}
		fmt.Printf("Restored pre-oh-my-zsh config to: %s\n", zshrc)
	} else {
		fmt.Println("No ~/.zshrc.pre-oh-my-zsh found — nothing to restore.")
	}
	return nil
}

// linesOnlyIn ports _db_omz_lines_only_in: lines present in other
// (stripped of oh-my-zsh template lines) but not present anywhere in
// base — order-preserving, duplicate-preserving.
func linesOnlyIn(other, base string) string {
	stripped := stripOmzTemplate(other)
	if base == "" {
		return stripped
	}
	baseLines := make(map[string]bool)
	for _, l := range splitLinesTrimmed(base) {
		baseLines[l] = true
	}
	var out []string
	for _, l := range splitLinesTrimmed(stripped) {
		if !baseLines[l] {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// migrateFromOhMyZsh ports core_omz.sh's db_run_migrate_from_oh_my_zsh —
// the Converge half of the omz_migration resource (see the init() below).
// No dryRun/yes parameters: this only runs from Execute, which the
// engine only calls when actually converging (never from
// ComputeDiff/Plan), and there's no confirmation gate anymore — apply
// converges this the same unconditional way it converges the other
// three optimization modules. Reversibility (not a pre-execution gate)
// is what makes that safe: every step below is a rename or a
// backup-then-overwrite, never a delete, so Undo (below) can put
// everything back.
func migrateFromOhMyZsh() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	zshrc := filepath.Join(home, ".zshrc")
	base := filepath.Join(home, ".zshrc.pre-oh-my-zsh")

	var baseSnapshot string
	if data, err := os.ReadFile(base); err == nil {
		baseSnapshot = string(data)
	}

	if err := uninstallOhMyZsh(home); err != nil {
		return err
	}

	uninstalled, err := findUninstalledBackup(home)
	if err != nil {
		return err
	}
	if uninstalled == "" {
		return fmt.Errorf("no ~/.zshrc.omz-uninstalled-* backup found — nothing to recover")
	}
	fmt.Printf("Found uninstall backup: %s\n", uninstalled)

	uninstalledData, err := os.ReadFile(uninstalled)
	if err != nil {
		return err
	}
	additions := linesOnlyIn(string(uninstalledData), baseSnapshot)

	if strings.TrimSpace(additions) == "" {
		fmt.Println("No post-install customizations found beyond oh-my-zsh's own template — nothing to recover.")
		fmt.Printf("Review %s, then run 'devboost apply' to add devboost's include block.\n", zshrc)
		return nil
	}

	if err := kinds.BackupFile(zshrc); err != nil {
		return err
	}
	var newContent strings.Builder
	if data, err := os.ReadFile(zshrc); err == nil {
		newContent.Write(data)
		newContent.WriteString("\n")
	}
	newContent.WriteString(additions)
	newContent.WriteString("\n")
	if err := os.WriteFile(zshrc, []byte(newContent.String()), 0o644); err != nil {
		return err
	}

	fmt.Printf("Recovered your customizations into: %s\n", zshrc)
	fmt.Println("Review the result, then run 'devboost apply' to add devboost's own setup.")
	return nil
}

// findOmzUndoMaterial locates whatever a prior migration left behind to
// undo from: the archived ~/.oh-my-zsh (see findLatestArchivedOmzDir)
// and/or the .zshrc backup taken right before the recovered-additions
// rewrite (see findLatestZshrcBackup). Either, both, or neither may
// exist depending on exactly how far a prior Converge got — shared
// between UndoSatisfied (which only needs to know whether anything
// exists) and undoOhMyZshMigration (which needs the actual paths).
func findOmzUndoMaterial(home string) (archived, backup string, err error) {
	archived, err = findLatestArchivedOmzDir(home)
	if err != nil {
		return "", "", err
	}
	backup, err = findLatestZshrcBackup(home)
	if err != nil {
		return "", "", err
	}
	return archived, backup, nil
}

// undoOhMyZshMigration reverses migrateFromOhMyZsh's physically
// reversible steps: move the archived ~/.oh-my-zsh back into place, and
// restore ~/.zshrc from the backup snapshot taken right before the
// recovered-additions rewrite. The diff-and-append recovery step itself
// isn't byte-for-byte invertible in isolation, but restoring .zshrc from
// its pre-rewrite backup achieves the same end state, since that backup
// was taken immediately before (and only before) that rewrite happened.
// Only called when UndoSatisfied (below) has already confirmed there's
// something to restore.
func undoOhMyZshMigration() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	zshrc := filepath.Join(home, ".zshrc")

	archived, backup, err := findOmzUndoMaterial(home)
	if err != nil {
		return err
	}

	if archived != "" {
		if err := os.Rename(archived, filepath.Join(home, ".oh-my-zsh")); err != nil {
			return fmt.Errorf("restore %s: %w", archived, err)
		}
		fmt.Printf("Restored: %s\n", filepath.Join(home, ".oh-my-zsh"))
	}

	if backup != "" {
		data, err := os.ReadFile(backup)
		if err != nil {
			return err
		}
		if err := os.WriteFile(zshrc, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("Restored %s from: %s\n", zshrc, backup)

		// Mark the whole backup snapshot (its timestamped parent
		// directory, not just the .zshrc file inside it) as consumed —
		// os.Rename, not deleted, so it stays reviewable on disk but a
		// later findLatestZshrcBackup (a second `devboost undo`, or a
		// future migration's own lookup) won't find and re-restore it.
		backupDir := filepath.Dir(backup)
		if err := os.Rename(backupDir, backupDir+revertedSuffix); err != nil {
			return fmt.Errorf("mark %s reverted: %w", backupDir, err)
		}
	}

	return nil
}

func init() {
	kinds.RegisterCommand("omz_migration_converged", kinds.GuardedCommand{
		Satisfied: func(any) (bool, error) {
			home, err := os.UserHomeDir()
			if err != nil {
				return false, err
			}
			if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh")); os.IsNotExist(err) {
				return true, nil
			} else if err != nil {
				return false, err
			}
			return false, nil
		},
		Converge: func(any) error { return migrateFromOhMyZsh() },
		UndoSatisfied: func(any) (bool, error) {
			home, err := os.UserHomeDir()
			if err != nil {
				return false, err
			}
			archived, backup, err := findOmzUndoMaterial(home)
			if err != nil {
				return false, err
			}
			return archived == "" && backup == "", nil
		},
		UndoConverge: func(any) error { return undoOhMyZshMigration() },
	})
}

// OhMyZshMigration ports core_omz.sh: detects a pre-existing oh-my-zsh
// installation and migrates away from it — removing ~/.oh-my-zsh
// (archived, not deleted) and recovering any customizations added to
// .zshrc after oh-my-zsh's own template — gated on optimize.enable, same
// as its three sibling optimization modules (see zinit_znap_dedup.go for
// the shared rationale: oh-my-zsh alongside devboost's own znap plugin
// manager, starship prompt, and curated plugin set is redundant and can
// slow shell startup or cause conflicting keybindings/completions).
func OhMyZshMigration(cfg *config.Config) []engine.Resource {
	if cfg.Get("optimize.enable", "true") != "true" {
		return nil
	}
	return []engine.Resource{
		{
			ID:   "omz_migration",
			Kind: kinds.CommandGuarded{ID: "omz_migration_converged", Wants: "migrate away from oh-my-zsh"},
		},
	}
}
