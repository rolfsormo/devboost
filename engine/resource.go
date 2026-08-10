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
// typed parameters, plus any other resources that must converge before
// this one is diffed — expressed two ways:
//
//   - DependsOn names other resources directly by ID. Use this for
//     same-module, always-true orderings where the target's identity is
//     genuinely part of this resource's own contract (e.g. tmux's
//     plugin-install step DependsOn its own TPM clone — they're
//     declared together, in the same function, so naming the sibling
//     resource directly isn't a leak of anything).
//   - NeedsProvider names a capability tag (see Provides) instead of a
//     resource ID. Use this whenever the resource that actually
//     satisfies the need varies by platform or config, and the
//     declaring module has no business knowing which specific resource
//     that turns out to be — e.g. "I need mise installed" is true
//     regardless of whether mise came from a Homebrew formula, a pacman
//     package, or its own install script on Linux (see
//     linuxvendor.go/pkg.go). A module that hardcoded the concrete
//     resource ID for each of those cases would need to know another
//     module's internals just to express "I need this tool" — the
//     opposite of what dependency declaration is supposed to hide. The
//     engine resolves NeedsProvider into a real DependsOn-equivalent
//     edge at topoSort time, by looking up which resource (if any)
//     Provides that tag; ambiguous (more than one provider) or missing
//     (zero providers) are both real errors, not silently ignored.
type Resource struct {
	ID            string
	Kind          ResourceKind
	DependsOn     []string
	Provides      []string // capability tags this resource satisfies, e.g. "mise"
	NeedsProvider []string // capability tags this resource requires some other resource to Provide
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

// ExecutionResult is what DiffAndExecute returns: every resource's fate,
// not just "it worked" or a single top-level error. A resource that
// fails does not stop unrelated resources from converging — only
// resources that transitively DependsOn a failed one are skipped, since
// running them would mean diffing/executing against state the failed
// resource never actually reached.
type ExecutionResult struct {
	Applied []PendingOp       // executed successfully
	Failed  map[string]error  // resource ID -> the error its Execute returned
	Skipped map[string]string // resource ID -> ID of the failed resource it transitively depends on
}

// DiffAndExecute walks resources in topological order, diffing and (if
// there's a pending change) immediately executing each one before moving
// on — so a later resource's diff always sees the real effect of an
// earlier resource it depends on, not a stale pre-execution view. before
// is called on each PendingOp right before it's executed, letting the
// caller report progress; pass nil to skip reporting.
//
// A resource whose Execute fails does not abort the whole run: it's
// recorded in Failed, everything transitively depending on it (via
// DependsOn) is recorded in Skipped without being diffed or executed,
// and every resource NOT in that failure's dependency chain still runs
// normally. This matters in practice — e.g. one package genuinely
// unavailable on a given Linux distro's repos shouldn't prevent zsh
// config, git config, and every other independent resource from
// converging (found via a real Ubuntu container test where multiple
// packages weren't in the default apt repos at all).
//
// The returned error is reserved for conditions that make the whole run
// meaningless to continue — a dependency cycle, an unknown dependency,
// or a resource whose Diff itself errors (as opposed to Execute failing,
// which is a normal, expected, per-resource outcome recorded in Failed).
func DiffAndExecute(resources []Resource, before func(PendingOp)) (ExecutionResult, error) {
	ordered, err := topoSort(resources)
	if err != nil {
		return ExecutionResult{}, err
	}
	// Resolved once more here (topoSort already validated it, this just
	// needs the map itself) — dependsOnFailure must check the SAME
	// effective dependency set topoSort ordered on, including
	// NeedsProvider tags resolved to concrete IDs, or a resource whose
	// dependency came from NeedsProvider (not DependsOn) would never
	// correctly get skipped when its provider fails.
	effectiveDeps, err := resolveProviders(resources)
	if err != nil {
		return ExecutionResult{}, err
	}

	result := ExecutionResult{Failed: map[string]error{}, Skipped: map[string]string{}}
	// failedAncestor[id] = the ID of the failed (or itself-skipped)
	// resource that id transitively depends on, if any.
	failedAncestor := map[string]string{}

	for _, r := range ordered {
		if blocker, blocked := dependsOnFailure(r.ID, effectiveDeps, failedAncestor); blocked {
			result.Skipped[r.ID] = blocker
			failedAncestor[r.ID] = blocker
			continue
		}

		op, err := r.Kind.Diff()
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("resource %s: %w", r.ID, err)
		}
		if op == nil {
			continue
		}
		op.ResourceID = r.ID
		if op.Execute == nil {
			return ExecutionResult{}, fmt.Errorf("resource %s: pending op has no Execute", r.ID)
		}
		if before != nil {
			before(*op)
		}
		if err := op.Execute(); err != nil {
			result.Failed[r.ID] = err
			failedAncestor[r.ID] = r.ID
			continue
		}
		result.Applied = append(result.Applied, *op)
	}
	return result, nil
}

