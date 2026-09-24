<!-- audience: framework-extension authoring — SPIs, point extensions, containers, queries, templates, jobs, watchdogs, action protection, minimal module -->
<!-- For rt control authoring (safety/timers, staging/interlocks, kitControl patterns): see `types/logic.md` -->
<!-- Split from types/logic.md at ## Author-side SPIs (line 91 in the pre-split file); content is verbatim — nothing rewritten or lost. -->
## Author-side SPIs

- **The N4 extension idiom (one shape for most SPIs):** to extend the framework, *subclass a framework base + register a `<type>`/agent in `module.xml` + hand back a self-describing SPI object* — there is NO central registry the author edits. Learn it once; B778 (service/ORD-scheme/subscription), B782 (query providers), and B785 (rdb dialect) are three instances. `[ev: corpus B778/B782/B785/B777]`
- **Custom SERVICE:** `extends BAbstractService`, override `getServiceTypes(){return new Type[]{TYPE};}` → registered by dropping it under `/Services` (auto at bootstrap); hook `serviceStarted()`; look up via `Sys.getService(Type)`. `[ev: corpus B778]`
- **New ORD SCHEME:** `extends BOrdScheme` (a `BSingleton`) + `@NiagaraType(ordScheme="<id>")` + `@NiagaraSingleton`; ctor `super("<id>")`; override `resolve()`; Slot-o-Matic emits `<type … ordScheme="<id>"/>`; registry resolves via `BOrdScheme.lookup`. `[ev: corpus B778]`
- **Server-side SUBSCRIPTION:** subclass `javax.baja.sys.Subscriber`, override `event(BComponentEvent)`, call `subscribe(component, depth, cx)` and `unsubscribeAll()` on stop (the SERVER complement to the BOX client). `[ev: corpus B778]`
- **Observed-side `subscribed()` / `unsubscribed()` — demand activation on the WATCHED component:** these callbacks fire ON a component when a remote client (Workbench, BOX browser) STARTS/STOPS watching it — distinct from `Subscriber.subscribe()` (which is the OBSERVER API: *I am watching*). Override to do demand-based resource acquisition/release (e.g., open a serial port only while someone is watching the status component). Caveat: fires only at the 0↔1 subscriber-count crossing, not on every client join/leave. `[ev: corpus B4 §4.3.1, B408 §408.4.2]`
- **BOX→SubscribeCallbacks→subscribed() causal chain:** browser/Workbench BOX `sub` op → `ProxyBroker.subscribeOp` → space subscriber-count 0→1 → `SubscribeCallbacks.subscribe()` fires → `component.subscribed()` fires. This is WHY `subscribed()` and `SubscribeCallbacks` fire — the trigger is a remote client starting a live subscription, not a local observer. A `BComponent` author who overrides `subscribed()` is therefore reacting to a Workbench panel or browser SPA opening a live view on that component. `[ev: corpus B408 §408.4.2, B512 §512.3–512.4]`
- **TYPE-LEVEL SUBSCRIPTION (`TypeSubscriber`) — watch every instance of a type across the space:** to receive events for EVERY mounted instance of a `BComponent` subtype T (vs instance-depth `Subscriber` which watches one component tree): (1) subclass `TypeSubscriber`; constructor receives `ComponentSpace` — call `super(space)` and store it; (2) override `abstract void event(BComponentEvent)` to handle each event; (3) in `started()`, call `subscribe(new Type[]{T.TYPE}, cx)` — the framework validates `t.is(BComponent.TYPE)` (throws on a non-component type) and calls `space.subscribe(t, this)`; (4) call `unsubscribeAll()` in `stopped()`. `[ev: corpus B867 §867.2-3]`
- **`TypeSubscriber` supertype-walk gotcha — subscribing to a BASE type catches ALL subtypes across the space:** `BComponentSpace.event()` recurses up `getSuperType()` — a `TypeSubscriber` on a BASE type (e.g., `BNumericPoint.TYPE`) catches ALL subtype instances' events across the ENTIRE space; subscribing to `BComponent.TYPE` = "watch everything" for the masked event-ids. Rule: be specific about the Type; profile event volume before subscribing to ANY framework base type; a too-broad base type causes a silent event flood with no compile-time warning. `[ev: corpus B867 §867.3]`
- **`TypeSubscriber` event-mask gotcha — default `SELF_EVENTS` covers lifecycle ONLY, not value changes:** `BComponentEventMask.SELF_EVENTS` (= 1 701 888) covers component lifecycle event ids 11–20 (`parented`, `started`, `stopped`, …). It does **NOT** include `PROPERTY_CHANGED` (id = 0) or the `PROPERTY_EVENTS` set (mask 395 263). A default-mask `TypeSubscriber` receives **no** value-change callbacks. To receive value-change notifications, call `setMask(BComponentEventMask.PROPERTY_EVENTS)` (or compose masks with bitwise OR) before subscribing. Using the default mask while expecting value changes is a **silent miss**: compiles, passes the verify gate, never fires. `[ev: corpus B867 §867.4, B900 §900.1]`
### Subscribe / unsubscribe symmetry rule `[ev: retro module-hardening-failure-modes-deltas Δ3]`

Every `subscribe()` call in `started()` MUST have a matching `unsubscribe()` (or `unsubscribeAll()`) in `stopped()`. Omitting the teardown has three compounding effects:

1. **Memory leak** — `subscribe()` holds a **bidirectional reference** between the observer and the watched component; neither can be GC'd while the reference exists.
2. **Handler fires after removal** — the subscriber continues receiving events even after the component is removed from the station tree.
3. **Duplicate subscriptions on re-enable** — each `started()` adds another subscription; `enable → disable → enable` stacks duplicate subscriptions with no error.

**Preferred alternatives:**
- Use `lease()` for a transient one-time read (self-manages its subscription; no teardown needed).
- Prefer `BLink` for value propagation across slots (self-manages its subscription; correct by design).
- Use `TypeSubscriber.subscribe(...)` / `unsubscribeAll()` (see §Author-side SPIs) when watching all instances of a type.

**Lint candidate:** `subscribe-without-unsubscribe` — flag a `subscribe()` call in `started()` with no paired `unsubscribe()` or `unsubscribeAll()` in `stopped()` in the same class.

- **`BEventService` routing — routing IS the Baja link graph, no imperative registry:** `BEventSource.routeTo(name, routable)` = `add(name, consumer)` + `linkTo(source.event → consumer.process)`. `BEventFilter` chains (consumer = process action, producer = event topic). `BEventRecipient` is terminal async delivery via `ThreadPoolWorker`. The event envelope is a uniform `BEvent` (uuid / timestamp / source ORD / open value). Cross-station delivery uses `BStationRecipient` (Fox `EventChannel`). `BComponentEventSource` wraps a `Subscriber.event(BComponentEvent)` into a `BEvent`; `BComponentTypeEventSource` uses `TypeSubscriber` internally — the event module is an OVERLAY on the B867 subscription primitives. **License required:** `tridium:eventService`. `[ev: corpus B873 §873.1-4]`
- **EXCEPTION — analytics nodes register by TYPE, not by an agent:** a custom analytics node is a `@NiagaraType` subclass of `javax.bajax.analytics.algorithm.BOutputBlock` (implement `getValue`/`getTrend`, or `BFunctionBlock.apply` for single-input); inputs are `BBlockPin` `@NiagaraProperty` wired by `BLink` DAG edges; registered by a plain `module.xml <type>` with NO `@AgentOn`; external feed = the duck-typed `AnalyticDataSource.Provider`. `[ev: corpus B773]`

### Nav-tree ordering and visibility `[ev: corpus B757 §757.3–757.4]`

Two independent levers control whether a slot appears in the nav sidebar:

- **`isNavChild()→false`** hides the component from the nav sidebar ONLY. The slot still exists, is wireable, and is visible everywhere else (property sheet, wireboard). Override this when a child should not clutter the nav tree but must remain accessible for configuration. Contrast with `Flags.HIDDEN`, which removes the slot from ALL UI (nav, property sheet, wireboard) — use `HIDDEN` only for truly internal machinery.
- **Nav order = slot declaration order:** the nav tree lists `getNavChildren()` in the same order as slots are declared with `@NiagaraProperty`. Reorder the field declarations to reorder the nav sidebar entries. There is no separate nav-order property.
- **Virtual nav node recipe:** to expose a nav node that has no backing `@NiagaraProperty` slot (e.g. a synthetic grouping folder), implement `BINavNode` manually — override `getNavChildren()`, `getNavOrd()` (must return a stable `BOrd`), and `getNavIcon()`. Return the synthetic instance from the parent's `getNavChildren()` alongside its real child components. Canonical example: `BModulePaletteNode`. `getNavOrd()` must be stable across renames and restarts; a `BOrd` based on a constant string or handle is preferred over a computed slot path.

## Link lifecycle callbacks `[ev: corpus B958 §958.1–958.2]`

The full `doCheckLink` / `added` / `removed` / INDIRECT-link recipe lives in `types/logic.md
§Linking across custom modules → Link lifecycle — gating and reacting`.  That section is the
canonical reference (with code examples) and is not duplicated here.

**Summary of the three hooks:**

- `doCheckLink(source, sourceSlot, targetSlot, cx)` — return `LinkCheck.makeInvalid("reason")` to
  reject a proposed link before it is created; `LinkCheck.makeValid()` to accept.
- `added(Property, Context)` — fires when a link is added to the component (guard: `bValue instanceof
  BLink && isRunning()`).
- `removed(Property, BValue, Context)` — fires when a link is removed; use symmetrically to tear down
  any state `added()` set up.

**INDIRECT-link rule (see `logic.md` for the full code snippet):** a back-link created inside
`added()` MUST pass `true` as the `indirect` flag — `new BLink(sessionOrd, srcSlot, mySlot, true)` —
so a station restart that starts the target before the source does not leave the link dangling.
`[ev: corpus B958 §958.2]`

→ See `types/moduleTest.md` for the station-test recipe that covers `doCheckLink` and `added`/`removed`
  (the only tier that can exercise these hooks in WSL-equivalent isolation with a real NRE).

## Inter-module communication `[ev: corpus B802]`
- **Within a station, runtime comms is module-AGNOSTIC:** `BLink`, service discovery (`Sys.getService(Type)`), and `Subscriber` NEVER check the source module — a cross-module link / lookup / subscription is identical to a same-module one. The only real boundaries are (a) the COMPILE-TIME `<dependency>` on the other module's `Type`, and (b) the `fox:` ORD hop to a SEPARATE station (a real JVM boundary). Extends B778 (same-space services + `Subscriber.event`) with the cross-module + distributed picture. `[ev: corpus B802]`

