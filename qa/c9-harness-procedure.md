# C9 Windows `niagaraTest` harness — procedure for the harness-only pins

Scheduled ONCE, after wave 3 has merged PR8 (CR-3), PR9 (CP-1) and R14 (config login) into client main.
These pins are `skip`-gated or absent in WSL by design: a Baja-typed JUnit compiles offline but any
`BComponent` construction throws `ExceptionInInitializerError` (proven 2026-09-06, C9 design). A SKIP is
not a PASS (retro campaign7 D9). Nothing below counts toward a PR's WSL pin count.

| Pin | PR | What only a running station can show |
|---|---|---|
| CRA1 / CRA2 / CRA3 (live halves) | PR8 | `freezeAlarmPt` + `BAlarmSourceExt` route a `BAlarmRecord` with `sourceState == offnormal` on a freeze trip and `normal` on recovery |
| CPB5 | PR9 | the `AlarmSupport.newOffnormalAlarm` record from `BCompressorControl` carries `sourceState == offnormal`; `toNormal` after recovery past the deadband |
| R14 lockout | R14 | the STATION locks the second operator after `maxBadLoginsBeforeLockOut` (5) failures inside `lockOutWindow`, for `lockOutPeriod` |
| R14 / PR6 AuditEvent | R14, PR6 | a config-session write lands in `AuditHistory` attributed to the re-authenticated user, not the kiosk user |

## 1. Mechanism (Tridium, cited)

- Test sources live in `srcTest/`; a test class extends `javax.baja.test.BTestNg` (or `BTestNgStation`
  for a running station) and declares its Type with `@NiagaraType` + the Slot-o-Matic block
  [CERT-doc devguide/test.html "Basic Test Case", "Test in a Running Station"].
- `gradlew moduleTestJar` builds `<module>Test.jar`; the console command `test <moduleName>` or
  `test <moduleName>:<TypeName>` runs it; output ends with `Total tests run: N, Failures: 0, Skips: 0`
  [CERT-doc devguide/test.html "Compile and execute"]. Single-method runs are not supported.
- `BTestNgStation` creates an in-memory test station (`stationHandler.getStation()`); override
  `configureTestStation(BStation, String, int webPort, int foxPort)` to add services, `addUser(name, role, password)`
  for extra users, `startWebServer()` for HTTP tests [CERT docSource test-wb
  `com/tridium/testng/BStationTestBase.java` :84 :271 :608-662, devguide "Station Creation", "Roles, Users, and Permissions"].
- The three modules already declare `moduleTestImplementation(":test-wb")` (CompPan-rt.gradle.kts:53,
  ColdRoomPan-rt.gradle.kts:53, DashboardPan-ux.gradle.kts:57) and the JaCoCo plugin for `niagaraTest`.
- Binaries on the Windows box: `C:\Honeywell\OptimizerSupervisor-N4.14.0.162\bin\test.exe` (present; `ls` 2026-09-06).

## 2. What it needs from Cristian (one slot, ~30 min, after wave 3 merges)

1. The Windows workstation with OptimizerSupervisor N4.14.0.162 and the **developer license** (the
   `test` command refuses without it — the existing `CompressorControlTest.java:16` header already records this).
2. A checkout of client `main` at the post-wave-3 tip (record the SHA in the run record).
3. **JUnit on the moduleTest classpath for two modules.** Only `DashboardPan-ux.gradle.kts:58` declares
   `moduleTestImplementation("junit:junit:4.13.2")`. `CompPan-rt` and `ColdRoomPan-rt` do not, so their
   `moduleTestJar` would fail compiling the plain-JUnit files under `srcTest/` (`package org.junit does not
   exist`). Add the same line to `CompPan-rt.gradle.kts` and `ColdRoomPan-rt.gradle.kts` dependencies
   (client PR, doc-only risk, no runtime effect: plain-JUnit classes carry no `@NiagaraType` and are inert
   to the TestNG runner) [INFER: not yet executed on Windows; verified only by reading the three .kts].
4. Run, per module, from the module's profile dir in a Niagara console:
   ```
   gradlew moduleTestJar
   test ColdRoomPan        (then)   test CompPan        (then)   test DashboardPan
   ```
   and paste the three verbatim outputs into the run record (§5). Verbosity `v:3` if a failure needs detail.
5. No production station is involved: `BTestNgStation` spins its own station. PANCCADIA is never touched.

## 3. Harness classes to add (one per module, `srcTest/com/angeles/<Mod>/test/`, package must match the dir)

All three are Baja-typed, so they are authored here but only compile-checked on Windows. Assertions use
`TestHelper.waitFor`/`assertWillBeTrue` (default 5000 ms) [CERT-doc devguide "Utility Methods"].

