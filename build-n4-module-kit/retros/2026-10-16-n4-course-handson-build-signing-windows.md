<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-16 · kit · n4-course-handson-build-signing-windows

**Session**: hands-on close of the Tridium N4 Developer Certification Course (4.15.6.20, Windows host driven from WSL): Labs 5-11, Review 2, the Certification Exam and the Bonus Lab, built and run live; the kit gate re-run on the resulting jars.
**Delta count**: 6

## What happened
Every course module was built on Windows with `cmd.exe /c "gradlew.bat :<part>:jar"`, signed with the Lab 5
`CodeSigning` profile, and loaded live by a station running `niagara.moduleVerificationMode=high`. Running the
kit's own gate (`toolbelt/verify-module.sh`) on those five jars afterwards FAILed all of them: the `signed`
check only accepts `META-INF/NIAGARA4.SF`, the bytecode check counts multi-release classes that Java 8 never
loads, and a pipefail/SIGPIPE race reports 15 of 15 types missing on the one large (uberjar) module. Separately,
the New Module Wizard's root `build.gradle.kts` carries no signing configuration, so the same "add the Lab 5
block by hand" step recurred three times (Review module, exam root, the `trafficLightServer` sample), and the
jar-lock failure seen from the Windows route is not recognized by `build.sh`'s exit-32 pattern. The doc-mined
retros of 2026-10-15 predicted part of this; this retro records what the live run confirmed, contradicted or
added.

## Evidence
- [CERT-live] `verify-module.sh` on `Cristian_Angeles_Cert-rt.jar`, `websiteDriver-rt.jar`, `trafficLightDriver-rt.jar`,
  `trafficLightDriver-wb.jar` (copied from `C:\Niagara\Niagara-4.15.6.20\modules`, built 2026-10-16) → each
  `FAIL signed … META-INF/NIAGARA4.SF missing (jar not signed)`; summary `10 passed, 4 failed … -> FAILED`, exit 1.
  `unzip -l` shows `META-INF/CODESIGN.SF` + `META-INF/CODESIGN.RSA` in every one (entry name = signing alias).
  The same jars ran live: exam P1-P3 and Bonus Parts 1-4 verified in the station `[ev: engram obs 11009]` `[ev: engram obs 11479]`.
- [CERT-live] Lab 5 set the station to high mode with the course root CA trusted and `training-rt.jar` signed by
  `CodeSigning` `[ev: engram obs 11148]`; before Lab 5 the Pre-Course step had `moduleVerificationMode=low` at
  `defaults/system.properties:440` `[ev: engram obs 11018]` `[ev: engram obs 11155]`.
- [CERT-live] `verify-module.sh` on `training-rt.jar` (Labs 1-10, Jackson uberjar, ran live in Lab 6 `[ev: engram obs 11155]`):
  `FAIL bytecode … 9 of 1126 classes not major 52; first: META-INF/versions/11/com/fasterxml/jackson/core/internal/shaded/fdp/v2_21_2/BigSignificand.class major=55`
  and `FAIL types … 15 of 15 declared types have no class; first: com.tridiumuniversity.training.BFirstComponent`,
  although `unzip -l` lists `com/tridiumuniversity/training/BFirstComponent.class` (entry 9 of the list).
- [CERT-live] Root cause of the types FAIL, reproduced in bash: with `set -euo pipefail` (verify-module.sh:43),
  `printf '%s\n' "$LIST" | grep -qx "<class>"` returns 141 three times out of three on the 79,624-byte entry list
  of `training-rt.jar` (grep exits on the early match, printf takes SIGPIPE), and succeeds on the small
  `websiteDriver-rt.jar` list. Same shape at verify-module.sh:214 (`signed`), :223 (`types`) and :411 (`palette`).
- [CERT] The sibling lint already skips multi-release entries: `toolbelt/lint-bundled-jar-class-version.sh` header
  "META-INF/versions/ entries are skipped to avoid false positives"; `check_bytecode_major` (verify-module.sh:202-212) does not.
- [CERT] Wizard root template `devkit-wb.jar!gradle/build.gradle.kts.vm` applies `id("com.tridium.niagara-signing")`
  (:18) and contains no `signingServices` / `niagaraSigning` block; the course ships the block separately as
  `Labs/Lab5 Code Signing/signing_configuration.kts`. The kit has no `allowDefaultProfile` / `niagaraSigning`
  text anywhere under `toolbelt/`, `types/`, `build-verify.md` or `METHODOLOGY.md` (grep, 0 hits).
- [CERT-live] Three roots needed the block added by hand: `CertificationExam/build.gradle.kts:47-58`,
  `N4_Labs/trafficLightServer/build.gradle.kts:47-58`, `Core-Class/Review/build.gradle.kts:47-56`; the Review module
  was "unsigned under high mode" on 2026-10-14 `[ev: engram obs 11148]` and carries `CODESIGN.SF` from 2026-10-15 on.
