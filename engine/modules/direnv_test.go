package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestDirenvDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "direnv:\n  enable: false\n")
	if got := Direnv(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestDirenvNoDefaultContent(t *testing.T) {
	// mise activate zsh (see zshdevboost.go) now owns toolchain
	// activation globally, so there's nothing left for a devboost-managed
	// .direnvrc to contain by default — see the Direnv doc comment.
	cfg := loadFixtureConfig(t, "")
	if got := Direnv(cfg); len(got) != 0 {
		t.Fatalf("expected no resources with default config, got %v", got)
	}
}

func TestDirenvCustomContent(t *testing.T) {
	cfg := loadFixtureConfig(t, "direnv:\n  content: \"custom content\"\n")
	got := Direnv(cfg)
	f, ok := got[0].Kind.(kinds.File)
	if !ok {
		t.Fatalf("expected a kinds.File resource, got %T", got[0].Kind)
	}
	if f.Content != "custom content" {
		t.Fatalf("got %q, want %q", f.Content, "custom content")
	}
}
