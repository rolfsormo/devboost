// Package engine implements devboost's typed-resource diff/apply model.
//
// A module declares desired state as a list of Resources. ComputeDiff is the
// single function that compares desired state to live system state and
// returns the pending operations needed to reconcile them. Both plan and
// apply call ComputeDiff identically — plan stops after printing the result,
// apply goes on to execute it. There is no dry-run flag threaded through the
// diffing logic anywhere: dry-run is purely "the caller that doesn't invoke
// PendingOp.Execute."
package engine

import "fmt"

// ResourceKind is implemented once per typed resource kind, in this package
// or a subpackage — never duplicated per module. A kind's own Diff is the
// only place its idempotency logic lives.
type ResourceKind interface {
	// Diff reports whether current system state already matches desired
	// state. A nil PendingOp means nothing needs to change.
	Diff() (*PendingOp, error)
}

// PendingOp is the output of a diff: the delta between desired and live
// state. It is never authored directly by a module — only ComputeDiff
// produces it.
type PendingOp struct {
	ResourceID  string
	Description string
	Execute     func() error
}

// Resource is what a module declares: an ID plus a kind carrying its own
// typed parameters.
type Resource struct {
	ID   string
	Kind ResourceKind
}

// ComputeDiff is the single function both plan and apply call.
func ComputeDiff(resources []Resource) ([]PendingOp, error) {
	var ops []PendingOp
	for _, r := range resources {
		op, err := r.Kind.Diff()
		if err != nil {
			return nil, fmt.Errorf("resource %s: %w", r.ID, err)
		}
		if op != nil {
			op.ResourceID = r.ID
			ops = append(ops, *op)
		}
	}
	return ops, nil
}
