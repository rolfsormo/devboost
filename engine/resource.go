// Package engine implements devboost's typed-resource diff/apply model.
//
// A module declares desired state as a list of Resources. Resources may
// declare dependencies on other resources (DependsOn), replacing what used
// to be an implicit, hand-maintained ordering in build.sh's registration
// list with an explicit, checkable graph — e.g. starship declares it
// depends on the pkg resource that installs the starship binary, tmux's
// plugin-install step depends on TPM's clone, zsh's rendered config
// depends on znap being installed.
//
// Plan and apply both traverse resources in topological order, but they
// diff differently: Plan diffs every resource once, up front, since
// nothing is actually converging as it goes — the printed result describes
// what a real run would do. Apply cannot do that in one batch: if resource
// B depends on resource A, B's diff must run *after* A's Execute, or B
// would diff against pre-A state (e.g. checking whether a binary exists
// before the resource that installs it has actually run). So Apply
// diffs-then-executes one resource at a time, in dependency order.
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
// state. It is never authored directly by a module — only a Diff call
// produces it.
type PendingOp struct {
	ResourceID  string
	Description string
	Execute     func() error
}

// Resource is what a module declares: an ID plus a kind carrying its own
// typed parameters, plus any other resources (by ID) that must converge
// before this one is diffed.
type Resource struct {
	ID        string
	Kind      ResourceKind
	DependsOn []string
}

// ComputeDiff diffs every resource once, in topological order, without
// executing anything — this is what Plan uses, since nothing needs to
// have actually converged for a description of "what would happen" to be
// accurate.
func ComputeDiff(resources []Resource) ([]PendingOp, error) {
	ordered, err := topoSort(resources)
	if err != nil {
		return nil, err
	}
	var ops []PendingOp
	for _, r := range ordered {
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

// DiffAndExecute walks resources in topological order, diffing and (if
// there's a pending change) immediately executing each one before moving
// on — so a later resource's diff always sees the real effect of an
// earlier resource it depends on, not a stale pre-execution view. before
// is called on each PendingOp right before it's executed, letting the
// caller report progress; pass nil to skip reporting. Returns every
// PendingOp that was executed.
func DiffAndExecute(resources []Resource, before func(PendingOp)) ([]PendingOp, error) {
	ordered, err := topoSort(resources)
	if err != nil {
		return nil, err
	}
	var ops []PendingOp
	for _, r := range ordered {
		op, err := r.Kind.Diff()
		if err != nil {
			return nil, fmt.Errorf("resource %s: %w", r.ID, err)
		}
		if op == nil {
			continue
		}
		op.ResourceID = r.ID
		if op.Execute == nil {
			return nil, fmt.Errorf("resource %s: pending op has no Execute", r.ID)
		}
		if before != nil {
			before(*op)
		}
		if err := op.Execute(); err != nil {
			return nil, fmt.Errorf("resource %s: %w", r.ID, err)
		}
		ops = append(ops, *op)
	}
	return ops, nil
}

// topoSort orders resources so that every resource comes after everything
// it DependsOn. Returns an error on an unknown dependency ID or a cycle.
func topoSort(resources []Resource) ([]Resource, error) {
	byID := make(map[string]Resource, len(resources))
	for _, r := range resources {
		if _, dup := byID[r.ID]; dup {
			return nil, fmt.Errorf("duplicate resource ID %q", r.ID)
		}
		byID[r.ID] = r
	}
	for _, r := range resources {
		for _, dep := range r.DependsOn {
			if _, ok := byID[dep]; !ok {
				return nil, fmt.Errorf("resource %s: depends on unknown resource %q", r.ID, dep)
			}
		}
	}

	// state[id] is absent (zero value, unvisited) / visiting / visited.
	const (
		visiting = 1
		visited  = 2
	)
	state := make(map[string]int, len(resources))
	var ordered []Resource

	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("dependency cycle involving resource %q", id)
		}
		state[id] = visiting
		r := byID[id]
		for _, dep := range r.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = visited
		ordered = append(ordered, r)
		return nil
	}

	for _, r := range resources {
		if err := visit(r.ID); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
