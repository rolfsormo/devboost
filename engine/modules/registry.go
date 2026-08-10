// Package modules holds the Go-engine ports of devboost's bash modules.
// This package coexists with the original bash modules/*.sh tree during
// the v2 migration; it does not replace or modify any bash file.
package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Module is what every module in this package presents to the CLI:
// a display Name (used to group doctor output — the tool-first grouping
// the architecture doc settled on, so 50 dedup checks across 10 tools
// still reads as 10 lines, not 50), a Resources function computing
// desired state from config and the detected OS, and an optional
// Diagnostics function for read-only findings (nil if the module has
// none).
type Module struct {
	Name        string
	Resources   func(cfg *config.Config, os kinds.OS) []engine.Resource
	Diagnostics func(cfg *config.Config, os kinds.OS) engine.DiagnosticFunc
}

// All is every module, in the same dependency-respecting order the bash
// tool's build.sh registration list encoded by hand (pkg first, since
// other modules assume tools are already installed; zsh after znap,
// since zsh's rendered config sources znap). Under the new engine this
// ordering is advisory/documentation only — the real ordering guarantee
// is each Resource's own DependsOn, not registration order — but keeping
// registration in a sensible order still matters for doctor's grouped
// output to read naturally top to bottom.
var All = []Module{
	{Name: "pkg", Resources: Pkg},
	{Name: "linux_vendor_installs", Resources: LinuxVendorInstalls},
	{Name: "zsh", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Zsh(cfg) }},
	{Name: "znap", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Znap(cfg) }},
	{Name: "starship", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Starship(cfg) }},
	{Name: "tmux", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Tmux(cfg) }},
	{Name: "mise", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Mise(cfg) }},
	{Name: "corepack", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Corepack(cfg) }},
	{Name: "direnv", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Direnv(cfg) }},
	{Name: "git", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return Git(cfg) }},
	{Name: "services", Resources: Services},
	{
		Name:        "security",
		Resources:   func(cfg *config.Config, os kinds.OS) []engine.Resource { return Security(cfg) },
		Diagnostics: func(cfg *config.Config, os kinds.OS) engine.DiagnosticFunc { return SecurityDiagnostics(cfg) },
	},
	{Name: "zinit", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return ZinitZnapDedup(cfg) }},
	{Name: "asdf", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return AsdfMiseDedup(cfg) }},
	{Name: "nvm", Resources: func(cfg *config.Config, os kinds.OS) []engine.Resource { return NvmMiseDedup(cfg) }},
}

// AllResources computes every module's desired-state resources for the
// given config/OS, in registration order — the list ComputeDiff/Apply
// operate on. Resource IDs must be unique across the whole set (topoSort
// already enforces this), so module authors need to keep their IDs
// distinctly namespaced, same discipline the bash tool's flat function
// names already required.
func AllResources(cfg *config.Config, os kinds.OS) []engine.Resource {
	var all []engine.Resource
	for _, m := range All {
		all = append(all, m.Resources(cfg, os)...)
	}
	return all
}
