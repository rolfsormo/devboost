package kinds

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/rolfsormo/devboost/engine"
)

// packageNameOverrides ports _db_pkg_map: almost every package name is
// identical across package managers, so only the exceptions are listed
// (fd is packaged as fd-find on Debian/Ubuntu and Fedora) rather than the
// bash version's full per-OS case statement repeating every identity
// mapping. Anything not listed here maps to itself.
var packageNameOverrides = map[OS]map[string]string{
	OSLinuxUbuntu: {"fd": "fd-find"},
	OSLinuxFedora: {"fd": "fd-find"},
}

func mapPackageName(os OS, name string) string {
	if overrides, ok := packageNameOverrides[os]; ok {
		if mapped, ok := overrides[name]; ok {
			return mapped
		}
	}
	return name
}

// packageProvider is the per-OS backend behind Package: how to check
// whether a package is already installed, and how to install it. This is
// the "provider" abstraction from the architecture doc — brew, apt, dnf,
// pacman are swappable backends behind one resource kind, the same way
// Terraform's AWS/GCP providers are swappable behind one resource type.
type packageProvider interface {
	installed(name string) (bool, error)
	install(name string) error
}

func providerFor(os OS) (packageProvider, error) {
	switch os {
	case OSDarwin:
		return brewProvider{}, nil
	case OSLinuxUbuntu:
		return aptProvider{}, nil
	case OSLinuxFedora:
		return dnfProvider{}, nil
	case OSLinuxArch:
		return pacmanProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported OS: %s", os)
	}
}

type brewProvider struct{}

func (brewProvider) installed(name string) (bool, error) {
	if err := ensureBrewInstalled(); err != nil {
		return false, err
	}
	err := exec.Command("brew", "list", name).Run()
	return err == nil, nil
}

func (brewProvider) install(name string) error {
	out, err := exec.Command("brew", "install", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("brew install %s: %w\n%s", name, err, out)
	}
	return nil
}

// ensureBrewInstalled ports db_install_packages' darwin branch: if brew
// itself isn't present, bootstrap it via Homebrew's own official
// installer before anything else can proceed.
func ensureBrewInstalled() error {
	if err := exec.Command("brew", "--version").Run(); err == nil {
		return nil
	}
	cmd := exec.Command("/bin/bash", "-c",
		`$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install Homebrew: %w\n%s", err, out)
	}
	return nil
}

type aptProvider struct{}

func (aptProvider) installed(name string) (bool, error) {
	out, err := exec.Command("dpkg", "-l").Output()
	if err != nil {
		return false, fmt.Errorf("dpkg -l: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "ii" && fields[1] == name {
			return true, nil
		}
	}
	return false, nil
}

var aptUpdateOnce sync.Once
var aptUpdateErr error

func (aptProvider) install(name string) error {
	// apt-get update runs once per process, immediately before the first
	// real install — not on every installed() check — matching the bash
	// tool's behavior of updating the index once per apply run.
	aptUpdateOnce.Do(func() {
		out, err := exec.Command("sudo", "apt-get", "update", "-qq").CombinedOutput()
		if err != nil {
			aptUpdateErr = fmt.Errorf("apt-get update: %w\n%s", err, out)
		}
	})
	if aptUpdateErr != nil {
		return aptUpdateErr
	}

	out, err := exec.Command("sudo", "apt-get", "install", "-y", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt-get install %s: %w\n%s", name, err, out)
	}
	return nil
}

type dnfProvider struct{}

func (dnfProvider) installed(name string) (bool, error) {
	err := exec.Command("rpm", "-q", name).Run()
	return err == nil, nil
}

func (dnfProvider) install(name string) error {
	out, err := exec.Command("sudo", "dnf", "install", "-y", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dnf install %s: %w\n%s", name, err, out)
	}
	return nil
}

type pacmanProvider struct{}

func (pacmanProvider) installed(name string) (bool, error) {
	err := exec.Command("pacman", "-Qi", name).Run()
	return err == nil, nil
}

func (pacmanProvider) install(name string) error {
	out, err := exec.Command("sudo", "pacman", "-S", "--noconfirm", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -S %s: %w\n%s", name, err, out)
	}
	return nil
}

// Package declares "these packages should be installed," mapped through
// the OS's own package-name conventions (fd -> fd-find on Debian/Fedora,
// etc.) and installed via the OS's own package manager. OS defaults to
// DetectOS() if unset — set it explicitly only in tests.
type Package struct {
	Names []string
	OS    OS
}

func (p Package) targetOS() OS {
	if p.OS == "" {
		return DetectOS()
	}
	return p.OS
}

func (p Package) Diff() (*engine.PendingOp, error) {
	os := p.targetOS()
	provider, err := providerFor(os)
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, name := range p.Names {
		mapped := mapPackageName(os, name)
		ok, err := provider.installed(mapped)
		if err != nil {
			return nil, fmt.Errorf("check %s installed: %w", mapped, err)
		}
		if !ok {
			missing = append(missing, mapped)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}

	return &engine.PendingOp{
		Description: fmt.Sprintf("install packages: %s", strings.Join(missing, " ")),
		Execute: func() error {
			return installAll(provider, missing)
		},
	}, nil
}

// installAll attempts every package in names rather than stopping at the
// first failure: a single package unavailable on this distro's repos
// (e.g. lazygit isn't in Ubuntu's default apt repos — confirmed
// installing on a real Ubuntu container) shouldn't block every other
// package, or every resource that comes after this one in apply's
// dependency order, from converging. Aggregates failures into one error
// so the caller learns about all of them, not just the first.
func installAll(provider packageProvider, names []string) error {
	var failed, errs []string
	for _, name := range names {
		if err := provider.install(name); err != nil {
			failed = append(failed, name)
			errs = append(errs, err.Error())
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install %d package(s): %s\n%s",
			len(failed), strings.Join(failed, ", "), strings.Join(errs, "\n"))
	}
	return nil
}
