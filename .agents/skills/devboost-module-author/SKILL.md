---
name: devboost-module-author
description: This skill should be used when adding a new module to devboost, changing which tool/default a devboost module uses, or writing/updating a module's rationale doc comment. Covers researching developer-community consensus before picking a default and documenting that research in the code.
version: 1.0.0
---

# Authoring devboost modules

## Overview

A devboost module is an opinionated default: "here is the tool we picked for
this job, and here is exactly how we configure it." Because devboost's whole
pitch is "new machine, run devboost, start coding in minutes" — a developer
trusts these choices without re-deriving them — every module needs both a
correct implementation and an honest, sourced explanation of why that choice
was made. This skill covers both halves: the research process, and how to
write it into the code so it survives as documentation, not just a commit
message.

This mirrors the project's own written principle (see `AGENTS.md` /
`CLAUDE.md`, "Research before defaulting"): before choosing a default value,
research actual developer-community preferences, don't just carry over
whatever the tool's own installer defaults to.

## When a module needs research

Any time you are:

- Adding a brand-new module that picks a specific tool (a new CLI utility,
  a new config default, a new "we install X instead of Y").
- Changing an existing module's tool choice or a specific config value.
- Writing/backfilling the rationale doc comment on an existing module that
  doesn't have one yet.

You do **not** need fresh research for pure mechanism/plumbing code (file
rendering, dependency wiring, CLI flag parsing) — the research obligation is
specifically about *which tool* and *which default value*, not the Go code
that applies it.

## Research checklist

For each tool-choice or default value, before writing the module:

1. **Check adoption/reputation, not just familiarity.** GitHub stars, last
   commit/push date, and whether the project is actively maintained are
   fast, checkable signals. Prefer a live check (`gh api
   repos/<owner>/<repo>` or WebFetch) over memory — package popularity and
   maintenance status genuinely change over time, and stale assumptions are
   exactly what this whole skill exists to prevent.
2. **Prefer first-party sources over third-party blog consensus when they
   conflict.** If tool A officially documents "the recommended way to use us
   with tool B is X," that beats a five-year-old blog post recommending Y,
   even if Y is more commonly copy-pasted. Go read the tool's own docs
   before trusting general "best practice" writeups.
3. **Check whether the choice is still current, not just whether it was
   ever correct.** Tools get deprecated, unbundled, or superseded (see
   corepack's unbundling from Node 25+, found during this project's own
   module-documentation pass). A default that was right when first chosen
   can silently become wrong. Look for the tool's own changelog/release
   notes/roadmap discussion, not just "does the tool still exist."
4. **Distinguish genuine consensus from a plausible-sounding default.** Some
   config values have real documented justification (tmux's
   `escape-time = 0` fixing a specific, widely-reported perceived-lag
   problem). Others are just "a reasonable choice nobody strongly
   disagrees with" (a prompt's truncation length). Both are fine to ship —
   but the doc comment must say which kind it is. Never write confident
   prose for an undocumented preference.
5. **When your own investigation produced the evidence, use that.**
   devboost's dedup modules exist because this project's own instrumented
   measurement (zprof) found nvm's shell hook costing ~850-900ms per login
   shell — that's stronger evidence than any external survey. If you did
   the measurement, cite the measurement.

## Writing the doc comment

Put the rationale directly above the module's exported constructor function
(the `func Foo(cfg *config.Config) []engine.Resource` — see
`engine/modules/git.go` or `engine/modules/tmux.go` for real examples), as a
Go doc comment. Requirements:

- **State the why, not just the what.** "Git ports modules/module_git.sh"
  describes behavior; "delta is the clear community-favorite modern git
  pager — 31.7k stars vs. diff-so-fancy's 18.1k, checked live" is rationale.
  Both belong in the comment, but the rationale is the part this skill is
  about.
