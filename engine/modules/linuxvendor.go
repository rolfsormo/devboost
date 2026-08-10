package modules

import (
	"path/filepath"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

// aptGapPackages and dnfGapPackages list packages devboost's default
// base list names (see pkg.go) that are NOT available through that
// distro's own package manager — confirmed by directly querying real
// Ubuntu 24.04 (apt-cache show) and Fedora (dnf info) containers, not
// assumed. Pkg excludes these from the apt-get/dnf-managed Package
// resource on the matching OS; LinuxVendorInstalls (below) converges
// them via each tool's own official install method instead.
//
// Arch's official repos + AUR are well known to package all of these —
// not independently re-verified here (no arm64 Arch Docker image was
// available to test against), so pacman is trusted rather than
// confirmed the same way apt/dnf were. If that trust turns out to be
// wrong for a given package, it'll surface the same way the apt/dnf
// gaps did — a real failed install on a real machine — and should be
// added here once confirmed, not guessed at preemptively.
var aptGapPackages = map[string]bool{
	"lazygit": true, "mise": true, "atuin": true, "starship": true, "dust": true, "procs": true,
}

var dnfGapPackages = map[string]bool{
	"lazygit": true, "mise": true, "starship": true, "dust": true,
}

// linuxGapFor returns the gap map for os, or nil for any OS with no
// known gap (macOS/Homebrew has all of these; Arch is trusted, see the
// doc comment above) — a nil map is safe to range/index in Pkg, since a
// nil map read always returns the zero value (false).
func linuxGapFor(os kinds.OS) map[string]bool {
	switch os {
	case kinds.OSLinuxUbuntu:
		return aptGapPackages
	case kinds.OSLinuxFedora:
		return dnfGapPackages
	default:
		return nil
	}
}

// LinuxVendorInstalls converges the tools listed in aptGapPackages/
// dnfGapPackages via each tool's own official install method — a
// VendorInstall (fetch-and-run the vendor's own installer script) for
// tools that ship one, GitHubReleaseInstall (download+extract the
// latest release binary) for the two that don't (lazygit, procs — see
// each kind's doc comment). All install into kinds.ManagedBinDir()
// (~/.local/bin), never touching devboost-managed shell config
// themselves — several of these tools' own installers do write shell
// rc files unconditionally if used naively (atuin's setup.atuin.sh
// wrapper is the confirmed example), which is why this module invokes
// the lower-level, non-mutating installer/env-var combination for each
// rather than the tool's most commonly copy-pasted one-liner. PATH
// exposure for ManagedBinDir is handled once, centrally, in
// zshdevboost.go — not per-tool here.
//
// Only returns resources for packages actually missing from this OS's
// package manager — a distro where apt/dnf/pacman already has
// everything (Arch, trusted per the doc comment above) gets no
// resources from this module at all.
func LinuxVendorInstalls(cfg *config.Config, os kinds.OS) []engine.Resource {
	gap := linuxGapFor(os)
	if gap == nil {
		return nil
	}

	// Every installer below needs to be told explicitly to install into
	// ManagedBinDir — several have their OWN different default location
	// (atuin: ~/.atuin/bin; starship: /usr/local/bin, needing sudo
	// unless redirected; dust: /usr/local/bin if writable, else
	// ~/.local/bin). Without forcing this, BinaryAvailable/ResolveBinary
	// (which only check PATH and ManagedBinDir) never find them there,
	// so every apply re-downloads and re-installs them — a real
	// idempotency bug found via a second real Ubuntu container apply run
	// (the first apply succeeds; only a follow-up run exposes that
	// several tools silently landed somewhere devboost doesn't check).
	//
	// If ManagedBinDir itself can't be resolved (os.UserHomeDir failing
	// — would mean $HOME is broken, breaking most of devboost anyway),
	// binDir stays empty and each installer just falls back to its own
	// tool-specific default location rather than corrupting its env with
	// an empty path; BinaryAvailable's PATH-only check still works for
	// anything that lands there.
	binDir, _ := kinds.ManagedBinDir()

	// withBinDir returns base plus key=binDir, but only when binDir is
	// actually known — an empty-string env var override could behave
	// differently (and worse) than simply leaving that variable unset,
	// which is what each tool's own installer already falls back to
	// safely.
	withBinDir := func(key string, base map[string]string) map[string]string {
		if binDir == "" {
			return base
		}
		out := make(map[string]string, len(base)+1)
		for k, v := range base {
			out[k] = v
		}
		out[key] = binDir
		return out
	}

	var resources []engine.Resource
	if gap["mise"] {
		env := map[string]string{"MISE_INSTALL_HELP": "0"}
		if binDir != "" {
			// mise.run's own default (MISE_INSTALL_PATH unset) is
			// already $HOME/.local/bin/mise — matches ManagedBinDir
			// exactly, set explicitly anyway so this doesn't silently
			// depend on that staying true.
			env["MISE_INSTALL_PATH"] = filepath.Join(binDir, "mise")
		}
		resources = append(resources, engine.Resource{
			ID:       "vendor_install_mise",
			Kind:     kinds.VendorInstall{BinaryName: "mise", ScriptURL: "https://mise.run", Env: env},
			Provides: []string{"mise"},
		})
	}
	if gap["atuin"] {
		resources = append(resources, engine.Resource{
			ID: "vendor_install_atuin",
			Kind: kinds.VendorInstall{
				BinaryName: "atuin",
				// The binary-only installer, not setup.atuin.sh — the
				// latter unconditionally appends init lines to .zshrc,
				// which devboost renders itself (see zshdevboost.go).
				ScriptURL: "https://github.com/atuinsh/atuin/releases/latest/download/atuin-installer.sh",
				// ATUIN_INSTALL_DIR: default is ~/.atuin/bin — redirect to ManagedBinDir.
				Env: withBinDir("ATUIN_INSTALL_DIR", map[string]string{"ATUIN_NO_MODIFY_PATH": "1"}),
			},
			Provides: []string{"atuin"},
		})
	}
	if gap["starship"] {
		resources = append(resources, engine.Resource{
			ID: "vendor_install_starship",
			Kind: kinds.VendorInstall{
				BinaryName: "starship",
				ScriptURL:  "https://starship.rs/install.sh",
				// BIN_DIR: default is /usr/local/bin, needs sudo unless redirected.
				Env: withBinDir("BIN_DIR", map[string]string{"FORCE": "true"}),
			},
			Provides: []string{"starship"},
		})
	}
	if gap["dust"] {
		resources = append(resources, engine.Resource{
			ID: "vendor_install_dust",
			Kind: kinds.VendorInstall{
				BinaryName: "dust",
				ScriptURL:  "https://raw.githubusercontent.com/bootandy/dust/master/install.sh",
				// DUST_INSTALL: default is /usr/local/bin if writable,
				// else ~/.local/bin — force consistency either way.
				Env: withBinDir("DUST_INSTALL", nil),
			},
			Provides: []string{"dust"},
		})
	}
	if gap["lazygit"] {
		resources = append(resources, engine.Resource{
			ID:       "github_release_install_lazygit",
			Kind:     lazygitReleaseInstall(),
			Provides: []string{"lazygit"},
		})
	}
	if gap["procs"] {
		resources = append(resources, engine.Resource{
			ID:       "github_release_install_procs",
			Kind:     procsReleaseInstall(),
			Provides: []string{"procs"},
		})
	}
	return resources
}

// lazygitReleaseInstall and procsReleaseInstall encode each project's
// own release-asset naming convention (verified via their GitHub
// releases directly — see linuxvendor_test.go for the exact strings
// confirmed) — kept as small named functions rather than inlined
// closures so the naming logic is independently testable.
func lazygitReleaseInstall() kinds.GitHubReleaseInstall {
	return kinds.GitHubReleaseInstall{
		BinaryName: "lazygit",
		Repo:       "jesseduffield/lazygit",
		AssetNameFor: func(version, goos, goarch string) (string, kinds.ArchiveKind) {
			v := versionWithoutV(version)
			arch := lazygitArch(goarch)
			return "lazygit_" + v + "_" + goos + "_" + arch + ".tar.gz", kinds.ArchiveTarGz
		},
	}
}

func procsReleaseInstall() kinds.GitHubReleaseInstall {
	return kinds.GitHubReleaseInstall{
		BinaryName: "procs",
		Repo:       "dalance/procs",
		AssetNameFor: func(version, goos, goarch string) (string, kinds.ArchiveKind) {
			arch := procsArch(goarch)
			return "procs-" + version + "-" + arch + "-" + goos + ".zip", kinds.ArchiveZip
		},
	}
}

func versionWithoutV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}

// lazygitArch maps Go's GOARCH to lazygit's release-asset arch naming
// (confirmed against real release asset filenames).
func lazygitArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		return goarch
	}
}

// procsArch maps Go's GOARCH to procs' release-asset arch naming
// (confirmed against real release asset filenames).
func procsArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return goarch
	}
}
