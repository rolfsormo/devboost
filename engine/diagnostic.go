package engine

// Diagnostic is a read-only finding with nothing to converge — e.g. "this
// toolchain is pinned to 'latest', which pulls unreviewed releases," or
// "oh-my-zsh is installed alongside devboost, which is redundant." This
// is deliberately a separate concept from Resource/PendingOp, not another
// use of PendingOp with a no-op Execute: a Diagnostic was never a pending
// *change* in the first place, so it shouldn't appear in apply's "N
// changes made" accounting the way a real converged resource does, and a
// module offering only diagnostics (nothing to install/fix) is a
// different, legitimate shape from a module with nothing to report at
// all — Diagnostics lets that distinction be real instead of implied by
// an empty PendingOp.
//
// A DiagnosticFunc returning nil means nothing to report.
type Diagnostic struct {
	Module  string // which module this finding belongs to, for grouped doctor output
	Message string
	Warn    bool // true for a warning-level finding, false for informational/success
}

// DiagnosticFunc is what a module supplies for doctor: a function that
// inspects live system state and returns zero or more findings. Unlike
// ResourceKind.Diff, there's no "desired vs. actual" comparison implied —
// a diagnostic can report on anything worth surfacing, including things
// with no notion of convergence at all (e.g. "an unrelated tool your
// devboost setup didn't install is present").
type DiagnosticFunc func() ([]Diagnostic, error)
