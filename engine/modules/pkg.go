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
func Pkg(cfg *config.Config) []engine.Resource {
	names := cfg.GetList("packages.base")
	if len(names) == 0 {
		names = defaultBasePackages
	}
	return []engine.Resource{
		{ID: "base_packages", Kind: kinds.Package{Names: names}},
	}
}
