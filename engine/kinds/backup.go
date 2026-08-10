package kinds

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// DefaultBackupDir returns ~/.devboost/backups, matching the bash tool's
// default (DB_BACKUP_DIR). No env-var override yet — the bash version's
// --config-adjacent DB_BACKUP_DIR override isn't wired through here since
// nothing in the Go CLI surface sets it yet; add one if/when that's needed.
// Exported so callers outside this package (migrate-from-oh-my-zsh) can
// archive things into the same backup root without duplicating the path.
func DefaultBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devboost", "backups"), nil
}

// BackupFile copies path into a fresh timestamped subdirectory of the
// backup dir before it's about to be overwritten, mirroring the bash
// tool's db_backup_file. A no-op if path doesn't exist yet (nothing to
// back up). Exported so module-local resource kinds outside this package
// (e.g. zsh's include-block handling, which has its own custom diff logic
// not shaped like any generic kind) can reuse the same backup behavior
// instead of duplicating it.
func BackupFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	dir, err := DefaultBackupDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(dir, time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(filepath.Join(dest, filepath.Base(path)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("backup %s: %w", path, err)
	}
	return nil
}

// ArchiveDir moves dir aside into the backup root instead of deleting
// it, named <basename>-<label>-<timestamp> — ports the bash tool's
// _db_legacy_archive_dir naming exactly (no extra nesting), reused here
// by migrate-from-oh-my-zsh for ~/.oh-my-zsh. A no-op if dir doesn't
// exist (already archived — idempotent).
func ArchiveDir(dir, label string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	backupRoot, err := DefaultBackupDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(backupRoot, fmt.Sprintf("%s-%s-%s", filepath.Base(dir), label, time.Now().Format("20060102_150405")))
	return os.Rename(dir, dest)
}
