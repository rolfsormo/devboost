package engine

import "fmt"

// Apply diffs and executes resources one at a time, in dependency order —
// see the package doc for why this can't be "compute the same diff Plan
// computes, then execute it": a resource that depends on another must be
// diffed after that dependency has actually executed, not against a
// stale, pre-execution view of the system.
//
// A failed resource doesn't stop the rest of the run — see
// DiffAndExecute's ExecutionResult. Apply reports every outcome (done,
// failed, skipped-because-a-dependency-failed) and returns a non-nil
// error only when at least one resource actually failed, so callers
// (e.g. the CLI) still exit non-zero, but everything that could
// converge, did.
func Apply(resources []Resource) error {
	result, err := DiffAndExecute(resources, func(op PendingOp) {
		fmt.Printf("%s...\n", op.Description)
	})
	if err != nil {
		return err
	}

	for _, op := range result.Applied {
		fmt.Printf("Done: %s\n", op.Description)
	}
	if len(result.Applied) == 0 && len(result.Failed) == 0 && len(result.Skipped) == 0 {
		fmt.Println("No changes.")
	}

	if len(result.Failed) == 0 {
		return nil
	}

	for id, ferr := range result.Failed {
		fmt.Printf("Failed: %s: %v\n", id, ferr)
	}
	for id, blocker := range result.Skipped {
		fmt.Printf("Skipped: %s (depends on failed resource %s)\n", id, blocker)
	}
	return fmt.Errorf("%d resource(s) failed to apply", len(result.Failed))
}
