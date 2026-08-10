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
  apply           Converge machine to config (default)
  plan            Show actions without changing anything
  doctor          Check prerequisites and report per-module findings
  undo            Reverse a prior optimization (zinit/asdf/nvm/oh-my-zsh dedup)
  uninstall       Remove managed files/blocks (leaves user custom files untouched)
  clean           Remove devboost-disabled optimization lines and archived dirs

Options:
  --config FILE        Config file path (default: ~/.devboost.yaml)
  --dry-run            Show what would be done without making changes
  --no-optimizations   Skip the startup optimizations (zinit/asdf/nvm/oh-my-zsh
                        dedup) for this run; same as optimize.enable: false in config
  --force              Let 'undo' proceed even though something it would
                        restore has changed since it last converged
                        (undo refuses by default)
  --help, -h           Show this help message
  --version            Show version
`

const version = "2.0.0"

type flags struct {
	cmd             string
	configPath      string
	dryRun          bool
	noOptimizations bool
	force           bool
}

func main() {
	f := parseArgs(os.Args[1:])

	cfg, err := config.Load(f.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}
	if f.noOptimizations {
		cfg.Set("optimize.enable", "false")
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
	case "undo":
		err = runUndo(cfg, detectedOS, f.dryRun, f.force)
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
// --config/--dry-run/--no-optimizations, mirroring the bash tool's
// db_parse_flags for the subset of flags this CLI currently supports.
// --verbose is not yet wired (nothing here has a verbose-vs-normal
// output distinction yet).
func parseArgs(args []string) flags {
	f := flags{cmd: "apply", configPath: config.DefaultPath()}

	i := 0
	if len(args) > 0 {
		switch args[0] {
		case "apply", "plan", "doctor", "undo", "uninstall", "clean":
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
		case "--no-optimizations":
			f.noOptimizations = true
		case "--force":
			f.force = true
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

// runUndo reverses whatever the most recent apply converged, for every
// resource whose Kind implements engine.Undoer AND currently has
// something to undo. Checking only the type assertion isn't enough:
// kinds.CommandGuarded implements Undo() unconditionally (it has to, to
// satisfy the interface at all), but returns (nil, nil) for the many
// registered commands with no UndoConverge — e.g. tmux plugin install,
// mise toolchains, corepack all type-assert as engine.Undoer without
// being meaningfully undoable. Calling Undo() up front (read-only, same
// contract Diff() has) and keeping only the resources that actually
// returned a pending op is what correctly narrows this down to the real
// four optimization resources, not everything that happens to share the
// interface.
//
// Refuses to run at all (unless force) when any of those real undoable
// resources themselves have a pending diff — deliberately scoped this
// narrowly, not a whole-system scan: several resources (tmux plugin
// install, mise toolchains, corepack) are correctly "always re-run" —
// their own Satisfied always reports false by design, not because
// anything drifted — so a literal zero-pending-diff-anywhere bar could
// never be met on a real machine. What actually matters for undo's
// correctness is narrower: has the specific thing it's about to restore
// drifted since it last converged (e.g. the user hand-edited the
// disabled line, or re-created ~/.oh-my-zsh) — if so, undo's own
// backups may no longer describe the current state accurately, which is
// exactly the case --force exists for.
func runUndo(cfg *config.Config, os kinds.OS, dryRun, force bool) error {
	resources := modules.AllResources(cfg, os)

	type undoTarget struct {
		resource engine.Resource
		op       *engine.PendingOp
	}
	var targets []undoTarget
	for _, r := range resources {
		undoer, ok := r.Kind.(engine.Undoer)
		if !ok {
			continue
		}
		op, err := undoer.Undo()
		if err != nil {
			fmt.Printf("%s: error checking undo: %v\n", r.ID, err)
			continue
		}
		if op == nil {
			continue
		}
		targets = append(targets, undoTarget{resource: r, op: op})
	}

	if !force {
		var undoable []engine.Resource
		for _, t := range targets {
			undoable = append(undoable, t.resource)
		}
		pending, err := engine.ComputeDiff(undoable)
		if err != nil {
			return err
		}
		if len(pending) > 0 {
			fmt.Println("Refusing to undo: the following have changed since they last converged:")
			for _, op := range pending {
				fmt.Printf("  ⚠ %s\n", op.Description)
			}
			fmt.Println("Run 'devboost apply' first, or pass --force to undo anyway.")
			return nil
		}
	}

	for _, t := range targets {
		op := t.op
		if dryRun {
			fmt.Printf("Would: %s\n", op.Description)
			continue
		}
		fmt.Printf("%s...\n", op.Description)
		if err := op.Execute(); err != nil {
			fmt.Printf("Failed: %s: %v\n", op.Description, err)
			continue
		}
		fmt.Printf("Done: %s\n", op.Description)
	}
	if len(targets) == 0 {
		fmt.Println("Nothing to undo.")
	}
	return nil
}
