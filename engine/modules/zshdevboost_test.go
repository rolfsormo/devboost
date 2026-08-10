package modules

import (
	"strings"
	"testing"
)

func TestRenderZshDevboostDefaults(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := renderZshDevboost(cfg)
	for _, want := range []string{
		"znap.zsh",
		"starship init zsh",
		"zsh-autosuggestions",
		"atuin init zsh",
		"fzf --zsh",
		"mise activate zsh",
		"direnv hook zsh",
		"CLICOLOR=1",
		"alias ls=",
		`PATH="$HOME/.local/bin:$PATH"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected default-enabled output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestRenderZshDevboostDisablesEachSection(t *testing.T) {
	cfg := loadFixtureConfig(t, `
prompt:
  enable_starship: false
zsh:
  history:
    use_atuin: false
  fzf:
    enable: false
  aliases:
    enable: false
toolchains:
  enable_mise: false
direnv:
  enable: false
aesthetics:
  clicolor: false
`)
	got := renderZshDevboost(cfg)
	for _, unwanted := range []string{
		"starship init zsh",
		"atuin init zsh",
		"fzf --zsh",
		"mise activate zsh",
		"direnv hook zsh",
		"CLICOLOR=1",
		"alias ls=",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("expected disabled section to be absent, found %q in:\n%s", unwanted, got)
		}
	}
	// znap/zoxide/plugins are unconditional in the bash version — always present.
	for _, want := range []string{"znap.zsh", "zoxide init zsh", "zsh-autosuggestions"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected unconditional section present, missing %q", want)
		}
	}
}

func TestRenderZshDevboostNeverCallsCompinitDirectly(t *testing.T) {
	// Regression guard for the real compinit-duplication bug fixed
	// earlier this session (module_zsh.sh used to call compinit itself
	// before sourcing znap, which redefines it as a no-op — pure wasted
	// work). The Go port must not reintroduce it.
	cfg := loadFixtureConfig(t, "")
	got := renderZshDevboost(cfg)
	if strings.Contains(got, "compinit -u") || strings.Contains(got, "autoload -Uz compinit") {
		t.Fatalf("expected no direct compinit call (znap owns it), got:\n%s", got)
	}
}

func TestRenderAtuinConfigDefault(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := renderAtuinConfig(cfg)
	if !strings.Contains(got, `filter_mode = "global"`) {
		t.Fatalf("expected global as default filter_mode (matches atuin's own upstream default), got %q", got)
	}
	if !strings.Contains(got, `filter_mode_shell_up_key_binding = "directory"`) {
		t.Fatalf("expected up-arrow recall scoped to directory by default, got %q", got)
	}
}

func TestRenderAtuinConfigCustomFilterMode(t *testing.T) {
	cfg := loadFixtureConfig(t, "zsh:\n  history:\n    atuin:\n      filter_mode: session\n      filter_mode_shell_up_key_binding: workspace\n")
	got := renderAtuinConfig(cfg)
	if !strings.Contains(got, `filter_mode = "session"`) {
		t.Fatalf("expected configured filter_mode, got %q", got)
	}
	if !strings.Contains(got, `filter_mode_shell_up_key_binding = "workspace"`) {
		t.Fatalf("expected configured filter_mode_shell_up_key_binding, got %q", got)
	}
}
