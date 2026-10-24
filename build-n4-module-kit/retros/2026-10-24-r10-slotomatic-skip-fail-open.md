<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-slotomatic-skip-fail-open

**Session**: niagara-research quality round 10 (Ayudante1), branch `fix/r10-build-sh-slotomatic-skip` from
origin/main 45ddb3b5
**Delta count**: 1

## What happened
A PANCCADIA DashboardPan-rt build passed although Slot-o-Matic had skipped `BHmiStatus.java`: its generated
region carried a stamp count other than one, Slot-o-Matic threw "Found multiple metadata blocks", logged
`Could not process <path>.java: <msg>` at WARNING level, listed the file under `The following files failed to
compile:`, and its gradle task ended BUILD SUCCESSFUL. The old generated code still compiled, so the jar shipped
stale slots. `toolbelt/build.sh` looked for a skip only in its gradle-FAILURE branch (the `have you run
slot-o-matic?` hint) and `scripts/ng-deploy.sh` `run_slotomatic` checked only the exit code: both failed open.

Now one sourced detector, `toolbelt/lib/slotomatic-skip.sh` `slotomatic_skip_scan <log>`, prints one
`FAIL slotomatic-skip <path> (<reason>)` row per skipped file (both log forms, each file once, one awk, a missing
log fails closed). build.sh runs it on the captured log in the success branch (exit 34, pre-build `modules/` jar
restored like a drift FAIL, gate not run) and in the exit-30 hint; ng-deploy tees the slotomatic output and dies 15.

## Evidence
- Slot-o-Matic source: devkit-wb vineflower `JavaUnit.java:265-272` (stamp count != 1 throws),
  `Compiler.java:240-246` (logs, counts, returns), `Slotomatic.java:148-158` (warning block, one
  `"  " + abs path + ": " + message` line per file) `[ev: sources/decompiled/devkit-wb/com/tridium/slottool/Slotomatic.java:156-157]`
- Real scratch runs on the BHmiStatus case (2026-10-24, coordinator): both lines print in a default run and the
  task ends BUILD SUCCESSFUL `[ev: R10 coordinator brief]`
- RED on 45ddb3b5: SK-* (8, helper absent), BS-slotomatic-skip-on-success and BS-slotomatic-skip-restore (exit 0,
  not 34), T19b (exit 0, built), BS-slotomatic-hint/-win (new row format); GREEN after. Named mutations (success
  call removed, block-form branch removed, ng-deploy call removed) each turn their tests red `[ev: tests/slotomatic-skip.bats, tests/build-sh.bats, tests/ng-deploy.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A Slot-o-Matic skip is a build FAIL whether or not compileJava passes.** A skipped file with an existing (stale) region still compiles, so the gradle exit code proves nothing; every kit entry point that runs slotomatic scans its log through `toolbelt/lib/slotomatic-skip.sh` (never a copy) and fails on a row. Recovery: delete the region and re-run slotomatic. | `METHODOLOGY.md` K15; `build-verify.md` build.sh exits; `BUILD-LOOP.md` § build recipes | `[ev: tests/build-sh.bats BS-slotomatic-skip-on-success]` |

Folded in the same unit (below).

## Lessons
- A WARNING-level tool failure inside a passing gradle task is invisible to an exit-code gate: scan the log.
- The failure-branch hint hid the gap: it looked like coverage, but only for the case where compileJava fails.

---
**Status**: FOLDED 2026-10-24 (branch `fix/r10-build-sh-slotomatic-skip`; log in `ledger.d/2026-10-24-r10-build-sh-slotomatic-skip.md`) — Δ1 `METHODOLOGY.md` K15 (`:169`, stale-region case + the shared detector), `build-verify.md` build.sh exits (`:17`, exit 34), `BUILD-LOOP.md` "Slot-o-Matic skip (exit 34)" recipe (`:153`).
