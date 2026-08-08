// Command devboost is the CLI: a Terraform-inspired typed-resource
// engine that replaced the original bash implementation (a hand-written
// plan/apply function pair per module, which had already drifted out of
// sync in production). One diff function (engine.ComputeDiff) now
// backs both plan and apply, closing that whole bug class structurally
// rather than by convention.
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

Usage: devboost [COMMAND] [OPTIONS]

Commands:
  apply                     Converge machine to config (default)
  plan                      Show actions without changing anything
  doctor                    Check prerequisites and report per-module findings
  uninstall                 Remove managed files/blocks (leaves user custom files untouched)
  clean                     Remove devboost-disabled legacy-tooling lines and archived dirs
  migrate-from-oh-my-zsh    Remove oh-my-zsh and recover .zshrc customizations (needs --yes)

Options:
  --config FILE    Config file path (default: ~/.devboost.yaml)
  --dry-run        Show what would be done without making changes
  --yes            Confirm a destructive command (required by migrate-from-oh-my-zsh)
  --help, -h       Show this help message
  --version        Show version
`

const version = "2.0.0"

type flags struct {
	cmd        string
	configPath string
	dryRun     bool
	yes        bool
}

func main() {
	f := parseArgs(os.Args[1:])

	if f.cmd == "migrate-from-oh-my-zsh" {
		if err := modules.MigrateFromOhMyZsh(f.dryRun, f.yes); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load(f.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	detectedOS := kinds.DetectOS()

	switch f.cmd {
	case "plan":
		err = engine.Plan(modules.AllResources(cfg, detectedOS))
	case "apply":
		if f.dryRun {
			err = engine.Plan(modules.AllResources(cfg, detectedOS))
		} else {
			err = engine.Apply(modules.AllResources(cfg, detectedOS))
		}
	case "doctor":
		err = runDoctor(cfg, detectedOS)
	case "uninstall":
		err = modules.Uninstall(cfg)
	case "clean":
		err = modules.Clean(cfg, f.dryRun)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", f.cmd, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// parseArgs is intentionally small: a subcommand token plus
// --config/--dry-run/--yes, mirroring the bash tool's db_parse_flags for
// the subset of flags this CLI currently supports. --verbose is not yet
// wired (nothing here has a verbose-vs-normal output distinction yet).
func parseArgs(args []string) flags {
	f := flags{cmd: "apply", configPath: config.DefaultPath()}

	i := 0
	if len(args) > 0 {
		switch args[0] {
		case "apply", "plan", "doctor", "uninstall", "clean", "migrate-from-oh-my-zsh":
			f.cmd = args[0]
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
				f.configPath = args[i+1]
				i++
			}
		case "--dry-run":
			f.dryRun = true
		case "--yes":
			f.yes = true
		case "--help", "-h":
			fmt.Print(usage)
			os.Exit(0)
		case "--version":
			fmt.Println("devboost", version)
			os.Exit(0)
		}
	}
	return f
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