## ORD resolution — calling BOrd from rt/ux code `[ev: corpus B5 §5.1.4–5.1.5]`

### Call-site recipe

```java
BOrd ord = BOrd.make("slot:/MyComp/mySlot");   // parse once; store in a field if reused
BObject result = ord.resolve(base, cx).get();  // base may be null for absolute ORDs
```

- `BOrd.make(string)` parses and normalizes the ORD string.
- `resolve(base, cx)` runs: parse → normalize → iterate query chain → `OrdTarget.get()` → `BObject`.
- **Absolute ORD** (host-anchored, starts with a host-type scheme): `base` may be `null`.
- **Relative ORD**: requires a non-null `base` component; resolution starts from the base context.
- **Normalization rule:** when a host-query fragment is encountered mid-chain, all prior fragments are trimmed — `slot:/a|ip:host|slot:/b` resolves as `ip:host|slot:/b` (the `ip:` fragment resets the root). Build cross-station ORDs from the host query outward. `[ev: corpus B5 §5.1.5]`

### Error types

| Exception | Meaning in practice |
|---|---|
| `UnresolvedException` | Target not reachable (access denied or node not found) |
| `UnknownSchemeException` | Scheme module not loaded in the JVM |
| `InvalidOrdBaseException` | Wrong or missing base for a relative ORD |
| `SyntaxException` | Malformed ORD string |
| `NullOrdException` | Empty (null) ORD supplied |

### ORD scheme typology `[ev: corpus B5 §5.1.1; B38 §38.1]`

29 registered schemes; 5 categories:

| Type | Description | Key examples |
|---|---|---|
| **host** | Identifies a network node (absolute; no base needed) | `local:`, `ip:host`, `fox:host` |
| **session** | Identifies a connected session | `fox:` (after host), `station:` |
| **space** | Navigates within a space (component tree, file system, JAR) | `slot:/path`, `h:N`, `module://mod/path` |
| **lookup** | Resolves a named service or type instance | `service:baja:AlarmManager`, `type:baja:Component` |
| **query** | Runs a search or query | `bql:`, `neql:`, `hierarchy:`, `nav:` |

Custom schemes register via `@NiagaraType(ordScheme="id")` — see §Author-side SPIs for the authoring recipe.

### `service:` lookup scheme `[ev: corpus B5 §5.1.1; B38 §38.1]`

`service:baja:AlarmManager` resolves the running service instance by type. Use this as a **configurable ORD pointer** stored in a `@NiagaraProperty` instead of a hardcoded `Sys.getService(Type)` call:

```java
// property: BOrd serviceRef = "service:mymod:FooService"
// at resolve time (null base OK — service: is a lookup scheme, hence absolute):
BObject svc = serviceRef.resolve(null, cx).get();
```

Warning: resolves the **first-registered** instance — same semantics as `Sys.getService`. Avoid if multiple instances may be registered and you need a specific one.

### `h:` handle scheme — BOG-document-local links `[ev: corpus B5 §5.2.4]`

- A handle (`h:N`) is **unique per BOG document, not globally** — the same `h:5` in two BOG files refers to two distinct entities.
- `BOrd` properties that link to another component in the same BOG file serialize as `v="h:N"`.
- `h:` ORDs are **path-independent**: rename or move the target component within the document without breaking the reference — contrasts with `slot:` which breaks on rename.
- Use `h:` for palette template intra-document links; use `slot:` for human-readable cross-document references.

### Cross-station ORD pipe composition `[ev: corpus B5 §5.1.1, §5.1.5]`

Chain fragments left-to-right with `|`:

```
ip:192.168.1.10|fox:|station:|slot:/MyComp/mySlot
```

- `ip:host` — establishes the TCP connection (host scheme; resets the chain root)
- `fox:` — opens the N4 tunneled Fox session (the real JVM boundary)
- `station:` — enters the remote station component space
- `slot:/path` — navigates the remote component tree

Compose programmatically with string concatenation or `BOrd.make(base, suffix)`. Because the normalization rule trims prior fragments when `ip:` appears, always build cross-station ORDs from the host query outward; do not prepend a `slot:` path before `ip:`. This extends the `fox:` boundary concept from §Inter-module communication with the full fragment pipe form.

### `nav:` ORD scheme — nav tree walk `[ev: corpus B35 §35.5.2]`

`BNavScheme` resolves `nav:Station/Floor1/Zone` by walking `getNavChild()` segment-by-segment:

- Each path segment calls `getNavChild(name)` on the current node until the leaf is reached.
- Permission enforcement is **inline**: a denied node raises `UnresolvedException` — the UI silently fails to expand that branch with **no error shown to the user**.
- A `nav:` ORD in a PX graphics binding targets a nav-file node, not a station component directly; the nav node's `getNavOrd()` provides the onward ORD to the actual component.

### `hierarchy:` ORD — stable cross-tree references `[ev: corpus B587 §587.1]`

`BHierarchyScheme extends BSpaceScheme` (`ordScheme="hierarchy"`); body parses as `HierarchyQuery extends SlotPath`.

- A `hierarchy:` ORD **survives component-tree reorganisation** — it resolves through tags/relations to the real component, not through slot paths. Prefer over `slot:` when a module must hold a reference to a component that operators may reorganize.
- The last path segment encoded as `station$3a$7c` (URL-escaped `station:|`) is the **ENTITY seam**: it dereferences to the real station component with an inline permission check.
- Choose `hierarchy:` when the reference must be stable across renames and tree moves; choose `slot:` when you control the component path and human readability of the ORD matters more than stability.

## Authoring a point extension

- **Extend `javax.baja.control.BPointExtension`** (there is NO `BAbstractPointExt`, no `onExtended`/`onRetracted`); implement the sole abstract `onExecute(BStatusValue out, Context cx)` — mutate `out` (control), or leave it (notification-only). Override `requiresPointSubscription()`→true only to see every change; reach the point via `final getParentPoint()`; restrict hosting with `isParentLegal`/`isSiblingLegal`; execution is slot-declaration order, proxyExt always first. `[ev: corpus B772]`

## Child-tree containers — pick by cardinality

- **Frozen `@NiagaraProperty`** (BComponent-typed) for fixed/known children; **runtime `add(name, BValue)` + `reorder(Property[])`** for data-driven children; **a typed `BFolder` subclass** for a homogeneous growable collection. There is NO `BComponentList`. Enforce a typed tree by overriding `isChildLegal`/`isParentLegal` (both default `true`) with `instanceof` vetoes. `[ev: corpus B779]`

## Grouping and relating declaration surfaces (three postures)

- **Categories:** author NOTHING — every `BComponent` is `BICategorizable`; categories are operator-runtime via `BCategoryService`. Emit NO category scaffold. `[ev: corpus B781]`
- **Relations:** never subclass `BRelation` (a concrete carrier); define a relation type by registering a `RelationInfo`/`BCustomRelation` in a tag dictionary. Full `BCustomRelation` recipe — declare a `BCustomRelation extends BRelationInfo` with source scope (`entity` or `station`), target `BTypeSpec`, and inbound/outbound relation-id facet maps; override `addRelations(entity, cx)` to resolve targets via an ORD/NEQL query:

  ```java
  BITable<?> t = (BITable<?>) BOrd.make("station:|slot:/|bql:select * from module:BTargetType where ...")
                                  .get(cx);
  TableCursor<?> c = t.cursor();
  while (c.next()) {
      BComponent target = (BComponent) c.cell(0);
      cx.add(new BasicRelation(relationId, target, /*inbound=*/false));
  }
  c.close();
  ```

  Consume with `entity.relations().get(Id, dir)` / `getAll(Id, dir)` — each `Relation`'s endpoint is the far component. Register the custom relation in the tag dictionary's constructor alongside the tag defs (it participates in the same `BSmartTagDictionary` registration; see `types/logic.md §Ship a tag dictionary`). `[ev: corpus B758 §758.2]`
- **Hierarchy:** compose a `BHierarchy` root + ordered `BLevelDef` children (`BQueryLevelDef`/`BRelationLevelDef` entity levels, `BGroupLevelDef`/`BListLevelDef` grouping) under `BHierarchyService`; subclass `BLevelDef`+`getElements` only for a bespoke level. `[ev: corpus B781]`

## Query/search/index surface

- Declare a typed `BQuery`/NEQL payload + plug the matching `BIAgent` provider (`BQueryEngine` execute / `BColumnsProvider` table columns / `BISearchProvider` station search [`@AgentOn` scope×scheme] / `BSystemIndexer`+`BIIndexQueryProvider` station index [scope = a `BOrdList` of NEQL queries]) → read the resulting `BITable` cursor. `[ev: corpus B782]`

## BQL from code — consumer cursor pattern `[ev: corpus B758 §758.4]`

The CONSUMER side of BQL — used by a dashboard servlet or service method that runs a query from code:

```java
BITable<?> t = (BITable<?>) BOrd.make(
        "station:|slot:/|bql:select * from <module:Type> where …")
    .get(Sys.getStation());
TableCursor<?> c = t.cursor();
while (c.next()) {
    BObject cell = (BObject) c.cell(columnIndex);
    // … process cell …
}
c.close();   // always close — leaks a resource if skipped
```

- `from <module:Type>` is the select-by-registered-type mechanism — it finds every mounted instance of `module:Type` in the station space.
- Always close the cursor in a `finally` block; a leaked cursor holds a read lock on the space.
- oBIX exposes the same query as `/obix/bql/<url-encoded-query>` — the same BQL string works over HTTP from a browser client.
- **Contrast with the PROVIDER surface** (`§Query/search/index surface` above): implementing `BQueryEngine` / `BColumnsProvider` is the server side that declares what a query can return; the cursor pattern here is the CALLER side that drives a query from Java code (a servlet, a `serviceStarted()` scan, a background job). Both sides are needed to build a full query-driven feature.

## Templates are artifact production, not a type SPI

- A "template type" is an `.ntpl` ZIP (a `.bog` + `template-manifest.xml`) produced by a make-job (`BMakeTemplateJob`/`NiagaraTemplate.createFrom().save()`) from a component subtree marked with a `BTemplateConfig` (+ `BConfigBinding` children + tagged parameter slots). Do NOT scaffold a `BTemplate` subclass — there is no such SPI. `[ev: corpus B783]`

## Background jobs

