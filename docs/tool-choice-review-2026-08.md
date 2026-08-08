# Tool-Choice Review — 2026-08-08

The first pass of the periodic adversarial re-review process (see
[issue #9](https://github.com/rolfsormo/devboost/issues/9)): seven
independent research passes, each asked two questions per category —
what would we pick with no history, and does the current default still
hold up. This document is the record of that pass; a formatted version
is also published as an artifact (link in the original PR/commit, if
still live — artifacts aren't guaranteed permanent, this file is the
durable copy).

19 defaults reviewed. 0 code changes made by the review itself — see
"Status" below for what was actually implemented afterward.

## Summary

| Category | Current (at time of review) | Verdict | Why |
|---|---|---|---|
| zsh plugin manager | znap | **Reconsider** | Benchmarks favor antidote/sheldon; zinit (issue #6) is actually slower without careful tuning |
| Syntax highlighting | zsh-syntax-highlighting | **Reconsider** | fast-syntax-highlighting is the widely-adopted faster successor |
| Shell prompt | starship | Keep | Lead widened — 2.5× stars, 3.7× installs vs. oh-my-posh |
| └ `command_timeout` | 700ms | Uncertain (doc-only) | Deliberate deviation from upstream's 500ms — reasonable, was mislabeled |
| └ `truncation_length` | 3 | Keep | Matches starship's own default exactly |
| └ `add_newline` | false | Uncertain (doc-only) | Overrides upstream default — no consensus either way, genuine taste call |
| Terminal multiplexer | tmux | Keep | Zellij real but hasn't overtaken; tmux fits unattended-bootstrap better |
| tmux plugin manager | TPM | Keep | Stagnant but not dead; successor (tpack) too new to trust |
| └ tmux-logging plugin | bundled always-on | **Reconsider** | 3–4× smaller adoption than the other three; should be opt-in |
| Toolchain manager | mise | Keep | Lead over asdf widened, not narrowed |
| └ python pin | 3.14 | Keep | Current stable line |
| └ go pin | 1.26 | **Stale value** | 1.27 release notes already published — GA imminent |
| └ node | lts (rolling) | Keep | Correct design — self-updates with zero devboost changes |
| cd / fuzzy / grep / find / cat / ls / du / git TUI / pager | zoxide, fzf, ripgrep, fd, bat, eza, dust, lazygit, delta | Keep | All confirmed clear favorites, no credible challenger found |
| df replacement | duf | Watch | Last release Sep 2025; nothing better exists yet |
| ps replacement | procs | Uncertain (doc-only) | Right pick for its niche, smaller community — was overstated in the doc comment |
| Shell history | atuin | Keep | Clear leader by adoption, activity, shell-support breadth |
| └ `filter_mode` | directory | **Reconsider** | Diverges from atuin's own default; most real configs scope only the up-arrow key |
| Node package-manager shim | corepack via npm | Keep | Matches Node TSC's own recommended path, no successor planned |

## What changed as a result

See git history following this file's commit for the actual
implementation of each **Reconsider**/**Stale** item and each doc-only
relabel. In short:

1. **zsh plugin manager (znap → reconsider)**: not migrated in this
   pass — a manager swap is a bigger, riskier change than the other
   items here and deserves its own dedicated PR, not a bundled
   drive-by. Issue #6 (which proposed znap → zinit) was updated with
   the benchmark evidence below, since its original premise doesn't
   hold up.
2. **Syntax highlighting plugin**: swapped `zsh-users/zsh-syntax-highlighting`
   → `zdharma-continuum/fast-syntax-highlighting`.
3. **tmux-logging**: gated behind `tmux.plugins.logging.enable`,
   default `false`.
4. **atuin `filter_mode`**: reverted to atuin's own default (`global`),
   with `filter_mode_shell_up_key_binding: directory` added for the
   quick-recall behavior the original setting likely intended.
5. **mise go pin**: left at 1.26 (still resolves to a current, patched
   release) with a doc-comment note to bump once 1.27 is GA — not
   urgent enough to guess a version number that isn't released yet.
6. **starship `command_timeout`/`add_newline`, procs**: doc comments
   relabeled to state these plainly as a deliberate deviation and a
   genuine taste call, rather than implying settled consensus.

## Full evidence, category by category

### zsh plugin manager (znap → reconsider, not yet migrated)

Two reproducible benchmark suites decide this.
[rossmacarthur/zsh-plugin-manager-benchmark](https://github.com/rossmacarthur/zsh-plugin-manager-benchmark)
(Docker + hyperfine, 23 real plugins, no turbo-mode tricks) puts
**antidote and sheldon** in the top, statistically-indistinguishable
tier — and puts **zinit in the "notably bad load time" tier**,
alongside zplug. [zsh-bench](https://github.com/romkatv/zsh-bench)
shows zinit can match antidote, but only with every plugin hand-annotated
with `wait`/`lucid` ice-modifiers — not the easy path. znap itself
appears in neither suite: there is no third-party-verified speed number
for it at all, only its own README claims.

Issue #6 proposed znap → zinit based on a conflict/dedup investigation,
not a controlled speed test — its premise ("the user's hand-tuned setup
uses zinit, so zinit must be fast") is contradicted by the benchmark
data. Migrating as originally scoped in that issue risks making startup
*worse* unless devboost also generates per-plugin turbo config.

**Recommendation**: don't migrate to zinit per issue #6 as originally
scoped. Evaluate antidote first (static-bundle architecture, hard to
misconfigure into slowness — matters for unattended config generation),
sheldon as an equally defensible second choice (Rust binary, TOML
config, easy to template from Go).

### Syntax highlighting plugin (reconsider → implemented)

`zsh-users/zsh-syntax-highlighting` is widely superseded by
`zdharma-continuum/fast-syntax-highlighting` for performance. Notably,
the user's own pre-existing zinit setup found in the original dedup
investigation was already using fast-syntax-highlighting — evidence
from inside this repo's own history that devboost's pick lagged a real
hand-tuned setup.

### tmux-logging plugin (reconsider → implemented)

resurrect (12,990★) + continuum (4,046★) are a deliberate,
still-recommended pair; yank (3,095★) solves an unmatched SSH-clipboard
problem. tmux-logging (1,251★) is a 3–4× adoption drop from yank and is
absent from most "essential setup" roundups that otherwise mirror the
other three exactly.

### atuin `filter_mode` (reconsider → implemented)

Atuin's own docs state the default is `global`, not `directory` —
devboost's override wasn't justified anywhere in the module. Real-world
configs that want directory-scoped recall typically keep ctrl-r search
global and scope *only* the up-arrow key via the separate
`filter_mode_shell_up_key_binding` setting. devboost's previous flat
setting restricted both, which is more limiting for interactive search
than most documented setups.

### mise go pin — 1.26 (stale value, watching)

Go 1.27 release notes are already published at go.dev — GA is
imminent. 1.26.5 (current patch) carries two CVE fixes, which the
`1.26` channel already resolves to automatically, so nothing is broken
today. This is exactly the drift pattern the module's own rationale
comment predicted.

### duf (df replacement) — watch, no action

Last release Sep 2025; recent commits are dependency bumps only. Still
the largest tool in its niche — the df-replacement space has very few
entrants at all, and no better alternative was found.

### Confirmed, no action

- **starship** — adoption lead widened (2.5× stars, 3.7× Homebrew
  installs vs. oh-my-posh); powerlevel10k confirmed unmaintained by its
  own maintainer.
- **tmux + TPM** — Zellij hasn't overtaken tmux; TPM's likely successor
  (tpack) is 6 months old at 201★, too early to trust.
- **mise** — lead over asdf widened; now 10th most-downloaded Homebrew
  formula overall.
- **zoxide, fzf, ripgrep, fd, bat, eza, lazygit, delta, atuin** — all
  confirmed clear community favorites by live star/activity data, no
  credible challenger found for any.
- **dust** — healthy and leading; gdu noted as a legitimate alternative
  for very large filesystem scans, not a replacement.
- **corepack** (`npm install -g corepack`) — matches Node's own
  TSC-recommended path; no successor exists or is planned.

### Uncertainty carried forward, not resolved

- **antidote vs. sheldon** — called "a very close, equally-defensible
  second choice" between the two; the original research didn't claim a
  single winner, only that both beat znap and zinit. Left open for
  whoever picks up the zsh-plugin-manager migration.
- **tpack** (possible TPM successor) — explicitly "not enough field use
  to trust yet." Revisit in a year, neither adopt nor dismiss now.

---

*Synthesized from 7 parallel research passes — zsh plugin manager,
prompt, tmux stack, toolchain manager, CLI tool bundle, git-delta &
shell history, corepack/Node currency. See individual module doc
comments in `engine/modules/` for how each finding was folded into the
code, and `.agents/skills/devboost-module-author/SKILL.md` for the
process this review followed.*
