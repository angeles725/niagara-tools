<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · symlink-root-walk

**Session**: niagara-tools quality round 8 T8k5, branch `r8/t8k5-symlink-root` from origin/main 0018c74 (issue #329)
**Delta count**: 1

## What happened
Issue #329: every kit lint passed clean on a root argument that is a symlink to a directory. `lib/scan-files.sh`
ran `find "$root" -mindepth 1 …` with the default `-P`, which does not descend into a symlinked root, so the lint
listed 0 files and exited 0. The same root with a trailing slash was scanned. The Go walkers had the same gap on
purpose: `go/internal/lint/walk.go` `Walk` and `go/internal/jarversion/walk.go` `ListJars` called `os.Lstat` on the
root and documented "a symlinked root is not followed (find -P)". Every walk over a lint root now follows the root
like `find -H` and still never follows a symlink below it: `find -H` in `lib/scan-files.sh` (which also serves
`lib/awk-lint.sh`) and in the scripts that run their own find over the root argument (lint-servlet, lint-structure,
split-package-check, slot-coverage, lint-write-path, report-module, commissioning-verify, preflight, build.sh,
triage-console), and `os.Stat` on the root in both Go walkers. Paths keep the root as given. A dangling root
symlink or a root symlink to a file was already fail-closed by the `[ -d ]` / `os.Stat` argument checks (exit 3
"not a directory"), because `find -H` alone lists nothing and exits 0 on both; `Walk` now also returns a
`*WalkError` for a root it cannot stat.

## Evidence
- RED on 0018c74: `tests/scan-files.bats` SF10, SF11; `tests/lint-no-system-out.bats` NSO-symroot (exit 0, no row);
  `tests/lint-null-context-write.bats` NCW-symroot (no WARN row); `go test` TestWalkOrderAndSelection (symlinked
  root, inner link as root, dangling root) and TestListJars (symlinked root) `[ev: tests/scan-files.bats SF10]`
- SF12 (dangling root symlink, root symlink to a file) was green before and after: it pins the existing
  fail-closed check `[ev: tests/scan-files.bats SF12]`
- Mutation: `find -L` in scan_files fails SF11; plain `find` fails SF10 and NSO-symroot; `os.Lstat` in `Walk`
  fails TestWalkOrderAndSelection and NCW-symroot `[ev: tests/scan-files.bats SF11]`
- Byte identity: 9 lints (null-context-write, timers, bundled-jar-class-version, no-system-out, delays,
  driver-scaffold, structure, servlet, split-package-check), old vs new, on Cliente/{Honeywell,panccadia-leon,
  Leon-Guanjuato} and tests/fixtures as plain directories: stdout, stderr and exit code identical
  `[ev: issue #329]`
- Symlinked root to panccadia-leon: timers 0 → 9 rows, delays 0 → 7, servlet 0 → 3, structure 3 → 6 (the three
  L10 rows were missing), split-package-check exit 3 "no module" → 0; to Honeywell: null-context-write 0 → 22,
  timers 0 → 1 (exit 1). Every new run matches the plain-root run with the root spelling swapped `[ev: issue #329]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A walk over a lint root follows the root like `find -H` (bash `find -H`, Go `os.Stat` on the root) and never a symlink below it; SF10/SF11 and TestWalk pin it | `toolbelt/lib/scan-files.sh` § header | `[ev: tests/scan-files.bats SF10]` |

## Lessons
- `find` without `-H` lists nothing under a symlinked root and exits 0; a lint root needs `find -H`, never `-L`.
- `find -H` on a dangling or file root symlink also exits 0 with no output; only an explicit `[ -d ]` keeps it fail-closed.
- A Go port of a find walk must pick `os.Stat` (find -H) or `os.Lstat` (find -P) for the root on purpose and pin it.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-symlink-root-walk.md | kit | 2026-10-22 | pending | 1 |`
