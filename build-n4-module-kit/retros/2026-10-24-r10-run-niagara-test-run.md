<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · r10-run-niagara-test-run

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test` phase 2b (the runner), branch
`r10/run-niagara-test-run` from 697beb3e (`r10/run-niagara-test-testng`)
**Delta count**: 3

## What happened
`run-niagara-test run --install <dir> --gradle-root <dir> --part <part> --module <module> [--timeout <dur>]
[--keep-home] [--scratch <dir>]` automates the route a live station-test run of a client module (`ModuleA`) proved on a 4.14
install: copy the install into a new dir under `%TEMP%`, build `:<part>:jar :<part>:moduleTestJar` in WSL against
the copy, run `bin\test.exe <module> -v:2 -output:` with a scratch `niagara_user_home`, and judge
`testng-results.xml` with the phase-2a verdict (stderr as the console log; the exit code a third check).

- **Seam.** Every process goes through a `Runner` (`Run(ctx, Cmd) (exit, error)`); the orchestration is tested
  on a fake runner without Windows: the step order, the refusals, cleanup on a build failure, a panic and a
  timeout, the timeout kill, and the verdict wired to the XML. `execRunner` starts each process in its own
  process group and kills the group when the context ends (tested with a real forked `sleep`).
- **Guards.** The build's `niagara_home` is the copy, never inside `--install` (canonical Windows paths,
  symlinks resolved, compared case-insensitively). The install is snapshotted before the run (every `modules/`
  entry by name, size and mtime; `etc/`, `security/` and the top level by name and mtime, not a full walk of
  1.3 GB over /mnt) and compared after: any change is FAIL, not exit 2, since the run already happened. An
  install named in the command line or executable path of a running `station.exe`, `nre.exe`, `java.exe` or
  `wb.exe`, or with a `stations\*\config.bog.lock` another process holds open (an exclusive read-only open fails
  with a sharing violation; opening it writes nothing), is refused; a probe that cannot tell refuses too.
- **Kill.** Cleanup lists the Windows processes whose command line or executable path names the scratch dir the
  run created (a random name), kills each by PID (`taskkill /PID <pid> /T /F`, never `/IM`), lists again to
  confirm none is left (a survivor is exit 3), and deletes only that dir. The WSL interop process of `cmd.exe`
  dying does not stop the Windows process, so the PID kill is what bounds a timed-out run.
- **Quoting.** Paths with a cmd or PowerShell metacharacter (`" % ! & ^ | < > ' $` and backtick) are refused, and
  the scratch path may hold no whitespace, so no command line needs escaping; PowerShell probes go through
  `-EncodedCommand`.
- **Not done here.** No live run (NRE time is scarce): `TestLiveRun` is opt-in with `RUN_NIAGARA_TEST_LIVE=1` and
  `RNT_LIVE_*`. The process listing and the lock probe are untested on a real host.

## Evidence
- `go test ./internal/niagaratest` green: run, refusal, guard, cleanup and timeout tests on the fake runner `[ev: go/internal/niagaratest/run_test.go]`
- the process-group kill on a real runner `[ev: go/internal/niagaratest/host_test.go]`
- bats RNT1-RNT4 green `[ev: tests/run-niagara-test.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `toolbelt/run-niagara-test.sh run` replaces the hand route of running a moduleTest from WSL (copy, build against the copy, test.exe, verdict, cleanup); the hand route stays as the fallback | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/run.go]` |
| Δ2 | Before a test run on an install, refuse one a running station, NRE or Workbench uses (process command line or executable path) or whose `stations\*\config.bog.lock` is held open; fail closed when the probe cannot tell | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/host.go]` |
| Δ3 | Bound a Windows-side run from WSL by killing its processes by PID, found by the unique dir the run created; killing the WSL interop process does not stop the Windows process | `types/driver-authoring.md` § 12.1 | `[ev: go/internal/niagaratest/run.go]` |

## Lessons
- A cmd.exe child started from WSL outlives the WSL process that started it; a timeout must find and kill the
  Windows process tree itself.
- Refusing metacharacters in every path is cheaper and safer than escaping for cmd.exe and PowerShell at once.

## Fold
- The runner is code already shipped in the PR that wrote this retro (`go/internal/niagaratest/run.go`, `host.go`, `toolbelt/run-niagara-test.sh`); not re-implemented.
- Δ1 was cited at merge in `types/moduleTest.md` § Running from WSL.
- Δ2 → the § Running from WSL refusal bullet, now naming the fail-closed probe (`TestRunRefusals` "guard 3: the probe cannot tell").
- Δ3 → the § Running from WSL cleanup bullet, not `types/driver-authoring.md` § 12.1, which defers to that section for the runner (`TestRunTimeoutKillsTree`, `run.go` taskkill by PID).
- "Not done here" (no live run) was closed by the live fixes of retro r10-rnt-run-live-fixes. Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail-4`; log in `ledger.d/2026-10-25-r10-fold-tail4.md`) — Δ1-Δ3 in `types/moduleTest.md` § Running from WSL.
