<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · buildsh-mirror-baseline-b6

**Session**: row B6 of an external session retro (a client A module build, 2026-11-16): `build.sh` left the
mirror's `modules/<MOD>-rt.jar` different from what the station runs, and a module without `.deploy-baseline` got a
bare INCOMPLETE (exit 53).
**Delta count**: 2

## What happened
The brief assumed `build.sh` copies the jar into the mirror. It does not: `build.sh` is a launcher over
`go/cmd/build-module`, and the copy is the niagara plugin's `:jar` step. The drift gate already backed up the deployed
jar before gradle, but only restored it on a FAIL (exit 34/50/51/52); on a pass the backup was a temp file deleted at
exit. Dependents compile against `<niagara_home>/modules`, so not copying (option b) would break a `-ux` that depends
on a freshly built `-rt`; keeping the replaced jar and offering a restore (option a) changes no build input.
The INCOMPLETE line came from `report-module`'s schema-risk row, which named a doc section but no command.

## Evidence
- RED before GREEN: `TestMirrorKeepsTheReplacedJar`, `TestMirrorLineAfterIncompleteReport`, `TestRestoreMirrorRefuses`
  (go/internal/buildmodule/replaced_test.go) and `TestSchemaRiskNoBaselineNamesTheCommand`, `TestShellWord`
  (go/internal/reportmodule/special_floor_test.go) failed on the base code; the schema-risk test runs the printed
  command against a real git repo and checks the extracted baseline. `[ev: go/internal/buildmodule/replaced.go]`
- Old behaviour pinned: `TestMirrorKeepsNothingWhenModulesIsUnchanged` (identical jar, drift FAIL exit 51,
  `--no-drift-check`, real home with `--install-into-home`), exit 53 still 53, `--first-deploy` row unchanged.
- Correction batch on #721: the module was built twice that day, so the second build's snapshot was the first
  build's jar. RED then GREEN: `TestTwoBuildsRestoreTheOriginal` (two builds, the restore names the pre-first-build
  jar), `TestRestoreRefusesAKeptJarWhoseHashDiffers`, `TestKeepFailureWarnsAndKeepsTheExitCode` (unwritable
  `.mirror-replaced/`, exits 0 and 53 unchanged), `TestIncompleteCrossRefOnlyForSchemaRisk`.
- Second correction batch (independent verifier): CI "bats unit tests" was red on kit-links L2 because the
  `build.sh` header said "git archive" (L2 greps toolbelt/*.sh for `git`); fixed. RED then GREEN:
  `TestUnrecordedFirstIsNotClaimed` (pointer unwritable: one WARN, no "first replaced" claim),
  `TestRestoreCommandIsQuoted` (toolbelt path with a space), and the schema-risk test now runs the printed command
  with a wrong ref and with `<deployed-ref>` pasted as is: neither leaves a `.deploy-baseline`.
- Mutation: dropping the `.niagara-mirror` guard of `--restore-mirror` fails bats `BS-restore-mirror` and
  `TestRestoreMirrorRefuses`.
- Differential vs origin/main: `build.sh --help` differs only in the new lines; `report-module.sh` on the 22
  `tests/fixtures/report-module` trees differs only in the schema-risk SKIP row, exit codes identical; the 100
  report-module goldens re-recorded with `-update` moved that row only (156 lines).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. A mirror build keeps the `modules/` jar it replaced under `<mirror>/.mirror-replaced/` and prints old/new vendorVersion + sha256 and the restore command; `build.sh --restore-mirror <kept>` puts it back (marker-guarded) | Go `internal/buildmodule` `replaced.go`, `command.go`, `options.go`, `messages.go`; `toolbelt/build.sh` header; `build-verify.md` §Mirror | `[ev: go/internal/buildmodule/replaced.go]` |
| Δ2 | APPLIED 2026-11-16. The schema-risk SKIP row of a missing `.deploy-baseline` names the `git archive` command that creates it from the deployed ref; exit 4 / 53 unchanged | Go `internal/reportmodule` `special_floor.go`; `build-verify.md` §Schema-risk deploy workflow | `[ev: go/internal/reportmodule/special_floor.go]` |

## Lessons
- A "restore to what the station runs" must name the FIRST jar a cycle replaced, not the last: one build per day
  is not the field pattern.
- A toolbelt script's comments are scanned by kit-links L2: write "archive command", never the word `git`, there.
- Read the launcher before scoping a `toolbelt/*.sh` change: most of them exec a Go binary, so the edit surface is
  `go/internal/<tool>`, not the script.
- `git archive <ref>:./` from a subdirectory archives nothing; `git -C <dir> archive <ref> .` writes the directory's
  paths relative to it — run a printed command in its test instead of trusting its shape.
