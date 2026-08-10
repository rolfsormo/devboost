package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

var defaultBasePackages = []string{
	"zsh", "zoxide", "fzf", "ripgrep", "fd", "bat", "eza", "jq", "yq",
	"git-delta", "lazygit", "direnv", "mise", "atuin", "starship", "tmux",
	"dust", "duf", "procs",
}

// Pkg ports modules/module_pkg.sh: install the configured base package
// list (packages.base in config, falling back to devboost's own default
// set). Per-OS package name mapping (fd -> fd-find, etc.) and per-OS
// install/already-installed logic live in the Package resource kind
// itself — this module just supplies the desired package list.
//
// Why these specific packages: each is a Rust/Go-rewrite-era CLI tool
// with substantial community adoption as a faster/friendlier
// alternative to a POSIX default — ripgrep over grep, fd over find, bat
// over cat, eza over ls, zoxide over cd, dust over du — the "modern CLI
// replacement" trend is well-documented and these specific tools are
// consistently the named examples in that space, not an arbitrary
// pick. jq/yq (JSON/YAML querying), fzf (fuzzy-finder, itself a
// dependency of a lot of shell tooling including atuin's search UX),
// lazygit (terminal git UI), and git-delta/mise/atuin/starship/tmux are
// each covered by their own module's rationale where the choice is
// more specific than "popular modern CLI tool." zsh itself is here
// because devboost's shell tooling (znap, starship prompt integration,
// the .zshrc.devboost render) is zsh-specific — see zsh.go.
//
// duf and procs verified separately during the 2026-08-08 adversarial
// review (docs/tool-choice-review-2026-08.md), with lower confidence
// than the rest of this list:
//   - duf: still the largest tool in the df-replacement niche (which
//     has very few entrants at all), but its last release was Sep
//     2025 and recent commits are dependency bumps only — maintenance
//     has visibly slowed. No better alternative exists yet; worth
//     watching, not replacing.
//   - procs: the right pick for its actual niche — non-interactive,
//     scriptable ps output — distinct from interactive process
//     monitors (btop, bottom) that dwarf it in stars but solve a
//     different problem. Its own community is real but noticeably
//     smaller than ripgrep/fd/bat's; don't read its inclusion here as
//     ripgrep-level consensus.
// Pkg excludes any package this OS's own package manager doesn't
// actually have (see linuxvendor.go's aptGapPackages/dnfGapPackages,
// confirmed by directly querying real Ubuntu/Fedora containers — apt
// and dnf both silently 404 on several of devboost's default packages
// otherwise, which used to abort the entire apply; see the Package
// kind's installAll fix). LinuxVendorInstalls converges the excluded
// packages via each tool's own official install method instead — see
// registry.go for where that gets wired in alongside this.
func Pkg(cfg *config.Config, os kinds.OS) []engine.Resource {
	names := cfg.GetList("packages.base")
	if len(names) == 0 {
		names = defaultBasePackages
	}

	gap := linuxGapFor(os)
	var apt []string
	for _, name := range names {
		if gap[name] {
			continue
		}
		apt = append(apt, name)
	}

	return []engine.Resource{
		{ID: "base_packages", Kind: kinds.Package{Names: apt}},
	}
}
