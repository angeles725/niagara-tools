# Build & verify — Java 8 + slotomatic (the ONLY valid build)

A `gradle :jar` with the default JDK is NOT a valid N4 build: it can use the wrong Java (major-65 bytecode Niagara's Java-8 runtime rejects) and it skips slotomatic, so annotation bugs never surface. Chihuahua's `deploy.sh` uses `JAVA8=/usr/lib/jvm/java-8-openjdk-amd64`.

## Doctrine — which command does what

| Command | Role |
|---|---|
| `toolbelt/verify-module.sh` | **THE verify gate** — run it on the built jars regardless of who built them. |
| `toolbelt/build.sh` | The recommended WSL build: clean + slotomatic (for every profile that has sources) + jar, then it calls the gate automatically. |
| `scripts/ng-deploy.sh` | The station DEPLOY wrapper: backup → build → prechecks (version bump; schema-risk in modes A/C) → type-count verify → `verify-module.sh` gate (modes A/C) → copy; every check runs on `build/libs` before the copy, so a failing one copies nothing. `NIAGARA_HOME` must be a mirror (a real one is exit 10 unless `--install-into-home`), and a failing check also puts `<NIAGARA_HOME>/modules` back as it was before the build (§Station deploy). Mode **D** deploys `-rt` + `-wb` (the `driver` scaffold's pair; `EXPECTED_WB_TYPES`; same checks and gate as A/C) `[ev: retro build-kit-bench-session Δ5]`. Slotomatic is opt-in (`--with-slotomatic`) in modes A/C: -rt, plus -ux in mode A when -ux is annotated; mode B runs none (run `verify-module.sh` on `build/libs` before a mode-B deploy). |

> A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.

The header of `toolbelt/verify-module.sh` lists every check the gate runs (21: `bytecode` to `driver`), in run order; `TestHeaderListsEveryCheck` (`go/internal/verifymodule/header_test.go`) fails when the header and the registered checks (`checks.go`) differ. `[ev: retro doc-coverage-audit Δ8]`

`verify-module.sh` default checks are conservative (major 52, a `META-INF/*.SF` signature block, module.xml types resolve); `--target-version`, `--stored`, `--src` and `--expected-signer` are opt-in.

`build.sh` usage: `build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--ui-profile hmi|lan|both|unknown] [--legacy] [--first-deploy | --schema-baseline-ref <deployed-ref>] [--gradle-prop k=v]... [--install-into-home] <module-root> <MOD> [niagara_home]`, or `build.sh --restore-mirror <mirror>/.mirror-replaced/<MOD>-<profile>.jar.<sha12>` (§Mirror: the replaced jar is kept) (exit **53**: the report-module verdict is INCOMPLETE — a floor check SKIPped; a missing `.deploy-baseline` SKIP row names the command that creates it, and `--schema-baseline-ref <deployed-ref>` runs it for the build, §Schema-risk deploy workflow; see BUILD-LOOP §5) — exits **0** build+gate passed · **1** a step the chain cannot run without failed (the temporary directory, the `--schema-baseline-ref` extraction) `[ev: retro build-kit-bench-session Δ20]` · **2** usage · **10** environment (no JDK 8, not a niagara_home, a real `niagara_home` without `--install-into-home` (§Building against a running station), no profile, preflight FAIL; a module root that does not exist or cannot be entered names the path as typed: `build.sh: module root not found: <path>`, `build.sh: cannot enter module root: <path>` `[ev: code go/internal/buildmodule/messages.go]`; a repo root passed instead of the gradle root lists the gradle roots below it `[ev: retro comppan-fase2-amps-alarms Δ2]`) · **30** gradle failed · **31** `:clean` blocked by a locked `modules/<jar>` · **32** the plugin's post-jar copy into `modules/` hit `FileAlreadyExistsException` (a Windows process holds the old jar; the `build/libs` jar rebuilt this run is the artifact of record, the gate did not run) `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]` · **33** a mirror `niagara_home` path, or in mirror mode a `$HOME` that `-Dmaven.repo.local` would carry, contains whitespace (§Mirror; gradle never ran) `[ev: retro r8-mirror-userhome Δ1]` `[ev: retro r8-buildsh-316-308]` · **34** gradle passed but Slot-o-Matic skipped a file (`FAIL slotomatic-skip <path> (<reason>)` rows; the pre-build `modules/` jar is restored and the gate does not run; METHODOLOGY K15) `[ev: retro r10-slotomatic-skip-fail-open Δ1]` · **50** gate or report-module failed · **51** deployed-baseline drift (vendorVersions compare numerically per component, a missing component = 0, so 1.3 → 1.3.0 is no bump; a LOWER version than the deployed jar is a WARN `vendorVersion went down`, not a FAIL — a deliberate rollback is legitimate; one comparator, `go/internal/vendorversion`, shared with `ng-deploy.sh`) · **52** the drift gate cannot verify: a deployed or built jar it cannot read (not a zip, no entry, two entries under one name, an entry that fails its CRC), or a changed jar whose own vendorVersion is unreadable; the pre-build jar is restored and the verify gate does not run. The gate's content hash covers every entry outside `META-INF/` in name order (name, size and bytes), so a renamed entry or bytes moved between entries are a change; it is taken and compared within one run and is never stored. `[ev: code go/internal/buildmodule/baseline.go]` `--plugin-version` is also forwarded to `preflight.sh --plugin-version` `[ev: retro continuous-fan-post-defrost-delay Δ1]`; `--ui-profile`/`--legacy` are forwarded to `report-module.sh --profile`/`--legacy`. It runs `verify-module.sh --src <module-root>/<MOD>` on every produced jar; `--plugin-version` (or `$NIAGARA_PLUGIN_VERSION`) forwards `-PniagaraPluginVersion` (each install ships one: 4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22).

## Station deploy: `ng-deploy.sh` (the established wrapper)

Use `niagara-tools/scripts/ng-deploy.sh` — it already does backup → build → (slotomatic) → prechecks → verify + gate → copy-to-station (nothing is copied when a check fails), and knows the flags:
```
scripts/ng-deploy.sh --mode A --with-slotomatic        # rt+ux, regenerate from annotations, deploy+verify
scripts/ng-deploy.sh --strict-slotomatic               # ABORTS (exit 15) if annotation changes but slotomatic wasn't run
scripts/ng-deploy.sh --no-deploy --with-slotomatic     # build-only (jars stay in build/libs)
```
Set env in `.env.local` (or `--env-file`): `MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME, JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64, STATION_MODULES_DIR`.
`--strict-slotomatic` is the guard that catches the annotation-vs-generated bug automatically — prefer it.
Its version-bump precheck uses the same comparator and requires the built vendorVersion to be HIGHER than the deployed jar's when their content differs: same version (1.3 → 1.3.0 included) exits **51**, unreadable or a differing non-numeric suffix exits **52**, a downgrade exits **54**.
- **A deploy can be repeated unchanged: "same content" is not "same bytes".** Two builds of the same sources by the vendor plugin are never byte-identical: the `buildMillis` attribute of `<module>` in `META-INF/module.xml` (confirmed in a built kit fixture jar: `… installable="true" buildMillis="<epoch ms>" buildHost="…"`), the manifest's per-entry `SHA-256-Digest` of that file, and the signature files differ. Under the same vendorVersion the precheck therefore compares the set of entry names and the bytes of every entry except exactly: the signature files directly under `META-INF/` (`*.SF`, `*.RSA`, `*.DSA`, `*.EC`, any case), the `*-Digest` attributes of the manifest's per-entry sections (its main section and any other per-entry attribute are compared), and the values of `buildMillis` and `buildHost` on the root element of `module.xml` (when and on which machine: a rebuild on another machine is the same content). Those two values are cut out of the root element's start tag wherever they stand among its attributes (`withoutBuildStamp` reads the tag with `encoding/xml`); every other byte of the descriptor is compared as it is, so the same names on a nested element, the attribute order and the quotes are content. Jars that differ in nothing else deploy with `version-bump ok (<jar>: vendorVersion still <v>, and the same content as the deployed jar apart from the build time and host, the manifest digests and the signature files)`; one changed class, an added or removed entry, or any other byte is still exit **51**; a jar whose content cannot be read (not a zip, no entry, a duplicate entry name, a CRC failure, a `module.xml` with no root element) or that is over a limit of the comparison (`maxJarContent`: 2 GiB of declared content per jar; `maxDescriptor`: 8 MiB for `MANIFEST.MF` and for `module.xml`; entries are hashed in 64 KiB chunks, never read whole) is exit **52** `version-bump precheck cannot verify <jar>: cannot compare its content with the deployed jar: …`, never a pass. `build.sh`'s drift gate keeps its own, wider rule (every entry outside `META-INF/`). Code: `sameContent`; pinned by `TestAnUnchangedRebuildPassesTheVersionPrecheck`, `TestChangedContentUnderTheSameVersionIsStillExit51`, `TestAJarThatCannotBeReadIsNotTheSameContent`, `TestTheBuildTimeAndHostOfTheDescriptorAreLeftOutWhereverTheyStand` and `TestAJarOverTheLimitsOfThePrecheckCannotBeVerified`. `[ev: code go/internal/ngdeploy/jarcontent.go]` `[ev: code go/internal/ngdeploy/version_content_test.go]` `[ev: retro build-kit-bench-session Δ3]`
- **The deploy gate expects the signer `build.sh` expects.** `ng-deploy.sh` runs `verify-module.sh --src <module dir> --expected-signer <signer> <jars>` with the signer resolved by the same code as `build.sh` (`modjar.ExpectedSigner`): the `niagaraSigningAlias` gradle property (`ORG_GRADLE_PROJECT_niagaraSigningAlias` in the environment or the env file, else `GRADLE_USER_HOME/gradle.properties`, else `gradle.properties` beside `gradlew`; the deploy has no `--gradle-prop`), else the one signer of the module's jars of the mode that `STATION_MODULES_DIR` holds before the copy (the jars the station runs), else, when those give none (none signed, two signers, or only the development signer), of those that were in `<NIAGARA_HOME>/modules` before the build (the station's are read in addition to the home's, never instead of them; the two are never mixed in one decision), never the default development signer `NIAGARA4`; a station jar whose signer cannot be told (unreadable, not a zip, a `.SF` without its block file) does not decide: the home's jars give the signer (`WARN the signer of <jar> cannot be told (<why>), so the jars of <NIAGARA_HOME>/modules give the expected signer`); with none there either, the gate accepts any signature and the WARN says so `[ev: retro build-kit-bench-session Δ24]`. A jar signed with another key fails the gate and the deploy is exit **50**. With no signer to expect the gate accepts any signature and the deploy prints, after the gate's rows, the warning `build.sh` prints with its own remedy: `[ng-deploy] WARN no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature: … (set it: ORG_GRADLE_PROJECT_niagaraSigningAlias=<alias> in the env file, or niagaraSigningAlias=<alias> in gradle.properties)`; the exit code is unchanged. Pinned by `TestTheGateIsToldTheSignerBuildShExpects`. `[ev: code go/internal/ngdeploy/gate.go]` `[ev: code go/internal/modjar/signer.go]` `[ev: retro build-kit-bench-session Δ2]`
- **"A failing check copies nothing" holds for `<NIAGARA_HOME>/modules` too.** The vendor plugin's `:jar` step installs the built jars there before any check of the deploy has run, and in a real installation that directory is a station's. `ng-deploy.sh` therefore (a) refuses, before the backup and before gradle (slotomatic included), a `NIAGARA_HOME` that is a directory without `.niagara-mirror`: exit **10**, `NIAGARA_HOME <path> is a real niagara_home (no .niagara-mirror): … build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home`; the option is the one `build.sh` has, it is read from the command line only (`INSTALL_INTO_HOME=1` in the env file is exit 10 without the flag); (b) in a mirror or with the opt-in, moves the module's own `-rt/-ux/-wb` jars in that directory aside (`<jar>.ng-deploy-old`, the rename the copy uses) just before the build, and when the build, a precheck, verify, the gate or the copy fails, or SIGINT/SIGTERM arrives before the copy (exit 130/143), moves them back and removes the jars the build added, with `NOTICE: the jars the build installed in <dir> were taken out again: it holds the module jars it held before the build`; a jar that cannot be moved back or removed is an ERROR line naming each jar (`<jar> is the NEW jar and the previous one is in <jar>.ng-deploy-old (…)`), the exit code staying the failed check's; a `.ng-deploy-old` file a killed run left there is refused before gradle (exit 10, `build refused: …`) and never deleted; (c) when `STATION_MODULES_DIR` is that same directory (symbolic links resolved), says `STATION_MODULES_DIR is <NIAGARA_HOME>/modules (<dir>): …`, runs the version-bump precheck against the jar that was there before the build, and copies nothing: the copy step checks that the directory holds the jars of `build/libs` (exit 40 otherwise, previous jars put back); `--no-deploy` (or `NO_DEPLOY=1`) there is exit **10** before the backup and before gradle (`--no-deploy refused: STATION_MODULES_DIR is <NIAGARA_HOME>/modules (<dir>): …`), since it would keep jars no check has passed in the station's directory `[ev: retro build-kit-bench-session Δ23]` `[ev: retro p1-build-deploy-safety Δ2]`. After a deploy that went through, and after `--no-deploy` into another station directory, the built jars stay in `<NIAGARA_HOME>/modules`. A `NIAGARA_HOME` that names no directory, or names a file, is exit **10** before the backup and before gradle (`NIAGARA_HOME <value> names no directory: …` / `… is not a directory: …` / `… cannot be looked at (Permission denied): …` for a path the deploy may not search; gradle gets it absolute; `homeIsDir`, pinned by `TestANiagaraHomeThatIsNotADirectoryIsRefusedBeforeGradle`) `[ev: retro build-kit-bench-session Δ25]`. Code: `guardHome`, `asideHome` and `settleHome`; pinned by `TestACheckThatFailsPutsTheHomeJarsBack` and the tests beside it. `[ev: code go/internal/ngdeploy/home.go]` `[ev: code go/internal/ngdeploy/home_test.go]` `[ev: retro build-kit-bench-session Δ1]`
- **The env file cannot switch the verify gate off, and cannot replace a check.** A line of the env file that would skip a guarded step by itself is exit **10**, naming the file and the line, unless the command line says the same thing: `NO_GATE=1`, `NO_BACKUP=1` and `INSTALL_INTO_HOME=1` (each lifted by its own flag), `MODE=B` (the mode with no verify-module gate; lifted by `--mode`, which then wins) and a non-empty `VERIFY_MODULE_BIN`, which would put any executable in the place of the gate (lifted by `--no-gate`, since the gate does not run then) `[ev: retro p1-build-deploy-safety Δ3]`; a `SCHEMA_RISK_BIN` there, which would put any executable in the place of the schema-risk precheck, is refused with no flag to lift it, and only the command's environment sets it `[ev: retro p1-build-deploy-safety Δ5]`. A CRLF env file does not slip `MODE=B` past the guard (`envfile.TrimCR` strips the CR while reading; `TestACRLFEnvFileMayNotSkipTheGateWithModeB`). Code: `guardedSkip`, `replacesSchemaTool` (`go/internal/ngdeploy/envguard.go`), pinned by `TestAnEnvFileMayNotSkipTheGateWithModeBOrAnotherGate`. `[ev: code go/internal/ngdeploy/envguard.go]`

## Raw gradle (quick WSL iteration, no deploy)
```
cd <module-root>
J8=/usr/lib/jvm/java-8-openjdk-amd64            # confirm: ls /usr/lib/jvm
./gradlew :MOD-rt:clean :MOD-ux:clean \
          :MOD-rt:slotomatic :MOD-ux:slotomatic \
          :MOD-rt:jar :MOD-ux:jar \
  -Pniagara_home=<niagara_home> \
  -Porg.gradle.java.installations.paths=$J8
```
- `slotomatic` = "Run Slot-o-matic on a module" — regenerates the AUTO region from `@NiagaraProperty` annotations. Run it so a regen can't reintroduce a fixed bug.
- **Reference sources (course labs, vendor samples) arrive WITHOUT the generated region — run `:<part>:slotomatic` before `:<part>:jar`:** 0 of the 47 course lab `.java` files carry a `BEGIN BAJA AUTO GENERATED` region, and the course's own loop is transcribe → `gradlew slotomatic` → fix the errors → `gradlew jar`. `jar` alone compiles classes that lack the generated `TYPE`/accessors `[INFER]`; `build.sh` already runs slotomatic first. The IDE stays red until slot-o-matic runs and every supertype module is a dependency (`types/module-wiring.md` § 1.1). Never pre-write empty markers (METHODOLOGY K15). `[ev: course Labs/Lab Instructions.pdf p81]` `[ev: retro n4-course-as-exemplar-source Δ4]`
- **The complement: a source that HAS the generated region must not also hand-write `TYPE`/`getType()` outside it.** Slot-o-Matic writes both into the region, so a `@NiagaraType` class carrying a hand-written `static final Type TYPE` or `getType()` beside the region fails `compileJava` with `variable TYPE is already defined` after `slotomatic` runs. A module only ever built without slotomatic hides it. Delete the hand-written pair and keep the generated region. `lint-slotomatic-markers.sh` FAILs it as `type-outside-region` (run by `report-module.sh`). `[ev: retro r10-client-gate-retro Δ1]`
- **Warning-clean proof: `toolbelt/build.sh --werror`** adds `-Xlint:all -Werror` to every `JavaCompile` through
  `--init-script toolbelt/lib/werror.init.gradle`, off by default, so a module that some vendors build with `-Werror`
  stays warning-clean; the fix is the warning, or a scoped `@SuppressWarnings`. `[ev: corpus B1419 §1419.4 b]` The args
  go on at `taskGraph.whenReady` and only when absent, so a build script or plugin that replaces `options.compilerArgs`
  cannot drop them. Proven on Gradle 7.6 + JDK 8 (a build script that sets `options.compilerArgs = [...]`: raw-type
  warnings fail the build; a clean source passes; an up-to-date compile recompiles); Gradle 4.0.2 is not run here, the
  script uses only API both share.
  `[ev: retro course-coverage Δ22]`
- On WSL, `gradle.properties` may pin a Windows JDK path (`C:\Program Files\Zulu\zulu-8`) — that breaks WSL (`Illegal character ... C:\`). Override with `-Porg.gradle.java.installations.paths=$J8`. On Windows the pinned Zulu 8 is correct.
- **`build.sh`↔`gradle.properties` JDK split:** `toolbelt/build.sh` line 74 sets
  `GARGS=(-Pniagara_home=… -Porg.gradle.java.installations.paths="$J8")` — the WSL gate
  **overrides** whatever `installations.paths` says in `gradle.properties`. The
  `gradle.properties` JDK path therefore governs the **Windows Workbench build only**; the
  WSL gate ignores it. A **commented** (auto-detect) `org.gradle.java.installations.paths`
  block still produces a major-52 WSL build (build.sh supplies the path) — but is **not
  reproducible on a Windows `gradlew` build**, where Gradle falls back to auto-detect and
  may pick the wrong JDK major. Set the JDK path in the user-level `gradle.properties` of the
  Windows machine (`BUILD-LOOP.md` bring-up step 2, R10-D8), not in the repository's.
  `[ev: build.sh:74]` `[ev: corpus B1016]` `[ev: retro r10-user-level-gradle-keys Δ1]`
- **Always run QUALIFIED tasks (`:<MOD>-<part>:jar`), on WSL and on Windows (`gradlew.bat`) alike:** a bare `gradlew jar` builds every subproject of the root, and the post-jar copy fails on an UNRELATED jar that a running station holds. The copy-lock signature differs by route — WSL prints `java.nio.file.FileAlreadyExistsException: …/modules/<jar>`, the Windows route prints backslash paths and the Kotlin message `kotlin.io.FileAlreadyExistsException: …\modules\<jar>: Tried to overwrite the destination, but failed to delete it.`; `build.sh` classifies both as exit 32. To find the holder on Windows, `tasklist` for `station.exe` / `wb.exe`: `java.exe` processes left after the station closes are Gradle daemons (`gradlew --stop`), not the lock. `[ev: retro n4-course-signed-jars-gate-fix Δ5]` `[ev: retro n4-course-handson-build-signing-windows Δ5]`
- **Windows-native route from WSL, for any Gradle root on the Windows side (course labs, vendor roots, `C:\` clones):** `cmd.exe /c "cd /d <win root> && gradlew.bat :<part>:slotomatic"`, then `:<part>:jar`. A root already under `/mnt/<drive>/` also runs in place: `cd /mnt/c/<path> && cmd.exe /c "gradlew.bat <qualified tasks>"` (cmd.exe inherits a drive-letter cwd); copy the tree to a Windows folder only when the root lives on ext4, since `cmd.exe` rejects a `\\wsl.localhost` cwd. The route uses the root's own `gradle.properties` (`niagara_home`, plugin version, JDK) and `jar` copies into `<niagara_home>\modules`: back up the installed jar first and close Workbench/stations that hold it. It is the only route where `niagaraTest` discovers tests (`types/moduleTest.md` § Running from WSL). `[ev: course Labs/Lab10 Jobs]` `[ev: retro review2-signing-build-route Δ4]` `[ev: retro n4-course-handson-workflow Δ3]`
- **The Windows route picks its JDK from Gradle's toolchains, not from `build.sh`:** the 4.15 course `gradle.properties` leaves `org.gradle.java.installations.*` commented and says a build needs a full JDK 8; list what Gradle sees with `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` and set `paths` when no JDK 8 is listed. `preflight.sh` `jdk8` checks only the WSL JDK that `build.sh` passes (`go/internal/preflight/jdk.go:16-62`); `preflight.sh --toolchains` checks the WSL/POSIX `gradlew` route only (a POSIX pin, or the POSIX `gradlew`, which auto-detects the WSL JDKs, not the Windows JDKs `gradlew.bat` sees) and FAILs when no full JDK 8 is pinned or listed (`go/internal/preflight/toolchains.go`); a root with a `gradlew.bat` gets a SKIP row saying the Windows route is not checked. For the Windows route run the listing above with `gradlew.bat` on the Windows side and read it yourself. `[ev: retro course-coverage Δ25]` `[ev: course Slide Code/gradle.properties:14-25]` `[ev: corpus B1235 §1235.2]` `[ev: retro n4-course-build-deps-permissions-signing Δ5]`

## Build target & plugin version
- **Build against the `niagara_home` of the LOWEST target version you must support:** the manifest stamps its `baja` dependency version, and a 4.14 station REJECTS a jar built against 4.15. Check `unzip -p <jar> META-INF/module.xml | grep baja`. [ev: retro coldroompan-rt-hardening #6]
- **The gradle plugin version pinned in `settings.gradle.kts` MUST exist in `<niagara_home>/etc/m2` — each install ships exactly ONE (table below):** Tridium plugins are not on the public portal, so a version the install lacks fails "plugin not found". Make it overridable — `val gradlePluginVersion = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")` + `-PniagaraPluginVersion=<v>`. This override is MANDATORY, not optional. [ev: retro coldroompan-rt-hardening #7]
- **Every gradle group of a client repo must keep the pin overridable — `lint-structure.sh` L14 FAILs a `settings.gradle.kts` that hardcodes `val gradlePluginVersion: String = "x.y.z"`:** a sibling group with a literal pin ignores `-PniagaraPluginVersion` (and `build.sh --plugin-version`) and silently builds against the plugin its default names. The same retro's second half — a client's own per-group build wrapper naming ONE shared default `niagara_home` for every group — is client-repo doctrine: give such a wrapper a per-group `niagara_home` + plugin table. `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ2]`
- **`gradle.properties` can LIE about the real target:** the true target = intersection(the `niagara_home` that has the pinned plugin, the station that accepts the stamped `baja`). Check `settings.gradle.kts` and `ls <niagara_home>/etc/m2/repository/com/tridium/niagara-module/...` before trusting `gradle.properties`. [ev: retro dashboardpan-5rooms #6]
- **A project you did not generate (course zip, vendor sample) is retargeted BEFORE its first build — run `toolbelt/preflight.sh --sample <niagara_home> <gradle-root>`:** it FAILs when `gradle.properties` `niagara_home` is absent on disk (a `c:\\…` value is checked as its `/mnt/c/…` mount) or names another version than the target install (`sample-home`), when `niagara_user_home` still has a `<username>`-style placeholder (`sample-user`), and when the ROOT `build.gradle.kts` lacks the signing block — `signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }` + `niagaraSigning { aliases…; signingProfileFile… }` (`sample-sign`); it WARNs on a shipped `build/` directory (`sample-build`: delete it). Opt-in, because a client tree's own `gradle.properties` legitimately names its Windows install. `[ev: retro n4-course-signed-jars-gate-fix Δ6]` `sample-sign` reads only the code outside comments and Kotlin strings (`//` and `/* */` stripped, a marker inside `"**/*.jar"` or `"https://…"` is text) and takes `allowDefaultProfile.set(false)` or `allowDefaultProfile = false`: `allowDefaultProfile.set(true)` / `= true`, a commented-out block or a block missing `aliases`/`signingProfileFile` FAILs and names what is missing — a substring match PASSed all three. `[ev: retro n4-course-sample-check-precision Δ1]` No row PASSes by omission: a missing `niagara_user_home` WARNs `sample-user`, and a `niagara_home` whose path or target has no X.Y.Z WARNs `sample-home` ("version could not be compared") instead of skipping the compare. `[ev: retro n4-course-sample-check-precision Δ2]` `[ev: retro n4-course-handson-build-signing-windows Δ6]`
- **A Windows `niagara_home` path breaks the m2 repo under WSL:** gradle resolves `maven(file:/C:/...)` → "plugin not found"; always use the `/mnt/c/...` mount. [ev: retro dashboardpan-5rooms #7]
- **A nested / multi-project gradle layout breaks the `<root>/<MOD>-<profile>` assumption — run `./gradlew :<MOD>-<profile>:jar` from the GRADLE ROOT, then verify `build/libs`:** e.g. DashboardPan's `gradlew` is at `Dashboard/` while the profiles are `Dashboard/DashboardPan/DashboardPan-{rt,ux,wb}` addressed as `:DashboardPan-ux:jar`. [ev: retro dashboardpan-ux-direct-build · B7]
- **Keep `src/rc/` asset-only — `processResources` copies ALL of `rc/` into the jar, so `index.html.bak` / `.orig` / scratch files ship as dead weight (~MB) AND are servable at `/<prefix>/index.html.bak`:** label-editing is preview-only, never committed; an orphan `img/plano.png` also ships when the real image is base64. [ev: retro dashboardpan-ux-direct-build · B4]

| `niagara_home` | niagara-module plugin (only one per install) |
|---|---|
| 4.13.2 | 7.3.40 |
| 4.14 | 7.6.17 |
| 4.15.3 | 7.6.22 |

There is no version common to all installs — this table supersedes the earlier "shared common set" claim. Always pass `-PniagaraPluginVersion` matching the chosen target.

## Module versioning & release
- **The MODULE's own version lives in `defaultModuleVersion("X.Y.Z")` in each module's `build.gradle.kts` `vendor{}` block → stamped as `vendorVersion` at build; a bump REQUIRES a rebuild to reach the jar.** Per-module version skew is fine (rebuild only the changed modules); the repo tracks `build/libs/*.jar`; tag `vX.Y.Z`. [ev: retro soft-start-staggered-startup · S4]
- **`BVersion.MEETS_MINIMUM` mask = 115** (bits: `LATER|SAME|EQUIVALENT|MORE_SPECIFIC|LESS_SPECIFIC`; excludes `EARLIER=4` and `DIFFERENT=8`) — this is why `<dependency vendorVersion="4.14">` (a minimum floor) is satisfied by 4.15 but NOT by 4.13 and NOT by a different vendor, and why the `--target-version` gate FAILS on a jar stamped above the target. (BVersion.java:58–66.) `[ev: corpus B755 §755.4]`

## niagara_home on WSL
A station or NRE command (`station <name>`, `nre`) needs `niagara_home` and the PATH set. On Windows run it from the
Start-menu **Niagara Console** (`console.exe` sets both); for a non-standard user home `set niagara_user_home=<dir>`
there first (`nre.dll` reads it into `-Dniagara.user.home`). On Linux source `<sysHome>/bin/.niagara` first.
`[ev: corpus B1420 §1420.3]` `[ev: corpus B1416 §1416.5 c]` `[ev: retro course-coverage Δ17]`

Building on WSL against `niagara_home=C:\...` breaks the m2 repo (see §Build target & plugin version); and when a `-ux` depends on `-rt`, the plugin's auto-copy of the built jar into a station-locked `modules/` fails. That copy failure is irrelevant — free the lock, or use the already-assembled `build/libs` jar (§Building against a running station: free the lock first, mirror only when you can't); the mirror is only for a live production supervisor.

## Build location — ext4 vs 9p/drvfs `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]`
- **Work under an ext4 path (e.g. `~/modulos_niagara_n4/Cliente/...`), never directly under `/mnt/<drive>` (a WSL 9p/drvfs mount), for both the source tree AND `niagara_home`.** A live field session ran the source tree at `/mnt/c/Users/.../Downloads` and `niagara_home=/mnt/c/Honeywell/...` (1013 module jars, 9p) — a measured file walk took 0.200s per file there vs 0.020s on ext4 (10x). Gradle's per-file stat/hash work multiplies that gap, turning a normal 3-5s build into minutes.
- `toolbelt/build.sh` WARNs (non-fatal — the build still runs) when either the gradle root or `niagara_home` resolves to a `9p`/`drvfs` mount. Fix by cloning the source tree to ext4 and pointing `niagara_home` at an ext4 mirror via `toolbelt/mirror-niagara-home.sh` (§Mirror above) instead of the raw `/mnt/c/...` install.
- The detection is filesystem-type based (`df -T`), not a path-string check — a symlink or bind-mount from ext4 back onto `/mnt/c` would still read as `9p`/`drvfs` and correctly WARN.

## Building against a running station: free the lock first, mirror only when you can't
- **The kit never writes a real `niagara_home` by default:** the plugin's `:jar` step copies the built jar into `<niagara_home>/modules`, and the drift gate's backup is a temp dir (a mirror keeps the replaced jar, below; a real install does not), so a build pointed at a real install replaces its jars with no copy kept. `toolbelt/build.sh` therefore exits **10** before gradle when `niagara_home` lacks the `.niagara-mirror` marker that `mirror-niagara-home.sh` writes; building into a real install takes the explicit `--install-into-home`. `[ev: retro r10-buildsh-mirror-guard Δ1]` `[ev: retro r10-buildsh-mirror-guard Δ2]`
- **Free the lock FIRST — it's the common case:** an `Unable to delete …/modules/<mod>.jar` at `:clean` is a Workbench / your-own-station lock; close Workbench (or, when the holder is NOT a live production supervisor, stop it) and build directly against the install (`build.sh --install-into-home`). This recurred 3× in one session and closing Workbench cleared it each time. Use the MIRROR (below) ONLY when the holder is a running station you must not stop, or a box you do not own — the mirror is the documented fallback, not the default. [ev: retro coldroompan-dashboardpan-freeze-stat-leds · B2]
- **A build against a RUNNING station reports FAILED at the `modules/` copy, but `build/libs/<jar>` is already fully assembled + signed:** the plugin auto-installs the jar as the last `jar` step; a locked `modules/` makes gradle exit FAILED, yet the artifact in `build/libs` passed the gate — verify THAT jar and use it. The copy failure is irrelevant when you are not deploying to that supervisor. [ev: retro dashboardpan-ux-direct-build · B1]
- **A brand-new module (not yet in `modules/`) needs NO station stop:** a first build has no deployed jar to overwrite, so `:clean :jar` builds green against a live install with the station running — only an UPDATE hits the lock. [ev: retro comppan-fase1-staging · B3]

### Mirror (the fallback for a live production supervisor)
- **A running station LOCKS `<niagara_home>/modules/<mod>.jar` — never stop a live supervisor to build:** with `station`/`niagarad` running, the plugin's `clean`/`jar` fails deleting the deployed jar (`IOException: Unable to delete file …/modules/<mod>.jar`). Build against a read-only MIRROR: symlink every top-level entry + every `modules/*.jar` EXCEPT the one being built into a dir with a WRITABLE `modules/`, and build with `-Pniagara_home=<mirror>`. The jar lands in `build/libs` (deliverable) + `<mirror>/modules` (throwaway); the real install is never written. [ev: retro coldroompan-rt-hardening #8]
- Use `toolbelt/mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]`. Two guards protect you: it refuses when the mirror equals, sits inside, or contains the source (exit 20), and refuses to wipe an existing dir that lacks a `.niagara-mirror` marker (exit 20); it writes `.niagara-mirror` on creation.
- **A mirror build isolates `user.home` — the signing plugin must not read your real keystore:** the niagara-signing plugin's default profile is `user.home/.tridium/security/niagara.signing.xml` (SigningProfileFactory, niagara-plugins 7.6.17). `org.gradle.jvmargs=-Duser.home=…` does NOT reach an already-running Gradle daemon, so a mirror build read the real `~/.tridium/security/niagara.signing.{xml,jceks}`. When `niagara_home` holds `.niagara-mirror`, `toolbelt/build.sh` appends `-Duser.home=<mirror>/.mirror-user-home` to `JAVA_TOOL_OPTIONS` (any existing value kept), keeps `GRADLE_USER_HOME` (default `~/.gradle`, so offline dependency resolution still works), appends `-Dmaven.repo.local=<real ~/.m2/repository>` so `mavenLocal()` does not move with `user.home` (unless `JAVA_TOOL_OPTIONS` already sets `maven.repo.local`) and passes `--no-daemon`. The JVM splits `JAVA_TOOL_OPTIONS` on whitespace (JDK 8 has no quoting), so a mirror path with whitespace is refused with exit 33 before gradle runs, and so is a `$HOME` with whitespace unless `JAVA_TOOL_OPTIONS` already sets `-Dmaven.repo.local=` (the appended `$HOME/.m2/repository` would be split the same way; outside mirror mode nothing is appended and such a `$HOME` builds). A raw `./gradlew` mirror build must do the same by hand. Such a build therefore cannot pick up your real DEFAULT profile. An explicit profile (`niagaraSigningProfile`, passed as `--gradle-prop niagaraSigningProfile=<path>` or `ORG_GRADLE_PROJECT_niagaraSigningProfile`, #308) still signs. A root with the signing block and no profile fails `:jar`. A root WITHOUT the block makes the plugin create `.mirror-user-home/.tridium/security/niagara.signing.{xml,jceks}` and sign with a throwaway Niagara4Modules dev key (observed with a redirected `user.home`); every profile under `.mirror-user-home` is plugin-created (never yours) and persists until the mirror is refreshed, so `build.sh` WARNs on stderr after every successful mirror build while that `niagara.signing.xml` exists, not only the build that created it, unless the root's uncommented code disables the default profile (`allowDefaultProfile.set(false)` or `= false`, the lint-structure L15 pattern from `go/internal/signblock`): such a root cannot fall back to that profile, so a passing build signed with the explicit one (exit stays 0 — do not deploy that jar; add the signing block, `templates/signing-root.gradle.kts`, lint-structure L15). So read the `signed` row of `verify-module.sh` before deploying. A mirror refresh wipes `.mirror-user-home` with the rest of the mirror. `[ev: CERT-live local build 2026-10-22]` `[ev: retro r8-mirror-userhome Δ1]` `[ev: retro r8-buildsh-316-308 Δ1]` `[ev: retro r8-buildsh-316-308 Δ2]`
- **A mirror reused between builds is not a deploy baseline:** `:jar` copies each build's jar into the mirror's writable `modules/`, so the next build's drift gate compares against the previous LOCAL build and fails with a false exit 51 (shipped bytes changed, `vendorVersion` unchanged). Recreate the mirror for every gate run (symlinks only, cheap), or pass `--no-drift-check` when the mirror is not the deploy target. `[ev: retro r10-client-gate-retro Δ2]`
- **The mirror keeps the jar a build replaced — put it back before the next gate run against the deployed baseline:** when a `build.sh` run ends (any exit) with a mirror `modules/<MOD>-<p>.jar` that differs from the drift gate's pre-build snapshot, the snapshot is copied to `<mirror>/.mirror-replaced/<MOD>-<p>.jar.<sha12>` (first 12 hex digits of its sha256; a kept jar is never overwritten or deleted, so a module built twice keeps both) and stdout ends with `==> mirror: <MOD>-<p>.jar in <mirror>/modules replaced (vendorVersion <old> sha256:<old> -> <new> sha256:<new>); first replaced this cycle: vendorVersion <v> sha256:<s>; restore with: <toolbelt>/build.sh --restore-mirror <kept>`. The first jar a cycle replaced is the one the station runs, so the restore command names it even after a second build (`<MOD>-<p>.jar.first` holds its hash). A cycle is the builds since the last `--restore-mirror`, and nothing else ends it: **after deploying a build to the station, delete `<mirror>/.mirror-replaced/<MOD>-<p>.jar.first`** (the mirror then holds what the station runs), or the next build still names the pre-deploy jar. A pointer that cannot be written is one `WARN — the first <MOD>-<p>.jar replaced this cycle is not recorded` line, and the stdout line then names no first jar (list `.mirror-replaced/` and restore the one the station runs). Printed paths are shell-quoted. `--restore-mirror <kept>` builds nothing: it copies the kept jar back into that mirror's `modules/` (the kept copy stays) and refuses a path not under `.mirror-replaced/` (exit 2), a missing kept jar, one whose sha256 prefix is not the `<sha12>` of its name, or a home without `.niagara-mirror` (exit 10, nothing written). A jar that cannot be kept (an unwritable `.mirror-replaced/`) is one `build.sh: WARN — the replaced <MOD>-<p>.jar was NOT kept in …` line; the exit code does not change. Dependents still compile against the new jar during the build; the restore is the operator's call after it. Nothing is kept for a real `niagara_home`, under `--no-drift-check`, on a first deploy, or when the run already restored `modules/`. `[ev: code go/internal/buildmodule/replaced.go]` `[ev: retro buildsh-mirror-baseline-b6 Δ1] (retro row B6, a client A module build 2026-11-16)`

## Verify
```
# THE gate — run on the built jars regardless of who built them:
toolbelt/verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] [--expected-signer ALIAS] <jar>...
# rows: PASS|FAIL|SKIP  <check>  <jar>  <detail> ; exit 0 pass · 1 fail · 2 usage · 3 env.
```
Default checks (conservative): every `.class` at bytecode major **52** (`META-INF/versions/**` multi-release classes skipped and counted — Java 8 never loads them; a jar with ONLY such classes FAILs `no .class entries outside META-INF/versions/` and still reports the count `[ev: retro n4-course-sample-check-precision Δ3]` `[ev: retro n4-course-handson-build-signing-windows Δ3]`), a `META-INF/*.SF` signature block (`<ALIAS>.SF` + `<ALIAS>.RSA|.DSA|.EC`; the row names the alias), every `module.xml` `<type>` resolves to a `.class` (presence is tested with here-strings, never `printf … | grep -q` under pipefail: on an entry list over 64 KiB that pipe returned 141 and reported present classes as missing — `tests/pipefail-grep.bats` gates it, #267 `[ev: retro n4-course-signed-jars-gate-fix Δ2]` `[ev: retro n4-course-handson-build-signing-windows Δ2]`). Opt-in: `--target-version` (baja stamp), `--stored` (0 `Defl:` entries), `--src` (type-count vs `module-include.xml` + no raw-double facet).

What each check does by hand (the gate automates these — it checks EVERY class, not just the first):
```
# bytecode major 52 = Java 8 (65 = Java 21, wrong)
unzip -p MOD-rt/build/libs/MOD-rt.jar com/.../SomeClass.class | od -An -t d1 -j6 -N2   # 2nd number = 52
# signed
unzip -l .../MOD-ux.jar | grep -E 'META-INF/[^/]+\.(SF|RSA|DSA|EC)$'   # <ALIAS>.SF + its block
# no raw-double facet anywhere (--src)
grep -rnE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' <module> --include='*.java'    # must be empty
# baja stamp (--target-version)
unzip -p <jar> META-INF/module.xml | grep baja
```

`--src` also runs `facets-req` (WARN): an OPERATOR numeric slot without a `facets` key, a setpoint-like slot without a unit, a count-like slot without a precision. `BFacets.makeNumeric(<unit>, <precision>, ...)` satisfies the unit test (it carries a real unit without the literal `UNITS` token) and any `makeNumeric(` satisfies the precision test; a precision-only `makeNumeric(<int>)` still WARNs missing UNITS. Do not hand-rewrite a working `makeNumeric` facet to silence the check. `[ev: retro comppan-fase2-amps-alarms Δ1]`

`--src` on a driver module (a class extending a network or device base) adds the `driver-*` rows: `driver-scaffold`,
`driver-comm-config`, `driver-discovery-alive`, `driver-server-enabled`, `driver-manager`; the rules and their doctrine
sections are in `types/driver-authoring.md` § 12. A non-driver module prints none of them. `[ev: retro r10-dr4-verify-driver Δ1]`

Reviewed false positives in the advisory source lints: `lint-arbitrary-ord.sh` WARNs on every `BOrd.make(<variable>)`; when the value is built from a code-controlled internal string (never client input), put `// lint-arbitrary-ord: reviewed <reason>` on the same line or the line directly above the call. The reason is mandatory — a bare marker still WARNs ("reviewed marker without a reason"). The marker records a review; it is not an allowlist for client-supplied ORDs (types/security.md §4). `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ4]`

## Pre-release real-jar smoke test
Before shipping a kit or toolbelt change, run `toolbelt/verify-module.sh` over at least one known-good and one known-bad real module jar — the bats suites use generated fixtures, which only simulate a malformed jar; a real one proves the gate catches a live defect. Worked known-bad: `ColdRoomPan-rt.jar` fails the `types` check because `module-include.xml` still declares `com.angeles.ColdRoomPan.BHoaMode` after that class was deleted — the live "Missing class ColdRoomPan:HoaMode" defect. DashboardPan's jars pass `--src` clean. Usage: see §Verify above.

- **Smoke every new parser on ≥1 real module before merge:** bats fixtures are synthetic and may miss edge cases (field ordering, encoding, missing files) that a real module tree exposes; use at least one known-good and one known-bad real module as the final gate. [ev: retro campaign7-schema-risk]

## Schema-risk deploy workflow `[ev: retro tree-selection-and-schema-risk-baseline Δ2]`

The `schema-risk.sh` "before" snapshot must be the **DEPLOYED source**, not the current work tree:
1. Extract the deployed baseline from the station backup or the original RAR/ZIP: the result must be a layout with `module-include.xml` + `*.java` source files (the same layout `schema-risk.sh` expects). When the deployed source is a git ref (a tag or commit), `T=$(mktemp -d) && git -C <artifact> archive -o "$T/b.tar" <deployed-ref> . && mkdir <artifact>/.deploy-baseline && tar -x -f "$T/b.tar" -C <artifact>/.deploy-baseline; rm -rf "$T"` writes it where `report-module.sh` (and so `build.sh`) looks (the directory is made only after the archive succeeded: a wrong ref, or `<deployed-ref>` pasted as printed, leaves no empty baseline); the `schema-risk` SKIP row of an artifact without it prints that command with the artifact path filled in. `[ev: code go/internal/reportmodule/special_floor.go]`
   `build.sh --schema-baseline-ref <deployed-ref>` does that extraction for the run, so the floor check runs without the manual step: before gradle, each artifact under `<module-root>/<MOD>` is extracted at the ref into a temporary directory and `<artifact>/.deploy-baseline` is a link to it; the link and the extraction are removed when the run ends, whatever the exit (a link a killed run leaves dangles, which reads as no baseline, never as a stale one; the next run with the option removes such a link with an INFO line — only a link to a missing `<mktemp dir>/baseline` this option made, and only when no artifact is refused: an operator link, even a dangling one, is refused and kept). Each extracted artifact prints `==> schema-risk baseline: <artifact> from <ref>`; an artifact that does not exist at the ref prints an INFO line and keeps its SKIP row. It refuses with exit 2, before gradle: a ref that is not a commit of the module's repository, a `<MOD>` directory absent at the ref, an artifact that already has a `.deploy-baseline` (an operator-made baseline is never moved or overwritten: remove it to take the ref, or drop the option to use it), and `--first-deploy` (no deployed ref). `[ev: retro mcpn4-history-batch-tools Δ4]`
2. Run `toolbelt/schema-risk.sh <before-dir> <after-dir>` where `<after-dir>` is the current build tree.
3. **A LOSSY verdict (`remove_slot_*`) must be cross-checked against the live bog before treating it as a blocker:** a slot that was removed but never saved in the `.bog` (0 persisted instances) is non-dangerous. Use `toolbelt/bog-audit.sh <config.bog> --module <MOD>` or grep the raw `config.bog` XML for the removed slot name to confirm no live instance persists it. A LOSSY + live instances = blocker; a LOSSY + 0 instances = safe for this station (but note it for other stations).
4. An OUTAGE verdict is always a blocker regardless of bog state — a retype crash has no safe-persistence case.

**Why the work-tree snapshot is wrong:** the current work tree is the BUILD baseline, not the DEPLOYED baseline. Version drift between the work tree and the live station (e.g. Dashboard 2.0 live vs 2.2.0 in the tree) makes the schema-risk diff report the wrong deltas — it sees code changes that were already deployed as new schema changes.

## Ready to load: `report-module.sh --ready` `[ev: retro ready-to-load-p4 Δ1]`

Before an operator loads a built client module on a production station, one command gathers what used to be checked by hand:

`toolbelt/report-module.sh <module-root> --ready --niagara-home <mirror> --expect-niagara <ver> --expect-signer <alias> [--target-version x.y] [--first-deploy] [--json]`

One row per check for each built jar (`<artifact>/build/libs`), `<artifact>  PASS|WARN|FAIL|SKIP|ERROR  <check>  <evidence>`; an artifact of the module (a directory with `module-include.xml`, its own `<artifact>.gradle.kts` or a `build.gradle[.kts]`) that has no built jar gets one `FAIL  jar  no built jar under build/libs: this artifact of the module would not be loaded (build it first)` row, while a `docs/` or `gradle/` directory beside the artifacts gets none:

| Check | Evidence | FAIL when |
|---|---|---|
| `module` | the module part and vendorVersion of `META-INF/module.xml` | it cannot be read |
| `niagara-target` | the `baja` dependency floor | the floor is above `--expect-niagara`, or cannot be compared with it (`4.14-rc` against `4.15`: the target named was not checked, so it never reads READY) (WARN without the flag) |
| `niagara-home` | `<niagara-home>/modules/baja.jar` vendorVersion: the Niagara it was compiled against | its release line is newer than the target (an older line WARNs; SKIP without `--niagara-home`) |
| `signer` | the complete `META-INF/<SIGNER>.SF` signatures | unsigned, the development signer `NIAGARA4`, or not `--expect-signer` (WARN without the flag) |
| `verify-module` | the gate run with `--expected-signer` and `--target-version` when given | the gate FAILs (its first FAIL line is the evidence) |
| `schema-risk` | the verdict against `<artifact>/.deploy-baseline` | OUTAGE, or no baseline (the row prints the command that creates it from the deployed ref, § Schema-risk deploy workflow); `--first-deploy` makes it a SKIP |
| `sha256` | the jar's full sha256 | the jar cannot be read (ERROR) |
| `mirror` | whether `<mirror>/modules/<jar>` is the deployed jar again | never; a WARN prints the `build.sh --restore-mirror <kept jar>` line (§ Mirror) |

Then `READY TO LOAD: <MOD> <version> for Niagara <x> (sha256 <12>)` when no row FAILs or ERRORs (`<x>` is `--expect-niagara`, else the compile home's version), followed by the post-load read-back: after loading, `n4_station_runtime` (mcp-n4) must list the part at that version in `loaded_modules` with no `mismatches`. Otherwise `NOT READY` with the failing rows. The tool never contacts a station. Exit 0 ready · 1 not ready · 2 usage · 3 env (an ERROR row, a module root that cannot be listed — `report-module: --ready: cannot list <root>: <error>` — or no built jar in any artifact). `--json` prints the same as one object (`root`, `verdict`, `jars[]` with `rows`, `line`, `readback`). No lint member runs and `.report-baseline` is not read; every lint-run option (`--lint-only`, `--write-baseline`, `--no-baseline`, `--console-dir`, `--profile`, `--legacy`, `--wiring-map`) is refused with `--ready` (exit 2, a first line naming it), and `--json`/`--expect-*` without `--ready` too.

## Signing per deploy target
- **Check the deploy target's signing policy before assuming a Workbench re-sign:** a Honeywell supervisor ACCEPTS gradle's per-machine DEV cert — no re-sign needed (chihuahua's `deploy.sh` only builds + copies and runs on the same supervisor). A JACE field controller enforces the project CA (e.g. `angelessigner`), so a JACE-bound module IS re-signed. [CERT-live 2026-09-01 · retro 5rooms #9]
- **Run the signing doctor on the exported material BEFORE the build:** `toolbelt/signing-doctor.sh --cert <certificate> [--key <private key export>]`
  (`go/internal/signingdoctor`, one row each: `cert-parse`, `cert-ca`, `cert-usage`, `cert-validity`, `key-format`; exit 0 pass /
  1 a FAIL row / 2 usage). A certificate that is a CA (`CA:TRUE`: the export took the CA that issued the signing key, not the
  signing certificate), has an extended key usage without code signing, or is expired is a FAIL, so a signing round is not
  spent finding it; a certificate with no usage extension at all is a WARN. A key export that is PBES2 with AES-GCM (what the
  Workbench private key export writes) is a FAIL: OpenSSL 3 cannot read it, convert it with Java, never with OpenSSL.
  Pinned by `go/internal/signingdoctor/command_test.go`. `[ev: retro mcpn4-history-batch-tools Δ5]`
  `[ev: code go/internal/signingdoctor/checks.go]`
- **A jar signed with the development key passes the default `signed` check — name the signer the module must carry:**
  `verify-module.sh --expected-signer ALIAS` FAILs the `signed` row unless the jar's complete signatures (an `.SF` with
  its block file) are exactly that one signer: another alias, a second signer next to the expected one, or no signature.
  The alias is compared in the form jarsigner names the files after — upper-cased, cut to 8 characters, a character
  outside `A-Z 0-9 _ -` as `_` (`acmesigner` → `META-INF/ACMESIGN.SF`; observed with the `jarsigner` of JDK 8 and JDK 21) —
  and the FAIL text prints both the alias as given and that file name; a signature file named exactly as the alias
  was given matches too. An empty alias is a usage error (exit 2), never "no comparison". Without the option every row
  is what it was. `build.sh` passes it by itself: the configured `niagaraSigningAlias` gradle property (the last
  `--gradle-prop`, else `ORG_GRADLE_PROJECT_niagaraSigningAlias`, else `GRADLE_USER_HOME/gradle.properties`, else the
  gradle root's `gradle.properties`), else the one signer of the deployed jars its drift gate snapshots (none under
  `--no-drift-check`, none when they disagree, and never the signing plugin's default development signer `NIAGARA4`:
  a deployed jar that carries it gives no expected signer and the WARN names it), else nothing — and then it prints one `build.sh: WARN — no
  niagaraSigningAlias is configured …` line after the gate's rows; the exit code does not change. An alias hardcoded
  in a root `build.gradle.kts` is not read; a configured alias is passed as given, whatever its value. A gate FAIL
  restores the pre-build `modules/` jar the drift gate backed up (one `Restored the previously-deployed` line, exit 50
  as before), so a bare re-run still expects the deployed signer and FAILs again instead of taking the failed jar as
  its baseline; under `--no-drift-check` there is no backup, nothing is restored and the deployed-jar signer is
  unavailable. On a first deploy (no jar of the profile in `modules/` before the build) there is nothing to restore,
  so a gate FAIL removes the jar gradle just installed (one `Removed the newly-installed` line, same exit code) and a
  gate PASS leaves it; under `--no-drift-check` nothing is removed. A SIGINT, SIGTERM or SIGHUP from the baseline
  snapshot until the chain passed does the same before the temporary directory goes (one `build.sh: interrupted
  (<signal>) before the gate passed; putting modules/ back as it was before the build` line, then the `Restored` /
  `Removed` lines, exit 128 + the signal); under `--no-drift-check` one line says `modules/` is left as the build left
  it. Before the restore the signal is passed on to the gradle the run started, which is waited for (killed after 10 s),
  and no gradle starts after it; a gradle daemon (builds outside a mirror home, which use `--no-daemon`) cancels on its
  own, so interrupting the process group stays the safe way
  `[ev: code go/internal/buildmodule/signal.go]` `[ev: retro go-build-module-port Δ6]` `[ev: retro p1-build-deploy-safety Δ1]`. What gradle itself started (a single-use daemon forked by the wrapper's client) is read from `/proc` before the signal is passed on and ended with a SIGKILL once gradle has ended, so a job it left running cannot install its jar after the restore (`go/internal/buildmodule/procs.go`, `TestAGrandchildOfGradleIsStoppedBeforeTheRestore`); gradle is not moved into a process group of its own, which would take it out of the terminal's foreground group (a rich console writing terminal modes from a background group is stopped by SIGTTOU) `[ev: retro p1-build-deploy-safety Δ10]`. The development signer is recognised in the
  compared form, so a deployed `niagara4.SF` is never expected either. The check compares the signature FILE NAME,
  not the certificate: two aliases that share their first 8 characters name the same file, and whether the key is
  trusted stays a station-side question. Signature files are read in any case of the extension (`.sf`, `.rsa`, `.dsa`,
  `.ec`), as the JDK verifier reads them, with or without the option: a jar whose only signature is `NAME.sf` +
  `NAME.rsa` is signed (it was "no META-INF/*.SF" before), and a second signer filed in lower case is reported. The first build of a client module therefore needs the alias configured:
  with no deployed jar to compare with, only the WARN says the jar may carry the development signer, which a
  station's Software Manager shows as "Signature Error / Self-signed". (source: delta B5 of an external session retro, 2026-11-13)
- **The signature entries are named after the signing ALIAS, not fixed:** `NIAGARA4.*` for the default profile, `CODESIGN.*` for the N4 course `CodeSigning` alias, `SERVER1.*` for Honeywell-signed jars. The station decides by `niagara.moduleVerificationMode` + cert chain, never by the entry name, and `verify-module.sh`'s `signed` check accepts any `META-INF/<ALIAS>.SF` with its `<ALIAS>.RSA|.DSA|.EC` block (an orphan `.SF` FAILs) and reports the alias. A wizard-generated ROOT `build.gradle.kts` applies `com.tridium.niagara-signing` with NO `signingServices`/`niagaraSigning` block (`devkit-wb.jar!gradle/build.gradle.kts.vm:18`, 4.15.6.20), so a missing profile signs with the self-signed default key and the build stays green: `lint-structure.sh` L15 WARNs such a root (same uncommented-code rule as `preflight.sh --sample` `sample-sign`, which FAILs a sample tree), `templates/signing-root.gradle.kts` holds the block (alias and profile path from the `niagaraSigningAlias` / `niagaraSigningProfile` gradle properties, never a tracked host path), and `scaffold-module.sh` emits it through both fixtures. `[ev: retro n4-course-signed-jars-gate-fix Δ4]` `[ev: retro n4-course-handson-build-signing-windows Δ4]` `[ev: corpus B817 §817.6]` `[ev: retro n4-course-signed-jars-gate-fix Δ1]`
- **The property-driven signing block builds offline and fails closed; a real signed build is still open.** The
  template is a property-driven variant of the course block (Lab 5 `signing_configuration.kts` uses literals). Built
  offline against an N4 4.14.0.162 mirror (Honeywell OptimizerSupervisor), niagara-module plugin 7.6.17, Gradle 7.6, on
  a `scaffold-module.sh` module via `build.sh`: with no properties, configure passes and `:<MOD>-rt:jar` FAILS
  ("property 'signingAwareArchive.aliases' doesn't have a configured value", exit 30); `gradlew --offline help` with
  dummy `-PniagaraSigningAlias`/`-PniagaraSigningProfile` is BUILD SUCCESSFUL (the Provider-API form configures); with
  the properties pointing at a missing profile file, `:jar` FAILS ("specifies file ... which doesn't exist", exit 30).
  Pass the two properties through `build.sh` as `--gradle-prop niagaraSigningAlias=<alias> --gradle-prop
  niagaraSigningProfile=<path>` (repeatable, forwarded as `-P<k>=<v>`; a missing `k=v` or empty key exits 2), or as
  `ORG_GRADLE_PROJECT_niagaraSigningAlias` / `ORG_GRADLE_PROJECT_niagaraSigningProfile` environment variables (kit
  issue #308). `[CERT-live local build]` `[ev: retro r8-buildsh-316-308 Δ3]`
  With a throwaway self-signed JCEKS key and a hand-written profile (Java properties XML, devguide
  `security/codeSigning.txt:29-47`; keys `niagara.signing.profileType`, `storetype`, `storepath`, `storepass`,
  `keypass.<alias>`), the build SIGNS: `META-INF/<ALIAS8>.SF`/`.RSA` (alias cut to 8 upper-case chars), `jarsigner
  -verify` passes; no CA chain or timestamp server is needed (the TSA is contacted only when
  `niagara.signing.standardtsa` is set, devguide `codeSigning.txt:113-135`). `allowDefaultProfile.set(false)` is what
  refuses on its own: with no profile, `:jar` FAILS "The default profile is not available. Specify one explicitly";
  with the line removed, the plugin creates `user.home/.tridium/security/niagara.signing.{xml,jceks}` and the build
  passes signed `NIAGARA4` by a "For Development Purposes Only" Niagara4Modules certificate. `[CERT-live local build]`
  A production CA-issued profile was not exercised (none on that machine).
- **A hand-written LocalSigningProfile needs validity - refresh >= 32 days, and lint-structure L16 FAILs one that
  has less.** The signing plugin sets a new key's notAfter to day `validity` of the current month instead of adding
  `validity` days, and re-mints the key when fewer than `refresh` days are left (niagara-plugins 7.6.22
  `LocalSigningProfile.kt:386-388` and `:216-225`), so a 30/15 profile re-mints at every signing and its jars carry
  a signature block no JVM verifies. `lint-structure.sh` L16 reads the `niagaraSigningProfile` the gradle root's
  `gradle.properties` names and FAILs such a profile with both values; `run-niagara-test run` refuses the same
  profile from the same predicate (`go/internal/signingprofile`). Use the plugin defaults, 365/183. Re-check the
  plugin on an upgrade (the vendor-defect section of the retro). `[ev: retro r10-signing-margin-lint Δ1]`
- **Six permission groups require a signed, trusted module:** `REFLECTION`, `ACCESS_CLASS`, `MBEAN_PERMISSION`,
  `THIRD_PARTY_PERMISSION`, `PROTECTION_DOMAIN`, `HSM_SIGNING` (`requiresSignature()` is true in exactly these six
  `com/tridium/security/*PermissionGroup` classes, vineflower 4.14.0.162; the devguide names only REFLECTION and
  HSM_SIGNING, `security/codeSigning.txt:185`). Typical triggers: Jackson databind or any `setAccessible` →
  REFLECTION; sockets/HTTP → `NETWORK_COMMUNICATION` (no signature needed; scope `hosts`/`ports`); `shutdownNow` /
  `interrupt` → `MANAGE_EXECUTION`. Trusted = a cert from a public CA, a cert from an internal CA whose root is in
  the user trust store, or a self-signed cert in the user trust store (refused in `high` mode;
  `codeSigning.txt:137-143`). `[ev: corpus B1312 §1312.3 C1-C3]` `[ev: corpus B1235 §1235.5]`
  `[ev: retro devguide415-horizons-authoring-security Δ4]` `[ev: retro n4-course-build-deps-permissions-signing Δ2]`
- **What an unsigned module gets is the group silently dropped, not a halt — at the permission-factory layer.** The
  devguide says the station or Workbench halts (`security/codeSigning.txt:185`, same sentence in the 4.13.2.18 and
  4.15.6.20 devguides) and `requestingPermissions.txt:160,294` only says the group "requires the module … to be
  signed". The code disagrees, and code wins for behaviour: `NiagaraPermissionGroupFactory` throws "Permission group
  (%s) requires signature but the module (%s) is not signed" inside its per-element `try`, whose `catch (Throwable)`
  logs `Error parsing Niagara permission group XML … Skipping element.` as a WARNING and goes on. Verified in
  `baja` 4.14.0.162 (organized/ vineflower `NiagaraPermissionGroupFactory.java:196-204`, decompiled `:243-248`, procyon
  `:215-220`) and by javap of `modules/baja.jar` 4.15.6.20 and of iC-Niagara 4.13.2.18 `baja.jar` (both: `athrow` at
  offset 537 inside the Throwable range 134-547, handler at 550). Other 4.13-4.15 builds: not verified. The module then
  loads without that grant, so its first privileged call should fail with `AccessControlException` `[INFER]`; whether
  the cert-chain layer also refuses the load is not live-proven (niagara-research B1312-G1). Either way: sign it with a
  trusted cert, and read the station log for `Skipping element`. `[ev: corpus B1312 §1312.3 C2]`
- **JACE=MEDIUM vs Win Supervisor=LOW:** the Win Supervisor's `defaults/system.properties` actively sets `niagara.moduleVerificationMode=low`; the JACE-8000 QNX leaves the property commented out, falling back to the N4 code default `medium`. A dev-signed module that loads on the supervisor (LOW) fails on a JACE (MEDIUM) unless the cert is imported into the JACE user trust store. The "JACE enforces the project CA" note above is correct; MEDIUM mode is the reason. Both files add the comment "will be removed in a future release" — treat `medium` as the stable long-term target. [ev: corpus B1004 §1004.2-1004.3]
- **Read the mode, do not assume it.** The 4.15.6.20 guide still says "the current default verification mode is
  Low" (`ModuleSign/VerificationModesModuleSigning-182D0E86.txt:17` "The current default verification mode is Low, but") while its own roll-out page says Medium from 4.9
  (`ModuleSign/StagedRoll-outModuleSigning-182E7CC0.txt:11` "Niagara 4.9 — Default verification mode is Medium."). Code wins for behaviour: with no property, `Nre` asks
  `ModuleVerificationMode.make`, which returns `medium` for baja ≥ 4.9 (vineflower baja 4.14.0.162 `Nre.java:752-763`;
  decompiled nre-ext 4.14 `ModuleVerificationMode.java:11,28-33`; 4.13 and 4.15 not verified). The authority on a
  given host is `niagara.moduleVerificationMode` in its `system.properties`. `[ev: corpus B1010]`
  `[ev: retro devguide415-r2-packaging-platform-signing Δ6]`
- **Mode × signature status** (4.15.6.20 `ModuleSign/SignatureStatusesModuleSigning-1F0024B2.txt:9-21` "Ok – The module is signed with a",
  `VerificationModesModuleSigning-182D0E86.txt:11-15`):

  | Status | low | medium | high |
  |---|---|---|---|
  | Ok (trusted CA, timestamped) | yes | yes | yes |
  | Not timestamped | yes, until the signing cert expires | same | same |
  | Self-signed, cert in the user trust store | yes | yes | no |
  | Certificate path validation failure | warns | no | no |
  | Unsigned | warns | no | no |
  | Invalid signature (modified after signing) | no | no | no |

  So timestamp production jars (`niagara.signing.standardtsa`, devguide `security/codeSigning.txt:127-133`).
- **The Signing Service is not module signing.** It issues CSR-signed TLS certificates for the Fox, Web and
  platform servers and MQTT client authentication (`SigningService/SupportedComponents-9530B192.txt:9-15` "Aws Jitp Mqtt Authenticator"); module,
  program-object and robot signatures come from a code-signing certificate. `[ev: corpus B1332 §1332.2]`
- **Harden the signing profile** `[ev: retro n4-course-build-deps-permissions-signing Δ4]` `[ev: corpus B1234 §1234.6]`:
  - Profile type `RestrictedSigningProfile` (the devguide sample, `codeSigning.txt:37`; plugin 7.6.17 `createProfile`
    writes it, corpus B1249 §1249.3): a missing alias fails the build, where a Local profile makes a new
    self-signed key. Key RSA 3072 (`codeSigning.txt:45`).
  - `allowDefaultProfile.set(false)` (above) stops the fallback to the default profile; together they make a
    wrong alias a build failure. The two root blocks (`signingServices{…}` + `niagaraSigning{…}`) are the whole
    portable unit: a part-level gradle file needs nothing (`templates/signing-root.gradle.kts`).
    `[ev: retro review2-signing-build-route Δ3]`
  - After changing the signing configuration run `:<part>:clean` before `jar`; then check Workbench Tools ›
    Module Info: a green check and, on double-click, the expected signer certificate
    (`ModuleSign/CheckingModuleSignaturesOnYourLocal-1B6825A0.txt:15-17` "In Workbench, click Tools > Module Info."). `verify-module.sh`'s `signed` row
    proves only that A signature exists, not whose; the signer-identity check is owed (Go lint G22).
    `[ev: retro review2-signing-build-route Δ1]`
  - The profile XML holds the store and key passwords in clear text: never print it, read element names only
    (`grep -o '<[a-zA-Z]*'`), keep it and every `*.p12`/`*.jks`/`*.jceks` outside the repo, and give a delegated
    task its path, never its contents. `[ev: retro review2-signing-build-route Δ2]`
- **"Unable to load …" at station start** (`ModuleSign/StationFailsWithErrorMessageModuleS-1BDB7827.txt:11-17` "If a module is installed via non-standard tools."):
  the module was installed by a non-standard tool, its signing certificate was removed from the trust store, or
  the certificate expired and the jar is not timestamped; Software Manager lists the signature errors. A
  third-party jar that cannot be re-signed by its author can be signed with Workbench Tools › Jar Signer Tool
  (`UsingJarSignerTool-29018E23.txt:9-13`, a last resort). Program objects and robots carry their own signature
  (`types/logic-authoring.md` § BProgram). `[ev: retro devguide415-r2-packaging-platform-signing Δ7]`
- **`skipModuleValidation` AND-gate — the property alone does nothing:** bypassing the sign-always groups (the six above) on the Win Supervisor requires BOTH `-Dniagara.classLoader.skipModuleValidation=true` (JVM system property) AND a `Webs.license` carrying `skipModuleValidation="true"` (verify expiry). Neither condition alone enables the bypass. NOT available on a JACE without a `Webs.license`. [ev: corpus B18 §18.3.2-18.3.3]
- **`exemptions.tes` — Workbench-user dev bypass:** the Workbench "Module XYZ is not validly signed. Trust permanently?" dialog adds a per-module entry to `user/security/exemptions.tes` (~16 KB binary), persisting across restarts. Scoped to the Workbench JVM only — does NOT affect the station daemon. Less invasive than JVM flags; requires admin credentials; Workbench audit-logs each addition. For Workbench-local inspection of unsigned modules only; `skipModuleValidation` (above) is the corresponding station-daemon path on the Win Supervisor. [ev: corpus B18 §18.9]

## Profile constraints — rt/ux Compact 3 `[ev: corpus B756 §756.2]`

The NRE ships the **Compact 3** JRE subset: `java.awt`, `javax.swing`, and `java.sql` are absent from the station classpath. Profile rules:

| Profile | JRE | Compact 3 required? |
|---------|-----|---------------------|
| `-rt` | NRE Compact 3 | **yes** — `java.awt`, `javax.swing`, `java.sql` absent |
| `-ux` | NRE Compact 3 | **yes** |
| `-wb` | Workbench full SE | no restriction |
| `-se` | full SE | no restriction |

An `-rt` class that imports `java.awt.*` compiles cleanly under JDK 8 (full SE) but throws `NoClassDefFoundError` on the station. `toolbelt/verify-module.sh --src` runs `check_compact3_imports` to WARN on these imports in `-rt`/`-ux` source.

- For guidance on bundling external Maven libraries (packaging paths, signing, the Java-8 ceiling, and a per-library viability table) see `types/third-party-libraries.md`.

## Workbench re-sign: STORED repackage
- **Workbench `JarFileSigner` "invalid entry compressed size (expected N got M)" is a deflater mismatch (WSL OpenJDK 8 vs Windows Zulu 8), NOT a build-state fluke a clean rebuild fixes — it recurs:** the same WSL deflater re-derives the same size, so `clean + slotomatic + jar` does not help. The fix is to repackage the jar STORED (uncompressed) so the mismatch is impossible by construction. [ev: retro dashboardpan-5rooms #10]
- **Local `jarsigner` is a FALSE NEGATIVE for this defect:** WSL `jarsigner` re-deflates with the SAME deflater that built the jar and reports "jar verified" while Workbench still fails — never use it to claim Workbench-signability. Only an operator's live Workbench sign proves it. [CERT-live 2026-09-01 — operator signed both STORED jars with no ZipException]
- **STORED is a MANUAL post-build step, only for the Workbench re-sign path — the gradle `build/libs` jars stay deflated:** run `toolbelt/stored-repack.sh <in.jar> <out.jar>` (manifest first, then every `META-INF/<ALIAS>.SF` with its `.RSA`/`.DSA`/`.EC` block — `NIAGARA4`, `CODESIGN`, `SERVER1`, … — then the rest, all STORED; it never overwrites) `[ev: retro n4-course-sample-check-precision Δ4]`. Since the supervisor accepts the DEV-signed deflated jar as-is, STORED matters only when a JACE forces a Workbench re-sign. Verify with `unzip -v` (0 `Defl:` entries) + `jarsigner -verify` ("jar verified"). [ev: retro dashboardpan-5rooms #10]

## Unit tests in WSL
- **`niagaraTest` is not a WSL-side gate (it needs a native Windows NRE with `bin/test` + a dev license; a WSL-side Gradle with plugin 7.6.17 discovers 0 tests — run it through `cmd.exe` per `types/moduleTest.md` § Running from WSL):** extract the control decision into a ZERO-Baja pure class and run it standalone. Resolve the cached jars with `JU=$(find ~/.gradle -name 'junit-4.13.2.jar' | head -1)` and `HC=$(find ~/.gradle -name 'hamcrest-core-1.3.jar' | head -1)` (never `find /` — it times out); the test must be same-package as the package-private pure class; use a fresh OUT dir. `toolbelt/run-pure-test.sh <module-rt-dir> <pkg.TestClass>` wraps this (exit 3 if the cache is empty — run one gradle build to fetch the jars); resolves src from any cwd — the runner sets the JVM working directory to `<module-rt-dir>` so a structural test that reads `Paths.get("src/…")` resolves regardless of where the caller invoked the script. [ev: retro coldroompan-rt-hardening] [ev: retro junit-standalone-cached-jar-locations-for-wsl-pure-tests] [ev: retro campaign10-run-pure-test-cwd]

### Logic series — scripted inputs, expected outputs, before the lab `[ev: retro o12-logic-series Δ1]`
A step-by-step JUnit test spells one scenario in Java; a **series** puts the scenario in a table the reader can
check against the spec line by line. The kit ships the runner as Java source (`toolbelt/java/n4kit/series`, Java 8,
no dependency); `run-pure-test.sh` puts it on the javac source path, so a module test imports `n4kit.series.*`
without copying it into the module.
1. **Write the series** under the module part, e.g. `srcTest/series/<scenario>.csv` (`#` comments, blank lines
   skipped):
   ```
   time_ms, in:temp, in:setpoint, in:enable, out:cooling, out:demand~0.5
   0,       4.0,     4,           true,      false,       0
   1000,    5.5,     4,           true,      true,        37.5
   4000,    6.0,     4,           true,      false,       -
   ```
   `time_ms` first and never decreasing; `in:<name>` / `out:<name>` columns; `out:<name>~<tol>` compares within
   `tol` (default exact); a cell is a number, `true` or `false` (1/0); an output cell may be `-` (not checked at
   that step) or `NaN`.
2. **Write the adapter** — a `SeriesModel` whose `step(SeriesRow r)` calls the pure core's `step(r.timeMs(),
   r.num("temp"), …, r.bool("enable"))` and whose `output(name)` returns each output (boolean as 1/0), throwing
   on an unknown name so a header typo fails loudly.
3. **Run it** in a JUnit test: `LogicSeries.fromFile("srcTest/series/<scenario>.csv").run(adapter)` returns the
   number of values checked; assert it is above 0. `run-pure-test.sh <module-rt-dir> <pkg.TestClass>` runs it
   (the path resolves from the module part, wherever the runner was called).

Results: exit 0 when every checked value matches. The first value outside its tolerance stops the run with exit 1
and one line `srcTest/series/<scenario>.csv:<line>: step <n> (t=<ms> ms) out:<name> expected <e> ±<tol>, actual
<a>`. A malformed series (missing `time_ms`, a column without `in:`/`out:`, no `out:` column, a wrong cell count, a
non-numeric cell, a negative tolerance, a duplicate column, time going back, no rows) is refused before any step as
`bad series <file>:<line>: <reason>`. Worked example: `tests/fixtures/logic-series/thermo-rt` (a toy thermostat,
its adapter test and series), pinned by `tests/logic-series.bats`.

A series checks the pure core only, offline and with exact time; it does not replace the lab restart proof (LP1,
below): it pins the decision sequence before the lab, and LP1 then proves what only a live station shows (boot
order, persisted state, the slot adapter). When LP1 or a field report finds a defect, add its input sequence as a
new series row block first (RED), then fix the core.

## How you know it's good — the 4-layer assurance stack
For any decision/safety logic, "done" means all four layers ran, in order:
1. **Pure JUnit** (`toolbelt/run-pure-test.sh`) — the ONLY executable coverage of control/safety logic in WSL; mandatory. Extract inline logic to a pure class first (see `types/logic.md` §Pure-class extraction).
2. **The verify gate** (`toolbelt/verify-module.sh`) — bytecode 52, signed, types resolve.
3. **A live cold-boot smoke on the TARGET distribution and version** (same vendor overlay or stock distribution and same Niagara release as the production station — a build `niagara_home` on another overlay or release proves nothing about a vendor/version mismatch; required before any production deploy, BUILD-LOOP §6) `[ev: retro panccadia-commissioning-lessons Δ10]` — for a timer-based control, read the live anchor slot (oBIX / Slot Sheet) after boot: anchor populated = the hook armed; anchor null in an arming mode = the hook did NOT run. Correct source ≠ correct behavior; this catches what a pure test cannot. [ev: retro hidden-actions-not-invocable-and-runtime-anchor-verification · T4]
4. **Adversarial pure-logic review BEFORE compile** — a second reader of the pure class catches control-logic defects the gate cannot (the `Long.MIN_VALUE` overflow and the dead-sensor false-fault both compiled and passed the gate). [ev: retro comppan-fase1-staging · T3]

## Lab restart proof (LP1) — boot outputs and persisted state `[ev: retro lc11-boot-outputs Δ2]`
Layer 3 for a control module with persisted state (`types/logic.md` LC10) or boot output holds (`types/logic.md`
LC11). A plain cold boot proves neither: with default timers and steady inputs the pre-steady cycle never runs, and
an unwatched boot hides a command that lasts seconds. This recipe found both defects of LC11 on client A's
compressor control module after a plain restart had passed. On a LAB station only, never production:
1. **Set NON-default timers** (power-on delay, minimum off, anti-short-cycle, start delays). A default of 0 disables
   the hold, so a boot bug in it cannot show. To tell two holds apart, set them far apart: a minimum-off of 5 min
   against a stage delay plus power-on delay of seconds shows which hold kept an output OFF; with close values the
   proof cannot say which one did. `[ev: retro lp1-lab-proof-traps Δ1]`
2. **Link EVERY input that gates demand**, each from a writable point. An unlinked numeric input reads `0.0 {ok}`,
   not null, and can win a min or lower-of-two choice, so the module never has demand and never commands an
   output: the proof then passes for the wrong reason.
3. **Keep writing inputs during boot.** Toggle each linked writable's value between restarts so it differs from the
   target slot's saved value; the boot-time link propagation is then a real change and runs the pre-steady cycle
   (`niagara-help/versions/4.15.6.20/devguide-clean/station.txt:26`). Make demand true before the restart so an
   output would be commanded ON.
4. **Start the read-backs before `Station Started`** and keep them going until well after steady state and the
   longest hold (every 2 s is enough). Timestamp every read and line it up against the station log's
   `Station Started` line; measure the offset between the reading host's clock and the station log's clock first
   (it was 15-16 s between a WSL host and a Windows station) and state it in the log.
5. **Check, per read**: every persisted slot keeps its saved value (LC10); every derived revision or "last valid"
   value is unchanged by the restart (LC11.3); every output command stays OFF until the hold expires, with no
   ON-OFF-ON at boot (LC11.1). One unexpected read fails the proof.
6. **Repeat once with each backup file missing** (move it out of `shared/` before the start, put it back after),
   so the seed runs from the slots alone and a "restore if newer" guard is exercised against the slot.
7. **Clean up and prove it**: remove the proof folder and save, remove the module jar if it was installed only for
   the proof, move the module's backup files out of `shared/`, restart, and show that the station is back
   (`Station Started` in the log, a `config.bog` grep for the proof names returns 0). Grep the module's TYPE
   PREFIX as it is written in `config.bog` (the `prefix` of the module's `module.xml`, e.g. `ABCPAN:` for a
   module named `AbcPan`), not the module name: a grep for `AbcPan:` can never match and passes for the wrong
   reason.

Two traps that make the proof pass without testing what it claims `[ev: retro lp1-lab-proof-traps Δ2]`:
- **A rollback is not a restart.** Removing the proof component and restoring it with `n4_rollback` creates a NEW
  instance (`started()` on a fresh object); it never runs `stopped()` then `started()` on the same object. To prove
  the stop/start path, restart the station, or disable and re-enable the component from Workbench.
- **Touch the jar after swapping a module under the same file name.** Removing and re-adding a jar with the same
  name can leave the module registry "Up-to-date" so the station runs the old classes. Run `touch` on the jar before
  the start and check that the log shows `Module changed … Rebuilding registry` (mcp-n4 METHODOLOGY, lab module
  swap, M14).

Keep the log (setup, every read, the station log lines, cleanup) as the proof's evidence file.

## Known gap — mode B ignores `--with-slotomatic`
- **Only mode A regenerates `-ux` slots, and only when `-ux` is annotated** (`scripts/ng-deploy.sh` `run_slotomatic`: `:<MOD>-ux:slotomatic` is added in mode A when `ux_has_annotations`). Mode C builds and regenerates `-rt` only; mode B (the `-ux`-only build) never runs slotomatic: given `--with-slotomatic` it prints two `WARN` lines on stderr and builds without it. So a `-ux` annotation edit deployed with mode B or C is not regenerated — use `--mode A --with-slotomatic`, or `toolbelt/build.sh` (slotomatic for every profile). [ev: QA audit · ng-deploy.sh]

## Verify — useful patterns
- **V1 / Consumer-absence delta proof:** to prove "what changed since X" when same-day timestamps don't discriminate, `grep -c <symbol>` the CONSUMER artifact — 0 hits = genuine delta; present = already consumed. Beats pinning a commit/deploy boundary by date. [ev: retro dashboardpan-2d-to-3d-port-multi-session-coordination-retro Δ1]
- **V2 / Live-vs-doc precedence:** for behavior the live system arbitrates (e.g. is a setpoint writable?), verify live and let it override the doc — then fix the doc. A PORT-SPEC said "setpoint writable"; live oBIX `PUT` returned 400; the spec was wrong. [ev: retro live-cutover-and-authenticated-control-retro Δ2]
- **V3 / Verify freshness before labeling "live":** probe max `ts` is advancing ≈ now; if frozen, label honestly (SNAPSHOT / última lectura `<ts>`). [ev: retro live-cutover-and-authenticated-control-retro Δ1]
- **V4 / Headless-QA/CORS boundary:** a headless-from-localhost e2e cannot cross a browser CORS origin by design — confirm the backend with `curl` and verify `Access-Control-Allow-Origin` separately; a CORS block is not a code bug. [ev: retro live-cutover-and-authenticated-control-retro Δ3]

## Frontend verify step (`-ux` dashboards)
For every `-ux` change, after the build and before hand-off:
1. `toolbelt/report-module.sh <module-root> --profile <ui_profile> [--legacy]` — the module's
   `ui_profile` comes from its BUILD-STATE envelope (read it with `toolbelt/kit-state.sh build-state`;
   the caller passes it; the toolbelt never parses BUILD-STATE). It relays `rc-scan.sh` (browser-floor FAIL for `hmi`/`both`), the kit ESLint config
   on the artifact's own rc js, and `lint-vendor-floor.sh` on `rc/vendor`. `--legacy` only for a
   deployed module not yet restructured (data-URI budget WARN instead of FAIL).
   `[ev: retro dashboard-frontend-standard Δ10]` `[ev: retro dashboard-frontend-standard Δ16]`
2. **No-scroll at the panel resolution** (`ui_profile` `hmi` or `both`): run
   `node toolbelt/hmi-sweep.js --url <preview-url> --target <alarm-banner selector>` (plus
   `--subtab`/`--scenario`/`--api-match` as the SPA needs) against the preview server; exit 1 = a
   view scrolls or the banner is covered. A text-only lint cannot see layout overflow, so this
   sweep is the check, not `rc-scan.sh`. `[ev: retro panccadia-persistent-config-hoa Δ4]`
   `[ev: retro comppan-fase2-amps-alarms Δ3]`
- Tools, installed once per machine: `npm install --prefix build-n4-module-kit/toolbelt/eslint`
  (pinned eslint + acorn, `toolbelt/eslint/package.json`); puppeteer-core + a Chrome for the sweep
  (`KIT_PUPPETEER`/`NODE_PATH`, `KIT_CHROME`). A missing tool is a SKIP row naming it (exit 4 for
  the standalone scripts) — report it as "not run", never as a pass.

## Client repository CI `[ev: retro dashboard-deployment-profiles Δ5]`
A client repository (private; module sources live there, not in the kit) gets CI by reusing this
kit's workflow pattern: Java 8 Temurin, JUnit 4.13.2 + hamcrest-core 1.3 fetched with pinned
sha256 (a mismatch is a hard stop, never a silent re-pin), then on every push the pure JUnit tests,
`rc-scan.sh`, the frontend lints and `report-module.sh`. No station, no secrets: everything runs on
sources. Check out the kit at a pinned tag, not a branch, so a kit change cannot break client CI
silently. Skeleton (adapt paths; `<GROUP>/<Module>` is the module root):

```yaml
name: module-ci
on: { push: { branches: [main] }, pull_request: {} }
permissions: { contents: read }
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/checkout@v5
        with: { repository: <owner>/niagara-tools, ref: <kit-tag>, path: kit }
      - uses: actions/setup-java@v5
        with: { distribution: temurin, java-version: '8' }
      - uses: actions/setup-node@v5
        with: { node-version: '24' }
      - name: JUnit + hamcrest (pinned sha256, same values as the kit CI)
        run: |
          set -eu
          JU_DIR="$HOME/.gradle/caches/modules-2/files-2.1/junit/junit/4.13.2/8ac9e16d933b6fb43bc7f576336b8f4d7eb5ba12"
          HC_DIR="$HOME/.gradle/caches/modules-2/files-2.1/org.hamcrest/hamcrest-core/1.3/42a25dc3219429f0e5d060061f71acb49bf010a0"
          mkdir -p "$JU_DIR" "$HC_DIR"
          curl -fsSL -o "$JU_DIR/junit-4.13.2.jar" https://repo1.maven.org/maven2/junit/junit/4.13.2/junit-4.13.2.jar
          curl -fsSL -o "$HC_DIR/hamcrest-core-1.3.jar" https://repo1.maven.org/maven2/org/hamcrest/hamcrest-core/1.3/hamcrest-core-1.3.jar
          echo "8e495b634469d64fb8acfa3495a065cbacc8a0fff55ce1e31007be4c16dc57d3  $JU_DIR/junit-4.13.2.jar" | sha256sum -c -
          echo "66fdef91e9739348df7a096aa384a5685f4e875584cce89386a7a47251c4d8e9  $HC_DIR/hamcrest-core-1.3.jar" | sha256sum -c -
      - name: Frontend lint tools (pinned)
        run: npm install --no-audit --no-fund --prefix kit/build-n4-module-kit/toolbelt/eslint
      - name: Pure tests
        run: kit/build-n4-module-kit/toolbelt/run-pure-test.sh <GROUP>/<Module>/<Module>-rt <pkg.TestClass>
      - name: Aggregated report (rc-scan, ESLint, vendor floor, rt lints)
        run: kit/build-n4-module-kit/toolbelt/report-module.sh <GROUP>/<Module> --profile <ui_profile>
```

One `run-pure-test.sh` line per pure test class. The report needs no built jar (verify-module rows
SKIP without one); the jar gate stays local where the Niagara SDK lives.

**When the build or tests need the proprietary Niagara jars, hosted CI cannot run them: gate locally.** On each
exact pushed head, run a fresh mirror (`mirror-niagara-home.sh`), `build.sh` against it and the client's WSL test
runner, then post a PR comment as the record (head SHA, build result, test counts, mirror source). Merge only after
that comment, and re-gate after every push or main merge. `[ev: retro r10-client-gate-retro Δ5]`

## Deploy (station)
Stop station → replace the jars in `<niagara_home>/modules/` → start (the jar is locked while the station runs; to avoid stopping a live supervisor, build against a mirror per §Building against a running station and copy `build/libs` → station). Signing per target: §Signing per deploy target. **A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.**

See also: `docs/module-dev-workflow.md` (toolchain, codegen round-trip, dev loop, testing).

## Toolbelt authoring — composition conventions
- **One rule, two scripts: duplicate only the rule table and pin the copies with a test.** When a check already lives in another toolbelt script and a shared `toolbelt/lib` file is not an option, copy the requirement table verbatim (e.g. `lint-structure.sh` `L15_SIGN_ERE` = `preflight.sh` `SAMPLE_SIGN_ERE`) and add a bats test that fails when they differ (`tests/lint-structure.bats` LS15-sync). `[ev: retro r8-t9i-libs Δ2]`
- **Prefer row-level output parsing over exit-code-only aggregation when composing toolbelt scripts:** a member tool that exits 0 may still emit WARN rows that must propagate (e.g. `slot-coverage.sh` exits 0 on <100% coverage); exit codes are a secondary ORed guard for crashed members. [ev: retro campaign7-report-module]
- **Emit a single count row (PASS N=0 / FAIL N>0) when relaying dup-count checks from a composed tool:** relaying individual WARN lines per duplicate key adds noise in multi-artifact reports; a count row is cleaner and machine-parseable. [ev: retro campaign7-report-module]
- **A bats assertion is one command on its own line, never an earlier element of an `&&` list or a bare `! cmd`:** bats runs a test body under errexit, and bash ignores errexit for every command of an `&&` list except the last one and for a negated command, so mid-test `[ "$status" -eq 1 ] && [[ "$output" == *x* ]]` never fails its test when the status is wrong (a mutation run of a new lint stayed green on exactly that line). Write one assertion per line; negate with `run ! cmd` (bats 1.5+, it overwrites `$status`/`$output`) or `! cmd || return 1`. `toolbelt/lint-bats-errexit.sh` FAILs both forms in `@test` bodies, and `tests/lint-bats-errexit.bats` BEX-kit runs it over the kit's own `tests/`. `[ev: retro r10-bats-list-hazard Δ1]`
- **A close-gate pin references a durable archived fixture, never a session scratchpad:** exact counts pinned to a file that lived only in a temporary session copy become unverifiable once it is cleaned (c8-close SC16-smoke, retired as a loud skip). A close fixture is a client station bog, so it lives in a private location outside this public repository (a private repository or a configured private directory), never in it; the public gate records only the fixture's sha256 and an env var naming the private file, and SKIPs loudly when that file is absent or its sha256 does not match. [ev: retro r10-c8-close-retire Δ1]
