package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Git ports modules/module_git.sh: configure delta as git's pager/diff
// filter, gated on git.delta.enable. All four settings use --global
// scope, matching the bash version (this module never touches --local).
func Git(cfg *config.Config) []engine.Resource {
	if cfg.Get("git.delta.enable", "true") != "true" {
		return nil
	}
	return []engine.Resource{
		{ID: "git_delta_pager", Kind: kinds.GitConfig{Key: "core.pager", Value: "delta"}},
		{ID: "git_delta_diff_filter", Kind: kinds.GitConfig{Key: "interactive.diffFilter", Value: "delta --color-only"}},
		{ID: "git_delta_navigate", Kind: kinds.GitConfig{Key: "delta.navigate", Value: cfg.Get("git.delta.navigate", "true")}},
		{ID: "git_delta_line_numbers", Kind: kinds.GitConfig{Key: "delta.line-numbers", Value: cfg.Get("git.delta.line_numbers", "true")}},
	}
}
