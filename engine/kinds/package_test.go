package kinds

import (
	"fmt"
	"strings"
	"testing"
)

type fakeProvider struct {
	failNames map[string]bool
	attempted []string
}

func (f *fakeProvider) installed(name string) (bool, error) { return false, nil }
func (f *fakeProvider) install(name string) error {
	f.attempted = append(f.attempted, name)
	if f.failNames[name] {
		return fmt.Errorf("simulated failure for %s", name)
	}
	return nil
}

func TestInstallAllAttemptsEveryPackageDespiteFailures(t *testing.T) {
	p := &fakeProvider{failNames: map[string]bool{"lazygit": true}}
	names := []string{"zsh", "lazygit", "ripgrep"}

	err := installAll(p, names)

	if len(p.attempted) != 3 {
		t.Fatalf("expected all 3 packages attempted despite lazygit failing, got %v", p.attempted)
	}
	if err == nil {
		t.Fatal("expected an aggregated error since lazygit failed")
	}
	if !strings.Contains(err.Error(), "lazygit") {
		t.Fatalf("expected error to mention the failed package, got %v", err)
	}
}

func TestInstallAllReportsAllFailures(t *testing.T) {
	p := &fakeProvider{failNames: map[string]bool{"lazygit": true, "eza": true}}
	names := []string{"zsh", "lazygit", "eza", "ripgrep"}

	err := installAll(p, names)

	if len(p.attempted) != 4 {
		t.Fatalf("expected all 4 packages attempted, got %v", p.attempted)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "lazygit") || !strings.Contains(err.Error(), "eza") {
		t.Fatalf("expected error to mention both failed packages, got %v", err)
	}
	if !strings.Contains(err.Error(), "2 package(s)") {
		t.Fatalf("expected error to report the failure count, got %v", err)
	}
}

func TestInstallAllNoErrorWhenAllSucceed(t *testing.T) {
	p := &fakeProvider{}
	if err := installAll(p, []string{"zsh", "ripgrep"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMapPackageNameAppliesOverride(t *testing.T) {
	if got := mapPackageName(OSLinuxUbuntu, "fd"); got != "fd-find" {
		t.Fatalf("got %q, want fd-find", got)
	}
	if got := mapPackageName(OSLinuxFedora, "fd"); got != "fd-find" {
		t.Fatalf("got %q, want fd-find", got)
	}
}

func TestMapPackageNameIdentityByDefault(t *testing.T) {
	for _, os := range []OS{OSDarwin, OSLinuxArch} {
		if got := mapPackageName(os, "fd"); got != "fd" {
			t.Fatalf("%s: got %q, want fd (no override on this OS)", os, got)
		}
	}
	if got := mapPackageName(OSLinuxUbuntu, "zsh"); got != "zsh" {
		t.Fatalf("got %q, want zsh (no override for this package)", got)
	}
}