- [CERT-live] The shipped `Labs/Lab11 NDriver/trafficLightServer.zip` `gradle.properties` has
  `niagara_home=c:\\niagara\\niagara-4.15.4.24` and `niagara_user_home=C:\\Users\\<username>\\Niagara4.15\\<brand>`,
  no signing block, and ships `trafficLightServer-rt/build/` directories; it was retargeted to 4.15.6.20 and the
  installed user home `[ev: engram obs 11477]`.
- [CERT-live] Jar-lock failure from the Windows route, exam P1 (transcript 2026-10-16):
  `kotlin.io.FileAlreadyExistsException: C:\…\build\libs\Cristian_Angeles_Cert-rt.jar -> c:\niagara\niagara-4.15.6.20\modules\Cristian_Angeles_Cert-rt.jar: Tried to overwrite the destination, but failed to delete it.`
  `printf '%s\n' "<that line>" | grep -cE 'FileAlreadyExistsException.*modules/.*\.jar'` → 0 (the build.sh:255
  pattern); with `modules[/\\]` → 1. The holder was the running station `[ev: engram obs 11009]`; the `java.exe`
  processes left after closing the station were Gradle daemons `[ev: engram obs 11478]`.
- [CERT-live] Bare `gradlew jar` in a multi-module root builds every subproject, so a jar locked by the station for
  an unrelated module fails the run (live 2026-10-16, reported by the session); every successful course build
  used qualified tasks (`:websiteDriver-rt:jar`, `:trafficLightDriver-wb:jar`). `build.sh:178-180` already qualifies its tasks.
- [CERT-live] `NETWORK_COMMUNICATION hosts="*" ports="*"` authored in `module-permissions.xml` appears inlined in
  `META-INF/module.xml <permissions>` of `websiteDriver-rt.jar` and `trafficLightDriver-rt.jar`, with no separate
  jar entry — confirms kit core `METHODOLOGY.md:65` (no delta needed).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | CORRECT the `signed` check: PASS when the jar has a `META-INF/<ALIAS>.SF` with a matching `<ALIAS>.RSA`/`.DSA`/`.EC` (the entry is named after the signing alias: `NIAGARA4` for the default profile, `CODESIGN` for the course `CodeSigning` alias, `SERVER1` for Honeywell); report the alias in the row detail. Drop the "OEM jars are expected to FAIL" note at build-verify.md:131 and reword build-verify.md:96/:103, METHODOLOGY.md:67 and BUILD-STATE.md:53 to "a `META-INF/*.SF` signature block". Confirms `devguide415-r2-packaging-platform-signing` Δ6 (the station decides by mode + chain, not by entry name). Lint: YES. Acceptance: the four course jars above PASS `signed` with detail `CODESIGN`; a jar with no `.SF` still FAILs; a `.SF` without its key file FAILs; new bats rows for all three. | `toolbelt/verify-module.sh` `check_signed` (:213-215) + `build-verify.md` § Verify (:96, :103) + § Signing per deploy target (:131) + `METHODOLOGY.md:67` | `[ev: live 2026-10-16 verify-module.sh on course jars]` |
