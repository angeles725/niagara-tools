<!-- review-status: pending -->
# 2026-09-06 · kit · campaign11-client-root

**Session**: C11 PR3 — centralise client-tree defaults in tests/lib/client-root.bash (T2)
**Delta count**: 3

## What happened
Ten bats files hardcoded the absolute path to the Leon-Guanjuato client tree, some with `${C9_CLIENT_ROOT:-...}` fallback and two (lint-delays:53, rc-scan:75) with no override at all. When the blessed worktree was retargeted from 4f5f1c7 to ff1b659, every one of those ten files had to be edited individually. The fix: one `tests/lib/client-root.bash` library defines the single default (`main-ff1b659`) and exports `CLIENT_READ_ROOT`, `C9_CLIENT_ROOT`, `C9_CLIENT_REPO`, `C8_CLIENT_REPO` with `:=` (env override wins). Each bats now does `load lib/client-root` at file scope and uses the exported variable with no fallback literal.

A second defect was unmasked: LD5 was asserting `exit 1 + FAIL BDefrostController` — a ColdRoomPan defrost time<=0 bug that was fixed post-C9. Once the path was retargeted to ff1b659 the assertion was wrong-class (pinning a fixed defect). The delay-floor rule is carried by synthetic fixtures LD1/LD3/LD6/LD11-misguard; LD5 was retargeted to assert the tree's current clean state (exit 0, no FAIL rows).

## Evidence
- 10 hardcoded `Leon-Guanjuato` literals across 10 bats at dab0807 `[ev: QA 54078f6 C11-T2-no-hardcode]`
- lint-delays.sh on 4f5f1c7: `exit 1, FAIL BDefrostController` (stale tree defect)
- lint-delays.sh on main-ff1b659: `exit 0, 0 FAIL rows, 4 PASS rows` (defect fixed) `[ev: R-T2.6]`
- rc-scan.sh on main-ff1b659: `exit 1, 1 FAIL host-literal src/rc/index.html:734` (steady-state detection, unchanged) `[ev: R-T2.9]`
- OBSERVED mutation 1: re-hardcode literal in ext-writable-shape.bats → C11-T2-no-hardcode: 1 offender (RED) → restore → 0 offenders (GREEN)
- OBSERVED mutation 2: delete tests/lib/client-root.bash → C11-T2-lib-exists: file absent (RED) → restore → file present (GREEN)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| D1 | Document the `tests/lib/` pattern and env-override rule in CONTRIBUTING.md or tests/README | CONTRIBUTING.md / docs/tests-README | `[ev: campaign11-client-root lesson 1]` |
| D2 | Smoke-assertion-class rule: codify in METHODOLOGY.md that a real-tree smoke must pin the tree's current correct verdict, not a known defect; a FAIL/WARN smoke citing a fixable defect must name the defect + its synthetic rule carrier | METHODOLOGY.md §RK5 | `[ev: campaign11-client-root lesson 2]` |
| D3 | C12 seed: LD5/LD10 both pin ColdRoomPan tree state at different commits; keep both (D3e), but a C12 lesson should record the class rule explicitly | retros/INDEX.md C12 seed | `[ev: design.md D3e]` |

## Lessons
- A real-tree smoke must pin the tree's current correct verdict; asserting a known-fixed defect makes the smoke go green for the wrong reason once the defect is fixed.
- A hardcoded path that works today is a retarget-debt that multiplies silently across N bats; centralise it the first time it appears in two places.
- The delay-floor rule is carried by synthetic fixtures (LD1/LD3/LD6); LD5 was only a tree-state witness and should never have been the rule carrier.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-06-campaign11-client-root.md | kit | 2026-09-06 | pending | 0 |`
