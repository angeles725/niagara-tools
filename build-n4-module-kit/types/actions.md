# Actions — `@NiagaraAction`, flags, threading, topics

How to author component actions correctly. The kit's slot docs cover properties well; this
doc is the missing companion for the ACTION side. Format per entry: what it is / how / gotcha.

---

## 1 · The `doX()` dispatch contract (the general law)

A `@NiagaraAction(name = "myAction")` generates a public *invoke wrapper* `myAction()` that
calls `invoke(myAction, arg, cx)`. The framework then dispatches to a **handler method you
write**, named `do` + Capitalized(action name):

```java
@NiagaraAction(name = "tick", flags = Flags.HIDDEN)
public void tick() { invoke(tick, null, null); }   // GENERATED wrapper — do not hand-write
public void doTick(Context cx) { /* your logic */ }  // YOU write this — the handler: public, takes Context
```

The type introspector resolves the handler when the type loads: `do<Cap>()` / `do<Cap>(P)` first, and only if
that is missing `do<Cap>(Context)` / `do<Cap>(P, Context)`. The handler must be public, non-static and return
the wrapper's type. [CERT] `com/tridium/sys/schema/ComplexIntrospector.java:316-343` (4.14 decompile, corpus B1231 §1231.4)

**Gotcha — a wrong frozen handler is a type-load error, not a no-op:** a missing or mis-named `do` method fails
the type with `Missing action do method`; a non-public one with `Do action method must be public`; a different
return type with `Action method and do method have mismatched signature`. A silent no-op is possible
at most on a reflectively dispatched DYNAMIC action [INFER].
[ev: retro n4-course-slots-components-contracts Δ1] [ev: corpus B1231 §1231.4]

**Declare ONE handler and take the `Context`** (Tridium's recommendation: the Context carries the invoking user,
needed for audit and per-user checks). If both `doX()` and `doX(Context)` exist, the no-Context one is found
first and the Context one never runs. A check for that pair is owed as Go lint G1 (BUILD-STATE `open_issues`).
[ev: corpus B1231 §1231.4]

## 2 · Threading model — ordinary vs `ASYNC`

- An **ordinary** action runs **synchronously on the invoker's thread** (the servlet thread,
  the linking component's thread, the Workbench Fox thread).
- An **`ASYNC`** action is **coalesced onto the single shared engine thread** and run there.

**Gotcha:** every timer callback and every `ASYNC` action in the VM shares that one engine
thread. Blocking inside an `ASYNC` action (HTTP call, disk I/O, `Thread.sleep`, a lock) stalls
**all** timers and other async actions station-wide. Do blocking work on a `BWorker`/off-engine
thread, never in an `ASYNC` handler. [ev: devguide `execution.txt`; code `flags=Flags.ASYNC` on escalateAlarms/cleanup]

**Which thread, exactly** (baja and fox-rt 4.14.0.162 vineflower; 4.15 not verified) `[ev: retro n4-course-threading-jobs-doctrine Δ3]` `[ev: corpus B1238 §1238.3-§1238.5]`:
- A plain action invoked from Workbench runs inline on that session's ONE serial `Fox:Dispatcher:<hash>` thread
  (`fox/session/SessionDispatcher.java:16,46-66`; `ComponentSlotMap.invoke` calls the `do` method on the caller's
  thread, `:1315-1340`). A `do` method that blocks freezes every later request of that session; the client gives
  up after `Fox.requestTimeout` = 60 s (`fox/session/Fox.java:28`) while the station thread stays pinned.
- An `ASYNC` action's default `post()` enqueues on `Nre:Engine`, returns nothing and only logs an exception.
- **The two coalescing winners are opposite** `[ev: retro devguide415-r2-runtime-contracts-formats Δ1]` `[ev: corpus B1316 §1316.1]`:
  the engine's `ActionQueue` keeps the FIRST pending entry per (component, action) and drops a later post with
  another argument (`com/tridium/sys/engine/ActionQueue.java:26-28,80-88`: `equals` ignores the argument), so post a
  no-argument "recalculate" and read the current slots inside it; a `CoalesceQueue` of `Invocation`s keeps the LAST
  one (`Invocation.coalesce` returns the newcomer, `javax/baja/util/Invocation.java:49-50`).
- **A queued ASYNC action runs with a null Context** (`EngineManager.java:306-310`: `doInvoke(…, entry.arg, null)`):
  never read `cx.getUser()` there; check permissions before posting (`types/security.md` § 2.4).
- **The sanctioned offload** for slow work is an `ASYNC` action whose `post()` override hands
  `new Invocation(this, action, arg, cx)` to the module's own executor or worker, which calls `invoke()` on it. A
  bare `this::doX` lambda skips the invoke path: no links, no action event, no audit, no Context. A bounded executor
  rejects work when it is full or shut down (`RejectedExecutionException` under the JDK default policy): catch it in
  `post()` and report it (fault the status and log), or fall back with `return super.post(action, arg, cx);`, which
  queues it on the engine's action queue (vineflower baja 4.14.0.162 `javax/baja/sys/BComponent.java:590`). Never
  call `invoke()` from `post()`: that runs the slow work on the caller's thread. Never let the exception escape. (Owed Go lint G28
  `async-blocking-action`.)