- Subclass **`BSimpleJob`** + implement **`run(Context)`** for the normal async case (dedicated thread + auto success/fail + interrupt-cancel are free); report `progress(pct)` + `log().*`; submit via `BJobService.submit(job,cx)` and track the returned `BOrd` (poll `getJobState()`/`getProgress()` — no join). Use raw `BJob` (`doRun`+`doCancel`) only to own threading. Multi-step = `BJobStep`/`BDeviceJobStep` under a `BBatchJob`. `[ev: corpus B774]`

### BJob submission safety rule `[ev: retro module-hardening-failure-modes-deltas Δ5]`

`BJobs.submit(job, cx)` posts to a `ForkJoinPool` sized `availableProcessors × niagara.job.threadsPerCPU`. Violations:

- **Never submit from a high-frequency callback (`changed()`, timer tick):** each submission drains one thread from the shared pool. A 1 s timer that submits a new job every tick can saturate the pool in seconds.
- **`submit()` itself is an `invoke` action (RUN6 RBAC rule applies):** the job runs under the submitted context. If the caller uses `post()` to defer submission, the RBAC context is dropped (see `types/security.md §2` for the post()-drops-context rule).
- **A saturated or shutting-down pool throws `RejectedExecutionException` as an unhandled `RuntimeException`** directly into the calling callback (the bare-callback death path — see logic.md §Tridium rt idioms). This exception is NOT a checked exception and is NOT caught by `catch(Exception)`.
- **Wrap `submit()` in `try/catch(RejectedExecutionException)`** and log or surface the failure; do not let it escape to the engine thread.

**Pattern:**
```java
try {
    BJobService.submit(myJob, cx);
} catch (RejectedExecutionException e) {
    logWarning("job-submit rejected (pool saturated)", e);
}
```

### Long RT operation → WB job-bar (PD-04 cross-reference) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ4]`

Any operation whose RT side takes >1 s (firmware upload, device scan, mass config write) MUST be authored as a `BSimpleJob` (see `§Background jobs` above) and exposed via a `BOrd` action. The WB side then follows the job-bar recipe (`submit → sync → resolve → registerForEvents → jobBar.load`). See `types/wb-widgets.md §Vendor-grade -wb UX patterns PD-04` for the full WB implementation. `[ev: corpus B1060]`

## @NiagaraTopic typed event payloads — live RT→WB state (PD-09) `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ9]`

When an RT component changes state in a way the WB view needs to reflect live (e.g. a device discovered, a scan completed, an alarm condition toggled), fire a typed `@NiagaraTopic` payload from the RT component rather than polling.

**Pattern:**
```java
// RT side — declare a topic slot on the rt component
@NiagaraType
@NiagaraProperty(name="stateChanged", type="baja:TopicFunction",
                 flags=Flags.HIDDEN|Flags.TRANSIENT)
public class BMyNetwork extends BBasicNetwork {
    // Fire a typed snapshot whenever relevant state changes
    private void notifyStateChange(MyStateSummary snapshot) {
        BMyStateSummary bSnapshot = BMyStateSummary.make(snapshot);
        getStruct().get(stateChangedProp).fire(bSnapshot);   // fires the topic
    }
}

// WB side — subscribe to the topic in doLoadValue() and update the UI
component.getStruct().get(BMyNetwork.stateChangedProp).subscribe(topicListener, cx);
```

**Rules:**
- The topic payload type (`BMyStateSummary`) must be a `BStruct` or `BSimple` serializable as Baja — no Java-only types.
- Unsubscribe in `doUnloadValue()` — see `types/logic-authoring.md §Subscribe / unsubscribe symmetry rule`.
- This is the correct alternative to a WB-side polling timer when the RT component already has the state; see `types/observability.md` for the observability angle.

`[ev: corpus B1058]`

## Poller robustness `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ18]`

A polling component (a service or rt component that calls a remote URL on a `Clock.Ticket`) MUST guard its outbound request on both a non-empty URL AND an `enabled` flag before attempting the poll:

```java
// In the timer callback or poll() method
if (!isEnabled()) return;                         // skip when component disabled
String url = getUrl().trim();
if (url.isEmpty()) {
    setLastPollStatus(BPollStatus.UNCONFIGURED);  // or equivalent slot
    logWarning("poller: url not configured — skipping");
    return;
}
// ... perform the poll
```

- A blank `url` with no guard causes the poll to attempt a connection to `""` and log a bare `"poll error:"` with a null-message exception — unhelpful for triage.
- An `enabled=false` component that still polls wastes I/O and may interfere with commissioning.
- Set `lastPollStatus=UNCONFIGURED` (or equivalent) so operators can distinguish "not configured" from "working" or "faulted".
- Observed LIVE: `[apillm] poll error:` (empty cause) at station boot 01:55 with a blank `url`. `[ev: retro apillm-headless-servlet-rt-4.14-deltas Δ18]`

## Watchdogs and timers

- **Watchdog/monitor:** subclass `BAbstractAlarmMonitor` (override `doRunCheck()`/domain `checkX()` + `getToNormal/OffnormalText`; maintain `status`/`lastAlarmTime`; edge-latch via `raiseAlarm(...)`). Cadence is a configurable `BIntervalTriggerMode` (default 15 min), NOT a 2s poll; distinguish from the native `EngineWatchdog` (engine/process heartbeat, a separate layer). `[ev: corpus B775]`
- **Timer:** `Clock.schedule` (one-shot) vs `Clock.schedulePeriodically` (repeating) — KEEP the returned `Clock.Ticket` in a field; cancel it in `stopped()`; re-arm in `changed()` when the interval is configurable (`BRandom` exemplar). `BTimer` is a clHVAC wiresheet block, NOT a scheduler. `[ev: corpus B775]`

## Schedule integration `[ev: corpus B874]`

Rules for integrating a control component with a `BControlSchedule`:

- **A link on `schedule.in` OUTRANKS ALL tree rules:** `BControlSchedule.doExecute()` checks the `in` slot FIRST — if anything is linked to `in`, its value wins over every week/special/event entry in the schedule tree. Always check whether a `schedule.in` link is active before assuming the schedule tree is driving output.
- **Composite priority = FIRST SLOT WINS (positional, no numeric field):** `getOutputSource(now)` does a positional walk and returns the FIRST slot whose time range covers `now`. There is no numeric priority field — slot order is the only arbitrator. `BWeeklySchedule` places `specialEvents` at slot 0 (beats `week` at slot 1 by the positional rule).
- **ONE `Clock.Ticket` per transition — no polling loop:** the scheduler arms a single `Clock.Ticket` at each transition and reschedules it at the NEXT transition boundary. There is no polling loop. A custom component that wraps a schedule must not substitute a periodic poll for this ticket.
- **Scan horizon = 90 days:** `scanLimit` defaults to 90 days. A schedule with no event within 90 days of `now` returns null from `getOutputSource`, so the linked output sees null/invalid. Design schedules so there is always a reachable event within the horizon.

## Action protection

- Gate an action DECLARATIVELY: `@NiagaraAction(name="…", flags=Flags.OPERATOR)` = operator-invoke (256); OMIT the flag = admin-invoke (the DEFAULT); enforced by `BComponent.canInvoke` + the fox/box `PermissionException` — no permission code in the body. Reserve operator for low-privilege writes; config/emergency stay admin-only. Use `AccessController.doPrivileged` ONLY for a JVM permission (read a `BPassword`, `setDefault` an authenticator, set a system property) — NEVER wrap a Niagara RBAC check. `[ev: corpus B776]`

## Minimal module (copy-start)

### Security module skeleton `[ev: corpus B777 §777.2–777.4]`

A **security module** provides a custom authentication scheme (e.g. LDAP, RADIUS, custom SSO) as a
first-class Niagara service.  The pattern is the same N4-extension idiom — subclass + register — but
with a few module-specific rules:

1. **Class hierarchy:** `extends BAbstractService` and `implements BAuthenticationScheme`.  `BAbstractService`
   makes the scheme auto-discoverable under `/Services`; the `BAuthenticationScheme` interface is the
   SPI contract that `AuthenticationService` calls.

2. **Permissions file (`module-permissions.xml`):** declare the scheme's required permissions in a
   `module-permissions.xml` alongside `module-include.xml` in the source tree.  The Gradle plugin
   **INLINES this file's content into the generated `META-INF/module.xml`** at build time — do NOT
   author a `META-INF/module.xml` manually, and do NOT ship `module-permissions.xml` as a separate
   resource; the plugin merges it.  Example:

   ```xml
   <!-- <part>/module-permissions.xml -->
   <?xml version="1.0" encoding="UTF-8"?>
   <modulePermissions>
     <permission name="javax.baja.sys.BajaPermission"
                 actions="authenticationScheme"/>
   </modulePermissions>
   ```

3. **`@AgentOn` registration:** in `module-include.xml`, register the scheme as an agent on
   `"baja:AuthenticationScheme"` so `AuthenticationService` discovers it:

   ```xml
   <type name="MyAuthScheme"
         class="com.vendor.mymod.BMyAuthScheme"
         agentOn="baja:AuthenticationScheme"/>
   ```

4. **Signing:** a security module MUST be signed with `NIAGARA4.RSA` / `NIAGARA4.SF` — same signing
   path as any production jar.  The station's `SecurityManager` enforces this; an unsigned or
   OEM-signed scheme is rejected at load time.

- **Every class listed as a `<type>` in `module-include.xml` MUST carry `@NiagaraType`** (so Slotomatic generates `TYPE = Sys.loadType(…)`). A class listed as a `<type>` without `@NiagaraType` is caught at registry-build time with a named SEVERE: `"Missing Sys.loadType()"` — it is NOT a silent NPE, but it BLOCKS the module from loading at station start. Fix: add `@NiagaraType` to the class. `[ev: retro module-hardening-failure-modes-deltas Δ15]`
- **The SMALLEST correct module (proven by build in B793)** = a SOURCE tree the gradle plugin turns into a signed jar: `<MOD>-rt/module-include.xml` (the `<type>` list — the plugin GENERATES `META-INF/module.xml`, you do NOT author it) + `<MOD>-rt/module.lexicon` (SOURCE name; the plugin renames it to `<MOD>-rt.lexicon` in the jar) + a non-empty `module.palette` (one `<p>` per component) + `<MOD>-rt.gradle.kts` (the profile gradle file — findProjects convention, NOT `build.gradle.kts`) + one `B<Comp> extends BComponent` with one `Flags.SUMMARY|Flags.OPERATOR` property + one `Flags.HIDDEN` engine action whose handler the developer HAND-WRITES as `do<Action>()` (Baja calls `doTickExpired()`, not the generated `tickExpired()` wrapper) + one `Clock.Ticket` armed in `started()`+`atSteadyState()`, cancelled in `stopped()`. Slot-o-matic markers use the `//region /*+ … +*/ … //endregion` form. `preferredSymbol` in source is ignored (the plugin assigns the profile-dir name). Built with Java 8 (bytecode 52) + SIGNED. **Verified GREEN by an actual build (B793, a7396ec06): gate exit 0, ALL PASS.** `[ev: corpus B790, B793]`

