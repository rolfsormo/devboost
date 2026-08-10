package kinds

import "testing"

func TestCommandGuardedErrorsWhenUnregistered(t *testing.T) {
	c := CommandGuarded{ID: "definitely-not-registered", Wants: "something"}
	_, err := c.Diff()
	if err == nil {
		t.Fatal("expected an error for an unregistered CommandGuarded ID, got nil — this must never silently no-op")
	}
}

func TestCommandGuardedDiffNilWhenSatisfied(t *testing.T) {
	RegisterCommand("test-satisfied", GuardedCommand{
		Satisfied: func(any) (bool, error) { return true, nil },
		Converge:  func(any) error { t.Fatal("Converge should not be called when Satisfied"); return nil },
	})
	c := CommandGuarded{ID: "test-satisfied", Wants: "should already be true"}
	op, err := c.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending op when already satisfied, got %+v", op)
	}
}

func TestCommandGuardedDiffPendingWhenUnsatisfied(t *testing.T) {
	converged := false
	RegisterCommand("test-unsatisfied", GuardedCommand{
		Satisfied: func(any) (bool, error) { return converged, nil },
		Converge: func(any) error {
			converged = true
			return nil
		},
	})
	c := CommandGuarded{ID: "test-unsatisfied", Wants: "convergence needed"}
	op, err := c.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op == nil {
		t.Fatal("expected a pending op when unsatisfied")
	}
	if op.Description != "convergence needed" {
		t.Fatalf("got description %q, want %q", op.Description, "convergence needed")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !converged {
		t.Fatal("expected Execute to call Converge")
	}
}

// TestCommandGuardedPassesParams confirms Params flows through to both
// Satisfied and Converge unmodified — the mechanism tmux's plugin-install
// use (a configured TPM path) and similar parameterized uses depend on.
func TestCommandGuardedPassesParams(t *testing.T) {
	var seenBySatisfied, seenByConverge string
	RegisterCommand("test-params", GuardedCommand{
		Satisfied: func(p any) (bool, error) {
			seenBySatisfied = p.(string)
			return false, nil
		},
		Converge: func(p any) error {
			seenByConverge = p.(string)
			return nil
		},
	})
	c := CommandGuarded{ID: "test-params", Params: "hello", Wants: "x"}
	op, err := c.Diff()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if seenBySatisfied != "hello" {
		t.Fatalf("Satisfied saw %q, want %q", seenBySatisfied, "hello")
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if seenByConverge != "hello" {
		t.Fatalf("Converge saw %q, want %q", seenByConverge, "hello")
	}
}

func TestCommandGuardedUndoErrorsWhenUnregistered(t *testing.T) {
	c := CommandGuarded{ID: "definitely-not-registered-undo", Wants: "something"}
	_, err := c.Undo()
	if err == nil {
		t.Fatal("expected an error for an unregistered CommandGuarded ID, got nil")
	}
}

func TestCommandGuardedUndoNilWhenNoUndoConverge(t *testing.T) {
	RegisterCommand("test-no-undo", GuardedCommand{
		Satisfied: func(any) (bool, error) { return true, nil },
		Converge:  func(any) error { return nil },
	})
	c := CommandGuarded{ID: "test-no-undo", Wants: "x"}
	op, err := c.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending undo op when UndoConverge is nil, got %+v", op)
	}
}

func TestCommandGuardedUndoCallsUndoConverge(t *testing.T) {
	undone := false
	RegisterCommand("test-with-undo", GuardedCommand{
		Satisfied:     func(any) (bool, error) { return true, nil },
		Converge:      func(any) error { return nil },
		UndoConverge:  func(any) error { undone = true; return nil },
		UndoSatisfied: func(any) (bool, error) { return false, nil },
	})
	c := CommandGuarded{ID: "test-with-undo", Wants: "x"}
	op, err := c.Undo()
	if err != nil || op == nil {
		t.Fatalf("op=%+v err=%v", op, err)
	}
	if err := op.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !undone {
		t.Fatal("expected Undo's Execute to call UndoConverge")
	}
}

func TestCommandGuardedUndoNilWhenUndoSatisfied(t *testing.T) {
	called := false
	RegisterCommand("test-undo-satisfied", GuardedCommand{
		Satisfied:     func(any) (bool, error) { return true, nil },
		Converge:      func(any) error { return nil },
		UndoConverge:  func(any) error { called = true; return nil },
		UndoSatisfied: func(any) (bool, error) { return true, nil },
	})
	c := CommandGuarded{ID: "test-undo-satisfied", Wants: "x"}
	op, err := c.Undo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op != nil {
		t.Fatalf("expected no pending undo op when UndoSatisfied is true, got %+v", op)
	}
	if called {
		t.Fatal("expected UndoConverge not to be called when nothing to undo")
	}
}

func TestCommandGuardedUndoErrorsWhenUndoConvergeSetButNoUndoSatisfied(t *testing.T) {
	RegisterCommand("test-undo-missing-satisfied", GuardedCommand{
		Satisfied:    func(any) (bool, error) { return true, nil },
		Converge:     func(any) error { return nil },
		UndoConverge: func(any) error { return nil },
	})
	c := CommandGuarded{ID: "test-undo-missing-satisfied", Wants: "x"}
	if _, err := c.Undo(); err == nil {
		t.Fatal("expected an error when UndoConverge is set without UndoSatisfied")
	}
}
