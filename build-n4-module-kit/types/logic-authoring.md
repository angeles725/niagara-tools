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
