# Type: moduleTest — BTestNg station-side integration tests

A `moduleTest` is a JVM integration test that spins up a REAL Niagara station in-process and exercises
component logic against it — the station-side complement to the pure-JUnit WSL tier.  It is the only
first-party way to instantiate and interact with `BComponent` subclasses from a test (a bare `new
BComponent()` requires a running NRE; see the offline-instantiation-boundary memory note).

Exemplar to read before writing: `componentLinks-rt/srcTest/…/BLinkCheckTestTest.java` (B958 §958.3–958.4).

> **NRE requirement:** `moduleTest` tests run on a native Windows Niagara install. From a WSL shell, run
> Gradle's `niagaraTest` through `cmd.exe` (§ Running from WSL); only a WSL-side
> Gradle with plugin 7.6.17 discovers 0 tests. Keep them as a separate QA tier — they verify inter-component
> link/station behaviour that pure JUnit cannot reach. `[ev: corpus B958 §958.3]` `[ev: retro testing-parity-btestng Δ1]`

---

## Lifecycle (`@BeforeClass` / `@Test` / `@AfterClass`)

```java
// srcTest/com/yourvendor/yourmod/test/BYourModuleTest.java
@NiagaraType                         // required — the type must be registered in moduleTest-include.xml
@Test(groups = { "yourmod" })        // TestNG group; choose a consistent module-scoped name
public class BYourModuleTest extends BTestNg {

    private TestStationHandler handler;
    private BStation           station;
    private BYourComponent     root;

    @BeforeClass
    public void setupBeforeClass() throws Exception {
        handler = createTestStation();          // from test-wb: boots an in-process NRE
        handler.startStation();
        station = handler.getStation();
        root = new BYourComponent();
        station.getSys().getRoot().add("root", root);
    }

    @AfterClass
    public void teardownAfterClass() throws Exception {
        handler.releaseStation();               // tears down the NRE; always in @AfterClass
    }

    @Test
    public void testSomeBehaviour() throws Exception {
        // arrange: set up component state on `root`
        // act: call an action or trigger a link
        // assert: use org.testng.Assert
        Assert.assertEquals(root.getSomeSlot().getDouble(), 1.0, 0.001);
    }
}
```

**Key points:**
- Extends `BTestNg` (from `Tridium:test-wb`), NOT `BTestCase` or JUnit `@Test`.
- TestNG annotations `@BeforeClass` / `@AfterClass` / `@Test` — NOT JUnit lifecycle annotations.
- `createTestStation()` + `handler.startStation()` boots a full in-process station; `releaseStation()` tears
  it down — call it in `@AfterClass` unconditionally.
- `@NiagaraType` on the test class is REQUIRED so the framework can instantiate it; the class is registered
  in `moduleTest-include.xml`, NOT in the production `module-include.xml`. `[ev: corpus B958 §958.3–958.4]`

---

## `moduleTest-include.xml` — separate from `module-include.xml`

The test type lives in a **separate descriptor** that feeds a separate jar (`moduleTestJar`).  It is NEVER
added to the production `module-include.xml` — that would ship test code in the production jar.

```xml
<!-- <part>/moduleTest-include.xml -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE moduleInclude PUBLIC
  'moduleInclude.dtd'
  'http://niagara.tridium.com/xml/dtd/moduleInclude/moduleInclude.dtd'>
<moduleInclude>
  <types>
    <type name="YourModuleTest"
          class="com.yourvendor.yourmod.test.BYourModuleTest"/>
  </types>
</moduleInclude>
```

**Separator rule:** `module-include.xml` → production types → shipped jar.  `moduleTest-include.xml` → test
types → `moduleTestJar` only.  A `moduleTestJar` is built by the Gradle plugin alongside the regular jar
when this file is present. `[ev: corpus B958 §958.4]`

**Plugin-version trap (gradle plugin 7.3.40, iC-Niagara 4.13, and 7.6.22, observed):** `:<part>:compileModuleTestJava` runs the
annotation processor against the production `module-include.xml` and appends every `@NiagaraType` class under
`srcTest/` to it. An ordinary `:<part>:jar` after that ships the test types in the production jar's
`module.xml`. Run `git status` after every moduleTest compile and revert any new `<type>` line in
`module-include.xml`. The check catches it: `lint-agent-on-include-sync.sh` sub-check `test-type-in-prod`
(run by `report-module.sh`) and the `moduletest` row of `verify-module.sh --src` FAIL a production `<type>`
whose class is declared under the part's `srcTest/` or listed in its `moduleTest-include.xml`.
`[ev: retro r10-lint-test-types-in-prod]`

---

## Gradle dependencies (`.gradle.kts`)

```kotlin
// <part>/<part>.gradle.kts (inside the runtimeProfile block)
dependencies {
    // --- production deps (as usual) ---
    api("Tridium:baja")
    api("Tridium:control-rt")            // whatever the module actually uses

    // --- test-only deps (go to moduleTestJar, NOT the shipping jar) ---
    moduleTestImplementation("Tridium:test-wb")       // provides BTestNg + TestStationHandler
    moduleTestImplementation("Tridium:bajaui-wb")     // often required by the test station
    moduleTestImplementation("Tridium:control-rt")    // any rt module the test exercises
}
```

**Rule:** every dep the test exercises but the production code does NOT use goes under
`moduleTestImplementation(…)`.  `test-wb` is always required.  `bajaui-wb` is needed whenever the test
station needs UI services (common in practice) `[INFER]`; the example declares it without a reason. `[ev: corpus B958 §958.4]`

---

## What to test in a moduleTest vs pure JUnit

