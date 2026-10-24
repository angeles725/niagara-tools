# Build & verify — Java 8 + slotomatic (the ONLY valid build)

A `gradle :jar` with the default JDK is NOT a valid N4 build: it can use the wrong Java (major-65 bytecode Niagara's Java-8 runtime rejects) and it skips slotomatic, so annotation bugs never surface. Chihuahua's `deploy.sh` uses `JAVA8=/usr/lib/jvm/java-8-openjdk-amd64`.

## Doctrine — which command does what

| Command | Role |
|---|---|
| `toolbelt/verify-module.sh` | **THE verify gate** — run it on the built jars regardless of who built them. |
| `toolbelt/build.sh` | The recommended WSL build: clean + slotomatic (for every profile that has sources) + jar, then it calls the gate automatically. |
| `scripts/ng-deploy.sh` | The station DEPLOY wrapper: backup → build → copy → type-count verify (its slotomatic guard is **rt-only**); run `verify-module.sh` on `build/libs` afterwards. |

> A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.

`verify-module.sh` default checks are conservative (major 52, a `META-INF/*.SF` signature block, module.xml types resolve); `--target-version`, `--stored` and `--src` are opt-in.

`build.sh` usage: `build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] [--ui-profile hmi|lan|both|unknown] [--legacy] [--first-deploy] [--gradle-prop k=v]... <module-root> <MOD> [niagara_home]` (exit **53**: the report-module verdict is INCOMPLETE — a floor check SKIPped; see BUILD-LOOP §5) — exits **0** build+gate passed · **2** usage · **10** environment (no JDK 8, not a niagara_home, no profile, preflight FAIL; a repo root passed instead of the gradle root lists the gradle roots below it `[ev: retro comppan-fase2-amps-alarms Δ2]`) · **30** gradle failed · **31** `:clean` blocked by a locked `modules/<jar>` · **32** the plugin's post-jar copy into `modules/` hit `FileAlreadyExistsException` (a Windows process holds the old jar; the `build/libs` jar rebuilt this run is the artifact of record, the gate did not run) `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ3]` · **33** a mirror `niagara_home` path, or in mirror mode a `$HOME` that `-Dmaven.repo.local` would carry, contains whitespace (§Mirror; gradle never ran) `[ev: retro r8-mirror-userhome Δ1]` `[ev: retro r8-buildsh-316-308]` · **34** gradle passed but Slot-o-Matic skipped a file (`FAIL slotomatic-skip <path> (<reason>)` rows; the pre-build `modules/` jar is restored and the gate does not run; METHODOLOGY K15) `[ev: retro r10-slotomatic-skip-fail-open Δ1]` · **50** gate or report-module failed · **51** deployed-baseline drift. `--plugin-version` is also forwarded to `preflight.sh --plugin-version` `[ev: retro continuous-fan-post-defrost-delay Δ1]`; `--ui-profile`/`--legacy` are forwarded to `report-module.sh --profile`/`--legacy`. It runs `verify-module.sh --src <module-root>/<MOD>` on every produced jar; `--plugin-version` (or `$NIAGARA_PLUGIN_VERSION`) forwards `-PniagaraPluginVersion` (each install ships one: 4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22).

## Station deploy: `ng-deploy.sh` (the established wrapper)

Use `niagara-tools/scripts/ng-deploy.sh` — it already does backup → build → (slotomatic) → copy-to-station → verify, and knows the flags:
```
scripts/ng-deploy.sh --mode A --with-slotomatic        # rt+ux, regenerate from annotations, deploy+verify
scripts/ng-deploy.sh --strict-slotomatic               # ABORTS (exit 15) if annotation changes but slotomatic wasn't run
scripts/ng-deploy.sh --no-deploy --with-slotomatic     # build-only (jars stay in build/libs)
```
Set env in `.env.local` (or `--env-file`): `MODULE_NAME, GRADLEW_PATH, NIAGARA_HOME, NIAGARA_USER_HOME, JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64, STATION_MODULES_DIR`.
`--strict-slotomatic` is the guard that catches the annotation-vs-generated bug automatically — prefer it.

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
- **The Windows route picks its JDK from Gradle's toolchains, not from `build.sh`:** the 4.15 course `gradle.properties` leaves `org.gradle.java.installations.*` commented and says a build needs a full JDK 8; list what Gradle sees with `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true` and set `paths` when no JDK 8 is listed. `preflight.sh` `jdk8` checks only the WSL JDK that `build.sh` passes (`toolbelt/preflight.sh:117-170`); the toolchain check is owed (G61). `[ev: course Slide Code/gradle.properties:14-25]` `[ev: corpus B1235 §1235.2]` `[deferred: retro n4-course-build-deps-permissions-signing Δ5]`

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
Building on WSL against `niagara_home=C:\...` breaks the m2 repo (see §Build target & plugin version); and when a `-ux` depends on `-rt`, the plugin's auto-copy of the built jar into a station-locked `modules/` fails. That copy failure is irrelevant — free the lock, or use the already-assembled `build/libs` jar (§Building against a running station: free the lock first, mirror only when you can't); the mirror is only for a live production supervisor.

