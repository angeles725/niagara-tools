<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-19 · kit · build-docs-not-artifact

**Session**: kit branch fix/build-docs-not-artifact — deltas K4 (a client 2.3.0 upgrade retro) and P4 (a client persist-writes retro)
**Delta count**: 2

## What happened
A client module root holds `<MOD>-rt/`, `<MOD>-ux/` and a plain `docs/` folder. With `--schema-baseline-ref`
build.sh extracted a baseline for `docs/` too and report-module ran schema-risk on it: an env fault (exit 4,
build exit 10). With hand-made baselines in the parts only, `docs/` still read as an artifact with no baseline
and no jar, so the run was INCOMPLETE (exit 53) for a folder that is not part of the module. Separately, a build
whose report-module punch-list FAILed (exit 50) left its jar in the mirror's `modules/`; the fixed rebuild at
the same version then read as drift (exit 51) and needed `--no-drift-check`.

## Fix
- One artifact definition, `reportmodule.IsModuleArtifact` / `SplitArtifacts` (module-include.xml, or a
  `<MOD>-rt/-ux/-wb/-se/-doc` part with its build file), now used by the lint run, `--ready`, build.sh's
  `--schema-baseline-ref` and its INCOMPLETE cross-reference. A non-artifact directory is one INFO line and no
  rows. A root where no directory reads as an artifact keeps walking every directory (the recorded golden cases
  and any unrecognized layout lose nothing).
- A report-module FAIL rolls `modules/` back exactly as a failed verify gate does (`rollBack`), so
  `.mirror-replaced/` keeps nothing and the next same-version fix passes the drift gate. INCOMPLETE and an
  environment error leave the new jar, as before.

## Evidence
- RED: `TestDocsDirectoryIsNoArtifact` exit 4 (INCOMPLETE) with `docs  SKIP  verify-module` / `schema-risk` rows;
  `TestSchemaBaselineRefSkipsADocsDirectory` extracted `docs`; `TestReportFailRestoresTheMirrorJar` left the
  FAILed jar in `modules/`. All green after the fix. `[ev: go/internal/reportmodule/artifacts_test.go]`
- Golden `fx-write-path-fail` (a fixture with `docs/`) now reads `1 artifact` and one INFO line for `docs`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Applied: report-module and build.sh share one artifact predicate; `docs/` is never an artifact | `go/internal/reportmodule/artifacts.go`, `go/internal/buildmodule/baselineref.go` | `[ev: retro client-2.3.0-big-upgrade K4]` |
| 2 | Applied: a report-module FAIL restores the pre-build mirror jar | `go/internal/buildmodule/gate.go` | `[ev: retro client-persist-writes-and-off-ranges P4]` |

## Lessons
- An artifact set computed in two tools drifts: keep one predicate and make every caller use it.
- Any exit that says "do not deploy" must leave the mirror as the station has it, or the next run's drift gate
  compares against a jar that never shipped.

---
**Status**: FOLDED — INDEX row generated from the header: `| 2026-11-19-build-docs-not-artifact.md | kit | 2026-11-19 | folded | 2 |`
