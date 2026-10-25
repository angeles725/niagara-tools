<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-25 · kit · r10-scaffold-station-test

**Session**: niagara-tools quality round 10, `scaffold-module` emits a BTestNgStation skeleton for the logic and
dashboard types, branch `r10/scaffold-station-test` from 1551c6bb (origin/main)
**Delta count**: 6

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

Both live boots were RED, not GREEN: the test NRE rejected the test module (`invalid test target(s)`). The gates
held every time. The second, diagnostic boot (`--keep-home`, runner at 1be6011a with the fail diagnostics) found
the cause, and it is in the session's own harness, not the scaffold: the throwaway signing profile was written
with `validity 30` and `refresh 15`, so every signing regenerated the key and both jars carried a signature block
that no JVM can verify. The NRE dropped both modules (`SecurityException: cannot verify signature block file
META-INF/THROWAWA`), so `getModules("ModuleA")` threw `ModuleNotFoundException` and TestRunner printed
`invalid test target(s)`. With the plugin's own defaults (`validity 365`, `refresh 183`) both jars are `jar
verified` offline. A third boot, from this branch's runner (with the Δ5 gate) and a 365/183 throwaway profile,
was GREEN: `testng: total 3, passed 3, failed 0, skipped 0` (the three `B<Mod>LinkTest` methods).

The offline checks also found that the dashboard `-ux` servlet never compiled, on 4.14 or 4.15: it declared
`@Override doService(WebOp)`, which is not a BWebServlet method. It now overrides `doGet` and `doPost`.

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
- First boot (runner on this branch's base) rows: `module verification: low (copy only)` PASS; `testng: cannot read …/testng-results.xml` FAIL;
  `console line 65: invalid test target(s): the test NRE rejected the test module` FAIL; `install unchanged (747
  entries checked)` PASS; `install defaults/system.properties unchanged` PASS; `real user homes unchanged (59
  entries checked)` PASS; exit 1.
- Second boot (kit-run-main 1be6011a, `--keep-home`): the same FAIL rows, plus `cleanup: kept <scratch>`,
  `gradle root unchanged (0 tracked changes before, same after)` and `diagnostics kept: <state>/20261025T092220Z-ModuleA.log`.
  Console: `Error processing module file ModuleA-rt.jar` and `… ModuleA-rtTest.jar`, both `java.lang.SecurityException:
  cannot verify signature block file META-INF/THROWAWA`, then `SEVERE [testrunner] invalid test target(s): ModuleA`
  with `javax.baja.sys.ModuleNotFoundException: ModuleA at RegistryDatabase.getModules`. Tail copied to
  `~/stationlogs/scf-20261025T092220Z-ModuleA-invalid-target.log`; the kept home (915 MB) was deleted at
  2026-10-25T09:27:38Z.
- The 30/15 profile came from this session's manual setup only; no kit code writes a signing profile (the runner,
  scaffold-module.sh and preflight only read or check the root signing block).
- The plugin's predicate (niagara-plugins 7.6.22, `LocalSigningProfile.kt`, `javap -c -l` in a scratch dir):
  `generateCert` refuses validity <= refresh (`:373-374`) and builds notAfter with `cal.set(Calendar.DAY_OF_MONTH,
  validity)` (`:386-388`, bytecode `iconst_5; getValidity; Calendar.set`), so the key ends on day `validity` of the
  current month, not `validity` days out; `getCertInfo` floors the days left to notAfter and re-mints when they are
  below `refresh` (`:216-225`). The model reproduces both observed keys exactly: 30 on 25 Oct gives notAfter 30 Oct
  (5 days left, below 15: re-minted at every signing), 365 gives 30 Sep 2027 (340 days left, above 183).
- Gate RED, on the merged origin/main runner: `TestRunRefusals/throwaway_signing_profile_re-mints_its_key_at_every_signing_(30/15)`
  `exit 0, want 2 with "niagara.signing.validity=30, niagara.signing.refresh=15"` and `8 copy/build/run calls after a
  refusal`. GREEN after `signingprofile.go`. Mutations, each failing the package tests: margin 32 -> 31 (the 214/183 case),
  the predicate `validity > refresh` (the plugin's own check, which passes 30/15), and the call removed from `validate`.
- Third boot (runner at 12f4fa5d, a fresh 365/183 keystore, scratch root a git worktree, `--module-verification low`,
  no `--keep-home`): preflight attribution x4 PASS; `module verification: low (copy only)` PASS; `testng: total 3,
  passed 3, failed 0, skipped 0`; `console line 27: run 3, failures 0, skips 0`; `console line 32: run 3, failures 0,
  skips 0 (all run tests)`; `verdict: 3 tests run, 0 failed, 0 skipped (testng-results)`; `install unchanged (747
  entries checked)`; `install defaults/system.properties unchanged (sha256 390e59e9)`; `real user homes unchanged (59
  entries checked)`; `gradle root unchanged (0 tracked changes before, same after)`; exit 0. The abstract fixture base
  was skipped as expected; the link test was the one class run.
- Offline: `jarsigner -verify` on the kept `ModuleA-rt.jar` failed the same way, so the jar was broken at build
  time. The keystore's cert read `Valid from 03:24:06 until Oct 30` (minted at that build). After rewriting the
  profile to 365/183 with a fresh keystore, two consecutive `clean jar moduleTestJar` builds gave `jar verified` for
  both jars (self-signed, which `low` accepts).
- The first runner refused the scratch root until it was a git worktree (`gradle root is not a git worktree: cannot
  verify it`, exit 3, before any copy); a `git init` with `build/` and `.gradle/` ignored satisfied it.
- The test module's `module.xml` has no dependency on `ModuleA-rt`. Bisected: the kit root's `niagaraSigning { aliases;
  signingProfileFile }` block drops it (plugin 7.6.22, same with the 4.14 mirror); a known-good client root without that
  block gets a `<dependency name="<Mod>-rt" …/>` line. The driver scaffold has the same gap and its test still ran 6/6
  on 4.14, so it was not the cause here; it stays an open question whether a 4.15 run needs it.
  **Erratum (2026-10-25):** this did not reproduce. A minimal rebuild of the logic, dashboard and driver scaffolds
  with the root block (plugin 7.6.22 on 4.15, 7.6.17 on 4.14) keeps `<dependency name="ModuleA-rt" …/>` in the
  test `module.xml`; `NiagaraModulePlugin.kt:202` adds the module itself as a moduleTest compileOnly dependency
  unconditionally and the plugin class references no signing. Removing only `niagaraSigning {}` while
  `allowDefaultProfile.set(false)` stays fails configuration, so the original comparison was not like-for-like.
- `-ux`: `:ModuleA-ux:compileJava` failed at `BModuleAServlet.java:74` (`method does not override or implement a method
  from a supertype`) on both the 4.15.3.28 and the 4.14.0.162 mirrors; after the fix it passes on both, with both
  `compileModuleTestJava` tasks.
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
| Δ5 | `run-niagara-test run` refuses (exit 2, before any copy, both values named) a LocalSigningProfile with validity - refresh < 32: the plugin sets a new key's notAfter to day `validity` of the current month and re-mints it when fewer than `refresh` days remain, so such a profile re-mints at every signing and leaves a signature block no JVM verifies, which the NRE reports only as a missing module. A throwaway profile uses the plugin defaults 365/183 | `go/internal/niagaratest/signingprofile.go`; `toolbelt/run-niagara-test.sh` header; `types/moduleTest.md` § Running from WSL | `[ev: code go/internal/niagaratest/signingprofile.go]` |
| Δ6 | The scaffold's `-ux` servlet overrides `doGet`/`doPost(WebOp)`, never `doService(WebOp)` | `fixtures/MinimalDash` `-ux`; TC-DASH3 | `[ev: code build-n4-module-kit/fixtures/MinimalDash/MinimalDash/MinimalDash-ux/src/com/angeles/MinimalDash/ux/BMinimalDashServlet.java]` |
| Δ4 | The driver scaffold's station test keeps Fox on a free port, which listens on every interface on the Windows NRE. A loopback-only policy needs the Fox-off fixture instead | `fixtures/MinimalDriver`; `types/driver-authoring.md` § 12.1 | `[ev: retro r10-run-niagara-test-testng Δ4]` |

## Lessons
- A guard that catches a real defect in the kit's own fixture is the point of the station test: the dashboard
  scaffold had shipped an unlinkable display slot since the first dashboard fixture.
- One boot is a scarce budget. Without the console log, a RED boot can only be classified, not diagnosed; with it,
  the cause was one line above the runner's summary.
- A plausible static cause (the missing self-dependency) was wrong; the console named the real one. Bisecting the
  build offline was still worth it: it found a second, real gap in the signing template.

## Owed
- Δ2 and Δ4 above (Δ3 landed as runner D6, #566; Δ5 is the runner gate in this branch).
