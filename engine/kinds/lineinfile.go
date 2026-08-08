package kinds

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rolfsormo/devboost/engine"
)

// markerPrefix and markerFor mirror the bash tool's
// _DB_LEGACY_MARKER_PREFIX/_db_legacy_marker_for exactly, so a file marked
// by the bash version and later processed by this Go version (or vice
// versa, during the migration period) is read identically by both.
const markerPrefix = "# devboost:disabled:"

func markerFor(migrationID string) string {
	return markerPrefix + migrationID + " "
}

// LineInFile declares "lines matching Pattern in this file should be
// disabled (commented out with a devboost:disabled marker), unless the
// user has manually restored one by hand." This is the mechanism behind
// devboost's redundant-tooling dedup modules (zinit/znap, asdf/mise,
// nvm/mise) — never deletes a line, always leaves it reviewable and
// reversible, and respects a user peeling the marker off by hand as an
// explicit override not to be undone by a later run.
//
// Unlike the bash version (core_legacy_shell.sh), there is no grep/awk
// dialect-mismatch risk here: Go's regexp package is the only engine used
// for both detecting matches and rewriting them, so the bash version's
// explicit dialect-mismatch detection/failure path has no equivalent
// failure mode to guard against.
type LineInFile struct {
	Path        string
	Pattern     string // an unanchored regexp, matched against each line
	MigrationID string
}

func (l LineInFile) re() (*regexp.Regexp, error) {
	return regexp.Compile(l.Pattern)
}

// wasEverMarked reports whether the most recent post-<MigrationID>
// snapshot of Path contains this pattern's marker — i.e. whether a line
// was ever actually disabled here before, which combined with an
// unmarked live match means the user peeled the marker off by hand.
func (l LineInFile) wasEverMarked() (bool, error) {
	dir, err := backupDir()
	if err != nil {
		return false, err
	}
	matches, err := latestSnapshotsMatching(dir, filepath.Base(l.Path)+".post-"+l.MigrationID+"-")
	if err != nil || matches == "" {
		return false, err
	}
	data, err := os.ReadFile(matches)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), markerFor(l.MigrationID)), nil
}

func (l LineInFile) Diff() (*engine.PendingOp, error) {
	re, err := l.re()
	if err != nil {
		return nil, fmt.Errorf("compile pattern %q: %w", l.Pattern, err)
	}

	data, err := os.ReadFile(l.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", l.Path, err)
	}

	marker := markerFor(l.MigrationID)
	lines := splitLines(string(data))
	var toDisable []int
	for i, line := range lines {
		if re.MatchString(line) && !strings.HasPrefix(line, marker) {
			toDisable = append(toDisable, i)
		}
	}
	if len(toDisable) == 0 {
		return nil, nil
	}

	restored, err := l.wasEverMarked()
	if err != nil {
		return nil, err
	}
	if restored {
		// A previously-marked line is now live and unmarked: the user
		// restored it by hand. Respect that — don't re-disable.
		return nil, nil
	}

	return &engine.PendingOp{
		Description: fmt.Sprintf("disable %d redundant line(s) (%s) in %s", len(toDisable), l.MigrationID, l.Path),
		Execute: func() error {
			if err := snapshot(l.Path, l.MigrationID, "pre"); err != nil {
				return err
			}
			disabled := make(map[int]bool, len(toDisable))
			for _, i := range toDisable {
				disabled[i] = true
			}
			out := make([]string, len(lines))
			for i, line := range lines {
				if disabled[i] {
					out[i] = marker + line
				} else {
					out[i] = line
				}
			}
			if err := os.WriteFile(l.Path, []byte(strings.Join(out, "\n")+trailingNewline(string(data))), 0o644); err != nil {
				return err
			}
			return snapshot(l.Path, l.MigrationID, "post")
		},
	}, nil
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func trailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return "\n"
	}
	return ""
}

// latestSnapshotsMatching returns the newest file in dir whose name has
// the given prefix, or "" if none exist.
func latestSnapshotsMatching(dir, prefix string) (string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var latest string
	var latestMod int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if mod := info.ModTime().UnixNano(); mod > latestMod {
			latestMod = mod
			latest = filepath.Join(dir, e.Name())
		}
	}
	return latest, nil
}

// snapshot mirrors _db_legacy_snapshot: a hand-named backup copy with a
// caller-chosen phase suffix, distinct from the plain backupFile helper
// (which db_backup_file's fixed one-arg signature required in bash but
// isn't a constraint here — kept as a separate function since it names
// its backups differently, matching the bash source it ports).
func snapshot(path, migrationID, phase string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	dir, err := backupDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("%s.%s-%s-%s", filepath.Base(path), phase, migrationID, time.Now().Format("20060102_150405"))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}