## New-type authoring checklists `[ev: corpus B4 §4.2.3 §4.2.7]`

**Three-question decision rule** — answer in order, stop at the first yes:

1. Atomic and indivisible (a single value, no sub-slots)? → **`BSimple`**
2. Compound but no actions, no lifecycle, no dynamic children at runtime? → **`BStruct`**
3. Needs actions, children, or lifecycle (`started`/`stopped`)? → **`BComponent`**

### BSimple boilerplate

```java
public final class BMyValue extends BSimple {
    public static final BMyValue DEFAULT = new BMyValue(0);
    public static final Type TYPE = ...;  // Slot-o-matic emits this region

    private int raw;

    private BMyValue(int raw) { this.raw = raw; }

    public static BMyValue make(int raw) { return new BMyValue(raw); }

    @Override public void encode(DataOutput out)  throws IOException { out.writeInt(raw); }
    @Override public void decode(DataInput in)    throws IOException { raw = in.readInt(); }

    @Override public String encodeToString()              { return String.valueOf(raw); }
    @Override public BMyValue decodeFromString(String s)  { return make(Integer.parseInt(s)); }

    @Override public boolean equals(Object o) { return o instanceof BMyValue && ((BMyValue)o).raw == raw; }
    @Override public int    hashCode()        { return raw; }

    @Override public Type getType() { return TYPE; }
}
```

Rules: private constructor + static `make()` factory; must implement `encode`/`decode`, `encodeToString`/`decodeFromString`, `equals`/`hashCode`, and a `DEFAULT` constant.

### BStruct boilerplate

```java
public class BMyConfig extends BStruct {
    public static final Type TYPE = ...;

    // frozen @NiagaraProperty slots only — NO dynamic slots, NO @NiagaraAction, NO topics
    public BMyConfig() {}  // public no-arg constructor required

    @Override public Type getType() { return TYPE; }
}
```

Rules: extend `BStruct`; frozen `@NiagaraProperty` slots only; `add(name, value)` is FORBIDDEN (no dynamic slots); no `@NiagaraAction` or topics; public no-arg constructor.

### BFrozenEnum boilerplate

```java
public final class BMyEnum extends BFrozenEnum {
    // EXPLICIT ordinals — never rely on auto-numbering; inserting a new value
    // later shifts subsequent ordinals and corrupts serialized .bog data.
    public static final int ORD_OFF    = 0;
    public static final int ORD_AUTO   = 1;
    public static final int ORD_MANUAL = 2;

    public static final BMyEnum OFF    = new BMyEnum(ORD_OFF);
    public static final BMyEnum AUTO   = new BMyEnum(ORD_AUTO);
    public static final BMyEnum MANUAL = new BMyEnum(ORD_MANUAL);

    public static final Type TYPE = ...;

    private BMyEnum(int ordinal) { super(ordinal); }

    public static BMyEnum make(int ordinal) {
        switch (ordinal) {
            case ORD_OFF:    return OFF;
            case ORD_AUTO:   return AUTO;
            case ORD_MANUAL: return MANUAL;
            default: throw new IllegalArgumentException("invalid ordinal: " + ordinal);
        }
    }

    public static BMyEnum make(String tag) {
        return (BMyEnum) BFrozenEnum.make(TYPE, tag);
    }

    @Override public Type getType() { return TYPE; }
}
```

Rules: static `int ORD_*` constants with EXPLICIT ordinals; one static instance per ordinal; private constructor `super(ordinal)`; `make(int)` + `make(String)` factories; register the type in `module-include.xml`; declare `@Range` entries in the type annotation with matching ordinal values. `@Range` keys must be in `module.lexicon` (SP6 known set). Use `BFrozenEnum` for an INTERNAL, single-module discrete selector — a value shared across custom modules via a link stays a plain `double` (see `logic.md §Linking across custom modules`).

### BEnumRange authoring grammar `[ev: corpus B1109]`

`BEnumRange` carries the {ordinal → tag} map for a **dynamic enum slot** — one whose value set is configured at runtime or driven by protocol metadata (DIFFERENT from `BFrozenEnum`, which bakes ordinals into the class at compile time). Use it when the options vary by hardware or installation.

**Inline BNF (BOG `E:` prefix form):**
```
rangeExpr  ::= ["frozen+"] "{" pairs "}" ["?" opts]
pairs      ::= pair ("," pair)*
pair       ::= integer ":" identifier
opts       ::= optEntry ("," optEntry)*
optEntry   ::= "def=" integer | "null=" integer
```

Example: `E:{0:auto,1:hand,2:off}` — three ordinals, not frozen.  
Frozen example: `frozen+{0:off,1:stage1,2:stage2}` — operator cannot add/remove entries.

**`make()` overload catalogue:**

| Overload | When to use |
|----------|-------------|
| `BEnumRange.make(String[] tags)` | Sequential ordinals 0..n-1 — ONLY when ordinals are truly contiguous |
| `BEnumRange.make(int[] ordinals, String[] tags)` | Non-contiguous ordinals — REQUIRED for protocol enums with gaps |
| `BEnumRange.make(int count)` | Placeholder range of `count` ordinals (default tag names) |

**Non-contiguous-ordinal gotcha:** `make(String[])` FORCES ordinals 0..n-1. If a BACnet Multi-state object or Modbus register defines non-sequential codes (e.g. values 1, 3, 7), you MUST use `make(int[], String[])` — using `make(String[])` silently remaps ordinals and breaks any control logic that compares them. `[ev: retro module-hardening-reference-cards-deltas Δ4]`

## Programmatic slot-name rules `[ev: corpus B1110]`

When adding dynamic slots (`add(name, value)`) or forming `SlotPath` strings from external data (BACnet device names, meter IDs, spreadsheet rows), the name must satisfy the slot-name grammar or the slot is created in a broken state with NO error at creation time.

**Grammar:**
- First character: MUST be alphabetic (`[a-zA-Z]`) — no leading digit, no leading underscore, no space
- Subsequent characters: `[a-zA-Z0-9_]` — only underscore is allowed unescaped
- `$` prefix: used to escape special characters (e.g. `$20` for space)

**Silent failure mode:** a slot name with a leading digit or space is accepted by `add()` without exception, but the resulting `SlotPath` is malformed — ORD resolution fails, Workbench cannot navigate it, and links break silently.

**Rule:** always call `SlotPath.escape(name)` on any name derived from external data before passing it to `add()` or embedding it in a `SlotPath`:

```java
String escaped = SlotPath.escape(bacnetDeviceName);  // handles spaces, leading digits, special chars
add(escaped, BDouble.DEFAULT);
```

**Display name is INDEPENDENT of slot name:** the slot name is the programmatic key; the display name (shown in Workbench and oBIX) is set separately. Setting a human-readable display name does not require a human-readable slot name. `[ev: retro module-hardening-reference-cards-deltas Δ5]`

## BFacets key reference card `[ev: corpus B1106]`

`BFacets` is the property-metadata map; keys are string literals. The 31 known keys (B49 listed 11; B1106 completes the catalogue):

| Key | Purpose |
|-----|---------|
| `"units"` | Engineering unit (`BUnit`); shown in slot value display |
| `"precision"` | Decimal places for numeric display (int) |
| `"min"` | Minimum allowed value (numeric) |
| `"max"` | Maximum allowed value (numeric) |
| `"range"` | Enum range (`BEnumRange`); required for dynamic enums |
| `"fieldWidth"` | Character-width hint for text fields |
| `"radix"` | Display radix for integers (2=binary, 8=octal, 10=decimal, 16=hex) |
| `"showSeconds"` | Show seconds in time display (boolean) |
| `"showUnits"` | Append engineering-unit string to the formatted value (boolean) |
| `"unitConversion"` | Unit-conversion factor when bridging unit systems |
| `"maxOverrideDuration"` | Maximum HOA override duration (`BRelTime`) |
| `"targetType"` | Type constraint for ORD/link pickers (`BTypeSpec`) |
| `"editorTemplate"` | Workbench editor template name |
| `"editorClass"` | Custom editor class name |
| `"editorArgs"` | Arguments forwarded to the custom editor |
| `"editorColumns"` | Column definitions for a table editor |
| `"editorRows"` | Row definitions for a table editor |
| `"trueText"` | Display string for `true` (boolean slots) |
| `"falseText"` | Display string for `false` (boolean slots) |
| `"nullText"` | Display string for null/invalid values |
| `"format"` | A `BFormat` pattern for display formatting |
| `"icon"` | Icon ORD for Workbench display |
| `"help"` | Help URL or key for context-sensitive help |
| `"category"` | Default category tag |
| `"history"` | History configuration hint |
| `"alarm"` | Alarm configuration hint |
| `"trended"` | Whether the slot is trended by default (boolean) |
| `"locked"` | Whether the slot is locked from operator edit |
| `"persistent"` | Override persistence behavior |
| `"sparse"` | Sparse rendering hint for large tables |
| `"summary"` | Summary display hint |

**Factory methods** (all on `BFacets`):
- `BFacets.make(String key, BValue value)` — single key-value pair
- `BFacets.make(String[] keys, BValue[] values)` — multiple pairs (parallel arrays)
- `BFacets.makeDouble(BUnit unit, int precision)` — convenience for a numeric slot
- `BFacets.makeBoolean(String trueText, String falseText)` — convenience for a boolean slot
- `BFacets.makeEnum(BEnumRange range)` — convenience for a dynamic-enum slot

**Rule:** always use factory methods, never `new BFacets(Map)` directly. For a slot projection, use `getSlotFacets()` (see `types/logic.md §Tridium rt idioms`) to apply config-slot facets to outputs once rather than hardcoding them per output. `[ev: retro module-hardening-reference-cards-deltas Δ1]`

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

