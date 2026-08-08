// Command devboost-v2 is the spike CLI proving the Go engine's plan/apply
// mechanism end to end against one ported module (znap). Not a full CLI
// replacement — see the v2 architecture proposal for the migration plan.
package main

import (
	"fmt"
	"os"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/modules"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: devboost-v2 plan|apply")
		os.Exit(1)
	}

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	resources := modules.Znap(cfg)

	switch os.Args[1] {
	case "plan":
		err = engine.Plan(resources)
	case "apply":
		err = engine.Apply(resources)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
