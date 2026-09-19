<!-- review-status: pending -->
# 2026-09-18 · kit · insights-issues-catalog-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — insights/issues catalog).
**Delta count**: 3

---

## What happened

Mined the niagara-research corpus for cross-cutting KNOWN ISSUES / gotchas / insights
that a module builder must know.  Sources: B729, B739, B740, B741, B754, B801, B807, B1015,
B1016, B1028, and the in-situ evidence from `baja-offline-instantiation-boundary` (memory)
+ `ColdRoomPan-defrost-time-le-0-bug` (memory).

The kit has **no consolidated issues-and-gotchas catalog** today.  Individual issues are
scattered across:
- `METHODOLOGY.md` (inline bullets at lines 28-31, 139, K15)
- `types/logic.md` (timer lifecycle §17)
- `types/structure.md` (L9, L10)
- individual retros (qa-stack, self-firing-timer, slot-type-change, …)
- toolbelt scripts (schema-risk.sh, lint-delays.sh, lint-structure.sh, lint-timers.sh)

A single `types/issues-and-gotchas.md` would let a builder scan all known traps in one
pass before starting any module work.

Dedup scan confirmed: all issues below have individual coverage somewhere in the kit;
the CATALOG FILE itself is net-new.  One additional genuinely new check (Δ2, `atSteadyState`-only
timer arming) has no existing retro or lint.

---

## Evidence

**Schema & upgrade safety**
- B739 §739.2 — `ValueDocDecoder` parse-desync on slot retype: `ClassCastException` / station
  boot failure, live PANCCADIA incident.  `[ev: corpus B739]`
- B754 §754.6 — remove/rename stored enum tag →  `InvalidEnumException` at decode, OUTAGE.
  `[ev: corpus B754]`
- B740 §740.2-3 — `@NiagaraProperty type=` pointing at a sibling custom-module enum compiles with
  `compileOnly(files(...))` (NOT on plugin classpath), passes verify gate, fails at station load as
  `Missing class <mod>:<Type>`.  Live `HoaMode` incident.  `[ev: corpus B740]`
- B754 §754.6 + schema-risk.sh CSV — `renumber_enum_ordinals` marked SAFE for `.bog`
  (tag-based decode) but the CSV note `fox-sync-unsafe` emits no WARN row; Fox encodes enum as
  integer ordinal, consumer decodes the wrong tag silently.  `[ev: corpus B754]`
- B754 §754.3 — `BModule` is `final`; no `upgrade()`/`migrate()` callback exists; the only
  self-migration path is detecting the orphaned dynamic slot by name in `started()`.
  `[ev: corpus B754]`

**Timer / clock lifecycle**
- B729 §729.4 + corpus evidence — `atSteadyState()` fires only during station bootstrap; a
  component mounted on an already-running station (commissioning) receives `started()` but
  NOT `atSteadyState()`.  A timer armed ONLY in `atSteadyState()` is silently never armed for
  late-mounted instances.  Live: `BDefrostController` on PANCCADIA León, defrost never fired.
  `[ev: corpus B729]`
- B729 §729.3 / B801 — `Clock.schedule`/`schedulePeriodically` with delay ≤ 0 throws
  `IllegalArgumentException` at runtime, silently killing the timer.  `lint-delays.sh` catches
  this statically.  `[ev: corpus B801]`
- B729 §729.2 — Tridium's canonical `BTimeTrigger` overrides THREE hooks: `started()`,
  `atSteadyState()`, AND `clockChanged(BRelTime)`.  Missing `clockChanged` leaves an absolute
  next-fire target stale after an NTP/DST wall-clock shift.  `[ev: corpus B729]`

**Test infrastructure**
- B807 / B1028 §1028.2 — Plugin 7.6.17 `moduleTestAnnotationProcessor` never produces
  `moduleTest-include.xml` for `@NiagaraType` test classes in srcTest.  `niagaraTest` runs
  0 tests silently.  `[ev: corpus B807, B1028]`
