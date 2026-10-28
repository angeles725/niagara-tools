<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 26 -->
# 2026-10-28 · kit · course-coverage

**Session**: niagara-tools kit backlog, branch `chore/kit-backlog` from 81db90d8 (`odd/tasks/kit-backlog-2026-10-27.md`
B10 run A, B11 run B); a read-only coverage audit of the kit against what the N4 Developer Course taught (corpus
B1416-B1419, the recordings) and what corpus B1420 verified about it
**Delta count**: 26

## What happened
A read-only explorer listed 70 course topics and looked each one up in the kit (`types/`, `METHODOLOGY.md`,
`BUILD-LOOP.md`, `build-verify.md`, `corpus-index.md`): 46 covered, 12 partial, 10 missing, and 2+2 kit lines that the
course evidence contradicts (two contradicted by B1420, two refined by B1417 and B1420 so that the kit sentence was
wrong as written). Run A fixes the four contradictions, fills the doc gaps with one cited line each, adds the
`n4-dev-course` corpus-index rows, and lands three cheap lints whose patterns are exact. Run B owes the five deltas
whose specs come from corpus B1242 §1242.7 (K3, K4, K6, K11) and B1419 §1419.4 (row b).

## Evidence
- Course recordings Day 1-4: corpus B1416 (lifecycle, dev environment), B1417 (signing, security manager, modules,
  Slot-o-Matic), B1418 (lexicon, logging, BQL, leases), B1419 (services, alarms, NEQL, threads, drivers).
- Child-gap bundle CR1: corpus B1420 §1420.1 (disposition), §1420.3 (non-standard home), §1420.6 (stale first read
  after `lease()`), §1420.7 (COV sensitivity, bounded queues), §1420.8 (alarm UUID), §1420.9 (SystemDB, relation IDs),
  §1420.10 (shutdown thread, thread priority).