- **Name the alternatives you rejected, and why.** A choice only reads as
  researched if the reader can see what else was on the table. For any
  category with a real competing option (a different pager, a different
  version manager, a different plugin manager), name it and give the
  concrete reason it lost — adoption gap, missing a feature the winner has,
  official guidance pointing the other way, or "close call, no strong
  reason, we just had to pick one." "We chose delta over diff-so-fancy
  because delta has ~1.75x the stars and comparison writeups consistently
  call it more capable" is a real comparison; "we chose delta" alone is
  not, even with a star count attached. If the runner-up is close enough
  that a reasonable person could disagree, say that too — don't manufacture
  a bigger gap than the research actually found.
- **Cite something checkable.** Star counts, a specific doc URL, a specific
  measurement from this project's own history. Avoid unsourced claims like
  "widely considered best practice."
- **Be honest about confidence level, explicitly.** Say outright when a
  value is "genuinely well-documented consensus" versus "a reasonable
  choice, no strong consensus found" versus "this is now questionable and
  here's why" (see `engine/modules/corepack.go` and
  `engine/modules/direnv.go` for real examples of the third case — a
  default that research revealed is no longer clearly right, documented as
  such rather than defended).
- **Flag values that will go stale.** Pinned versions (a specific language
  version, not a rolling "lts"/"stable" channel) need periodic revisiting —
  say so in the comment so a future reader knows not to treat the number as
  permanent (see `engine/modules/mise.go`'s python/go version handling).
- **Cross-reference, don't duplicate.** If three modules share one
  underlying investigation (see the three `*_dedup.go` modules and their
  shared measured-lag rationale), write the substantive explanation once
  and have the others point to it, rather than copy-pasting.
- **It's fine to say "no strong consensus."** Not every value has a
  research trail — say that plainly rather than inventing a justification.
  A module that names its uncertainty is more trustworthy than one that
  fabricates confidence.

## Keeping rationale current: adversarial re-review

Defaults don't stay correct forever — the ecosystem moves. When revisiting
an existing module (not just writing a new one), treat it as a live
question, not an inherited fact:

- Ask "if I were choosing this tool for the first time today, with no
  history, would I still pick it?" — not "can I justify what's already
  there?"
- Actively look for reasons the original choice might now be wrong
  (deprecation notices, a newer tool that's overtaken the incumbent,
  official guidance that changed), not just evidence that supports keeping
  it.
- If you find a default is now questionable, say so directly in the doc
  comment (see corepack's unbundling, or direnv's discouraged mise
  integration, both documented as open concerns rather than silently kept
  or silently swapped) — devboost's config choices are meant to be
  "opened up for anyone to challenge," per the project's own instructions,
  which means surfacing doubt is more useful than hiding it.
- Log genuinely open questions (a default worth reconsidering but not yet
  decided) as a tracked issue rather than either fixing it unilaterally or
  letting the finding evaporate.

## Cross-agent compatibility

This file's real location is `.agents/skills/devboost-module-author/SKILL.md`
— the vendor-neutral convention (e.g. Codex CLI reads it there directly).
`.claude/skills` is a symlink to `.agents/skills`, so Claude Code and other
tools that read `.claude/skills/` (OpenCode, Cursor, Copilot) see the exact
same content with no separate copy to keep in sync. Edit only the
`.agents/skills/` original; any future skill just needs adding once, under
`.agents/skills/`.

## Quick checklist for a new module

- [ ] Researched adoption/reputation for the tool choice (checkable source)
- [ ] Checked the choice is still current, not just historically correct
- [ ] Checked the tool's own first-party docs for integration guidance
      where relevant (not just third-party consensus)
- [ ] Doc comment above the constructor states why, with a source
- [ ] Confidence level stated honestly (consensus vs. taste vs. questionable)
- [ ] Pinned/version-specific values flagged as needing periodic review
- [ ] Shared rationale cross-referenced instead of duplicated
- [ ] `go build ./...` and `go test ./...` still pass
