package engine

import (
	"strings"
	"testing"
)

type fakeKind struct {
	pending bool
	desc    string
	ran     *[]string
	id      string
}

func (f fakeKind) Diff() (*PendingOp, error) {
	if !f.pending {
		return nil, nil
	}
	ran := f.ran
	id := f.id
	return &PendingOp{
		Description: f.desc,
		Execute: func() error {
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

	_, err := DiffAndExecute([]Resource{b, a}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 1 || ran[0] != "a" {
		t.Fatalf("expected a to have executed, ran = %v", ran)
	}
}

type dynamicKind struct {
	diff func() (*PendingOp, error)
}

func (d dynamicKind) Diff() (*PendingOp, error) { return d.diff() }