- B807 — Even with the bug fixed, `niagaraTest` runs ONLY BTestNg / TestNG tests discovered
  via the XML; pure `org.junit.Test` classes are NOT discovered by `niagaraTest` regardless of
  the plugin version.  Use a standalone `javac + junit` invocation for WSL-viable pure-JUnit gates.
  `[ev: corpus B807]`
- B741 / B1028 §1028.5 — A BComponent cannot be constructed in plain JUnit / WSL; `Sys.init()`
  and the NRE kernel are required; any method calling `BAbsTime.make()`, `Sys.loadType()`, or
  baja slot machinery fails with NPE or initialization errors.  Extract all testable logic into
  Baja-free static helpers to stay WSL-viable.  `[ev: corpus B741, B1028]`

**Build & tooling false positives**
- B1015 §1015.5 — `report-module.sh` plano check (`IMG_W`/`IMG_H` not found) fires as FAIL on a
  3D SPA module that has no plano overlay; the check is only meaningful for 2D plano dashboards.
  `lint-structure.sh` already has flag `--plano` to make this opt-in; `report-module.sh`
  should not auto-run it.  `[ev: corpus B1015]`
- B1015 §1015.4 — `verify-module.sh` phantom-dep check flags `api(project(":MOD-rt"))` as a
  phantom dep; the check recognizes only string-form `api(":x")` and `nre(":x")` deps, not
  Gradle project-dependency form.  `[ev: corpus B1015]`
- B1016 §1016.2 — `settings.gradle.kts` pin `gradlePluginVersion = "7.6.17"` is COUPLED to the
  N4.14 SDK family; the plugins resolve against `niagara_home/etc/m2`.  Using a plugin version
  from a different SDK family breaks resolution silently.  `[ev: corpus B1016]`

**Already-folded / existing lint coverage confirmed**
- I1 (slot retype → OUTAGE): `METHODOLOGY.md` S28 + `schema-risk.sh` → **FOLDED**
- I2 (survival matrix SAFE/LOSSY/OUTAGE): `METHODOLOGY.md` S29 + `schema-risk.sh` → **FOLDED**
- I3 (JACE needs version bump for auto-install): `METHODOLOGY.md` S30 → **FOLDED**
- I4 (delete @NiagaraType → delete module-include.xml): `METHODOLOGY.md` S31 → **FOLDED**
- I5 (delay ≤ 0 → IllegalArgumentException): `METHODOLOGY.md` S139 + `lint-delays.sh` → **FOLDED**
- I6 (timer only in atSteadyState → silently never arms): `types/logic.md` §17 → **FOLDED**
- I7 (empty skeleton -wb/-ux L9): `types/structure.md` + `lint-structure.sh` → **FOLDED**
- I8 (absolute host paths in gradle.properties L10): `lint-structure.sh L10` → **FOLDED**
- I9 (pre-Slotomatic empty skeleton "multiple metadata blocks" K15): `METHODOLOGY.md` K15 → **FOLDED**

**Pending in today's sibling retros** (not re-proposed here; cross-references only)
- I10 (cross-module @NiagaraProperty type= WARN): schema-versioning-upgrade-safety-deltas Δ5 (pending)
- I11 (Fox ordinal renumber WARN in schema-risk.sh): schema-versioning-upgrade-safety-deltas Δ6 (pending)
- I12 (moduleTest-include.xml missing WARN in verify-module.sh): sdk-examples-kit-deltas Δ1 (pending)
- I13 (plano false-positive fix in report-module.sh): umbrelladashboard-module-creation Δ1 (pending)
- I14 (phantom-dep fix for project-form deps): umbrelladashboard-module-creation Δ2 (pending)
- I15 (gradle plugin coupling documented in METHODOLOGY): n4-client-build-config-standard Δ2 (pending)

**Genuinely new item — no retro captures it yet:**
- I16 (`atSteadyState`-only timer class has NO override of `started()` → the omission-pair is
  never flagged by any lint; `lint-timers.sh` checks `changed-sched` and `discarded-ticket` but
  NOT whether a class that arms a timer in `atSteadyState()` also overrides `started()`).
  `[ev: corpus B729; retro self-firing-timer (folded, but no verify check emerged)]`

