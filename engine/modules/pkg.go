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
// over cat, eza over ls, zoxide over cd, dust/duf/procs over
// du/df/ps — the "modern CLI replacement" trend is well-documented and
// these specific tools are consistently the named examples in that
// space, not an arbitrary pick. jq/yq (JSON/YAML querying), fzf
// (fuzzy-finder, itself a dependency of a lot of shell tooling
// including atuin's search UX), lazygit (terminal git UI), and
// git-delta/mise/atuin/starship/tmux are each covered by their own
// module's rationale where the choice is more specific than "popular
// modern CLI tool." zsh itself is here because devboost's shell tooling
// (znap, starship prompt integration, the .zshrc.devboost render) is
// zsh-specific — see zsh.go.
func Pkg(cfg *config.Config) []engine.Resource {
	names := cfg.GetList("packages.base")
	if len(names) == 0 {
		names = defaultBasePackages
	}
	return []engine.Resource{
		{ID: "base_packages", Kind: kinds.Package{Names: names}},
	}
}
