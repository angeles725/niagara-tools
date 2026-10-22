<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · r8-script-dir

**Session**: niagara-research quality round 8, issue #310, branch `fix/r8-script-dir` from origin/main 1e39076 (rebased onto 42b9ffa)
**Delta count**: 1

## What happened
42 toolbelt files (41 scripts and `lib/awk-lint.sh`, 54 lines) found their own directory with
`${BASH_SOURCE[0]%/*}`. When BASH_SOURCE carries no slash, the strip leaves the file name, so `cd` fails and every
lib is sourced from `/lib/<lib>.sh`. The lint then dies on an unset lib variable, or, inside `$(...)`, skips the
check and exits 0. The defect only appears with `bash lint-x.sh` run from the toolbelt dir: a PATH lookup, by
shebang or by `bash <name>`, always hands bash a path that contains a slash, so a bare name on PATH from another
dir already worked. That is why the brief's PATH-only reproduction stayed green on the old code and the RED came
from the from-own-dir case. Every occurrence now uses the dirname form `$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)`
(`$(dirname "${BASH_SOURCE[0]}")` in the lib, where no `cd` was used before), applied with one sed. Behaviour is
unchanged otherwise, because dirname and the strip agree for every path that contains a slash.

Why this idiom: gen-retro-index.sh, lint-structure.sh and four other toolbelt files already used it, and issue #310
names it, so after the change the toolbelt has one form. dirname is POSIX and its `.` result for a slash-less name is
exactly the case the strip got wrong. `readlink -f` was rejected because it also follows a symlinked script to
its target, which changes behaviour beyond the bug fix, and it needs GNU coreutils. `pwd -P` was rejected for the
same reason: the old form used `pwd`. A lib helper cannot carry the fix, because the dir is needed before any lib
is sourced.

## Evidence
- RED on 1e39076: `cd build-n4-module-kit/toolbelt && bash lint-timers.sh <fixture>` printed
  `cd: lint-timers.sh: Not a directory`, `/lib/method-boundary.sh: No such file or directory`,
  `scan_files_or_exit: command not found` and `MB_AWK: unbound variable`, with exit 1 but no finding. SD1 failed, and
  SD2 (bare name on PATH, unrelated cwd) passed `[ev: tests/script-dir.bats SD1]`
- GREEN: `bats tests/script-dir.bats` 4/4 `[ev: tests/script-dir.bats]`
- The premise of the brief, that a bare name run through PATH is broken, was wrong. The real trigger is
  `bash <name>` run from the toolbelt dir. SD2 keeps the PATH case as a regression check, and it is green on both
  the old and the new code `[ev: tests/script-dir.bats SD2]`
- The full ci-local run then failed preflight.bats PF-jarlock-skip-9p. Its `_pf_fakebin` PATH did not link
  `dirname`, so `lib/fs-type.sh` was not sourced and the 9p skip was lost. `dirname` is now in that list
  (preflight.sh already calls it at L272/L287) `[ev: tests/preflight.bats _pf_fakebin]`
- Rebased onto origin/main 42b9ffa. lint-structure.sh:405 now has a comment that names the idiom, so SD3 skips
  comment lines. SD3-neg pins that skip, and dropping it fails SD3 and SD3-neg `[ev: tests/script-dir.bats SD3-neg]`
- Mutations: putting `${BASH_SOURCE[0]%/*}` back on lint-timers.sh:122 fails SD1 and SD3. Writing
  `${BASH_SOURCE%/*}` into lib/awk-lint.sh:73 fails SD3 only, because SD1 sources that lib by an absolute path `[ev: tests/script-dir.bats SD3]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A toolbelt script derives its dir with `$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)`, never a `${BASH_SOURCE…%…}` / `${0%…}` strip; SD3 ratchets it | `tests/script-dir.bats` § SD3 | `[ev: tests/script-dir.bats SD3]` |

## Lessons
- A script that gains an external tool must also gain it in every test's fake PATH. Search `PATH="$FAKEBIN"`
  fixtures before you swap a builtin form for a fork.
- `${x%/*}` is not a dirname: it returns the whole value when there is no slash. Use `dirname` for any
  path-derived directory.
- A reproduction of a path bug has to pin the exact BASH_SOURCE shape. A PATH lookup always produces a slash, and
  only a from-own-dir `bash <name>` produces none.
- A lint test that asserts only the exit code can pass on a crash. Assert the finding text, and assert that no
  lib-load error appears.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-r8-script-dir.md | kit | 2026-10-22 | pending | 1 |`
