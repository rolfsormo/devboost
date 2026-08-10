package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
