# Known issues & gotchas — N4 module development

Cross-cutting traps a builder MUST know.  Format per entry:
**Symptom** / **Root cause** / **Fix/avoidance** / Evidence + kit coverage.

---

## A — Schema & upgrade safety

### A1 · Slot retype → station fails to boot (OUTAGE)
**Symptom:** station refuses to load (`App Failed`, `Cannot load station`, `ClassCastException`
in `ValueDocDecoder`) after deploying a module update.
**Root cause:** the `.bog` file binds each slot by Java type; if the on-disk stored type no
longer matches the class, `ValueDocDecoder` cannot decode → OUTAGE.
**Fix:** NEVER retype a live slot.  ADD a new slot with the new type; optionally migrate
the value in `started()` by detecting the orphaned dynamic slot name; deprecate the old slot.
[ev: corpus B739 §739.2] — **Kit coverage: METHODOLOGY.md S28 + schema-risk.sh (FOLDED)**

### A2 · Remove/rename stored enum tag → OUTAGE
**Symptom:** station fails to boot or throws `InvalidEnumException` on a `.bog` with a stored
enum value whose tag was removed.
**Root cause:** `.bog` stores enum values by tag name; the `getRange().get(staleTag)` path throws
`InvalidEnumException` unwrapped.  `@Range` enum-tag changes are NOT auto-detected by
`schema-risk.sh parse_slots()` — it harvests only `@NiagaraProperty` entries.
**Fix:** ADD enum tags, never remove or rename a tag deployed to a live station.  Manually diff
`@Range({...})` declarations before a schema-change deploy.
[ev: corpus B754 §754.6] — **Kit coverage: METHODOLOGY.md S29 + schema-risk.sh; @Range blind spot
= schema-versioning-upgrade-safety-deltas Δ4 (PENDING)**

### A3 · Cross-module @NiagaraProperty type= → Missing class at station load
**Symptom:** `Missing class <module>:<EnumType>` at station load; `verify-module.sh` and
`gradlew build` are both GREEN.
**Root cause:** A `compileOnly(files(...))` dep makes the foreign class visible to `javac`
but NOT to the Niagara gradle plugin classpath; the type is not bundled in the module jar;
at load time the station cannot resolve it.  [ev: corpus B740 §740.2]
**Fix:** Use a plain `double` (or a local re-declaration of the enum) for cross-module values.
If you must share the type, declare it as an `nre()` runtime dep (not `compileOnly`).
**Proposed check:** `check_cross_module_type` WARN in verify-module.sh (schema-versioning Δ5 PENDING)
[ev: corpus B740] — **Kit coverage: schema-versioning-upgrade-safety-deltas Δ5 (PENDING)**

### A4 · Renumbering enum ordinals is safe for .bog but breaks Fox/binary links
**Symptom:** a supervisor reads the wrong enum tag from a JACE point over Fox; no error logged.
**Root cause:** `.bog` is tag-based (safe); Fox protocol encodes enum values as integer ordinals;
renumbered ordinals decode to the wrong tag silently on the remote side.
**Fix:** Never renumber enum ordinals on a value that crosses a Fox link.  The `schema-risk.sh`
CSV entry `renumber_enum_ordinals,SAFE` is technically correct for .bog but hides the Fox hazard.
**Proposed check:** `schema-risk.sh` WARN row for this kind (schema-versioning Δ6 PENDING)
[ev: corpus B754 §754.6] — **Kit coverage: schema-versioning-upgrade-safety-deltas Δ6 (PENDING)**

---

## B — Timer / clock lifecycle

### B1 · Self-armed timer only in atSteadyState() → silently never fires at commissioning
**Symptom:** a periodic timer-driven action (defrost cycle, compressor rotation, poll)
never executes when the component is mounted onto a station that is already running
(e.g., drag-drop during commissioning or after a component enable).  No error is logged.
**Root cause:** `atSteadyState()` fires ONCE during station bootstrap, after the steady-state
timeout (default 10 s).  A component mounted AFTER the station reaches steady-state receives
`started()` but NOT `atSteadyState()`.  Arming only in `atSteadyState()` leaves the ticket
permanently un-armed for late-mounted instances.
**Fix:** Override BOTH hooks with the Tridium canonical idiom:
```
  atSteadyState(){ if(isRunning()) arm(); }
  started(){ super.started(); if(Sys.atSteadyState()) arm(); }
  clockChanged(BRelTime shift){ try { arm(); } catch(Throwable t){ … } }  // optional, cheap
```
The `started()` + `Sys.atSteadyState()` guard fires exactly-once across both paths and avoids
the NotRunningException race on boot.  Missing `clockChanged` leaves a stale absolute
next-fire target after an NTP/DST shift.
**Live case:** BDefrostController (ColdRoomPan-rt), PANCCADIA León, defrost never fired.
[ev: corpus B729 §729.4, retro self-firing-timer] — **Kit coverage: types/logic.md §17 (FOLDED)**
**Lint check:** `lint-timers.sh atSteadyState-only-timer` WARN — detects a class that arms in
`atSteadyState()` without a `started()` override.
[ev: retro 2026-09-18-insights-issues-catalog-deltas Δ2]

### B2 · Clock.schedule/schedulePeriodically with delay ≤ 0 → IllegalArgumentException
**Symptom:** a timer silently fails to arm; the action never fires; no startup error visible
unless debug logging is enabled.  The component appears healthy (running, no fault).
**Root cause:** `Clock.java:223` — `schedulePeriodically` throws `IllegalArgumentException`
if `period <= 0`.  A user config value of 0 (default before commissioning) reaches this code.
**Fix:** validate `delay > 0` before every `Clock.schedule`/`schedulePeriodically` call;
set the property's `min` facet > 0 so the UI enforces the constraint.
[ev: corpus B801] — **Kit coverage: METHODOLOGY.md S139 + lint-delays.sh FAIL check (FOLDED)**

