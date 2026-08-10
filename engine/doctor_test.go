package engine

import "testing"

func TestDoctorGroupsByModuleAndOmitsClean(t *testing.T) {
	pendingKind := fakeKind{pending: true, desc: "fix me", ran: &[]string{}, id: "a"}
	cleanKind := fakeKind{pending: false}

	modules := []string{"dirty_module", "clean_module"}
	resources := [][]Resource{
		{{ID: "a", Kind: pendingKind}},
		{{ID: "b", Kind: cleanKind}},
	}
	diagnostics := []DiagnosticFunc{nil, nil}

	reports, err := Doctor(modules, resources, diagnostics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected only the dirty module to be reported, got %d reports: %+v", len(reports), reports)
	}
	if reports[0].Name != "dirty_module" {
		t.Fatalf("got %q, want dirty_module", reports[0].Name)
	}
	if len(reports[0].Pending) != 1 {
		t.Fatalf("expected one pending op, got %v", reports[0].Pending)
	}
}

func TestDoctorIncludesDiagnosticsOnlyModules(t *testing.T) {
	cleanKind := fakeKind{pending: false}
	modules := []string{"diag_only"}
	resources := [][]Resource{{{ID: "a", Kind: cleanKind}}}
	diagnostics := []DiagnosticFunc{
		func() ([]Diagnostic, error) {
			return []Diagnostic{{Module: "diag_only", Message: "something worth knowing"}}, nil
		},
	}

	reports, err := Doctor(modules, resources, diagnostics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected the diagnostics-only module to be reported, got %d", len(reports))
	}
	if len(reports[0].Pending) != 0 {
		t.Fatalf("expected no pending ops, got %v", reports[0].Pending)
	}
	if len(reports[0].Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %v", reports[0].Diagnostics)
	}
}

func TestDoctorErrorsOnMismatchedLengths(t *testing.T) {
	_, err := Doctor([]string{"a", "b"}, [][]Resource{{}}, []DiagnosticFunc{nil})
	if err == nil {
		t.Fatal("expected an error for mismatched slice lengths")
	}
}

// TestDoctorResolvesCrossModuleDependencies is a regression test for a
// real bug: an earlier version of Doctor diffed each module's resources
// in isolation, one at a time. A resource legitimately depending on
// ANOTHER module's resource (like security's alias-block injection
// depending on zsh having already written .zshrc.devboost — both target
// the same file with incompatible write semantics) then failed with
// "depends on unknown resource", because the dependency target wasn't in
// that module's own resource list. Doctor must diff the combined graph
// across all modules together, then group results by module afterward —
// exactly what Plan already does, just with an extra grouping step.
// (Doctor, like Plan, never executes anything — so this only asserts
// dependency *resolution* succeeds, not that execution effects
// propagate, which DiffAndExecute's own test already covers.)
func TestDoctorResolvesCrossModuleDependencies(t *testing.T) {
	moduleA := fakeKind{pending: true, desc: "converge a"}
	moduleBDependsOnA := fakeKind{pending: false}

	modules := []string{"module_a", "module_b"}
	resources := [][]Resource{
		{{ID: "a", Kind: moduleA}},
		{{ID: "b", Kind: moduleBDependsOnA, DependsOn: []string{"a"}}},
	}
	diagnostics := []DiagnosticFunc{nil, nil}

	reports, err := Doctor(modules, resources, diagnostics)
	if err != nil {
		t.Fatalf("expected the cross-module dependency to resolve, got error: %v", err)
	}
	if len(reports) != 1 || reports[0].Name != "module_a" {
		t.Fatalf("expected only module_a to report a pending change, got %+v", reports)
	}
}
