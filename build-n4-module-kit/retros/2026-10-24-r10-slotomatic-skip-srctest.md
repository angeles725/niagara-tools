<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-slotomatic-skip-srctest

**Session**: niagara-research quality round 10 (Ayudante2), branch `fix/slotomatic-skip-srctest` from origin/main
**Delta count**: 1

## What happened
A CompPan build exited 34 after the r10-slotomatic-skip-fail-open fix although its module jar was sound: Slot-o-Matic
also walks `srcTest/`, logs `Could not process` for a plain `B*Test.java` with no metadata block, and
`lib/slotomatic-skip.sh` counted that as a stale-jar FAIL. Test sources build only the `<part>Test.jar`
(`types/moduleTest.md`), never the module jar, so their skip cannot ship stale generated slots.

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | `slotomatic_skip_scan` ignores a skipped path under `srcTest/` (either separator); a `src/` skip in the same log still FAILs. | `toolbelt/lib/slotomatic-skip.sh` + `tests/slotomatic-skip.bats` SK-srctest-only, SK-srctest-mixed | `[ev: retro r10-slotomatic-skip-fail-open Δ1]` |