**`BRelTime` wire encoding is raw milliseconds, NOT ISO-8601:** `BRelTime.encodeToString()` emits `String.valueOf(millis)` — a plain decimal integer. This is ASYMMETRIC with `BAbsTime` (which uses `"13,<ISO-8601Z>"`). An external client (oBIX/REST/JSON) receiving a `BRelTime` value must explicitly convert: divide by 1000 for seconds, by 60000 for minutes, etc. Never assume ISO-8601 duration format (`PT30M` etc.) from a `BRelTime`. When exposing a duration slot to external clients, document the unit in the slot's facets or in the API contract. `[ev: retro module-hardening-reference-cards-deltas Δ6]`

`toolbelt/lint-ext-writable-shape.sh <src>` flags the anti-shape (an OPERATOR complex property with no `@NiagaraAction` whose body writes the slot). The exemption is per-slot: action `x` maps to `doX()` (B831-G1 convention); a `doAckAlarm` body writing `alarmAck` must NOT exempt `faultReset` — the write-target must match the OPERATOR slot being checked. The adapter→pure follow uses the `B<Pure>` naming pair only (prepend `"B"` to the pure class name). `[ev: retro campaign9-ext-writable-shape]` `[ev: retro campaign10-ext-writable-per-slot]`

## Control model limits and design patterns `[ev: retro control-model-limits-live-commissioning]`

### Split / sequenced-unit pattern (fan-first, reverse shutdown) `[ev: retro control-model-limits-live-commissioning Δ1]`

The `BEvaporatorUnit` model is **refrigeration-shaped** (valve-first: open the expansion valve, then start the fans, then call compressors). A split A/C unit or fan-coil unit is the **opposite**: fans must start BEFORE the compressor, and the compressor must stop BEFORE the fans (fan outlasts the compressor to clear residual heat).

**Pattern:** build a wire-sheet sequencer in the Niagara station (not in code): use timed gates and comparators (`COMPdelay`, `FANruns`) to: (1) prove fans are running before enabling the compressor, (2) stop the compressor N seconds before stopping the fans. This cannot be done through the `BEvaporatorUnit` slots — it requires a standalone sequencer component or a parallel override relay. State this in the module design up front so the integrator knows they need the sequencer.

**Multi-stage by demand = per-stage thresholds:** to stage two units by temperature, define per-stage setpoints/differentials. The second stage's differential must be DERIVED from the first (via `Add`/`Subtract` kitControl blocks linked into the second stage's `diffUp`/`diffDown`) so operator edits to the first stage automatically propagate — never hardcode the second stage's differential as a constant offset.

### Derived-differential staging `[ev: retro control-model-limits-live-commissioning Δ2]`

- **Pattern:** `stageUpDiff = Add(stage1.evapDifferentialUp, offsetUp)` and `stageDownDiff = Subtract(stage1.evapDifferentialDown, offsetDown)` linked into `EvaporatorUnit2.diffUp`/`diffDown`. The operator only edits the primary stage's band; the secondary tracks it automatically.
- **Anti-pattern:** hardcoding the secondary stage's thresholds. An operator edit to the primary breaks the staging relationship silently.

### Defrost has priority over HOA — there is no disable `[ev: retro control-model-limits-live-commissioning Δ5]`

- **Defrost overrides HOA unconditionally:** while a `BDefrostController` is active (`inDefrost=true`), `applyHoaOutputs` returns early — the HOA setting is ignored. This is by design: defrost must complete to prevent ice buildup.
- **There is no `enabled` slot on `BDefrostController`:** `BDefrostMode` has only `interval` and `schedule` variants (no `off`). "Turn off defrost" requires a code change (add a `defrostEnable` slot) or an out-of-band relay OR to keep the valve open.
- **Integrators must know this upfront:** a "keep the valve open" requirement during defrost cannot be solved through HOA. The options are: (a) add a per-evaporator `defrostEnable` slot to `BDefrostController`/`BEvaporatorUnit` (code change); (b) use an out-of-band valve OR relay; (c) gate the defrost schedule to never fire when the operator needs the valve open.
- **Relay-OR trap:** feeding a multi-state `valveMode` (double 0=auto/1=on/2=off) directly into a boolean `Or` block converts `!=0` to `true`, so BOTH `Encender(1)` AND `Apagar(2)` force the valve ON. Always use `Equal(valveMode, 1)` to isolate the ON state.

## Cross-field invariants and transient-flag recovery `[ev: retro live-commissioning-verification-gaps]`

### Cross-field invariants must be enforced in code or a lint `[ev: retro live-commissioning-verification-gaps Δ4]`

A constraint that relates two or more configuration fields (e.g. `hasDefrost` + `airDefrost` must both be `true` to enable air defrost; `interval` must exceed `duration`) is a **cross-field invariant**. Leaving it ONLY in a code comment is a commissioning hazard: there is no signal when the constraint is violated, and the defect is silent (defrost silently disabled, or defrosts non-stop).

**Enforce at one of these layers (in preference order):**
1. **Code derivation:** derive one field from the other so the invalid combination is structurally impossible (e.g. `airDefrostEnabled = hasDefrost && airDefrost` computed in the engine, not stored separately).
2. **Code rejection:** validate in the `changed()` handler and reset the field to a safe value, logging the reason.
3. **Lint:** add a `lint-config-sanity.sh` rule that detects the bad combination in source or in the live bog via `bog-audit.sh`.
4. **NEVER:** leave it in a comment only. `BEvaporatorUnit.java:182` is the live example — `hasDefrost=false` + `airDefrost=true` silently disabled air defrost on 3 rooms.

### Transient mode flag gating output writers must have unconditional recovery `[ev: retro live-commissioning-verification-gaps Δ5]`

A boolean flag that gates ALL output writers (e.g. `if(inDefrost) return;` at the top of `applyRunCmd` and `applyHoaOutputs`) creates a recovery gap: if the module stops while the flag is `true`, the outputs are left in their last state, and no subsequent code path writes them back because the flag still reads `true` from the stale transient.

