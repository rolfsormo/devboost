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

// zshUnmarkedSourceRe matches a line that actually sources
// .zshrc.devboost (a `source`/`.` shell keyword immediately preceding
// the reference, allowing the quotes/`$HOME`/whitespace real usage
// wraps it in) — not merely a line that mentions the filename, e.g. in
// a comment. Deliberately not anchored to line start: devboost's own
// generated block (`[ -f "$HOME/.zshrc.devboost" ] && source
// "$HOME/.zshrc.devboost"`) puts the real `source` after a condition
// guard, and a user's pre-existing line could do the same — anchoring
// to line start would miss that and reintroduce the more dangerous
// failure mode this mechanism exists to prevent (double-sourcing).
//
// This replaces an earlier, looser port of the bash version's
// grep -Eq '(^|[^#].*)\.zshrc\.devboost', which matched the bare
// substring anywhere in a non-comment-led line — flagging something
// like "# see .zshrc.devboost for details" as an unmarked source line
// even though nothing there actually sources it. That false positive
// was safe (it only produces an extra WARNING pending op telling the
// user to remove a line that isn't really there, never a silent
// double-source), but unnecessary now that the real forms in use are
// known and can be matched directly. See issue #12.
var zshUnmarkedSourceRe = regexp.MustCompile(`(^|[^#].*)(^|\s)(?:source|\.)\s+['"]?[^'"\n]*\.zshrc\.devboost`)

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
