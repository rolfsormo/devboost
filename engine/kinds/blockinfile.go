package kinds

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rolfsormo/devboost/engine"
)

// BlockInFile declares "this marked block should be present in this file
// with this content," porting the bash tool's db_upsert_block. Three
// cases: the file doesn't exist yet (create it with just the block); the
// start marker is already present (replace everything between the
// markers); neither (append the block to the end of the file).
type BlockInFile struct {
	Path        string
	StartMarker string
	EndMarker   string
	Content     string
}

func (b BlockInFile) block() string {
	return b.StartMarker + "\n" + b.Content + "\n" + b.EndMarker + "\n"
}

func (b BlockInFile) Diff() (*engine.PendingOp, error) {
	data, err := os.ReadFile(b.Path)
	if os.IsNotExist(err) {
		return &engine.PendingOp{
			Description: fmt.Sprintf("create %s with managed block", b.Path),
			Execute: func() error {
				if err := os.MkdirAll(filepath.Dir(b.Path), 0o755); err != nil {
					return err
				}
				return os.WriteFile(b.Path, []byte(b.block()), 0o644)
			},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", b.Path, err)
	}

	current := string(data)
	desired := replaceOrAppendBlock(current, b.StartMarker, b.EndMarker, b.block())
	if desired == current {
		return nil, nil
	}

	verb := "update block in"
	if !strings.Contains(current, b.StartMarker) {
		verb = "append block to"
	}
	return &engine.PendingOp{
		Description: fmt.Sprintf("%s %s", verb, b.Path),
		Execute: func() error {
			if err := BackupFile(b.Path); err != nil {
				return err
			}
			return os.WriteFile(b.Path, []byte(desired), 0o644)
		},
	}, nil
}

// RemoveBlock strips a marked block (start/end markers inclusive) from
// path, ports the bash tool's db_remove_block. A no-op if the file
// doesn't exist or the start marker isn't present. Used by uninstall,
// not by BlockInFile.Diff/Execute (which only ever adds/updates a
// block) — removal is a deliberate, separate action, same split as the
// bash tool's db_upsert_block vs. db_remove_block.
func RemoveBlock(path, startMarker, endMarker string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	current := string(data)
	if !strings.Contains(current, startMarker) {
		return nil
	}
	if err := BackupFile(path); err != nil {
		return err
	}
	desired := replaceOrAppendBlock(current, startMarker, endMarker, "")
	return os.WriteFile(path, []byte(desired), 0o644)
}

// replaceOrAppendBlock ports db_upsert_block's awk logic line-by-line: if
// startMarker is found, everything from that line through the endMarker
// line (inclusive) is replaced with block; otherwise block is appended
// (preceded by a blank line, matching the bash version).
func replaceOrAppendBlock(content, startMarker, endMarker, block string) string {
	if !strings.Contains(content, startMarker) {
		if content == "" {
			return block
		}
		sep := "\n"
		if strings.HasSuffix(content, "\n") {
			sep = ""
		}
		return content + sep + "\n" + block
	}

	var out strings.Builder
	inBlock := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !inBlock && strings.Contains(line, startMarker) {
			inBlock = true
			out.WriteString(block)
			continue
		}
		if inBlock && strings.Contains(line, endMarker) {
			inBlock = false
			continue
		}
		if !inBlock {
			out.WriteString(line)
			out.WriteString("\n")
		}
	}
	return out.String()
}
