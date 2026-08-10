package engine

import "fmt"

// ModuleReport is one module's worth of doctor output: its pending
// resource changes (same shape plan/apply already compute, just grouped
// by module) plus any read-only Diagnostics.
type ModuleReport struct {
	Name        string
	Pending     []PendingOp
	Diagnostics []Diagnostic
}

// Doctor computes ONE combined diff across every module's resources
// together — same as Plan — then groups the resulting PendingOps back by
// which module owns each resource ID for readable, tool-first output.
//
// This must diff the combined graph, not each module in isolation: a
// resource can legitimately DependsOn another module's resource (e.g.
// security's alias-block injection depends on zsh having already
// written .zshrc.devboost, since both target the same file with
// incompatible write semantics — diffing security's resources alone
// would make that dependency unresolvable, since the resource it depends
// on wouldn't even be in the list). Grouping happens after the diff, as
// a pure reporting step, not by fragmenting the diff itself.
func Doctor(modules []string, resourcesByModule [][]Resource, diagnosticsByModule []DiagnosticFunc) ([]ModuleReport, error) {
	if len(modules) != len(resourcesByModule) || len(modules) != len(diagnosticsByModule) {
		return nil, fmt.Errorf("doctor: modules/resources/diagnostics length mismatch")
	}

	moduleOf := make(map[string]string) // resource ID -> owning module name
	var combined []Resource
	for i, name := range modules {
		for _, r := range resourcesByModule[i] {
			moduleOf[r.ID] = name
			combined = append(combined, r)
		}
	}

	pending, err := ComputeDiff(combined)
	if err != nil {
		return nil, err
	}

	pendingByModule := make(map[string][]PendingOp)
	for _, op := range pending {
		name := moduleOf[op.ResourceID]
		pendingByModule[name] = append(pendingByModule[name], op)
	}

	reports := make([]ModuleReport, 0, len(modules))
	for i, name := range modules {
		var diags []Diagnostic
		if fn := diagnosticsByModule[i]; fn != nil {
			diags, err = fn()
			if err != nil {
				return nil, fmt.Errorf("module %s diagnostics: %w", name, err)
			}
		}
		modulePending := pendingByModule[name]
		if len(modulePending) == 0 && len(diags) == 0 {
			continue // nothing to report for this module at all
		}
		reports = append(reports, ModuleReport{Name: name, Pending: modulePending, Diagnostics: diags})
	}
	return reports, nil
}
