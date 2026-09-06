# C9 Windows harness — run-sheet (execute on the Windows box)

One-page, step-by-step version of `qa/c9-harness-procedure.md`. Running this closes the last C9 pin
(`CLOSE-harness-run`). It proves the harness-only alarm/lockout pins that WSL cannot: CRA1/2/3 live
(CR-3 freeze routes a `BAlarmRecord`), CPB5 (CP-1 `sourceState==offnormal`), R14 real lockout + AuditEvent
attribution. ~30 min. Nothing touches PANCCADIA — `BTestNgStation` spins its own in-memory station.

## Prerequisites (once)
1. Windows workstation with **OptimizerSupervisor N4.14.0.162** and a **developer license** (the `test`
   command refuses without it).
2. A checkout of client `main` at the post-C9 tip (record the SHA). Modules: Compresores/CompPan-rt,
   Paccadia/ColdRoomPan-rt, Dashboard/DashboardPan-ux.
3. **Add JUnit to two modules' moduleTest classpath** (only DashboardPan-ux has it today). In
   `Compresores/CompPan/CompPan-rt/CompPan-rt.gradle.kts` and
   `Paccadia/ColdRoomPan/ColdRoomPan-rt/ColdRoomPan-rt.gradle.kts`, add to `dependencies { }`:
   ```
   moduleTestImplementation("junit:junit:4.13.2")
   ```
   (Doc-only risk: the plain-JUnit classes carry no `@NiagaraType`, inert to the TestNG runner.)
4. The three station test classes exist under each module's `srcTest/…/test/` (H1/H2/H3 from
   `qa/c9-harness-procedure.md`); if not yet authored, that section has them.

## Run (per module, from the module's profile dir in a Niagara console)
```
gradlew moduleTestJar
test ColdRoomPan        # H1: BFreezeAlarmStationTest — CRA1/2/3 live
test CompPan            # H2: BCompressorAlarmStationTest — CPB5
test DashboardPan       # H3: BConfigLoginStationTest — R14 lockout + AuditEvent
```
Add `v:3` for detail on a failure. Each must end with `Total tests run: N, Failures: 0, Skips: 0`.
**A `Skips: N` line is a REJECT of the run, not a pass.**

## What to paste back
Create `qa/c9-harness-run.md` on the kit repo with this exact shape (the close pin greps for three
`Total tests run: …, Failures: 0, Skips: 0` lines and refuses any `Skips: [1-9]`):
```
# C9 harness run — <date>
Windows: OptimizerSupervisor-N4.14.0.162 (<about string>) · dev license OK
Client main: <sha> · CompPan 2.2.0 · ColdRoomPan 2.1.0 · DashboardPan 2.2.0
## test ColdRoomPan
<verbatim console output, incl. the Total tests run line>
## test CompPan
<verbatim>
## test DashboardPan
<verbatim>
| Pin | Verdict |
| CRA1/2/3 live | PASS |
| CPB5 | PASS |
| R14 lockout | PASS |
| R14/PR6 AuditEvent | PASS |
```
Then `C10_CLOSE=1 … bats tests/c10-close.bats` (or the C9 gate re-run) turns CLOSE-harness-run green.

## If a pin FAILS
It is a defect in the PR that owns it (PR8 CRA, PR9 CPB5, R14 lockout/AuditEvent), filed against client
main; the WSL BLESS of that PR stands (its WSL contract held) and the fix is a follow-up PR with its own
RED. Do not edit the harness to make it pass.
