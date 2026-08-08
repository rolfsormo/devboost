package modules

import (
	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

const starshipConfigContent = `add_newline = false
command_timeout = 700

[character]
success_symbol = "[❯](bold green)"
error_symbol   = "[❯](bold red)"

[directory]
truncation_length = 3
style = "bold blue"

[git_branch]
symbol = ""
style = "bold yellow"

[git_status]
style = "bold red"
format = '([\[$all_status\]]($style))'

[nodejs]
symbol = ""
style = "green"

[python]
symbol = ""
style = "yellow"

[rust]
symbol = ""
style = "red"

[package]
disabled = true
`

// Starship ports modules/module_starship.sh: write the devboost-managed
// starship prompt config, gated on prompt.enable_starship. The content is
// entirely static in the bash version too (db_render_starship_config
// takes no config-driven branches), so this is a plain File resource.
//
// Why starship: it's the dominant cross-shell prompt choice in the
// current ecosystem (single Rust binary, no shell-specific plugin
// framework required), which matters directly for devboost's
// znap-based zsh setup — a prompt that doesn't depend on oh-my-zsh or a
// zsh-specific framework avoids yet another redundant-tooling class
// like the ones the dedup modules exist to clean up.
//
// Per-setting confidence varies — not every value here has strong
// external consensus behind it, and this doc comment says so honestly
// rather than post-hoc justifying every number:
//   - add_newline = false and command_timeout = 700 are common,
//     reasonable choices in shared starship configs, but not something
//     with a single documented "best practice" behind them — treat as
//     "sensible default," not "researched consensus."
//   - truncation_length = 3 is similarly a matter of taste; other
//     popular configs use anywhere from 2 to full-path.
//   - Per-language symbols/styles and [package] disabled = true are
//     mostly about visual noise reduction and match common community
//     presets, but again no single canonical source to point to.
// If you're revisiting this file, it's a reasonable candidate for the
// periodic adversarial re-review described in issue #9 — there's more
// room for taste-based disagreement here than in, say, delta's config.
func Starship(cfg *config.Config) []engine.Resource {
	if cfg.Get("prompt.enable_starship", "true") != "true" {
		return nil
	}
	path := cfg.Get("prompt.starship_config", "~/.config/starship.toml")
	return []engine.Resource{
		{
			ID:   "starship_config",
			Kind: kinds.File{Path: path, Content: starshipConfigContent},
		},
	}
}
