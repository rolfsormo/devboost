package modules

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// miseToolchains carries the desired toolchain versions through
// CommandGuarded's Params — plain data, same rule as tmux's TPM path.
type miseToolchains struct {
	node, python, goVersion, rust, deno string
}

func (m miseToolchains) args() []string {
	return []string{
		"node@" + m.node,
		"python@" + m.python,
		"go@" + m.goVersion,
		"rust@" + m.rust,
		"deno@" + m.deno,
	}
}

func currentMiseNodeVersion() string {
	out, err := exec.Command("mise", "current", "node").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// npmGlobals lists globally installed npm packages, excluding npm/corepack
// themselves — ports _db_mise_npm_globals.
func npmGlobals() []string {
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		return nil
	}
	npmBin := nodeBin[:strings.LastIndex(nodeBin, "/")+1] + "npm"
	if _, err := os.Stat(npmBin); err != nil {
		return nil
	}
	out, err := exec.Command(npmBin, "list", "-g", "--depth=0", "--parseable").Output()
	if err != nil {
		return nil
	}
	return parseNpmGlobalsOutput(string(out))
}

// parseNpmGlobalsOutput extracts package names from `npm list -g --depth=0
// --parseable` output, excluding npm/corepack themselves — split out from
// npmGlobals so the parsing logic is testable without a real npm/node on
// the test machine.
//
// npm always prints the global node_modules directory itself as the
// first line, with no package name after it — its last path segment is
// literally "node_modules". The bash tool's original awk filter (NF>1 &&
// ...) didn't exclude this by name, only by "has a slash," which the
// node_modules directory line also satisfies — so it leaked through as a
// false-positive "package" (confirmed by testing the actual awk command
// against real npm-shaped output). Harmless in practice (worst case:
// offering to `npm install -g node_modules`, which just fails) but a
// real bug nonetheless — fixed here by name rather than ported
// faithfully, since a bug found while porting doesn't need to survive
// the port, and filtering by name (not position) doesn't depend on
// npm's output ordering staying stable.
func parseNpmGlobalsOutput(out string) []string {
	var pkgs []string
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "/")
		if len(parts) < 2 {
			continue
		}
		name := parts[len(parts)-1]
		if name != "" && name != "npm" && name != "corepack" && name != "node_modules" {
			pkgs = append(pkgs, name)
		}
	}
	return pkgs
}

func confirm(prompt string) bool {
	fmt.Printf("? %s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line)) == "y"
}

func init() {
	kinds.RegisterCommand("mise_toolchains_converged", kinds.GuardedCommand{
		// mise use/install are both idempotent no-ops when already at the
		// desired versions, and the bash version never checked beforehand
		// either — it always ran both commands. Faithful port: always
		// pending when this resource exists at all.
		Satisfied: func(any) (bool, error) { return false, nil },
		Converge: func(params any) error {
			t := params.(miseToolchains)

			prevNode := currentMiseNodeVersion()
			var globals []string
			if prevNode != "" {
				globals = npmGlobals()
			}

			useArgs := append([]string{"use", "-g"}, t.args()...)
			_ = exec.Command("mise", useArgs...).Run()
			if err := exec.Command("mise", "install").Run(); err != nil {
				fmt.Fprintln(os.Stderr, "warning: some toolchains may not be available")
			}

			if len(globals) == 0 {
				return nil
			}
			newNode := currentMiseNodeVersion()
			if newNode == prevNode {
				return nil
			}

			fmt.Printf("Node upgraded: %s -> %s\n", prevNode, newNode)
			fmt.Println("The following global npm packages were present in the old version:")
			for _, pkg := range globals {
				fmt.Println("  -", pkg)
			}
			fmt.Println("Note: any version pins or custom configuration for these packages will NOT be migrated.")
			if !confirm(fmt.Sprintf("Reinstall these packages into node@%s?", newNode)) {
				return nil
			}

			nodeBin, err := exec.LookPath("node")
			if err != nil {
				return nil
			}
			npmBin := nodeBin[:strings.LastIndex(nodeBin, "/")+1] + "npm"
			for _, pkg := range globals {
				fmt.Println("Installing", pkg, "...")
				if err := exec.Command(npmBin, "install", "-g", pkg).Run(); err != nil {
					fmt.Println("  x", pkg, "(failed — install manually if needed)")
				} else {
					fmt.Println("  ✓", pkg)
				}
			}
			return nil
		},
	})
}

// Mise ports modules/module_mise.sh: converge mise-managed toolchain
// versions, gated on toolchains.enable_mise, then (if node's version
// actually changed as a result) offer to reinstall previously-global npm
// packages into the new node version — the CommandGuarded escape hatch's
// first real, substantial use, since neither "converge toolchains" nor
// "conditionally offer an interactive migration" maps onto any of the
// typed kinds.
//
// Why mise over asdf/nvm/pyenv/rbenv/etc: mise (jdx.dev) is a single
// Rust binary that replaces the whole "one version manager per
// language, each with its own shell hook" pattern — that pattern is
// exactly what this investigation found costing real login-shell time
// (nvm's hook alone measured ~850-900ms; see the dedup modules). A
// single tool with one shell hook is a direct fix for the redundant-
// tooling class devboost exists partly to clean up, not just a
// preference. mise is also the actively-maintained, faster-growing
// successor in this space (asdf's own plugin ecosystem is largely
// shell-script based and slower; nvm predates and doesn't share mise's
// activate-once model).
//
// Per-toolchain default versions — flagged for the periodic
// adversarial re-review (issue #9), not asserted as settled:
//   - node: "lts" and rust: "stable", deno: "lts" are genuinely safe,
//     low-maintenance defaults — "track the vendor's own stability
//     channel" is uncontroversial and doesn't go stale on its own.
//   - python: "3.14" and go: "1.26" are specific pinned versions, not
//     rolling channels — these WILL go stale as new versions ship and
//     need periodic bumping; there's no equivalent to Node's clearly-
//     defined LTS channel for either. Treat these two as needing
//     active upkeep, not "set once and correct forever."
//   - Checked 2026-08-08 (docs/tool-choice-review-2026-08.md): python
//     3.14 is still the current stable line, no action needed. go 1.26
//     is not yet stale (the "1.26" channel auto-resolves to the
//     current patched 1.26.5), but Go 1.27's release notes are already
//     published — GA is imminent. Bump this pin to "1.27" once that
//     release actually ships; don't guess a version number ahead of
//     release.
func Mise(cfg *config.Config) []engine.Resource {
	if cfg.Get("toolchains.enable_mise", "true") != "true" {
		return nil
	}
	if _, err := exec.LookPath("mise"); err != nil {
		return nil
	}

	t := miseToolchains{
		node:      cfg.Get("toolchains.globals.node", "lts"),
		python:    cfg.Get("toolchains.globals.python", "3.14"),
		goVersion: cfg.Get("toolchains.globals.go", "1.26"),
		rust:      cfg.Get("toolchains.globals.rust", "stable"),
		deno:      cfg.Get("toolchains.globals.deno", "lts"),
	}
	return []engine.Resource{
		{
			ID:   "mise_toolchains",
			Kind: kinds.CommandGuarded{ID: "mise_toolchains_converged", Params: t, Wants: "configure mise toolchains"},
		},
	}
}
