package kinds

import (
	"fmt"

	"github.com/rolfsormo/devboost/engine"
)

// CommandGuarded is the architecture's one deliberate escape hatch for
// state that doesn't fit any of the other typed kinds. A module still
// only ever declares data — {ID, Wants} — never imperative logic at the
// declaration site. What makes this a real escape hatch, not a loophole,
// is that CommandGuarded{ID: "x", ...} does nothing on its own: ID must
// match an entry hand-registered in this package via RegisterCommand,
// with real Go diff/apply logic behind it. There is no generic "run a
// script and check the exit code" shortcut — adding a new use requires
// writing an implementation in core, the same amount of real work as
// adding a proper new kind, which is the whole point: this must never be
// the easy path when a real typed kind (File, Package, GitConfig, ...) is
// achievable instead.
//
// An unregistered ID is a startup-time error (Diff returns an error), not
// a silent no-op — declaring one without an implementation should fail
// loudly, the same way a struct literal referencing an undefined type
// wouldn't compile.
type CommandGuarded struct {
	ID    string
	Wants string
}

// GuardedCommand is what RegisterCommand takes: the real diff/apply logic
// behind one CommandGuarded ID.
type GuardedCommand struct {
	// Satisfied reports whether the desired state already holds.
	Satisfied func() (bool, error)
	// Converge brings the system to the desired state. Only called when
	// Satisfied returned false.
	Converge func() error
}

var guardedCommands = map[string]GuardedCommand{}

// RegisterCommand registers the real implementation behind a
// CommandGuarded ID. Intended to be called from an init() in the file
// that owns the concrete use case (e.g. mise's npm-globals-migrated
// check), not from module declaration sites.
func RegisterCommand(id string, cmd GuardedCommand) {
	guardedCommands[id] = cmd
}

func (c CommandGuarded) Diff() (*engine.PendingOp, error) {
	cmd, ok := guardedCommands[c.ID]
	if !ok {
		return nil, fmt.Errorf("CommandGuarded %q has no registered implementation — see kinds.RegisterCommand", c.ID)
	}
	ok, err := cmd.Satisfied()
	if err != nil {
		return nil, fmt.Errorf("CommandGuarded %q: %w", c.ID, err)
	}
	if ok {
		return nil, nil
	}
	return &engine.PendingOp{
		Description: c.Wants,
		Execute:     cmd.Converge,
	}, nil
}