| Concern | Pure JUnit (WSL gate) | moduleTest (station-side) |
|---|---|---|
| Control math / state machines | ✅ preferred | overkill |
| Link validation (`doCheckLink`) | ❌ needs NRE | ✅ |
| Link reactions (`added`/`removed`) | ❌ needs NRE | ✅ |
| Timer scheduling (use `Sched` DI) | ✅ preferred | rarely |
| Alarm routing (`BAlarmSourceExt`) | ❌ needs station | ✅ |
| oBIX read/write round-trip | ❌ | ✅ |
| Driver frame codec (encode/decode, bounds, tag) | ✅ preferred (Baja-free codec) | rarely |
| Driver comm stack vs a loopback mock (ping/read/write/timeout) | ❌ needs NRE | ✅ — `types/driver-authoring.md` § 12 |
| Slot-o-matic / type hash | check at build | rarely needed |

`[ev: corpus B958 §958.3]` (the link-validation row); the other rows `[INFER]`.

**Client parity doctrine — behaviour is tested through real slots** `[ev: retro testing-parity-btestng Δ4]`:
component behaviour (slot outputs, notify chains, HOA/defrost interplay, links) is a `BTestNgStation` test that drives
real slots and waits with `TestHelper.assertWillBeTrue` (§ TEST-G2); pure seams keep their WSL JUnit. A
source-structural test (reading a `.java` file as text) is the last resort, kept only where no station route exists,
and its header names the station test that replaces it or says why none can. Pilot: the ColdRoomPan-rt
`BTestNgStation` suite, 6 tests, five production mutations observed biting (§ Running from WSL). The advisory that flags
a structural test in a part with no `BTestNg` class is owed as Go lint G34 `structural-tests`.
`[ev: retro testing-parity-btestng Δ5]`

**Which base class** (course suite, corpus B1252 §1252.3.2; base classes in docSource 4.14.0.162 `test-wb`)
`[ev: retro n4-course-lexicon-logging-testing Δ3]`:
- `BTestNg` for value and slot logic, callbacks on a DETACHED component (no `started()`), a `do` method as a function.
- `BTestNgStation` for `started()`, service lookup, `JobService`, the alarm or history database, BQL/NEQL over the
  station, permissions. Its station is set up once per TestNG `<test>` run (`@BeforeTest setupStation()`,
  `javax/baja/test/BTestNgStation.java:93-94`), not per method: fixtures must be idempotent (remove-if-present, then
  add). Seed alarms with `getAlarmService().getAlarmDb().getDbConnection(null)` and `append`, histories through the
  history database's `getDbConnection(null)` (`createHistory`, `append`).
- Mock I/O at a static factory seam: `testUberjar("org.mockito:mockito-inline:4.11.0")` and `Mockito.mockStatic(...)`
  in try-with-resources (course deck 7 `7 Logging, Lexicons, and Testing.txt:436`; Lab 7 `BWorkOrderServiceTest.java:53`). To assert a FINE log line, lower the logger to `Level.ALL` in the test.
- Wait with `TestHelper.assertWillBeTrue`, never a fixed `Thread.sleep` (seven course tests sleep a fixed time;
  corpus B1252 §1252.3.3). Do not name a method `setup()`: it overrides `BTest.setup()` (`BTest.java:72`); use
  `@BeforeMethod`/`@AfterMethod` with distinct names. `[ev: retro testing-parity-btestng Δ6]`

**A BQL query needs no station:** build the tree in a plain `BFolder` and run
`BqlQuery.make(bql).resolve(OrdTarget.unmounted(folder)).get()`; give points values with `setOut(...)`, because
`set()` does not work on an unmounted component. [CERT-doc] devguide `bql.txt:338-357`; `OrdTarget.java:196,208`
(corpus B1308 §1308.1 row 4). `[ev: retro devguide415-nav-subscriptions-tags-ords Δ7]`

**A dedupe or helper extraction is tested at two levels** (`BUILD-LOOP.md` § 5): the new helper gets a pure JUnit
test, seen RED before any caller moves to it; each caller keeps a test of its observable output (servlet JSON,
rendered HTML, slot values) that passes unchanged before and after the move: a `BTestNgStation` test when the output
is slot behaviour, a golden or pure test otherwise. `[ev: retro modular-line-efficient-quality Δ4]`

**A client's WSL JUnit runner discovers its suites, never lists them by hand:** every `*Test.java` under its
`srcTest`/`test` roots, minus a commented exclude list with one reason per class (NRE kernel, live station,
`BTest`). An exclude that names no file, or zero discovered suites, fails the run. A suite no gate runs is not a
test: a hand-listed runner skipped four new suites and an always-failing probe, and one never-run suite had been
stale since its first commit. `[ev: retro r10-client-gate-retro Δ3]`

---

## Worked example — asserting link validation

From `componentLinks-rt` (B958 §958.3):

```java
@Test(enabled = true)
public void ngTestLinkCheck1() {
    root.removeAll();
    buildStation();
    // Link from ramp to linkCheckTest should succeed (doCheckLink returns makeValid())
    ramp.add("link", new BLink(linkCheckTest.getHandleOrd(), "p", "p", true));
    Assert.assertEquals(ramp.getLinks().length, 1);
}

@Test(enabled = true)
public void ngTestLinkCheck2() {
    root.removeAll();
    buildStation();
    // Link from sineWave to linkCheckTest should fail (BRamp required)
    sineWave.add("link", new BLink(linkCheckTest.getHandleOrd(), "p", "p", true));
    Assert.assertEquals(ramp.getLinks().length, 0);  // rejected → no link
}
```

`[ev: corpus B958 §958.3 :48-70]`

---

## Checklist before adding a moduleTest

