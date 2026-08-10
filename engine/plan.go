package engine

import "fmt"

// Plan computes the diff and prints it without executing anything.
func Plan(resources []Resource) error {
	ops, err := ComputeDiff(resources)
	if err != nil {
		return err
	}
	if len(ops) == 0 {
		fmt.Println("No changes.")
		return nil
	}
	for _, op := range ops {
		fmt.Printf("Would: %s\n", op.Description)
	}
	return nil
}
