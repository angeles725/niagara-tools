<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-reuse-home

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, a second boot on a kept home
(`--reuse-home`), branch `r10/rnt-reuse-home` from 591b88db (origin/main)
**Delta count**: 1

## What happened
A station test could not prove behaviour across a restart: every run made a fresh scratch dir and deleted it, and
`--keep-home` only kept it for inspection. `--reuse-home <dir>` boots again on that kept dir. It is the one flag that
points the runner at an existing dir, so it is guarded first. Each rule got a fake-runner RED first on generic
`ModuleA` fixtures; no station boot was run:
1. Every run writes `run-niagara-test.marker` in the dir it creates or takes, before the build: a run id
   `<UTC stamp>-<module>-<dir suffix>-<8 hex>`, the install path, the sha256 of the install's
   `defaults/system.properties` (`absent` without one) and an empty `log:`. A `--keep-home` run fills `log:` with
   its run log after writing it; the log header records the same `run id:`. A reuse run rebinds the marker to
   itself (new id, empty log) before it builds, so a run that dies leaves a dir no later reuse accepts.
2. The gate, in order: the dir must lie strictly under the scratch parent; D3 refuses a dir that is, holds or sits
   in a real `%USERPROFILE%\Niagara4.*\<brand>`, or has `registry\`/`security\` at its top (a copied marker changes
   nothing); the marker must exist (regular file, at most 4 KiB, each field once); its install and sha256 must be
   this run's; its run id must name this dir and equal the `run id:` the kept log in `--log-dir` records. Every
   refusal is exit 2 naming the field. Guard 1, guard 3 and the snapshots still run.
3. Reused: the home copy (its registry included), the scratch `niagara_user_home` and `user.home`. Rebuilt:
   `:jar`/`:moduleTestJar` into the kept `modules\`; `out\` is emptied so boot 1's testng-results.xml is never
   judged. No robocopy runs. `--exclude-jar` and `--module-verification` are exit 2 with it.
4. The dir is deleted at the end with a `cleanup: deleted <dir>` row, or kept with `--keep-home` again.

## Evidence
- the REDs on 591b88db (16 failing tests and subtests, the flag undefined), then green: `TestRunReuseHomeSeesPreviousBoot`, `TestRunReuseHomeRefusals` (b)-(f), `TestRunReuseHomeRefusesCopyFlags`, `TestRunReuseHomeCleanup`, `TestRunReuseHomeRebindsMarkerFirst` `[ev: go/internal/niagaratest/run_reusehome_test.go]`
- the marker, the gate and the rebind `[ev: go/internal/niagaratest/reusehome.go]`
- mutation: dropping the D3 check turned both (f) subtests red; dropping the marker-vs-log run id comparison turned (d) red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `run-niagara-test run --reuse-home <kept dir>` boots again on a `--keep-home` dir, bound by its marker (run id = the kept log's, install and system.properties sha256 = this run's) and refused for a real user home (D3); nothing is copied again, the module jars are rebuilt, the dir is deleted at the end unless `--keep-home` | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/reusehome.go]` |

## Lessons
- A flag that reuses an existing dir needs a record written by the runner and bound to an independent record (the run log), not a path check alone.
- A reused output dir must be emptied before the run: a results file left by the last boot would otherwise be judged as this one's.

## Fold
- The flag is code in the branch that wrote this retro (`go/internal/niagaratest/reusehome.go`).
- Δ1 is cited in `types/moduleTest.md` § Running from WSL (the `--reuse-home` bullet). Owed: a live two-boot reload proof on a client root.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-reuse-home`; log in `ledger.d/2026-10-25-r10-rnt-reuse-home.md`) — Δ1 in `types/moduleTest.md` § Running from WSL.
