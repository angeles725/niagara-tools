# Observability — logging, spy pages, fault surfacing, audit

Four diagnostic surfaces: **logger** (async, BQL-queryable), **spy page** (live snapshot,
adminRead), **fault slot** (wire-sheet visible), **audit trail** (attributed user-action log).

---

## 1 · Logging — `java.util.logging.Logger`

`javax.baja.log.Log` is `@Deprecated`. Use the JDK logger exclusively.

```java
// One static field per class; name = module short name, NOT the FQCN.
private static final Logger LOG = Logger.getLogger("modbusCore");
```

A `BDevice` inherits the network's logger name + device path automatically:
`device.getLogger()` returns a logger named `"<moduleShortName>.<deviceName>"` —
do not construct a separate one in the device class. [ev: code modbusCore-rt `BModbusClientPollGroup.java` `device.getLogger()`; code bacnetAws-rt `BacnetAwsServlet.java` `Logger.getLogger("bacnetAws")`]

### Level mapping

| `java.util.logging.Level` | Meaning |
|---|---|
| `FINE` / `FINEST` | Trace — normally off in production |
| `INFO` | Operational message (startup, major state change) |
| `WARNING` | Recoverable problem; auto-captured by `BLogHistoryService` for BQL alarms |
| `SEVERE` | Error |

**Guard expensive strings** at `FINE`/`FINEST`:

```java
if (LOG.isLoggable(Level.FINE)) {
    LOG.fine("poll: device=" + device.toPathString() + " bytes=" + Arrays.toString(data));
}
```

[ev: code clientCertAuth-rt `ClientCertAuthUtils.java` `if (log.isLoggable(Level.FINE))`]

**Why `System.out.println` is wrong:** bypasses `BLogHistoryService`; output lands only
in `stdout.txt`, is not BQL-queryable, not level-filtered, and unmonitored on a headless JACE.
[ev: corpus B92/B106]

**`module-permissions.xml`:** declare `java.util.logging.LoggingPermission "control"` so
station admins can change log levels at runtime via the Workbench log viewer. The Gradle
plugin inlines this into `META-INF/module.xml` at build time.

---

## 2 · Spy pages — `spy:/` diagnostic tree

The spy tree is accessible to admin users (`adminRead`). Register in `started()`:

```java
@Override
public void started() throws Exception {
    super.started();
    Spy.ROOT.add("myMod", new MyModuleSpy(this));  // key = module short name
}
```

[ev: code workbench-wb `NavMonitor.java` `Spy.ROOT.add("navMonitor", ...)`; code fox-rt `BFoxConnection.java` `Spy.ROOT.add("fox", ...)`]

### `SpyDir` + `SpyWriter` recipe

```java
private static class MyModuleSpy implements SpyDir {
    private final BMyNetwork net;
    MyModuleSpy(BMyNetwork net) { this.net = net; }

    @Override
    public void spy(SpyWriter out) throws Exception {
        out.startProps();
        out.trTitle("MyNetwork", 2);
        out.prop("CRC errors", net.totalCrcErrors);
        out.prop("Queue depth", net.queue.size());
        out.endProps();
        out.startTable(true);
        out.trTitle("Device stats", 3);
        out.w("<tr>").th("Device").th("Requests").th("NoResponse").w("</tr>\n");
        for (BMyDevice d : net.getDeviceList())
            out.w("<tr>").td(d.getName()).td(d.requests).td(d.noResponse).w("</tr>\n");
        out.endTable();
    }
}
```

`startProps`/`endProps` = two-column key-value table. `startTable(true)`/`endTable` = full HTML
table. [ev: code modbusCore-rt `BModbusNetwork.java` spy() — startProps/trTitle/prop/endProps + startTable; code knxnetIp-rt `BConnection.java`; code cloudIotHubDep-rt `BMessageQueue.java`]

### Override on `BDeviceNetwork` / `BDevice` — call `super` first

`BDeviceNetwork` and `BDevice` already implement `spy()` with framework-supplied rows
(comm state, device address). Always call `super.spy(out)` first:

```java
@Override
public void spy(SpyWriter out) throws Exception {
    super.spy(out);      // framework rows first
    out.startProps();
    // … your counters …
    out.endProps();
}
```

[ev: code modbusCore-rt `BModbusNetwork.java` and `BModbusDevice.java` — both call `super.spy(out)`]

**HogsPage:** `spy:/sysManagers/engineManager?hogs` lists top CPU consumers on the engine
thread (long `ASYNC` handlers, expensive `execute()` cycles).

---

## 3 · Fault surfacing — `appFail` / `configFail` / `configFatal`

| Method | Auto-recovery | Use |
|---|---|---|
| `appFail(msg)` | Yes, when condition clears | Runtime error (poll timeout, decode failure) |
| `configFail(cause)` | Yes, on config change | Invalid or missing configuration |
| `configFatal(cause)` | **No** — stays faulted until reset/reconfigured | License missing, irrecoverable init |

```java
try { readDevice(); statusOk(); }
catch (IOException e) { appFail("read timeout: " + device.getAddress()); }

// In started() for a license check:
if (!licensed) { configFatal("Unlicensed: " + toPathString()); return; }
```

[ev: code bacnetAws-rt `BBacnetAwsNetwork.java` `configFatal("Unlicensed: " + ...)`; code maxpro-rt `BMaxproCamera.java` `configFatal(cameraLimitFault)`]

