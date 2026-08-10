package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rolfsormo/devboost/engine"
)

// TestZshAndSecurityBothTargetZshrcDevboost is a load-bearing test for a
// real risk found while wiring the full registry: zsh's zshrc_devboost
// resource is a File (full-content overwrite) and security's
// security_check_alias resource is a BlockInFile (append/replace a
// marked block) — both targeting the SAME path
// (~/.zshrc.devboost/zsh.include_file). If security's block gets applied
// before zsh's File resource runs, zsh's Execute would silently
// overwrite the whole file and wipe out security's block. This test
// applies both resources together, in registry order, and asserts the
// final file has both zsh's rendered content AND security's block —
// catching the corruption if dependency ordering (or ordering by
// coincidence) ever breaks it.
func TestZshAndSecurityBothTargetZshrcDevboost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	// Registry order: zsh registers before security (see registry.go),
	// so exercise exactly that order here.
	resources := append(Zsh(cfg), Security(cfg)...)

	if _, err := engine.DiffAndExecute(resources, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc.devboost"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "znap.zsh") {
		t.Fatalf("expected zsh's rendered content to survive, got %q", got)
	}
	if !strings.Contains(got, "devboost-check()") {
		t.Fatalf("expected security's block to survive alongside zsh's content — "+
			"if this fails, zsh's File resource overwrote security's BlockInFile "+
			"(or vice versa) rather than the two composing safely. Got: %q", got)
	}
}

// TestSecurityThenZshDoesNotLoseSecurityBlock is the reverse order —
// confirms the outcome doesn't depend on which of the two happens to run
// first, since nothing currently declares an explicit DependsOn between
// them.
func TestSecurityThenZshDoesNotLoseSecurityBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := loadFixtureConfig(t, "")

	resources := append(Security(cfg), Zsh(cfg)...)

	if _, err := engine.DiffAndExecute(resources, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".zshrc.devboost"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "znap.zsh") || !strings.Contains(got, "devboost-check()") {
		t.Fatalf("expected both zsh's content and security's block to survive regardless of order, got %q", got)
	}
}
