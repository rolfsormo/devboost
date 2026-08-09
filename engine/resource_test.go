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