Pair `appFail`/`configFail` with `READONLY|TRANSIENT|SUMMARY` status and `faultCause` slots
so the wire-sheet color-codes the component's health and operators see a one-line reason:

```java
@NiagaraProperty(name="status",     type="BStatus", defaultValue="BStatus.ok",
                 flags=Flags.READONLY|Flags.TRANSIENT|Flags.SUMMARY)
@NiagaraProperty(name="faultCause", type="String",  defaultValue="",
                 flags=Flags.READONLY|Flags.TRANSIENT|Flags.SUMMARY)
```

[ev: code tagdictionary-rt `BTagDictionary.java` `faultCause` String `READONLY|TRANSIENT|SUMMARY`; code devIpDriver-rt `BDdfUdpMulticastHelper.java`]

---

## 4 · Audit vs logger — decision table

| | Logger (`java.util.logging`) | Audit trail (`AuditHistory`) |
|---|---|---|
| Trigger | Internal / autonomous | User-initiated only |
| Delivery | Async, non-blocking | Synchronous |
| Storage | `BLogHistoryService` (ring buffer, BQL-queryable) | Append-only, attributed (user + before/after) |
| BQL alarm support | Yes — WARNING+ auto-appear | No |
| Framework handles it | No | **Yes** — slot writes via oBIX/Fox/Workbench are attributed automatically |

A custom servlet writing a slot out-of-band (not through a `@NiagaraAction`) must call
`Sys.getAuditor().audit(...)` explicitly. **Fire-and-forget: a failed audit must NOT fail
the write.**

```java
Auditor a = Sys.getAuditor();
if (a != null)
    a.audit(new AuditEvent("setpointChanged", component.toPathString(),
                           "setpoint", oldVal, newVal, cx.getUser().toString()));
```

[ev: code exportTags-rt `BSupervisorJoinJob.java` `auditor.audit(new AuditEvent(...))`; code bacnet-rt `BVirtualPropertyWrite.java`; corpus B167/B30/B31]

---

## 5 · Exposing runtime state — summary

Prefer these surfaces in order:

1. **`READONLY|TRANSIENT` property slots** — wire-sheet visible, linkable, serialized as
   zero/false on restart. Use for cycle counters, error counts, last-poll timestamp.
2. **`faultCause` string slot** — one-liner for the current fault; keep it actionable.
3. **Spy page** — live snapshot of internal structures too large or volatile for a slot.
4. **`WARNING`+ log messages** — auto-captured by `BLogHistoryService`; a BQL alarm rule
   can pattern-match them. Use `WARNING` for any recoverable condition operators should know.
   [ev: corpus B33; code bacnetAws-rt `BBacnetAwsNetwork.java`]

---

## Lint worth adding

**`no-System.out`** — grep `System.out.print` in `*.java` under the module source tree:

```sh
find "$SRC" -name "*.java" | xargs grep -l "System\.out\.print" \
  && echo "HARD: System.out bypasses BLogHistoryService — use java.util.logging.Logger"
```

[ev: corpus OBS-2]

---

---

## Credential header redaction in logging [ev: retro honeywell-wb-rt-wb-deltas Δ7]

Redact `Authorization`, `X-Api-Key`, and similar credential headers **at every log level, including `FINEST`**. Cloud/HTTP connectors that log the full `HttpURLConnection` request properties at FINEST leak Bearer tokens to the station log history, where they are visible to anyone with `BLogHistoryService` read access.

```java
private static final Set<String> REDACT_HEADERS = new HashSet<>(Arrays.asList(
    "Authorization", "X-Api-Key", "X-Auth-Token"));

// Build a safe copy before logging
Map<String, List<String>> safe = new LinkedHashMap<>();
for (Map.Entry<String, List<String>> e : conn.getRequestProperties().entrySet()) {
    safe.put(e.getKey(),
             REDACT_HEADERS.contains(e.getKey())
                 ? Collections.singletonList("***")
                 : e.getValue());
}
if (LOG.isLoggable(Level.FINEST)) LOG.finest("request headers: " + safe);
```

Add this guard to any code that iterates HTTP headers or logs connection state — even at FINEST, because a support engineer enabling fine tracing must not accidentally capture credentials. See `types/security.md §3.3` for the full security context. `[ev: corpus B1082]`

## AtomicBoolean guard for UI-triggered async jobs — observability note [ev: retro honeywell-wb-rt-wb-deltas Δ6]

When a WB button or manager command launches a background job, use an `AtomicBoolean` to prevent duplicate submissions and log the guard firing at `WARNING` so duplicate triggers are visible in the log history:

```java
if (!running.compareAndSet(false, true)) {
    LOG.warning("Job already running — ignoring duplicate trigger from user " +
                (cx != null ? cx.getUser() : "unknown"));
    return;
}
```

Log the guard trip at `WARNING` (not FINE): `BLogHistoryService` captures WARNING+ automatically, so a repeated trigger shows up in the station log without requiring fine tracing to be enabled. See `types/logic-authoring.md §AtomicBoolean single-run guard` for the full recipe. `[ev: corpus B1082]`

---

**See also:** `types/actions.md` (`Flags.NO_AUDIT` on internal callback actions),
`types/logic-authoring.md` (`BHistoryExt` point logging), `types/driver-authoring.md`
(`configFail`/`doPing`, poll health counters).
