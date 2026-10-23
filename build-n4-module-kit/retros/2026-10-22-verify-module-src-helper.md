<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · verify-module-src-helper

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T8d), branch
`r8/t8d-verify-src` from origin/main 829675f
**Delta count**: 1

## What happened
`toolbelt/verify-module.sh` opened each of its eight src-walking checks with the same three lines (SKIP "no --src",
`pd=$(profile_dir ...)`, SKIP "no <pd>/src"), three more checks repeated the first two, and the eight walking
checks each carried the same D9b `find "$pd/src" -type d -name '.*' -prune -o -name '*.java' -print`. They now
call `src_or_skip` / `src_dir_or_skip <check> <jar> || return 0` (which set the caller's local `pd`) and
`java_files "$pd/src"`. Every row and exit code stays byte-identical; the script loses 4 net lines.

The shared fail-closed walk `toolbelt/lib/scan-files.sh` was not adopted: it sorts its list with `LC_ALL=C`
and lists only regular files, while verify-module walks in find (directory) order. Sorting would change which
file the `facets` check reports first (it stops at the first hit) and the order of WARN rows, so the walk keeps
its find call inside `java_files` and stays fail-open (find errors are still not checked), exactly as before.

## Evidence
- Golden: the full report of 12 runs (4 jars: -rt with three offending files plus a dot-dir copy, -ux, -wb,
  -rt without a profile dir; × no flag, `--src`, `--src --strict`) is byte-identical before and after
  (`cmp` clean, 288 lines, 65 WARN rows, exits 8×0 and 4×1) `[ev: T8d session golden run]`
- New `tests/verify-module.bats` VSRC1 (exact "no --src" SKIP rows for all 11 source checks), VSRC2 (exact
  "no <pd>/src" rows for the 8 walking checks, plus the typecount/phantom-dep/moduletest rows) and VSRC3 (a
  `.deploy-baseline/` copy that trips all 8 walking checks is pruned) are characterization tests: green before
  and after. Named mutations make them fail: reword the "no --src" detail in `src_or_skip` → VSRC1 fails;
  drop the dot-dir prune from `java_files` (or from the eight inline finds before the refactor) → VSRC3 fails
  with FAIL facets plus a WARN from each of the other 7 walking checks `[ev: tests/verify-module.bats VSRC1-VSRC3]`
- Before VSRC3 only the facets check had a dot-dir test (`tests/facets-lint.bats` F8); the other seven walking
  checks claimed "D9b: dot-dirs pruned" in a comment with no test `[ev: tests/facets-lint.bats F8]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **Make the verify-module source walk fail-closed in its own work unit.** `java_files` keeps verify-module's fail-open find, so an unreadable directory under `<profile>/src` is skipped and the check reports PASS. A follow-up switches `java_files` to `lib/scan-files.sh` (or checks find's status) and turns a walk error into a FAIL row for that check (exit 1, like other check failures) or an environment exit 3, accepting the sorted file order as a documented output change with updated tests. Why: the same fail-open walk was already fixed for the lints (issue #226) and now has one place to fix here. | `toolbelt/verify-module.sh` `java_files`; `tests/verify-module.bats` | `[ev: toolbelt/lib/scan-files.sh header, issue #226]` |

Not applied; a fold review decides it and flips this retro.

## Lessons
- A shared helper is only a drop-in when its output order matches; a sorted walk is a behaviour change for a
  check that stops at the first hit.
- A helper that sets the caller's `local` variable (bash dynamic scope) keeps the `|| return 0` call sites
  one line each without a subshell.

---
**Status**: FOLDED (quality round 8 pointer unit, `docs/r8-pointer-lines`) — Δ1 6f27bf2 (PR #354) `java_files` on `lib/scan-files.sh`, fail closed + `tests/verify-module.bats` VM-walk-locked; tokens in `ledger.d/2026-10-23-r8-pointer-lines.md`. INDEX row generated from the header: `| 2026-10-22-verify-module-src-helper.md | kit | 2026-10-22 | folded | 1 |`
