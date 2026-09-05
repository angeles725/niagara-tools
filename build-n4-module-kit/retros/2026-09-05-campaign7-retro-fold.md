<!-- review-status: pending -->
<!-- kit-retro -->
# Campaign 7 PR1: fold the 9 Campaign-6 retros — K11–K14, §7 envelope rule, SDD-ledger notes, two-reads rule

Date: 2026-09-05 · Module: kit · SDD: build-n4-module-campaign7 PR1 (v0.17.0 → v0.18.0 chain)

PR1 of 8 in the Campaign-7 chain. Doc-only fold: zero new bats tests (padding = defect).
Branch: `docs/c7-retro-fold`. Gate exits: R1.1–R1.8 (sweep + fold-audit + markers + INDEX rows).

## What was folded

9 Campaign-6 retros (all pending since 2026-09-05) folded into three kit files:

| Retro | Token | Destination |
|---|---|---|
| campaign6-close-process-meta-lessons | `close-process-meta-lessons` | METHODOLOGY K11–K14; BUILD-LOOP §7; CONTRIBUTING §9 |
| campaign6-doctrine-fold | `doctrine-fold` | BUILD-LOOP §7 (envelope-pairing rule) |
| campaign6-types-fold | `types-fold` | BUILD-LOOP §7 (envelope-pairing rule) |
| campaign6-marker-index-sweep | `marker-index-sweep` | METHODOLOGY folded-as-code line |
| campaign6-fold-audit-and-coverage | `fold-audit` | METHODOLOGY folded-as-code line |
| campaign6-preflight-and-slot-coverage | `preflight-and-slot-coverage` | METHODOLOGY folded-as-code line |
| campaign6-conformance-lints | `conformance-lints` | METHODOLOGY folded-as-code line |
| campaign6-close | `campaign6-close` | METHODOLOGY folded-as-code line |
| campaign6-research-fold | `research-fold` | METHODOLOGY code-fold rule + CONTRIBUTING §9 |

## Grep-before-fold results (R1.1 — all 0 in non-retro kit files)

| Term | Hits in non-retro kit files |
|---|---|
| K11 | 0 |
| K12 | 0 |
| K13 | 0 |
| K14 | 0 |
| envelope-pairing | 0 |
| SDD-ledger / SDD ledger | 0 |
| two-independent-reads / two independent reads | 0 |

## Rules folded (one terse line each)

**METHODOLOGY.md §Kit-maintenance (K11–K14 + folded-as-code lines):**
- K11: grep commit bodies + PR body for attribution before publishing, amend locally.
- K12: workers write only in their worktree; live docs merged never overwritten; numstat before PR.
- K13: cite QA RED branches by name; re-read tip at apply — hashes go stale in hours.
- K14: metric name states what it measures (type-set vs per-slot).
- folded-as-code citations added for: marker-index-sweep, fold-audit, preflight-and-slot-coverage, conformance-lints, campaign6-close, research-fold.

**BUILD-LOOP.md §7 (envelope-pairing rule):**
- All close-exits pair their anchor with BUILD-STATE.md self-envelope in the same push range; branch push not proof.

**CONTRIBUTING.md §9 (SDD ledger discipline):**
- One active attempt per change; single-line evidence goal; inventory sha from error; docs early or budget sized.
- Two independent reads for every research fold.

## Self-verify

| Claim | Evidence |
|---|---|
| 9 pending rows → 0 | `grep -c '| pending |' retros/INDEX.md` = 0 after flip |
| 9 retro files stamped folded | `head -1 retros/2026-09-05-campaign6-*.md` all = `<!-- review-status: folded -->` |
| sweep-fold-audit --strict exits 0 | run in guard step |
| sweep-build-state exits 0 | run in guard step |
| bats 152/152 | no new tests (doc-only PR) |
| shellcheck exit 0 | no script changes |