## Build location — ext4 vs 9p/drvfs `[ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ7]`
- **Work under an ext4 path (e.g. `~/modulos_niagara_n4/Cliente/...`), never directly under `/mnt/<drive>` (a WSL 9p/drvfs mount), for both the source tree AND `niagara_home`.** A live field session ran the source tree at `/mnt/c/Users/.../Downloads` and `niagara_home=/mnt/c/Honeywell/...` (1013 module jars, 9p) — a measured file walk took 0.200s per file there vs 0.020s on ext4 (10x). Gradle's per-file stat/hash work multiplies that gap, turning a normal 3-5s build into minutes.
- `toolbelt/build.sh` WARNs (non-fatal — the build still runs) when either the gradle root or `niagara_home` resolves to a `9p`/`drvfs` mount. Fix by cloning the source tree to ext4 and pointing `niagara_home` at an ext4 mirror via `toolbelt/mirror-niagara-home.sh` (§Mirror above) instead of the raw `/mnt/c/...` install.
- The detection is filesystem-type based (`df -T`), not a path-string check — a symlink or bind-mount from ext4 back onto `/mnt/c` would still read as `9p`/`drvfs` and correctly WARN.

## Building against a running station: free the lock first, mirror only when you can't
- **Free the lock FIRST — it's the common case:** an `Unable to delete …/modules/<mod>.jar` at `:clean` is a Workbench / your-own-station lock; close Workbench (or, when the holder is NOT a live production supervisor, stop it) and build directly against the install. This recurred 3× in one session and closing Workbench cleared it each time. Use the MIRROR (below) ONLY when the holder is a running station you must not stop, or a box you do not own — the mirror is the documented fallback, not the default. [ev: retro coldroompan-dashboardpan-freeze-stat-leds · B2]
- **A build against a RUNNING station reports FAILED at the `modules/` copy, but `build/libs/<jar>` is already fully assembled + signed:** the plugin auto-installs the jar as the last `jar` step; a locked `modules/` makes gradle exit FAILED, yet the artifact in `build/libs` passed the gate — verify THAT jar and use it. The copy failure is irrelevant when you are not deploying to that supervisor. [ev: retro dashboardpan-ux-direct-build · B1]
- **A brand-new module (not yet in `modules/`) needs NO station stop:** a first build has no deployed jar to overwrite, so `:clean :jar` builds green against a live install with the station running — only an UPDATE hits the lock. [ev: retro comppan-fase1-staging · B3]

