package modules

import (
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestDirenvDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "direnv:\n  enable: false\n")
	if got := Direnv(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

func TestDirenvDefaultContent(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Direnv(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	f, ok := got[0].Kind.(kinds.File)
	if !ok {
		t.Fatalf("expected a kinds.File resource, got %T", got[0].Kind)
	}
	if !strings.Contains(f.Content, "use_mise") {
		t.Fatalf("expected default content to define use_mise, got %q", f.Content)
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
