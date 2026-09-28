<!-- review-status: pending -->
# 2026-09-28 · kit · decision-logic-decomposition

**Session**: PANCCADIA persistent-config/HOA campaign (Cliente/panccadia-leon, feat/panccadia-persistent-config-hoa); user question on how large code blocks should be split
**Delta count**: 3

## What happened
On 2026-09-28 11:18 the live PANCCADIA station (CompPan <= 2.6.1) stopped compressor 1 (AUTO, running,
4 rooms calling) when the operator set compressor 2 to OFF. The defect sat inside the single long
`CompressorControl.step()`: one count (`available`) excluded MODE_OFF units while the adjacent count
(`onCount`) still included the unit about to be forced OFF, so `onCount(2) > target(min(3,1)=1)` shed the
most-hours unit (C1) via `pickMostHoursOn`, which also did not exclude OFF/HAND units. The existing kit
rule "extract a pure class with one step()" was followed, but the pure `step()` itself grew into one
monolithic method with numbered comment sections, so two counts with different membership rules lived a
few lines apart unnamed. Measured file sizes: BCompressorControl 4,421 lines (2,482 slotomatic-generated),
BRoomPanel 3,197 (2,617 generated), BEvaporatorUnit 2,252 (846 generated, ~1,400 hand-written mixing
defrost, drip, restart sequencing, freeze-stat and HOA); the pure CompressorControl is 1,118 lines.
Tridium reference point: kitControl splits one function per component (largest kitControl class
BElectricalDemandLimit, 2,192 lines). USER DECISION (2026-09-28): existing, working modules (PANCCADIA
ColdRoomPan/CompPan/DashboardPan) keep their current structure — no structural refactor, to avoid
introducing regressions; these deltas apply to NEW modules only.

## Evidence
- Pre-fix count mismatch: `CompressorControl.java` at d0d0c7b lines 376-381 (`available` excludes outOfService; `onCount` counts raw `cmd[]`), 453 (`target = Math.min(target, available)`), 488-490 (`onCount > target` -> `pickMostHoursOn`), 646-656 (`pickMostHoursOn` has no mode filter) `[ev: d0d0c7b]`
- Fix introduced a separately named count `stagingOnCount` (line 833) and a mode filter in `pickMostHoursOn` (line 1032) `[ev: c28b552]`
- Live audit: AuditHistory 11:18:54 `Condensadoras/comp2Mode 0.00 -> 2.00 API`; live hours C1 81.7 / C2 54.1; minOff PT3M `[ev: live oBIX read 2026-09-28 11:30]`
- File-size measurement (wc -l + slotomatic region count) on Cliente/panccadia-leon `[ev: 4bafabd]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | NEW modules: the pure core's `step()` is an orchestrator of named phase methods (e.g. `computeAvailability()` → `computeTarget()` → `countRunningForStaging()` → `selectUnitToStart/Stop()` → `applyHoaOverrides()` → `applySafetyEnvelope()`); every count used in a comparison is produced by ONE named method whose javadoc states its membership rule (AUTO/HAND/OFF/locked/about-to-stop); quantities compared against each other use the same membership rule or the difference is named; every `pick*` selection helper enforces its own eligibility filter (mode, outOfService, min-on/off). | `types/logic.md` § `Pure-class extraction — test the decision before you wire it` | `[ev: d0d0c7b]` |
| Δ2 | NEW modules: decompose by concern from the first version — a BComponent whose hand-written body (excluding slotomatic regions) exceeds ~800 lines or mixes 3+ concerns splits each concern into its own pure class (and a child component per concern where it owns slots, per the existing Composition rule); a pure method over ~80 lines is a review smell. Advisory lint `lint-size.sh` (WARN-only): hand-written lines per class excluding `BEGIN/END BAJA AUTO GENERATED CODE`, longest method per pure class. | `types/logic.md` § `Composition & organization` + `METHODOLOGY.md` § `Conformance rules — lintable vs advisory` | `[ev: 4bafabd]` |
| Δ3 | EXISTING working/deployed modules: do NOT restructure (no split/move/rename of working control code) just to meet Δ1/Δ2; fix defects in place with targeted tests. Only if a restructure is ever explicitly requested, first land a safety-invariant suite over the pure core proven to bite (pre-fix commit or documented mutation) and require it green unchanged after the move. | `METHODOLOGY.md` § `Schema / upgrade safety` | `[ev: odd/tasks/panccadia-persistent-config-hoa.md T2b]` |

## Lessons
- Extracting a pure class is necessary but not sufficient: a monolithic pure `step()` hides membership mismatches between adjacent counts.
- Name every count with its membership rule; compare only quantities built from the same rule.
- Selection helpers must enforce eligibility themselves, not rely on callers.
- Measure hand-written lines (exclude slotomatic regions) when judging decomposition.
- Working deployed modules are not refactored for structure; structure rules apply to new builds.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-28-decision-logic-decomposition.md | kit | 2026-09-28 | pending | 3 |`
