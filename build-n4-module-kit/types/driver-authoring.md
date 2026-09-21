# Driver authoring — network/device/proxy SPI reference

Custom Niagara driver in N4: four classes to subclass, one comm engine to wire, one tuning policy
to configure. This doc is the author-side reference; protocol-specific wire format is out of scope.

---

## 1 · Class ladder — which base to extend

### 1.1 · The four required subclasses

| Class | Your subclass base | What you override |
|---|---|---|
| Network | `BDeviceNetwork` / `BBasicNetwork` / `BLoadableNetwork` (§1.2) | `getDeviceType()`, `getDeviceFolderType()` |
| Device | `BDevice` / `BBasicDevice` / `BLoadableDevice` | `getNetworkType()`, `doPing(cx)` |
| Point folder | `BPointDeviceExt` | `getProxyExtType()` |
| Proxy ext | `BProxyExt` | `getMode()`, `readSubscribed(cx)`, `readUnsubscribed(cx)`, `write(cx)` |

`BProxyExt` is a point extension (`implements BAbstractProxyExt`) — it sits as a child of a
`BControlPoint`, not as a standalone component. [ev: code BProxyExt.java:120; corpus B810 §810.1]

### 1.2 · Which network base?

| Base | When to use |
|---|---|
| `BDeviceNetwork` | Raw: you own the full comm layer from scratch |
| `BLoadableNetwork` | Same as `BDeviceNetwork` but adds upload/download SPI (`BUploadParameters`/`BDownloadParameters`) for config round-trips to the device |
| `BBasicNetwork extends BLoadableNetwork` | Add this when you need the basicDriver request/response dispatch engine (§3): `makeComm()`, transaction retry, three Niagara workers, coalescing write queue. 12 Tridium drivers ride it. |
| `BSerialNetwork extends BBasicNetwork` | Narrows to serial: adds `serialPortConfig` (`BSerialHelper`), `interMessageDelay`, and a `SerialRcv` rx thread. Use for RS-485/TTY fieldbus. |

**Gotcha:** `BBasicNetwork` ships **no TCP transport**. TCP is each driver's own `Comm` subclass.
[ev: corpus B517 §517.5]

### 1.3 · WB device-manager presence rule and plugin-extensible SPI [ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ1]

**Every `BBasicNetwork` driver ships a `-wb` device manager (WB-presence rule, PD-01):** a driver that adds a `BBasicNetwork` subclass MUST register a WB view via `@AgentOn` on the network or device type — at minimum a `BAbstractManager` or `BWbComponentView`. Relying on the default property-sheet view forces operators into raw slot editing. See `types/wb-widgets.md §Vendor-grade -wb UX patterns`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ1]` `[ev: corpus B1057]`

**Driver modules with NO Hx/UX intent MUST state that explicitly (PD-13):** when a driver module deliberately ships no `-ux` profile and relies entirely on the base Niagara hx-wb for its browser view, add a comment in the module's `SKILL.md` / README: `# UX: none — relies on base hx-wb`. This prevents a future developer from adding a redundant `-ux` scaffold when the driver's intent is station-side only. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ13]` `[ev: corpus B1061]`

### 1.4 · Plugin-extensible manager SPI — multi-device-family alternative [ev: retro honeywell-wb-rt-wb-deltas Δ1]

When multiple device families share one `BAbstractManager` container (wb side), avoid subclassing the manager once per family. Instead, implement a **plugin SPI** (`BIHonDeviceModel` or equivalent `BIHonBacnetDeviceModel`) per family module; the shared framework discovers all registered implementations via `NiagaraRegistryUtil.getImplementersOfTypeSpec()`.

- Each SPI implementer declares `getSupportedModelSpecs()`, `createColumns()`, and `createCommands()`.
- Register via a plain `<type>` in `module.xml` — no `@AgentOn`.
- Prefer this over a Manager subclass when the manager framework already exists and you are contributing a new device-family module on top of it.

Full wb-side recipe in `types/wb-widgets.md §Plugin-extensible device-type manager via SPI`. `[ev: corpus B1077]`

---

## 2 · `BProxyExt` point SPI

Three overrides — the framework calls them, never call them yourself:

