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
