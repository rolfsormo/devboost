package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSandboxedApplyPlanDoctorIdempotent builds the real binary, points
// HOME at a fresh temp directory, and runs plan/apply/doctor for real —
// this touches real brew/git, same as a genuine `devboost apply` would,
// since the point is verifying the actual end-to-end binary works, not
// just the in-process resource graph TestAllResourcesResolveWithDefaultConfig
// already covers. Skipped outside macOS/Linux since package installation
// behavior is platform-dependent.
func TestSandboxedApplyPlanDoctorIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow end-to-end integration test in -short mode")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("skipping on %s — package installation behavior is platform-dependent", runtime.GOOS)
	}

	bin := buildDevboost(t)
	home := t.TempDir()

	writeFile(t, filepath.Join(home, ".devboost.yaml"), "version: \"1.0.0\"\n")

	// Overriding only HOME does NOT fully sandbox this test — two
	// separate real gaps, both confirmed by direct investigation of a
	// real hang in this test on a real dev machine:
	//
	//  1. XDG-aware tools (confirmed for real — mise) read
	//     $XDG_CONFIG_HOME directly, inherited unset from os.Environ()
	//     otherwise.
	//  2. mise's own config discovery walks UP the process's working
	//     directory tree looking for config files — not HOME or
	//     XDG_CONFIG_HOME at all for this part. Since the test binary's
	//     subprocess inherits cmd.Dir unset (= the repo checkout's own
	//     working directory, itself a real subdirectory of the real
	//     $HOME), mise found and tried to load the developer's actual
	//     ~/.config/mise/config.toml regardless of every env var
	//     override above — confirmed directly: `env -i HOME=/tmp/fake
	//     mise config`, run from inside this repo checkout, still
	//     resolved the real global config; the same command run from
	//     /tmp instead found nothing. mise then refused to proceed
	//     because that real config file isn't "trusted" from this
	//     process's perspective (a real mise security feature) — which
	//     is what actually produced the observed hang, not an infinite
	//     loop in devboost's own code.
	//
	// Fixed by setting cmd.Dir to the sandboxed home for every
	// subprocess call below, so mise's upward directory walk starts (and
	// stays) inside the sandbox — matching what a genuinely fresh
	// machine's working directory tree looks like, not this repo's own
	// checkout location.
	env := append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
	)

	run := func(args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = env
		cmd.Dir = home
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	if _, err := run("plan"); err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if _, err := run("apply", "--dry-run"); err != nil {
		t.Fatalf("apply --dry-run failed: %v", err)
	}
	if _, err := run("doctor"); err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	if _, err := run("apply"); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".zshrc.devboost")); err != nil {
		t.Fatalf("expected .zshrc.devboost created in temp home: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); err != nil {
		t.Fatalf("expected .zshrc created in temp home: %v", err)
	}

	realHome, err := os.UserHomeDir()
	if err == nil && realHome != home {
		if _, err := os.Stat(filepath.Join(realHome, ".zshrc.devboost.__devboost_test_marker_should_not_exist")); err == nil {
			t.Fatal("unexpectedly found a test marker in the real home directory")
		}
	}

	firstZshrc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := run("apply"); err != nil {
		t.Fatalf("second apply failed: %v", err)
	}

	secondZshrc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstZshrc) != string(secondZshrc) {
		t.Fatalf("expected second apply to be idempotent (no .zshrc changes), got a diff:\nfirst:\n%s\nsecond:\n%s", firstZshrc, secondZshrc)
	}

	// undo right after a clean apply must proceed (regression test for
	// the bug where a naive whole-system pending-diff pre-check always
	// refused, because always-rerun resources like mise/tmux/corepack
	// never show zero pending diffs even when nothing has drifted).
	if out, err := run("undo", "--dry-run"); err != nil {
		t.Fatalf("undo --dry-run failed: %v\n%s", err, out)
	} else if !strings.Contains(out, "Would:") && !strings.Contains(out, "Nothing to undo") {
		t.Fatalf("expected undo --dry-run to either preview a restore or report nothing to undo, got:\n%s", out)
	}
}

// TestSandboxedUndoRefusesOnDrift builds its own sandboxed home with a
// fake pre-existing oh-my-zsh installation, applies (which genuinely
// migrates away from it), then simulates the user recreating
// ~/.oh-my-zsh afterward — undo must refuse without --force and proceed
// with it. Kept separate from TestSandboxedApplyPlanDoctorIdempotent so
// the oh-my-zsh setup here can't perturb that test's own .zshrc-content
// assertions, which assume no oh-my-zsh is present.
func TestSandboxedUndoRefusesOnDrift(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow end-to-end integration test in -short mode")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("skipping on %s — package installation behavior is platform-dependent", runtime.GOOS)
	}

	bin := buildDevboost(t)
	home := t.TempDir()

	writeFile(t, filepath.Join(home, ".devboost.yaml"), "version: \"1.0.0\"\n")
	omzDir := filepath.Join(home, ".oh-my-zsh")
	if err := os.MkdirAll(omzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(omzDir, "oh-my-zsh.sh"), "# fake oh-my-zsh\n")
	writeFile(t, filepath.Join(home, ".zshrc"),
		"export ZSH=\"$HOME/.oh-my-zsh\"\nsource $ZSH/oh-my-zsh.sh\nexport MY_VAR=\"hello\"\n")

	env := append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
	)
	run := func(args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = env
		cmd.Dir = home
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	if out, err := run("apply"); err != nil {
		t.Fatalf("apply failed: %v\n%s", err, out)
	}
	if _, err := os.Stat(omzDir); !os.IsNotExist(err) {
		t.Fatalf("expected apply to have migrated ~/.oh-my-zsh away, stat err: %v", err)
	}

	// Simulate the user recreating ~/.oh-my-zsh after the migration
	// already ran — real drift, since it no longer matches what undo's
	// own backups describe.
	if err := os.MkdirAll(omzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(omzDir, "oh-my-zsh.sh"), "# reappeared after migration\n")

	out, err := run("undo", "--dry-run")
	if err != nil {
		t.Fatalf("undo --dry-run (drifted) failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Refusing to undo") {
		t.Fatalf("expected undo to refuse when omz_migration has drifted, got:\n%s", out)
	}

	out, err = run("undo", "--dry-run", "--force")
	if err != nil {
		t.Fatalf("undo --dry-run --force failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "Refusing to undo") {
		t.Fatalf("expected --force to bypass the drift refusal, got:\n%s", out)
	}
}

func buildDevboost(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "devboost")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build devboost: %v\n%s", err, out)
	}
	return bin
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