```java
// Called once when the first subscriber (Workbench panel / BOX client) watches this point.
// Register a native COV subscription or push the point into the poll scheduler.
@Override
protected void readSubscribed(Context cx) throws Exception {
    getNetwork().getPollScheduler().subscribe(this);   // or COV register
}

// Called for a one-shot read with no active subscriber (e.g. a "refresh" from Workbench).
@Override
protected void readUnsubscribed(Context cx) throws Exception {
    getNetwork().sendAsync(buildPollRequest(), cx);
}

// Called when our control logic BLinks a value into a proxy writable point.
// NEVER block here — post to an async worker instead (§3 CoalesceQueue).
@Override
protected boolean write(Context cx) throws Exception {
    getNetwork().postWrite(buildWriteRequest(cx));
    return false;   // false = async; the framework does not wait
}
```

Callback pair: `readOk(value)` / `readFail(cause)` and `writeOk()` / `writeFail(cause)` — call
these from your comm thread to update the point's `out` slot. [ev: code BProxyExt.java:859,883,895;
corpus B810 §810.3]

**Gotcha:** a write to a DOWN/FAULT/DISABLED device is **silently dropped** — `Tuning.write()`
calls `isOperational()` first and returns without queuing if false. The write is NOT retried unless
`BTuningPolicy.writeOnUp=true` (default true). [ev: code Tuning.java:332-333; corpus B810 §810.4]

---

## 3 · `BBasicNetwork` comm layer

```
makeComm() → Comm { CommReceiver rx + CommTransmitter tx + CommTransactionManager }
                        ↑ unsolicited frames          ↑ request→response tag matching
```

Three Niagara worker threads the network owns:

| Worker | Purpose |
|---|---|
| `worker` | solicited poll dispatching |
| `writeWorker` | write requests; uses a `CoalesceQueue(1000)` so rapid setpoint changes collapse |
| `dispatcher` | unsolicited-frame fan-out |

Retry loop lives in `Comm.transmit(msg, timeout, retryCount)`: sends → waits `responseTimeout`
on a `CommTransaction` → retries `retryCount+1` times. [ev: corpus B517 §517.2]

Send API:

```java
network.sendAsync(request, cx);          // reads — never block
network.sendAsyncWrite(request, cx);     // writes — routes through writeWorker
```

**Gotcha:** `sendSync` exists but blocks the calling thread. Call it only from your own background
`BWorker`, never from the engine thread or a timer callback. [ev: corpus B517 §517.2]

---

## 4 · `BTuningPolicy` map

Place a `BTuningPolicyMap` named `tuningPolicies` on your network. The map always contains a
`defaultPolicy` child; add more by name. Per-point binding: `BProxyExt.tuningPolicyName` (String,
default `"defaultPolicy"`).

| Slot | Default | Effect |
|---|---|---|
| `minWriteTime` | 0 (off) | Minimum gap between device writes; closer writes suppressed |
| `maxWriteTime` | 0 (off) | Heartbeat: force re-write even if value unchanged |
| `staleTime` | 0 (off) | Mark point STALE if no successful read within this window |
| `writeOnStart` | **true** | Write on station/point start |
| `writeOnUp` | **true** | Re-write when device returns UP after comm loss |
| `writeOnEnabled` | **true** | Write when point re-enabled |

[ev: code BTuningPolicy.java:19,24,29,33,41,47-49; corpus B872 §872.2]

**Gotcha:** the Tuning scan thread runs at `min(minWriteTime/2, maxWriteTime/2, staleTime/2)`
clamped to `[200 ms, 20 000 ms]`. With all-zero defaults the scan fires every 20 s — this is
correct; the policy kicks in only when specific slots are set. [ev: corpus B872 §872.2]

---

## 5 · Health — `doPing` and fault reporting

```java
@Override
public BObject doPing(Context cx) throws Exception {
    try {
        boolean ok = comm.sendPing();          // protocol-specific
        if (ok) return pingOk();               // clears DOWN bit, fires offnormal→normal alarm
        else    return pingFail("no response"); // sets DOWN bit on device + all its proxy points
    } catch (IOException e) {
        return pingFail(e.getMessage());
    }
}
```

