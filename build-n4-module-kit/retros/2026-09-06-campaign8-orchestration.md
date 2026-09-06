<!-- review-status: folded -->
# campaign8-orchestration

**Date**: 2026-09-06
**PR**: PR17 — docs/c8-orchestration
**Branch**: docs/c8-orchestration
**Module/Scope**: kit (doc-only)

## What happened

Created `ORCHESTRATION.md` — a session-level contract explaining when to use each tool (research-sdd / gentle SDD / BUILD-LOOP), which model runs each gentle-SDD phase, how the three hand off end-to-end, and the hard per-run retro/ticket loop. Extended `skill/SKILL.md` with steps 1b (explore shard audit-first >3 files, sonnet), 1c (design shard for schema/new slot, opus), and 5b (peer QA session before every code PR, K13). Added `kit-links.bats` L8 asserting every script named in `ORCHESTRATION.md` exists at `toolbelt/<script>.sh`.

## Evidence

grep-before-fold counts (task 17.1):

| Pattern | Hits before |
|---|---|
| `8 sections` | 0 |
| `Delegation triggers` | 0 |
| `Escalation gate` | 0 |
| `Judgment-Day.*high-risk` | 0 |

`bats tests/` results: **284/285 pass**, 1 fail (L8 — `kit-ticket.sh` and `new-retro.sh` missing from `toolbelt/`, pending PR16 merge).

L8 bite proof: added `<!-- MUTATION-PROOF: fake-extra.sh -->` to ORCHESTRATION.md → L8 failed with "script in ORCHESTRATION.md missing from toolbelt/: fake-extra.sh"; removed → back to exactly 2 missing (new-retro.sh, kit-ticket.sh).

`sweep-fold-audit --strict`: exit 0 (56 folded, 56 cited, 0 uncited).
`sweep-build-state`: exit 0.

Commits: 9adb79d (ORCHESTRATION.md), b46d972 (SKILL.md), d69454a (kit-links.bats L8).

## Proposed kit deltas

| # | Delta | Target | Action |
|---|---|---|---|
| Δ1 | L8 numbering deviation: wave3.md calls it "L7 extension" but L7 already existed (PR13 §6.a step scripts); added as L8 | `build-n4-module-kit/retros/2026-09-06-campaign8-orchestration.md` | record only — numbering is correct, deviation documented |
| Δ2 | ORCHESTRATION.md drops `toolbelt/` prefix for `new-retro.sh` / `kit-ticket.sh` references to avoid L1 failures before PR16 merges; process content is identical | `ORCHESTRATION.md` | re-add prefix when PR16 merges and L1 can resolve them |
| Δ3 | step 5b is labelled after step 5 (comes logically after pre-gate checks, before reload triage); additive, no renumber | `skill/SKILL.md` | no follow-up needed |

## Lessons

1. The L7-vs-L8 numbering collision is a predictable hazard when two wave-3 PRs both say "extend L7"; confirm the current maximum L-number before writing the new assertion label — the correct check is `grep -c '"L[0-9]' tests/kit-links.bats` not the wave spec.
2. Avoiding L1 false failures from pending-PR script references requires either (a) dropping the `toolbelt/` path prefix from doc examples until the scripts land, or (b) adding an L1 exclude-list; option (a) is the zero-config default — note it in the retro so the prefix is restored when PR16 merges.
3. L8's regex `[a-z][a-z0-9-]+\.sh` correctly harvests bare script names; future scripts with uppercase characters or underscores would need the pattern extended.
