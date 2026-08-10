package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Git ports modules/module_git.sh: configure delta as git's pager/diff
// filter, gated on git.delta.enable. All four settings use --global
// scope, matching the bash version (this module never touches --local).
//
// Why delta: it's the clear community-favorite modern git diff pager —
// 31.7k GitHub stars vs. diff-so-fancy's 18.1k (dandavison/delta vs.
// so-fancy/diff-so-fancy, checked live), actively developed, and
// comparison writeups consistently describe it as strictly more
// capable (syntax highlighting, word-level diffs, side-by-side mode)
// rather than a close call.
//
// navigate=true is delta's own suggested first-run config (straight
// from its README's "Get Started" example) — first-party endorsed, not
// just a popular third-party choice. line-numbers=true is documented
// separately as an optional customization; it's common in dotfiles
// writeups but one rung below "official recommendation," included here
// because it's genuinely useful and low-risk, not because delta itself
// leads with it.
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
