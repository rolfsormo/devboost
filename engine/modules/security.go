package modules

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rolfsormo/devboost/config"
	"github.com/rolfsormo/devboost/engine"
	"github.com/rolfsormo/devboost/engine/kinds"
)

const (
	securityStartMarker = "# >>> devboost security start"
	securityEndMarker   = "# <<< devboost security end"
)

const devboostCheckAlias = `# Run 'devboost-check' at any time to see a summary of outdated tools.
devboost-check() {
    echo "=== devboost security check ==="
    local issues=0

    # Homebrew outdated
    if command -v brew &>/dev/null; then
        local outdated
        outdated=$(brew outdated --quiet 2>/dev/null | head -20)
        if [[ -n "$outdated" ]]; then
            echo "[brew] Outdated packages (run: brew upgrade):"
            echo "$outdated" | sed 's/^/  /'
            issues=$((issues + 1))
        else
            echo "[brew] All packages up to date"
        fi
    fi

    # mise outdated toolchains
    if command -v mise &>/dev/null; then
        local mise_outdated
        mise_outdated=$(mise outdated 2>/dev/null | grep -v "^Tool" | grep -v "^$" | head -10)
        if [[ -n "$mise_outdated" ]]; then
            echo "[mise] Outdated toolchains (run: mise upgrade):"
            echo "$mise_outdated" | sed 's/^/  /'
            issues=$((issues + 1))
        else
            echo "[mise] All toolchains up to date"
        fi
    fi

    # npm audit (only if a package.json is in the current directory)
    if command -v npm &>/dev/null && [[ -f "$(pwd)/package.json" ]]; then
        echo "[npm] Running audit in $(pwd)..."
        npm audit --audit-level=high 2>/dev/null || true
    fi

    if [[ $issues -eq 0 ]]; then
        echo "All checks passed."
    else
        echo ""
        echo "Tip: keep tools at stable LTS, not bleeding edge — run update checks weekly."
    fi
}`

// Security ports modules/module_security.sh's apply half: injects the
// devboost-check alias block into .zshrc.devboost, gated on
// security.enable.
func Security(cfg *config.Config) []engine.Resource {
	if cfg.Get("security.enable", "true") != "true" {
		return nil
	}
	includeFile := cfg.Get("zsh.include_file", "~/.zshrc.devboost")
	return []engine.Resource{
		{
			ID: "security_check_alias",
			Kind: kinds.BlockInFile{
				Path:        includeFile,
				StartMarker: securityStartMarker,
				EndMarker:   securityEndMarker,
				Content:     devboostCheckAlias,
			},
		},
	}
}

// SecurityDiagnostics ports modules/module_security.sh's doctor half:
// five read-only findings with nothing to converge — see engine.Diagnostic
// for why these are a separate mechanism from Resource/PendingOp rather
// than resources with a no-op Execute.
func SecurityDiagnostics(cfg *config.Config) engine.DiagnosticFunc {
	return func() ([]engine.Diagnostic, error) {
		if cfg.Get("security.enable", "true") != "true" {
			return nil, nil
		}
		var diags []engine.Diagnostic

		for _, key := range []string{"node", "python", "go", "rust", "deno"} {
			ver := cfg.Get("toolchains.globals."+key, "")
			if ver == "latest" {
				diags = append(diags, engine.Diagnostic{
					Module: "security",
					Warn:   true,
					Message: fmt.Sprintf(
						"toolchain '%s' is pinned to 'latest' — prefer a specific version or 'lts'/'stable' to avoid pulling unreviewed releases",
						key),
				})
			}
		}

		home, err := os.UserHomeDir()
		if err == nil {
			if _, err := os.Stat(filepath.Join(home, ".oh-my-zsh")); err == nil {
				diags = append(diags, engine.Diagnostic{
					Module: "security",
					Warn:   true,
					Message: "~/.oh-my-zsh detected — this is redundant with devboost's znap+starship setup and can " +
						"slow shell startup or conflict with it. Consider removing it (see README).",
				})
			}

			zshrc := filepath.Join(home, ".zshrc")
			if data, err := os.ReadFile(zshrc); err == nil {
				count := strings.Count(string(data), ".zshrc.devboost")
				if count > 1 {
					diags = append(diags, engine.Diagnostic{
						Module: "security",
						Warn:   true,
						Message: fmt.Sprintf(
							"%s sources .zshrc.devboost %d times — likely double-sourced, which doubles shell startup cost. Run: grep -n zshrc.devboost %s",
							zshrc, count, zshrc),
					})
				}
			}
		}

		tpmPath := cfg.Get("tmux.tpm_path", "~/.tmux/plugins/tpm")
		if _, err := os.Stat(filepath.Join(tpmPath, ".git")); err == nil {
			out, err := exec.Command("git", "-C", tpmPath, "remote", "get-url", "origin").Output()
			if err == nil && strings.HasPrefix(strings.TrimSpace(string(out)), "http://") {
				diags = append(diags, engine.Diagnostic{
					Module:  "security",
					Warn:    true,
					Message: "TPM remote uses plain HTTP — re-clone over HTTPS",
				})
			}
		}

		if brewRepo, err := exec.Command("brew", "--repository").Output(); err == nil {
			repoPath := filepath.Join(strings.TrimSpace(string(brewRepo)), "Library", "Taps", "homebrew", "homebrew-core")
			if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
				out, err := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%ct").Output()
				if err == nil {
					if lastFetch, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
						ageDays := int(time.Since(time.Unix(lastFetch, 0)).Hours() / 24)
						if ageDays > 7 {
							diags = append(diags, engine.Diagnostic{
								Module: "security",
								Warn:   true,
								Message: fmt.Sprintf(
									"Homebrew index is %d days old — run 'brew update' to get security patches", ageDays),
							})
						}
					}
				}
			}
		}

		if len(diags) == 0 {
			diags = append(diags, engine.Diagnostic{Module: "security", Message: "No obvious issues found"})
		}
		return diags, nil
	}
}
