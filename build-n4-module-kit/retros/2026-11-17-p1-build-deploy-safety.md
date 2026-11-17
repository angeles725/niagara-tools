<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 7 -->
# 2026-11-17 · kit · p1-build-deploy-safety

**Session**: P1 backlog rows OD002 (build-mod Δ6), OD003 (bench Δ23) and ng-deploy Δ12, three ways a build or
deploy could leave jars no check had passed where a station loads them.
**Delta count**: 7

## What happened
- build.sh: a SIGINT/SIGTERM/SIGHUP while gradle ran removed the temporary directory (which holds the baseline
  snapshot) and exited, leaving the new, unverified jar in `<niagara_home>/modules`. The handler now restores the
  pre-build jars (and removes a first-deploy jar) first. Writing the test surfaced a pre-existing data race: the
  handler read the list of temporary paths the chain was appending to; it is now under a mutex.
- ng-deploy: `--no-deploy` with `STATION_MODULES_DIR` = `<NIAGARA_HOME>/modules` built straight into the
  station's directory and kept the jars with no precheck, gate or copy. It is now refused before gradle (exit 10).
- ng-deploy: the env file could still skip the gate with `MODE=B` or `VERIFY_MODULE_BIN`; both are now refused like
  `NO_GATE=1` (exit 10, file and line) unless the command line says the same thing.

## Evidence
- RED before the fix: `TestASignalBeforeTheGateRestoresThePreBuildJars` (modules/Foo-rt.jar not the pre-build jar,
  Foo-ux.jar left), `TestNoDeployIntoTheStationThatIsTheHomeModulesDirectoryIsRefused` (exit 0, gradle ran, the
  station holds the new jars), `TestAnEnvFileMayNotSkipTheGateWithModeBOrAnotherGate` (exit 0, `copy ok` with a
  failing gate).
- Native review (risk, resilience, reliability) approved with non-blocking findings, handled in a follow-up
  commit: the chain's restores and cleanup and the signal handler now share one mutex (`modulesMu`), so the
  snapshot cannot be removed mid-restore and two restores cannot write one jar at once; the snapshot marks the
  window open only once it is whole; SIGHUP got its test case. The claim that a CRLF env file slips `MODE=B` past
  the guard was refuted (`envfile.TrimCR` strips during the read) and pinned by `TestACRLFEnvFileMayNotSkipTheGateWithModeB`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. build.sh restores the pre-build modules/ jars on SIGINT/SIGTERM/SIGHUP between the baseline snapshot and a passing chain, before the temporary directory goes | `go/internal/buildmodule/signal.go` | `[ev: retro go-build-module-port Δ6]` |
| Δ2 | APPLIED 2026-11-17. ng-deploy refuses `--no-deploy` when `STATION_MODULES_DIR` is `<NIAGARA_HOME>/modules` (exit 10, before gradle) | `go/internal/ngdeploy/home.go` | `[ev: retro build-kit-bench-session Δ23]` |
| Δ3 | APPLIED 2026-11-17. The env file may not set `MODE=B` (unless `--mode`) or `VERIFY_MODULE_BIN` (unless `--no-gate`): exit 10 with file and line | `go/internal/ngdeploy/envguard.go` · `usage.txt` | `[ev: retro go-ng-deploy-port Δ12]` |
| Δ4 | APPLIED 2026-11-17 (`gradleproc.go`: the handler passes the signal on to the running gradle and waits, killing it after 10 s, before the restore; `TestASignalReachesTheRunningGradleBeforeTheRestore`, `TestAGradleThatIgnoresTheSignalIsKilledBeforeTheRestore`). A signal sent to build.sh's pid alone (not its process group) does not reach gradle, which can install its jar after the restore; the handler could kill and wait for the running gradle first | `go/internal/buildmodule/signal.go` · `gradle.go` | `[ev: code go/internal/buildmodule/signal.go]` |
| Δ5 | APPLIED 2026-11-17 (refused in the env file, no flag lifts it; the command's environment still sets it). `SCHEMA_RISK_BIN` in the env file can put any executable in the place of the schema-risk precheck, the same shape as Δ3; decide whether to refuse it the same way | `go/internal/ngdeploy/envguard.go` | `[ev: code go/internal/ngdeploy/optiontable.go]` |
| Δ6 | APPLIED 2026-11-17 (header names the interrupt behaviour and the process-group rule of Δ4). build.sh's usage header (`toolbelt/build.sh`) does not say what an interrupt does; add the two lines `build-verify.md` now carries | `build-n4-module-kit/toolbelt/build.sh` | `[ev: code go/internal/buildmodule/signal.go]` |
| Δ7 | APPLIED 2026-11-17 (`signal_order_test.go`: a signal while the chain holds modulesMu waits; dropping the handler's lock fails it). No test drives a signal while the chain itself is inside restoreBaseline, removeFirstDeploy or cleanup, so the `modulesMu` ordering is unpinned (native review of the follow-up, R3-001) | `go/internal/buildmodule/signal_test.go` | `[ev: code go/internal/buildmodule/signal.go]` |
| Δ8 | APPLIED 2026-11-17. A signal during report-module (after the gate passed) rolled back the verified jar: `gate()` returns an exit code, not `proceed`, so Main's `unverified` reset never ran; the gate now clears it when it passes (`TestASignalAfterTheGatePassedKeepsTheVerifiedJar`, verifier probe P6 of #795) | `go/internal/buildmodule/gate.go` | `[ev: code go/internal/buildmodule/gate.go]` |
| Δ9 | APPLIED 2026-11-17. #795's `WaitDelay` on gradle failed a passing build (exit 30) when a child of gradle kept the output open past 10 s; removed, only the signal handler's wait is bounded (`TestAChildHoldingGradlesOutputDoesNotFailTheBuild`, probe P9) | `go/internal/buildmodule/gradleproc.go` | `[ev: code go/internal/buildmodule/gradleproc.go]` |
| Δ10 | Open. `stopGradle` signals and kills only gradle's own pid: a grandchild (a single-use daemon) can still install its jar after the restore (probe P8). Starting gradle in its own process group and signalling the group would cover it, but takes it out of the terminal's foreground group; weigh that first | `go/internal/buildmodule/gradleproc.go` | `[ev: code go/internal/buildmodule/gradleproc.go]` |
| Δ11 | APPLIED 2026-11-17 (`keepReplaced` takes `modulesMu`; `TestKeepReplacedWaitsForTheRestoreLock`). `keepReplaced` ran without `modulesMu`, so when gradle ends during the signal handler it can keep a `.mirror-replaced/` copy and print a misleading row (no wrong jar; probe P11b, also with Ctrl-C at the baseline) | `go/internal/buildmodule/command.go` | `[ev: code go/internal/buildmodule/command.go]` |

## Lessons
- A signal handler that runs beside the chain shares its state: anything it reads (temporary paths, the snapshot)
  needs a happens-before edge, and `go test -race` with a real signal is what shows it.
