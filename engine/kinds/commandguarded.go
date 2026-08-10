package kinds

import (
	"fmt"

	"github.com/rolfsormo/devboost/engine"
)

// CommandGuarded is the architecture's one deliberate escape hatch for
// state that doesn't fit any of the other typed kinds. A module still
// only ever declares data — {ID, Params, Wants} — never imperative logic
// at the declaration site. What makes this a real escape hatch, not a
// loophole, is that CommandGuarded{ID: "x", ...} does nothing on its own:
// ID must match an entry hand-registered in this package via
// RegisterCommand, with real Go diff/apply logic behind it. There is no
// generic "run a script and check the exit code" shortcut — adding a new
// use requires writing an implementation in core, the same amount of real
// work as adding a proper new kind, which is the whole point: this must
// never be the easy path when a real typed kind (File, Package,
// GitConfig, ...) is achievable instead.
//
// Params carries whatever plain data the registered implementation needs
// (e.g. a configured path) — still just data the module declares, not
// logic; the implementation registered for ID decides what shape it
// expects and type-asserts accordingly.
//
// An unregistered ID is a startup-time error (Diff returns an error), not
// a silent no-op — declaring one without an implementation should fail
// loudly, the same way a struct literal referencing an undefined type
// wouldn't compile.
type CommandGuarded struct {
	ID     string
	Params any
	Wants  string
}

// GuardedCommand is what RegisterCommand takes: the real diff/apply logic
// behind one CommandGuarded ID. Both functions receive the Params value
// from the CommandGuarded that triggered them.
type GuardedCommand struct {
	// Satisfied reports whether the desired state already holds.
	Satisfied func(params any) (bool, error)
	// Converge brings the system to the desired state. Only called when
	// Satisfied returned false.
	Converge func(params any) error
	// UndoConverge reverses the effect of the most recent Converge, if
	// possible. Optional — nil (the default for every registered command
	// except the ones that explicitly set it) means this command is
	// simply not undoable, which CommandGuarded.Undo reports as "nothing
	// to undo" rather than an error. Only set this when Converge's
	// mutation is genuinely and safely reversible (e.g. the oh-my-zsh
	// migration, whose Converge only ever renames/archives, never
	// deletes) — most registered commands (starting a brew service,
	// enabling corepack, running mise install) have no clean undo and
	// should leave this nil rather than fake one.
	UndoConverge func(params any) error
	// UndoSatisfied reports whether there is currently nothing to undo
	// (e.g. Converge never ran, or a previous Undo already restored
	// everything) — the undo-side mirror of Satisfied, checked the same
	// way Diff checks Satisfied before deciding whether to return a
	// PendingOp at all. Required whenever UndoConverge is set (a
	// registered command with UndoConverge but no UndoSatisfied is a
	// startup-time error from CommandGuarded.Undo, the same "fail
	// loudly, don't silently no-op" contract an unregistered ID gets)
	// — without it, Undo would have no way to tell "already undone,
	// nothing to do" apart from "run UndoConverge unconditionally,"
	// which would make repeated `devboost undo` runs either error out on
	// the second call or silently redo work that shouldn't repeat.
	UndoSatisfied func(params any) (bool, error)
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
	satisfied, err := cmd.Satisfied(c.Params)
	if err != nil {
		return nil, fmt.Errorf("CommandGuarded %q: %w", c.ID, err)
	}
	if satisfied {
		return nil, nil
	}
	return &engine.PendingOp{
		Description: c.Wants,
		Execute:     func() error { return cmd.Converge(c.Params) },
	}, nil
}

// Undo implements engine.Undoer. An unregistered ID is still a real
// error (same as Diff — declaring one without an implementation should
// fail loudly), and a registered command with UndoConverge set but no
// UndoSatisfied is also an error (see UndoSatisfied's doc comment) — but
// a command with no UndoConverge at all is simply not undoable: nil
// PendingOp, nil error, exactly like Diff reporting nothing pending.
func (c CommandGuarded) Undo() (*engine.PendingOp, error) {
	cmd, ok := guardedCommands[c.ID]
	if !ok {
		return nil, fmt.Errorf("CommandGuarded %q has no registered implementation — see kinds.RegisterCommand", c.ID)
	}
	if cmd.UndoConverge == nil {
		return nil, nil
	}
	if cmd.UndoSatisfied == nil {
		return nil, fmt.Errorf("CommandGuarded %q registers UndoConverge but no UndoSatisfied — see kinds.GuardedCommand", c.ID)
	}
	satisfied, err := cmd.UndoSatisfied(c.Params)
	if err != nil {
		return nil, fmt.Errorf("CommandGuarded %q: %w", c.ID, err)
	}
	if satisfied {
		return nil, nil
	}
	return &engine.PendingOp{
		Description: "undo: " + c.Wants,
		Execute:     func() error { return cmd.UndoConverge(c.Params) },
	}, nil
}