### Mirror (the fallback for a live production supervisor)
- **A running station LOCKS `<niagara_home>/modules/<mod>.jar` — never stop a live supervisor to build:** with `station`/`niagarad` running, the plugin's `clean`/`jar` fails deleting the deployed jar (`IOException: Unable to delete file …/modules/<mod>.jar`). Build against a read-only MIRROR: symlink every top-level entry + every `modules/*.jar` EXCEPT the one being built into a dir with a WRITABLE `modules/`, and build with `-Pniagara_home=<mirror>`. The jar lands in `build/libs` (deliverable) + `<mirror>/modules` (throwaway); the real install is never written. [ev: retro coldroompan-rt-hardening #8]
- Use `toolbelt/mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]`. Two guards protect you: it refuses when the mirror equals, sits inside, or contains the source (exit 20), and refuses to wipe an existing dir that lacks a `.niagara-mirror` marker (exit 20); it writes `.niagara-mirror` on creation.
- **A mirror build isolates `user.home` — the signing plugin must not read your real keystore:** the niagara-signing plugin's default profile is `user.home/.tridium/security/niagara.signing.xml` (SigningProfileFactory, niagara-plugins 7.6.17). `org.gradle.jvmargs=-Duser.home=…` does NOT reach an already-running Gradle daemon, so a mirror build read the real `~/.tridium/security/niagara.signing.{xml,jceks}`. When `niagara_home` holds `.niagara-mirror`, `toolbelt/build.sh` appends `-Duser.home=<mirror>/.mirror-user-home` to `JAVA_TOOL_OPTIONS` (any existing value kept), keeps `GRADLE_USER_HOME` (default `~/.gradle`, so offline dependency resolution still works), appends `-Dmaven.repo.local=<real ~/.m2/repository>` so `mavenLocal()` does not move with `user.home` (unless `JAVA_TOOL_OPTIONS` already sets `maven.repo.local`) and passes `--no-daemon`. The JVM splits `JAVA_TOOL_OPTIONS` on whitespace (JDK 8 has no quoting), so a mirror path with whitespace is refused with exit 33 before gradle runs, and so is a `$HOME` with whitespace unless `JAVA_TOOL_OPTIONS` already sets `-Dmaven.repo.local=` (the appended `$HOME/.m2/repository` would be split the same way; outside mirror mode nothing is appended and such a `$HOME` builds). A raw `./gradlew` mirror build must do the same by hand. Such a build therefore cannot pick up your real DEFAULT profile. An explicit profile (`niagaraSigningProfile`, passed as `--gradle-prop niagaraSigningProfile=<path>` or `ORG_GRADLE_PROJECT_niagaraSigningProfile`, #308) still signs. A root with the signing block and no profile fails `:jar`. A root WITHOUT the block makes the plugin create `.mirror-user-home/.tridium/security/niagara.signing.{xml,jceks}` and sign with a throwaway Niagara4Modules dev key (observed with a redirected `user.home`); every profile under `.mirror-user-home` is plugin-created (never yours) and persists until the mirror is refreshed, so `build.sh` WARNs on stderr after every successful mirror build while that `niagara.signing.xml` exists, not only the build that created it, unless the root's uncommented code disables the default profile (`allowDefaultProfile.set(false)` or `= false`, the lint-structure L15 pattern from `toolbelt/lib/signing-block.sh`): such a root cannot fall back to that profile, so a passing build signed with the explicit one (exit stays 0 — do not deploy that jar; add the signing block, `templates/signing-root.gradle.kts`, lint-structure L15). So read the `signed` row of `verify-module.sh` before deploying. A mirror refresh wipes `.mirror-user-home` with the rest of the mirror. `[ev: CERT-live local build 2026-10-22]` `[ev: retro r8-mirror-userhome Δ1]`

## Verify
```
# THE gate — run on the built jars regardless of who built them:
toolbelt/verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] <jar>...
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

Reviewed false positives in the advisory source lints: `lint-arbitrary-ord.sh` WARNs on every `BOrd.make(<variable>)`; when the value is built from a code-controlled internal string (never client input), put `// lint-arbitrary-ord: reviewed <reason>` on the same line or the line directly above the call. The reason is mandatory — a bare marker still WARNs ("reviewed marker without a reason"). The marker records a review; it is not an allowlist for client-supplied ORDs (types/security.md §4). `[ev: retro panccadia-restart-seq-comp-lockout-hours Δ4]`

## Pre-release real-jar smoke test
Before shipping a kit or toolbelt change, run `toolbelt/verify-module.sh` over at least one known-good and one known-bad real module jar — the bats suites use generated fixtures, which only simulate a malformed jar; a real one proves the gate catches a live defect. Worked known-bad: `ColdRoomPan-rt.jar` fails the `types` check because `module-include.xml` still declares `com.angeles.ColdRoomPan.BHoaMode` after that class was deleted — the live "Missing class ColdRoomPan:HoaMode" defect. DashboardPan's jars pass `--src` clean. Usage: see §Verify above.

- **Smoke every new parser on ≥1 real module before merge:** bats fixtures are synthetic and may miss edge cases (field ordering, encoding, missing files) that a real module tree exposes; use at least one known-good and one known-bad real module as the final gate. [ev: retro campaign7-schema-risk]

## Schema-risk deploy workflow `[ev: retro tree-selection-and-schema-risk-baseline Δ2]`

The `schema-risk.sh` "before" snapshot must be the **DEPLOYED source**, not the current work tree:
1. Extract the deployed baseline from the station backup or the original RAR/ZIP: the result must be a layout with `module-include.xml` + `*.java` source files (the same layout `schema-risk.sh` expects).
2. Run `toolbelt/schema-risk.sh <before-dir> <after-dir>` where `<after-dir>` is the current build tree.
3. **A LOSSY verdict (`remove_slot_*`) must be cross-checked against the live bog before treating it as a blocker:** a slot that was removed but never saved in the `.bog` (0 persisted instances) is non-dangerous. Use `toolbelt/bog-audit.sh <config.bog> --module <MOD>` or grep the raw `config.bog` XML for the removed slot name to confirm no live instance persists it. A LOSSY + live instances = blocker; a LOSSY + 0 instances = safe for this station (but note it for other stations).
4. An OUTAGE verdict is always a blocker regardless of bog state — a retype crash has no safe-persistence case.

