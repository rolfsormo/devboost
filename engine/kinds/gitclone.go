package kinds

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rolfsormo/devboost/engine"
)

// GitClone declares "this URL should be cloned to this path." Its diff
// reuses DirExists (does Dest exist) rather than reimplementing that check —
// its Apply shells out to git clone itself, since git's own clone
// implementation is the correct primitive here, not something to
// reimplement in Go.
type GitClone struct {
	URL  string
	Dest string
}

func (g GitClone) Diff() (*engine.PendingOp, error) {
	op, err := DirExists{Path: g.Dest}.Diff()
	if err != nil || op == nil {
		return op, err
	}
	return &engine.PendingOp{
		Description: fmt.Sprintf("git clone %s to %s", g.URL, g.Dest),
		Execute: func() error {
			if err := os.MkdirAll(filepath.Dir(g.Dest), 0o755); err != nil {
				return err
			}
			// -c credential.helper= (empty value) disables credential
			// helpers for this invocation only, overriding any helper
			// configured system-wide (e.g. macOS's credential.helper =
			// osxkeychain in /opt/homebrew/etc/gitconfig). Without this,
			// a successful anonymous HTTPS clone still triggers
			// `git-credential-osxkeychain store` afterwards, which blocks
			// on a Keychain GUI prompt that never comes in a headless
			// apply run — confirmed by direct process-tree inspection of
			// a real hung `devboost apply`. devboost only ever clones
			// public repos anonymously, so there is never a credential to
			// store.
			cmd := exec.Command("git", "-c", "credential.helper=", "clone", "--depth", "1", g.URL, g.Dest)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}, nil
}
