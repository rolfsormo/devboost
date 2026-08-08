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
		Satisfied: func() (bool, error) { return true, nil },
		Converge:  func() error { t.Fatal("Converge should not be called when Satisfied"); return nil },
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
		Satisfied: func() (bool, error) { return converged, nil },
		Converge: func() error {
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
