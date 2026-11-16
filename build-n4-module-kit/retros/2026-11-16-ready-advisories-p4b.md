<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · ready-advisories-p4b

**Session**: row P4b of the 2026-11-16 client A over-amps retro: the native-review advisories of niagara-tools #729
(`report-module --ready`), and the T34 doc follow-up of the 2026-11-15 coordination stretch.
**Delta count**: 2

## What happened
The review of #729 left six advisories. A module root that could not be listed read as "no built jar under ...",
hiding the real error. An artifact with no built jar got no row, so a module with one part unbuilt could still read
READY. A baja floor that could not be compared with `--expect-niagara` was a WARN, so the jar read READY with its
target unchecked. `schemaRiskVerdict` returned the same SKIP for `--first-deploy` and for a missing baseline, so
each caller re-derived the difference from the flag. The helpers at the top of `ready_test.go` built the argument
list by index (`a[2] = home`, `args[6] = signer`), and no test pinned `--target-version` reaching verify-module.
All six are fixed. Separately, #726 added a 100,000 entries-read bound to lint-ntpl-manifest-deps whose header
and `types/logic-authoring.md` still named only the 128 MiB bound.

## Evidence
- RED before GREEN: `TestReadyUnreadableRootNamesTheReadError` (stderr was the "no built jar" line),
  `TestReadyArtifactWithoutJarIsAFailRow` (no Mod-ux row, exit 0) and `TestReadyIncomparableFloorBlocksReady`
  (WARN, READY, exit 0) failed on origin/main and pass after the fix; `TestSchemaRiskVerdictTellsFirstDeployFromMissingBaseline`
  did not compile against the two-value verdict. `[ev: code go/internal/reportmodule/ready_advisories_test.go]`
- `TestReadyTargetVersionReachesVerifyModule` passed on origin/main (a missing test, not a bug); dropping the
  `--target-version` append in `verifyGate` turns it red (mutation, not committed). `[ev: code go/internal/reportmodule/ready_checks.go]`
- A real client module root holds `docs/` beside its `-rt` artifact, so only a directory with `module-include.xml`
  or a gradle build file counts as an artifact; the test pins that `docs/` gets no row. `[ev: code go/internal/reportmodule/ready.go]`
- Differential of the full report on `tests/fixtures/report-module` against origin/main: identical stdout, stderr
  and exit on every fixture. `[ev: code go/internal/reportmodule/special_floor.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. `--ready`: an unreadable root names its error (exit 3), an unbuilt artifact is a `FAIL  jar` row, an incomparable baja floor FAILs | `build-verify.md` § Ready to load | `[ev: code go/internal/reportmodule/ready.go]` |
| Δ2 | APPLIED 2026-11-16. The entries-read bound of lint-ntpl-manifest-deps and its FAIL row are documented beside the 128 MiB bound | `toolbelt/lint-ntpl-manifest-deps.sh` header · `types/logic-authoring.md` | `[ev: code go/internal/lint/rules/ntplmanifest/budget.go]` |

## Lessons
- A load verdict must fail closed on every input it could not judge: an unread root, a missing part and an
  incomparable version are each a reason not to print READY, never a WARN beside it.
