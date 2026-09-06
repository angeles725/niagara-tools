<!-- audience: framework-extension authoring — SPIs, point extensions, containers, queries, templates, jobs, watchdogs, action protection, minimal module -->
<!-- For rt control authoring (safety/timers, staging/interlocks, kitControl patterns): see `types/logic.md` -->
<!-- Split from types/logic.md at ## Author-side SPIs (line 91 in the pre-split file); content is verbatim — nothing rewritten or lost. -->
## Author-side SPIs

- **The N4 extension idiom (one shape for most SPIs):** to extend the framework, *subclass a framework base + register a `<type>`/agent in `module.xml` + hand back a self-describing SPI object* — there is NO central registry the author edits. Learn it once; B778 (service/ORD-scheme/subscription), B782 (query providers), and B785 (rdb dialect) are three instances. `[ev: corpus B778/B782/B785/B777]`
- **Custom SERVICE:** `extends BAbstractService`, override `getServiceTypes(){return new Type[]{TYPE};}` → registered by dropping it under `/Services` (auto at bootstrap); hook `serviceStarted()`; look up via `Sys.getService(Type)`. `[ev: corpus B778]`
- **New ORD SCHEME:** `extends BOrdScheme` (a `BSingleton`) + `@NiagaraType(ordScheme="<id>")` + `@NiagaraSingleton`; ctor `super("<id>")`; override `resolve()`; Slot-o-Matic emits `<type … ordScheme="<id>"/>`; registry resolves via `BOrdScheme.lookup`. `[ev: corpus B778]`
- **Server-side SUBSCRIPTION:** subclass `javax.baja.sys.Subscriber`, override `event(BComponentEvent)`, call `subscribe(component, depth, cx)` and `unsubscribeAll()` on stop (the SERVER complement to the BOX client). `[ev: corpus B778]`
- **EXCEPTION — analytics nodes register by TYPE, not by an agent:** a custom analytics node is a `@NiagaraType` subclass of `javax.bajax.analytics.algorithm.BOutputBlock` (implement `getValue`/`getTrend`, or `BFunctionBlock.apply` for single-input); inputs are `BBlockPin` `@NiagaraProperty` wired by `BLink` DAG edges; registered by a plain `module.xml <type>` with NO `@AgentOn`; external feed = the duck-typed `AnalyticDataSource.Provider`. `[ev: corpus B773]`

## Inter-module communication `[ev: corpus B802]`
- **Within a station, runtime comms is module-AGNOSTIC:** `BLink`, service discovery (`Sys.getService(Type)`), and `Subscriber` NEVER check the source module — a cross-module link / lookup / subscription is identical to a same-module one. The only real boundaries are (a) the COMPILE-TIME `<dependency>` on the other module's `Type`, and (b) the `fox:` ORD hop to a SEPARATE station (a real JVM boundary). Extends B778 (same-space services + `Subscriber.event`) with the cross-module + distributed picture. `[ev: corpus B802]`

## Authoring a point extension

- **Extend `javax.baja.control.BPointExtension`** (there is NO `BAbstractPointExt`, no `onExtended`/`onRetracted`); implement the sole abstract `onExecute(BStatusValue out, Context cx)` — mutate `out` (control), or leave it (notification-only). Override `requiresPointSubscription()`→true only to see every change; reach the point via `final getParentPoint()`; restrict hosting with `isParentLegal`/`isSiblingLegal`; execution is slot-declaration order, proxyExt always first. `[ev: corpus B772]`

## Child-tree containers — pick by cardinality

- **Frozen `@NiagaraProperty`** (BComponent-typed) for fixed/known children; **runtime `add(name, BValue)` + `reorder(Property[])`** for data-driven children; **a typed `BFolder` subclass** for a homogeneous growable collection. There is NO `BComponentList`. Enforce a typed tree by overriding `isChildLegal`/`isParentLegal` (both default `true`) with `instanceof` vetoes. `[ev: corpus B779]`

## Grouping and relating declaration surfaces (three postures)

- **Categories:** author NOTHING — every `BComponent` is `BICategorizable`; categories are operator-runtime via `BCategoryService`. Emit NO category scaffold. `[ev: corpus B781]`
- **Relations:** never subclass `BRelation` (a concrete carrier); define a relation type by registering a `RelationInfo`/`BCustomRelation` in a tag dictionary. `[ev: corpus B781]`
- **Hierarchy:** compose a `BHierarchy` root + ordered `BLevelDef` children (`BQueryLevelDef`/`BRelationLevelDef` entity levels, `BGroupLevelDef`/`BListLevelDef` grouping) under `BHierarchyService`; subclass `BLevelDef`+`getElements` only for a bespoke level. `[ev: corpus B781]`

## Query/search/index surface

