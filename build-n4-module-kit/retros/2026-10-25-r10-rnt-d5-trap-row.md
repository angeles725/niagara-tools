<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-d5-trap-row

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, the gradle-root (D5) FAIL row
names the plugin module-include trap, branch `r10/rnt-d5-row` from 817860d9 (origin/main)
**Delta count**: 1

## What happened
Live boots of a client root on 2026-10-25 showed the gradle-root gate firing only on runs whose test sources had
changed: the plugin's moduleTest compile had appended the test types to the production `module-include.xml`. The
row said only `gradle root changed by the build: <paths>`, so the operator had to know the trap from
`types/moduleTest.md`. Each rule got a fake-runner RED first on generic `ModuleA` fixtures; no station boot was run:
1. When a changed path ends in `/module-include.xml`, the FAIL row goes on to name the cause (niagara gradle plugin
   7.3.40 and 7.6.22, observed, append the test types to the PRODUCTION file in `:<part>:moduleTestJar` or the
   moduleTest compile when the test sources changed), `--restore-gradle-root`, and the doc note.
2. With `--restore-gradle-root` the restore runs first with its rows held back, so the FAIL row can say whether it
   checked a `module-include.xml` out ("restored the ones that were clean before", or "did not restore it"); the
   restore rows then follow it, in the old order.
3. Any other changed file keeps the generic row. The gate is unchanged: the run stays FAIL, exit 1.

## Evidence
- the REDs on 817860d9: `TestRunGradleRootModuleIncludeNamesTrap` and both `TestRunGradleRootModuleIncludeRestoredTrap`
  subtests failed; `TestRunGradleRootOtherFileKeepsGenericRow` passed on the old code (it pins the generic row)
  `[ev: go/internal/niagaratest/run_gradleroot_test.go]`
- the hint `[ev: go/internal/niagaratest/gradleroot.go]`
- mutation: returning "" from the hint always turned the trap tests and `TestRunGradleRootRestore` red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The `gradle root changed by the build` FAIL row of `run-niagara-test run` names the plugin-version trap when a `*/module-include.xml` changed, points at `--restore-gradle-root` and the trap note, and says whether the restore checked it out; other files keep the generic row, the run stays FAIL | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/gradleroot.go]` |

## Lessons
- A FAIL row for a known recurring cause should name that cause and its fix, not only the symptom.

## Fold
- The row is code in the branch that wrote this retro (`go/internal/niagaratest/gradleroot.go`).
- Δ1 is cited in `types/moduleTest.md` § Running from WSL (the source-tree bullet).

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-d5-row`; log in `ledger.d/2026-10-25-r10-rnt-d5-trap-row.md`) — Δ1 in `types/moduleTest.md` § Running from WSL.
