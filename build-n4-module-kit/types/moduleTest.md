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