- Declare a typed `BQuery`/NEQL payload + plug the matching `BIAgent` provider (`BQueryEngine` execute / `BColumnsProvider` table columns / `BISearchProvider` station search [`@AgentOn` scope×scheme] / `BSystemIndexer`+`BIIndexQueryProvider` station index [scope = a `BOrdList` of NEQL queries]) → read the resulting `BITable` cursor. `[ev: corpus B782]`

## Templates are artifact production, not a type SPI

- A "template type" is an `.ntpl` ZIP (a `.bog` + `template-manifest.xml`) produced by a make-job (`BMakeTemplateJob`/`NiagaraTemplate.createFrom().save()`) from a component subtree marked with a `BTemplateConfig` (+ `BConfigBinding` children + tagged parameter slots). Do NOT scaffold a `BTemplate` subclass — there is no such SPI. `[ev: corpus B783]`

## Background jobs

- Subclass **`BSimpleJob`** + implement **`run(Context)`** for the normal async case (dedicated thread + auto success/fail + interrupt-cancel are free); report `progress(pct)` + `log().*`; submit via `BJobService.submit(job,cx)` and track the returned `BOrd` (poll `getJobState()`/`getProgress()` — no join). Use raw `BJob` (`doRun`+`doCancel`) only to own threading. Multi-step = `BJobStep`/`BDeviceJobStep` under a `BBatchJob`. `[ev: corpus B774]`

## Watchdogs and timers

- **Watchdog/monitor:** subclass `BAbstractAlarmMonitor` (override `doRunCheck()`/domain `checkX()` + `getToNormal/OffnormalText`; maintain `status`/`lastAlarmTime`; edge-latch via `raiseAlarm(...)`). Cadence is a configurable `BIntervalTriggerMode` (default 15 min), NOT a 2s poll; distinguish from the native `EngineWatchdog` (engine/process heartbeat, a separate layer). `[ev: corpus B775]`
- **Timer:** `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` (repeating) — KEEP the returned `Clock.Ticket` in a field; cancel it in `stopped()`; re-arm in `changed()` when the interval is configurable (`BRandom` exemplar). `BTimer` is a clHVAC wiresheet block, NOT a scheduler. `[ev: corpus B775]`

## Action protection

- Gate an action DECLARATIVELY: `@NiagaraAction(name="…", flags=Flags.OPERATOR)` = operator-invoke (256); OMIT the flag = admin-invoke (the DEFAULT); enforced by `BComponent.canInvoke` + the fox/box `PermissionException` — no permission code in the body. Reserve operator for low-privilege writes; config/emergency stay admin-only. Use `AccessController.doPrivileged` ONLY for a JVM permission (read a `BPassword`, `setDefault` an authenticator, set a system property) — NEVER wrap a Niagara RBAC check. `[ev: corpus B776]`

## Minimal module (copy-start)

- **The SMALLEST correct module (proven by build in B793)** = a SOURCE tree the gradle plugin turns into a signed jar: `<MOD>-rt/module-include.xml` (the `<type>` list — the plugin GENERATES `META-INF/module.xml`, you do NOT author it) + `<MOD>-rt/module.lexicon` (SOURCE name; the plugin renames it to `<MOD>-rt.lexicon` in the jar) + a non-empty `module.palette` (one `<p>` per component) + `<MOD>-rt.gradle.kts` (the profile gradle file — findProjects convention, NOT `build.gradle.kts`) + one `B<Comp> extends BComponent` with one `Flags.SUMMARY|Flags.OPERATOR` property + one `Flags.HIDDEN` engine action whose handler the developer HAND-WRITES as `do<Action>()` (Baja calls `doTickExpired()`, not the generated `tickExpired()` wrapper) + one `Clock.Ticket` armed in `started()`+`atSteadyState()`, cancelled in `stopped()`. Slot-o-matic markers use the `//region /*+ … +*/ … //endregion` form. `preferredSymbol` in source is ignored (the plugin assigns the profile-dir name). Built with Java 8 (bytecode 52) + SIGNED. **Verified GREEN by an actual build (B793, a7396ec06): gate exit 0, ALL PASS.** `[ev: corpus B790, B793]`

## Logging a point to history `[ev: corpus B804]`

