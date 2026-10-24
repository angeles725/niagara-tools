<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-23 · kit · go-lint-structure

**Session**: niagara-research quality round 8 unit E3d (Ayudante2 lint-engine owner), branch `r8/e3-lint-structure-go` from origin/main 94a6a8d
**Delta count**: 3

## What happened
`toolbelt/lint-structure.sh` (449 lines of bash, find/grep plus a private gawk `blank()` for L15) is now the thin
`n4lint lint-structure` wrapper over `go/internal/lint/rules/structure`, header, usage and `# Mutation:` pins kept.
lint-structure is not a source lint: it reads profile directories (an empty `-wb` directory is a finding), the
gradle.properties of every parent up to the `.git` sentinel and those of sibling modules, so it fits neither
`lint.Rule` nor `lint.TreeRule`. Two small engine features landed first, each with a table test: `lint.RootRule`
(`Spec.Root`: the runner walks nothing for it, `Select`/`ContinueOnWalkError`/`EmptyError` do not apply, an error
fails closed; `n4lint --all` leaves it out because its root is a module root) and `Spec.BarePathRow` (the
`<path>  L<n>: <reason>` row, no colon). L15 reuses `javasrc.Strip` (the mb_strip port) and `javasrc.BlankLiterals`,
which is the script's private `blank()` line for line; the rule's file listings reuse `lint.Walk`.

Declared differences of the port (byte identity otherwise): (a) a file or directory the lint cannot read now
prints `lint-structure: cannot scan|list ...` and exits 3 with the rows still printed, where bash `grep 2>/dev/null`
read it as no match and `find` printed its error and exited by the rows (the unreadable root build.gradle.kts keeps
its visible L15 WARN); (b) `*-worktrees` sub-directories are pruned like every engine walk (bash reported each
worktree's gradle.properties again); (c) sibling `model/` directories are reported in name order, not find's readdir
order. LS15-sync now pins preflight.sh's SAMPLE_SIGN_ERE against the Go `signBlock`.

Truth fixes after byte identity (R8-D14, truth audit 2026-10-22 T7), each RED first: L10 reads a Properties line
(indent, `=`/`:`/blanks, drive path incl. escaped `C\:`, POSIX path, UNC share) instead of only `key=X:\`; L10 and
L14 rows print `<path>:<line>`, one per offending line, instead of the rule ID where the line belongs; the L10 text
no longer says "tracked" (nothing checks git, the lint is VCS-free by contract and three client trees are not
repositories; the check holds either way because build.sh passes host paths with -P).

## Evidence
- RED engine: `root_test.go` did not compile before `Spec.Root` (`undefined: RootRule`) and `TestCommandAllSkipsRootRules` got the root rule's row under `--all` before the skip `[ev: go/internal/lint/root_test.go TestCommandAllSkipsRootRules]`
- Goldens: 42 cases recorded from the bash lint (26 runs of `tests/lint-structure.bats`, 16 edge trees), each replayed byte-identically against bash on a re-materialised tree before the port existed `[ev: go/internal/lint/rules/structure/structure_test.go TestGolden]`
- Byte identity old vs new (stdout, stderr, exit): 18 Cliente module roots (Honeywell, Juarez, LLM, Leon-Guanjuato, panccadia-leon, ford, _salvage), 28 kit fixtures (lint-structure and report-module), 20 argument shapes (none, empty, missing, a file, `--strict`, extra args, `.`, `./`, trailing `/` and `//`, relative, all of `Cliente`); 65 of 66 identical, the 66th being all of `Cliente` (difference b, 42 worktree rows); time 4.41 s bash vs 1.13 s Go over the 66 runs
- RED T7 shapes: `TestHostPathShapes` failed 8 shapes, bats LS10-shapes failed on M1 `[ev: tests/lint-structure.bats LS10-shapes]`
- RED T7 lines: `TestRowLines` failed on the bare path `[ev: go/internal/lint/rules/structure/structure_test.go TestRowLines]`; report-module relays `gradle.properties:2  L10:` (was `gradle.properties:  L10:`) `[ev: tests/report-module.bats RM19-line]`
- Real trees after the truth fixes: the shapes add no Cliente row (every real pin is `key=C:\\...`); the line fix turns each L10 file row into one row per key (panccadia-leon Paccadia gradle.properties:5/8/12, chihuahua :18/21/25) and L14 into settings.gradle.kts:84

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A lint over a module layout (directories, parents, siblings) ports as a `lint.RootRule`; a Selection widened past source files is not a substitute | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/root_test.go TestRunRootRule]` |
| Δ2 | `find <root> \( -type d -name '.*' -prune \)` prunes the start point too, so `lint-structure.sh .` skips L10 under the root; audit every toolbelt find with that shape and drop the prune on the start point (`-mindepth 1` or a path test) | `build-n4-module-kit/toolbelt/lib/scan-files.sh` | `[ev: go/internal/lint/rules/structure/scan_test.go TestDotName]` |
| Δ3 | A row whose site is a line prints `<path>:<line>`; a rule ID in the site column is a T7-class defect (L1, L2, L7 still print the bare path) | `go/README.md` § Adding a lint rule, step 5 | `[ev: go/internal/lint/rules/structure/structure_test.go TestRowLines]` |

## Lessons
- Recording goldens through the bats suite needs a base-directory rule per tree shape (fixture dir, mktemp root,
  BATS_TEST_TMPDIR); empty directories (`.git` sentinels) and symlinks are not representable, so they get Go tests.
- Replaying every recorded golden against the bash lint on a re-materialised tree catches recorder bugs before a
  single line of the port is written.
- The engine's fail-closed contract and a bash `2>/dev/null` disagree on every unreadable input; the port takes the
  engine's side and lists it as a declared difference instead of copying the fail-open.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-go-infra`) — Δ1 `BUILD-LOOP.md` § 5 port rule (`lint.RootRule`); Δ2 `lib/scan-files.sh` already uses `-mindepth 1`, the Go `dotName` quirk is owed (fail-closed listing line); Δ3 already folded (truth-audit Δ3 row-site line), token added.
