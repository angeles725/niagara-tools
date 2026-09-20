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

### A5 · Missing class at station start: class-name/module-version mismatch
**Symptom:** `SEVERE sys.registry: Missing class for "Module:Type"` logged at station start
after a module deploy; affected components fail to instantiate or the station loads only
partially.
**Root cause:** the deployed jar carries a different class name or module version than the
`.bog` file that references it; the registry cannot resolve the type at runtime.  Distinct
from A3 (compile-only dep not bundled): here the class exists in the jar but the symbolic
identity does not match what the .bog stored.
**Fix:** after every deploy, grep the station log for `Missing class` before marking the
deploy complete.  A post-deploy gate (currently manual) should block commissioning until the
log is clean.
[ev: mem panccadia-station-audit-log] — **Kit coverage: none (post-deploy gate = PENDING)**

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

### B4 · Clock.schedule zero-delay: Math.max(0L, delay) floor at 0 is still rejected
**Symptom:** a self-firing timer silently never arms despite a `Math.max(0L, delay)` guard;
the component appears healthy (running, no fault) and logs nothing.
**Root cause:** the Niagara EngineManager rejects `time <= 0` the same as a negative delay;
`Math.max(0L, delay)` floors at 0, which the engine still rejects.  A naive zero-floor is
NOT equivalent to "validate delay > 0" (B2): the safe minimum is 1 ms, not 0.
**Fix:** replace `Math.max(0L, delay)` with `Math.max(1L, delay)` before every
`Clock.schedule` / `schedulePeriodically` call.  Enforce at the property level with a
`min >= 1` facet so the UI prevents zero entry.
[ev: mem coldroompan-defrost-time-le-0-bug] — **Kit coverage: METHODOLOGY.md S139 +
lint-delays.sh FAIL check (B2 sibling; FOLDED)**

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

### C3 · rt lifecycle seam (cancelRunTickets vs cancelTicket) invisible to pure JUnit
**Symptom:** outputs lock OFF (or ON) permanently after a refactor; all lint checks and pure
JUnit tests stay GREEN; the defect surfaces only on a live station.
**Root cause:** the choice between `cancelRunTickets()` (cancels ALL tickets) and
`cancelTicket(t)` (cancels ONE ticket) inside a lifecycle method (`stopped`, `atSteadyState`,
etc.) is not observable by static lint or by pure JUnit — there is no live NRE seam in WSL.
The incorrect cancel intent silently passes every offline check.
**Fix:** include a read-level review step that audits ticket-cancel intent against the
described lifecycle contract; supplement with a live-harness or BTestNg station check on
Windows.
[ev: mem coldroompan-defrost-time-le-0-bug] — **Kit coverage: none (gap; live harness check
recommended per retro C9-QA)**

### C4 · -ux modules missing Jasmine → silent browser-side test failures
**Symptom:** no JavaScript/Jasmine tests run for a `-ux` module; the build stays GREEN;
`niagaraTest` may also report 0 tests (see C1 for the Java-side plugin 7.6.17 bug).
**Root cause:** `-ux` module scaffolding does not add Jasmine as a devDependency by default,
so browser-side test suites are never discovered or executed.  The two issues are independent:
the missing Jasmine dep silences ux tests even when the plugin bug (C1) is resolved.
**Fix:** add Jasmine (and any required test runner) to the `-ux` module's `package.json` or
build config; see `types/dashboard.md` DUX-TEST1 for the required devDeps.  Also pin the
plugin away from 7.6.17 as described in C1.
[ev: corpus B1028] — **Kit coverage: types/dashboard.md DUX-TEST1; C1 covers the Java-side
plugin angle**

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

