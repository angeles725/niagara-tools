<!-- review-status: pending -->
# 2026-10-21 · kit · shared-sha256-fail-closed

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T8a), branch
`r8/t8a-sha256` from origin/main 1197614
**Delta count**: 2

## What happened
`scripts/install-skill.sh` and `scripts/check-skill-drift.sh` each carried a copy of `sha256_of`; only the
error prefix differed. The copy in install-skill.sh ended with `exit 3` when neither `sha256sum` nor `shasum`
was on PATH, but every call ran inside `$( )`, so the exit only left the subshell: the script printed the
missing-tool message and then installed the launcher on two empty digests, exit 0. check-skill-drift.sh had
already met the same trap and worked around it with a second, inline tool check. Both copies now live in
`scripts/lib/sha256.sh` (`sha256_of` + `sha256_require <prefix>`), and both scripts call `sha256_require`
before any `$(sha256_of ...)`. Messages and exit codes stay byte-identical; net lines go down.

## Evidence
- RED: new `tests/install-skill.bats` IS5 (PATH without sha256sum/shasum) failed at `[ "$status" -eq 3 ]`;
  a manual run printed the missing-tool message, then `install-skill: installed …`, exit 0
  `[ev: tests/install-skill.bats IS5]`
- The workaround comment in check-skill-drift.sh ("sha256_of runs in a subshell, so its own exit 3 cannot
  stop this script") named the trap; install-skill.sh never got the fix `[ev: scripts/check-skill-drift.sh@1197614]`
- Helper test first: `tests/sha256.bats` SHA1-SHA5 RED (exit 127, lib absent) before the lib existed
  `[ev: tests/sha256.bats]`
- CI's `shellcheck scripts/*.sh` glob and shell-hygiene SH1 do not reach `scripts/lib/`; SHA5 shellchecks the
  lib from bats until the globs widen `[ev: .github/workflows/ci.yml shellcheck step]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **An `exit` inside a helper is only fatal outside `$( )`.** A helper that can fail fatally is split into a check that runs at top level (`require_x <prefix> \|\| exit N`) and a pure producer used inside `$( )`; the producer never relies on `exit`. Why: install-skill.sh installed on empty digests because its helper's `exit 3` ran in a subshell. Reviewer check: an `exit` inside a function that is called in `$( )` is a finding. | `CONTRIBUTING.md` §6 (shell checklist) | `[ev: tests/install-skill.bats IS5]` |
| 2 | **Shared shell helpers of `scripts/` live in `scripts/lib/`, and every lint glob covers it.** CI shellcheck and shell-hygiene SH1 list `scripts/lib/*.sh` next to `scripts/*.sh`, as they already list `toolbelt/lib/*.sh`. Why: a new lib dir is silently outside the gates until its glob is added. | `.github/workflows/ci.yml` shellcheck step; `tests/shell-hygiene.bats` SH1 | `[ev: tests/sha256.bats SHA5]` |

Neither is applied; a fold review decides each one and flips this retro.

## Lessons
- A workaround in one copy of a duplicated helper is a bug report for the other copy.
- Deduplicating a helper is the moment to fix its contract, so the fix lands once for every caller.

---
**Status**: PENDING — INDEX row appended: `| 2026-10-21-shared-sha256-fail-closed.md | kit | 2026-10-21 | pending | 2 |`
