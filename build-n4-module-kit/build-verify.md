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
- **A nested / multi-project gradle layout breaks the `<root>/<MOD>-<profile>` assumption — run `./gradlew :<MOD>-<profile>:jar` from the GRADLE ROOT, then verify `build/libs`:** e.g. DashboardPan's `gradlew` is at `Dashboard/` while the profiles are `Dashboard/DashboardPan/DashboardPan-{rt,ux,wb}` addressed as `:DashboardPan-ux:jar`. [ev: retro dashboardpan-ux-direct-build · B7]
- **Keep `src/rc/` asset-only — `processResources` copies ALL of `rc/` into the jar, so `index.html.bak` / `.orig` / scratch files ship as dead weight (~MB) AND are servable at `/<prefix>/index.html.bak`:** label-editing is preview-only, never committed; an orphan `img/plano.png` also ships when the real image is base64. [ev: retro dashboardpan-ux-direct-build · B4]

| `niagara_home` | niagara-module plugin (only one per install) |
|---|---|
| 4.13.2 | 7.3.40 |
| 4.14 | 7.6.17 |
| 4.15.3 | 7.6.22 |

There is no version common to all installs — this table supersedes the earlier "shared common set" claim. Always pass `-PniagaraPluginVersion` matching the chosen target.

## Module versioning & release
- **The MODULE's own version lives in `defaultModuleVersion("X.Y.Z")` in each module's `build.gradle.kts` `vendor{}` block → stamped as `vendorVersion` at build; a bump REQUIRES a rebuild to reach the jar.** Per-module version skew is fine (rebuild only the changed modules); the repo tracks `build/libs/*.jar`; tag `vX.Y.Z`. [ev: retro soft-start · S4]

## niagara_home on WSL
Building on WSL against `niagara_home=C:\...` breaks the m2 repo (see §Build target & plugin version); and when a `-ux` depends on `-rt`, the plugin's auto-copy of the built jar into a station-locked `modules/` fails. That copy failure is irrelevant — free the lock, or use the already-assembled `build/libs` jar (§Building against a running station: free the lock first, mirror only when you can't); the mirror is only for a live production supervisor.

## Building against a running station: free the lock first, mirror only when you can't
- **Free the lock FIRST — it's the common case:** an `Unable to delete …/modules/<mod>.jar` at `:clean` is a Workbench / your-own-station lock; close Workbench (or, when the holder is NOT a live production supervisor, stop it) and build directly against the install. This recurred 3× in one session and closing Workbench cleared it each time. Use the MIRROR (below) ONLY when the holder is a running station you must not stop, or a box you do not own — the mirror is the documented fallback, not the default. [ev: retro coldroompan-dashboardpan-freeze-stat-leds · B2]
- **A build against a RUNNING station reports FAILED at the `modules/` copy, but `build/libs/<jar>` is already fully assembled + signed:** the plugin auto-installs the jar as the last `jar` step; a locked `modules/` makes gradle exit FAILED, yet the artifact in `build/libs` passed the gate — verify THAT jar and use it. The copy failure is irrelevant when you are not deploying to that supervisor. [ev: retro dashboardpan-ux-direct-build · B1]
- **A brand-new module (not yet in `modules/`) needs NO station stop:** a first build has no deployed jar to overwrite, so `:clean :jar` builds green against a live install with the station running — only an UPDATE hits the lock. [ev: retro comppan-fase1 · B3]

### Mirror (the fallback for a live production supervisor)
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

## Pre-release real-jar smoke test
Before shipping a kit or toolbelt change, run `toolbelt/verify-module.sh` over at least one known-good and one known-bad real module jar — the bats suites use generated fixtures, which only simulate a malformed jar; a real one proves the gate catches a live defect. Worked known-bad: `ColdRoomPan-rt.jar` fails the `types` check because `module-include.xml` still declares `com.angeles.ColdRoomPan.BHoaMode` after that class was deleted — the live "Missing class ColdRoomPan:HoaMode" defect. DashboardPan's jars pass `--src` clean. Usage: see §Verify above.

