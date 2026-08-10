package modules

import (
	"testing"

	"github.com/rolfsormo/devboost/engine/kinds"
)

func TestLinuxVendorInstallsReturnsNilOnNonGapOS(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	if got := LinuxVendorInstalls(cfg, kinds.OSDarwin); got != nil {
		t.Fatalf("expected nil on macOS (no gap there), got %v", got)
	}
	if got := LinuxVendorInstalls(cfg, kinds.OSLinuxArch); got != nil {
		t.Fatalf("expected nil on Arch (trusted, no known gap), got %v", got)
	}
}

func TestLinuxVendorInstallsCoversAllUbuntuGaps(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := LinuxVendorInstalls(cfg, kinds.OSLinuxUbuntu)
	ids := make(map[string]bool)
	for _, r := range got {
		ids[r.ID] = true
	}
	// Every one of these was confirmed missing from Ubuntu 24.04's apt
	// (apt-cache show <pkg>) via a real container — including procs,
	// which a first attempt at this fix incorrectly omitted from
	// aptGapPackages (it was checked against Fedora's dnf, which DOES
	// have it, and that result was mistakenly assumed to apply to apt
	// too) — caught only by a real end-to-end apply run failing on it.
	for _, want := range []string{
		"vendor_install_mise", "vendor_install_atuin", "vendor_install_starship",
		"vendor_install_dust", "github_release_install_lazygit", "github_release_install_procs",
	} {
		if !ids[want] {
			t.Fatalf("expected resource %s for Ubuntu's apt gap, got %v", want, ids)
		}
	}
}

func TestLinuxVendorInstallsCoversAllFedoraGaps(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := LinuxVendorInstalls(cfg, kinds.OSLinuxFedora)
	ids := make(map[string]bool)
	for _, r := range got {
		ids[r.ID] = true
	}
	for _, want := range []string{
		"vendor_install_mise", "vendor_install_starship", "vendor_install_dust",
		"github_release_install_lazygit",
	} {
		if !ids[want] {
			t.Fatalf("expected resource %s for Fedora's dnf gap, got %v", want, ids)
		}
	}
	// atuin and procs ARE in dnf on Fedora — must NOT get vendor install resources.
	if ids["vendor_install_atuin"] {
		t.Fatal("atuin is available via dnf on Fedora — should not get a vendor install resource")
	}
	if ids["github_release_install_procs"] {
		t.Fatal("procs is available via dnf on Fedora — should not get a vendor install resource")
	}
}

func TestLazygitAssetNameForMatchesRealReleaseFilenames(t *testing.T) {
	// Confirmed against the actual jesseduffield/lazygit v0.64.0 release
	// assets: lazygit_0.64.0_linux_x86_64.tar.gz /
	// lazygit_0.64.0_linux_arm64.tar.gz — lowercase "linux", version
	// without the leading "v".
	kind := lazygitReleaseInstall()
	name, archive := kind.AssetNameFor("v0.64.0", "linux", "amd64")
	if name != "lazygit_0.64.0_linux_x86_64.tar.gz" {
		t.Fatalf("got %q", name)
	}
	if archive != kinds.ArchiveTarGz {
		t.Fatalf("expected tar.gz archive kind, got %v", archive)
	}

	nameArm, _ := kind.AssetNameFor("v0.64.0", "linux", "arm64")
	if nameArm != "lazygit_0.64.0_linux_arm64.tar.gz" {
		t.Fatalf("got %q", nameArm)
	}
}

func TestProcsAssetNameForMatchesRealReleaseFilenames(t *testing.T) {
	// Confirmed against the actual dalance/procs v0.14.12 release
	// assets: procs-v0.14.12-x86_64-linux.zip /
	// procs-v0.14.12-aarch64-linux.zip — version KEEPS the leading "v"
	// (unlike lazygit).
	kind := procsReleaseInstall()
	name, archive := kind.AssetNameFor("v0.14.12", "linux", "amd64")
	if name != "procs-v0.14.12-x86_64-linux.zip" {
		t.Fatalf("got %q", name)
	}
	if archive != kinds.ArchiveZip {
		t.Fatalf("expected zip archive kind, got %v", archive)
	}

	nameArm, _ := kind.AssetNameFor("v0.14.12", "linux", "arm64")
	if nameArm != "procs-v0.14.12-aarch64-linux.zip" {
		t.Fatalf("got %q", nameArm)
	}
}

// TestLinuxVendorInstallsForceManagedBinDir is a regression test for a
// real idempotency bug found via a second real Ubuntu container apply
// run: atuin, starship, and dust each have their own DIFFERENT default
// install location if not told otherwise (atuin: ~/.atuin/bin;
// starship/dust: /usr/local/bin), so a first apply run appeared to
// succeed but every subsequent apply re-downloaded and re-installed all
// three, since BinaryAvailable/ResolveBinary only check PATH and
// kinds.ManagedBinDir. Each installer's env must explicitly redirect to
// ManagedBinDir.
func TestLinuxVendorInstallsForceManagedBinDir(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := LinuxVendorInstalls(cfg, kinds.OSLinuxUbuntu)

	binDir, err := kinds.ManagedBinDir()
	if err != nil {
		t.Fatalf("ManagedBinDir: %v", err)
	}

	for _, r := range got {
		v, ok := r.Kind.(kinds.VendorInstall)
		if !ok {
			continue // GitHubReleaseInstall (lazygit, procs) extracts directly, no installer env to check
		}
		var wantKey string
		switch v.BinaryName {
		case "atuin":
			wantKey = "ATUIN_INSTALL_DIR"
		case "starship":
			wantKey = "BIN_DIR"
		case "dust":
			wantKey = "DUST_INSTALL"
		default:
			continue // mise sets MISE_INSTALL_PATH (a full path, not a bare dir) — checked separately below
		}
		if got := v.Env[wantKey]; got != binDir {
			t.Fatalf("%s: expected %s=%q, got %q", v.BinaryName, wantKey, binDir, got)
		}
	}
}

func TestLinuxVendorInstallsMiseInstallPathIsInManagedBinDir(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := LinuxVendorInstalls(cfg, kinds.OSLinuxUbuntu)
	binDir, err := kinds.ManagedBinDir()
	if err != nil {
		t.Fatalf("ManagedBinDir: %v", err)
	}
	for _, r := range got {
		v, ok := r.Kind.(kinds.VendorInstall)
		if !ok || v.BinaryName != "mise" {
			continue
		}
		want := binDir + "/mise"
		if v.Env["MISE_INSTALL_PATH"] != want {
			t.Fatalf("expected MISE_INSTALL_PATH=%q, got %q", want, v.Env["MISE_INSTALL_PATH"])
		}
		return
	}
	t.Fatal("expected a mise vendor install resource")
}
