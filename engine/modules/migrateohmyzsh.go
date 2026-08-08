// Package modules also houses migrate-from-oh-my-zsh, ported from
// core/core_omz.sh. This is deliberately NOT a Resource/ResourceKind —
// it's a one-shot migration operation (remove ~/.oh-my-zsh, recover
// customizations), not a "converge to desired state" module, same
// distinction the bash version drew by making it its own explicit
// subcommand rather than folding it into apply.
package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
func uninstallOhMyZsh(home string, dryRun bool) error {
	omzDir := filepath.Join(home, ".oh-my-zsh")
	zshrc := filepath.Join(home, ".zshrc")
	base := filepath.Join(home, ".zshrc.pre-oh-my-zsh")

	if _, err := os.Stat(omzDir); os.IsNotExist(err) {
		return nil
	}

	if dryRun {
		fmt.Printf("Would remove: %s\n", omzDir)
		if _, err := os.Stat(zshrc); err == nil {
			fmt.Printf("Would rename %s to a timestamped .omz-uninstalled-* backup\n", zshrc)
		}
		if _, err := os.Stat(base); err == nil {
			fmt.Printf("Would restore %s to %s\n", base, zshrc)
		}
		return nil
	}

	if err := kinds.ArchiveDir(omzDir, "omz-migrate"); err != nil {
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

// MigrateFromOhMyZsh ports core_omz.sh's db_run_migrate_from_oh_my_zsh.
// yes must be true to actually run (matching the bash tool's --yes
// requirement for this destructive-ish command) unless dryRun is set.
func MigrateFromOhMyZsh(dryRun, yes bool) error {
	if !dryRun && !yes {
		return fmt.Errorf("this removes ~/.oh-my-zsh and rewrites ~/.zshrc — pass --yes to confirm (preview first with --dry-run)")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	zshrc := filepath.Join(home, ".zshrc")
	base := filepath.Join(home, ".zshrc.pre-oh-my-zsh")

	var baseSnapshot string
	if !dryRun {
		if data, err := os.ReadFile(base); err == nil {
			baseSnapshot = string(data)
		}
	}

	if err := uninstallOhMyZsh(home, dryRun); err != nil {
		return err
	}

	uninstalled, err := findUninstalledBackup(home)
	if err != nil {
		return err
	}
	if uninstalled == "" {
		if dryRun {
			fmt.Println("No existing ~/.zshrc.omz-uninstalled-* backup yet — nothing further to recover in a dry-run.")
			return nil
		}
		return fmt.Errorf("no ~/.zshrc.omz-uninstalled-* backup found — nothing to recover")
	}
	fmt.Printf("Found uninstall backup: %s\n", uninstalled)

	if dryRun {
		if _, err := os.Stat(base); err == nil {
			fmt.Printf("Would recover your additions from '%s' not already in '%s'\n", filepath.Base(uninstalled), filepath.Base(base))
		} else {
			fmt.Printf("Would recover your additions from '%s' (no pre-install base to compare against)\n", filepath.Base(uninstalled))
		}
		fmt.Printf("Would append them to: %s\n", zshrc)
		return nil
	}

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
	fmt.Println("Review the result, then run 'devboost apply' to add devboost's include block.")
	return nil
}
