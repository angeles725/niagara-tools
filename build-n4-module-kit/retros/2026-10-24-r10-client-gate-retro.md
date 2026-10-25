<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-24 · kit · r10-client-gate-retro

**Session**: quality round 10, landing a chain of 8 rt/ux/wb fix PRs and 2 follow-ups in the private
repository of client A (a single module group with rt, ux and wb parts), kit at origin/main 973d9770
**Delta count**: 5

## What happened
The PRs had to merge "on green CI for the exact head", but client A's repository has no CI at all: no
workflow, no Actions run ever, and its JUnit tests need the proprietary Niagara jars on the classpath, so a
hosted runner cannot build or test it. The coordinator chose a local gate instead. For each exact pushed
head, the gate ran a fresh `mirror-niagara-home.sh` mirror, `build.sh` against that mirror and the
module's own WSL JUnit runner. A PR comment then recorded the head SHA, the build result and the test
counts, and only then did the PR merge.

Running that gate on every head exposed five things the kit does not say yet:

1. The first build of the module's main failed in the wb part with `variable TYPE is already defined`. The
   wb view class declared `TYPE`/`getType()` by hand and had no generated region. `build.sh` runs `clean +
   slotomatic`, and slotomatic added a generated Type region next to the hand-written pair. The module had
   only ever been built without slotomatic, so the duplicate never showed.
2. The second build on the same mirror failed the drift gate (exit 51): "shipped bytes changed but
   vendorVersion is still 1.3". The baseline was not a deployed jar. It was the jar that the FIRST build's
   `:jar` step had copied into the mirror's writable `modules/`.
3. The module's WSL runner listed its suites by hand. Four new suites added by the PRs in this chain were
   never compiled or run by it. A probe test that always fails was skipped by the old runner (202 tests,
   PASS) and run by a discovering runner (292 tests, FAIL). The discovering runner, with 6 commented
   excludes, also picked up 9 older suites that pass in WSL and had never run.
4. One suite that never ran had failed since its first commit. Its 5 assertions used an earlier
   architecture's naming: hyphenated names as the slot, and an id built from the slot plus the plant number.
   The reader has always implemented the current naming: underscore slots, id = lowercase slot with
   `_` -> `-`, and the label read from its own slot. The tests were stale, not the code.
5. After the lint FAILs were cleared, `report-module` changed from ISSUES (exit 50) to INCOMPLETE
   (exit 53). The schema-risk floor had SKIPped (no `.deploy-baseline`) the whole time, but the FAIL
   verdict hid it.

## Evidence
- Duplicate TYPE: the wb compile error `TYPE is already defined` at the hand-written field, right after the
  generated `//endregion ... END BAJA AUTO GENERATED CODE`. The fix kept only the generated region.
  `[ev: client A commit 925d2c0]`
- Mirror baseline: `build.sh: FAIL — <ux part> shipped bytes changed but vendorVersion is still 1.3`. The
  mirror was built with the module's jars excluded, so the only jar in `modules/` was the previous build's
  copy. A fresh mirror per gate run (or `--no-drift-check`) fixed it. `[ev: kit toolbelt/build.sh drift gate
  Δ2 header]`
- Runner list: old runner `OK (202 tests)` with an always-failing probe class present; discovering runner
  `Tests run: 292, Failures: 1` (the probe). Without the probe: 26 suites, 291 tests, then 27 suites,
  328 tests after the stale suite was fixed. `[ev: client A commit 4e762fc]` `[ev: client A commit 0c3cc2e]`
