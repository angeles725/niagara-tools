<!-- review-status: pending -->
# 2026-10-14 · kit · ws-placement

**Session**: feature `odd/tasks/ws-placement.md` (W1-W4, branch `feat/ws-placement` from origin/main 5179bf2)
**Delta count**: 4

## What happened
DashboardFord `doSyncInventory` creates 88 inventory nodes with `parent.add(name, create(type))` and gives none of them
a wire-sheet position, so Workbench hands every one its in-memory diagonal stagger and the integrator opens the
facade to a pile of overlapping glyphs. chihuahua already placed its children on a grid, but added the slot without
`Flags.READONLY` and without an exists-check. The kit had no rule, helper or check for it. The user decided the
behavior (WS1 grid, WS2 never move a placed component, WS3 the Workbench slot shape) and this feature added the
doctrine, a Baja-free grid helper with a pure test, and an advisory lint wired into `report-module.sh`.

## Evidence
- Workbench adds `wsAnnotation` only when absent, with flags `1` = READONLY, else sets it (`WsController.java:189-200`);
  an unannotated component gets `+4`, wrap 2, width 8 (`WsController.java:398-405`); the sheet clamps to BWsOptions
  600 x 600 `[ev: corpus B1172]`
- Defect target: `BFordDashboardService.java:199` — `lint-ws-placement.sh` on DashboardFord-rt/src prints exactly one
  row, `BFordDashboardService.java:199  add(op.path.substring(slash + 1), create(...)) in doSyncInventory() never sets a
  wsAnnotation`, exit 0 `[ev: e268d52]`
- chihuahua-rt/src: 0 rows; with the guard dropped the same tree shows the three monitors (`BChiUpMonitor.java:212`,
  `BChiCarcamoMonitor.java:123`, `BChiDataloggerMonitor.java:122`), so the clean result comes from the guard, not from
  a missed shape `[ev: e268d52]`
- `WsGridTest` RED 6/6 on a compiling stub, GREEN 6/6 under JDK 8 via `toolbelt/run-pure-test.sh` `[ev: 2a45e1c]`
- WSP-list first mutation run did NOT bite: the list fixture's `c.add("anno", new BWsAnnotation())` line mentioned
  `BWsAnnotation`, which guards the whole method, so the fixture was vacuous; split into one shape per method, it bites
  `[ev: e268d52]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Common checklist item: every code-created component gets a grid `wsAnnotation`, add if absent with READONLY, never re-placed, inside 600 x 600 | `METHODOLOGY.md` § rt (components) | `[ev: corpus B1172]` |
| 2 | Recipe section with the add-if-absent snippet, stable-index rule and row-spacing rule; dashboard facade rule for trees built from code | `types/logic-authoring.md` §Wire-sheet placement; `types/dashboard.md` § Facade contract | `[ev: corpus B1172]` |
| 3 | Baja-free `WsGrid` helper + pure test | `templates/ws-placement/` | `[ev: 2a45e1c]` |
| 4 | Advisory `lint-ws-placement.sh` (8 named mutations) wired into `report-module.sh` as an advisory member | `toolbelt/lint-ws-placement.sh`, `toolbelt/report-module.sh` | `[ev: e268d52]` |

All four are applied on `feat/ws-placement` (e268d52, 2a45e1c) and cite `[ev: retro ws-placement]`; the fold review
flips this retro.

## Lessons
- A guard that silences a whole method makes any fixture method that also carries the guard word vacuous: keep one
  shape per fixture method and prove each negative fixture with its mutation.
- Prove a clean real-tree result by re-running it with the guard dropped; zero rows alone cannot tell a guarded shape
  from an unseen one.
- Index generated layout by the stable data row, not the add order, so a re-sync that recreates one node does not
  stack it on a sibling.
- The DashboardFord fix itself stays in the client repo (`dashboardford-web-parity.md` T5d); the kit only carries the
  rule, helper and check.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-14-ws-placement.md | kit | 2026-10-14 | pending | 4 |`
