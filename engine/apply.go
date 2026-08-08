package engine

import "fmt"

// Apply diffs and executes resources one at a time, in dependency order —
// see the package doc for why this can't be "compute the same diff Plan
// computes, then execute it": a resource that depends on another must be
// diffed after that dependency has actually executed, not against a
// stale, pre-execution view of the system.
func Apply(resources []Resource) error {
	ops, err := DiffAndExecute(resources, func(op PendingOp) {
		fmt.Printf("%s...\n", op.Description)
	})
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		fmt.Println("No changes.")
		return nil
	}
	for _, op := range ops {
		fmt.Printf("Done: %s\n", op.Description)
	}
	return nil
}
