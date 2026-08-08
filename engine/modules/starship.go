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
// rather than post-hoc justifying every number. Verified against
// starship's own current defaults during the 2026-08-08 adversarial
// review (see docs/tool-choice-review-2026-08.md):
//   - truncation_length = 3 matches starship's own upstream default
//     exactly — this one IS the researched-consensus case, not just a
//     reasonable pick.
//   - command_timeout = 700 is a deliberate +200ms deviation from
//     starship's own default of 500 — not wrong, but a real deviation,
//     not a community-standard number. No external justification found
//     for the specific value; treat as "we chose to be more lenient,"
//     not "this is what everyone uses."
//   - add_newline = false overrides starship's own default of true.
//     Genuinely a taste call — no consensus found either way in shared
//     configs.
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
