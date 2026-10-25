<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-10-25 · kit · r10-scaffold-station-test

**Session**: niagara-tools quality round 10, `scaffold-module` emits a BTestNgStation skeleton for the logic and
dashboard types, branch `r10/scaffold-station-test` from 1551c6bb (origin/main)
**Delta count**: 4

## What happened
`scaffold-module.sh --type logic|dashboard` now copies `templates/station-test` into `<Mod>-rt`:

- `B<Mod>StationTestBase`, a `BTestNgStation` with Fox off. It tolerates only the exact `Fox service did not
  start in time` failure, and only while the station runs.
- `B<Mod>LinkTest`. For each slot in `LINK_IN_SLOTS`, LinkCheck accepts a link from a plain source, the slot is
  not READONLY, and a BLink carries a change to each double slot.
- `moduleTest-include.xml`.

The driver type is unchanged. Writing the guard exposed a real defect in the dashboard fixture: `BMinimalDash.status`
is documented as the slot a control output is linked into, but it was `SUMMARY | READONLY`. Niagara refuses every
property link whose target is READONLY (`LinkCheck.java:108-109`, `linkcheck.propReadonly`), and
`types/dashboard.md:12` already forbade it. It is now `SUMMARY` only. The logic fixture's gradle file gained
`junit:junit:4.13.2`; without it the overlay makes lint-structure L11 FAIL.

The one live boot was RED, not GREEN: the test NRE rejected the test module (`invalid test target(s)`). The gates
held (install unchanged, real user homes unchanged), so no second boot was spent. The GREEN boot is owed.

## Evidence
- RED before the overlay: TC-ST1, TC-ST2, TC-ST3 and TC-ST5 failed (no emitted files). TC-ST4 (driver gets no
  overlay) passed.
- RED before the truth fix, TC-ST6 (source-level twin of `linkInSlotsAreNotReadonly`): `status Flags.SUMMARY |
  Flags.READONLY`. Named mutation after the fix: adding `Flags.READONLY` to `setpoint` fails TC-ST6 again.
- Build against a mirror of a 4.15.3.28 install (plugin 7.6.22, a throwaway LocalSigningProfile): `ModuleA-rt.jar`
  and `ModuleA-rtTest.jar` built; slotomatic generated both test types' TYPE blocks; the test jar holds
  `BModuleAStationTestBase`, `BModuleALinkTest` and the plain JUnit `ModuleALogicTest`.
- The `-ux` part of the dashboard scaffold does not compile against 4.15.3.28: `BModuleAServlet.java:74 method does
  not override or implement a method from a supertype` (`doService(WebOp)`). This was not caused by this change. The
  runner builds only `-rt`.
- Runner rows: `module verification: low (copy only)` PASS; `testng: cannot read …/testng-results.xml` FAIL;
  `console line 65: invalid test target(s): the test NRE rejected the test module` FAIL; `install unchanged (747
  entries checked)` PASS; `install defaults/system.properties unchanged` PASS; `real user homes unchanged (59
  entries checked)` PASS; exit 1.
- The built test module's `module.xml` lists only `baja` and `test-wb` as dependencies, not `ModuleA-rt`, although
  `BModuleALinkTest` imports `com.example.ModuleA.BModuleA`. This is a candidate cause, not a proven one: the
  console log went with the scratch dir.
- `lint-link-target-flags` LTF1 stayed silent on the old `status`. In the Go rule (`linktargetflags.go`, `Check`),
  a non-comment, non-property line resets the comment block. The class javadoc that says the integrator links into
  `status` is followed by `@NiagaraType`, so that javadoc never heads the property run. A probe that rewrote the
  javadoc to "has linked from" (a `linkText` keyword) still exited 0. A `// status: link-in display slot` comment
  directly above the annotation made LTF1 FAIL at `BModuleA.java:36`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | The logic and dashboard scaffolds ship a Fox-off station-test fixture and a link-in sample test from `templates/station-test` | `templates/station-test`; `toolbelt/scaffold-module.sh`; `types/moduleTest.md` § Checklist | `[ev: code build-n4-module-kit/toolbelt/scaffold-module.sh]` |
| Δ2 | LTF1 should read a class javadoc above `@NiagaraType` as heading the property run, or the scaffold convention should put a link-in comment on each link-in slot | `go/internal/lint/rules/linktargetflags/linktargetflags.go`; `toolbelt/lint-link-target-flags.sh` header | `[ev: code go/internal/lint/rules/linktargetflags/linktargetflags.go]` |
| Δ3 | A FAIL run of `run-niagara-test run` should print the console lines around the rejection (or keep the console log) so one boot is enough to diagnose it | `go/internal/niagaratest/run.go` | `[ev: code go/internal/niagaratest/run.go]` |
| Δ4 | The driver scaffold's station test keeps Fox on a free port, which listens on every interface on the Windows NRE. A loopback-only policy needs the Fox-off fixture instead | `fixtures/MinimalDriver`; `types/driver-authoring.md` § 12.1 | `[ev: retro r10-run-niagara-test-testng Δ4]` |

## Lessons
- A guard that catches a real defect in the kit's own fixture is the point of the station test: the dashboard
  scaffold had shipped an unlinkable display slot since the first dashboard fixture.
- One boot is a scarce budget. Without the console log, a RED boot can only be classified, not diagnosed.

## Owed
- The GREEN boot of a scaffolded dashboard module: rerun with `--keep-home` and read the console for the
  rejection cause; check the missing `ModuleA-rt` dependency of the test module first.
- The `-ux` servlet compile failure on 4.15 (`doService(WebOp)` override).
- Δ2, Δ3 and Δ4 above.
