<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-23 · kit · r10-dedupe-pins-measured-cuts

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`, unit R10-1), branch
`docs/r10-modular-doctrine` from origin/main 80fef28
**Delta count**: 3

## What happened
The user restated the authoring rule on 2026-10-23: code is always modular and maintainable, with fewer lines but
always with quality, and asked for it as a /build-n4-module delta. Most of it is already kit doctrine, so this
retro first checked what is folded and kept only what is new.

Already folded, referenced here and not restated:
- One shared helper before a second copy, site data in one table, no line cuts by deleting tests or comments,
  a refactor keeps its output with the helper tested first, no machine default paths
  (`retros/2026-10-21-modular-line-efficient-quality.md` Δ1-Δ5).
- New tools in Go, the Go-port criterion, the layered Go lint stack with thin `cmd/` binaries, and bash reusing
  `toolbelt/lib/*` instead of private copies (`retros/2026-10-22-r8-t22-retro-deltas.md` Δ1, Δ8, Δ9).

Two proposed topics were dropped as covered:
- **Thin entry points over a tested core.** The rt BComponent as a thin slot adapter over a Baja-free `step()`
  core is `types/logic.md` § Pure-class extraction (comppan-fase1 L6). The `-ux` servlet as a thin `instanceof`
  adapter over a pure router is `types/dashboard.md` DUX1. Thin `cmd/` and `.sh` wrappers are r8-t22 Δ1 and Δ9.
- **One module per concern for bash glue.** "Reuse `toolbelt/lib/*` instead of a private copy" is r8-t22 Δ9.

What is new comes from two refactors of quality round 9. `report-module.sh` went from 1610 to 885 lines with
byte-identical output (PR #352). The root signing-block regex table moved into one sourced
`toolbelt/lib/signing-block.sh` (PR #360). Both kept the share with a test, both hit a limit on how far a dedupe
may go, and both reported their size in ways that did not agree.

## Evidence
- PR #352 (merge 9213b39): 25 pasted per-row parsers in `report-module.sh` became one shared `ROW_AWK` /
  `member_rows`. The file is 1610 lines at the merge base fe10704 and 885 at 9213b39 (`git show <sha>:<path> | wc -l`).
  A 68-case matrix kept stdout, stderr and exit identical `[ev: retro r9-m3-report]`
- The same file was reported three ways: "~900" in the PR title, 882 in the r9-m3 retro, and 885 at the merged
  tip. No count named the commit it was measured on `[ev: retro r9-m3-report]`
- PR #352 kept the literal `"$TOOLBELT/<lint>.sh"` call sites instead of a table-driven runner, because
  `gen-lint-index` marks a lint Auto only when a non-comment line of `report-module.sh` invokes it literally
  (`toolbelt/gen-lint-index.sh:22`). A generic runner would have flipped the whole Auto column to "no"
  `[ev: retro r9-m3-report]`
- `tests/report-module.bats` RM-row-parser fails when a pasted per-line parser comes back; the 1610-line original
  fails it (28 pasted parsers) `[ev: tests/report-module.bats RM-row-parser]`
- PR #360 (merge 70ab7df): `toolbelt/lib/signing-block.sh` is the one table, sourced by `build.sh:100`,
  `lint-structure.sh:401` and `preflight.sh:327`. `tests/lint-structure.bats` LS15-sync fails when any consumer
  re-inlines a copy of the table `[ev: tests/lint-structure.bats LS15-sync]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A shared helper is pinned by a sync test.** When a dedupe moves copies into one `toolbelt/lib/*.sh` file or one shared function, the same PR adds a bats test that fails if any caller stops using it or a private copy comes back (shape: LS15-sync, RM-row-parser). Why: the rule "never a private copy" is prose; without a pin the next quick fix pastes the copy back and nothing fails. Reviewer check: a PR that creates or widens a shared helper names the test that pins every caller to it, with an OBSERVED mutation (re-inline one copy, the test goes red). | `BUILD-LOOP.md` § 5 kit tooling doctrine | `[ev: tests/lint-structure.bats LS15-sync]` |
| 2 | **A dedupe stops at a call site that a generator or lint reads.** Before repeated call sites are replaced by a table, a loop or a generic runner, grep the kit for a generator, lint or test that matches those call sites literally (`gen-lint-index` reads `"$TOOLBELT/<lint>.sh"` in `report-module.sh`). If one exists, keep the literal call and share everything after it. Why: a table there removes lines and silently breaks a generated index. Reviewer check: a refactor that turns literal invocations into data shows the grep and an unchanged `gen-*.sh --check`. | `BUILD-LOOP.md` § 5 kit tooling doctrine | `[ev: retro r9-m3-report]` |
| 3 | **A claimed line reduction is measured on named commits.** A PR or retro that claims fewer lines states `wc -l` of each touched file at the base commit and at the tip it is merged from, with both SHAs, next to the unchanged-output proof (`BUILD-LOOP.md` § 5, modular-line-efficient-quality Δ4). An estimate ("~900") is not a measurement. Why: one file was reported as ~900, 882 and 885. Reviewer check: the numbers can be reproduced with `git show <sha>:<path> \| wc -l`. | `BUILD-LOOP.md` § 5 (refactor-keeps-output bullet) | `[ev: retro r9-m3-report]` |

All three are folded in this branch (see Status). No owed lint: Δ1 and Δ2 are review checks backed by existing
bats pins, and Δ3 is a PR-body rule.

## Lessons
- Most of a restated rule is usually already folded. Check the folded retros and their landing files first, and
  add only the gaps.
- A sharing rule that no test pins is advisory, however strongly it is worded.
- Line counts drift between the PR title, the retro and the merge. Measure once, on named commits.

---
**Status**: FOLDED 2026-10-23 (branch `docs/r10-modular-doctrine`) — Δ1 `BUILD-LOOP.md` § 5 kit tooling doctrine "Pin every shared helper with a sync test" (`:241`); Δ2 `BUILD-LOOP.md` § 5 kit tooling doctrine "A dedupe stops at a call site that a generator or lint reads" (`:248`); Δ3 `BUILD-LOOP.md` § 5 "A refactor keeps observable output" bullet, line-count paragraph (`:214`). Dropped as covered: thin entry points (`types/logic.md` § Pure-class extraction, `types/dashboard.md` DUX1, r8-t22 Δ1/Δ9) and bash one-module-per-concern (r8-t22 Δ9). INDEX row generated from the header: `| 2026-10-23-r10-dedupe-pins-measured-cuts.md | kit | 2026-10-23 | folded | 3 |`
