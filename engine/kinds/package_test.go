package kinds

import "testing"

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
