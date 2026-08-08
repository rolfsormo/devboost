// Command devboost-v2 is the Go-engine CLI, coexisting with the bash
// tool (devboost.sh) during the v2 migration. Not yet wired into a
// bootstrap/release pipeline — see the v2 architecture proposal and
// issue #4 for the migration plan and cutover criteria.
package main

import (
	"fmt"
	"os"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
	"github.com/rolfsormo/devboost/engine/modules"
)

const usage = `devboost - Bootstrap a modern dev environment

Usage: devboost-v2 [COMMAND] [OPTIONS]

Commands:
  apply     Converge machine to config (default)
  plan      Show actions without changing anything
  doctor    Check prerequisites and report per-module findings

Options:
  --config FILE    Config file path (default: ~/.devboost.yaml)
  --help, -h       Show this help message
  --version        Show version
`

const version = "2.0.0-dev"

func main() {
	cmd, configPath := parseArgs(os.Args[1:])

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	detectedOS := kinds.DetectOS()

	switch cmd {
	case "plan":
		err = engine.Plan(modules.AllResources(cfg, detectedOS))
	case "apply":
		err = engine.Apply(modules.AllResources(cfg, detectedOS))
	case "doctor":
		err = runDoctor(cfg, detectedOS)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// parseArgs is intentionally small: a subcommand token plus --config,
// mirroring the bash tool's db_parse_flags for the subset of flags this
// CLI currently supports. --dry-run/--verbose/--yes and the
// uninstall/migrate-from-oh-my-zsh subcommands are not yet wired here —
// see tasks #16/#17.
func parseArgs(args []string) (cmd string, configPath string) {
	cmd = "apply"
	configPath = config.DefaultPath()

	i := 0
	if len(args) > 0 {
		switch args[0] {
		case "apply", "plan", "doctor":
			cmd = args[0]
			i = 1
		case "--help", "-h":
			fmt.Print(usage)
			os.Exit(0)
		case "--version":
			fmt.Println("devboost", version)
			os.Exit(0)
		}
	}

	for ; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Print(usage)
			os.Exit(0)
		case "--version":
			fmt.Println("devboost", version)
			os.Exit(0)
		}
	}
	return cmd, configPath
}

// runDoctor computes and prints per-module findings, grouped by module
// name — the tool-first grouping from the architecture doc, so this
// stays readable regardless of how many resources/dedup-checks
// accumulate inside any one module over time.
func runDoctor(cfg *config.Config, os kinds.OS) error {
	names := make([]string, len(modules.All))
	resourcesByModule := make([][]engine.Resource, len(modules.All))
	diagnosticsByModule := make([]engine.DiagnosticFunc, len(modules.All))
	for i, m := range modules.All {
		names[i] = m.Name
		resourcesByModule[i] = m.Resources(cfg, os)
		if m.Diagnostics != nil {
			diagnosticsByModule[i] = m.Diagnostics(cfg, os)
		}
	}

	reports, err := engine.Doctor(names, resourcesByModule, diagnosticsByModule)
	if err != nil {
		return err
	}
	if len(reports) == 0 {
		fmt.Println("Everything looks good.")
		return nil
	}
	for _, r := range reports {
		fmt.Printf("%s:\n", r.Name)
		for _, op := range r.Pending {
			fmt.Printf("  ⚠ %s\n", op.Description)
		}
		for _, d := range r.Diagnostics {
			mark := "✓"
			if d.Warn {
				mark = "⚠"
			}
			fmt.Printf("  %s %s\n", mark, d.Message)
		}
	}
	return nil
}
