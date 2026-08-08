package engine

import "fmt"

// Apply computes the same diff Plan does, then executes each pending
// operation. It calls the identical ComputeDiff — there is no separate
// "what apply would do" computation.
func Apply(resources []Resource) error {
	ops, err := ComputeDiff(resources)
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		fmt.Println("No changes.")
		return nil
	}
	for _, op := range ops {
		fmt.Printf("%s...\n", op.Description)
		if op.Execute == nil {
			return fmt.Errorf("resource %s: pending op has no Execute", op.ResourceID)
		}
		if err := op.Execute(); err != nil {
			return fmt.Errorf("resource %s: %w", op.ResourceID, err)
		}
		fmt.Printf("Done: %s\n", op.Description)
	}
	return nil
}