**Rule:** any output writer that the module owns must have a guaranteed recovery path that does NOT depend on the flag:
- Clear protection/heater outputs unconditionally in `started()` and/or when the guarded flag is cleared, regardless of the flag's current value.
- The ONLY safe exit for a mode flag is: when the mode ends (e.g. `exitDefrost()`), clear the flag AND drive all guarded outputs to their safe default in one atomic step.
- `stopped()` must NOT be the sole recovery path for a flag that can be `true` when the module stops — `stopped()` deliberately skips output writes in some patterns (e.g. ColdRoomPan's `stopped()` clears `inDefrost` but does NOT write `resistanceOut`), leaving the output stuck ON until the next `started()` + `execute()` cycle.

**Lint candidate:** `lint-recovery-path.sh` — flag any transient boolean field set by an enter-mode method whose corresponding exit-mode method does NOT unconditionally write the guarded output(s). Until that lint exists, review every `if(flagField) return;` guard manually to confirm an exit path writes the output.

## Authoring a driver `[ev: corpus B810]`

When a module WRITES to a proxy point (a `BBooleanWritable` or `BNumericWritable` that sits above a driver point), follow these rules to avoid silent command loss and stuck relays.

- **Write to `in8`, not `in1`:** Niagara's priority-16 array arbitrates commands; level 8 is the MANUAL/OPERATOR write level (persisted across restarts). Writing to `in1` (supervisor override) can lock out the operator; writing to `in8` is the correct level for our own module's command. (`BBooleanWritable.java:490` — `in8` is the persisted manual level.)
- **Set a non-null `fallback`:** the `fallback` slot is the value the writable holds when ALL 16 priority levels are null (the relinquished state). If `fallback` is null on a `BBooleanWritable`, the relay HOLDS its last command on station stop/reload — 22 PANCCADIA relays exhibited this until the fallback was set. (`BBooleanWritable.java:726`: `fallback = newProperty(0, new BStatusBoolean(false, ...))`.)
- **`writeOnUp=true` (verify it is set):** `BTuningPolicy.writeOnUp` defaults `true` (`BTuningPolicy.java:206`), so a proxy point whose TuningPolicy uses the default is self-correcting — on device-UP it replays the last desired write. If `writeOnUp` is `false`, a write that was issued while the device was DOWN is silently dropped and the relay never receives the command. (`Tuning.java:201-204`.) Do NOT override `writeOnUp=false`.
- **Verify the write landed — read the DOWN/STALE status bit:** `set()` returning normally does NOT mean the device received the command; it means Niagara queued it. Always check the proxy point's `BStatus.isDown()` / `isStale()` bit after a write and surface it via an alarm or a health-status slot.
- **Device-side `relinquishDefault`:** configure the field device's own relinquish-default so the device falls back safely (e.g. valve closed) when the network drops, independent of the Niagara priority array.
- **`pingEnabled` monitor:** enable `pingEnabled` on the device network to get prompt DOWN detection; a stale ping-less point stays UP in Niagara while the physical relay is unresponsive.

- **Demand-based native-COV subscription via `SubscribeCallbacks`:** when the driver should register a native device COV subscription only while at least one client is watching a space component, implement `SubscribeCallbacks` and set it via `space.setSubscribeCallbacks(impl)`. Override `subscribe(BComponent[] c, int depth)` to register the native subscription when subscriber count goes 0→1; override `unsubscribe(BComponent[] c)` to deregister on 1→0. Use `BComponentSpace.update(c, depth)` for a one-time snapshot (no ongoing subscription). Rule: the framework tracks the subscriber count — one `subscribe()` call per 0→1 crossing, not per individual client joining. `[ev: corpus B408 §408.4]`
- **DDF driver: pick the transaction manager to match device concurrency:** DDF comm splits solicited (request→response) from unsolicited via two SPI managers. **`defaultComm`** (serialized, one-outstanding-at-a-time, `Clock.Ticket` timeout, blocking-queue dequeue) is for devices that cannot handle overlapping requests. **`multipleTransaction`** (tag-correlated concurrent, `Hashtable<tag, requests>` two-level map) is for capable protocols that can pipeline. Wrong pick = poll jams (serialized device overwhelmed by pipelined requests) or framing errors (pipelined device receiving one request at a time in the wrong order). `[ev: corpus B909 §909.1-3]`
- **`BPollScheduler` diagnostics — ONE thread per network, three rate buckets:** `BPollScheduler` runs ONE poll thread per `BDeviceNetwork`, named `"Poll:<networkName>"`. Three rate buckets: fast = 1 000 ms, normal = 5 000 ms, slow = 30 000 ms (defaults). `dibs` stack = immediate first-poll on subscribe (the proxy point is polled once right away before entering the regular bucket). Debugging a non-updating proxy point: (1) check which `BPollFrequency` bucket it is in; (2) check `pollEnabled` on the network; (3) confirm the `"Poll:<networkName>"` thread is alive in JVM diagnostics. `[ev: corpus B872 §872.1]`

- **Type-factory hooks: a `BPointDeviceExt` subclass must override three hooks** to tell the framework which concrete types to instantiate for the driver subtree: `getDeviceType()→BMyDevice.TYPE`, `getProxyExtType()→BMyProxyExt.TYPE`, `getPointFolderType()→BMyPointFolder.TYPE`. Pair these with trivial `extends BDeviceFolder` / `extends BPointFolder` subclasses (empty body + generated `TYPE`) — they exist solely to supply a driver-specific concrete `Type` handle; copy them verbatim, no business logic belongs in them. Without these hooks the framework cannot build the driver-specific subtree. `[ev: corpus B955 §955.1]`

- **Discovery learn-entry: discovery jobs `extend BSimpleJob`; each discovered entity is a `B<X>LearnEntry extends BStruct`** (fields: deviceName/deviceId or pointName/pointId/pointType) added to a `HIDDEN|READONLY|TRANSIENT BFolder` held on the job itself. The wb manager reads this transient folder after the job completes and calls `addDevice(entry, cx)` to build the real component tree. **Rule:** the learn-entry must carry enough state to reconstruct the full device/point representation at instantiation time — device address, type discriminator, and any runtime-fixed config the wizard dialog cannot infer. `[ev: corpus B955 §955.2]`

- **Async write via `CoalesceQueue`:** `write(Context)` posts an `ASYNC` action (`postWrite`) and returns `false` — NEVER block the engine thread in `write()`. Override `post(Action, BValue, Context)` to wrap each incoming action as an `Invocation` and route it to `getWriteHandler().postWork(...)`. The handler is a `BWorker` subclass holding a `CoalesceQueue(1000)` backed by a background `Worker` thread; `CoalesceQueue` collapses duplicate-key pending writes that have not yet been sent, so rapid setpoint changes coalesce instead of flooding the wire. This is the canonical "never block the calling thread" write pattern. `[ev: corpus B955 §955.5]`

- **Value-type-agnostic proxy poll:** in `poll()`, inspect `getParentPoint().getOutStatusValue() instanceof BStatusNumeric|BStatusBoolean|BStatusString|BStatusEnum` to select the matching `readOk(...)` overload. The proxy ext MUST be value-type-agnostic — the parent control point owns the output type; never hardcode a single response type in the proxy ext. A hardcoded type forces a specific control point kind, breaks the generic proxy pattern, and produces type errors silently when the parent is a different point type. `[ev: corpus B955 §955.6]`

**Lints (from `verify-module.sh --src` + `bog-audit.sh`):**
- HARD — null `fallback` on an own-module-driven `BBooleanWritable` / `BNumericWritable` (CHECK11 in `bog-audit.sh`): the writable holds last command on stop/reload, masking the fault.
- HARD — no `pingEnabled` on the device network / no `tuningPolicies` slot referencing our policy.
- WARN — `writeOnUp=false` on any tuning policy we control (defaults true; explicit false is a hazard).
- REVIEW — synchronous I/O in the write path (a `write()` callback that blocks on device I/O can freeze the engine thread).

**NOTE:** `writeOnUp`, `writeOnStart`, and `writeOnEnabled` all default `true` in `BTuningPolicy` — the DOWN self-correction behavior is ON by default. A fresh out-of-box proxy point is self-correcting; do not override these flags unless you understand the consequence.

## Adding a block icon `[ev: corpus B738 §738.4]`

Every `BComponent` subclass can override `getIcon()` to return a custom icon shown in Workbench's
component tree, property sheet, and palette.

### Directory convention

| Path | Usage |
|------|-------|
| `icons/x16/` | 16×16 raster PNG — the normal Workbench icon size |
| `icons/x32/` | 32×32 raster PNG — high-DPI / larger-format views |
| `icons/<file>.svg` | Single scalable file, replaces both raster sizes |

These are **module resource directories** (bundled into the jar root), distinct from `rc/` which
holds servlet static web assets (HTML/JS/CSS for the `-ux` browser view). Using `rc/icon16.png`
works at runtime because `module://` ORDs resolve any packaged path, but it conflates two different
resource purposes. **Use `icons/x16/` for component icons; use `rc/` for browser assets.**

### Raster icon recipe (two sizes)

```java
// Cache in a static final field — NEVER construct per-call (BIcon is not free to build).
private static final BIcon ICON = BIcon.std("myicon.png");
// BIcon.std(fileName) resolves to module://icons/x16/<fileName> (BIcon.java:69-71).

@Override
public BIcon getIcon() { return ICON; }
```

Place `myicon.png` (16×16) in `icons/x16/` and `myicon.png` (32×32) in `icons/x32/`. The
framework automatically picks the appropriate size.

### SVG icon recipe (single scalable file)

When a single vector file is preferred over raster pairs:

```java
private static final BIcon ICON =
    BIcon.make(BOrd.make("module://mymod/icons/myicon.svg"));

@Override
public BIcon getIcon() { return ICON; }
```

Place `myicon.svg` in `icons/` (module root, not in a sub-directory). SVG scales to any display
density without separate raster pairs. `BIcon.make(BOrd)` is the path for any non-`icons/x16/`
ORD. `[ev: BIcon.java:40-56]`

### Layered / badge icons

To compose a base icon with an overlay badge (e.g. a small lock for read-only, an alarm badge):

```java
private static final BIcon ICON = BIcon.make(
    new BOrdList(new BOrd[]{
        BOrd.make("module://mymod/icons/x16/base.png"),
        BOrd.make("module://mymod/icons/x16/badge-alarm.png")
    })
);
```

`BIcon.make(BOrdList)` renders the ORDs as stacked layers (base first, overlay on top).
`[ev: corpus B738 §738.4]`

### Static-final caching rule

A `BIcon` instance must be held in a `private static final` field. Constructing a new `BIcon` on
every `getIcon()` call creates unnecessary object churn and, for SVG/ORD-resolved icons, triggers a
module resource lookup on every call. The `static final` pattern is enforced by the idiomatic
`BIcon.std(fileName)` shorthand, which itself returns a cached instance.

`[ev: corpus B738 §738.4]`

## BSingleton + @NiagaraSingleton + @AgentOn — general agent pattern `[ev: code BLocalAlarmResolver.java]`

When the framework needs an implementation for a given target type (resolvers, providers, search agents), it looks up a registered `BIAgent` via `@AgentOn`. The canonical shape:

```java
@NiagaraType(agent = {@AgentOn(types = {"baja:LocalHost"})})
@NiagaraSingleton
public final class BLocalAlarmResolver extends BSingleton
        implements BIAlarmResolver, BIAgent {

    public static final BLocalAlarmResolver INSTANCE = new BLocalAlarmResolver();
    public static final Type TYPE = Sys.loadType(BLocalAlarmResolver.class);

    @Override public Type getType() { return TYPE; }

    private BLocalAlarmResolver() {}   // private constructor — singleton only

    @Override
    public OrdTarget resolve(BISession session, OrdTarget base, AlarmQuery query) {
        // implementation …
    }
}
```

Rules:
- **`extends BSingleton`** — `BIAgent` implementors that have no state extend `BSingleton`; stateful agents that need a lifecycle extend `BComponent` or `BAbstractService` instead.
- **`@NiagaraSingleton`** — tells Slot-o-Matic to emit the singleton registration; required alongside `BSingleton`.
- **`@AgentOn(types = {"module:TypeName"})`** — the target type the framework looks up; multiple targets are an array. The framework finds the agent by iterating mounted instances that `is(BIAgent.TYPE)` and comparing the registered target type.
- **Private constructor + static `INSTANCE`** — enforces single-instance contract; the framework calls `INSTANCE` directly, never `new`.
- **Does NOT appear in a palette** — a `BSingleton` cannot be dropped by the operator; it is registered in `module.xml` only. `[ev: code BLocalAlarmResolver.java]`

## Lifecycle guard contract `[ev: code BColdRoom.java, BCompressorControl.java]`

Every `changed` / `added` / `removed` override in a `BComponent` subclass must follow the same guard contract:

```java
@Override
public void changed(Property p, Context cx) {
    super.changed(p, cx);       // (1) ALWAYS call super FIRST
    if (!isRunning()) return;   // (2) guard — do nothing if not running
    try {
        // (3) business logic here
    } catch (Throwable t) {
        logError("changed", t); // (4) NEVER let an exception escape to the engine thread
    }
}

@Override
public void added(Property property, Context context) {
    super.added(property, context);   // super FIRST
    if (!isRunning()) return;
    // react to the newly added child / link
}

@Override
public void removed(Property property, BValue oldValue, Context context) {
    super.removed(property, oldValue, context);   // super FIRST
    if (!isRunning()) return;
    // tear down whatever added() set up
}
```

`stopped()` inverts the order — cancel tickets FIRST, call super LAST:

```java
@Override
public void stopped() throws Exception {
    if (tickTicket != null) { tickTicket.cancel(); tickTicket = null; }
    // release any other resources / clear transient state
    super.stopped();            // super LAST in stopped()
}
```

Summary of the three rules:
1. **`super` FIRST in `changed`/`added`/`removed`**, `super` **LAST in `stopped`** — the framework uses these calls to maintain its own internal state; violating the order leaves the component in a half-initialized or half-stopped state.
2. **`if (!isRunning()) return;`** immediately after `super.*` in `changed`/`added`/`removed` — guards against calls that arrive during start-up or shut-down before the component is fully operational.
3. **Wrap the body in `try/catch(Throwable)`** in `changed` and related callbacks — an uncaught exception from `changed` kills the engine thread. `[ev: code BColdRoom.java, BCompressorControl.java]`

### Lifecycle callback contract `[ev: retro module-hardening-failure-modes-deltas Δ2]`

The full lifecycle callback contract for a `BComponent` subclass:

- **`started()` / `stopped()` pair fires on BOTH enable/disable AND mount/unmount transitions.** A station start fires `started()`; a component enable fires `started()`; a component drop onto a running station fires `started()`. Both directions are symmetric: `stopped()` fires on component disable, on station stop, and on unmount.
- **`save` (bog serialize) does NOT fire `stopped()` / `started()`** — a save writes the component to disk without touching the lifecycle. There is no `saving()` / `saved()` callback.
- **`started()` runs on EVERY enable cycle, so it must be IDEMPOTENT.** It may run multiple times per station session (stop/start, enable/disable). Arm timers in exactly ONE place (`started()` only); do NOT use `atSteadyState()` as the sole arm point (see issues-and-gotchas.md §B1).
- **Tear down in `stopped()` — match every acquire in `started()` with a release in `stopped()`:** cancel `Clock.Ticket`s, call `unsubscribeAll()`, release resources. If the acquire path in `started()` is conditional, make the release in `stopped()` unconditional (null-safe cancel).
- **super order:** `super.started()` FIRST; cancel resources BEFORE `super.stopped()` (super LAST in stopped).

## Dynamic-slot lifecycle — orphan hygiene and schema migration

### Dynamic-slot orphan hygiene rule `[ev: retro module-hardening-reqexec-closed-deltas Δ5]`

If a component adds **dynamic slots** at runtime (`add(name, value)` → a non-frozen slot), its
`started()` **MUST** prune orphans — dynamic slots that the current schema no longer recognizes.

**Why orphans accumulate silently:** non-transient dynamic slots are persisted in the `.bog` and
re-instantiate on every station start. There is **NO framework prune-orphans utility** — the only
bulk removal is `removeAll()` (`BComponent.java:976`), which drops ALL dynamic slots
non-selectively. Orphans silently bloat the `DynamicTable` (memory + BOG size) with no error or
warning.

**Iteration API (safe snapshot):**

```java
// getDynamicPropertiesArray() returns a snapshot — safe to remove during the loop
for (Property p : getDynamicPropertiesArray()) {          // BComplex.java:606
    if (!isKnownSlot(p.getName())) {                      // your schema check
        try { remove(p); } catch (FrozenSlotException e) { /* skip frozen */ }
    }
}
```

- `getDynamicPropertiesArray()` returns a defensive copy; mutating during the iteration is safe.
- `remove(Property)` (`BComponent.java:957`) throws `FrozenSlotException` on a frozen slot —
  guard with try/catch or check `p.isDynamic()` first.
- `removeAll()` (`:976`) removes ALL dynamic slots; use only when a full reset is intended.

**Mark ephemeral dynamic slots `TRANSIENT`:** if a dynamic slot carries only in-memory state that
need not survive a restart, create it with `Flags.TRANSIENT` (`0x00000002`) — the BOG encoder
skips transient slots (`Flags.java:184,389`), so they are never persisted and never become orphans.

**Proposed lint candidate:** `dynamic-slot-orphan-prune` — flag a component that calls `add(name,
value)` without a corresponding orphan-prune loop in `started()`.

`[ev: corpus B1137]`

### `started()` slot-migration recipe (schema-evolution) `[ev: retro module-hardening-reqexec-closed-deltas Δ6]`

To migrate a renamed or restructured dynamic slot across module versions — without an OUTAGE — use
this pattern in `started()`:

```java
@Override
public void started() throws Exception {
    super.started();
    // Detect the orphan by old name (null if already migrated or never existed)
    Property orphan = getProperty("oldSlotName");         // BComplex.java:527
    if (orphan != null && orphan.isDynamic()) {           // Slot.java:59
        Object oldValue = get(orphan);                    // BComplex.java:666
        set(newSlotProperty, (BValue) oldValue);          // BComponent.java:854 — copy to new frozen slot
        remove(orphan);                                   // BComponent.java:957 — prune the orphan
    }
}
```

**Contract:**
- `getProperty(name)` returns `null` after the first migration — the block is **idempotent** (no-op
  on subsequent enable cycles).
- `started()` runs on every enable cycle (station start, component enable, re-enable after a fault),
  so idempotence is a requirement, not just a convenience.
- For **many slots**, snapshot `getDynamicPropertiesArray()` FIRST — never remove a slot while
  iterating a live cursor.
- **Type-guard the copy** when the migration also changes the slot's value type; a direct cast
  without checking will throw `ClassCastException` silently if the stored type differs.

**Pairing with the orphan-hygiene rule:** run the migration step BEFORE the orphan-prune loop so
that a renamed slot's value is safely copied before it is pruned as "unknown".

`[ev: corpus B1138]`

---

## BProgram / BRobotCode scripting SPI `[ev: corpus B-program]`

`BProgram` is a `BComponent` that hosts a signed `BProgramCode` (which hosts `BRobotCode`). The execution entry point is `BProgramService`:

```java
// Look up the service and run a robot synchronously (blocks until done):
BProgramService svc = (BProgramService) Sys.getService(BProgramService.TYPE);
BRobotResult result = svc.runRobot(myRobotCode);  // superuser only

// Or run a batch routine (returns a log string):
BString log = svc.runBatchRoutine(myBatchRoutine);
```

Security model:
- **`BRobotCode` class bytes must be signed** — `BCode.newInstance()` verifies the class signature via `CertUtils` / `SigningUtil` before loading. An unsigned class blob is rejected at instantiation with `CertificateNotTrustedException`.
- **`compactProfile` = `"compact3"` (default)** — the sandbox ClassLoader enforces the Java SE compact3 API subset; classes that reference SE APIs outside compact3 fail to load.
- **`allowProgramRuntimeExec` gate** — `BProgramService` has a `HIDDEN` boolean property `allowProgramRuntimeExec` (default `false`). When `false`, the sandbox SecurityManager denies `Runtime.exec()` calls; when `true`, the station administrator has explicitly opened that gate. Leave it `false` for normal automation scripts.
- **`doRunRobot` checks `cx.getUser().getPermissions().isSuperUser()`** — only a super-user may invoke `runRobot`; all other users get `PermissionException`.
- **`BProgramService` is `BIRestrictedComponent`** — it can only be placed under `/Services` (see `§BIRestrictedComponent` below). `[ev: corpus B-program]`

## BBatchRoutine mass-edit SPI `[ev: code BRenameBatchRoutine.java]`

`BBatchRoutine` is the SPI for mass-edit operations that apply a transformation to a list of target components. Distinct from `BJobStep` (which is part of a sequenced job) — a batch routine is submitted as one atomic action against a `BOrdList` of targets.

```java
// 1. Subclass BBatchRoutine — declare your config slots + implement run():
public class BMyBatchRoutine extends BBatchRoutine {
    // @NiagaraProperty config slots here (e.g. String find, String replace)
    public static final Type TYPE = Sys.loadType(BMyBatchRoutine.class);

    @Override
    public void run(BComponent component, PrintWriter log, Lexicon lex, Context cx) {
        // called once per resolved target; log.println() for the report;
        // throw to skip and continue (runAll catches per-target exceptions)
    }
}

// 2. Populate targets and submit via ProgramService:
BMyBatchRoutine routine = new BMyBatchRoutine();
routine.setTargets(BOrdList.make(new BOrd[]{ ... }));
BProgramService svc = (BProgramService) Sys.getService(BProgramService.TYPE);
BString report = svc.runBatchRoutine(routine);  // returns the log as a string
```

Contract:
- **`targets: BOrdList`** (the one property inherited from `BBatchRoutine`) holds the component ORDs to iterate.
- **`runAll(BObject base, PrintWriter log, Context cx)`** resolves each ORD against `base`, calls `run(component, log, lex, cx)` for each; errors per target are logged but do not abort the remaining targets.
- **Abstract `run(BComponent, PrintWriter, Lexicon, Context)`** is the only method you must implement; log each action to `PrintWriter` so the returned report is useful.
- **`runBatchRoutine` is `HIDDEN` on `BProgramService`** (flags = 4); invoke it programmatically, not from a Workbench action. `[ev: code BRenameBatchRoutine.java]`

## BEmailService.send(BEmail) `[ev: code BEmailService.java]`

Send an email from a `BSimpleJob` or service method:

```java
BEmailService emailSvc = (BEmailService) Sys.getService(BEmailService.TYPE);

BEmail email = new BEmail();
email.setAddressList("ops@example.com");
email.setSubject("Alarm notification");
email.setTextPart("Sensor fault detected on room 3.");
// Optional: email.setBlobPart(BBlob.make(pdfBytes));

emailSvc.send(email);  // queues the message; delivery is async via the outgoing account
```

Rules:
- **`BEmailService` is license-gated** — `getLicenseFeature()` returns `"tridium:email"`. If the license is absent the service starts but `doSend` throws at delivery time.
- **`send(BEmail)` is an `@NiagaraAction`** — it calls `invoke(send, email, null)` internally; the actual delivery runs in `doSend(BEmail, Context)` through the first registered `BOutgoingAccount`.
- **Exactly one `BOutgoingAccount` child must be configured** under the service; if none exists `doSend` throws `BajaRuntimeException`.
- **`BEmailService` is `BIRestrictedComponent`** — it may only be placed under `/Services`; any other parent rejects the add with a framework diagnostic (see `§BIRestrictedComponent` below). `[ev: code BEmailService.java]`

## BIRestrictedComponent — placement constraint `[ev: code BEmailService.java]`

Implement `BIRestrictedComponent` on any service or component that must only live in a specific parent context (e.g., directly under `/Services`). The framework calls `checkParentForRestrictedComponent(parent, cx)` on every `add` — a mis-drop fails with a diagnostic before the component is attached.

```java
public class BMyRestrictedService extends BAbstractService
        implements BIRestrictedComponent {

    @Override
    public final void checkParentForRestrictedComponent(BComponent parent, Context cx) {
        // Delegate to the static helper; it throws a descriptive runtime exception
        // if parent is not the expected container type.
        BIRestrictedComponent.checkParentForRestrictedComponent(parent, this);
    }
}
```

Rules:
- **Always delegate to the static `BIRestrictedComponent.checkParentForRestrictedComponent(parent, this)`** unless you need a custom placement rule — the static form checks that `parent` is a `BIService` container (i.e., the `/Services` subtree).
- **The check is `final` by convention** — mark `checkParentForRestrictedComponent` as `final` so subclasses cannot accidentally remove the constraint.
- **No explicit placement description needed** — the framework exception message names the required parent type automatically.
- Used by: `BEmailService`, `BProgramService`, and any service whose mis-placement would cause a silent malfunction. `[ev: code BEmailService.java]`

## Provider-in-service pattern `[ev: code BWeatherService.java]`

When a service needs to manage a growable set of typed children that each perform periodic or on-demand work, use the provider-in-service shape: the service holds a `BFolder` of typed provider children and dispatches work to them via `CoalesceQueue` + `Worker` + a periodic `Clock.Ticket`.

```java
public class BWeatherService extends BAbstractService {
    // slots: updatePeriod (BRelTime), provider children live in the default BFolder
    private final CoalesceQueue queue = new CoalesceQueue();
    private Worker worker;
    private Clock.Ticket updateTicket;

    @Override
    public void serviceStarted() throws Exception {
        if (worker == null) {
            worker = new Worker(queue);
            worker.start("WeatherService");   // names the background thread
        }
    }

    @Override
    public void stationStarted() {
        if (getEnabled()) {
            updateTicket = Clock.schedulePeriodically(
                this, getUpdatePeriod(), updateWeatherReports, null);
        }
    }

    @Override
    public void serviceStopped() throws Exception {
        if (updateTicket != null) { updateTicket.cancel(); updateTicket = null; }
        if (worker != null) { worker.stop(); worker = null; }
        super.serviceStopped();
    }

    @Override
    public void changed(Property property, Context context) {
        super.changed(property, context);
        if (isRunning()) {
            if (property.equals(enabled) || property.equals(updatePeriod)) {
                if (updateTicket != null) { updateTicket.cancel(); updateTicket = null; }
                if (getEnabled() && getUpdatePeriod().getMillis() != 0L)
                    updateTicket = Clock.schedulePeriodically(
                        this, getUpdatePeriod(), updateWeatherReports, null);
            }
        }
    }

    public void doUpdateWeatherReports() {
        if (getEnabled()) {
            for (BWeatherReport report : getReports()) {
                // CoalesceQueue collapses duplicate-key pending work
                queue.enqueue(new Invocation(report, BWeatherReport.updateWeatherReport, null, null));
            }
        }
    }
}
```

Key decisions in this pattern:
- **`serviceStarted` vs `stationStarted`** — start the `Worker` thread in `serviceStarted` (runs early, before the station is fully up); arm the periodic `Clock.Ticket` in `stationStarted` so it fires only after all services are running.
- **`CoalesceQueue`** — collapses pending work items with the same key; if a provider is slow and a new tick fires before the previous dispatch completes, the duplicate is merged rather than queued twice. Use `Queue` (unbounded) when coalescing is not needed.
- **`changed()` re-arms the ticket** — cancels and re-creates the ticket when `updatePeriod` or `enabled` changes so the new interval takes effect immediately.
- **Provider children in a `BFolder`** — retrieve via `getChildren(BProviderType.class)` or by walking sub-folders; each provider handles its own HTTP/network call or polling logic independently.
- **`serviceStopped()` must cancel the ticket AND stop the worker** — in that order; failing to stop the worker leaves a daemon thread running after the service is torn down. `[ev: code BWeatherService.java]`

## BSimpleJob + Fox file-channel streaming `[ev: code BFoxBackupJob.java]`

To stream large output (e.g., a backup archive) to a remote file space over an existing Fox session — without opening a new connection — use `BFileChannel.write(BFoxFileStore)`:

```java
public class BFoxBackupJob extends BSimpleJob {
    // HIDDEN slots: postSessionId (String), postPath (String)

    @Override
    public void run(Context cx) throws Exception {
        BBackupService service = (BBackupService) Sys.getService(BBackupService.TYPE);

        // 1. Retrieve the caller's existing Fox session by its opaque ID
        FoxSession session = Fox.getSession(getPostSessionId());
        if (session == null) throw new Exception("Invalid fox session id");

        Object pauseToken = null;
        try {
            pauseToken = session.pauseSessionTimeout(); // prevent idle timeout during transfer

            // 2. Navigate to the file channel on the already-connected Fox session
            BFoxConnection conn = (BFoxConnection) session.conn();
            BFileChannel chan = conn.getChannels().getFileChannel();
            Context sessionCx = chan.getSessionContext();

            // 3. Permission check using the session's credentials
            if (!service.getPermissions(sessionCx).has(48 /*backup-read|backup-write*/))
                throw new PermissionException();

            FilePath path = new FilePath(getPostPath());

            // 4. Create the target file on the REMOTE file space, then open a write stream
            chan.makeFile(null, path);
            OutputStream out = chan.write(new BFoxFileStore(null, path));

            // 5. Stream the content — here a ZIP backup — directly into the Fox channel
            service.zip(this, out, true /*includeStation*/, null);

        } finally {
            session.resumeSessionTimeout(pauseToken); // always restore the timeout
        }
    }
}
```

Rules:
- **Reuse an existing Fox session, never open a new one** — `Fox.getSession(id)` finds an already-authenticated session by an opaque string ID the caller embedded in the job; the job does not authenticate, so no credentials are stored in the job itself.
- **`session.pauseSessionTimeout()` / `resumeSessionTimeout()`** — a long-running stream can exceed the session idle timeout; pause it for the duration of the transfer and restore it in a `finally` block.
- **`chan.write(new BFoxFileStore(null, path))`** returns an `OutputStream` backed by the remote file; write any `byte[]` or stream directly into it; the Fox channel handles chunking and delivery.
- **Permission check via `sessionCx`** — check permissions using the session's `Context`, not the job's `Context`; the job runs under the scheduler's context, not the remote caller's credentials.
- **`BSimpleJob` provides the thread + progress + log infrastructure** — `this.log().message(...)` records to the job's progress log; `progress(pct)` is optional but useful for large transfers. `[ev: code BFoxBackupJob.java]`

## Chunked Base64 transfer for firmware/file OTA [ev: retro honeywell-wb-rt-wb-deltas Δ5]

When an rt action can accept only small strings (e.g. a `BString` action argument with a platform cap near 4–8 KB), split the binary payload into ≈5 000-character Base64-encoded segments and invoke the action once per segment. The rt side accumulates segments, reassembles on a terminal sentinel, then Base64-decodes and writes the binary to disk.

```java
// WB side — split and send (in a BSimpleJob or manager command)
private static final int CHUNK = 5_000;

byte[] fw = Files.readAllBytes(fwPath);
String b64 = Base64.getEncoder().encodeToString(fw);
for (int i = 0; i < b64.length(); i += CHUNK) {
    String seg = b64.substring(i, Math.min(i + CHUNK, b64.length()));
    component.invoke(UPLOAD_CHUNK_ACTION, BString.make(seg), cx);
}
component.invoke(UPLOAD_CHUNK_ACTION, BString.make("END"), cx);
```

```java
// rt side — accumulate in a StringBuilder field
@Override
public BObject uploadChunk(BString chunk, Context cx) {
    if ("END".equals(chunk.getString())) {
        byte[] data = Base64.getDecoder().decode(buf.toString());
        writeFirmwareToFlash(data);  // platform-specific
        buf.setLength(0);
    } else {
        buf.append(chunk.getString());
    }
    return null;
}
```

`CHUNK` ≈ 5 000 characters leaves headroom under typical Niagara action-arg limits. For large transfers wrap the WB loop in a `BSimpleJob` so it runs off the EDT and reports progress. `[ev: corpus B1080]`

See also: `types/distribution.md §11` for the distribution-side note on this pattern.

## AtomicBoolean single-run guard for UI-triggered async jobs [ev: retro honeywell-wb-rt-wb-deltas Δ6]

When a WB button or action triggers a background job (firmware upload, device discovery, OTA), guard entry with an `AtomicBoolean` to prevent duplicate submissions from rapid clicks or concurrent manager opens:

```java
private final AtomicBoolean running = new AtomicBoolean(false);

public void startJob(Context cx) throws Exception {
    if (!running.compareAndSet(false, true)) {
        LOG.warning("Job already running — ignoring duplicate trigger");
        return;
    }
    try {
        doHeavyWork(cx);
    } finally {
        running.set(false);   // always reset, even on exception
    }
}
```

`compareAndSet(false, true)` is atomic — if two threads race, exactly one proceeds; the other returns immediately. Reset in `finally` so the guard resets even on exception. For jobs launched from a manager toolbar button, also disable the button while `running.get()` is true and re-enable in the job's completion callback. `[ev: corpus B1082]`

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

## rt point READ path — reference card (Δ4) `[ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ4]`

Reading a control point's current value/status from rt code (station-side, not `-wb`). `[ev: corpus B1141]`

- **Generic accessor:** `BControlPoint.getOutStatusValue()` returns the type-erased `BStatusValue` — use it when you only hold a `BControlPoint` reference. Concrete subtypes narrow the return type via `getOut()`: `BNumericPoint`/`BNumericWritable` → `BStatusNumeric`, `BBooleanPoint`/`BBooleanWritable` → `BStatusBoolean`, `BEnumPoint`/`BEnumWritable` → `BStatusEnum`, `BStringPoint`/`BStringWritable` → `BStatusString`. `[ev: corpus B1141 §1141.1]`
- **Typed value extraction:** `BStatusNumeric.getValue()` (or `.getNumeric()` / `point.getNumeric()`); `BStatusBoolean.getValue()` (or `.getBoolean()`); `BStatusString.getValue()`. For enum, `BStatusEnum.getValue()` returns a `BDynamicEnum`: `dyn.getOrdinal()`, `dyn.getTag()` (stable machine key, not locale-sensitive), `dyn.getDisplayTag(cx)` (locale-sensitive label). `[ev: corpus B1141 §1141.2]`
- **`BStatus.isValid()` vs `isOk()` — do not use them interchangeably:** `isValid()` passes when the DATA-QUALITY bits are clear (fails on DISABLED/FAULT/DOWN/STALE/NULL); `isOk()` passes only when ALL bits are zero, including ALARM/OVERRIDDEN/UNACKED_ALARM. A point in ALARM but otherwise communicating normally is `isValid()==true`, `isOk()==false`. **Rule for rt code:** use `isValid()` to decide whether a value is usable for control logic; reserve `isOk()` for a case that genuinely needs a fully clean status (no override, no alarm). `[ev: corpus B1141 §1141.3]`
- **`BOrd.get()` and dangling resolution:** `ord.get()` resolves against the local station root; `ord.get(myComponent)` resolves relative to `myComponent`. A dangling ORD (deleted/moved slot) throws `UnresolvedException` — always wrap a resolve in `try/catch` and log-and-skip rather than letting the exception propagate out of a control callback. `[ev: corpus B1141 §1141.5–1141.6]`

```java
// Canonical rt resolve + typed read + validity check
BOrd ord = BOrd.make("station:|slot:/Services/CompControl/setpointOut");
try {
    BObject obj = ord.get();
    BNumericPoint pt = (BNumericPoint) obj;
    if (pt.getOut().getStatus().isValid()) {
        double val = pt.getOut().getValue();
        // ... use val
    }
} catch (UnresolvedException e) {
    Sys.getLog().warning(getType(), "dangling ord: " + ord, e);
}
```
