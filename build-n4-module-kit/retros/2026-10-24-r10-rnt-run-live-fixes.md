<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-24 · kit · r10-rnt-run-live-fixes

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run`, fixes for the four defects of
its first live use, branch `r10/rnt-run-live-fixes` from b7840922 (origin/main)
**Delta count**: 4

## What happened
The first live `run-niagara-test run` against a client gradle root showed four defects. Each got a fake-runner
RED first, then the fix; no Windows boot was run for this work.

- **D1, the build fails on a Windows JDK path.** A client `gradle.properties` sets
  `org.gradle.java.installations.paths` to a Windows JDK path; WSL gradle fails parsing it
  (`java.net.URISyntaxException: Illegal character in opaque part at index 2`). The build now always passes
  `-Porg.gradle.java.installations.paths=<jdk8>` and `JAVA_HOME=<jdk8>`, `--jdk8 <dir>` (default
  `/usr/lib/jvm/java-8-openjdk-amd64`, refused without `bin/javac`).
- **D2, gradle downloads its distribution every run.** The scratch `user.home` (it holds the throwaway signing
  profile) also moved gradle's user home, so each run fetched gradle 7.6 (about 13 min). `GRADLE_USER_HOME` is now
  the caller's cache, `--gradle-user-home <dir>` (default `~/.gradle`); a manual build this way took 89 s.
- **D3, the NRE wrote the real user home, unnoticed.** The NRE ignored the scratch `niagara_user_home` set inside
  `cmd /v:on /c` and rebuilt the real user home's registry and signers cache. The runner now snapshots every
  `C:\Users\<you>\Niagara4.*\<brand>\registry\` and `security\` (`%USERPROFILE%` read through the runner seam)
  before the copy and after cleanup: any change is `FAIL  run-niagara-test  real user home changed during the run:
  <paths>`, and a scratch `niagara_user_home` without the `etc\` and `security\` the NRE creates is `FAIL ... scratch
  niagara_user_home was not used`. A separate non-booting probe then proved the override: `nre.exe -version` run from WSL with
  `WSLENV=niagara_user_home/w niagara_user_home=<win path>` created the scratch home (`etc/nre.properties`,
  `logging/`, `security/` keystores) and left the real homes unchanged (69 entries compared). The likely cause
  [INFER]: WSL re-quotes the inner `"` of the cmd line, so `set "niagara_user_home=…"` set another name, and the
  NRE fell back to `%USERPROFILE%\Niagara<maj>.<min>\<brand>`. The test step now runs `bin/test.exe` straight
  from WSL (cwd the copy) with that environment, `WSLENV` extended, never replaced, and reads its WSL exit
  status (-1 shows as 255); the `cmd /v:on` line is gone. D3 stays the gate.
- **D4, the in-use probe failed open on service stations.** A `niagarad.exe` service (LocalSystem) ran three
  `station.exe`; a non-elevated `Win32_Process` query returns an empty `ExecutablePath` and `CommandLine` for them,
  so the probe could not tie them to an install and let the run go. Such a process now prints `UNREAD <pid> <ppid>
  <image>`, every niagarad service `SVC <pid> <name> <PathName>` (`Win32_Service`, readable without elevation); a
  process that is, or was started by, a niagarad service belongs to the `niagara_home` its `PathName` names. An
  attributed install equal to, inside or containing `--install` is refused; an unattributable one is refused
  (`cannot attribute running station process <pid> to an install`); the PASS row names the service.

## Evidence
- the REDs, then green: `TestRunTestExeGetsUserHomeThroughWSLENV`, `TestRunBuildUsesWSLJDK8`, `TestRunBuildKeepsGradleUserHome`, `TestRunRealUserHomeChangedIsFail`, `TestRunScratchUserHomeUnusedIsFail`, `TestRunPreflightAttributesServiceStations` `[ev: go/internal/niagaratest/run_sandbox_test.go]`
- the user-home snapshot `[ev: go/internal/niagaratest/userhome.go]`, the service attribution `[ev: go/internal/niagaratest/attribute.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A tool that sandboxes a vendor runtime must verify the sandbox was used, by checking that the real location stayed unchanged, not trust the knob | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/userhome.go]` |
| Δ2 | The NRE ignored the scratch `niagara_user_home` set inside `cmd /v:on /c` and rebuilt the real user home's registry and signers cache; D3 is the gate. Pass it instead through the WSL interop environment: run `bin/test.exe` from WSL with `niagara_user_home=<win path>` and `WSLENV` extended with `niagara_user_home/w` | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/run.go]` |
| Δ3 | A non-elevated process probe reads no path or command line of a LocalSystem service's processes: attribute them through `Win32_Service` `PathName`/`ProcessId` (the process or its parent) and refuse what stays unattributable | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/attribute.go]` |
| Δ4 | WSL gradle on a client root: pass `-Porg.gradle.java.installations.paths` and `JAVA_HOME` of a WSL JDK 8, and keep `GRADLE_USER_HOME` on the caller's cache when `user.home` is scratch | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/run.go]` |

## Lessons
- A knob set is not a knob honoured: check the place the runtime would write without it.
- An empty field from a privileged process is "unknown", never "not ours"; a guard that skips unknowns fails open.
