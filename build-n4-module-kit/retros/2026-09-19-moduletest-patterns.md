<!-- review-status: folded -->
# 2026-09-19 · kit · moduletest-patterns

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU expanding moduleTest.md with real test-wb patterns.
**Delta count**: 8

## What happened
moduleTest.md only showed a bare BTestNg + createTestStation. Real test-wb offers a full station
fixture, async assertions, data providers, retry, OS-skip, out-of-process integration, palette
assertions. Cluster TEST-G1..TEST-G8 in the master register.

## Evidence
- Signatures verified against `organized/test/test-wb/decompiled/`: BTestNgStation, TestHelper, BTridiumTestNg, NRetryAnalyzer, RequiresListener, StationRunner. `[ev: code test-wb]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | BTestNgStation full-services fixture | `types/moduleTest.md` | `[ev: code BTestNgStation.java]` |
| Δ2 | waitFor/assertWillBeTrue async | `types/moduleTest.md` | `[ev: code TestHelper.java]` |
| Δ3 | @DataProvider + toDataProviderArray | `types/moduleTest.md` | `[ev: code BTridiumTestNg.java]` |
| Δ4 | NRetryAnalyzer flaky retry | `types/moduleTest.md` | `[ev: code NRetryAnalyzer.java]` |
| Δ5 | @Requires OS-conditional | `types/moduleTest.md` | `[ev: code RequiresListener.java]` |
| Δ6 | StationRunner out-of-process | `types/moduleTest.md` | `[ev: code StationRunner.java]` |
| Δ7 | loadPaletteItem palette assertion | `types/moduleTest.md` | `[ev: code TestHelper.java]` |
| Δ8 | test layout + R6 (7.6.17 + ux Jasmine) cross-ref | `types/moduleTest.md` | `[ev: corpus B1028]` |

## Lessons
- Use BTestNgStation (not bare createTestStation) when the test needs live services; waitFor over Thread.sleep.
- ux JS tests (Jasmine) are a separate seam from rt moduleTest; the 7.6.17 plugin bug bites both.

---
**Status**: FOLDED into `types/moduleTest.md` (2026-09-19).