- **`BHistoryExt` IS a point extension** (`extends BPointExtension`), not a service or component child. Drop it as a child of a `BControlPoint` (e.g. `BNumericPoint`); it intercepts `onExecute` and records the point's value on every applicable event. `[ev: corpus B804]`
- **Interval vs COV** — `BIntervalHistoryExt` records on a timer (default 15 min; configure `interval`); `BCovHistoryExt` records on Change-Of-Value above a `changeTolerance` deadband. Pick COV for setpoints and slow-moving analog values that change infrequently; use Interval for continuous telemetry where a regular time-series is required. `[ev: corpus B804]`
- **`BHistoryConfig` capacity + `fullPolicy`** — `capacity` is the ring-buffer depth (record count, not bytes); `fullPolicy` is `roll` (overwrite oldest, default) or `stop` (reject new records when full). For an always-running production module, use `roll` so the history never silently stops accumulating. `[ev: corpus B804]`
- **One ext per logged slot** — attach exactly ONE `BHistoryExt` child per logged slot; two extensions on the same point log duplicate records and double the storage cost. `[ev: corpus B804]`
- **B804-G1 OPEN (requires-execution):** `BHistoryExt` on a custom `BComponent` (vs a `BControlPoint`) needs a live station smoke to confirm the extension mechanism works with a non-point parent. `[ev: corpus B804]`

## Slot types for externally written values `[ev: corpus B823]`
When a value is written by an EXTERNAL client (oBIX/write-server, the -ux servlet, a fox/BajaScript client), the slot
TYPE decides whether the write is even possible and whether it lands safely. Pick by value class: `[ev: corpus B823]`

| Value class | Recommended slot | Flags | How it is written externally | Audit path | Anti-pattern |
|---|---|---|---|---|---|
| numeric setpoint / config | plain `double` (if oBIX writes it) — or `BStatusNumeric` ONLY when the status must display | `SUMMARY\|OPERATOR` | `double`: oBIX bare `<real val="..">`. `BStatusNumeric`: the WRAPPED body `<obj is="…:StatusNumeric"><real name="value" val=".."/></obj>` (LIVE-verified) — NEVER attr-only `<obj … val>` (200 but writes 0.0); OR the servlet `POST /api/setpoint`; OR an OPERATOR action | servlet `auditLog` / write-server audit / a Niagara event when via an action | a bare `BStatusNumeric` written by external clients → the silent-zero footgun `[ev: corpus B823]` |
| timing / delay | `BRelTime` | `SUMMARY\|OPERATOR` | oBIX `<reltime val="PT..S"/>` | as above | a `double` seconds field that skips the reltime unit `[ev: corpus B823]` |
| switch / on-off | `boolean` | `SUMMARY\|OPERATOR` | oBIX `<bool val="true">` | as above | a `BStatusBoolean` (complex) written bare → "Cannot translate" `[ev: corpus B823]` |
| mode / HOA | today `double` 0/1/2 written as `<real>`; for future modules a **FROZEN enum** (`BFrozenEnum` via `@NiagaraEnum`/`@Range`, e.g. a `BHoaMode` auto/hand/off = 0/1/2) carries its range INTRINSICALLY — no explicit facet needed | `SUMMARY\|OPERATOR` | `double`: `<real val="2"/>`. FROZEN enum: renders `<enum val="hand" display=… range=…/>` and decodes `<enum val="hand"/>` with NO explicit `BFacets.RANGE` — the encoder (`ObixUtils:358`) and decoder (`ObixDecoder:184/245/333`, `setFromVal`) fall back to the value's `getRange()`. A DYNAMIC enum is the only case that needs an explicit range facet. The `@Range` tags need `module.lexicon` keys (SP6 known set). `[ev: corpus B828]` | as above | a `double`→enum switch is a LOSSY retype (OUTAGE) → future modules only; existing RoomPanel modes stay `double` 0/1/2 `[ev: corpus B828]` Carve-out: a FROZEN enum HOA is for an INTERNAL, single-module slot (it self-describes over oBIX); a value LINKED across custom modules stays a plain `double` (see `logic.md` §Linking) — a shared enum forces the cross-module dependency and re-invites the deleted-`BHoaMode` `Missing class ColdRoomPan:HoaMode` station crash. `[ev: corpus B828]` `[ev: corpus B818]` |
| button / command | an OPERATOR `@NiagaraAction` | `Flags.OPERATOR` | oBIX `<op>` — POST → `BComponent.invoke` under `OPERATOR_INVOKE`, arg from `<real>`/`<bool>` `[ev: corpus B822]` | the Niagara invoke event (attributed) | a `HIDDEN` action (0 oBIX exposure) or a boolean "pulse" slot |

**The rule:** a slot that EXTERNAL clients write is **either a SIMPLE value or has an ACTION — never a bare complex
property.** A bare complex (`BStatusNumeric`/`BStatusBoolean`/`BStatusEnum`) either rejects the write ("Cannot
translate") or, via the wrapped-`obj` shorthand, silently writes a DEFAULT (the live silent-zero: a setpoint set to
0.0 on a 200 OK). If the status MUST be displayed (so the slot has to be complex), expose an `OPERATOR` action that
writes it, or accept the exact wrapped-`obj` contract in the client — never leave a bare complex OPERATOR property as
the write target. `[ev: corpus B823]`

