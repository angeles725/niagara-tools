<!-- review-status: pending -->
# 2026-09-06 · kit · campaign10-write-path-stale

**Session**: Campaign 10 PR5 — S25 lint-write-path STALE advisory + --strict flag
**Delta count**: 3

## What happened

The write-path matrix lint only checked the FAIL direction (source OPERATOR slot with no matrix row). The opposite direction — a matrix row naming a slot that no longer exists in source — was invisible; designers had to grep manually and results were unreliable. PR5 added the STALE advisory class (per-row, with a [concept] exemption and a --strict promotion flag) and the matrix-root-wide covered set that makes the count invariant across module roots. The gate's exit contract was already hard (exit 1 for uncovered), so --strict needed a clear semantics: it promotes STALE to exit 1 without weakening the FAIL direction in either mode.

[ev: corpus B831 §831.4] [ev: kit lint-write-path.sh:383 @ 61fa1f5] [ev: client docs/write-path-matrix.md:31-52 @ ff1b659]

## Evidence

- Real-tree: 5 STALE rows from CompPan-rt and ColdRoomPan-rt (same matrix root → root-invariant); action rows :64/:65 NOT STALE (covered set includes @NiagaraAction) `[ev: lint-write-path.sh smoke @ ff1b659]`
- Mutation RED: OPERATOR-only scanner drops @NiagaraAction → WP-stale-action FAILS + real-tree count grows `[ev: bats tests/lint-write-path.bats WP-stale-action]`
- shellcheck 0.11.0 exit 0 on modified lint-write-path.sh `[ev: shellcheck run 2026-09-06]`
- 22/22 bats lint-write-path.bats GREEN; 393/393 full bats suite GREEN `[ev: bats run 2026-09-06]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | STALE advisory class + --strict flag added to lint-write-path.sh | toolbelt/lint-write-path.sh | [ev: retro campaign10-write-path-stale] |
| 2 | BUILD-LOOP.md routing entry extended with STALE + --strict facts | BUILD-LOOP.md §5 pre-gate | [ev: retro campaign10-write-path-stale] |
| 3 | skill/SKILL.md toolbelt entry extended with STALE + --strict facts | skill/SKILL.md step 5 | [ev: retro campaign10-write-path-stale] |

## Lessons

- Verify a gate's exit contract before proposing a flag: if uncovered FAIL already exits 1, --strict must add something NEW (STALE promotion), not weaken the existing gate. [ev: D5e]
- STALE is per-row, not per-name: one [concept] mark on one hoaMode row must never silently exempt two other hoaMode rows — the exemption test runs BEFORE any sort -u de-dup. [ev: D5b]
- The covered set must be matrix-root-wide (all modules, @NiagaraProperty ∪ @NiagaraAction, any flag); a per-module or OPERATOR-only set produces false STALE on documented actions and non-OPERATOR slots. [ev: D5a]
- Comment-strip before [concept] matching: a [concept] token inside a markdown <!-- --> comment on a different line must not exempt data rows; test with a comment-only decoy pin (WP-stale-concept-decoy). [ev: D5d]
- Multi-line annotation harvest requires matching the name="X" FIELD LINE, not a single-line @Niagara…name= pattern; the single-line approach undercounts and produces false STALE. [ev: apply-package S25 §covered-side]

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign10-write-path-stale.md | kit | 2026-09-06 | pending | 3 |`
