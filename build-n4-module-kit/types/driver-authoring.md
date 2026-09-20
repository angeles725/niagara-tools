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

### 1.3 · Plugin-extensible manager SPI — multi-device-family alternative [ev: retro honeywell-wb-rt-wb-deltas Δ1]

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

**See also:** `types/logic.md` (BStatus bits, BControlPoint), `types/logic-authoring.md`
(§Authoring a driver — write-path safety checklist, writeOnUp/fallback lints).
