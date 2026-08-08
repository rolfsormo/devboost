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