- The four contradicted kit lines (81db90d8): `types/logic-authoring.md:9` ("a course student overflowed a queue and
  restarted the station at ~5000 points"), `types/driver-authoring.md:403` ("COV sensitivity is a tuning-policy
  setting"), `types/logic.md:264` (`stopped()` always on `Nre:Console`), `types/structure.md:74` ("Disabling it …
  needs the `smDeveloperMode` licence feature").
- `@NiagaraProperty` defaults: Slot-o-Matic copies `defaultValue` verbatim into `newProperty(flags, <default>, …)`
  (client source: `newProperty(Flags.SUMMARY | Flags.OPERATOR, 2d, null)`), Java overload resolution then picks
  `newProperty(int, boolean|int|long|float|double|String|BValue, …)` by the literal's static type
  (`organized/baja/baja/vineflower/javax/baja/sys/BComplex.java:690-750`), and every typed getter casts
  (`organized/baja/baja/vineflower/com/tridium/sys/schema/ComplexSlotMap.java:411-464`).
- Palette module symbols: the decoder keeps one module per symbol and a later `m=` with the same symbol replaces it
  (`organized/baja/baja/vineflower/javax/baja/io/ValueDocDecoder.java:1038-1051`), so an identical repeated
  `m="x=Mod"` changes nothing and a different module under the same symbol retargets every later entry.
- Corpus-index: `corpus-index.md` § n4-dev-course lists B1230-B1242 only; B1300 (the course's command set), B1302
  (deck-to-docs study map), B1354 (verified course exemplars) and B1416-B1420 had no row.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A restart was reported at about 5000 subscribed points; its cause is unconfirmed: bounded driver worker queues throw `QueueFullException` to the poster, they do not restart a station (open gap B1420-G2) | `types/logic-authoring.md` § Observed-side `subscribed()` | `[ev: corpus B1420 §1420.7]` |
| Δ2 | COV sensitivity is driver-specific: the base `BTuningPolicy` has no COV slot; BACnet adds `covPropertyIncrement` (default 1.0, COV-Property subscriptions only); ordinary COV uses the device's COV_Increment; never a display facet | `types/driver-authoring.md` § Sizing | `[ev: corpus B1420 §1420.7]` |
| Δ3 | `stopped()` runs on `Nre:Console` for `quit` and an Application Director stop; a platform reboot or rename stops the station on the platform service thread: do not assume a thread name in `stopped()` | `types/logic.md` § Callback thread map | `[ev: corpus B1420 §1420.10]` |
| Δ4 | `niagara.security.manager.disable` does not turn the security manager off: licensed (`smDeveloperMode`) it switches to a logging developer mode, unlicensed it logs a WARNING and changes nothing | `types/structure.md` § Security Manager card | `[ev: corpus B1417 §1417.2]` |
| Δ5 | The first read after `lease()` is stale because driver reads are asynchronous (`readOk()` later): lease early, or wait for a `changed` event or a non-stale status | `types/logic-authoring.md` § Server-side SUBSCRIPTION | `[ev: corpus B1420 §1420.6]` |
| Δ6 | Return-to-normal updates the alarm record and keeps its UUID; a new off-normal makes a new record; count occurrences as records per source | `types/dashboard.md` § Real alarms | `[ev: corpus B1420 §1420.8]` `[ev: corpus B1419 §1419.2 a]` |
| Δ7 | The System DB is an embedded OrientDB under `<station home>/orientSystemDb`, excluded from backups; a relation ID must be defined in a tag dictionary before the Relation Manager accepts it | `types/logic-authoring.md` § Relation IDs (before Hierarchy) | `[ev: corpus B1420 §1420.9]` `[ev: corpus B1419 §1419.2 d-e]` |
| Δ8 | Engine watchdog defaults: Terminate after 1 minute without engine cycles; lower long-running threads with `setPriority(getPriority() - 1)` on every platform (Tridium's own practice) | `types/logic.md` § Name every thread | `[ev: corpus B1419 §1419.3]` `[ev: corpus B1420 §1420.10]` |
| Δ9 | Editing the policy files under `bin/` breaks the installation signature; every requested permission is granted, and the only remedy for a module's permissions is to uninstall it | `types/distribution.md` § Hard signing gate | `[ev: corpus B1417 §1417.2]` |
| Δ10 | A non-`B` class can be a type, but its `module-include.xml` entry and slot code are hand-written | `types/structure.md` § module-include.xml | `[ev: corpus B1417 §1417.3 d]` |
| Δ11 | A brand-new module builds with Workbench open; a changed module needs Workbench and the station restarted to load | `BUILD-LOOP.md` § 6 | `[ev: corpus B1417 §1417.3 n]` |
| Δ12 | Copy/paste gives the copy a new UUID (`DEFAULT_ON_CLONE`, then `started()`): never key state on a pasted component's id | `types/logic.md` § Flags | `[ev: corpus B1417 §1417.3 o]` |
| Δ13 | Spy paths: the registry's installed modules and types (`spy:/sysManagers/registryManager`), and Log Setup lists every defined logger while the DebugService lists only configured ones | `types/diagnostics.md` § 2 | `[ev: corpus B1417 §1417.3 f]` `[ev: corpus B1418 §1418.2 f]` |
| Δ14 | A lexicon translates display text only; it cannot change property order | `types/lexicon.md` § 1 | `[ev: corpus B1418 §1418.1]` |
| Δ15 | The search service, quick search and sidebar take bare NEQL (no `neql:`), return ORD links and cannot pipe into `|bql` | `types/dashboard.md` § BQL case rules | `[ev: corpus B1419 §1419.2 c]` |
| Δ16 | Device-specific write permissions (a PIN, a password level) belong on the proxy ext, not the point | `types/driver-authoring.md` § 1.6 | `[ev: corpus B1419 §1419.4 f]` |
| Δ17 | A non-standard system or user home: run `station`/`nre` from the Niagara Console (Windows, sets `niagara_home` and PATH; `set niagara_user_home=<dir>` there) or after sourcing `<sysHome>/bin/.niagara` (Linux) | `build-verify.md` § niagara_home on WSL | `[ev: corpus B1420 §1420.3]` `[ev: corpus B1416 §1416.5 c]` |
| Δ18 | `corpus-index.md` § n4-dev-course: rows for B1300, B1302, B1354 and B1416-B1420 | `corpus-index.md` § n4-dev-course | `[ev: corpus B1420]` |
| Δ19 | New lint `lint-charset-default` (WARN): no-argument `getBytes()`, a byte-array `new String(…)` without a charset, a one-argument `InputStreamReader`/`OutputStreamWriter`, any `FileReader`/`FileWriter` (Java 8 has no charset overload) | `toolbelt/lint-charset-default.sh`; `METHODOLOGY.md` | `[ev: corpus B1419 §1419.4 c]` |
| Δ20 | `lint-structure` L17 over each `module.palette`: FAIL a dotted class name in `t=` and a `B`-prefixed type the module declares without it; WARN another `B`-prefixed name and a symbol redeclared for a different module | `go/internal/lint/rules/structure`; `types/structure.md` | `[ev: corpus B1418 §1418.5 e]` `[ev: corpus B1419 §1419.4 d]` |
| Δ21 | New lint `lint-slot-default-type` (FAIL): an `@NiagaraProperty` of type `int`/`long`/`float`/`double`/`boolean`/`String` whose `defaultValue` literal has another Java type (Slot-o-Matic copies it verbatim, overload resolution stores the wrong value, the getter throws `ClassCastException`) | `toolbelt/lint-slot-default-type.sh`; `types/structure.md` | `[ev: corpus B1417 §1417.3 l]` |
| Δ22 | `build.sh --werror`: compile with `-Werror` (and `-Xlint` warnings on) as an opt-in, so a module that some vendors build warning-clean stays so | `toolbelt/build.sh`; `build-verify.md` | `[ev: corpus B1419 §1419.4 b]` |
| Δ23 | K4 lint: a `changed`/`added`/`removed` override without an `isRunning()` guard as its first statement WARNs; and a startup body in `stationStarted()` with no `started()` counterpart WARNs (it never runs for a component added to a running station) | new Go lint; `types/logic.md` | `[ev: corpus B1242 §1242.7]` `[ev: corpus B1416 §1416.3]` |
| Δ24 | K3 lint: `synchronized` methods and `synchronized (this)` blocks in `BComponent` subclasses WARN (lock a `private final Object` or use `Atomic*`) | new Go lint; `types/logic.md` | `[ev: corpus B1242 §1242.7]` |
| Δ25 | K6 preflight: run `gradlew -q javaToolchains` and fail when no full JDK 8 is listed for a 4.15 target | `toolbelt/preflight.sh` (`go/internal/preflight`) | `[ev: corpus B1242 §1242.7]` |
| Δ26 | K11 lint: a single `'` in a lexicon value that code reads with arguments (MessageFormat drops it; write `''`) WARNs, as a `lint-lexicon-ascii` companion | `toolbelt/lint-lexicon-ascii.sh` sibling | `[ev: corpus B1242 §1242.7]` |

## Owed to run B
Run B (`odd/tasks/kit-backlog-2026-10-27.md` B11) owes Δ22-Δ26. Exact specs, from corpus B1242 §1242.7 and the blocks
it cites:
- **Δ22 (#64)**: `build.sh --werror` adds `-Werror` to the Java compile of every part (plus the `-Xlint` set the
  build uses), off by default; a module that builds warning-clean keeps doing so. Source: B1419 §1419.4 row b
  ("some vendors build with `-Werror`"; `@SuppressWarnings("unchecked")` for `(BITable<T>) ord.get()`).
- **Δ23 (#4, K4)**: "New lint: `changed/added/removed` override without an `isRunning()` guard" (§1242.7 K4, from
  [B1232] §1232.2; checklist rule 22: the first line of `changed/added/removed` is `if (!isRunning()) return;`). Plus
  a WARN when a component's startup logic lives only in `stationStarted()` (B1416 §1416.3: it runs once at station
  bootstrap, never for a component dropped into a running station).
- **Δ24 (K3)**: "New lint: `synchronized` methods / `synchronized(this)` in `BComponent` subclasses" (§1242.7 K3,
  from [B1238] §1238.7; checklist rule 34: lock a `private final Object` or use `Atomic*`, no nested locks).
- **Δ25 (K6)**: "`preflight.sh`: run `gradlew -q javaToolchains` and fail when no full JDK 8 is listed for a 4.15
  target" (§1242.7 K6, from [B1235] §1235.2; the course diagnoses with
  `gradlew -q javaToolchains -Porg.gradle.java.installations.auto-detect=true`).
- **Δ26 (K11)**: "`lint-lexicon-ascii.sh` companion: WARN on a single `'` in a key that code reads with arguments"
  (§1242.7 K11, from [B1236] §1236.4; checklist rule 49: entries read with args follow MessageFormat, `''` for an
  apostrophe).

## Lessons
- A spoken course claim is a lead, not a rule: two of the four contradictions copied a recording's explanation
  ("overflowed a queue and restarted") that the code does not support. Cite the child-gap block that checked it.
- An exact lint can come straight from an overload table: when a generator copies a literal into a call, the
  literal's Java type decides the stored value, and the mismatch is decidable from source alone.