**Why the work-tree snapshot is wrong:** the current work tree is the BUILD baseline, not the DEPLOYED baseline. Version drift between the work tree and the live station (e.g. Dashboard 2.0 live vs 2.2.0 in the tree) makes the schema-risk diff report the wrong deltas — it sees code changes that were already deployed as new schema changes.

## Signing per deploy target
- **Check the deploy target's signing policy before assuming a Workbench re-sign:** a Honeywell supervisor ACCEPTS gradle's per-machine DEV cert — no re-sign needed (chihuahua's `deploy.sh` only builds + copies and runs on the same supervisor). A JACE field controller enforces the project CA (e.g. `angelessigner`), so a JACE-bound module IS re-signed. [CERT-live 2026-09-01 · retro 5rooms #9]
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
  issue #308). `[CERT-live local build]` `[ev: retro r8-buildsh-316-308]`
  With a throwaway self-signed JCEKS key and a hand-written profile (Java properties XML, devguide
  `security/codeSigning.txt:29-47`; keys `niagara.signing.profileType`, `storetype`, `storepath`, `storepass`,
  `keypass.<alias>`), the build SIGNS: `META-INF/<ALIAS8>.SF`/`.RSA` (alias cut to 8 upper-case chars), `jarsigner
  -verify` passes; no CA chain or timestamp server is needed (the TSA is contacted only when
  `niagara.signing.standardtsa` is set, devguide `codeSigning.txt:113-135`). `allowDefaultProfile.set(false)` is what
  refuses on its own: with no profile, `:jar` FAILS "The default profile is not available. Specify one explicitly";
  with the line removed, the plugin creates `user.home/.tridium/security/niagara.signing.{xml,jceks}` and the build
  passes signed `NIAGARA4` by a "For Development Purposes Only" Niagara4Modules certificate. `[CERT-live local build]`
  A production CA-issued profile was not exercised (none on that machine).
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
  Low" (`ModuleSign/VerificationModesModuleSigning-182D0E86.txt:17`) while its own roll-out page says Medium from 4.9
  (`ModuleSign/StagedRoll-outModuleSigning-182E7CC0.txt:11`). Code wins for behaviour: with no property, `Nre` asks
  `ModuleVerificationMode.make`, which returns `medium` for baja ≥ 4.9 (vineflower baja 4.14.0.162 `Nre.java:752-763`;
  decompiled nre-ext 4.14 `ModuleVerificationMode.java:11,28-33`; 4.13 and 4.15 not verified). The authority on a
  given host is `niagara.moduleVerificationMode` in its `system.properties`. `[ev: corpus B1010]`
  `[ev: retro devguide415-r2-packaging-platform-signing Δ6]`
- **Mode × signature status** (4.15.6.20 `ModuleSign/SignatureStatusesModuleSigning-1F0024B2.txt:9-21`,
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
  platform servers and MQTT client authentication (`SigningService/SupportedComponents-9530B192.txt:9-15`); module,
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
    (`ModuleSign/CheckingModuleSignaturesOnYourLocal-1B6825A0.txt:15-17`). `verify-module.sh`'s `signed` row
    proves only that A signature exists, not whose; the signer-identity check is owed (Go lint G22).
    `[ev: retro review2-signing-build-route Δ1]`
  - The profile XML holds the store and key passwords in clear text: never print it, read element names only
    (`grep -o '<[a-zA-Z]*'`), keep it and every `*.p12`/`*.jks`/`*.jceks` outside the repo, and give a delegated
    task its path, never its contents. `[ev: retro review2-signing-build-route Δ2]`
- **"Unable to load …" at station start** (`ModuleSign/StationFailsWithErrorMessageModuleS-1BDB7827.txt:11-17`):
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

## How you know it's good — the 4-layer assurance stack
For any decision/safety logic, "done" means all four layers ran, in order:
1. **Pure JUnit** (`toolbelt/run-pure-test.sh`) — the ONLY executable coverage of control/safety logic in WSL; mandatory. Extract inline logic to a pure class first (see `types/logic.md` §Pure-class extraction).
2. **The verify gate** (`toolbelt/verify-module.sh`) — bytecode 52, signed, types resolve.
3. **A live cold-boot smoke on the TARGET distribution and version** (same vendor overlay or stock distribution and same Niagara release as the production station — a build `niagara_home` on another overlay or release proves nothing about a vendor/version mismatch; required before any production deploy, BUILD-LOOP §6) `[ev: retro panccadia-commissioning-lessons Δ10]` — for a timer-based control, read the live anchor slot (oBIX / Slot Sheet) after boot: anchor populated = the hook armed; anchor null in an arming mode = the hook did NOT run. Correct source ≠ correct behavior; this catches what a pure test cannot. [ev: retro hidden-actions-not-invocable-and-runtime-anchor-verification · T4]
4. **Adversarial pure-logic review BEFORE compile** — a second reader of the pure class catches control-logic defects the gate cannot (the `Long.MIN_VALUE` overflow and the dead-sensor false-fault both compiled and passed the gate). [ev: retro comppan-fase1-staging · T3]

## Known gap — mode B ignores `--with-slotomatic`
- **Modes A and C regenerate `-ux` slots when `-ux` is annotated** (`scripts/ng-deploy.sh` lines ~493-500: `run_slotomatic` is called for every profile with sources); **mode B never runs slotomatic** (lines ~552-553). So a `-ux` annotation edit under mode B silently skips regeneration — build with `toolbelt/build.sh` (runs slotomatic for every profile) or with `--mode A --with-slotomatic` instead. Mode B is the exception: slotomatic is skipped for all profiles, not just the runtime profile. [ev: QA audit · ng-deploy.sh]

## Verify — useful patterns
- **V1 / Consumer-absence delta proof:** to prove "what changed since X" when same-day timestamps don't discriminate, `grep -c <symbol>` the CONSUMER artifact — 0 hits = genuine delta; present = already consumed. Beats pinning a commit/deploy boundary by date. [ev: retro dashboardpan-2d-to-3d-port-multi-session-coordination-retro Δ1]
- **V2 / Live-vs-doc precedence:** for behavior the live system arbitrates (e.g. is a setpoint writable?), verify live and let it override the doc — then fix the doc. A PORT-SPEC said "setpoint writable"; live oBIX `PUT` returned 400; the spec was wrong. [ev: retro live-cutover-and-authenticated-control-retro Δ2]
- **V3 / Verify freshness before labeling "live":** probe max `ts` is advancing ≈ now; if frozen, label honestly (SNAPSHOT / última lectura `<ts>`). [ev: retro live-cutover-and-authenticated-control-retro Δ1]
- **V4 / Headless-QA/CORS boundary:** a headless-from-localhost e2e cannot cross a browser CORS origin by design — confirm the backend with `curl` and verify `Access-Control-Allow-Origin` separately; a CORS block is not a code bug. [ev: retro live-cutover-and-authenticated-control-retro Δ3]

## Frontend verify step (`-ux` dashboards)
For every `-ux` change, after the build and before hand-off:
1. `toolbelt/report-module.sh <module-root> --profile <ui_profile> [--legacy]` — the module's
   `ui_profile` comes from its BUILD-STATE envelope (the caller passes it; the toolbelt never parses
   BUILD-STATE). It relays `rc-scan.sh` (browser-floor FAIL for `hmi`/`both`), the kit ESLint config
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

## Deploy (station)
Stop station → replace the jars in `<niagara_home>/modules/` → start (the jar is locked while the station runs; to avoid stopping a live supervisor, build against a mirror per §Building against a running station and copy `build/libs` → station). Signing per target: §Signing per deploy target. **A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.**

See also: `docs/module-dev-workflow.md` (toolchain, codegen round-trip, dev loop, testing).

## Toolbelt authoring — composition conventions
- **One rule, two scripts: duplicate only the rule table and pin the copies with a test.** When a check already lives in another toolbelt script and a shared `toolbelt/lib` file is not an option, copy the requirement table verbatim (e.g. `lint-structure.sh` `L15_SIGN_ERE` = `preflight.sh` `SAMPLE_SIGN_ERE`) and add a bats test that fails when they differ (`tests/lint-structure.bats` LS15-sync). `[ev: retro r8-t9i-libs Δ2]`
- **Prefer row-level output parsing over exit-code-only aggregation when composing toolbelt scripts:** a member tool that exits 0 may still emit WARN rows that must propagate (e.g. `slot-coverage.sh` exits 0 on <100% coverage); exit codes are a secondary ORed guard for crashed members. [ev: retro campaign7-report-module]
- **Emit a single count row (PASS N=0 / FAIL N>0) when relaying dup-count checks from a composed tool:** relaying individual WARN lines per duplicate key adds noise in multi-artifact reports; a count row is cleaner and machine-parseable. [ev: retro campaign7-report-module]
