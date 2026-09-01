# Build & verify — Java 8 + slotomatic (the ONLY valid build)

A `gradle :jar` with the default JDK is NOT a valid N4 build: it can use the wrong Java (major-65 bytecode Niagara's Java-8 runtime rejects) and it skips slotomatic, so annotation bugs never surface. Chihuahua's `deploy.sh` uses `JAVA8=/usr/lib/jvm/java-8-openjdk-amd64`.

## Doctrine — which command does what

| Command | Role |
|---|---|
| `toolbelt/verify-module.sh` | **THE verify gate** — run it on the built jars regardless of who built them. |
| `toolbelt/build.sh` | The recommended WSL build: clean + slotomatic (for every profile that has sources) + jar, then it calls the gate automatically. |
| `scripts/ng-deploy.sh` | The station DEPLOY wrapper: backup → build → copy → type-count verify (its slotomatic guard is **rt-only**); run `verify-module.sh` on `build/libs` afterwards. |

> A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.

`verify-module.sh` default checks are conservative (major 52, NIAGARA4.SF, module.xml types resolve); `--target-version`, `--stored` and `--src` are opt-in.

`build.sh` usage: `build.sh [--profiles rt,ux,wb] [--target-version X.Y] [--plugin-version V] <module-root> <MOD> [niagara_home]` — exits **0** build+gate passed · **2** usage · **10** environment (no JDK 8, not a niagara_home, no profile) · **30** gradle failed · **50** gate failed. It runs `verify-module.sh --src <module-root>/<MOD>` on every produced jar; `--plugin-version` (or `$NIAGARA_PLUGIN_VERSION`) forwards `-PniagaraPluginVersion` (each install ships one: 4.13.2→7.3.40, 4.14→7.6.17, 4.15.3→7.6.22).

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
- On WSL, `gradle.properties` may pin a Windows JDK path (`C:\Program Files\Zulu\zulu-8`) — that breaks WSL (`Illegal character ... C:\`). Override with `-Porg.gradle.java.installations.paths=$J8`. On Windows the pinned Zulu 8 is correct.

## Build target & plugin version
- **Build against the `niagara_home` of the LOWEST target version you must support:** the manifest stamps its `baja` dependency version, and a 4.14 station REJECTS a jar built against 4.15. Check `unzip -p <jar> META-INF/module.xml | grep baja`. [ev: retro rt-hardening #6]
- **The gradle plugin version pinned in `settings.gradle.kts` MUST exist in `<niagara_home>/etc/m2` — each install ships exactly ONE (table below):** Tridium plugins are not on the public portal, so a version the install lacks fails "plugin not found". Make it overridable — `val gradlePluginVersion = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")` + `-PniagaraPluginVersion=<v>`. This override is MANDATORY, not optional. [ev: retro rt-hardening #7]
- **`gradle.properties` can LIE about the real target:** the true target = intersection(the `niagara_home` that has the pinned plugin, the station that accepts the stamped `baja`). Check `settings.gradle.kts` and `ls <niagara_home>/etc/m2/repository/com/tridium/niagara-module/...` before trusting `gradle.properties`. [ev: retro 5rooms #6]
- **A Windows `niagara_home` path breaks the m2 repo under WSL:** gradle resolves `maven(file:/C:/...)` → "plugin not found"; always use the `/mnt/c/...` mount. [ev: retro 5rooms #7]

| `niagara_home` | niagara-module plugin (only one per install) |
|---|---|
| 4.13.2 | 7.3.40 |
| 4.14 | 7.6.17 |
| 4.15.3 | 7.6.22 |

There is no version common to all installs — this table supersedes the earlier "shared common set" claim. Always pass `-PniagaraPluginVersion` matching the chosen target.

## niagara_home on WSL
Building on WSL against `niagara_home=C:\...` breaks the m2 repo (see §Build target & plugin version); and when a `-ux` depends on `-rt`, the plugin's auto-copy of the built jar into a station-locked `modules/` fails. Build against a writable mirror instead — see §Building against a running station.

## Building against a running station: mirror
- **A running station LOCKS `<niagara_home>/modules/<mod>.jar` — never stop a live supervisor to build:** with `station`/`niagarad` running, the plugin's `clean`/`jar` fails deleting the deployed jar (`IOException: Unable to delete file …/modules/<mod>.jar`). Build against a read-only MIRROR: symlink every top-level entry + every `modules/*.jar` EXCEPT the one being built into a dir with a WRITABLE `modules/`, and build with `-Pniagara_home=<mirror>`. The jar lands in `build/libs` (deliverable) + `<mirror>/modules` (throwaway); the real install is never written. [ev: retro rt-hardening #8]
- Use `toolbelt/mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]`. Two guards protect you: it refuses when the mirror equals, sits inside, or contains the source (exit 20), and refuses to wipe an existing dir that lacks a `.niagara-mirror` marker (exit 20); it writes `.niagara-mirror` on creation.

## Verify
```
# THE gate — run on the built jars regardless of who built them:
toolbelt/verify-module.sh [--target-version X.Y] [--stored] [--src <module-dir>] <jar>...
# rows: PASS|FAIL|SKIP  <check>  <jar>  <detail> ; exit 0 pass · 1 fail · 2 usage · 3 env.
```
Default checks (conservative): every `.class` at bytecode major **52**, `META-INF/NIAGARA4.SF` present, every `module.xml` `<type>` resolves to a `.class`. Opt-in: `--target-version` (baja stamp), `--stored` (0 `Defl:` entries), `--src` (type-count vs `module-include.xml` + no raw-double facet).

What each check does by hand (the gate automates these — it checks EVERY class, not just the first):
```
# bytecode major 52 = Java 8 (65 = Java 21, wrong)
unzip -p MOD-rt/build/libs/MOD-rt.jar com/.../SomeClass.class | od -An -t d1 -j6 -N2   # 2nd number = 52
# signed
unzip -l .../MOD-ux.jar | grep NIAGARA4.SF
# no raw-double facet anywhere (--src)
grep -rnE 'BFacets\.make\(BFacets\.(MIN|MAX), *-?[0-9]' <module> --include='*.java'    # must be empty
# baja stamp (--target-version)
unzip -p <jar> META-INF/module.xml | grep baja
```

## Signing per deploy target
- **Check the deploy target's signing policy before assuming a Workbench re-sign:** a Honeywell supervisor ACCEPTS gradle's per-machine DEV cert — no re-sign needed (chihuahua's `deploy.sh` only builds + copies and runs on the same supervisor). A JACE field controller enforces the project CA (e.g. `angelessigner`), so a JACE-bound module IS re-signed. [CERT-live 2026-09-01 · retro 5rooms #9]

## Workbench re-sign: STORED repackage
- **Workbench `JarFileSigner` "invalid entry compressed size (expected N got M)" is a deflater mismatch (WSL OpenJDK 8 vs Windows Zulu 8), NOT a build-state fluke a clean rebuild fixes — it recurs:** the same WSL deflater re-derives the same size, so `clean + slotomatic + jar` does not help. The fix is to repackage the jar STORED (uncompressed) so the mismatch is impossible by construction. [ev: retro 5rooms #10]
- **Local `jarsigner` is a FALSE NEGATIVE for this defect:** WSL `jarsigner` re-deflates with the SAME deflater that built the jar and reports "jar verified" while Workbench still fails — never use it to claim Workbench-signability. Only an operator's live Workbench sign proves it. [CERT-live 2026-09-01 — operator signed both STORED jars with no ZipException]
- **STORED is a MANUAL post-build step, only for the Workbench re-sign path — the gradle `build/libs` jars stay deflated:** run `toolbelt/stored-repack.sh <in.jar> <out.jar>` (manifest first, then `.SF`/`.RSA`, then the rest, all STORED; it never overwrites). Since the supervisor accepts the DEV-signed deflated jar as-is, STORED matters only when a JACE forces a Workbench re-sign. Verify with `unzip -v` (0 `Defl:` entries) + `jarsigner -verify` ("jar verified"). [ev: retro 5rooms #10]

## Unit tests in WSL
- **`niagaraTest` does NOT run in WSL (needs the native `bin/test` + a dev license; `srcTest/` is dead weight there):** extract the control decision into a ZERO-Baja pure class and run it standalone — `javac -source 8 -target 8 -cp <junit> Pure.java PureTest.java`, then `java -cp out:<junit>:<hamcrest> org.junit.runner.JUnitCore <pkg>.PureTest`. This is the only runnable coverage of control/safety logic in WSL. [ev: retro rt-hardening #5]

## Known gap — ng-deploy.sh runs slotomatic for -rt only
- **`scripts/ng-deploy.sh` runs slotomatic for `-rt` only, not `-ux` — documented, not fixed:** when you edit `@NiagaraProperty` annotations in a `-ux` profile, build with `toolbelt/build.sh` (it runs slotomatic for every profile with sources) so the `-ux` annotation change is actually regenerated. [ev: QA audit · ng-deploy.sh]

## Deploy (station)
Stop station → replace the jars in `<niagara_home>/modules/` → start (the jar is locked while the station runs; to avoid stopping a live supervisor, build against a mirror per §Building against a running station and copy `build/libs` → station). Signing per target: §Signing per deploy target. **A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.**

See also: `docs/module-dev-workflow.md` (toolchain, codegen round-trip, dev loop, testing).
