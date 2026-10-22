<!-- review-status: folded -->
# 2026-10-22 · kit · ws-row-spacing

**Session**: niagara-research quality round 8, work unit T9d-2 (issue #259), branch `fix/r8-t9d-ws-spacing` from origin/main 3a58392
**Delta count**: 2

## What happened
The first consumer of the ws-placement helper (DashboardFord T5d, client commit d298b35) found that
`WsGrid.standard()`'s fixed 10-wixel row spacing is shorter than a tall facade glyph: `BVavPanel` shows 22 SUMMARY
slot bars, so its glyph is 2 + 22 + 1 = 25 wixels and every row overlaps the next; the module hand-picked 28. The
kit guidance only said "pick `rowSpacing` above the tallest glyph", with no helper to derive it. This unit adds
`WsGrid.rowSpacingFor(n)` / `WsGrid.forSummaryCount(n)` and states the derived rule where the fixed 10 was taught.

## Evidence
- Glyph height = 2 + visible SUMMARY slot bars + 1 `[ev: corpus B1194]`; `BVavPanel` 22 SUMMARY -> 25 wixels, module
  spacing 28 `[ev: issue #259]`
- Formula `max(10, 2 + n + 1 + 3)`: the 3-wixel gap is the one `standard()` already leaves under a 4-bar glyph
  (10 - 7), so `n <= 4` returns exactly `standard()` and `n = 22` returns the consumer's 28 `[ev: templates/ws-placement/srcTest/com/angeles/wsplacement/WsGridTest.java]`
- `WsGridTest` RED `Tests run: 8, Failures: 2` (`expected:<28> but was:<10>`, `expected:<11> but was:<10>`) on a
  compiling stub, GREEN `OK (8 tests)` under JDK 8 via `toolbelt/run-pure-test.sh` `[ev: templates/ws-placement/srcTest/com/angeles/wsplacement/WsGridTest.java]`
- No lint rule: the SUMMARY count lives in the created type's slotomatic annotations (another class, often chosen at
  runtime by `create(type)`), so `lint-ws-placement.sh`'s one-method scope cannot see it; the module's pure test pins
  it instead `[ev: toolbelt/lint-ws-placement.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Row spacing is derived from the largest SUMMARY count among the placed children (`WsGrid.forSummaryCount(n)`, `max(10, 2 + n + 1 + 3)`), pinned by the module's pure test against the facade's SUMMARY slots; `standard()` only for children of up to 4 SUMMARY slots | `types/logic-authoring.md` §Wire-sheet placement; `templates/ws-placement/` | `[ev: issue #259]` |
| Δ2 | The dashboard facade rule for code-built trees says the same: never the fixed 10 for a facade with more than 4 SUMMARY slots | `types/dashboard.md` § Facade contract | `[ev: issue #259]` |

## Lessons
- A layout constant derived from one sample (chihuahua's small glyphs) needs its derivation in code, not a number:
  expose the formula so the next consumer computes its own value.
- A rule whose input lives in another class (the SUMMARY count of a created type) belongs in a module pure test, not a
  single-method lint.

---
**Status**: FOLDED (T9d-2, same unit) — Δ1 `types/logic-authoring.md` §Wire-sheet placement + `templates/ws-placement/` `WsGrid.rowSpacingFor`/`forSummaryCount`, Δ2 `types/dashboard.md` § Facade contract.
