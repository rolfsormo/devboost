// Package kinds implements devboost's built-in resource kinds — the native
// Go diff/apply logic behind what modules declare as struct literals.
package kinds

import (
	"fmt"
	"os"

	"github.com/rolfsormo/devboost/engine"
)

// DirExists is a built-in, parametrized diff-kind: "this directory should
// exist." It is the first candidate for the diff-kind shorthand devboost's
// architecture decided it wants — most modules needing "ensure a directory
// exists" declare this kind directly rather than writing a bespoke
// ResourceKind implementation.
type DirExists struct {
	Path string
}

func (d DirExists) Diff() (*engine.PendingOp, error) {
	info, err := os.Stat(d.Path)
	if err == nil && info.IsDir() {
		return nil, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat %s: %w", d.Path, err)
	}
	return &engine.PendingOp{
		Description: fmt.Sprintf("create directory %s", d.Path),
		Execute: func() error {
			return os.MkdirAll(d.Path, 0o755)
		},
	}, nil
}