**Cleaner alternative — now PREFERRED `[CERT-live]` (B826-G1/G2 CLOSED):** the child ORD `…/setpoint/value` is NOT
advertised (the agent collapses the struct to a leaf `<real>`) but IS structurally resolvable
(`BStationLobbyAgent.decodeSlotPath`) — and the façade DOES serve it: a GET returns `200 <real … writable="true"/>` on
BOTH the RoomPanel and the ColdRoom (B826-G1, record §8 `b4e6d8a4f`), and a bare `<real val="N"/>` PUT to it writes AND
propagates to control in ~1.5 s (B826-G2, record §9 `f99f2e45b`), via the nested-child bubbling path (B825 §825.3, now
live-proven end-to-end). So the child bare-`<real>` is a `BSimple` write with NO silent-zero hazard, and is the
PREFERRED external write for a `BStatusNumeric`; the wrapped-`obj`-to-parent-slot form (B825) is the proven FALLBACK.
Rule: **a complex property is writable externally through its child leaf ORD (bare `<real>`); the parent slot needs the
wrapped `<obj>` and carries the silent-zero hazard.** `[ev: corpus B826]` `[ev: corpus B825]`

**Propagates through links? YES, synchronously (mechanism settled by [Block 825]):** an external write that lands as a
TOP-SLOT REPLACEMENT (an oBIX wrapped `<obj>` PUT, the servlet, or fox — all decode into a detached copy then
`parent.set(slot, copy)`, `ObixUtils.java:543/:558`) fires the slot's outgoing links SYNCHRONOUSLY on the writing
thread (`SlotKnobs.propagate:31-46`, <1 ms). So a write to the façade SOURCE (`Cuarto1/setpoint`) propagates to the
control TARGET (`ColdRoom_1/setpoint`) in <1 ms — the read-back "settle" is the READER's poll cadence (~1 s a
control-slot poll / ~6 s the dashboard poller), NOT a propagation delay. Rule: an external write must land on the slot
the control READS (or its link SOURCE), or on an action — a write to a display-only mirror with no link, or to the
link-TARGET side (which the next source propagation overwrites, B816 §816.2), does not move the plant.
`[ev: corpus B823]` `[ev: corpus B825]` `[ev: corpus B823]`

**Overlap caveat:** if the written slot is a link TARGET (driven BY a link, not a source), the external write is
EPHEMERAL — the next propagation overwrites it (B816 §816.2). Confirm write-source vs write-target before relying on
the write sticking. `[ev: corpus B816]`

## Write-path test matrix `[ev: corpus B816]`
Every writable slot a dashboard/operator can hit gets a ROW: (writable slot × writer × timing) → the invariant it must hold, and the TEST that proves it. The template is 5 columns — slot · writer · timing · invariant · test. `lint-write-path.sh` parses only the 4 STRUCTURAL columns (slot · writer · timing · test); **`Invariant` is a human-facing column the lint does NOT parse** (a lint cannot decide a semantic invariant). The `≤0`-delay class is OWNED by `lint-delays.sh` (PR1, B820 §820.1c) — `lint-write-path.sh` does NOT re-implement the `Clock.schedule` ≤0 scan; it cross-references it so the two lints never double-bite the same site. For the LINK_TARGET ephemeral-write fact that motivates the WARN row, see §Slot types for externally written values above. `[ev: corpus B816]`

| Writable slot | Writer | Timing | Invariant | Test |
|---|---|---|---|---|
| `setpoint` | Dashboard / Workbench | mid-cycle (latched) | cv in new band → HOLD, no chatter; crosses → flip exactly once; INVALID status → fail-safe HOLD | `w1_setpointChangeWhileLatched` |
| `hoaMode` | Dashboard operator | mid-cycle | HAND→ON, OFF→OFF, AUTO→autoValue | `w3_hoaFlipMidCycle` |
| `defrostInterval` | Workbench | mid-cycle, shortened → overdue | new interval < elapsed → `1L`, never `Clock.schedule(0)` | `w6_intervalWriteMidCycleOverdue` |
| a LINK-TARGET slot | Dashboard | any | write is EPHEMERAL (overwritten next propagation) — UI must not imply it stuck | `TODO(test)` |
| `resistanceMode` (HOA) | Dashboard operator | mid-defrost | OFF LOCKS OUT the heater even during the defrost sequence (OFF > sequence > HAND > AUTO); re-applies after exitDefrost | `❌ C9` |

(All rows above credited to `[ev: corpus B816]` — the section header token covers the table.)

**Coverage legend for the Test cell:** a real `srcTest/` test name (lint checks it exists); `🔶` an earlier-campaign test; `❌ C9` for an invariant that needs the rt-lifecycle seam (issue #815 — `changed()`-ordering, minOff/minOn, seedRestart). `[ev: corpus B816]`
