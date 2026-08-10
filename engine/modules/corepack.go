package modules

import (
	"fmt"
	"os/exec"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

func init() {
	kinds.RegisterCommand("corepack_enabled", kinds.GuardedCommand{
		// corepack itself has no direct "is enable already run" query, and
		// the bash version doesn't attempt one either — it just runs
		// `corepack enable` every apply, relying on corepack's own command
		// being idempotent. Faithfully ported for the "corepack is
		// present" case: Satisfied only ever returns true by actually
		// running Converge (it can't cheaply know without side effects
		// whether shims are already set up), same as upstream.
		//
		// Unlike the original port, "corepack absent" is no longer treated
		// as satisfied — see the Corepack doc comment below for why:
		// Node 25+ no longer bundles it, so absence is now the expected
		// case on a current toolchain, not a signal there's nothing to do.
		Satisfied: func(any) (bool, error) {
			return false, nil
		},
		Converge: func(any) error {
			// mise-installed binaries (node, npm, corepack once
			// installed) are NOT on bare PATH in a non-interactive
			// subprocess like this one — mise's shim/activation only
			// takes effect through its shell hook (a real interactive
			// shell's precmd), which this exec.Command call is not.
			// `mise exec node -- <cmd>` is mise's own documented
			// primitive for running a command against a managed
			// toolchain without that shell integration — confirmed via
			// a real Ubuntu container run where a bare `exec.LookPath`
			// approach found nothing even immediately after `mise
			// install` had genuinely succeeded in the very same process
			// tree.
			miseBin, err := kinds.ResolveBinary("mise")
			if err != nil {
				// Resolved fresh here, not via a bare "mise" command
				// name — devboost's own process PATH never includes
				// kinds.ManagedBinDir, so a literal exec.Command("mise",
				// ...) silently fails "not found" on a fresh machine
				// where mise was just installed moments earlier in this
				// same apply run. Found via a real Ubuntu container run.
				return fmt.Errorf("mise not found — its own install step may have failed earlier in this apply: %w", err)
			}
			runViaMiseNode := func(args ...string) *exec.Cmd {
				return exec.Command(miseBin, append([]string{"exec", "node", "--"}, args...)...)
			}

			if err := runViaMiseNode("corepack", "--version").Run(); err != nil {
				// Node's own TSC decision (nodejs/node#51981) explicitly
				// names this as the replacement workflow: "instead of
				// running `corepack enable` they'd run `npm install -g
				// corepack`." Matching upstream's own recommendation
				// rather than working around their decision.
				fmt.Println("corepack not found (expected on Node 25+, which no longer bundles it) — installing via npm")
				installErr := runViaMiseNode("npm", "install", "-g", "corepack").Run()
				if installErr != nil {
					return fmt.Errorf("npm install -g corepack: %w (is toolchains.enable_mise's node toolchain actually installed?)", installErr)
				}
			}
			return runViaMiseNode("corepack", "enable").Run()
		},
	})
}

// Corepack ports modules/module_corepack.sh: enable pnpm/yarn shims via
// corepack, gated on toolchains.enable_mise (corepack ships bundled with
// Node.js, which mise manages) — and, as of the fix documented here,
// installs corepack itself via npm when it's missing rather than
// silently skipping.
//
// Why this needed a real fix, not just a doc caveat: Node's TSC voted
// to stop bundling corepack starting with Node 25 (nodejs/TSC#1697,
// "Phase out later" — remove from Node 25+, keep experimental in Node
// ≤24 until that line's EOL). Node 24 LTS still ships it; Node 26,
// becoming the next LTS around October 2026, does not. The reasoning
// (read directly from nodejs/node#50963, #51981, and #52107, not just
// the announcement issues) was mostly about liability/scope — Node
// didn't want to ship a command that silently downloads and runs
// third-party software (yarn/pnpm) on first use, and npm's own team
// declined to integrate with corepack, which undercut its original
// premise of unifying package-manager installation. It was not a
// judgment that corepack itself is unnecessary or poorly maintained —
// it remains actively developed as an independent project.
//
// Critically, the removal PR's own author states the replacement
// workflow explicitly: "Corepack isn't getting deleted or abandoned...
// instead of running `corepack enable` they'd run `npm install -g
// corepack`." That is exactly what Converge above now does when
// corepack is missing — this aligns with upstream's stated intent
// rather than working around it. The one thing worth carrying from the
// security side of that discussion: corepack's own maintainer has
// called its download-integrity model (HTTPS + an optional hash pin)
// "OK but not great," so pinning a project's packageManager field with
// its hash (corepack's own supported mechanism) remains the safer path
// for what corepack fetches after this installs it — nothing this
// module does weakens that.
func Corepack(cfg *config.Config) []engine.Resource {
	if cfg.Get("toolchains.enable_mise", "true") != "true" {
		return nil
	}
	return []engine.Resource{
		{
			ID:   "corepack",
			Kind: kinds.CommandGuarded{ID: "corepack_enabled", Wants: "corepack pnpm/yarn shims enabled"},
			// corepack needs mise's Node toolchain to exist first (it
			// ships bundled with Node on versions that still bundle it,
			// or needs `npm install -g corepack` on 25+, either way
			// requiring mise's node@... to have already converged).
			// Found missing via a real Ubuntu container run, where mise
			// itself had to be installed (see linuxvendor.go) before its
			// own toolchain-converge resource could run, and corepack
			// ran before either had happened — see mise_toolchains's ID
			// in mise.go.
			DependsOn: []string{"mise_toolchains"},
		},
	}
}
