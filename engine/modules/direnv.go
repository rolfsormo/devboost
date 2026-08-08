package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// Direnv ports modules/module_direnv.sh's .direnvrc management, gated on
// direnv.enable, but — as of the switch documented below — writes nothing
// by default. direnv.content lets a user still opt into managed content
// (e.g. their own .direnvrc helpers) if they want devboost to own the
// file.
//
// Why direnv at all: it's the established, general-purpose tool for
// per-directory environment loading (15k+ GitHub stars, actively
// maintained) — broader in scope than asdf's own .tool-versions
// auto-activation, which only handles tool versions, not arbitrary env
// vars. No competing tool has comparable mindshare here, and devboost
// still installs it by default (see pkg.go) for exactly this purpose.
// direnv and mise are not an either/or choice: a project with a plain
// .envrc (arbitrary env vars, no mise involvement) is completely
// unaffected by mise running — mise's own docs state plainly "mise will
// not interfere with direnv" for that case.
//
// Why this module no longer writes a use_mise .direnvrc helper: that
// helper existed so a project's .envrc could opt in per-directory via
// `use mise`. mise's own docs (mise.jdx.dev/direnv.html) call that
// pattern deprecated, state "the official stance is you should not use
// direnv with mise" for it, and say direnv-compatibility bugs in it
// won't be fixed upstream. mise's current recommended replacement —
// `mise activate zsh`, run globally and unconditionally in
// .zshrc.devboost whenever toolchains.enable_mise is set (see
// zshdevboost.go) — already handles per-project toolchain switching via
// mise's own directory-change hook, with no direnv involvement needed.
// Once that global hook covers toolchains, a devboost-authored
// .direnvrc has nothing useful left to contain, so this module stops
// writing one rather than shipping dead/discouraged content. Confirmed
// via research, not just inferred: mise's docs describe this same split
// (global mise activate owns toolchains; direnv, if present, owns only
// unrelated env vars) as the currently-supported combination, not a
// deprecated one — see engine/modules/mise.go for the mise side.
func Direnv(cfg *config.Config) []engine.Resource {
	if cfg.Get("direnv.enable", "true") != "true" {
		return nil
	}
	content := cfg.Get("direnv.content", "")
	if content == "" {
		return nil
	}
	path := cfg.Get("direnv.rc_path", "~/.direnvrc")
	return []engine.Resource{
		{
			ID:   "direnvrc",
			Kind: kinds.File{Path: path, Content: content},
		},
	}
}