| Δ2 | FIX the pipefail/SIGPIPE false negatives: replace every `printf '%s\n' "$LIST" \| grep -q…` membership test with a here-string (`grep -qx -- "$f" <<<"$LIST"`) or a one-time associative set, in `check_signed`, `check_types_have_classes` and `check_palette`; sweep the other toolbelt scripts that pipe a captured listing into `grep -q` under `pipefail`. This is a kit-check defect: open a `kit-ticket.sh` line for it. Lint: YES (self-test). Acceptance: `training-rt.jar` reports `PASS types 15 declared types resolve`; a bats fixture jar whose entry list exceeds 64 KiB with the type class near the top PASSes three runs out of three; the named mutation "restore the pipe" turns it red. | `toolbelt/verify-module.sh` (:214, :223, :411) + `tests/` new bats row | `[ev: live 2026-10-16 bash repro rc=141]` |
| Δ3 | ALIGN the bytecode check with `lint-bundled-jar-class-version.sh`: skip `META-INF/versions/**` entries in `check_bytecode_major` (a Java 8 runtime never loads them) and print their count as detail (`… all major 52; 9 multi-release classes ignored`). Confirms `n4-course-authoring-files-and-libraries` Δ1 (Path 0 `uberjar()` is the working route: `uberjar("com.fasterxml.jackson.core:jackson-databind:2.21.2")` embedded databind + core + annotations, 1107 classes, and ran live) and refines its "pin to Java-8 builds (Jackson 2.x)" clause: Jackson 2.21.2 is a multi-release jar that still runs on Java 8. Lint: YES. Acceptance: `training-rt.jar` PASSes `bytecode`; a fixture with a major-55 class OUTSIDE `META-INF/versions/` still FAILs. | `toolbelt/verify-module.sh` `check_bytecode_major` (:202-212) + `types/third-party-libraries.md` § packaging paths | `[ev: live 2026-10-16 verify-module.sh training-rt.jar]` |
| Δ4 | Confirms and extends `n4-course-build-deps-permissions-signing` Δ4: the lint must read the ROOT `build.gradle.kts` (the folder chosen in the New Module Wizard becomes the Gradle root project; a GROUP-only check misses it) and WARN when `com.tridium.niagara-signing` is applied without BOTH `signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }` and `niagaraSigning { aliases.set(…); signingProfileFile.set(…) }`; add the block as `templates/signing-root.gradle.kts` and make `scaffold-module.sh` emit it. Lint: YES. Acceptance: the wizard template `devkit-wb.jar!gradle/build.gradle.kts.vm` WARNs; the three course roots (exam, server, Review) PASS; a root with the block but `allowDefaultProfile` missing WARNs. | `toolbelt/lint-structure.sh` + `build-verify.md` § Signing per deploy target + `templates/` + `toolbelt/scaffold-module.sh` | `[ev: code devkit-wb.jar!gradle/build.gradle.kts.vm:18]` |
| Δ5 | Make the jar-lock diagnosis route-independent: build.sh:255 pattern `FileAlreadyExistsException.*modules[/\\].*\.jar` (the Windows `gradlew.bat` route prints backslash paths and the Kotlin message "Tried to overwrite the destination, but failed to delete it"); in BUILD-LOOP §4 exit-32 and build-verify.md § Raw gradle add: always run QUALIFIED tasks (`:<MOD>-<part>:jar`), because a bare `gradlew jar` builds every subproject and fails on an unrelated jar the station holds; `java.exe` processes left after the station closes are Gradle daemons (`gradlew --stop`), so identify the real holder with `tasklist` for `station.exe`/`wb.exe`. Lint: YES (pattern). Acceptance: a bats row feeds the captured exam line to the exit-32 classifier and gets 32; the LF/slash variant still gets 32. | `toolbelt/build.sh` (:255) + `BUILD-LOOP.md` § 4 Post-jar copy lock (:131) + `build-verify.md` § Raw gradle (:30) | `[ev: transcript 2026-10-16 exam P1 FileAlreadyExistsException]` |
| Δ6 | Add a "sample / course project retarget" preflight: before building a project the user did not generate (course zip, vendor sample), check that `gradle.properties` `niagara_home` exists on disk and matches the installed version, that `niagara_user_home` contains no `<…>` placeholder, that the ROOT has the signing block (Δ4), and that no shipped `build/` directory is present (delete it). Extends `n4-course-as-exemplar-source` Δ4 (which covers only the missing slot-o-matic region). Lint: YES. Acceptance: the original `trafficLightServer.zip` contents produce three FAIL rows (4.15.4.24 home absent, `<username>` placeholder, no signing block) and one WARN (shipped `build/`); the retargeted tree passes. | `toolbelt/preflight.sh` (new check) + `build-verify.md` § Build target & plugin version | `[ev: code Labs/Lab11 NDriver/trafficLightServer.zip!gradle.properties:5,8]` |

## Lessons
- Run the kit gate on jars that are KNOWN good (loaded live, signed, trusted) before trusting a FAIL from it: today
  it rejected every course jar for reasons a station does not care about.
- A membership test written as `printf … | grep -q` under `pipefail` is a size-dependent false negative; it passes
  on small fixtures and fails on real uberjars, so the fixture must be larger than a pipe buffer.
- The signature block is named after the signing alias; any gate keyed to one file name encodes one machine's profile.
- The wizard root is the place signing must be configured; a check that only reads GROUP build scripts never sees it.
- Diagnose the lock holder by process name; leftover `java.exe` after a station closes are Gradle daemons.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — carried by issue #260 into the folded retro `2026-10-22-n4-course-signed-jars-gate-fix` and its follow-up `2026-10-22-n4-course-sample-check-precision`: Δ1 `types/logic-authoring.md:409`, Δ2 `build-verify.md:99` (here-strings, `tests/pipefail-grep.bats`), Δ3 `build-verify.md:99` (`META-INF/versions/**` skipped and counted), Δ4 `build-verify.md:134`, Δ5 `build-verify.md:51` (`toolbelt/build.sh:310` route-independent exit 32), Δ6 `build-verify.md:58` (`toolbelt/preflight.sh --sample`, `sample-build` row at `toolbelt/preflight.sh:367`). INDEX row generated from the header: `| 2026-10-16-n4-course-handson-build-signing-windows.md | kit | 2026-10-16 | folded | 6 |`