A scaffolded module already has one. `toolbelt/scaffold-module.sh --type logic|dashboard` emits the station test
from `templates/station-test` into `<Mod>-rt`: `B<Mod>StationTestBase` (a `BTestNgStation` with Fox off, which
tolerates only the exact `Fox service did not start in time` failure while the station runs; unique per-test
mounts removed in `@AfterMethod`; `await` and `holds` helpers), `B<Mod>LinkTest` (for each slot in
`LINK_IN_SLOTS`: LinkCheck accepts a link from a plain source, the slot is not READONLY, and a BLink carries a
change to each double slot), `moduleTest-include.xml`, and both test dependencies in the gradle file. `--type
driver` ships its loopback test instead (`types/driver-authoring.md` § 12.1). Add every new link-in slot to
`LINK_IN_SLOTS`. `[ev: retro r10-scaffold-station-test Δ1]`

1. `srcTest/com/<vendor>/<mod>/test/BYourModuleTest.java` — extends `BTestNg`, `@NiagaraType`, TestNG
   annotations.
2. `<part>/moduleTest-include.xml` — registers the test type; does NOT appear in `module-include.xml`.
3. `.gradle.kts` — `moduleTestImplementation("Tridium:test-wb")` (and any other needed test deps).
4. Run the tests on a native (non-WSL) Niagara install with `bin/test` and a developer license:
   - **Inner loop:** Workbench → Tools → Run Tests, or the console runner `test <module>`, `test <module:Type>`,
     `test <module:Type.method>` for one module, class or method (the course instructor's choice).
   - **Gradle `niagaraTest`** (+ JaCoCo), as the course uses it (Lab 7, 9/9), on Windows; from WSL through
     `cmd.exe` (§ Running from WSL). This is the primary route; Workbench Run Tests is the fallback.
   `[ev: corpus B1418 §1418.5 f]` `[ev: corpus B807 §807.1]` `[ev: retro course-recordings Δ7]`

### Running from WSL `[ev: retro testing-parity-btestng Δ1]` `[ev: retro testing-parity-btestng Δ3]` `[ev: retro n4-course-handson-workflow Δ3]`

One command runs the module's station tests on a throwaway copy of the install and judges them
`[ev: retro r10-run-niagara-test-run Δ1]`:

```bash
toolbelt/run-niagara-test.sh run --install 'C:\Niagara\Niagara-4.x' --gradle-root <client gradle root> \
  --part ModuleA-rt --module ModuleA [--timeout 30m] [--keep-home] [--scratch '<dir under %TEMP%>'] \
  [--jdk8 /usr/lib/jvm/java-8-openjdk-amd64] [--gradle-user-home ~/.gradle] [--module-verification low] \
  [--exclude-jar '<glob>']... [--restore-gradle-root] [--log-dir <dir>] [--reuse-home '<kept dir>']
```

- It copies the install into a new dir under `%TEMP%` (robocopy: `bin etc jre lib security defaults`, then the
  top-level `modules\*.jar` without dotted-name strays), builds `:<part>:jar :<part>:moduleTestJar` in WSL against
  the copy (`-Pniagara_home=<copy>`, a scratch `niagara_user_home`, `user.home` in scratch for the throwaway signing
  profile, `-PniagaraPluginVersion` from the copy's `etc/m2`; `JAVA_HOME` and
  `-Porg.gradle.java.installations.paths` the WSL `--jdk8`, which overrides a client `gradle.properties` naming a
  Windows JDK; `GRADLE_USER_HOME` the caller's cache, so the gradle distribution is not downloaded per run), runs `bin\test.exe <module> -v:2 -output:` on it,
  and judges `testng-results.xml` with `verdict --testng` (stderr as the console log; a non-zero exit with green
  results is FAIL) `[ev: retro r10-rnt-run-live-fixes Δ4]`.
- It refuses (exit 2) an install under a user home, without `bin/test.exe`, or used by a running `station.exe`,
  `nre.exe`, `java.exe` or `wb.exe` (by command line or executable path), or with a `stations\*\config.bog.lock`
  another process holds open, and a probe that cannot tell refuses too `[ev: retro r10-run-niagara-test-run Δ2]` (a service's `station.exe`, whose path a non-elevated probe cannot read, is attributed
  through its `niagarad` service's `PathName`; one that cannot be attributed is refused `[ev: retro r10-rnt-run-live-fixes Δ3]`); a scratch outside `%TEMP%`; a build home inside the install. The install is never
  written: a change in its `modules/`, `etc/`, `security/` or top level is FAIL.
- It also refuses (exit 2, before any copy) a `LocalSigningProfile` named by `niagaraSigningProfile` (in
  `GRADLE_USER_HOME/gradle.properties`, else the gradle root's) whose `niagara.signing.validity` minus
  `niagara.signing.refresh` is below 32 days, and names both values. The signing plugin sets a new key's notAfter
  to day `validity` of the current month (`Calendar.set(DAY_OF_MONTH, validity)`, niagara-plugins 7.6.22
  `LocalSigningProfile.kt:388`) and re-mints the key when fewer than `refresh` days are left (`:216-225`). A
  throwaway 30/15 profile on 25 October therefore minted a 5-day key at every signing; the jars failed `cannot
  verify signature block file`, the NRE dropped them and test.exe printed only `invalid test target(s)`. Write a
  throwaway profile with the plugin defaults, 365/183, and no `niagara.signing.keypass.<alias>` entry (the plugin
  adds it; a pre-seeded one fails `:jar` with `Password for <alias> already present`).
  `[ev: retro r10-scaffold-station-test Δ5]`
- The scratch user home reaches the NRE through the environment: `bin/test.exe` runs straight from WSL with the
  copy as cwd, `niagara_user_home=<Windows path>` and `WSLENV` extended with `niagara_user_home/w` (the NRE reads
  `niagara_user_home` before it falls back to `%USERPROFILE%\Niagara<maj>.<min>\<brand>`). A `set
  "niagara_user_home=…"` inside `cmd /v:on /c` does not reach it: WSL re-quotes the inner quotes, and the NRE
  rebuilt the real user home instead `[ev: retro r10-rnt-run-live-fixes Δ2]`. WSL reports a Windows exit -1 as 255.
- The runner fails closed if the NRE writes the real user home: every `C:\Users\<you>\Niagara4.*\<brand>\registry\`
  and `security\` is compared before the copy and after cleanup, and a change there, or a scratch
  `niagara_user_home` without the `etc\` and `security\` the NRE creates, is FAIL `[ev: retro r10-rnt-run-live-fixes Δ1]`.
- Cleanup runs on every path: it kills this run's Windows processes by PID (`taskkill /PID <pid> /T /F`), confirms
  none is left (a WSL interop process that dies leaves its Windows process running, so only the PID kill bounds a
  timed-out run `[ev: retro r10-run-niagara-test-run Δ3]`), and deletes only the dir it created (kept with `--keep-home`). Exits: 0 PASS, 1 FAIL, 2 usage or
  refused, 3 environment (cmd.exe, git, robocopy, gradle, cleanup), 4 timeout (the run is killed).
- The first boot of a copy takes about 4.5 min (registry rebuild); later boots 26-80 s. The copy itself takes
  about 5 s; one batched boot of every class of a module took 7-10 min, so budget a mutation RED per boot
  accordingly `[ev: retro r10-station-test-campaign Δ3]`.
- **A client root that hardcodes the niagara gradle plugin version ignores `-PniagaraPluginVersion`** and cannot
  build against an install that ships another plugin version. Make the root read the property, with the old
  literal as its default, so its own builds are unchanged; ask before changing a client root
  `[ev: retro r10-station-test-campaign Δ2]`.
- **`--module-verification low` is TEST-ONLY, for the disposable copy; never use it on a commissioning or
  production `niagara_home`.** Off by default, `low` its only value (anything else is exit 2). 4.14 ships
  `niagara.moduleVerificationMode=low` in `defaults/system.properties` (warn only); 4.15 ships the line commented
  out, so the code default `medium` applies and a test jar whose cert chain cannot be validated fails `failed
  signature validation`, then `invalid test target(s)` with exit 0 (no `testng-results.xml`: FAIL). After robocopy
  the runner appends that one line, in the file's own line ending, to the copy's `defaults\system.properties`
  only (`PASS … module verification: low (copy only)`); a copy whose file resolves inside `--install` or outside
  the run's scratch dir is refused (exit 2). The install's file is hashed before the copy and after cleanup (`install
  defaults/system.properties unchanged (sha256 <8hex>)`; a change is FAIL). `low` still rejects a module that
  requests one of the six signature-required permission groups. Without the flag, a rejected signature adds a
  `hint:` row naming it `[ev: retro r10-rnt-module-verification Δ1]`.
- `--exclude-jar '<glob>'` (repeatable) leaves the matching top-level `modules\*.jar` out of the copy, so a stale
  jar of the module under test (an old 1.8 `moduleA-ux.jar` in the install) never sits next to the freshly built
  `moduleA-rt.jar`; the glob is a case-insensitive jar basename of `*` and `?` only, and one matching no jar is FAIL,
  exit 2, before anything is copied `[ev: retro r10-rnt-exclude-jar Δ1]`.
- **The build must not change the client's source tree.** `--gradle-root` must be a git worktree (else `gradle
  root is not a git worktree: cannot verify it`, exit 3, before any copy). The runner takes `git status
  --porcelain=v1 -z --untracked-files=all` of it, with a sha256 of every listed file (so a rewrite of a file that
  was already dirty counts), before the copy and again after cleanup; any new, modified, deleted or rewritten path
  is FAIL (`gradle root changed by the build: <paths>`, 10 names, then "and N more"), else `gradle root unchanged
  (<n> tracked changes before, same after)`. `--restore-gradle-root` (opt-in) then checks out (`git
  --literal-pathspecs checkout -- <paths>`) only the files that were clean before and that the build modified or
  deleted, never a file already dirty, never deletes an untracked file (it lists the ones the build created),
  re-reads the status and prints `gradle root restored: <paths>`; the run stays FAIL, since the restore is hygiene,
  not a pass. No `git clean`, `git reset` or `git checkout .` is ever run `[ev: retro r10-rnt-gradle-root-gate Δ1]`.
- **A failed run keeps its diagnostics; attach the log, not the home.** On exit 1, 3 or 4 (once the scratch dir
  exists) the runner reads, before cleanup deletes the dir, the last 200 lines of test.exe stderr and the last 50 of
  its stdout (each line cut at 4 KiB, a file read from at most its last 8 MiB) and lists the `-output:` dir (names
  and sizes, 200 files at most), then writes `<UTC timestamp>-<module>.log` (mode 0600) under `--log-dir`, default
  `$XDG_STATE_HOME/run-niagara-test`, else `~/.local/state/run-niagara-test` (made 0700), with the runner's own
  rows (the last 500) and the exit status, and prints `FAIL  run-niagara-test  diagnostics kept: <path>`. So a
  rejection such as `invalid test target(s)` stays explainable after a one-boot run. Before writing, the scratch dir
  becomes `<scratch>` and its home copy `<copy>` (Windows, forward-slash and `/mnt` forms, any case), and
  `%USERPROFILE%` and the WSL `$HOME` become `<you>`. Install paths stay as they are: they are the user's own and
  needed to diagnose, so check the log before sharing it outside the team. A green run writes no log unless
  `--log-dir` is given or `--keep-home` is on (`run log: <path>`); `--keep-home` keeps the dir as before and
  writes the log too. A run refused or failing before its scratch dir exists prints its reason only
  `[ev: retro r10-rnt-fail-diagnostics Δ1]`.
- **A failed run keeps testng-results.xml and every failure's whole trace beside its log.** The 200-line stderr
  tail can start inside a long stack trace, so on exit 1, 3 or 4 the runner also reads `testng-results.xml` before
  cleanup and writes, next to `<stem>.log`, `<stem>.testng-results.xml` (a copy) and `<stem>.traces.txt` (one
  `== FAIL <class>.<method> ==` section per FAIL test-method, configuration methods marked `(config)`, holding its
  `<full-stacktrace>` whole), both 0600 and redacted like the log. Each is cut at a line end under 4 MiB with a
  `(truncated: ...)` row, so a cut never splits a path the redactor must see whole. A missing or unreadable results
  file is a `testng-results.xml unavailable: <why>` row in the log's `kept evidence` section; a green run keeps
  neither file `[ev: retro r10-rnt-fail-evidence Δ1]`.
- **`--reuse-home '<kept dir>'` boots a second time on a home a `--keep-home` run kept**, so a test can prove what
  survives a station restart (data written in boot 1, read back in boot 2). It is the one flag that points the
  runner at an existing dir, so it is guarded: the dir must lie under the scratch parent, must not be, hold or sit
  in a real `%USERPROFILE%\Niagara4.*\<brand>` or have `registry\`/`security\` at its top (D3; a copied marker
  changes nothing), and must carry the `run-niagara-test.marker` every run writes in its dir (run id, install,
  sha256 of the install's `defaults\system.properties`, and the run log a `--keep-home` run wrote). The marker's
  run id must be the one that log, in `--log-dir`, records, and its install and sha256 this run's; each mismatch
  is exit 2 naming the field. Nothing is copied again: the home copy, its registry and the scratch
  `niagara_user_home` (station data) stay; `:jar`/`:moduleTestJar` rebuild the module jars into it and `out\`
  starts empty. `--exclude-jar` and `--module-verification` are exit 2 with it (the kept copy keeps boot 1's).
  The dir is deleted at the end (`cleanup: deleted <dir>`) unless `--keep-home` is given again. Proven on the fake
  runner only; a live two-boot reload on a client root is owed `[ev: retro r10-rnt-reuse-home Δ1]`.
- **Plugin-version trap: with niagara gradle plugin 7.3.40 (4.13) and 7.6.22 (observed), `:<part>:compileModuleTestJava` appends every
  `@NiagaraType` test class to the PRODUCTION `<part>/module-include.xml`** (never to `moduleTest-include.xml`), and
  it depends on `:jar`, so a test build silently edits a tracked source file that then ships the test types (a
  7.6.22 `moduleTestJar` build added 3 test types to the `-rt` and 2 to the `-ux` production file of a client root). Check
  `git status` of the gradle root after any moduleTest compile, by hand or through the runner's gate
  `[ev: retro r10-rnt-gradle-root-gate Δ2]`.

Fallback, by hand (the route the runner automates):

```bash
# Root already on a Windows drive: run in place (cmd.exe inherits the drive-letter cwd).
cd /mnt/c/<path> && cmd.exe /c "gradlew.bat \
  -PniagaraPluginVersion=<the plugin version in THIS niagara_home's etc/m2> :<part>:clean :<part>:niagaraTest"
# Root on ext4: copy it to %TEMP% first (cd /d rejects a \\wsl.localhost UNC path).
cmd.exe /c "cd /d C:\Users\<you>\AppData\Local\Temp\<copy> && gradlew.bat <same tasks>"
```

- **Always `:clean` first:** after a failed run `niagaraTest` is UP-TO-DATE and reports nothing.
- **Count the tests, not the exit code.** `BUILD SUCCESSFUL` with `Total tests run: 0` is a failed run. The two
  known causes: a WSL-side Gradle (plugin 7.6.17), and a test jar the test NRE cannot trust: the default
  self-signed `Niagara4Modules` dev key logs `Could not validate cert chain` → `invalid test target(s)` (on
  4.14.0.162 with a fresh user home the same WARN did not stop a throwaway-signed test jar from running, § 12.1 of
  `types/driver-authoring.md`; check the summary line, not the WARN). Sign with a
  CA-chained profile (`niagaraSigning { aliases; signingProfileFile }`, `build-verify.md` § Signing per deploy
  target) or import the signer's root into that NRE's user trust store; ask before changing a client root's
  signing block, since it also signs the shipped jars.
- **`:jar` writes into `niagara_home`:** it overwrites `<niagara_home>/modules/<part>.jar` and installs
  `<part>Test.jar`; on a reference install back up the first, restore it and remove the second afterwards, then
  delete any `%TEMP%` copy. Same route for `slotomatic`/`jar`: `build-verify.md` § Raw gradle.
- Proof: the ColdRoomPan-rt `BTestNgStation` pilot ran 6/6 this way on 4.15.3.28, and five production mutations
  each failed 1-3 tests (client commit 337ee6c, retro `testing-parity-btestng`).
- **Replace source-text tests with station tests, one mutation per boot.** A test that asserts on the Java source
  proves the text, not the behaviour. The round-10 campaign replaced six such client tests with `BTestNgStation`
  tests run by `run-niagara-test run` (11/11 on a 4.14 copy, 14/14 and 3/3 on 4.15 copies, a facade's links
  checked through a per-slot `checkLink` and real `BLink` propagation). Pin each test with a named production
  mutation, and boot once per mutation: RED on the class it targets and GREEN on every other class shows the RED
  is that class's and the classes are independent `[ev: retro r10-station-test-campaign Δ1]`.

See also: `types/structure.md §PASS state + scaffold` (scaffold emits a stub `moduleTest-include.xml`);
`types/logic.md §Link lifecycle — gating and reacting` (the `doCheckLink`/INDIRECT-link patterns
that moduleTests are best suited to verify). `[ev: corpus B958 §958.3–958.4, B961 §961.3]`

<!-- source: promoted from retro 2026-09-18-sdk-examples-kit-deltas [ev: retro sdk-examples-kit-deltas] -->

---

## TEST-G1 — `BTestNgStation`: full-services fixture

Use `BTestNgStation` (extends `BTestNg`) when the test needs a station with real services pre-wired.
Override `configureTestStation()` to add module-specific services on top of the defaults.

```java
@NiagaraType
@Test(groups = { "yourmod" })
public class BYourIntegrationTest extends BTestNgStation {

    @BeforeTest
    public void setupStation() throws Exception {
        // BTestNgStation.setupStation() starts the station and wires:
        //   RoleService, UserService (TestSuper/TestAdmin/TestOperator, pw Test@1234_5678),
        //   AlarmService, HistoryService, JobService, FoxService (port 1911)
        // Override isWebServiceEnabled() → true to also add WebService.
        super.setupStation();
    }

    @Override
    protected boolean isWebServiceEnabled() { return false; }   // default; set true when needed
}
```

**When to use:**
- Alarm routing, FOX round-trips, history writes, job scheduling — anything that requires real services.
- Use bare `createTestStation()` (from `BTestNg`) when you only need the NRE with no pre-configured services.

`[ev: code docSource BTestNgStation.java :93-129, :370-404, :760-763, :954-957, :985]`

---

## TEST-G2 — Async assertions (`TestHelper.waitFor` / `assertWillBeTrue`)

Never use `Thread.sleep` for async component/station behaviour.  Use:

```java
import javax.baja.test.TestHelper;

// Polls every 10 ms for up to 5 000 ms (defaults):
TestHelper.assertWillBeTrue(() -> myComp.getStatus().isOk(), "status should be OK");

// Custom timeout (ms) + message:
TestHelper.assertWillBeTrue(() -> myComp.getOut().getDouble() > 0, 10_000L, "output > 0 after 10 s");

// Raw boolean helper — returns true/false without asserting:
boolean settled = TestHelper.waitFor(() -> myComp.isRunning(), 3_000L, 50L);
```

Signature: `waitFor(BooleanSupplier condition, long timeout, long interval)`.
`assertWillBeTrue` delegates to `waitFor` and calls `Assert.assertTrue` on the result.

A single check of an output right after a synchronous action races the control cycle: either wait for the
expected value as above, or, to prove a value holds, keep sampling it over a window that covers at least one
cycle (a `holds`-style helper over `waitFor`). `[ev: retro r10-station-test-campaign Δ3]`

`[ev: code docSource TestHelper.java :75-197]`

---

## TEST-G3 — Parameterized tests (`@DataProvider` + `BTridiumTestNg`)

```java
import com.tridium.testng.BTridiumTestNg;
import org.testng.annotations.DataProvider;
import org.testng.annotations.Test;

@DataProvider(name = "inputs")
public Object[][] inputs() {
    // Simple: wrap an Iterable or varargs into Object[][]
    return BTridiumTestNg.toDataProviderArray(List.of("a", "b", "c"));
}

@Test(dataProvider = "inputs")
public void testWithParam(String value) throws Exception {
    Assert.assertNotNull(value);
}
```

For richer cases (named parameters, multiple columns), use the `DataProviderResults` builder:

```java
return StreamSupport.stream(cases.spliterator(), false)
    .collect(
        () -> new BTridiumTestNg.DataProviderResults(signature),
        (acc, t) -> processor.processInputValue(acc::add, acc::newCase, t),
        BTridiumTestNg.DataProviderResults::combine
    ).toArray();
```

`[ev: code docSource BTridiumTestNg.java :59-111, :228-482]`

---

## TEST-G4 — Flaky-test retry (`NRetryAnalyzer`)

```java
import com.tridium.testng.NRetryAnalyzer;
import org.testng.annotations.Test;

@Test(retryAnalyzer = NRetryAnalyzer.class)
public void sometimesFlaky() throws Exception { ... }
```

- Default: **3 attempts** (`MAX_ATTEMPTS = 3`).
- Override per-run: `-Dniagara.testng.retryAnalyzer.maxAttempts=N` on the JVM.
- Use sparingly — only for genuinely non-deterministic timing issues.  Fix the flakiness first; add the
  retry only when timing is inherent (e.g., network-dependent FOX handshake).

`[ev: code docSource NRetryAnalyzer.java :33-64]`

---

## TEST-G5 — OS-conditional tests (`@Requires`)

```java
import com.tridium.testng.annotation.Requires;

// Skip the test on non-Linux platforms:
@Requires(os = { Requires.OsType.LINUX })
@Test
public void linuxOnlyBehaviour() throws Exception { ... }

// Optional: a predicate method name (must return boolean, no args):
@Requires(predicate = "isPlatformReady")
@Test
public void conditionalTest() throws Exception { ... }

public boolean isPlatformReady() { return ...; }
```

`Requires.OsType` values: `WIN32`, `LINUX`, `QNX` (from the decompiled enum).  The listener
(`RequiresListener`) is wired into the TestNG suite by `test-wb`; no extra setup needed.

`[ev: code docSource RequiresListener.java :25-196, TestRunnerNg.java:341; organized/test/test-wb/vineflower/com/tridium/testng/annotation/Requires.java:11-19]`

---

## TEST-G6 — Out-of-process integration (`StationRunner`)

`StationRunner` launches a **real station in a separate JVM** and connects via FOX.  Use for file-system,
certificate, or session-level integration tests that cannot run in-process.  Do NOT use for routine CI —
the cost is high (separate process + FOX handshake).

```java
import com.tridium.testng.StationRunner;

StationRunner runner = StationRunner.make()
    .withBogFile(Path.of("src/test/resources/test-station.bog"))
    .start();   // blocks until "FOX server started on port [N]" appears in stdout

// connect via FOX, run tests…

runner.stop();
```

Use `StationRunner.make(debugPort)` to attach a debugger to the spawned JVM.

`[ev: code docSource StationRunner.java :90-102, :142-154, :179-272, :289, :399-400]`

---

## TEST-G7 — Palette assertions (`TestHelper.loadPaletteItem`)

Verify that a palette entry is present and instantiable without a running station:

```java
import javax.baja.test.TestHelper;
import javax.baja.sys.BModule;

@Test
public void paletteEntryPresent() throws Exception {
    BModule module = Sys.loadModule("yourvendor-yourmod");
    Object item = TestHelper.loadPaletteItem(module, "YourComponent");
    Assert.assertNotNull(item, "palette entry YourComponent must exist");
}
```

`loadPaletteItem(BModule, String...)` accepts a vararg path for nested palette folders.
Pair with the known gap: `-ux` modules lack Jasmine palette coverage (see R6 / DUX-TEST1).

`[ev: code docSource TestHelper.java :999]`

---

## TEST-G8 — Layout, registration, and known gotchas

### File layout

Two layouts exist. The 4.14+ wizard and course layout puts tests under a `test.` package root — the devkit-wb
4.14.0.162 test template declares `package test.${package};` (`organized/devkit/devkit-wb/extracted/gradle/videodriver/BNfooVideoDriverTest.java.vm:4`), and the
course suite lives in `srcTest/test/com/tridiumuniversity/slides/…` with `package test.com.tridiumuniversity.slides.<topic>`
and a class-level `@Test` (corpus B1252 §1252.3.1). New modules use it. The older layout below is what the kit showed before; whether 4.14+ tooling still discovers it was not verified `[INFER]`. Start the
registration file from `templates/moduleTest-include.xml`. `[ev: retro n4-course-lexicon-logging-testing Δ3]`

```
<part>/
  srcTest/
    test/com/<vendor>/<mod>/
      BYourModuleTest.java          # package test.com.<vendor>.<mod>; @NiagaraType + class-level @Test
  moduleTest-include.xml            # <type class="test.com.<vendor>.<mod>.BYourModuleTest" name="YourModuleTest"/>
```

Older layout:

```
<part>/
  srcTest/
    com/<vendor>/<mod>/test/
      BYourModuleTest.java          # @NiagaraType, extends BTestNg or BTestNgStation
  moduleTest-include.xml            # registers test types — NOT in module-include.xml
  <part>.gradle.kts                 # moduleTestImplementation(":test-wb")
```

### `moduleTest-include.xml` registration

Every test class needs `@NiagaraType` + a `TYPE` constant, and must be declared in
`moduleTest-include.xml` (separate from the production `module-include.xml`):

```xml
<moduleInclude>
  <types>
    <type name="YourModuleTest"
          class="com.yourvendor.yourmod.test.BYourModuleTest"/>
  </types>
</moduleInclude>
```

### Gradle dependency

```kotlin
moduleTestImplementation("Tridium:test-wb")
```

`bajaui-wb` is required when the test station needs UI services (common in practice; already noted in
the main Gradle section above).

### Known gotchas (R6)

- **`niagaraTest` from a WSL-side Gradle (plugin 7.6.17) discovers 0 `moduleTest` tests.** Run it on Windows
  (§ Running from WSL); a 0-test `BUILD SUCCESSFUL` is a failure, never a pass. `[ev: corpus B807 §807.1]`
- **`niagaraTest` can report BUILD SUCCESSFUL with zero tests or a rejected test jar.** Save the run's output
  (`test.exe <module> -v:2` or the gradle log) and judge it with `toolbelt/run-niagara-test.sh verdict <log>` (or
  `-` for stdin): PASS (exit 0) only when every `Total tests run: N` has N > 0 and 0 failures; FAIL (exit 1) on
  `invalid test target(s)`, `failed signature validation`, a stray failure count or no summary; never the run's
  exit code. `run-niagara-test.sh run` (§ Running from WSL) runs the test and applies this verdict itself.
  `[ev: retro testing-parity-btestng Δ2]`
  `[ev: retro r10-run-niagara-test-go]`
- **`test.exe <module> -v:2` prints its `Total tests run` summaries on stderr:** its stdout holds only the
  shutdown thread dumps, so a log of stdout alone has no summary (a console verdict on it is FAIL). Capture
  stderr (`2>&1`), and with `-output:<dir>` judge the authoritative `<dir>/testng-results.xml`, the console as
  a cross-check: `toolbelt/run-niagara-test.sh verdict --testng <dir>/testng-results.xml [<stderr log>]`.
  PASS only when the root has total > 0, failed = 0, skipped = 0, no `test-method` (config methods included:
  a failed `setupStation` skips its tests, which the root counts as skipped, not failed) has status FAIL
  and the root counts match the test-methods; a missing or malformed file is FAIL.
  `[ev: retro r10-run-niagara-test-testng Δ1]`
- **The root `total` counts ignored methods that never ran:** `@Test(enabled = false)` (or a group exclusion) adds
  to `ignored` and `total` only, with no `test-method` element, so a run of 43 with 20 disabled reads green once its
  2 failures are fixed. `verdict --testng` and `run` FAIL any ignored count unless `--allow-ignored-count <n>
  --allow-ignored-reason <why>` waives exactly n (the reason is printed); more ignored than n, fewer (a stale
  allowance), or a count with no reason (exit 2) is never a pass `[ev: retro r10-rnt-ignored-methods Δ1]`.
- **A counted allowance misses a swapped disable:** disable one new test and re-enable an old one, and the
  ignored count still matches. `run` names the ignored methods from the source: every method-level
  `@Test(enabled = false)` a type of `<part>/moduleTest-include.xml` declares, or inherits from a `srcTest/`
  superclass, is `<SimpleName>.<method>`; their count must equal the root `ignored`, and `--allow-ignored
  <Class.method>` (repeatable, with `--allow-ignored-reason`) waives exactly that set, a new or a stale name FAIL. A
  class-level disable, a non-literal `enabled` or an unreadable registered source is exit 3 before the build
  `[ev: retro r10-rnt-ignored-names Δ1]`.
- **The count fallback can hide the same swap:** a count that skips the source passes a swapped disable again. `run`
  scans the source under `--allow-ignored-count` too and prints the scanned names; the count is accepted only when
  the scan refused (case (i): a NOTE row, not exit 3) or the source count differs from the root `ignored` (case
  (ii): a group exclusion, a base outside `srcTest/`), and the row names the case. When the scan names every
  ignored method the count FAILs ("use --allow-ignored names"); the reason is a printed note and never overrides
  that. `verdict --testng` has no source and keeps the count pair `[ev: retro r10-rnt-fallback-strict Δ1]`.
- **A green run proves nothing for a test that cannot fail or never runs:** an `assertTrue(true)` placeholder, a
  test with no assertion, and a JUnit `assumeTrue(false)` (or a first-statement `throw new SkipException`) all
  pass or skip silently. `lint-vacuous-tests.sh <gradle-root>` FAILs each `@Test` method under `srcTest/` of these
  shapes; `expectedExceptions` / `expected =` and an assertion made through a helper of the same class, an
  enclosing class, a `srcTest/` superclass or a statically imported `srcTest/` class (one level deep) keep a test
  clean. Waive one only by name:
  `--allow <Class.method> --allow-reason <why>` (a stale name FAILs) `[ev: retro r10-lint-vacuous-tests Δ1]`.
- **`niagaraTest` is `bin/test`:** the task only execs `<niagara_home>/bin/test <target> [-groups:…] [-output:…]`
  (`RunNiagaraTestTask`, niagara-plugins 7.6.17), so `test.exe <module>` on a scratch COPY of the home (not a
  symlink mirror: Windows sees its links as dangling junctions) runs the same tests without gradle; set
  `niagara_user_home` to a fresh Windows-local dir (a `\\wsl.localhost` user home fails with a `FileLockException`
  on `security\.km`). Pass it to Windows with `WSLENV=niagara_user_home/w` (run `bin/test.exe` from WSL), never
  through `cmd /c "set …"`: WSL escapes the inner quotes, the variable never arrives, and the NRE writes the real
  user home. Verify: after the run the scratch home holds `security\` and `etc\`, and the real `Niagara4.x` homes
  are unchanged `[ev: retro r10-rnt-run-live-fixes Δ2]`. It exits 0 when every test passes and -1 (255 or 4294967295, by reader) when any fails;
  `cmd /c "… & echo %ERRORLEVEL%"` reads 0 either way, because `%ERRORLEVEL%` expands when the line is parsed,
    before test.exe runs: use `cmd /v:on /c "… & echo !ERRORLEVEL!"`. Judge the output anyway (§ above). Steps
  and timings: `types/driver-authoring.md` § 12.1.
  `[ev: retro r10-dr5-driver-exemplar Δ1]` `[ev: retro r10-run-niagara-test-testng Δ3]`
- **`BTestNgStation` needs its Fox service:** `startStation` takes the first `BFoxService` child, so
  `makeDefaultAuthFoxServices = false` fails setup with `ArrayIndexOutOfBoundsException: 0` and skips every test.
  Keep Fox on a free `foxPort` (the default 1911 is taken whenever a station or platform runs on the host).
  `[ev: retro r10-dr5-driver-exemplar Δ1]`
- **A station outlives a single test:** give each test its own component names (a name reused across tests
  collides with the previous test's component), and size every wait to the component's debounce plus one
  tick, not to a default timeout (about 20 s for one module). `[ev: retro r10-station-test-campaign Δ4]`
- **No test depends on the network or Wi-Fi:** a test that needs a reachable host fails for reasons the code does
  not control. Pure seams (`run-pure-test.sh`) or a station-local fixture replace it. `[ev: corpus B1418 §1418.5 g]` `[ev: retro course-recordings Δ7]`
- **Do not run `gradle -t` with IDE autosave:** continuous build compiles half-written files and reports
  failures that are not real. `[ev: corpus B1418 §1418.5 g]`
- **`-ux` modules lack Jasmine:** JS-side palette/component tests for dashboard UX modules must use the
  Jasmine harness documented in `types/dashboard.md §DUX-TEST1` — `moduleTest` covers the Java side only.

`[ev: corpus B1028 SDE7-G1; code BTestNgStation.java; code TestHelper.java]`

<!-- source: TEST-G1..TEST-G8 appended 2026-09-19 from odd/tasks/kit-improvement-candidates-2026-09-19.md; signatures verified against test-wb decompiled [ev: retro moduletest-patterns] -->
