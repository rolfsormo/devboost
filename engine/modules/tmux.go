package modules

import (
	"fmt"
	"os"
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
	var loggingPlugin string
	if cfg.Get("tmux.plugins.logging.enable", "false") == "true" {
		loggingPlugin = "set -g @plugin 'tmux-plugins/tmux-logging'\n"
	}
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
%sset -g @continuum-restore '%s'
set -g @resurrect-capture-pane-contents '%s'
run '%s/tpm'`,
		cfg.Get("tmux.settings.base_index", "1"),
		cfg.Get("tmux.settings.pane_base_index", "1"),
		onOff(cfg, "tmux.settings.mouse", "true"),
		cfg.Get("tmux.settings.history_limit", "50000"),
		cfg.Get("tmux.settings.escape_time", "0"),
		onOff(cfg, "tmux.settings.focus_events", "true"),
		loggingPlugin,
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
			// TPM's own scripts shell out to `git clone` internally for
			// each plugin. Without disabling credential helpers, a
			// successful anonymous HTTPS clone still triggers
			// `git-credential-osxkeychain store` afterwards, which blocks
			// on a Keychain GUI prompt that never comes in a headless
			// apply run — confirmed by direct process-tree inspection of
			// a real hung `devboost apply` stuck inside TPM's
			// install_plugins.sh. GIT_CONFIG_COUNT/KEY/VALUE is git's
			// documented way to inject config into a subprocess we don't
			// control the invocation of, equivalent to `git -c
			// credential.helper=` but via environment instead of args.
			noCredHelperEnv := append(os.Environ(),
				"GIT_CONFIG_COUNT=1",
				"GIT_CONFIG_KEY_0=credential.helper",
				"GIT_CONFIG_VALUE_0=",
			)
			install := exec.Command(tpmPath + "/bindings/install_plugins")
			install.Env = noCredHelperEnv
			_ = install.Run()
			update := exec.Command(tpmPath+"/bindings/update_plugins", "all")
			update.Env = noCredHelperEnv
			_ = update.Run()
			return nil
		},
	})
}

// Tmux ports modules/module_tmux.sh: clone TPM, upsert the devboost tmux
// config block, and install/update plugins (gated on
// system.auto_install_plugins), gated overall on tmux.enable.
//
// Why TPM plus this specific plugin set: TPM (tmux-plugins/tpm) is the
// de facto standard tmux plugin manager — there isn't a serious
// competing option with comparable adoption. tmux-resurrect +
// tmux-continuum (session persistence across restarts/reboots) and
// tmux-yank (system-clipboard integration) are consistently the
// highest-adoption plugins in shared tmux configs, installed
// unconditionally. tmux-logging has 3-4x less adoption than yank (per
// live GitHub star counts checked 2026-08-08) and is absent from most
// "essential tmux setup" roundups — not consensus-backed enough to
// bundle by default, so it's gated behind
// tmux.plugins.logging.enable (default false) instead. See
// docs/tool-choice-review-2026-08.md for the review this came from.
//
// escape-time = 0 (tmux.settings.escape_time default) is the one
// setting here with unambiguous, widely-documented justification: tmux
// defaults to 500ms so it can distinguish a literal Escape keypress
// from the start of an escape-sequence-based key (arrow keys, Alt
// combos, etc.), but that delay is directly perceptible as lag when
// using vi-mode or any Alt-key bindings in a terminal, and setting it
// to 0 is close to universal advice in tmux configuration guides. The
// rest of the settings (mouse, history-limit, base-index,
// pane-base-index, focus-events) are common quality-of-life defaults
// without a single canonical source behind the exact values chosen.
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