### H1 — `BFreezeAlarmStationTest extends BTestNgStation` (ColdRoomPan-rt) — CRA1/CRA2/CRA3 live

| Step | Code shape | Assert |
|---|---|---|
| configure | `super.configureTestStation(...)`; add `new BAlarmService()` under `station.getServices()`; add a `BEvaporatorUnit` under the station root | station starts |
| arm | drive the unit into freeze (same inputs `ColdRoomControl.freezeTrip` uses: temperature below the freeze limit for the configured time) | `unit.getFreezeTripped()` true (CRA2 source half) |
| CRA1 | `unit.getFreezeAlarmPt()` is the child `BBooleanPoint`; its `BAlarmSourceExt` (`getAlarmSourceExt()` or slot lookup) | `ext.getAlarmState()` reaches offnormal [CERT docSource alarm-rt `BAlarmSourceExt.java:386`] |
| CRA3 | `BAlarmService.getAlarmDb()` → `getDbConnection(cx)` → `getOpenAlarms()` cursor [CERT `BAlarmService.java:764`, `BAlarmDatabase.java:100`, `AlarmDbConnection.java:199`] | exactly one record whose `getSource()` ORD list names the ext and `getSourceState() == BSourceState.offnormal` [CERT `BAlarmRecord.java:256`, `BSourceState.java:56`] |
| recovery | raise the temperature past the deadband | the record (or a new one) shows `BSourceState.normal` (`BSourceState.java:54`); no second offnormal record |

### H2 — `BCompressorAlarmStationTest extends BTestNgStation` (CompPan-rt) — CPB5

| Step | Code shape | Assert |
|---|---|---|
| configure | AlarmService + a `BCompressorControl` with N=2 compressors, LP floor `suctionLowLimit > 0` | started() ran: `AlarmSupport` created, `AlarmEdge.reseed` called (W2/W4 are structural in WSL) |
| trip | suction below `suctionLowLimit` with `suctionValid` | one open alarm record, `getSourceState() == offnormal`, source ORD = the `BCompressorControl` (it `implements BIAlarmSource`) |
| hold | execute() again several times while still low | still exactly ONE record (edge-only; CPB2 pure pin is the WSL half) |
| recover | suction above limit + deadband | `toNormal` record: `getSourceState() == normal` |

### H3 — `BConfigLoginStationTest extends BTestNgStation` (DashboardPan-ux) — R14 lockout + AuditEvent

| Step | Code shape | Assert |
|---|---|---|
| configure | `super.configureTestStation`; `startWebServer()`; add the `BDashboardServlet` (+ `DashboardService` tree it serves); `addUser("operator2", <operator role with the OPERATOR_WRITE permission on the target>, "pw-ok")`; add a `BAuditHistoryService`-backed audit history (the service class is not in docSource — resolve on Windows via bajadoc, `history` module) | |
| lockout | POST `/api/config/login` 5× with a wrong password inside the lock-out window, then once with the right password | 6th answer is **401** (CL5 semantics) while `userService.getLockOutEnabled()`, `getMaxBadLoginsBeforeLockOut()==5`, `getLockOutPeriod()`, `getLockOutWindow()` are the defaults [CERT docSource baja `BUserService.java:225,256,287,319`]; the user's `getLockOut()` flag is set [CERT `BUser.java:412`] |
| after period | wait `lockOutPeriod` (or set it short in `configureTestStation`) and login correctly | 200 + session cookie |
| attributed write | POST a setpoint write with the session | 200; the audit history's newest record names `operator2` (never the kiosk user) — read it through the history ORD `history:/<stationName>/AuditHistory` and the record's user field |
| PR6 half | the same write without the session | 403 `config_login_required`; no audit record added |

## 4. Verdict rules

- Every H-class must print `Failures: 0, Skips: 0`; a `Skips: N` line is a REJECT of the run, not a pass.
- Record the exact Windows build string (`about` in the console), the client SHA, the three module
  versions read from the GROUP `build.gradle.kts` (`defaultModuleVersion`), and the three verbatim outputs.
- If a pin fails: it is a defect in the PR that owns it (table at the top), filed against client main; the
  WSL BLESS of that PR stands (its WSL contract held) and the fix is a follow-up PR with its own RED.

## 5. Run record

Write `qa/c9-harness-run.md` on the kit repo (the C9 close gate `CLOSE-harness-run` checks it exists and
carries three `Total tests run:` lines with `Failures: 0` and no `Skips: [1-9]`). Template:

```
# C9 harness run — <date>
Windows: OptimizerSupervisor-N4.14.0.162 (<about string>) · dev license OK
Client main: <sha> · CompPan 2.2.0 · ColdRoomPan 2.1.0 · DashboardPan 2.2.0
## test ColdRoomPan
<verbatim>
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
