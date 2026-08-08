package modules

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

const (
	zshIncludeStart = "# >>> devboost include start"
	zshIncludeEnd   = "# <<< devboost include end"
)

// zshUnmarkedSourceRe matches a live (non-comment) line sourcing
// .zshrc.devboost — ports the bash version's
// grep -Eq '(^|[^#].*)\.zshrc\.devboost'.
var zshUnmarkedSourceRe = regexp.MustCompile(`(^|[^#].*)\.zshrc\.devboost`)

// zshIncludeBlock is module-local, not a general resource kind: its
// behavior (detect a pre-existing unmarked line sourcing the same file
// and refuse to inject rather than double-source it) is specific to this
// one situation, not a shape any other module needs. Still fully typed
// and diff-based like any resource kind — "module-local" is about where
// it's registered, not about being any less rigorous than a promoted
// kind.
type zshIncludeBlock struct {
	zshrcPath string
}

func (z zshIncludeBlock) Diff() (*engine.PendingOp, error) {
	block := zshIncludeStart + "\n" +
		`[ -f "$HOME/.zshrc.devboost" ] && source "$HOME/.zshrc.devboost"` + "\n" +
		zshIncludeEnd + "\n"

	data, err := os.ReadFile(z.zshrcPath)
	if os.IsNotExist(err) {
		return &engine.PendingOp{
			Description: fmt.Sprintf("create %s with devboost include block", z.zshrcPath),
			Execute: func() error {
				return os.WriteFile(z.zshrcPath, []byte(block), 0o644)
			},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", z.zshrcPath, err)
	}
	content := string(data)

	if strings.Contains(content, zshIncludeStart) {
		return nil, nil // already present, nothing to do
	}

	if zshUnmarkedSourceRe.MatchString(content) {
		// An unmarked line already sources .zshrc.devboost — likely left
		// over from a prior manual edit or migrate-from-oh-my-zsh
		// recovery. Appending our own marked block on top would source
		// it twice on every shell start. This resource has nothing safe
		// to converge to here — surface it as a warning-shaped pending op
		// with no Execute, so plan/apply can show it without silently
		// double-sourcing (mirrors the bash version's db_log_warn + skip).
		return &engine.PendingOp{
			Description: fmt.Sprintf(
				"WARNING: %s already has an unmarked line sourcing .zshrc.devboost — "+
					"skipping include-block injection to avoid double-sourcing it. "+
					"Remove that line and re-run apply.", z.zshrcPath),
			Execute: func() error { return nil }, // acknowledge, do nothing
		}, nil
	}

	return &engine.PendingOp{
		Description: fmt.Sprintf("append devboost include block to %s", z.zshrcPath),
		Execute: func() error {
			if err := kinds.BackupFile(z.zshrcPath); err != nil {
				return err
			}
			sep := "\n"
			if strings.HasSuffix(content, "\n") {
				sep = ""
			}
			return os.WriteFile(z.zshrcPath, []byte(content+sep+"\n"+block), 0o644)
		},
	}, nil
}
