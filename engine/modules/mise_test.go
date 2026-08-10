package modules

import "testing"

func TestMiseDisabledWhenMiseConfigDisabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "toolchains:\n  enable_mise: false\n")
	if got := Mise(cfg); len(got) != 0 {
		t.Fatalf("expected no resources when disabled, got %v", got)
	}
}

// TestMiseAlwaysDeclaresResourceWhenEnabled is a regression test for a
// real bug found via a live Ubuntu container run: Mise() used to gate
// itself off entirely with exec.LookPath("mise") at the moment this
// resource list was built — meaning if a DIFFERENT resource (see
// linuxvendor.go) installs the mise binary earlier in the SAME apply
// run, Mise()'s own toolchain-converge step would never even be
// declared, let alone run, because the check happened before any
// resource had executed. Mise() must always declare its resource when
// enabled; ordering is now handled structurally via NeedsProvider (see
// TestMiseDependsOnMiseProvider below), not by gating construction on
// current PATH state.
func TestMiseAlwaysDeclaresResourceWhenEnabled(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Mise(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource regardless of whether mise is currently on PATH, got %d", len(got))
	}
}

// TestMiseDependsOnMiseProvider is a regression test for the second,
// deeper fix: mise_toolchains previously had NO real dependency at all
// on whatever resource actually installs mise — it worked only by
// accident of module registration order. NeedsProvider: []string{"mise"}
// makes this a real, engine-enforced dependency instead, resolved to
// whichever concrete resource Provides "mise" on a given platform
// (pkg.go's base_packages, or linuxvendor.go's vendor_install_mise) —
// this module deliberately does NOT know or care which one that is.
func TestMiseDependsOnMiseProvider(t *testing.T) {
	cfg := loadFixtureConfig(t, "")
	got := Mise(cfg)
	if len(got) != 1 {
		t.Fatalf("expected exactly one resource, got %d", len(got))
	}
	needs := got[0].NeedsProvider
	if len(needs) != 1 || needs[0] != "mise" {
		t.Fatalf("expected mise_toolchains to NeedsProvider [\"mise\"], got %v", needs)
	}
}

func TestMiseToolchainsArgs(t *testing.T) {
	m := miseToolchains{node: "lts", python: "3.14", goVersion: "1.26", rust: "stable", deno: "lts"}
	args := m.args()
	want := []string{"node@lts", "python@3.14", "go@1.26", "rust@stable", "deno@lts"}
	if len(args) != len(want) {
		t.Fatalf("got %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("got %v, want %v", args, want)
		}
	}
}

func TestParseNpmGlobalsOutputExcludesNpmAndCorepack(t *testing.T) {
	out := "/opt/node/lib/node_modules\n" +
		"/opt/node/lib/node_modules/npm\n" +
		"/opt/node/lib/node_modules/corepack\n" +
		"/opt/node/lib/node_modules/typescript\n" +
		"/opt/node/lib/node_modules/eslint\n"
	got := parseNpmGlobalsOutput(out)
	want := []string{"typescript", "eslint"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestParseNpmGlobalsOutputEmpty(t *testing.T) {
	if got := parseNpmGlobalsOutput(""); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}