### D4 · `LocalSigningProfile` auto-generates a dev cert on missing alias → silent station load failure
**Symptom:** `gradlew build` succeeds and the jar is signed, but the station rejects the module at
load time with a `ValidationException` (BLD1 trust failure).  No build error.
**Root cause:** under the default `LocalSigningProfile` a missing or renamed signing alias triggers
`generateCert()` which auto-creates a self-signed dev cert — the build SUCCEEDS with a
signed-but-untrusted jar.  Under `RestrictedSigningProfile` (CI/release) the same condition throws
`IllegalArgumentException` and FAILS the build.  A missing profile FILE fails with
`ProfileNotFoundException` regardless of profile type.
**Fix:** before a release build, verify `niagara.signing.profileType` in
`~/.tridium/security/niagara.signing.xml` and confirm the intended alias exists in the keystore.
Treat "build signed OK but station rejects" as the tell-tale of a dev-profile auto-generated cert.
[ev: corpus B1135] — `[ev: retro module-hardening-reqexec-closed-deltas Δ3]` —
**Kit coverage: types/distribution.md §5 signing-profile build-triage (FOLDED)**

---

## E — Client / HMI compatibility

### E1 · EC-Net 4.3/Hx browser: BooleanWritable set(value) fails silently
**Symptom:** calling `set(value)` on a writable point in the EC-Net 4.3/Hx browser HMI
silently has no effect; the output value does not change and no error is reported.
**Root cause:** EC-Net 4.3/Hx browser incompatibility with the standard Niagara `set(value)`
call path for writable points.
**Fix:** model the control as a `BooleanWritable` HOA with states `active` / `inactive` /
`auto` (empty Override); drive the point through the `/set` dialog and use
"Paste Special → Keep all links" when wiring.  Do not rely on a direct `set(value)` call
from EC-Net 4.3/Hx.
[ev: mem harbor-greenmax-b851-pilot] — **Kit coverage: none**

---

## F — Security

### F1 · Destructive HTTP GET endpoints → CSRF / accidental data loss
**Symptom:** a configuration or dataset can be wiped by loading a URL (e.g., via a stray
browser pre-fetch, an `<img src="…/reset">` tag, or a link click) because a state-changing
operation is exposed over HTTP GET with no CSRF protection.
**Root cause:** state-changing operations (delete, reset, overwrite) exposed as GET endpoints
carry no CSRF protection; any same-origin `<img>`, link, or redirect can trigger them without
user intent.
**Fix:** require HTTP POST + a CSRF token for every endpoint that mutates state.  Audit all
`GET` routes in web-facing module servlets and move destructive actions to POST handlers.
[ev: mem nmodsreflow] — **Kit coverage: none (security audit = PENDING)**

---

## G — Boot and deployment recovery

### G1 · Station stuck at boot due to bad/wrong-version module → Platform Software Manager recovery
**Symptom:** station will not boot after a module deploy; log shows `App Failed` or a
`MissingClass` / `ValidationException` for the deployed module; the station cannot be started
from Workbench.
**Root cause:** the deployed module is incompatible — wrong version, unsigned/untrusted (BLD1/BLD7),
or a bad manifest (BLD5).  The station cannot load the module at boot; it halts.
**Fix:** recover at the PLATFORM level while the station is DOWN (platform daemon must be running):

1. Connect to the **Platform node** in Workbench (NOT the station node).
2. Open **Platform > Software Manager**.
3. Identify the offending module by status: "Bad Target" (bad manifest or unusable) or "Out of Date".
4. Choose a remediation action — Downgrade, Uninstall, Import+Re-Install, or
   **Rebuild Module Signatures** (repairs the trust failure; station must be not-running).
5. Click **Commit**; restart the station.

**Critical gotcha:** NEVER use the File Transfer Client to deploy or replace module JARs —
it copies raw bytes without applying runtime profiles or signing chains, producing the same
BLD1 trust failure on next boot even with a correct JAR.
[ev: corpus B1139] — `[ev: retro module-hardening-reqexec-closed-deltas Δ7]` —
**Kit coverage: types/distribution.md §10 (FOLDED)**