## Signing per deploy target
- **Check the deploy target's signing policy before assuming a Workbench re-sign:** a Honeywell supervisor ACCEPTS gradle's per-machine DEV cert — no re-sign needed (chihuahua's `deploy.sh` only builds + copies and runs on the same supervisor). A JACE field controller enforces the project CA (e.g. `angelessigner`), so a JACE-bound module IS re-signed. [CERT-live 2026-09-01 · retro 5rooms #9]

## Workbench re-sign: STORED repackage
- **Workbench `JarFileSigner` "invalid entry compressed size (expected N got M)" is a deflater mismatch (WSL OpenJDK 8 vs Windows Zulu 8), NOT a build-state fluke a clean rebuild fixes — it recurs:** the same WSL deflater re-derives the same size, so `clean + slotomatic + jar` does not help. The fix is to repackage the jar STORED (uncompressed) so the mismatch is impossible by construction. [ev: retro 5rooms #10]
- **Local `jarsigner` is a FALSE NEGATIVE for this defect:** WSL `jarsigner` re-deflates with the SAME deflater that built the jar and reports "jar verified" while Workbench still fails — never use it to claim Workbench-signability. Only an operator's live Workbench sign proves it. [CERT-live 2026-09-01 — operator signed both STORED jars with no ZipException]
- **STORED is a MANUAL post-build step, only for the Workbench re-sign path — the gradle `build/libs` jars stay deflated:** run `toolbelt/stored-repack.sh <in.jar> <out.jar>` (manifest first, then `.SF`/`.RSA`, then the rest, all STORED; it never overwrites). Since the supervisor accepts the DEV-signed deflated jar as-is, STORED matters only when a JACE forces a Workbench re-sign. Verify with `unzip -v` (0 `Defl:` entries) + `jarsigner -verify` ("jar verified"). [ev: retro 5rooms #10]

## Unit tests in WSL
- **`niagaraTest` is DOCUMENTATION, not a WSL gate (needs the native `bin/test` + a dev license, and plugin 7.6.17 discovers 0 tests from WSL; `srcTest/` is dead weight there):** extract the control decision into a ZERO-Baja pure class and run it standalone. Resolve the cached jars with `JU=$(find ~/.gradle -name 'junit-4.13.2.jar' | head -1)` and `HC=$(find ~/.gradle -name 'hamcrest-core-1.3.jar' | head -1)` (never `find /` — it times out); the test must be same-package as the package-private pure class; use a fresh OUT dir. `toolbelt/run-pure-test.sh <rt-dir> <pkg> <PureClass> <TestClass>` wraps this (exit 3 if the cache is empty — run one gradle build to fetch the jars). This is the only runnable coverage of control/safety logic in WSL. [ev: retro rt-hardening #5; junit-standalone · B11]

## How you know it's good — the 4-layer assurance stack
For any decision/safety logic, "done" means all four layers ran, in order:
1. **Pure JUnit** (`toolbelt/run-pure-test.sh`) — the ONLY executable coverage of control/safety logic in WSL; mandatory. Extract inline logic to a pure class first (see `types/logic.md` §Pure-class extraction).
2. **The verify gate** (`toolbelt/verify-module.sh`) — bytecode 52, signed, types resolve.
3. **A live cold-boot smoke** — for a timer-based control, read the live anchor slot (oBIX / Slot Sheet) after boot: anchor populated = the hook armed; anchor null in an arming mode = the hook did NOT run. Correct source ≠ correct behavior; this catches what a pure test cannot. [ev: retro hidden-actions · T4]
4. **Adversarial pure-logic review BEFORE compile** — a second reader of the pure class catches control-logic defects the gate cannot (the `Long.MIN_VALUE` overflow and the dead-sensor false-fault both compiled and passed the gate). [ev: retro comppan-fase1 · T3]

## Known gap — mode B ignores `--with-slotomatic`
- **Modes A and C regenerate `-ux` slots when `-ux` is annotated** (`scripts/ng-deploy.sh` lines ~493-500: `run_slotomatic` is called for every profile with sources); **mode B never runs slotomatic** (lines ~552-553). So a `-ux` annotation edit under mode B silently skips regeneration — build with `toolbelt/build.sh` (runs slotomatic for every profile) or with `--mode A --with-slotomatic` instead. The blanket "-rt only" wording was stale for modes A/C, correct for mode B only. [ev: QA audit · ng-deploy.sh]

## Verify — useful patterns
- **V1 / Consumer-absence delta proof:** to prove "what changed since X" when same-day timestamps don't discriminate, `grep -c <symbol>` the CONSUMER artifact — 0 hits = genuine delta; present = already consumed. Beats pinning a commit/deploy boundary by date. [ev: retro dashboardpan-2d-to-3d-port Δ1]
- **V2 / Live-vs-doc precedence:** for behavior the live system arbitrates (e.g. is a setpoint writable?), verify live and let it override the doc — then fix the doc. A PORT-SPEC said "setpoint writable"; live oBIX `PUT` returned 400; the spec was wrong. [ev: retro live-cutover-and-authenticated-control Δ2]
- **V3 / Verify freshness before labeling "live":** probe max `ts` is advancing ≈ now; if frozen, label honestly (SNAPSHOT / última lectura `<ts>`). [ev: retro live-cutover-and-authenticated-control Δ1]
- **V4 / Headless-QA/CORS boundary:** a headless-from-localhost e2e cannot cross a browser CORS origin by design — confirm the backend with `curl` and verify `Access-Control-Allow-Origin` separately; a CORS block is not a code bug. [ev: retro live-cutover-and-authenticated-control Δ3]

## Deploy (station)
Stop station → replace the jars in `<niagara_home>/modules/` → start (the jar is locked while the station runs; to avoid stopping a live supervisor, build against a mirror per §Building against a running station and copy `build/libs` → station). Signing per target: §Signing per deploy target. **A jar that has not passed `toolbelt/verify-module.sh` does not go to a station.**

See also: `docs/module-dev-workflow.md` (toolchain, codegen round-trip, dev loop, testing).