`pingFail` propagates DOWN to every `BProxyExt` under the device via:
`BDevice.updateStatus()` → `BPointDeviceExt.updateStatus()` → `BProxyExt.updateStatus()` sets
`BStatus.DOWN`. [ev: code BDevice.java:400; BProxyExt.java:475; corpus B810 §810.2]

### `configFatal` vs `configFail` — recovery reference card

| Call | Status bit | Recovery | Cleared by |
|---|---|---|---|
| `configFail(cause)` | FAULT | **Transient** — cleared on next successful operation | `configOk()` |
| `readFail(cause)` | FAULT | Transient | `readOk(value)` |
| `writeFail(cause)` | FAULT | Transient | `writeOk()` |
| `configFatal(cause)` | FAULT + DISABLED | **Permanent** — cleared ONLY by a full station restart | station restart |

**`configFatal` is permanent:** `configOk()` clears `configFault` but returns early before
`fatalFault` (`BDevice.java:425,431`); no code path resets `fatalFault`; `checkFatalFault()`
short-circuits component restart (`:577-578`); the `FATAL_FAULT` flag is locked until station stop.
The following do NOT clear a fatal fault: `configOk()`, a config-slot change, a re-ping, an enable
cycle, or a component restart. A point in `fatalFault` is permanently unoperational — tuning stops
dispatching to it.

**Valid uses for `configFatal`:** licensing failure, invalid parentage. Do NOT call `configFatal`
for transient comm conditions (device unreachable, timeout, protocol error) — those belong to
`configFail()` / `pingFail()` so the point self-recovers when the device returns.

**Neither `configFail` nor `configFatal` sets `DOWN`** — `DOWN` is exclusively ping-owned
(`pingFail()` propagates DOWN to every `BProxyExt` under the device).

`[ev: retro module-hardening-reqexec-closed-deltas Δ4]`

---

## 6 · Poll — rate buckets and subscription lifecycle

Implement `BIPollable` (or `BIBasicPollable` for basicDriver networks) on your proxy ext and
override `getPollFrequency()`:

```java
@Override
public BPollFrequency getPollFrequency() {
    return BPollFrequency.normal;   // fast=1000ms / normal=5000ms / slow=30000ms
}
```

| Bucket | `BPollFrequency` | Default rate |
|---|---|---|
| fast | `fast` (ordinal 0, DEFAULT) | 1 000 ms |
| normal | `normal` (ordinal 1) | 5 000 ms |
| slow | `slow` (ordinal 2) | 30 000 ms |
| dibs | immediate (LIFO priority stack) | — |

[ev: code BPollFrequency.java:12,18-21; BPollScheduler.java:23-31,145; corpus B872 §872.1]

Lifecycle rule: **register in `readSubscribed`, unregister in `readUnsubscribed`**. The scheduler
pushes the point onto the `dibs` stack immediately on subscribe for a first-poll before the bucket
cadence starts. [ev: corpus B872 §872.1]

One poll thread per network, named `"Poll:<networkName>"`. `pollEnabled` on the network gates the
entire loop. [ev: corpus B872 §872.1; corpus B810 in `logic-authoring.md §Authoring a driver`]

---

## 7 · Advanced — history, schedule, virtual, serial

### 7.1 · `BHistoryDeviceExt` + `BHistoryPollScheduler`

A separate `BHistoryPollScheduler` (default rates: fast 10 000 / normal 45 000 / slow 120 000 ms)
drives `BHistoryImport` children that read records from the field device and append them to the
local history DB. Drop a `BHistoryDeviceExt` on your `BDevice` class to opt in.
[ev: code BHistoryPollScheduler.java:36-38; corpus B872 §872.3]

### 7.2 · `BScheduleDeviceExt` — makeExport / makeImportExt

Abstract; implement both abstract methods to make your driver push/pull schedules.
`processImport` (export down to device) is version-gated: only pushes when `lastModified` differs
from the device's cached version. `subscribeWindow` staggers the initial subscribe to spread
network load. [ev: code BScheduleDeviceExt.java:34; corpus B872 §872.4]

### 7.3 · `BVirtualComponent` / `BVirtualGateway` (virtual: scheme)

