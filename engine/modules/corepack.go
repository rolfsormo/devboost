package modules

import (
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
		// being idempotent. Faithfully ported: Satisfied is only about
		// whether corepack exists at all (if not, there's nothing to
		// converge — matches the bash version's "skip if missing" path,
		// not an error).
		Satisfied: func(any) (bool, error) {
			_, err := exec.LookPath("corepack")
			return err != nil, nil // corepack absent -> "satisfied" (nothing to do)
		},
		Converge: func(any) error {
			return exec.Command("corepack", "enable").Run()
		},
	})
}

// Corepack ports modules/module_corepack.sh: enable pnpm/yarn shims via
// corepack, gated on toolchains.enable_mise (corepack ships bundled with
// Node.js, which mise manages).
//
// This default needs a caveat, not just a justification: Node's TSC
// voted to stop bundling corepack starting with Node 25 (it was
// experimental even while bundled). Node 24 LTS still ships it; Node
// 26 — becoming the next LTS around October 2026 — does not
// (confirmed via Node's own v25.8.0 docs and TSC decision reporting).
// Corepack itself isn't being deprecated as a project (still actively
// maintained standalone), just unbundled — `npm install -g corepack`
// remains the fix. Because Satisfied() above treats "binary not found"
// as "nothing to do" rather than surfacing a warning, this module will
// silently stop providing pnpm/yarn shims for anyone on Node 25+ once
// that becomes the common case. That's a real gap, not a hypothetical
// one — see the follow-up to add a fallback install or an explicit
// warning rather than silent success.
func Corepack(cfg *config.Config) []engine.Resource {
	if cfg.Get("toolchains.enable_mise", "true") != "true" {
		return nil
	}
	return []engine.Resource{
		{
			ID:   "corepack",
			Kind: kinds.CommandGuarded{ID: "corepack_enabled", Wants: "corepack pnpm/yarn shims enabled"},
		},
	}
}