- Stale suite: `git blame` puts all 5 assertions in the initial commit. The test header cites the earlier
  design. The reader's id and plant-number helpers are unchanged since that commit. `[ev: client A commit
  fc37673]`
- Verdict masking: `go/internal/reportmodule/report.go:65-72` sets `INCOMPLETE` when a floor check SKIPs,
  then overwrites it with `ISSUES` when any row FAILs. The summary line names one verdict only.
  `[ev: code go/internal/reportmodule/report.go:65]`
- No CI: `gh workflow list --all` was empty and `actions/runs` reported `total_count` 0. The WSL runner puts
  `<niagara_home>/modules/baja.jar` and other platform jars on the classpath. The kit's `## Client repository
  CI` recipe assumes Baja-free sources. `[ev: kit build-verify.md § Client repository CI]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A `@NiagaraType` class that declares `TYPE`/`getType()` by hand outside the `BEGIN/END BAJA AUTO GENERATED CODE` region fails to compile after `slotomatic` (duplicate `TYPE`). Delete the hand-written pair and keep the generated region. Lint candidate: lint-structure FAIL on a `static final Type TYPE` outside the AUTO region | `build-verify.md` § `## Raw gradle (quick WSL iteration, no deploy)` (slotomatic bullets), `toolbelt/lint-structure.sh` | `[ev: client A commit 925d2c0]` |
| Δ2 | A mirror reused between builds is not a deploy baseline. `:jar` copies each build's jar into the mirror's `modules/`, so the drift gate then compares against the previous local build (false exit 51). Recreate the mirror for every gate run (symlinks only, cheap), or pass `--no-drift-check` when the mirror is not the deploy target | `build-verify.md` § `### Mirror (the fallback for a live production supervisor)` | `[ev: kit toolbelt/build.sh drift gate]` |
| Δ3 | A client's WSL JUnit runner discovers every `*Test.java` under its `srcTest/test` roots, minus a commented exclude list (one reason per class: NRE kernel, live station, BTest). An exclude naming no file, or zero discovered suites, fails the run. A suite no gate runs is not a test: it rots silently (5 assertions stale from day one) | `types/moduleTest.md` § `## What to test in a moduleTest vs pure JUnit` | `[ev: client A commit 4e762fc]` |
| Δ4 | `report-module` shows a floor SKIP even when rows FAIL: e.g. `ISSUES (+INCOMPLETE: schema-risk not run)`, so clearing the FAILs does not reveal a new INCOMPLETE late | `go/internal/reportmodule/report.go` summary; `BUILD-LOOP.md` § `## 5. Verify gate (before "done")` | `[ev: code go/internal/reportmodule/report.go:65]` |
| Δ5 | When a client's build or tests need the proprietary Niagara jars, hosted CI cannot run them. Gate locally instead: on each exact pushed head, run a fresh mirror + `build.sh` + the client's WSL runner, then post a PR comment as the record (head SHA, build result, test counts, mirror source). Merge only after that comment, and re-gate after every push or main merge | `build-verify.md` § `## Client repository CI` | `[ev: kit build-verify.md § Client repository CI]` |

## Lessons
- Run the kit's full build (`clean + slotomatic`) once on a client module before trusting its history. A
  module only ever built without slotomatic can hide a duplicate-TYPE compile error.
- A gate is only as wide as its suite list. Discovery with a stale-checked exclude list turns "added a test"
  into "the gate runs it" without anyone editing the runner.
- A verdict that names one state hides the others. Read the SKIP rows, not just the summary word.
- Stale tests and code regressions look the same in a failing run. `git blame` on the assertion plus the
  naming in the repository's own docs and seed tables tells them apart, without a station.

## Fold (partial, 2026-10-25, branch `docs/r10-fold-tail3`)
- Δ2 → `build-verify.md` § Mirror (a reused mirror is not a deploy baseline); Δ3 → `types/moduleTest.md` § What to test (a discovering WSL runner with a checked exclude list); Δ5 → `build-verify.md` § Client repository CI (local exact-head gate, PR comment as the record).
- Δ4 landed on main in #539; Δ1 (lint) is in flight. The header stays `pending` until Δ1 merges.

## Owed
- Δ1 FOLDED (branch `feat/r10-lint-type-dup`): the `build-verify.md` § Raw gradle bullet, and as code the
  `type-outside-region` FAIL sub-check of `toolbelt/lint-slotomatic-markers.sh`
  (`go/internal/lint/rules/slotomaticmarkers`, not lint-structure: it reads the same region markers).
- Δ4's summary change is a code change in `go/internal/reportmodule`; it needs its own RED-first PR.