`BVirtualGateway extends BComponent` exposes a lazy, transient-only `BVirtualComponentSpace`.
Points inside resolve via `virtual:/gw/device/point` ORDs — NOT `slot:`. The space is read-only
and not persisted; writes do not survive GC. Use for on-demand browsing of large address spaces
(e.g. a BACnet router with thousands of objects) where creating a full proxy tree would be
prohibitive. [ev: corpus B28 §28.9-28.12]

### 7.4 · `BSerialNetwork` / `SerialComm` — RS-485 twists

`BSerialNetwork extends BBasicNetwork` adds `serialPortConfig` (`BSerialHelper`:
baud/data/stop/parity/port) and `interMessageDelay`. `SerialComm.performInterMessageDelay()`
sleeps the deficit since the last received message (min 10 ms) before each transmit — a
fixed-ms gap, NOT the baud-relative silence prescribed by Modbus RTU spec.
[ev: code BSerialNetwork.java:51; corpus B517 §517.3]

---

## 8 · Transport-specific twists — KNX and Z-Wave

### 8.1 · KNX (`eibnetIp` driver)

```java
public class BEibnetIpNetwork extends BLoadableNetwork implements EibnetConst, BIService
```

KNX addressing is by **group address**, not by object/property (unlike BACnet). Points are
organized under `BGroupAddressFolder` children. The network owns two stack layers as
`@NiagaraProperty` children: `linkLayer` (`BEibnetIpLinkLayer`) for UDP multicast/unicast
EIBnet/IP discovery, and `tunnelLayer` (`BEibnetIpTunnelLayer`) for the tunneled connection.
Per-datatype proxy families: `BKnxBooleanProxyExt`, `BKnxEnumProxyExt`, `BKnxStringProxyExt` —
pick by KNX DPT, not by a single generic proxy ext. ETS project import available via
`BKnxDiscoverDevicesJob`. [ev: code BEibnetIpNetwork.java:63-77; corpus DR-01]

### 8.2 · Z-Wave (`zwave` driver)

```java
public class BZWaveNetwork extends BBasicNetwork implements ZWaveMessageConst
```

Z-Wave device discovery ("inclusion") is not a one-shot job — it is a persistent lifecycle.
The network holds a `BInclusionMonitor extends BComponent` that tracks completion flags
(`commandClassParsingComplete`, `wakeUpDataRetrievalComplete`, `associationsRetrevalComplete`,
etc.) as `@NiagaraProperty` booleans across station restarts. A node is usable only when all
flags are true. [ev: code BInclusionMonitor.java:26-52; BZWaveNetwork.java:101; corpus DR-02]

Command-class capability tree: each `BZWaveDevice` carries a `BCmdClassObject` subtree;
`BZWaveCommandClass` is a frozen enum of class IDs. Firmware upgrade: `.hex` image +
device template files are bundled inside the jar under `rc/` and served at runtime via
`module://` ORDs. [ev: code BZWaveNetwork.java imports; corpus DR-02]

---

### `getDeviceManagerSubscribeDepth()` — deep slot-tree discovery `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ20]`

**Override `getDeviceManagerSubscribeDepth()` in a `BNNetwork` subclass to return >1 for deep slot trees (PD-20):** the default implementation is hardcoded to return `1`, which subscribes only the immediate children of the device manager. When a device exposes a multi-level slot hierarchy (e.g. zone → device → point), override this method to return the correct depth (e.g. `3`). Without the override, discover and poll only reach the first level and the deeper points are never subscribed.

```java
@Override
public int getDeviceManagerSubscribeDepth() {
    return 3;  // depth: network → device → proxy ext → point
}
```

`[ev: corpus B1072]`