## 3 · Action flags

| Flag | Hex | Effect |
|---|---|---|
| `HIDDEN` | 0x04 | not shown in the Workbench Actions menu; not exported as an oBIX `<op>` |
| `ASYNC` | 0x10 | coalesce + run on the shared engine thread (see §2) |
| `OPERATOR` | 0x100 | invocable with operator-level permission (else admin-invoke is required) |
| `CONFIRM_REQUIRED` | 0x80 | Workbench shows a confirmation dialog before invoking |
| `NO_AUDIT` | 0x800 | suppress the action-invocation audit entry |

**`CONFIRM_REQUIRED`** is the right guard for destructive / irreversible actions — a
`faultReset`, `resetStatistics`, `clearLastTimestamp`, `changeDeviceUuid`. Zero-cost safety
against a fat-finger invoke. [ev: code `Flags.java` CONFIRM_REQUIRED=0x80; code driver-rt javax/baja/driver/util/BPollScheduler.java:136 resetStatistics; code bacnet-rt javax/baja/bacnet/export/BLocalBacnetDevice.java:609 changeDeviceUuid]

**`NO_AUDIT`** belongs on internal callback actions (a `HIDDEN|ASYNC` timer tick that fires
every cycle). Without it, a per-minute callback **floods the audit trail**. Tridium's own code
consistently sets `HIDDEN|NO_AUDIT|ASYNC` on internal callbacks (escalateAlarms, expire,
retrieveResults). [ev: code `Flags.java` NO_AUDIT=0x800]

Timer-callback actions are conventionally `HIDDEN|ASYNC` (+ `NO_AUDIT`). [ev: code alarm-rt `BAlarmService.java:168` escalateAlarms = `newAction(2068)` = HIDDEN|ASYNC|NO_AUDIT]

## 4 · Typed arguments and return values

Declare a parameter and/or return type on the annotation:

```java
@NiagaraAction(
    name = "ackAlarm",
    parameterType = "BAlarmRecord",
    defaultValue = "new BAlarmRecord()",   // REQUIRED whenever parameterType is set
    returnType = "BBoolean")
public BBoolean ackAlarm(BAlarmRecord r) { return (BBoolean) invoke(ackAlarm, r, null); }
public BBoolean doAckAlarm(BAlarmRecord r) { ... }
```

- `parameterType` / `returnType` are ORD type spec strings (`"baja:RelTime"`, `"BAlarmRecord"`).
- **`defaultValue` is mandatory when `parameterType` is present** — omit it and slot-o-matic
  errors at build. A `BStruct` parameter renders in Workbench as a multi-field argument dialog [INFER].
[ev: devguide `slot-o-matic.txt`; code ackAlarm(BAlarmRecord)]

## 5 · Actions vs `@NiagaraTopic`

An **action** is behavior invoked by a command or a link. A **topic** is an *event source*:
no storage, no behavior — it is the SOURCE side of a trigger link, fired in code with
`fire(topic, event, cx)`.

```java
@NiagaraTopic(name = "defrostStarted", eventType = "BStatusBoolean", flags = Flags.SUMMARY)
public Topic defrostStarted() { return defrostStarted; }
// in your logic:
fire(defrostStarted, evt, null);   // cx = null is the normal case for autonomous events
```

Choose a **topic** when many listeners should react to one event (let links drive their target
actions); choose an **action** when a caller issues a direct command. A control component that
wants downstream reaction to "defrost started" should fire a topic, not expose an action callers
must know to invoke. [ev: devguide `execution.txt`; corpus B538; code report-rt BReportSource newTopic()/fireOut()]

## 6 · Serving an action over the wire — surfaces differ

The same logical "call a server method" maps to **different surfaces** with different RBAC and
consumers:

- **`@NiagaraAction`** → invoked over oBIX `<op>` POST, Fox, and the Workbench Actions menu;
  routed through `ObixUtils.serviceInvoke` → `BComponent.invoke`; admin-invoke by default
  unless `OPERATOR`.
- **`@NiagaraRpc`** → the bajaux/SPA path over `/rpc` (JSON POST, CSRF-gated, session auth);
  no servlet subclass needed; the method registry a bajaux UI view calls.

**Gotcha:** a custom `-ux` view that needs a server-side callable should use `@NiagaraRpc`
(the bajaux-native, CSRF-gated path), NOT `@NiagaraAction` (which is oBIX/Fox-facing). Mixing
them up yields either a missing CSRF gate or a non-discoverable action. See `types/dashboard.md`
for the serving-recipe decision table. [ev: corpus B507; B822]

---

**See also:** `types/logic.md` (slot flags, BStatus), `types/logic-authoring.md` (lifecycle,
BQL, ORD), `types/security.md` (per-action RBAC, audit), `types/dashboard.md` (@NiagaraRpc serving).
