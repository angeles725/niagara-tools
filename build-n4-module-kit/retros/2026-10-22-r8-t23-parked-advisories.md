<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · r8-t23-parked-advisories

**Session**: niagara-research quality round 8, work unit T23 (issues #226 and #246), branch `fix/r8-t23-parked-advisories` from origin/main e3713cf
**Delta count**: 2

## What happened
T23 triaged the ~70 parked review advisories of #226 (polish 2026-10-02, plus its D1-D5b comments) and #246 (audit
2026-10-03) against main. Most were already obsolete (lib/awk-lint.sh, scan_files_or_exit, the PF-jarlock-lsof-fail
pin); the still-valid ones were fixed test-first. Two were real fail-opens still on main: an ambient `FIRST_DEPLOY=1`
skipped ng-deploy's schema-risk precheck (#246 A2b, never landed), and a concise-body arrow poll in
lint-spa-poll-no-recovery read a later function's `lastOk*` as its own success time. New checks that only a real
parser can do were recorded as owed Go-lint items, not written in bash.

## Evidence
- NG-first-deploy-ambient RED: `[ "$status" -eq 12 ]' failed` with `FIRST_DEPLOY=1` exported; GREEN after
  `FIRST_DEPLOY=0` at `scripts/ng-deploy.sh:34-37` `[ev: tests/ng-deploy.bats]`
- SPR11 RED `concise.js:2` row missing; GREEN after the concise-body bound in `lint-spa-poll-no-recovery.sh` step 2
  `[ev: tests/lint-spa-poll-no-recovery.bats]`
- BS-drift-restore-fail-truth RED: no `Permission denied` cause; the old text claimed modules/ "now holds the NEW jar"
  while `cmp` proved it still held the pre-build jar `[ev: tests/build-sh.bats]`
- CV-owed-wide: `[[ a ]] && [[ b ]]` as a non-final command never failed on `a` `[ev: tests/commissioning-verify.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A bats assertion is its own command line: an `[[ a ]] && [[ b ]]` list that is not the test's last command never fails on `a` (bats errexit ignores a failing `&&` list member) | `CONTRIBUTING.md` § tests | `[ev: tests/commissioning-verify.bats CV-owed-wide]` |
| Δ2 | A gate never reads an opt-out (skip/first-deploy/accept) from the ambient environment: only a flag or an explicit env-file key, so a variable left exported in a shell cannot silently disable a check | `METHODOLOGY.md` § fail closed | `[ev: tests/ng-deploy.bats NG-first-deploy-ambient]` |

## Lessons
- A failure message states the observed state (`cmp` the file), never the assumed one.
- Parked-advisory issues rot fast: re-verify every item against main before acting; half were already fixed.
- A precision gap that needs a parser (call graph, generics, overloads) is an owed Go-lint item, not more awk.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-22-r8-t23-parked-advisories.md | kit | 2026-10-22 | pending | 2 |`
