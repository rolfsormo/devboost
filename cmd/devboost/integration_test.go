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

	run := func(args ...string) (string, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "HOME="+home)
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