---

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|-------|-----------------|-------|
| Δ1 | New file `types/issues-and-gotchas.md` — consolidated INSIGHTS & ISSUES catalog.  Each entry follows the format: **symptom → root cause → fix/avoidance → [ev: corpus B<n>] → kit coverage (folded/pending/check)**.  Groups: (A) Schema & upgrade safety, (B) Timer/clock lifecycle, (C) Test infrastructure, (D) Build & tooling false positives.  Full proposed content in the fenced block below. | `types/issues-and-gotchas.md` (new file) | `issues-gotchas-catalog-v1` |
| Δ2 | Add new WARN check `atSteadyState-only-timer` to `toolbelt/lint-timers.sh`: for each Java file, detect a class that (a) overrides `atSteadyState` containing `Clock.schedule` or `schedulePeriodically` AND (b) does NOT override `started()`; emit `WARN atSteadyState-only-timer <file>: timer armed only in atSteadyState() — add started() override with Sys.atSteadyState() guard or commissioning-time mounts will never arm the timer`; exit 0 (WARN-only, not FAIL — a BTimeTrigger subclass may legitimately rely on the parent's started()). | `toolbelt/lint-timers.sh` — new check after existing `discarded-ticket` block | `atSteadyState-only-timer-warn` |
| Δ3 | Add one-line pointer in `METHODOLOGY.md` after the Schema/upgrade safety bullet block (after line 31): `- **Quick-reference of ALL known runtime traps and tooling false positives:** see \`types/issues-and-gotchas.md\` — grouped symptom → root-cause → fix catalog (14 entries as of 2026-09-18).` | `METHODOLOGY.md` — after schema/upgrade safety section | `issues-gotchas-ptr` |

---

### Δ1 — Proposed `types/issues-and-gotchas.md` content (fenced block)

```markdown
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
  atSteadyState(){ if(isRunning()) arm(); }
  started(){ super.started(); if(Sys.atSteadyState()) arm(); }
  clockChanged(BRelTime shift){ try { arm(); } catch(Throwable t){ … } }  // optional, cheap
The `started()` + `Sys.atSteadyState()` guard fires exactly-once across both paths and avoids
the NotRunningException race on boot.  Missing `clockChanged` leaves a stale absolute
next-fire target after an NTP/DST shift.
**Live case:** BDefrostController (ColdRoomPan-rt), PANCCADIA León, defrost never fired.
[ev: corpus B729 §729.4, retro self-firing-timer] — **Kit coverage: types/logic.md §17 (FOLDED)**
**Proposed check:** see Δ2 of this retro (`atSteadyState-only-timer` WARN in lint-timers.sh)

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
```

---

## Lessons

1. **The kit is thorough but scattered** — all 14 catalog entries have individual coverage
   somewhere (METHODOLOGY.md, retros, lint scripts); the GAP is a single scannable entry point
   before starting work, not missing coverage.

2. **One genuinely new check (Δ2)** — `lint-timers.sh` already has `changed-sched` and
   `discarded-ticket`, but does NOT flag the omission pattern: a class that arms in `atSteadyState()`
   without also overriding `started()`.  This is the most common commissioning gotcha in our
   codebase and it is not currently detectable by any lint.

3. **Two pending sibling retros expand the schema-risk tooling** — `schema-versioning-upgrade-safety-
   deltas.md` (today, pending) adds Δ4 (@Range blind spot), Δ5 (cross-module type= WARN), and Δ6
   (Fox ordinal WARN); they should be folded before or together with this catalog so cross-references
   are accurate.

4. **Issues D1 and D2 are paper-cuts that create noise on every new 3D-SPA module** — they are
   low-risk but cost a builder time on each campaign; prioritize umbrelladashboard-module-creation
   Δ1/Δ2 early in the next campaign.

5. **`types/issues-and-gotchas.md` should be a LIVING document** — every new retro that
   surfaces a first-class trap should add a row; it is the right home for symptom-first lookup.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-insights-issues-catalog-deltas.md | kit | 2026-09-18 | pending | 3 |`
