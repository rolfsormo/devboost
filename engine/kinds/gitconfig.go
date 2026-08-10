package kinds

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/rolfsormo/devboost/engine"
)

// GitConfig declares "this git config key should have this value" at the
// given scope. Both the read (diff) and write (apply) shell out to git
// config itself — git's own config format/parsing is the correct
// primitive here, not something to reimplement.
type GitConfig struct {
	Key   string
	Value string
	// Scope is a git config scope flag, e.g. "--global" (the bash tool's
	// only current use) or "--local". Defaults to "--global" if empty.
	Scope string
}

func (g GitConfig) scope() string {
	if g.Scope == "" {
		return "--global"
	}
	return g.Scope
}

func (g GitConfig) current() (string, bool, error) {
	out, err := exec.Command("git", "config", g.scope(), "--get", g.Key).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// git config --get exits 1 when the key isn't set — not an error.
			return "", false, nil
		}
		return "", false, fmt.Errorf("git config --get %s: %w", g.Key, err)
	}
	return strings.TrimSuffix(string(out), "\n"), true, nil
}

func (g GitConfig) Diff() (*engine.PendingOp, error) {
	current, set, err := g.current()
	if err != nil {
		return nil, err
	}
	if set && current == g.Value {
		return nil, nil
	}
	return &engine.PendingOp{
		Description: fmt.Sprintf("git config %s %s %s", g.scope(), g.Key, g.Value),
		Execute: func() error {
			return exec.Command("git", "config", g.scope(), g.Key, g.Value).Run()
		},
	}, nil
}
