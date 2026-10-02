<!-- review-status: folded -->
# 2026-09-27 · CompPan · comppan-pressure-staging

**Session**: PANCCADIA CompPan 2.7.0 suction-pressure staging + DashboardPan 2.8.2, ODD + RDD (local commits on `feat/comppan-pressure-staging`, not deployed)
**Delta count**: 4

## What happened
CompPan moved from calling-room staging to three configurable suction-pressure steps (defaults 25/30, 20/26, 17/24 psi) with dual-sensor validation, three selection modes, HOA-always-wins, minOff default 0, and a JSON backup of the web-writable config. The whole-branch native review was refused with `lens_context_budget_exceeded`, so review ran per commit in detached worktrees. Per-commit review found one CRITICAL in the new config backup (restore kept the last parsing candidate, so a stale `.bak`/`.tmp` overrode a valid dest), corrected inside the lifecycle and validator-approved. The follow-up that added a revision-gated restore was approved but raised new restart-persistence findings (revision slot clobbered before seed, rotation state no longer restored, rejected values landing in the backup). They are recorded as pre-upgrade blockers.

## Evidence
- Whole-range review refused: `lens_context_budget_exceeded`, no authority created `[ev: gentle-ai review start, lineage review-a6ea10f13aa78c59]`
- T3 CRITICAL R3-staging-seed-candidate-precedence-ignores-recency, fixed by `286ad6c` (cherry-picked as `606b27d`) `[ev: 286ad6c]`
- Adapter `minOff` default left at 3 min while the pure model said 0; caught by parent review, fixed `e3f7e59` `[ev: e3f7e59]`
- f2 findings on revision-gated restore, recorded as blockers in `odd/tasks/comppan-pressure-staging.md` `[ev: 20dc596]`
- No new psi slot uses a BUnit facet (2026-09-25 static-init outage lesson applied) `[ev: 23ded68]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Size RDD candidates at planning time: when a feature forecasts > ~1000 authored lines (adapter + generated slotomatic region count), plan one review per work-unit commit from the start, using a detached worktree per commit (`git worktree add --detach <wt> <commit>` + `--base-ref <parent>`), instead of discovering `lens_context_budget_exceeded` at the end. | `BUILD-LOOP.md` § review / delivery | `[ev: lineage review-a6ea10f13aa78c59]` |
| Δ2 | Config backups are NOT monotonic: any backup of operator config must restore by recency (dest > .tmp > .bak) with an explicit revision, never "any valid candidate" or a max-merge copied from an hours-style backup. Add a checklist line + a pure `selectForRestore`-style helper pattern. | `types/logic.md` persistence section | `[ev: 286ad6c]` |
| Δ3 | A revision-gated restore needs four proofs before deploy: revision loaded before any execute writes it, increments only on accepted changes (not on boot or idle steps), every persisted field (e.g. rotation state) bumps or has its own recency, and only last-valid values are backed up. | `types/logic.md` persistence section | `[ev: 20dc596]` |
| Δ4 | When a spec changes a default, add a source-structural test on the adapter's generated default (2nd argument of `newProperty`), not only the pure-model value; the two drifted here. | `METHODOLOGY.md` slot checklist | `[ev: e3f7e59]` |

## Lessons
- Plan per-commit native review for large features; the whole-branch candidate will not fit.
- Operator-config backups need recency plus a revision, never max-merge or "last parsed wins".
- Pure-model defaults and adapter slot defaults drift; test both.
- Every new persistence mechanism must be reviewed specifically for its restart path before a JACE upgrade.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-27-comppan-pressure-staging.md | CompPan | 2026-09-27 | pending | 4 |`
