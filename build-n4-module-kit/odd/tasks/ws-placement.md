# Feature: wire-sheet placement for programmatically created components (ws-placement)

## Objective
Every component a module creates in code gets a deterministic wire-sheet position (`wsAnnotation`), so the
Workbench wire sheet shows a readable grid instead of a pile of overlapping glyphs.

## Problem / why
- Without `wsAnnotation` the wire sheet hands out an in-memory diagonal stagger (+4, wrap to 2, width 8), so
  every code-created sibling overlaps (corpus B1172; `WsController.java:398-405`).
- chihuahua already places its children on a grid (`BChiUpMonitor.java:231-236`), but adds the slot without
  READONLY and without an exists-check (relies on a try/catch).
- DashboardFord `doSyncInventory` creates 88 nodes with no annotation (`BFordDashboardService.java:199`).
- The kit has no rule, helper or lint for it.

## Behavior decisions (user, 2026-10-14)
| # | Question | Answer |
|---|----------|--------|
| WS1 | Should code-created components get wire-sheet coordinates? | Yes, a grid layout. |
| WS2 | A component the user already moved by hand? | Never moved: add the annotation only when the slot is absent. |
| WS3 | Slot shape | Same as Workbench: `add("wsAnnotation", anno, Flags.READONLY)` when absent (B1172 §1, `WsController.java:189-200`). |

## Scope
Kit only (this repo). The DashboardFord fix is tracked in the client repo
(`Cliente/ford/dashboard/odd/tasks/dashboardford-web-parity.md`, task T5d).

## Tasks
- [x] W1 Doctrine: METHODOLOGY.md rule + types/dashboard.md (and logic-authoring.md where components are created)
      section citing B1172. Route: delegated writer. Commit 2a45e1c.
- [x] W2 Shared helper: a Baja-free grid calculator (`templates/` reference source) + its pure test, plus the
      add-if-absent READONLY recipe. `templates/ws-placement/src/com/angeles/wsplacement/WsGrid.java` +
      `srcTest/.../WsGridTest.java`. Route: delegated writer. Commit 2a45e1c.
- [x] W3 Lint `toolbelt/lint-ws-placement.sh` (WARN, `--strict` -> exit 1, exit 3 usage) + bats with a
      `# Mutation:` pin, wired into report-module.sh, INDEX regenerated. Route: delegated writer. Commit e268d52.
- [x] W4 Retro + BUILD-STATE/INDEX per the pre-push retro gate. `retros/2026-10-14-ws-placement.md` (4Δ, pending).
      Route: delegated writer. Commit: the W4 docs commit on this branch.

## Checks
bats for the new lint (and report-module if touched); shellcheck 0.10.0; gen-lint-index.sh --check;
lint-guard-pins.sh; run the lint on DashboardFord (must WARN at BFordDashboardService.java:199) and chihuahua.

## Progress
- 2026-10-14 worktree `niagara-tools-worktrees/ws-placement`, branch `feat/ws-placement` from origin/main 5179bf2.
  Route: delegated writer (W1-W4 touch 2+ non-trivial files).
- 2026-10-14 W1-W4 done by the delegated writer. Evidence (all observed):
  - Pure test: `PATH=/usr/lib/jvm/java-8-openjdk-amd64/bin:$PATH toolbelt/run-pure-test.sh templates/ws-placement
    com.angeles.wsplacement.WsGridTest` -> RED `Tests run: 6, Failures: 6` on a compiling stub, then `OK (6 tests)`.
  - `bats tests/lint-ws-placement.bats` 13/13 ok; `bats -j 6 tests/report-module.bats` 63/63 ok (RM56, RM57 new);
    kit-links 12/12, gen-lint-index 20/20, guard-pins 7/7, build-retro-sync and c11-close 0 not ok.
  - Mutations (each applied, suite run, restored): WSP1, WSP1-local, WSP-guard, WSP-scope, WSP-list, WSP-value,
    WSP-awkfail, WSP-finderr each flip their pinned test; the numeric-first-argument rule flips WSP-list too.
  - `shellcheck` (0.11.0 locally; CI pins 0.10.0) on the lint + bats: exit 0. `gen-lint-index.sh --check`: exit 0.
    `lint-guard-pins.sh --strict .`: exit 0 (8 WSP pins MATCH). sweep-build-state exit 0; sweep-fold-audit --strict
    exit 0.
  - Real trees: DashboardFord-rt/src -> one WARN at `BFordDashboardService.java:199` (`add(op.path.substring(slash +
    1), create(...)) in doSyncInventory()`), exit 0; chihuahua-rt/src -> 0 rows, exit 0 (guard-dropped mutant shows
    the three monitors, so the clean result comes from the guard).

## Next step
Parent: native review of the work-unit commits, then push/PR (user decision). DashboardFord fix stays in the client
repo (T5d). Retro fold review flips `retros/2026-10-14-ws-placement.md`.
