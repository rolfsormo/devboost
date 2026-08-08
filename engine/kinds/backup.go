package kinds

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// backupDir returns ~/.devboost/backups, matching the bash tool's default
// (DB_BACKUP_DIR). No env-var override yet — the bash version's
// --config-adjacent DB_BACKUP_DIR override isn't wired through here since
// nothing in the Go CLI surface sets it yet; add one if/when that's needed.
func backupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devboost", "backups"), nil
}

// backupFile copies path into a fresh timestamped subdirectory of the
// backup dir before it's about to be overwritten, mirroring the bash
// tool's db_backup_file. A no-op if path doesn't exist yet (nothing to
// back up).
func backupFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	dir, err := backupDir()
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