### B3 · State/anchor slots must be set BEFORE calling Clock.schedule
**Symptom:** after a schedule that throws (e.g., `NotRunningException` in `activateLinks`),
the display anchor slot (e.g., `nextDefrostTime`) remains null/uninitialized permanently.
**Root cause:** if the `Clock.schedule(...)` line throws, subsequent lines in the same method
(including `setNextX(...)`) never execute, leaving UI state orphaned.
**Fix:** write all state and anchor slots BEFORE calling `Clock.schedule(...)`.  Guard any
`Clock.schedule` that is reachable from `activateLinks`/`changed` with
`if(!Sys.atSteadyState()) return;` to prevent early scheduling on boot.
[ev: corpus B729 §729.4, retro self-firing-timer item 3] — **Kit coverage: retro self-firing-timer
(FOLDED); not yet a lint check**

---

## C — Test infrastructure

### C1 · Plugin 7.6.17 moduleTestAnnotationProcessor bug → niagaraTest 0 tests
**Symptom:** `gradlew niagaraTest` completes successfully with `Total tests run: 0`; no
diagnostic message explains why.
**Root cause:** `com.tridium.niagara-module` plugin 7.6.17 — the `moduleTestAnnotationProcessor`
does not generate `moduleTest-include.xml` for `@NiagaraType` test classes in srcTest.
`writeTestModuleXml` task finds no input and produces no output; `niagaraTest` discovers zero types.
**Fix (current workaround):** use pure-JUnit tests (no Baja types, no BTestNg) as the WSL gate.
These run via standalone `javac + junit-4.13.2.jar`, not via `niagaraTest`.  When Tridium
ships a fix: add `BTestNg`-extending class + BAJA AUTO region + `testModuleManifest {}` block
to srcTest; verify `moduleTest-include.xml` is produced; check `niagaraTest` count > 0 on Windows.
**Note:** even without the bug, `niagaraTest` ONLY runs BTestNg/TestNG tests discovered via the XML.
Pure `org.junit.Test` classes are never run by `niagaraTest` regardless of plugin version.
[ev: corpus B807, B1028 §1028.2] — **Kit coverage: retro qa-stack (FOLDED); proposed verify-module.sh
WARN for missing moduleTest-include.xml = sdk-examples-kit-deltas Δ1 (PENDING)**

### C2 · BComponent cannot be constructed in plain JUnit / WSL (offline instantiation boundary)
**Symptom:** `NullPointerException` or `ExceptionInInitializerError: Sys not initialized`
when constructing a `BComponent` subclass in a plain JUnit test or WSL environment.
**Root cause:** `BComponent` construction requires `Sys.init()` + the NRE kernel; `BAbsTime.make()`,
`Sys.loadType()`, `BComponent.getSlotValue()`, and all Baja slot machinery are non-functional
without the full runtime environment.
**Fix:** extract all logic that can be unit-tested into Baja-free static helper methods (no imports
from `javax.baja.*`).  Test those with plain `org.junit.Test` in WSL.  For lifecycle and
link-graph tests, use BTestNg + a test station (Windows only, requires niagaraTest fix).
[ev: corpus B741, B1028 §1028.5, memory baja-offline-instantiation-boundary] —
**Kit coverage: retro qa-stack (FOLDED)**

---

## D — Build & tooling false positives

### D1 · report-module.sh plano FAIL false positive on a 3D / non-plano SPA
**Symptom:** `report-module.sh` exits non-zero with `plano Rc: IMG_W/IMG_H not found`; the
module is a 3D three.js SPA with no plano overlay.
**Root cause:** `report-module.sh` auto-runs the plano check on any -ux profile; the check is
only meaningful for 2D plano-overlay dashboards.
**Fix/avoidance:** read the module's UI kind before treating the FAIL as a defect; it is a false
positive for any 3D or non-plano SPA.  Pending: make `--plano` opt-in only.
[ev: corpus B1015 §1015.5] — **Kit coverage: umbrelladashboard-module-creation Δ1 (PENDING)**

### D2 · verify-module.sh phantom-dep false WARN on Gradle project-dependency form
**Symptom:** `verify-module.sh` emits `WARN phantom-dep` for a dep declared as
`api(project(":MOD-rt"))` in `build.gradle.kts`.
**Root cause:** the phantom-dep check recognizes only string-form deps (`api(":x")`,
`nre(":x")`); Gradle project-dependency form is not parsed.
**Fix/avoidance:** this is a known false positive; the dep is legitimate.  Pending: fix
phantom-dep to recognize `project(":")` form.
[ev: corpus B1015 §1015.4] — **Kit coverage: umbrelladashboard-module-creation Δ2 (PENDING)**

### D3 · Gradle plugin version 7.6.17 is coupled to the N4.14 SDK family
**Symptom:** build fails to resolve the plugin (`Plugin [id: 'com.tridium.niagara-module',
version: '7.6.17'] was not found in any of the following sources`) when `niagara_home`
points to a different SDK version.
**Root cause:** the `pluginManagement` block in `settings.gradle.kts` pins plugin 7.6.17; the
plugins resolve from `niagara_home/etc/m2`; mixing SDK families creates a version mismatch.
**Fix:** keep `gradlePluginVersion` in `settings.gradle.kts` aligned with the SDK family in
`gradle.properties niagara_home`.  Changing the SDK requires a matching plugin version change.
[ev: corpus B1016 §1016.2] — **Kit coverage: n4-client-build-config-standard Δ2 (PENDING)**
