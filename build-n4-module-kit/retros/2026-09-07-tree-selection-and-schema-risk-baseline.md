<!-- review-status: folded -->
# 2026-09-07 · kit · tree-selection-and-schema-risk-baseline

**Session**: PANCCADIA multi-module session — cross-cutting workflow defects (wrong work tree; schema-risk baseline).
**Delta count**: 3

## What happened
Two workflow issues surfaced. (1) WRONG WORK TREE: work started in `/home/cristian/niagara-panccadia-leon` (Dashboard 2.2.0/CompPan 2.0.3/ColdRoomPan 2.0.7, git "against hon414") while the LIVE/canonical baseline was `modulos_niagara_n4/Cliente/Leon-Guanjuato` (2.0/2.0.1/2.0.3 = the `Leon-Guanjuato.rar` deployed source). The divergence (BDashboardService.java differed 38 lines) was only caught when the operator asked. (2) SCHEMA-RISK BASELINE: to run `schema-risk` the "before" snapshot had to be reconstructed from the deployed source (the RAR), and its LOSSY verdict on CompPan needed a live-bog cross-check to prove it harmless.

## Evidence
- version drift live vs work tree: Dashboard 2.0 vs 2.2.0; CompPan 2.0.1 vs 2.0.3; ColdRoomPan 2.0.3 vs 2.0.7 [ev: build.gradle.kts both trees]
- `schema-risk` baseline = RAR source snapshot (`module-include.xml + *.java` layout) vs current tree → ColdRoomPan SAFE, DashboardPan SAFE, CompPan LOSSY [ev: schema-risk.sh]
- CompPan LOSSY reason = removed `suctionBand`; bog-nav on live `Programacion/CompressorControl` = 0 persisted → non-issue [ev: bog-nav]
- `build.sh` failed twice (exit 2 usage, exit 10 env) before passing `<group-dir> <MOD> <niagara_home>` [ev: build.sh runs]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Orient/preflight version-drift guard: when several checkouts of the same module exist, warn if the chosen work tree is NOT the one matching the deployed/live version (or if a sibling checkout has a higher version). Print the tree + version per candidate before editing. | toolbelt/preflight.sh; BUILD-LOOP §0 orient | [ev: version drift] |
| 2 | Document the `schema-risk` deploy workflow: baseline "before" = the DEPLOYED source (extract the station backup / RAR into the `module-include.xml + *.java` snapshot shape), "after" = the build; and a LOSSY `remove_slot_*` must be cross-checked against the live bog for actual persistence (0 persisted → not a blocker). | build-verify.md §schema-risk | [ev: schema-risk + bog-nav] |
| 3 | `build.sh` invocation reminder near the top of BUILD-LOOP: args are `<module-root=GROUP dir> <MOD> [niagara_home]`; niagara_home (arg 3 or env) is REQUIRED, and a client multi-project layout puts `./gradlew` at an ancestor. | BUILD-LOOP §build | [ev: build.sh exit 2/10] |

## Lessons
- Confirm the work tree matches the DEPLOYED baseline (version + git) BEFORE editing; multiple divergent checkouts of the same client module are a real hazard.
- All module work should live under `modulos_niagara_n4/Cliente/`; a scattered checkout (`niagara-panccadia-leon`) caused the drift (operator rule).
- `schema-risk` needs a reconstructed deployed-source baseline; its LOSSY is pessimistic — a removed slot not persisted in the live bog is safe.
- Zero OUTAGE across all modules = no load-crash; the residual proof is `triage-console` after a reload on a backup.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-tree-selection-and-schema-risk-baseline.md | kit | 2026-09-07 | pending | 3 |`
