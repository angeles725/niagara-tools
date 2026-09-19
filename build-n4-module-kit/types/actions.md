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
void doTick() { /* your logic */ }                  // YOU write this — the handler
```

Baja resolves the handler by reflection as `do<Cap>(paramType)` and, if present, the overload
`do<Cap>(paramType, Context)` — the Context form gives you the invoking user/session (needed
for auditing and per-user privilege). `EngineUtil.doInvoke` tries both signatures.

**Gotcha:** the rule applies to EVERY action, parameterized or not. Name the handler wrong
(`myAction()` instead of `doMyAction()`) and the action dispatches to nothing — a **silent
no-op**, no error. [ev: code slot-o-matic `getMethod("do"+cap, paramType[, Context])`; corpus authoring-exemplars]

## 2 · Threading model — ordinary vs `ASYNC`

- An **ordinary** action runs **synchronously on the invoker's thread** (the servlet thread,
  the linking component's thread, the Workbench Fox thread).
- An **`ASYNC`** action is **coalesced onto the single shared engine thread** and run there.

**Gotcha:** every timer callback and every `ASYNC` action in the VM shares that one engine
thread. Blocking inside an `ASYNC` action (HTTP call, disk I/O, `Thread.sleep`, a lock) stalls
**all** timers and other async actions station-wide. Do blocking work on a `BWorker`/off-engine
thread, never in an `ASYNC` handler. [ev: devguide `execution.txt`; code `flags=Flags.ASYNC` on escalateAlarms/cleanup]

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
against a fat-finger invoke. [ev: code `Flags.java` CONFIRM_REQUIRED=0x80; real uses resetStatistics/changeDeviceUuid]

**`NO_AUDIT`** belongs on internal callback actions (a `HIDDEN|ASYNC` timer tick that fires
every cycle). Without it, a per-minute callback **floods the audit trail**. Tridium's own code
consistently sets `HIDDEN|NO_AUDIT|ASYNC` on internal callbacks (escalateAlarms, expire,
retrieveResults). [ev: code `Flags.java` NO_AUDIT=0x800]

Timer-callback actions are conventionally `HIDDEN|ASYNC` (+ `NO_AUDIT`). [ev: corpus slots-flags-java8]

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
  errors at build. A `BStruct` parameter renders in Workbench as a multi-field argument dialog.
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
