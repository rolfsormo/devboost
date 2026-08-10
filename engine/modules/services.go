package modules

import (
	"os/exec"
	"strings"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

func init() {
	kinds.RegisterCommand("atuin_brew_service_running", kinds.GuardedCommand{
		Satisfied: func(any) (bool, error) {
			// Checked fresh here, not once when Services()'s resource
			// list was built — atuin might be installed by a different
			// resource earlier in the same apply run (e.g. Pkg's brew
			// install). A module-construction-time-only availability
			// check would silently never see that; the same class of
			// bug found via a real Ubuntu container run and fixed the
			// same way in mise.go — see kinds.BinaryAvailable's doc
			// comment for the fuller reasoning.
			if available, err := kinds.BinaryAvailable("atuin"); err != nil {
				return false, err
			} else if !available {
				return true, nil // nothing to converge yet — not a failure, atuin's own install may just not have run yet
			}
			out, err := exec.Command("brew", "services", "list").Output()
			if err != nil {
				// brew not present/working — nothing this resource can
				// converge; matches the bash version's "skip if brew
				// missing" behavior.
				return true, nil
			}
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == "atuin" && fields[1] == "started" {
					return true, nil
				}
			}
			return false, nil
		},
		Converge: func(any) error {
			return exec.Command("brew", "services", "start", "atuin").Run()
		},
	})
}

// Services ports modules/module_services.sh, darwin only: start the
// atuin brew service if it isn't already running, gated on
// zsh.history.use_atuin. The bash version's Linux branch is purely
// informational (no actual action — it just logs a suggestion to check
// systemd) with nothing to converge, so it declares no resource here;
// that informational note belongs to whatever doctor/info-only
// diagnostic mechanism lands with the security module (task #14), not
// forced into a resource with nothing to do.
//
// Why atuin: it's the leading modern shell-history tool — SQLite-backed
// searchable history with sync support, a significant step up from zsh's
// built-in history file, and the config this module manages (see
// zshdevboost.go's renderAtuinConfig) is what makes atuin actually take
// over shell history search rather than just being installed inert.
// Running it as a persistent brew service (vs. invoking it per-shell)
// is what atuin's own daemon mode is for — matches atuin's documented
// setup for its optional background sync/daemon behavior. The
// Darwin-only-action / Linux-informational split isn't a value
// judgment on Linux, it's because brew services is the actual
// convergence primitive here (see git.go/pkg.go for the broader
// "shell out to the tool's own CLI when it's the best primitive" rule)
// and Linux init systems vary enough (systemd vs. others) that there's
// no single equivalent primitive to shell out to yet.
func Services(cfg *config.Config, os kinds.OS) []engine.Resource {
	if cfg.Get("zsh.history.use_atuin", "true") != "true" {
		return nil
	}
	if os != kinds.OSDarwin {
		return nil
	}
	// Not gated on exec.LookPath("atuin") here — that check now lives in
	// the registered command's Satisfied (see the init() above), checked
	// fresh at Diff time rather than once when this resource list is
	// built, so it correctly sees atuin if Pkg's brew install (same
	// apply run) installs it first.
	return []engine.Resource{
		{
			ID:   "atuin_service",
			Kind: kinds.CommandGuarded{ID: "atuin_brew_service_running", Wants: "atuin service started via brew"},
		},
	}
}
