package modules

import (
	"fmt"
	"os/exec"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

const (
	tmuxStartMarker = "# >>> devboost tmux start"
	tmuxEndMarker   = "# <<< devboost tmux end"
	tpmGitURL       = "https://github.com/tmux-plugins/tpm"
)

func onOff(cfg *config.Config, key, def string) string {
	if cfg.Get(key, def) == "true" {
		return "on"
	}
	return "off"
}

func renderTmuxBlock(cfg *config.Config, tpmPath string) string {
	return fmt.Sprintf(`set -g base-index %s
setw -g pane-base-index %s
set -g mouse %s
set -g history-limit %s
set -s escape-time %s
set -g focus-events %s
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @plugin 'tmux-plugins/tmux-yank'
set -g @plugin 'tmux-plugins/tmux-logging'
set -g @continuum-restore '%s'
set -g @resurrect-capture-pane-contents '%s'
run '%s/tpm'`,
		cfg.Get("tmux.settings.base_index", "1"),
		cfg.Get("tmux.settings.pane_base_index", "1"),
		onOff(cfg, "tmux.settings.mouse", "true"),
		cfg.Get("tmux.settings.history_limit", "50000"),
		cfg.Get("tmux.settings.escape_time", "0"),
		onOff(cfg, "tmux.settings.focus_events", "true"),
		onOff(cfg, "tmux.settings.continuum_restore", "true"),
		onOff(cfg, "tmux.settings.resurrect_capture_pane_contents", "true"),
		tpmPath,
	)
}

func init() {
	kinds.RegisterCommand("tmux_plugins_installed", kinds.GuardedCommand{
		// Matches the bash version: plugin install/update is fire-and-forget
		// (both commands' own errors are swallowed there too), and it only
		// runs when system.auto_install_plugins is true — module-level
		// gating decides whether this resource exists at all, so once it
		// does exist there's no cheap way to know "are plugins already
		// installed" short of re-running install, which TPM's own script
		// already makes idempotent. Always pending is the faithful port.
		Satisfied: func(any) (bool, error) { return false, nil },
		Converge: func(params any) error {
			tpmPath := params.(string)
			_ = exec.Command(tpmPath + "/bindings/install_plugins").Run()
			_ = exec.Command(tpmPath+"/bindings/update_plugins", "all").Run()
			return nil
		},
	})
}

// Tmux ports modules/module_tmux.sh: clone TPM, upsert the devboost tmux
// config block, and install/update plugins (gated on
// system.auto_install_plugins), gated overall on tmux.enable.
func Tmux(cfg *config.Config) []engine.Resource {
	if cfg.Get("tmux.enable", "true") != "true" {
		return nil
	}

	tpmPath := cfg.Get("tmux.tpm_path", "~/.tmux/plugins/tpm")
	confFile := cfg.Get("tmux.conf_file", "~/.tmux.conf")

	resources := []engine.Resource{
		{ID: "tmux_tpm", Kind: kinds.GitClone{URL: tpmGitURL, Dest: tpmPath}},
		{
			ID: "tmux_config_block",
			Kind: kinds.BlockInFile{
				Path:        confFile,
				StartMarker: tmuxStartMarker,
				EndMarker:   tmuxEndMarker,
				Content:     renderTmuxBlock(cfg, tpmPath),
			},
			DependsOn: []string{"tmux_tpm"},
		},
	}

	if cfg.Get("system.auto_install_plugins", "true") == "true" {
		resources = append(resources, engine.Resource{
			ID:        "tmux_plugins",
			Kind:      kinds.CommandGuarded{ID: "tmux_plugins_installed", Params: tpmPath, Wants: "install/update tmux plugins via TPM"},
			DependsOn: []string{"tmux_tpm", "tmux_config_block"},
		})
	}

	return resources
}