**`NMgrControllerUtil.network.getAgents().filter()` is the NATIVE device-manager-agent extension point (PD-14):** the static utility `NMgrControllerUtil.getAgents(network)` returns the list of registered `BINDeviceMgrAgent` implementations that apply to the current network. Filter this list in `getAgents()` to include only the agents that are relevant to the current device type. This is NOT CCN-specific — any `BNNetwork` subclass may override `getAgents()` and use this filter. Registering a `BINDeviceMgrAgent` via a plain `<type>` in `module.xml` is sufficient; no `@AgentOn` is required. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ14]` `[ev: corpus B1066]`

---

## 9 · Vendor/platform driver reference patterns (wave-3 survey)

Reference archetypes from obixDriver, lonworks, honBACnetUtilities, niagaraDriver, mbus, and opc modules. Our own modules stay at rung 0–1; these sections exist for scope discussions and as copy references only.

### 9.1 · oBIX driver vs. servlet — local-export-discover idiom [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ1]

**Driver vs. servlet decision:** use the Network/Device/ProxyExt driver pattern for bidirectional telemetry where the external system initiates reads/writes and the driver provides subscription + polling. Use a `BWebServlet` / SPA when you need a browser-facing UI on top of data that already lives in the station.

**Local-export-discover idiom** (pushing local N4 data outward, e.g. oBIX export): resolve the `slot:/` ORD to the station component space, then filter children by `BIWritablePoint` to discover exportable points. Iterate the filtered set to build the export payload; no separate Network subclass is needed for a pure-export path.

```java
// Resolve local station space and collect all writable points for export
BOrd root = BOrd.make("slot:/");
BComponent space = (BComponent) root.get(cx);
for (BObject child : space.getChildObjects()) {
    if (child instanceof BIWritablePoint) {
        BIWritablePoint wp = (BIWritablePoint) child;
        // push wp.getOut() to the external system
    }
}
```

Separate the discovery walk from the push loop — discovery is a one-time enumeration; push is per-cycle. [ev: corpus B1094]

### 9.2 · LonWorks — NV-binding UX is not a point list [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ2]

LonWorks commissioning requires three **distinct** manager patterns — do NOT collapse them into a single point-list table:

| Phase | Pattern |
|-------|---------|
| **XIF/LNML typing** | Load the `.xif` / `.lnml` descriptor; type each network variable (NV) from the device template, not from a live poll. A manager that shows raw poll values before typing is misleading. |
| **Changeable-NV discovery** | Some NV slots are dynamically changeable; a separate Discover manager enumerates them from the device and lets the user bind them. |
| **Service-pin commissioning** | Physical button press on the device triggers a service-pin message; the manager listens for this event to associate the node address — not a scripted API call. |

NV-binding UX is a separate workflow from point-list monitoring. Merging them into one manager produces a table that is either incomplete (pre-typing) or misleading (post-typing polluted by unbound NVs). [ev: corpus B1095]

### 9.3 · OEM-on-stock-driver patterns (honBACnetUtilities) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ3]

When building an OEM module on top of a stock Tridium driver (e.g. the stock BACnet driver), four patterns avoid duplicating the stock driver's architecture:

| Pattern | Mechanism |
|---------|-----------|
| **Type-float slot** | Add a `@NiagaraProperty BTypeSpec deviceModel` slot to the device subclass; the WB manager reads this slot to determine which OEM device-model plugin to activate — enables polymorphic behaviour without a Manager subclass per device family. |
| **Two-anchor manager mount** | The shared manager container registers two `@AgentOn` anchors: one on the stock base type (e.g. `bacnet:BacnetDevice`) and one on the OEM type; only the OEM anchor activates OEM columns/commands. |
| **Dual FE registration** | The OEM module registers its own `BWbFieldEditor` on the OEM slot type; the stock FE remains registered on the stock slot type. Both coexist without conflict — Workbench dispatches by the exact type match. |
| **ORD-carrier navigation** | ORDs that carry a type annotation (`ord|view:module:MyView`) let the WB tree navigate directly to the OEM view for devices that match the OEM type spec, without touching the stock driver's navigation. |

See also `types/wb-widgets.md §Plugin-extensible device-type manager via SPI` for the complementary WB-side pattern. [ev: corpus B1096]

### 9.4 · Command-driven manager SPI — BStationMgrCommand registry discovery [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ4]

`BStationMgrCommand` is a registry-discovered command SPI for adding toolbar commands to a station-navigation manager (e.g. the Drivers Manager). Key points:

- Implementers register via a plain `<type>` in `module.xml`; the framework discovers all registered `BStationMgrCommand` implementations via `NiagaraRegistryUtil.getImplementersOfTypeSpec()` — no static registration call required.
- **Session-keyed learn state:** each learn session holds its progress in a per-session map keyed by `Context.getSessionId()`. On manager close/reopen the session key is different; old state is GC'd automatically. Do NOT use a static map for session learn state — it causes cross-session contamination.
- **Offline `.bog` guard:** before executing any command that modifies the station model, check `Sys.isOnline()` / `BStation.isRunning()`. Commands that run against an offline `.bog` snapshot must be read-only; write-mode operations must be gated with a "Station must be running" error.
- **`CredentialsColumn` lease + `newCopy`:** when a manager column carries credentials, call `column.getCredentials().lease()` to get a snapshot and `snapshot.newCopy()` to produce a safe copy for the background thread — never hold a reference to the live `BCredentials` object across a thread boundary.
- **Shorthand `HistoryId`:** use `BHistoryId.make(stationName, historyName)` rather than constructing the ORD string manually; the shorthand handles the namespace separator rules correctly across Fox links.

[ev: corpus B1097]

### 9.5 · Field-bus protocol driver archetype — dual-addressing discovery wizard (M-Bus) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ7]

M-Bus and similar field-bus drivers require a **dual-addressing discovery wizard** — a two-phase discovery manager that handles both address spaces:

| Phase | Address type | Range | Manager pattern |
|-------|-------------|-------|----------------|
| **Primary scan** | Primary address (1 byte, 0–250) | Linear; one request per address | Progress bar; abort on first valid response per address |
| **Secondary scan** | Secondary address (8 bytes: manufacturer + ident + medium + version) | 2³² filtered bitmask walk | Wildcarded broadcast; narrows on each confirmed response |

Additional patterns for field-bus protocol drivers:
- **Multi-baud scan:** the wizard must iterate across supported baud rates (300/600/1200/2400/4800/9600) if the device baud rate is unknown at commissioning; cache the confirmed baud on the device component.
- **Manufacturer-specific data model:** `DataRecord` (DIF + VIF + data) is decoded differently per manufacturer extension code; keep the decoder table in a separate `ManufacturerDecoder` registry, not inlined in the proxy ext — this allows new manufacturer codes to be added without touching the polling path.
- **Hardcoded-facet lint candidate:** when a history import assigns `facet precision=4` independently of the proxy point's `-exponent` facet, the two can diverge silently. Flag this mismatch in import-learn flows (see `types/issues-and-gotchas.md §H1` for the proposed lint). [ev: corpus B1100]

[ev: corpus B1100]

### 9.6 · Protocol-adapter manager — action-slot bridge and structured error decode (OPC) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ10]

OPC and similar protocol-adapter drivers expose three recurring WB patterns that differ from standard BACnet/Modbus drivers:

**Action-slot bridge:** OPC items are exposed as Niagara action slots rather than as proxy points; invoking the action calls the underlying COM/native OPC operation. Keep the COM/native error handling inside the driver layer — never let `COMException` or `NativeException` propagate to the WB view. Decode the HRESULT/native error code into a Niagara `BFacets` display string before reporting to the manager.

```java
// Inside the driver comm layer — structured COM error decode
try {
    opcServer.write(itemId, value);
} catch (COMException e) {
    String decoded = OpcErrorDecoder.decode(e.getHResult()); // map HRESULT → human string
    throw new DriverException("OPC write failed: " + decoded); // stripped of COM type
}
// WB view receives only DriverException — no COM imports needed in -wb
```

**Lazy hierarchical browse:** OPC address spaces can contain millions of nodes; do NOT eagerly expand the full tree on manager open. Implement `isLeaf(node)` and `getChildren(node)` on demand (on node expand only); cache expanded nodes in a `WeakHashMap` keyed by node path.

**Security-gated state:** OPC security (DCOM, certificate-based OPC-UA) may leave the connection in a partially authenticated state. Model this as a distinct `SECURITY_FAULT` status separate from `COMM_FAULT` — display a locked-padlock column in the manager and gate write actions on `isSecurityReady()`. The WB manager must never expose a write path when the security state is not fully resolved. [ev: corpus B1103]

---

**See also:** `types/logic.md` (BStatus bits, BControlPoint), `types/logic-authoring.md`
(§Authoring a driver — write-path safety checklist, writeOnUp/fallback lints).
