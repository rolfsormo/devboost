package kinds

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rolfsormo/devboost/engine"
)

// ManagedBinDir is where devboost installs tools that aren't available
// through the platform's own package manager (mise, atuin, starship,
// dust on apt-based Linux — none of these are packaged in Ubuntu/
// Debian's default repos, confirmed via a real Ubuntu container run).
// XDG-conventional user-writable location, chosen specifically so
// installing here never needs sudo. Not on PATH by default in a bare
// shell (confirmed: Ubuntu's default .profile only conditionally adds
// it), so the zsh module adds it to PATH in its own managed config —
// see zshdevboost.go — rather than relying on any tool's own installer
// to do that (several of them do, unconditionally, which devboost
// deliberately avoids: see the atuin doc comment below for the specific
// installer that does this).
func ManagedBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// BinaryAvailable reports whether name is runnable right now — checked
// fresh on every call (PATH via exec.LookPath, plus ManagedBinDir
// directly, since a PATH change devboost writes to .zshrc only takes
// effect in a future shell, not this process), never cached or decided
// once at module-construction time.
//
// This distinction matters for real correctness, not just style: a
// resource that gates itself off entirely if a tool isn't found BEFORE
// apply starts executing anything can never see a tool a different
// resource installs earlier in that same run — found as a real bug via
// a live Ubuntu container apply, where Mise()'s toolchain-converge
// resource silently never ran because it checked exec.LookPath("mise")
// at the moment modules.AllResources() was called, before the
// vendor-install resource that installs mise itself had executed.
// Kinds and modules should call this from inside Diff()/Satisfied()
// (execution time, re-evaluated every apply/plan/doctor run) rather
// than once when constructing a module's resource list — if a check
// like this ever becomes expensive enough to need memoizing, that's an
// explicit opt-in at the call site, not something baked into every
// caller by default.
func BinaryAvailable(name string) (bool, error) {
	_, err := ResolveBinary(name)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// ResolveBinary returns the actual absolute path to name — checking
// PATH first via exec.LookPath, then ManagedBinDir directly only if
// that fails — for callers that need to actually exec it, not just
// check presence. PATH is tried first, deliberately: on a machine where
// the tool genuinely is already on PATH (the common case once a user
// has actually restarted their shell after a previous apply), this is
// exactly what exec.LookPath would have done anyway — ManagedBinDir is
// purely a fallback for the one real gap, not a first-choice override.
//
// The gap it fills: devboost's own process PATH never includes
// ManagedBinDir (it's only added to *future* shells via the zsh
// module's rendered PATH export — see zshdevboost.go), so a bare
// exec.Command("mise", ...) call from inside devboost itself silently
// fails with "executable file not found in $PATH" on a genuinely fresh
// machine where mise was JUST installed into ManagedBinDir moments
// earlier in the same apply run. Found via a real Ubuntu container run
// where this exact failure mode was masked by a swallowed error (see
// mise.go's Converge, which discarded `mise use`'s error entirely) —
// any module invoking a managed-install tool (mise, atuin, starship,
// dust) by literal command name must resolve it through here first
// (or, better, via Command below).
func ResolveBinary(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	dir, err := ManagedBinDir()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("%s not found on PATH or in %s", name, dir)
}

// Command is exec.Command for managed-install tools (mise, atuin,
// starship, dust, lazygit, procs): it resolves name through
// ResolveBinary first, so it correctly finds a tool that landed in
// ManagedBinDir instead of on PATH. Falls back to the bare name
// unresolved if ResolveBinary fails entirely, so the resulting exec
// still produces exec.Command's familiar "executable file not found in
// $PATH" error rather than a confusing empty-path failure.
//
// Every call site that invokes one of these tools by name should use
// this instead of exec.Command directly — the one place this
// resolution logic needs to exist, rather than re-implemented (or
// forgotten) at each site, the direct cause of a real bug found via a
// live Ubuntu container run (mise.go and corepack.go each separately
// needed the same fix). Tools that are never installed via
// VendorInstall/GitHubReleaseInstall (git, brew, apt-get, sh, and any
// plain PATH-only system tool) have no reason to route through here —
// this is specifically the managed-tool case, not a blanket
// exec.Command replacement.
func Command(name string, args ...string) *exec.Cmd {
	resolved, err := ResolveBinary(name)
	if err != nil {
		resolved = name
	}
	return exec.Command(resolved, args...)
}

// VendorInstall declares "install this tool via its own official
// installer script, into devboost's managed bin dir, without letting it
// touch shell config files devboost manages itself." This is the
// escape-hatch-shaped-like-a-typed-kind for tools with no apt package
// but a real, official, non-interactive install script — the same
// "shell out to the tool's own CLI when it's the best available
// primitive" rule that already justifies ensureBrewInstalled fetching
// Homebrew's own installer.
//
// ScriptURL is fetched fresh each Execute (not cached) — these are
// small, versioned installer scripts, not large binaries, and always
// fetching the current one avoids devboost embedding a URL that
// silently starts serving a broken/moved script.
type VendorInstall struct {
	BinaryName string            // e.g. "mise" — used for the idempotency check
	ScriptURL  string            // e.g. https://mise.run
	Env        map[string]string // extra env vars passed to the installer (install dir, non-interactive flags, etc.)
}

func (v VendorInstall) Diff() (*engine.PendingOp, error) {
	found, err := BinaryAvailable(v.BinaryName)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, nil
	}

	return &engine.PendingOp{
		Description: fmt.Sprintf("install %s via its official installer", v.BinaryName),
		Execute: func() error {
			return v.install()
		},
	}, nil
}

func (v VendorInstall) install() error {
	dir, err := ManagedBinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	script, err := fetchBody(v.ScriptURL)
	if err != nil {
		return fmt.Errorf("fetch installer for %s: %w", v.BinaryName, err)
	}
	defer script.Close()

	cmd := exec.Command("sh", "-s")
	cmd.Stdin = script
	cmd.Env = os.Environ()
	for k, val := range v.Env {
		cmd.Env = append(cmd.Env, k+"="+val)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install %s: %w\n%s", v.BinaryName, err, out)
	}
	return nil
}

// fetchBody performs an HTTP GET and returns the response body for the
// caller to read and close — a small shared helper since VendorInstall
// and GitHubReleaseInstall both need to fetch remote content.
func fetchBody(url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, url)
	}
	return resp.Body, nil
}
