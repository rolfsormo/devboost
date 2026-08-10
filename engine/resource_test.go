package engine

import (
	"errors"
	"strings"
	"testing"
)

type fakeKind struct {
	pending bool
	desc    string
	ran     *[]string
	id      string
	failErr error // if set, Execute returns this instead of succeeding
}

func (f fakeKind) Diff() (*PendingOp, error) {
	if !f.pending {
		return nil, nil
	}
	ran := f.ran
	id := f.id
	failErr := f.failErr
	return &PendingOp{
		Description: f.desc,
		Execute: func() error {
			if failErr != nil {
				return failErr
			}
			*ran = append(*ran, id)
			return nil
		},
	}, nil
}

func TestTopoSortOrdersDependenciesFirst(t *testing.T) {
	resources := []Resource{
		{ID: "c", Kind: fakeKind{}, DependsOn: []string{"b"}},
		{ID: "a", Kind: fakeKind{}},
		{ID: "b", Kind: fakeKind{}, DependsOn: []string{"a"}},
	}
	ordered, err := topoSort(resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var ids []string
	for _, r := range ordered {
		ids = append(ids, r.ID)
	}
	want := "a b c"
	if got := strings.Join(ids, " "); got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestTopoSortDetectsCycle(t *testing.T) {
	resources := []Resource{
		{ID: "a", Kind: fakeKind{}, DependsOn: []string{"b"}},
		{ID: "b", Kind: fakeKind{}, DependsOn: []string{"a"}},
	}
	if _, err := topoSort(resources); err == nil {
		t.Fatal("expected a cycle error, got nil")
	}
}

func TestTopoSortDetectsUnknownDependency(t *testing.T) {
	resources := []Resource{
		{ID: "a", Kind: fakeKind{}, DependsOn: []string{"missing"}},
	}
	if _, err := topoSort(resources); err == nil {
		t.Fatal("expected an unknown-dependency error, got nil")
	}
}

func TestTopoSortDetectsDuplicateID(t *testing.T) {
	resources := []Resource{
		{ID: "a", Kind: fakeKind{}},
		{ID: "a", Kind: fakeKind{}},
	}
	if _, err := topoSort(resources); err == nil {
		t.Fatal("expected a duplicate-ID error, got nil")
	}
}

// TestDiffAndExecuteSeesDependencyEffects is the load-bearing test for the
// whole reason Apply can't reuse ComputeDiff's batch diff: resource b's
// Diff must observe the real effect of resource a's Execute, not a
// pre-execution snapshot.
func TestDiffAndExecuteSeesDependencyEffects(t *testing.T) {
	var ran []string

	a := Resource{
		ID: "a",
		Kind: fakeKind{
			pending: true,
			desc:    "converge a",
			ran:     &ran,
			id:      "a",
		},
	}
	// b's Diff asserts that a has already executed by the time it's
	// called — simulating "b depends on state a's Execute establishes".
	b := Resource{
		ID:        "b",
		DependsOn: []string{"a"},
		Kind: dynamicKind{
			diff: func() (*PendingOp, error) {
				if len(ran) == 0 {
					t.Fatal("b was diffed before a executed")
				}
				return nil, nil
			},
		},
	}

	result, err := DiffAndExecute([]Resource{b, a}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 1 || ran[0] != "a" {
		t.Fatalf("expected a to have executed, ran = %v", ran)
	}
	if len(result.Failed) != 0 || len(result.Skipped) != 0 {
		t.Fatalf("expected no failures/skips, got %+v", result)
	}
}

// TestDiffAndExecuteContinuesPastUnrelatedFailure is the regression test
// for the real bug this behavior fixes: found via a live Ubuntu
// container run where several packages weren't in the default apt
// repos, aborting a resource's Execute previously stopped EVERY other
// resource from converging, even ones with no relationship to the
// failure. A failed resource must not block independent resources.
func TestDiffAndExecuteContinuesPastUnrelatedFailure(t *testing.T) {
	var ran []string
	failing := Resource{
		ID:   "failing",
		Kind: fakeKind{pending: true, desc: "fails", ran: &ran, id: "failing", failErr: errors.New("boom")},
	}
	independent := Resource{
		ID:   "independent",
		Kind: fakeKind{pending: true, desc: "converge independent", ran: &ran, id: "independent"},
	}

	result, err := DiffAndExecute([]Resource{failing, independent}, nil)
	if err != nil {
		t.Fatalf("unexpected top-level error (failures should be reported in the result, not returned): %v", err)
	}
	if len(ran) != 1 || ran[0] != "independent" {
		t.Fatalf("expected the independent resource to still execute, ran = %v", ran)
	}
	if len(result.Applied) != 1 || result.Applied[0].ResourceID != "independent" {
		t.Fatalf("expected independent in Applied, got %+v", result.Applied)
	}
	if _, ok := result.Failed["failing"]; !ok {
		t.Fatalf("expected failing to be recorded in Failed, got %+v", result.Failed)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("expected no skips (nothing depends on the failure), got %+v", result.Skipped)
	}
}

// TestDiffAndExecuteSkipsTransitiveDependents verifies that a resource
// depending (even transitively) on a failed one is skipped rather than
// diffed/executed against state the failed resource never reached —
// while a sibling with no such dependency still converges normally.
func TestDiffAndExecuteSkipsTransitiveDependents(t *testing.T) {
	var ran []string
	base := Resource{
		ID:   "base",
		Kind: fakeKind{pending: true, desc: "fails", ran: &ran, id: "base", failErr: errors.New("boom")},
	}
	directDependent := Resource{
		ID:        "direct",
		DependsOn: []string{"base"},
		Kind:      fakeKind{pending: true, desc: "direct", ran: &ran, id: "direct"},
	}
	transitiveDependent := Resource{
		ID:        "transitive",
		DependsOn: []string{"direct"},
		Kind:      fakeKind{pending: true, desc: "transitive", ran: &ran, id: "transitive"},
	}
	unrelated := Resource{
		ID:   "unrelated",
		Kind: fakeKind{pending: true, desc: "unrelated", ran: &ran, id: "unrelated"},
	}

	result, err := DiffAndExecute([]Resource{base, directDependent, transitiveDependent, unrelated}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 1 || ran[0] != "unrelated" {
		t.Fatalf("expected only unrelated to execute, ran = %v", ran)
	}
	if _, ok := result.Failed["base"]; !ok {
		t.Fatalf("expected base to be recorded as failed, got %+v", result.Failed)
	}
	if blocker, ok := result.Skipped["direct"]; !ok || blocker != "base" {
		t.Fatalf("expected direct skipped due to base, got %+v", result.Skipped)
	}
	if blocker, ok := result.Skipped["transitive"]; !ok || blocker != "base" {
		t.Fatalf("expected transitive skipped due to base (root cause, not just its immediate parent), got %+v", result.Skipped)
	}
	if len(result.Applied) != 1 || result.Applied[0].ResourceID != "unrelated" {
		t.Fatalf("expected only unrelated in Applied, got %+v", result.Applied)
	}
}

type dynamicKind struct {
	diff func() (*PendingOp, error)
}

func (d dynamicKind) Diff() (*PendingOp, error) { return d.diff() }

// TestNeedsProviderResolvesToConcreteProvider is the core scenario this
// mechanism exists for: a resource declares NeedsProvider: "mise"
// without knowing (or caring) which concrete resource ID actually
// installs mise on this platform — that's resolved by the engine, not
// hardcoded by the declaring module.
func TestNeedsProviderResolvesToConcreteProvider(t *testing.T) {
	var ran []string
	provider := Resource{
		ID:       "vendor_install_mise",
		Kind:     fakeKind{pending: true, desc: "install mise", ran: &ran, id: "vendor_install_mise"},
		Provides: []string{"mise"},
	}
	consumer := Resource{
		ID:            "mise_toolchains",
		NeedsProvider: []string{"mise"},
		Kind: dynamicKind{
			diff: func() (*PendingOp, error) {
				if len(ran) == 0 {
					t.Fatal("mise_toolchains was diffed before its provider (vendor_install_mise) executed")
				}
				return nil, nil
			},
		},
	}

	_, err := DiffAndExecute([]Resource{consumer, provider}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 1 || ran[0] != "vendor_install_mise" {
		t.Fatalf("expected the provider to have executed, ran = %v", ran)
	}
}

// TestNeedsProviderDifferentConcreteProviderPerPlatform proves the
// actual point: two DIFFERENT resources can each Provide "mise" across
// two separate resource lists (simulating two platforms), and the same
// consumer resolves to whichever one is actually present — the
// consuming module never names either concrete ID.
func TestNeedsProviderDifferentConcreteProviderPerPlatform(t *testing.T) {
	makeConsumer := func(ran *[]string) Resource {
		return Resource{
			ID:            "mise_toolchains",
			NeedsProvider: []string{"mise"},
			Kind:          fakeKind{pending: true, desc: "configure mise", ran: ran, id: "mise_toolchains"},
		}
	}

	t.Run("linux platform: vendor_install_mise provides it", func(t *testing.T) {
		var ran []string
		provider := Resource{ID: "vendor_install_mise", Kind: fakeKind{pending: true, desc: "x", ran: &ran, id: "vendor_install_mise"}, Provides: []string{"mise"}}
		result, err := DiffAndExecute([]Resource{makeConsumer(&ran), provider}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Applied) != 2 {
			t.Fatalf("expected both resources to apply, got %+v", result)
		}
	})

	t.Run("macOS platform: base_packages provides it instead", func(t *testing.T) {
		var ran []string
		provider := Resource{ID: "base_packages", Kind: fakeKind{pending: true, desc: "x", ran: &ran, id: "base_packages"}, Provides: []string{"mise"}}
		result, err := DiffAndExecute([]Resource{makeConsumer(&ran), provider}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Applied) != 2 {
			t.Fatalf("expected both resources to apply, got %+v", result)
		}
	})
}

func TestNeedsProviderErrorsWhenNoProviderExists(t *testing.T) {
	consumer := Resource{ID: "mise_toolchains", NeedsProvider: []string{"mise"}, Kind: fakeKind{}}
	if _, err := DiffAndExecute([]Resource{consumer}, nil); err == nil {
		t.Fatal("expected an error when no resource Provides the required capability")
	}
}

func TestNeedsProviderErrorsWhenProviderIsAmbiguous(t *testing.T) {
	a := Resource{ID: "a", Provides: []string{"mise"}, Kind: fakeKind{}}
	b := Resource{ID: "b", Provides: []string{"mise"}, Kind: fakeKind{}}
	consumer := Resource{ID: "consumer", NeedsProvider: []string{"mise"}, Kind: fakeKind{}}
	if _, err := DiffAndExecute([]Resource{a, b, consumer}, nil); err == nil {
		t.Fatal("expected an error when two resources both Provide the same capability")
	}
}

// TestNeedsProviderSkipsConsumerWhenProviderFails proves the
// partial-failure mechanism (ExecutionResult.Skipped) correctly follows
// NeedsProvider-resolved edges, not just literal DependsOn — a
// NeedsProvider dependency is a real dependency for skip-propagation
// purposes too.
func TestNeedsProviderSkipsConsumerWhenProviderFails(t *testing.T) {
	var ran []string
	failingProvider := Resource{
		ID:       "vendor_install_mise",
		Kind:     fakeKind{pending: true, desc: "fails", ran: &ran, id: "vendor_install_mise", failErr: errors.New("boom")},
		Provides: []string{"mise"},
	}
	consumer := Resource{
		ID:            "mise_toolchains",
		NeedsProvider: []string{"mise"},
		Kind:          fakeKind{pending: true, desc: "configure mise", ran: &ran, id: "mise_toolchains"},
	}

	result, err := DiffAndExecute([]Resource{failingProvider, consumer}, nil)
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if _, ok := result.Failed["vendor_install_mise"]; !ok {
		t.Fatalf("expected vendor_install_mise in Failed, got %+v", result.Failed)
	}
	if blocker, ok := result.Skipped["mise_toolchains"]; !ok || blocker != "vendor_install_mise" {
		t.Fatalf("expected mise_toolchains skipped due to vendor_install_mise, got %+v", result.Skipped)
	}
	if len(ran) != 0 {
		t.Fatalf("expected mise_toolchains to never execute, ran = %v", ran)
	}
}

func TestComputeDiffResolvesNeedsProvider(t *testing.T) {
	provider := Resource{ID: "vendor_install_mise", Kind: fakeKind{}, Provides: []string{"mise"}}
	consumer := Resource{ID: "mise_toolchains", NeedsProvider: []string{"mise"}, Kind: fakeKind{}}
	// Just needs topoSort (called internally by ComputeDiff) to resolve
	// without error — Plan and Doctor both go through ComputeDiff, not
	// DiffAndExecute, so NeedsProvider must resolve there too, not only
	// in the execute path.
	if _, err := ComputeDiff([]Resource{consumer, provider}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComputeDiffErrorsOnUnresolvedNeedsProvider(t *testing.T) {
	consumer := Resource{ID: "mise_toolchains", NeedsProvider: []string{"mise"}, Kind: fakeKind{}}
	if _, err := ComputeDiff([]Resource{consumer}); err == nil {
		t.Fatal("expected an error when Plan/Doctor's ComputeDiff hits an unresolvable NeedsProvider")
	}
}

func TestNeedsProviderCombinesWithDependsOn(t *testing.T) {
	var ran []string
	miseInstall := Resource{ID: "vendor_install_mise", Kind: fakeKind{pending: true, desc: "x", ran: &ran, id: "vendor_install_mise"}, Provides: []string{"mise"}}
	miseToolchains := Resource{
		ID:            "mise_toolchains",
		NeedsProvider: []string{"mise"},
		Kind: dynamicKind{diff: func() (*PendingOp, error) {
			if len(ran) < 1 {
				t.Fatal("mise_toolchains diffed before its provider ran")
			}
			return nil, nil
		}},
	}
	corepack := Resource{
		ID:        "corepack",
		DependsOn: []string{"mise_toolchains"}, // plain DependsOn, alongside the other resource's NeedsProvider
		Kind: dynamicKind{diff: func() (*PendingOp, error) {
			return nil, nil // mise_toolchains has no PendingOp (nil diff) in this test, so nothing to check ordering against beyond "no panic"
		}},
	}

	if _, err := DiffAndExecute([]Resource{corepack, miseToolchains, miseInstall}, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
