# Type: moduleTest — BTestNg station-side integration tests

A `moduleTest` is a JVM integration test that spins up a REAL Niagara station in-process and exercises
component logic against it — the station-side complement to the pure-JUnit WSL tier.  It is the only
first-party way to instantiate and interact with `BComponent` subclasses from a test (a bare `new
BComponent()` requires a running NRE; see the offline-instantiation-boundary memory note).

Exemplar to read before writing: `componentLinks-rt/srcTest/…/BLinkCheckTestTest.java` (B958 §958.3–958.4).

> **NRE requirement:** `moduleTest` tests do NOT pass the WSL gate (`niagaraTest` task has a known
> plugin bug in sdk 7.6.17; run from Workbench → **Tools → Run Tests** until the task is fixed).
> Keep them as a separate QA tier — they verify inter-component link/station behaviour that pure JUnit
> cannot reach. `[ev: corpus B958 §958.3]`

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
station needs UI services (common in practice). `[ev: corpus B958 §958.4]`

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
| Slot-o-matic / type hash | check at build | rarely needed |

`[ev: corpus B958 §958.3]`

---

## Worked example — asserting link validation

From `componentLinks-rt` (B958 §958.3):

```java
@Test
public void ngTestLinkCheck1() throws Exception {
    // A BRamp→BLinkCheckTest link MUST succeed (doCheckLink returns makeValid())
    station.getDriver().link(ramp, BRamp.output, root, BLinkCheckTest.rampPoint);
    Assert.assertEquals(ramp.getLinks().length, 1);
}

@Test
public void ngTestLinkCheck2() throws Exception {
    // A BSineWave→BLinkCheckTest link MUST be rejected (BRamp required)
    station.getDriver().link(sineWave, BSineWave.output, root, BLinkCheckTest.rampPoint);
    Assert.assertEquals(ramp.getLinks().length, 0);  // rejected → no link
}
```

`[ev: corpus B958 §958.3 :48-70]`

---

## Checklist before adding a moduleTest

1. `srcTest/com/<vendor>/<mod>/test/BYourModuleTest.java` — extends `BTestNg`, `@NiagaraType`, TestNG
   annotations.
2. `<part>/moduleTest-include.xml` — registers the test type; does NOT appear in `module-include.xml`.
3. `.gradle.kts` — `moduleTestImplementation("Tridium:test-wb")` (and any other needed test deps).
4. Test verifies in Workbench → Tools → Run Tests (WSL `niagaraTest` task is blocked by sdk bug 7.6.17).

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

`[ev: code BTestNgStation.java :96-270]`

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

`[ev: code TestHelper.java :97-145]`

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

`[ev: code BTridiumTestNg.java :49-233]`

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

`[ev: code NRetryAnalyzer.java :30-40]`

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

`Requires.OsType` values: `LINUX`, `WINDOWS`, `MAC` (from the decompiled enum).  The listener
(`RequiresListener`) is wired into the TestNG suite by `test-wb`; no extra setup needed.

`[ev: code RequiresListener.java :10-111, annotation/Requires.java :14-20]`

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

`[ev: code StationRunner.java :66-164]`

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

`[ev: code TestHelper.java :494]`

---

## TEST-G8 — Layout, registration, and known gotchas

### File layout

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

- **`niagaraTest` task bug (sdk 7.6.17):** the Gradle `niagaraTest` task has a known plugin bug and cannot
  run `moduleTest` suites.  Run tests from Workbench → **Tools → Run Tests** until the task is fixed.
- **`-ux` modules lack Jasmine:** JS-side palette/component tests for dashboard UX modules must use the
  Jasmine harness documented in `types/dashboard.md §DUX-TEST1` — `moduleTest` covers the Java side only.

`[ev: corpus B1028 SDE7-G1; code BTestNgStation.java, TestHelper.java]`

<!-- source: TEST-G1..TEST-G8 appended 2026-09-19 from odd/tasks/kit-improvement-candidates-2026-09-19.md; signatures verified against test-wb decompiled [ev: retro 2026-09-19-test-wb-extension] -->
