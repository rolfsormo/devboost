package modules

import (
	"fmt"
	"os"
	"strings"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// cleanManagedFiles lists every file an optimization migration might
// mark — ports core_legacy_shell.sh's _db_legacy_managed_files. Extend
// here if a future migration_id targets a different file.
func cleanManagedFiles(cfg *config.Config) []string {
	return []string{
		optimizeZshrc(cfg),
		cfg.Get("optimize.zprofile", "~/.zprofile"),
	}
}

// Clean ports core_legacy_shell.sh's db_run_clean: the opt-in, separate
// step that permanently deletes devboost:disabled-marked lines (see
// kinds.LineInFile / kinds.MarkerFor) rather than just leaving them
// commented out. Deliberately not part of apply — apply only ever
// disables redundant lines in place, reversibly; a human decides when
// they're confident enough to actually remove them, by running `devboost
// clean` explicitly. Idempotent and order-independent: it re-derives
// what to clean by reading the live file each run, not from any
// in-process state carried over from a prior apply. dryRun previews
// what would be removed without changing anything.
func Clean(cfg *config.Config, dryRun bool) error {
	for _, path := range cleanManagedFiles(cfg) {
		if err := cleanFile(path, dryRun); err != nil {
			return err
		}
	}
	if !dryRun {
		if dir, err := kinds.DefaultBackupDir(); err == nil {
			fmt.Printf("Archived directories remain under: %s (remove that folder to purge everything)\n", dir)
		}
	}
	return nil
}

// cleanFile strips every devboost:disabled-marked line (any migration
// ID) from path, restoring nothing — this is real deletion, unlike
// LineInFile's reversible comment-out. No-op if path doesn't exist or
// has no marked lines.
func cleanFile(path string, dryRun bool) error {
	resolved := expandHomeForLegacyCheck(path)
	data, err := os.ReadFile(resolved)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if !strings.Contains(string(data), kinds.MarkerPrefix) {
		return nil
	}

	trailing := ""
	if strings.HasSuffix(string(data), "\n") {
		trailing = "\n"
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	var kept []string
	removed := 0
	for _, line := range lines {
		if strings.HasPrefix(line, kinds.MarkerPrefix) {
			removed++
		} else {
			kept = append(kept, line)
		}
	}

	if dryRun {
		fmt.Printf("Would remove %d devboost-disabled line(s) from: %s\n", removed, path)
		return nil
	}

	if err := kinds.BackupFile(resolved); err != nil {
		return fmt.Errorf("backup %s: %w", path, err)
	}
	out := strings.Join(kept, "\n") + trailing
	if err := os.WriteFile(resolved, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("Removed devboost-disabled lines from: %s\n", path)
	return nil
}
