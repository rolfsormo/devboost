package kinds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rolfsormo/devboost/engine"
)

// File declares "this file should exist with this exact content." Its
// diff is byte-for-byte content comparison — any difference (including a
// missing file) is pending. Mode defaults to 0644 if unset. Backup
// defaults to true (matching the bash tool's db_write_file, which always
// backs up before overwriting).
type File struct {
	Path     string
	Content  string
	Mode     os.FileMode
	NoBackup bool // set true to skip the pre-write backup
}

func (f File) mode() os.FileMode {
	if f.Mode == 0 {
		return 0o644
	}
	return f.Mode
}

func (f File) Diff() (*engine.PendingOp, error) {
	current, err := os.ReadFile(f.Path)
	if err == nil && string(current) == f.Content {
		return nil, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", f.Path, err)
	}

	verb := "write"
	if err == nil {
		verb = "update"
	}
	return &engine.PendingOp{
		Description: fmt.Sprintf("%s %s", verb, f.Path),
		Execute: func() error {
			if !f.NoBackup {
				if err := BackupFile(f.Path); err != nil {
					return err
				}
			}
			if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
				return err
			}
			return os.WriteFile(f.Path, []byte(f.Content), f.mode())
		},
	}, nil
}