// dependsOnFailure reports whether resource id transitively depends
// (via its resolved effective dependencies — DependsOn plus any
// NeedsProvider tags already resolved to concrete IDs, see
// resolveProviders) on a resource that already failed or was itself
// skipped, and if so, the ID of that originating failure (so every
// downstream resource in the chain reports the same root cause, not
// just its immediate parent).
func dependsOnFailure(id string, effectiveDeps map[string][]string, failedAncestor map[string]string) (blocker string, blocked bool) {
	for _, dep := range effectiveDeps[id] {
		if origin, ok := failedAncestor[dep]; ok {
			return origin, true
		}
	}
	return "", false
}

// topoSort orders resources so that every resource comes after
// everything it DependsOn and everything it NeedsProvider (resolved to
// concrete resource IDs first — see resolveProviders). Returns an error
// on an unknown dependency ID, an unresolvable/ambiguous NeedsProvider,
// or a cycle.
func topoSort(resources []Resource) ([]Resource, error) {
	byID := make(map[string]Resource, len(resources))
	for _, r := range resources {
		if _, dup := byID[r.ID]; dup {
			return nil, fmt.Errorf("duplicate resource ID %q", r.ID)
		}
		byID[r.ID] = r
	}

	// effectiveDeps[id] = r.DependsOn plus every NeedsProvider tag
	// resolved to the concrete resource ID that Provides it. Kept
	// separate from mutating Resource.DependsOn itself, so a Resource's
	// own DependsOn field stays exactly what the caller declared.
	effectiveDeps, err := resolveProviders(resources)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		for _, dep := range effectiveDeps[r.ID] {
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
		for _, dep := range effectiveDeps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = visited
		ordered = append(ordered, byID[id])
		return nil
	}

	for _, r := range resources {
		if err := visit(r.ID); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

// resolveProviders builds, for every resource ID, its DependsOn list
// plus its NeedsProvider tags resolved to the concrete resource ID that
// Provides each tag. A tag with zero providers or more than one
// provider is a real error — silently picking "the first one" would
// hide a genuine configuration mistake (a module's NeedsProvider tag
// that no longer matches any Provides tag after a rename, or two
// modules accidentally claiming to Provide the same capability).
func resolveProviders(resources []Resource) (map[string][]string, error) {
	providerOf := make(map[string]string) // tag -> resource ID
	ambiguous := make(map[string][]string)
	for _, r := range resources {
		for _, tag := range r.Provides {
			if existing, ok := providerOf[tag]; ok {
				ambiguous[tag] = append(ambiguous[tag], existing, r.ID)
				continue
			}
			providerOf[tag] = r.ID
		}
	}

	effective := make(map[string][]string, len(resources))
	for _, r := range resources {
		deps := append([]string{}, r.DependsOn...)
		for _, tag := range r.NeedsProvider {
			if ids, isAmbiguous := ambiguous[tag]; isAmbiguous {
				return nil, fmt.Errorf("resource %s: capability %q is provided by more than one resource: %v", r.ID, tag, ids)
			}
			providerID, ok := providerOf[tag]
			if !ok {
				return nil, fmt.Errorf("resource %s: no resource provides required capability %q", r.ID, tag)
			}
			deps = append(deps, providerID)
		}
		effective[r.ID] = deps
	}
	return effective, nil
}
