# Driver authoring — network/device/proxy SPI reference

Custom Niagara driver in N4: four classes to subclass, one comm engine to wire, one tuning policy
to configure. This doc is the author-side reference; protocol-specific wire format is out of scope.

---

## 1 · Class ladder — which base to extend

### 1.1 · The four required subclasses

| Class | Your subclass base | What you override |
|---|---|---|
| Network | `BDeviceNetwork` / `BBasicNetwork` / `BLoadableNetwork` (§1.2) | `getDeviceType()`, `getDeviceFolderType()` |
| Device | `BDevice` / `BBasicDevice` / `BLoadableDevice` | `getNetworkType()`, `void doPing()` (§5) |
| Point folder | `BPointDeviceExt` | `getProxyExtType()` |
| Proxy ext | `BProxyExt` | `getMode()`, `readSubscribed(cx)`, `readUnsubscribed(cx)`, `write(cx)` |

`BProxyExt` is a point extension (`extends BAbstractProxyExt`) — it sits as a child of a
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

**Every `BBasicNetwork` driver ships a `-wb` device manager (WB-presence rule, PD-01):** a driver that adds a `BBasicNetwork` subclass MUST register a WB view via `@AgentOn` on the network or device type — at minimum a `BAbstractManager` or `BWbComponentView`. Relying on the default property-sheet view forces operators into raw slot editing. See `types/wb-widgets.md §Vendor-grade -wb UX patterns`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ1]` `[ev: corpus B1061 PD-01; corpus B1057 (CCN example)]`

**Driver modules with NO Hx/UX intent MUST state that explicitly (PD-13):** when a driver module deliberately ships no `-ux` profile and relies entirely on the base Niagara hx-wb for its browser view, add a comment in the module's `SKILL.md` / README: `# UX: none — relies on base hx-wb`. This prevents a future developer from adding a redundant `-ux` scaffold when the driver's intent is station-side only. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ13]` `[ev: corpus B1061]`

### 1.4 · Plugin-extensible manager SPI — multi-device-family alternative [ev: retro honeywell-wb-rt-wb-deltas Δ1]

When multiple device families share one `BAbstractManager` container (wb side), avoid subclassing the manager once per family. Instead, implement a **plugin SPI** (`BIHonDeviceModel` or equivalent `BIHonBacnetDeviceModel`) per family module; the shared framework discovers all registered implementations via Honeywell's own helper `NiagaraRegistryUtil.instance().getImplementersOfTypeSpec(String)`.

- Each SPI implementer declares `getSupportedModels()`, `getSupportedColumns()`, `getSupportedDiscoveryColumns()`, `getSupportedCommands(BWidget)`, and `getDeviceType()`.
- Register via a plain `<type>` in `module.xml` — no `@AgentOn`.
- Prefer this over a Manager subclass when the manager framework already exists and you are contributing a new device-family module on top of it.

Full wb-side recipe in `types/wb-widgets.md §Plugin-extensible device-type manager via SPI`. `[ev: corpus B1077]`

---

### 1.5 · Browser-side managers (`-ux`) — the JavaScript twin of the Workbench manager `[ev: retro t9c-reference-gaps Δ4]`

The Java `BAbstractManager` framework (corpus B1088–B1093, `types/wb-widgets.md`) has a browser twin in
webEditors-ux / driver-ux that renders the same device/point manager in the HTML5 (hx) profile. Paths below
are under niagara-research `organized/`; `W=webEditors/webEditors-ux/extracted/rc/wb/mgr`.

- **AMD ids:** `nmodule/webEditors/rc/wb/mgr/Manager`, `.../mgr/model/MgrModel`, `.../mgr/model/MgrColumn`,
  `nmodule/driver/rc/wb/mgr/DeviceMgr`, `DeviceMgrModel`, `PointMgr`, `PointMgrModel`. [CERT-doc]
  `niagara-help/versions/4.15.6.20/devguide-clean/jsdoc/driver-ux/module-nmodule_driver_rc_wb_mgr_DeviceMgrModel.txt:34`
- **Extend `DeviceMgr`/`PointMgr` and `DeviceMgrModel`/`PointMgrModel`, never `Manager`/`MgrModel`** — Tridium
  marks the base framework incubating. [CERT] `W/Manager.js:72-75`, `W/model/MgrModel.js:44-48`
- **View contract:** constructor `{moduleName, keyName}` (lexicon + saved state); `makeModel(value)` returns
  a `Promise<MgrModel>` (the base throws, `W/Manager.js:415-417`); extra commands go in `makeCommands()`, which
  is async since 4.14 (`frontend-standard.md` § UI API changes). [CERT] `W/Manager.js:77-85`
- **Model contract:** `DeviceMgrModel` wraps the network in a `ContainerComponentSource` filtered to
  `driver:Device` and overrides `makeRow(subject)`; `MgrModel` adds `newTypes` and `newInstance(typeInfo)`.
  [CERT] `driver/driver-ux/extracted/rc/wb/mgr/DeviceMgrModel.js:8-11,91`; `W/model/MgrModel.js:54-66,160`.
  Columns hook `buildCell`, `mgrValidate`, `commit` (`W/model/MgrColumn.js:82` and below); ready-made
  columns live in `W/model/columns/`.
- **Discovery:** the manager provides `makeLearnModel()` and `doDiscover()`, and maps a dragged discovery
  node with `getProposedValuesFromDiscovery()`. [CERT] `W/MgrLearn.js:45-97`
- **Registration needs a small Java class in the `-ux` jar:** a `BSingleton implements BIJavaScript`
  (optionally `BIFormFactorMax`, `BIOffline`) with `@AgentOn(types=...)` on your network/folder type and
  `getJsInfo()` → `JsInfo.make(BOrd.make("module://<mod>/rc/.../MyMgr.js"), <your BJsBuild>.TYPE)`. Worked
  example: [CERT] `modbusCore/modbusCore-ux/vineflower/com/tridium/modbusCore/ux/mgr/BModbusClientPointUxManager.java:17-38`.
  The agent's `requiredPermissions` controls visibility only, not server-side access. `[ev: corpus B752 §752.5]`
- **Which one to build** [INFER]: the Java manager when Workbench is the only client or you need its mature
  learn/edit transactions; the ux manager when operators manage devices/points from a browser. A bespoke
  dashboard is neither — it stays a servlet-SPA (`dashboard.md`).

### 1.6 · NDriver — comm config, managers and discovery

New drivers are written on NDriver (`com.tridium.ndriver`); Tridium's older driver frameworks are deprecated or slated
to be, per the course `[CERT-a]` `[ev: corpus B1419 §1419.4 row f]`.

- **Client drivers keep `serverEnabled` false.** `BTcpCommConfig.serverEnabled` defaults to `false` in the framework
  `[CERT]` `organized/docSource/docSource-doc/extracted/ndriver-rt/com/tridium/ndriver/datatypes/BTcpCommConfig.java:32-36,94`;
  when true, the link layer opens a listening socket on the configured port (`TcpLinkLayer.java:123`, vineflower). A
  network that came up with it true (course Lab 11) faulted `Address already in use: JVM_Bind` because the real
  server held the port, with Health `Can not connect to local:0`. A client driver's `B<X>TcpCommConfig` never
  overrides the default to true, and commissioning checks Tcp Config › Server Enabled = false before the first ping.
  [ev: retro n4-course-handson-ndriver Δ4] [ev: retro r8-t9e-drivers Δ2]
- **Manager columns and discovery** (verified live on the course driver): a device column comes from a property facet
  `SfUtil.incl(SfUtil.MGR_EDIT)` (or `MGR_EDIT_READONLY`) (`ndriver-rt/…/util/SfUtil.java:15,28`, vineflower); a point
  manager is `@NiagaraType(agent = @AgentOn(types = {"<mod>:<X>PointDeviceExt", "<mod>:<X>PointFolder"}))` extending
  `BNPointManager` with `NPointModel`/`NPointController` (`ndriver-wb/…/ui/point/BNPointManager.java:36-41,70,98`, docSource); a
  proxy-ext column is `new MgrColumn.PropPath(new Property[] {BControlPoint.proxyExt, B<X>ProxyExt.<slot>}, MgrColumn.EDITABLE)`
  (`workbench-wb/…/mgr/MgrColumn.java:730`); `getNewTypes()` restricts the New menu. `doSubmitDiscoveryJob(prefs, cx)`
  submits with `job.submit(cx)` — the wizard's `job.submit(null)` runs the job with no user (`lint-driver-scaffold.sh`
  WARN). Discovery and learn jobs run on their own job threads: a Workbench freeze during discovery is not the engine
  thread, and no driver I/O ever runs on the engine thread `[ev: corpus B1419 §1419.3]`.
  [ev: retro n4-course-handson-ndriver Δ6]

### 1.7 · NDriver — layers, threads and request/response correlation `[ev: corpus B1239 §1239.12-§1239.13]`

Paths are `ndriver-rt/com/tridium/ndriver/…` (docSource) unless named.

- **Layers** (course Lab 11 `[CERT]`): the network owns the comm stack (a `BCommConfig` subclass such as
  `BTcpCommConfig`, driving `NComm`); one `NMessage` subclass per message type; one `LinkMessage` subclass frames
  every message on the wire.
- **Threads:** `doPoll` runs on the scheduler's `Poll:<network>` thread (`BNPollScheduler extends BPollScheduler`,
  `driver-rt/…/util/BPollScheduler.java:809`); `BNDevice.postPing()` posts the ping to the network's `BNWorker`
  (`BNDevice.java:72-79`, `BNNetwork.java:101,145-149`). Post proxy-ext I/O there with `postAsync(Runnable)` and
  return `true` from `write()` (§ 2).
- **Correlation rule.** Open transactions are keyed by `NMessage.getTag()` (`comm/NCommTimer.java:162`) and a
  response is matched the same way (`comm/NComm.java:356`). The default tag is the shared `nullTag`
  (`comm/NMessage.java:63-68`) while `BCommConfig.getMaxOutstandingTransactions()` returns 32
  (`datatypes/BCommConfig.java:436-444`). So: a protocol with a request id overrides `getTag()` on requests and
  responses; a protocol without one returns 1 from `getMaxOutstandingTransactions()` in its comm-config subclass.
  With neither, the poll thread, the worker and a discovery job share one key, and a response can complete the
  wrong request `[INFER]` (B1239 §1239.12 row 1).
- **Timeouts:** a request waits `getResponseTimeOut()` (default 0) plus `getRespTimePadding()` (1000 ms) and
  retries `getRetryCount()` (default 0) times (`comm/NMessage.java:151-200,233`, `comm/NCommTimer.java:524-527`):
  1 s, no retry. Set both per message (`setResponseTimeOut`, `setRetryCount`) from the device's real latency.
- **Defect checklist** — the course reference driver ships each one (B1239 §1239.12): keep subscription bookkeeping
  per instance (the reference holds `private static final Set<String> subscribedPoints`,
  course `…/Lab11 NDriver/4 Reading and Writing Points/BTrafficLightDriverDevice.java:317`, shared by every device); bounds-check `LinkMessage.receive` against the
  buffer; check a response's error flag before parsing its payload; log frames at FINE, not INFO; guard every
  proxy-ext cast with `instanceof` (`BNPollScheduler.doPoll` catches it and only calls `printStackTrace()`,
  `poll/BNPollScheduler.java:48-59`).
- Owed advisory lints G66 `ndriver-static-collection` and G67 `ndriver-tagless-correlation`
  (`ledger.d/2026-10-23-r8-u5-c59-c62.md`). [ev: retro n4-course-drivers-bql-ords-files Δ3]
  [ev: retro n4-course-handson-ndriver Δ7]

### 1.8 · Which driver framework `[ev: corpus B877 §877.6]`

Three generic chassis share the `javax.baja.driver` root; pick one before the first class.

| Framework | Network base | Transport plug-in | You write | Pick it when |
|---|---|---|---|---|
| NDriver (`ndriver`) | `BNNetwork extends BDeviceNetwork` | a `BCommConfig` subclass: `makeLinkLayer(NComm)` → shipped TCP, UDP, serial and HTTP link layers | messages, a link message, a message factory, the comm config subclass (§ 1.9-§ 1.10) | **a new driver.** The New Driver Wizard scaffolds it, and the course says the older frameworks are deprecated or will be `[CERT-a]` `[ev: corpus B1419 §1419.4 row f]` |
| basicDriver (`basicDriver`) | `BBasicNetwork extends BLoadableNetwork` | `makeComm()` → a `Comm` subclass; serial through `BSerialNetwork`; no TCP transport (§ 1.2) | the `Comm` subclass, messages, the receiver | maintaining or extending a driver already on it (§ 3) |
| DDF (`devDriver` + `devIpDriver` / `devSerialDriver` / `devHttpDriver`) | `BDdfNetwork extends BLoadableNetwork` | a `communicator` slot holding a `BIDdfCommunicator`; TCP, UDP, serial and HTTP networks ship (`BDdfUdpNetwork`, …) | the DDF request/response and discovery classes | maintaining or extending a driver already on it |
| raw `javax.baja.driver` | `BDeviceNetwork` | none: you own sockets and threads | everything, including the worker threads | only to wrap an existing protocol stack that brings its own threads `[INFER]` |

Chassis and transport columns: `[ev: corpus B877 §877.1]` `[ev: corpus B877 §877.5]`. There is no
`BIpNetwork` class in the corpus: an NDriver IP driver uses `BTcpCommConfig` / `BUdpCommConfig`, a DDF one
`BDdfTcpNetwork` / `BDdfUdpNetwork` (`organized/devIpDriver/devIpDriver-rt/vineflower/com/tridium/ddfIp/udp/BDdfUdpNetwork.java`).

### 1.9 · NDriver comm config — TCP, UDP, serial

Paths are `ndriver-rt/com/tridium/ndriver/…` (docSource) unless named; wizard templates are
`organized/devkit/devkit-wb/extracted/gradle/ndriver/`.

- **Hang the config on the network** as a property (`tcpConfig`, `udpConfig`, `serialConfig`) and reach the
  stack with `(NComm) getUdpConfig().comm()`; call `statusUpdate()` on it when the network's `status` changes
  (wizard `BNfooNetwork.java.vm:60-79,169-180,222-240`). `comm()` starts the stack on first use
  (`datatypes/BCommConfig.java:250-260`).
- **Subclass the config; never ship a bare `BUdpCommConfig`.** The wizard's UDP option declares a plain
  `BUdpCommConfig` (`BNfooNetwork.java.vm:67-72`), which keeps the base message factory: it builds a plain
  `NMessage` (`datatypes/BCommConfig.java:413-425`) whose `fromInputStream` throws "not implemented"
  (`comm/NMessage.java:49-54`), so every received datagram fails. Override `makeMessageFactory()` (your
  `IMessageFactory`) and `makeLinkMessageFactory()` (your `LinkMessage` subclass and its maximum length), as the
  wizard's TCP template does (`BNfooTcpCommConfig.java.vm:25,44-57`).
- **The config's `address` is the LOCAL side; the peer travels on each message.** `BTcpCommConfig.address`
  (default `local:0`) is the listen address used only with `serverEnabled` (§ 1.6); `BUdpCommConfig.address`
  (default `local:-1`) is the bind address: `local` binds every interface, a port ≤ 0 an ephemeral port
  (`comm/udp/UdpLinkLayer.java:59-77,126-139`, `datatypes/BTcpCommConfig.java:28-36`). Both link layers send to
  `(BIpAddress) msg.address`, which `LinkMessage.setMessage` copies from `NMessage.getAddress()`
  (`comm/udp/UdpLinkLayer.java:164-172`, `comm/tcp/TcpLinkLayer.java:206-209`, `comm/LinkMessage.java:62-66`).
  So keep the device's host and port on the DEVICE and set it on every request. A TCP connect is bounded at
  1000 ms (`comm/tcp/TcpLinkLayer.java:432`).
- **UDP: one datagram is one frame.** The UDP receiver sets the frame length and the source address from the
  packet and never calls `LinkMessage.receive` (`comm/udp/UdpLinkLayer.java:193-208`); the TCP and serial
  receivers do call it, so a stream protocol frames there (`comm/tcp/TcpLinkLayer.java:355`,
  `comm/serial/SerialLinkLayer.java:322`). The receive buffer is the link message's array, 1500 bytes with the
  default factory (`datatypes/BCommConfig.java:474-477`); size it to the protocol's largest datagram, since a longer
  one is cut to the buffer `[INFER]` (Java `DatagramPacket`).
- **UDP: check the sender.** A response is matched by tag alone (`comm/NComm.java:355`), so any host that
  sends a matching tag completes the transaction; in `validateResponse` check
  `getAddress().sameDevice(resp.getAddress())` (it compares the host, not the port,
  `ndriver/datatypes/BIpAddress.java:196-204`) and return `FAILED_RESPONSE` on a mismatch (`comm/NMessage.java:94-103`).
- **DNS per send.** `BIpAddress.getInetAddress()` calls `InetAddress.getByName` on every call and returns `null`
  on failure (`ndriver/datatypes/BIpAddress.java:178-188`); prefer IP literals for devices, and see § 5 for a cached
  resolve.

### 1.10 · NDriver message framing — the skeleton

One request class and one response class per message type, one `LinkMessage` subclass for the wire frame, one
factory. Shape (write the codec yourself; course Lab 11 is example code only and ships the defects of § 1.7):

```java
// Request: encode, tag, timeout. Keep the byte layout in a Baja-free codec class (§ 12).
public class XReadReq extends NMessage {
    public XReadReq(BIpAddress to, int seq, int reg) { super(to); this.seq = seq; this.reg = reg;
        setResponseTimeOut(2000); setRetryCount(1); }               // real device latency, § 1.7
    @Override public boolean toOutputStream(OutputStream out) throws Exception {
        out.write(XCodec.encodeRead(seq, reg)); return false; }     // false = no more fragments
    @Override public Object getTag() { return seq; }                 // same key on the response
    @Override public int validateResponse(NMessage resp) {
        if (!getAddress().sameDevice(resp.getAddress())) return FAILED_RESPONSE;   // UDP: § 1.9
        return ((XResp) resp).isError() ? FAILED_RESPONSE : SUCCESS_RESPONSE; }
    private final int seq, reg;
}
// Response: isResponse() true, decode, the same tag.
public class XResp extends NMessage {
    @Override public boolean isResponse() { return true; }
    @Override public void fromInputStream(InputStream in) throws Exception { frame = XCodec.decode(in); }
    @Override public Object getTag() { return frame.seq; }
    public boolean isError() { return frame.error; }
    XCodec.Frame frame;
}
// Factory: pick the class from the frame; return null for a frame you do not handle (NComm drops it).
public NMessage makeMessage(LinkMessage lm) throws Exception {
    NMessage m = new XResp(); m.setAddress((BAddress) lm.address);
    m.fromInputStream(lm.getInputStream()); return m; }
```

- **Stream transports frame in `LinkMessage.receive(InputStream)`**: return true only for a complete frame,
  bounds-check every write into `buffer` against its length, and resynchronise on a start byte; `setMessage`
  writes start byte, payload and end byte to `getOutputStream()` (`comm/LinkMessage.java:36-66,140`).
  `initReceive(msgToSend)` lets a streaming link layer constrain the next receive with the request just sent
  (`comm/LinkMessage.java:105-118`). UDP skips `receive` (§ 1.9).
- **Correlation:** a request with an id overrides `getTag()` on both classes; without one, return 1 from
  `getMaxOutstandingTransactions()` (§ 1.7). `NComm.sendRequest(msg)` blocks for the matched response and
  returns `msg.modifyResponse(resp)`; `sendMessage(msg)` sends with no response (`comm/NComm.java:150,226-240`).
  A response with no open transaction is logged SEVERE and dropped (`comm/NComm.java:355-360`).
- **Unsolicited frames** (`isResponse()` false) go to the config's default listener, set in the network's
  `started()` with `setDefaultListener(...)` (`datatypes/BCommConfig.java:377-384`; wizard
  `BNfooNetwork.java.vm:102-120`). The factory, the listener and the codec run on the comm receive thread, never
  the engine thread.
- Course reference for the shape only: `sources/n4-dev-course/course/Labs/Lab11 NDriver/1 Creating Common Classes/TrafficLightDriverLinkMessage.java:30-61`
  (its `receive` has no bounds check) `[ev: corpus B1239 §1239.6]`.

### 1.11 · NDriver discovery — the job body

You do not write a `BJob`. `BNDiscoveryJob.doRun` starts a `JobThread` that calls your
`BINDiscoveryHost.getDiscoveryObjects(prefs)`, adds what it returns, and repeats while `prefs.isMultiStep()` is
true, the call returned non-null and the job is running; `end()` then sets failed (an exception), canceled, or
success (`discover/BNDiscoveryJob.java:150-157,186-210,256-271`). Cancel sets `canceling` and interrupts the
thread (`discover/BNDiscoveryJob.java:163-173`). The network implements `BINDiscoveryHost` for devices, the
point device ext for points (`point/BNPointDeviceExt.java:54,177-186`).

Your `getDiscoveryObjects(prefs)`:
1. Return `null` at once when `!prefs.getJob().isAlive()` (not running: canceled or done,
   `baja/javax/baja/job/BJob.java:450-453`), and check it again between requests in a long scan.
2. Send with the blocking `sendRequest` (this is the job thread, not the engine thread).
3. Report progress: the job never calls `progress()` itself, so call `prefs.getJob().progress(pct)` per step
   (it also heartbeats), or `heartbeat()` when the total is unknown (`baja/javax/baja/job/BJob.java:30-38,510-521`).
   Log per item with `prefs.getJob().log()`.
4. On a bad reply call `prefs.getJob().discoverFail(reason)` and return `null`: `discoverFail` only logs a
   message (`discover/BNDiscoveryJob.java:282-288`), so the job still ends in success unless you throw.
5. Return one discovery leaf per object. A `BNDiscoveryLeaf` subclass maps it to a component:
   `getValidDatabaseTypes()` (first entry = best type), `getDiscoveryName()`, `updateTarget(target)` copies the
   discovered address and settings, `isExisting(component)` greys rows already in the station
   (`discover/BNDiscoveryLeaf.java:53-95`). The Workbench manager calls them
   (`organized/ndriver/ndriver-wb/vineflower/com/tridium/ndriver/ui/NMgrLearn.java:91,121-122,145`); the browser
   manager's `getProposedValuesFromDiscovery` asks the station for the same name and types over RPC
   (`organized/ndriver/ndriver-ux/extracted/rc/Automanager.js:370,439-449`).
6. Multi-step scans (an address range, a paged browse) return `true` from `isMultiStep()` in their preferences
   and return `null` when finished (`discover/BNDiscoveryPreferences.java:104-108`).

The framework's own `BNPointDeviceExt.doSubmitDiscoveryJob` submits with `job.submit(null)`
(`point/BNPointDeviceExt.java:185`); override it with the `Context` form as in § 1.6 so the job runs as the user.
Course device and point discovery: `[ev: corpus B1239 §1239.11]`.

## 2 · `BProxyExt` point SPI

Three overrides — the framework calls them, never call them yourself. Neither read callback may block: do the
I/O on another thread and report each result with `readOk(value)` / `readFail(cause)` `[CERT]`
`organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/point/BProxyExt.java:851-869`.
`getPollScheduler()` and `postWrite(...)` stand for your network's own accessors (`getNetwork()` returns `BDeviceNetwork`,
which has neither; cast to your network type, e.g. a `BBasicNetwork`); `BPollScheduler.subscribe(p)` / `unsubscribe(p)` are
`util/BPollScheduler.java:1130,1154`. [ev: retro devguide415-nav-subscriptions-tags-ords Δ1] `[ev: corpus B1306 §1306.2 C1]`

```java
// The point ENTERS the subscribed state (a watcher, a link, or permanent subscription — see
// logic-authoring.md § Author-side SPIs): start polling or register a native COV subscription.
@Override
public void readSubscribed(Context cx) throws Exception {
    getNetwork().getPollScheduler().subscribe(this);     // or COV register
}

// The point EXITS the subscribed state: stop polling or unregister. Never read here.
@Override
public void readUnsubscribed(Context cx) throws Exception {
    getNetwork().getPollScheduler().unsubscribe(this);   // or COV unregister
}

// Called when our control logic BLinks a value into a proxy writable point.
// NEVER block here — post to an async worker instead (§3 CoalesceQueue).
@Override
public boolean write(Context cx) throws Exception {
    getNetwork().postWrite(buildWriteRequest(cx));
    return true;    // true = write enqueued: call writeOk()/writeFail() when it completes
}
```

Callback pair: `readOk(value)` / `readFail(cause)` and `writeOk()` / `writeFail(cause)` — call
these from your comm thread to update the point's `out` slot. [ev: code BProxyExt.java:895,923,937,970;
corpus B810 §810.3]

**`write()` returns whether a write is pending.** `true` = enqueued, the driver later calls `writeOk()` or
`writeFail()`; `false` = cancelled at once, and `Tuning` clears the pending flag (`driver-rt/…/point/BProxyExt.java:872-883`,
`Tuning.java:346-351`, docSource). An async driver that returns `false` reports "not pending" for a write it
still sends. NDriver threading: § 1.7. [ev: retro n4-course-drivers-bql-ords-files Δ3]

**Enum proxy points show state names only with a frozen-enum range.** Set the point's `range` facet through
"Use Frozen Enum in Range (module:name)" (`EnumRangeEditor.useFrozen`, `webEditors-ux.lexicon:92`) with the
driver's enum type, e.g. `trafficLightDriver:TrafficLightState`; without it the course Lab 11 points showed ordinals
`[CERT-live]` (corpus B1257 L11.40). Commissioning checks the range of every enum proxy before handover.
[ev: retro n4-course-handson-ndriver Δ5]

**BACnet: a field alarm or fault reaches Niagara status only with the `statusFlags` facet**
`[ev: retro devguide415-r2-drivers-protocols Δ4]` `[ev: retro devguide415-r3-drivers-integration Δ3]` `[ev: corpus B1348 §1348.2]`.
`useStatusFlags()` is true only for a Present_Value proxy whose object type has Status_Flags AND whose point facets
carry `statusFlags=true` (or the ext's own override); with the facet absent it is false (vineflower bacnet-rt
4.14.0.162 `javax/baja/bacnet/point/BBacnetProxyExt.java:808-822`). Without it a point that is in alarm, faulted or
overridden in the controller reads `{ok}` in Niagara. With it, polled and COV proxies alike take ALARM, FAULT and
OVERRIDDEN from the flags; the cost is one extra Status_Flags read per poll, or a metadata read when a COV
notification omits the flags (corpus B1348 §1348.2 rows 4-7). A dashboard health read is only as good as this facet
(`types/dashboard.md` § Facade contract). Owed Go lint G36 `bacnet-status-flags` (bog check).

**Gotcha:** a write to a DOWN, DISABLED or fatally faulted device is **silently dropped** — `Tuning.write()`
calls `isOperational()` first and returns without queuing if false. The write is NOT retried unless
`BTuningPolicy.writeOnUp=true` (default true). [ev: code Tuning.java:332-333, :549-552, :629; corpus B810 §810.4]

---

## 3 · `BBasicNetwork` comm layer

```
makeComm() → Comm { CommReceiver rx + CommTransmitter tx + CommTransactionManager }
                        ↑ unsolicited frames          ↑ request→response tag matching
```

Three Niagara worker threads the network owns:

| Worker | Purpose |
|---|---|
| `worker` | `post`/`sendAsync` requests (a `BBasicCoalescingWorker`) |
| `writeWorker` | `postWrite`/`sendAsyncWrite` requests; a `CoalesceQueue` sized by `maxQueueSize` (default 256), so queued writes with the same coalesce key collapse |
| `dispatcher` | runs every transmit: `sendSync` dispatches to it and waits; unsolicited frames go from the receiver to registered listeners (`comm/Comm.java:240-251`) |

[ev: code basicDriver-rt vineflower `BBasicNetwork.java:30-66,240-278`, `util/BBasicWorker.java:17-22,48-54`, `util/BBasicCoalescingWorker.java:19-20`]

Retry loop lives in `Comm.transmit(msg, timeout, retryCount)`: sends → waits `responseTimeout`
on a `CommTransaction` → tries `retryCount+1` times in all (`comm/Comm.java:146-160`, vineflower). [ev: corpus B517 §517.2]

Send API:

```java
network.sendAsync(msg, listener);        // reads — never block; the MessageListener gets the response
network.sendAsyncWrite(msg, listener);   // writes — routes through writeWorker
```

**Gotcha:** `sendSync` exists but blocks the calling thread until the dispatcher returns the response
(`BBasicNetwork.java:257-262,281-294`, vineflower). Call it only from your own background
`BWorker`, never from the engine thread or a timer callback `[INFER]`. [ev: corpus B517 §517.2]

---

## 4 · `BTuningPolicy` map

Place a `BTuningPolicyMap` named `tuningPolicies` on your network. The map always contains a
`defaultPolicy` child; add more by name. Per-point binding: `BProxyExt.tuningPolicyName` (String,
default `"defaultPolicy"`).

| Slot | Default | Effect |
|---|---|---|
| `minWriteTime` | 0 (off) | Minimum gap between device writes; closer writes suppressed |
| `maxWriteTime` | 0 (off) | Heartbeat: force re-write even if value unchanged |
| `staleTime` | 0 (timer off) | Mark point STALE when this long passes without a `readOk` |
| `writeOnStart` | **true** | Write on station/point start; set `false` except for critical points on a large network (write-queue overflow) |
| `writeOnUp` | **true** | Re-write when device returns UP after comm loss |
| `writeOnEnabled` | **true** | Write when point re-enabled |

[ev: code BTuningPolicy.java:112,146,177,206,235,268 (docSource); corpus B872 §872.2; corpus B1327 §1327.1 row 2 (writeOnStart advice)]

**Gotcha:** the Tuning scan thread runs at `min(minWriteTime/2, maxWriteTime/2, staleTime/2)`
clamped to `[200 ms, 20 000 ms]`. With all-zero defaults the scan fires every 20 s — this is
correct; the policy kicks in only when specific slots are set. [ev: corpus B872 §872.2]

**Stale time.** The generic framework applies a stale timer only when `staleTime > 0` (`Tuning.java:484`); 0 = no
timer. The guides add "0 = stale immediately when unsubscribed" (the `BTuningPolicy.staleTime` javadoc says the same),
but no code in driver-rt, basicDriver or bacnet-rt implements it — keep it out of a design `[ev: corpus B1365 §1365.4]`
`[ev: corpus B1419 §1419.4 row e]`. COV points keep the default 0; a polled point gets a few poll periods (a 1-min poll →
a 3-min stale time, course practice `[CERT-a]`). [ev: retro devguide415-r2-drivers-protocols Δ3]

**Sizing.** Keep the poll scheduler's busy time ≤ ~75 % with only the permanent subscriptions running (linked points,
points with history or alarm extensions). Ship a few named policies (Slow / Normal / Fast, and a no-`writeOnStart`
variant) and bind them per point; keep `defaultPolicy` for the rest and add a custom one only for a problem point.
COV sensitivity is a tuning-policy setting, never a display facet. Proxy points have no inputs: put history and alarm
extensions on the SOURCE point, not on the proxy. `[ev: corpus B1337 §1337.1 rows 5-7]` `[ev: corpus B1418 §1418.4]`
`[ev: corpus B1419 §1419.4 row e]` [ev: retro devguide415-r2-drivers-protocols Δ8]

---

## 5 · Health — `doPing` and fault reporting

The API is `public abstract void doPing() throws Exception`; `pingOk()` and `pingFail(String cause)` are void calls
routed to the device's health `[CERT]` `organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/BDevice.java:552,570,580`.
A device with no protocol of its own (reachability only) pings with a bounded socket connect (`getHost()`/`getPort()`
are the device's own slots); the timeout bounds the
connect on the `Ping:` thread (not a DNS lookup, below), and the module needs the station-scoped `NETWORK_COMMUNICATION` permission:

```java
@Override
public void doPing() throws Exception {
    try (Socket s = new Socket()) {
        s.connect(new InetSocketAddress(getHost(), getPort()), 2000);   // 2000 ms bounds the connect, not DNS
        pingOk();                                  // clears DOWN, fires offnormal→normal alarm
    } catch (IOException e) {
        pingFail(e.getLocalizedMessage());         // sets DOWN on the device + all its proxy points
    }
}
```

[ev: retro n4-course-handson-ndriver Δ1] [ev: retro n4-course-handson-ndriver Δ3]

**The timeout does not bound DNS.** `new InetSocketAddress(host, port)` resolves a host name in its constructor
("An attempt will be made to resolve the hostname into an InetAddress"), before `connect` and its 2000 ms start
`[CERT-doc]` (https://docs.oracle.com/javase/8/docs/api/java/net/InetSocketAddress.html), so a slow resolver holds
the `Ping:` thread for the resolver's own timeout. An IP-literal `host` costs no lookup: `InetAddress.getByName`
only checks the format of a literal `[CERT-doc]` (https://docs.oracle.com/javase/8/docs/api/java/net/InetAddress.html),
and the constructor going through that path is `[INFER]`. For a host-name device, never resolve in `changed()`
(that blocks the engine thread): resolve on the `Ping:` thread, keep the `InetAddress`, ping
`new InetSocketAddress(addr, getPort())`, and drop the cache when `host` changes or a connect fails, so DNS costs one
lookup per change instead of one per ping. That lookup is still bounded only by the OS resolver `[INFER]`.
`InetSocketAddress.createUnresolved` is no way out: `Socket.connect`'s javadoc does not say what an unresolved
endpoint does `[CERT-doc]`. [ev: retro r8-276-driver-scaffold Δ1]

**Never ship the wizard's ping.** The NDriver wizard generates a `doPing()` whose only statement is `pingOk();` under
`// TODO - add ping implementation` (`devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:102-111`): every device then
reports UP forever. `toolbelt/lint-driver-scaffold.sh <src-root>` FAILs that stub and WARNs on the wizard's other
placeholders (poll TODO, `TODO - convert linkMessage`, a message factory that only returns null, a discovery job
submitted with a null Context). It names a method from its header with annotations dropped, at any class depth, so an
annotated or nested-class stub is caught too. [ev: retro n4-course-handson-ndriver Δ2] [ev: retro r8-t9e-drivers Δ1]
[ev: retro r8-276-driver-scaffold Δ2]

`pingFail` propagates DOWN to every `BProxyExt` under the device via:
`BDevice.updateStatus()` → `BPointDeviceExt.updateStatus()` → `BProxyExt.updateStatus()` sets
`BStatus.DOWN`. [ev: code BDevice.java:400; BProxyExt.java:475; corpus B810 §810.2]

**A failed poll is not a failed ping** `[ev: retro devguide415-r3-drivers-integration Δ1]` `[ev: corpus B1348 §1348.1]`. The
4.15.6.20 guide says "in most drivers" a point poll with no response makes the monitor note a ping fail
(`Drivers/driver-PingMonitor.txt:13`); that is driver policy, not framework behaviour. `BProxyExt.readFail(cause)`
only records the fault on the point and re-executes it (vineflower driver-rt 4.14.0.162
`javax/baja/driver/point/BProxyExt.java:463-469`); nothing in driver-rt calls the device's ping from there. Code wins
for behaviour: in a custom driver, on a transport timeout in the poll path call `device.ping()` (a fresh ping now) or
`pingFail(cause)`, and fail the points only after N consecutive timeouts — the BACnet pattern: `this.ping()`, then
`readFail` on the poll list once `++pollTimeouts > getMaxPollTimeouts()` (vineflower bacnet-rt 4.14.0.162
`javax/baja/bacnet/BBacnetDevice.java:1093-1101`; `maxPollTimeouts` default 0, `:201`). 4.13 and 4.15 not verified.

**Ping monitor rules** `[ev: retro devguide415-r2-drivers-protocols Δ7]` `[ev: corpus B1337 rows 1-3]` (vineflower driver-rt
4.14.0.162 `javax/baja/driver/ping/BPingMonitor.java`):
- A disabled or faulted device is never pinged (`isPingEnabled`, `:216-218`), so its recovery must come from the
  config or read path, not from the ping.
- DOWN is set on the first failed ping; the ping ALARM waits for more than `numRetriesUntilPingFail` consecutive fails
  (default 0) and for `startupAlarmDelay` after station start (default 5 min, `:65-66`).
- A failing device is retried once per `pingFrequency` counted from the last ATTEMPT, not the last success
  (corpus B1337 row 3).

### `configFatal` vs `configFail` — recovery reference card

| Call | Status bit | Recovery | Cleared by |
|---|---|---|---|
| `configFail(cause)` | FAULT | **Transient** — cleared on next successful operation | `configOk()` |
| `readFail(cause)` | FAULT | Transient | `readOk(value)` |
| `writeFail(cause)` | FAULT | Transient | `writeOk()` |
| `configFatal(cause)` | FAULT | **Permanent** — cleared ONLY by a full station restart | station restart |

**`configFatal` is permanent:** `configOk()` clears `configFault` but returns early before
`fatalFault` (`BDevice.java:425,431`); no code path resets `fatalFault`; `checkFatalFault()`
short-circuits component restart (`BProxyExt.java:577-578`, `BDevice.java:484-485`); the proxy ext's `FATAL_FAULT`
flag is locked until station stop (`BProxyExt.java:1134`).
The following do NOT clear a fatal fault: `configOk()`, a config-slot change, a re-ping, an enable
cycle, or a component restart. A point in `fatalFault` is permanently unoperational — tuning stops
dispatching to it.

**Valid uses for `configFatal`:** licensing failure, invalid parentage. Do NOT call `configFatal`
for transient comm conditions (device unreachable, timeout, protocol error) — those belong to
`configFail()` / `pingFail()` so the point self-recovers when the device returns.

**Neither `configFail` nor `configFatal` sets `DOWN`** — `DOWN` is exclusively ping-owned
(`pingFail()` propagates DOWN to every `BProxyExt` under the device).

`[ev: retro module-hardening-reqexec-closed-deltas Δ4]`

### Licensing — how device and point counts are enforced

Facts only, from docSource driver-rt (`D=organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver`
below; full paths in the citations). The guide says only that "Licensing failures can also trigger fatal faults"
(devguide `driverFramework.txt:48`).

- **Two counts run when each component starts.** On `Fw.STARTED` a device calls
  `checkFatalFault(Metrics.incrementDevice(this))` and a proxy ext `checkFatalFault(Metrics.incrementPoint(this))`
  (`organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/BDevice.java:630-633`,
  `organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/point/BProxyExt.java:1055-1059`). The
  first is the station's global capacity count; over it, the component goes fatal with "Exceeded device limit
  for …" / "Exceeded point limit for …" (`BDevice.java:471-479`; `point/BProxyExt.java:564-571`, which calls
  `configFatal`).
- **The driver's own limits apply only if the network names a feature.** `BDeviceNetwork.getLicenseFeature()`
  returns `null` by default, and then no driver limit is checked
  (`organized/docSource/docSource-doc/extracted/driver-rt/javax/baja/driver/BDeviceNetwork.java:581-589,678-681`).
  When it returns a feature, the network checks it (`feature.check()`; a failure makes the network fatal with
  "Unlicensed: …"), and turns every `*.limit` attribute of that feature into a counter, `none` meaning unlimited
  (`BDeviceNetwork.java:594-655`). The counters are pooled per `vendor:feature` for the whole station, so several
  network instances share one limit (`BDeviceNetwork.java:613-625`). Feature attributes are name/value pairs of a
  signed, host-bound license file (devguide `licensing.txt:14,54-58`) `[ev: corpus B14]` `[ev: corpus B301]`.
- **What is counted.** Each started device whose type is the network's `getDeviceType()` adds to `device.limit`;
  each started proxy ext whose type is its point device ext's `getProxyExtType()` adds to `point.limit`
  (`BDevice.java:520-531`; `point/BProxyExt.java:647-658`). The increment is 1; the sub-licence prefix and other
  increments apply only to a fixed list of Tridium types
  (`organized/baja/baja/vineflower/com/tridium/sys/metrics/BISubLicenseable.java:17-34,42-56`), so a custom
  driver's keys are plain `device.limit` / `point.limit`.
- **The faults.** A key missing from the feature counts as limit 0 and faults "Unlicensed: point.limit"; going
  over faults "Exceeded point.limit of N" (`BDeviceNetwork.java:678-697`). Both set the component's fatal fault
  (`BDevice.java:526-531`; `point/BProxyExt.java:653-657`), which only a station restart clears (card above). In
  `BDeviceNetwork` the counter is only ever incremented (`BDeviceNetwork.java:690`): deleting points does not
  return headroom before a restart `[INFER]` (no decrement in that file; other driver-rt paths not searched).
- **Commissioning:** a point or device in fault with one of these causes is a licence problem, not a comm problem;
  fix the licence or the count, then restart the station.

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

[ev: code BPollFrequency.java:25-27,38-47,70; BPollScheduler.java:247,273,299,1222 (docSource); corpus B872 §872.1]

Lifecycle rule: **register in `readSubscribed`, unregister in `readUnsubscribed`**. The scheduler
pushes the point onto the `dibs` stack immediately on subscribe for a first-poll before the bucket
cadence starts. [ev: corpus B872 §872.1]

One poll thread per network, named `"Poll:<networkName>"`. `pollEnabled` is a property of the network's POLL
SERVICE (`BAbstractPollService.java:62`, default true) and gates the entire loop. [ev: corpus B872 §872.1;
corpus B810 §810.3] (B810 is cited in `logic-authoring.md §Authoring a driver`) `[ev: corpus B1337 §1337.2 C1]` [ev: retro devguide415-r2-drivers-protocols Δ8]

---

## 7 · Advanced — history, schedule, virtual, serial

### 7.1 · `BHistoryDeviceExt` + `BHistoryPollScheduler`

A separate `BHistoryPollScheduler` (default rates: fast 10 000 / normal 45 000 / slow 120 000 ms)
drives `BHistoryImport` children that read records from the field device and append them to the
local history DB. Drop a `BHistoryDeviceExt` on your `BDevice` class to opt in.
[ev: code BHistoryPollScheduler.java:72,84,96 (docSource); corpus B872 §872.3]

### 7.2 · `BScheduleDeviceExt` — makeExport / makeImportExt

Abstract; implement both abstract methods to make your driver push/pull schedules.
`processImport` (export down to device) is version-gated: only pushes when `lastModified` differs
from the device's cached version. `subscribeWindow` staggers the initial subscribe to spread
network load. [ev: code BScheduleDeviceExt.java:34; corpus B872 §872.4]

### 7.3 · `BVirtualComponent` / `BVirtualGateway` (virtual: scheme)

`BVirtualGateway extends BComponent` exposes a lazy, transient-only `BVirtualComponentSpace`.
A point inside resolves as the gateway's ORD followed by `|virtual:/A/B/point` — NOT `virtual:/gw/device/point` and NOT
`slot:` (devguide 4.15.6.20 `virtualComponents.txt:68-79`). Each virtual slot is named `SlotPath.escape(virtualPath)`; a load
can add a slot under a parent that is already subscribed, so register the new point with the poll scheduler too. The
space forbids links and structure edits, not property writes; it is not persisted and is GC-cached, tuned by
`niagara.virtual.cache.{maxLifeDefault,minLifeDefault,threshold,thresholdScanRate}` (45 s / 25 s / 1000 / 1000 ms).
Use it for on-demand browsing of large address spaces (e.g. a BACnet router with thousands of objects) where creating
a full proxy tree would be prohibitive. [ev: corpus B28 §28.9-28.12] `[ev: corpus B1306 §1306.1 rows 7-8]`
[ev: retro devguide415-nav-subscriptions-tags-ords Δ3]

### 7.4 · `BSerialNetwork` / `SerialComm` — RS-485 twists

`BSerialNetwork extends BBasicNetwork` adds `serialPortConfig` (`BSerialHelper`:
baud/data/stop/parity/port) and `interMessageDelay`. `SerialComm.performInterMessageDelay()`
sleeps the deficit since the last received message (min 10 ms) before each transmit — a
fixed-ms gap, NOT the baud-relative silence prescribed by Modbus RTU spec.
[ev: code basicDriver-rt vineflower BSerialNetwork.java:34-38, SerialComm.java:146-160; corpus B517 §517.3]

---

### 7.5 · `BAlarmDeviceExt` — alarms raised by the device, not by points `[ev: retro t9c-reference-gaps Δ2]`

Use it when the field device sends its own alarm/event notifications (BACnet event notifications, SNMP traps,
an oBIX alarm feed); per-point `BAlarmSourceExt` stays the choice for limits the station evaluates itself.
Official page: devguide `driverAlarm`. Worked client example: opcUa `AlarmExt`, corpus B1226.
Paths below: `D=organized/docSource/docSource-doc/extracted` (original Tridium source), all under niagara-research.

- **Contract** [CERT] `D/driver-rt/javax/baja/driver/alarm/BAlarmDeviceExt.java`: `public abstract class
  BAlarmDeviceExt extends BDeviceExt implements BIRemoteAlarmSource, BIRemoteAlarmRecipient` (`:66-68`); slots
  `alarmClass` (String, default `defaultAlarmClass`) and `lastReceivedTime`, hidden actions `routeAlarm` /
  `ackAlarm` (`:40-65`); implement `doRouteAlarm(BAlarmRecord)` and `doAckAlarm(BAlarmRecord): BBoolean`
  (`:170-174`); `isChildLegal` returns false (`:179-182`) — override it if the ext holds feeds/processors.
- **Declare it on your device** with your own property — `BDevice` has no alarm-ext hook. BACnet:
  `@NiagaraProperty name="alarms" type="BBacnetAlarmDeviceExt"`. [CERT] `D/bacnet-rt/javax/baja/bacnet/BBacnetDevice.java:155-156,372`
- **Recipe — receive:** build the record with the ext as source, `new BAlarmRecord(ext, ext.getAlarmClass(),
  alarmData, BUuid.make())`, put the field point in the `sourceName` facet, `ext.routeAlarm(record)`, then
  `setLastReceivedTime`. `doRouteAlarm` hands it to `BAlarmService.routeAlarm(record)`. [CERT]
  `organized/bacnet/bacnet-rt/vineflower/com/tridium/bacnet/stack/server/BEventHandler.java:893-895,1330-1335`;
  `D/bacnet-rt/javax/baja/bacnet/alarm/BBacnetAlarmDeviceExt.java:121-163`
- **Recipe — ack:** the alarm service routes an ack to a `BIRemoteAlarmSource` source (`doRouteToSource` →
  `ackAlarm` → your `doAckAlarm`). [CERT] `D/alarm-rt/javax/baja/alarm/BAlarmService.java:1003-1006,1108-1114`.
  `doAckAlarm` sends the ack and returns; route the record again only when the device confirms. [CERT]
  `D/alarm-rt/javax/baja/alarm/BIRemoteAlarmSource.java:10-16`. A device with no remote ack (SNMP) sets
  ackTime, `BAckState.acked`, `ackRequired=false` itself and routes. [CERT]
  `organized/snmp/snmp-rt/vineflower/com/tridium/snmp/alarm/BSnmpAlarmDeviceExt.java:197-203`
- **Pitfalls:**
  - The ext must be the record source, or acks never come back to it; carry the real point in `sourceName`
    or `alarmData`. [CERT-doc] `niagara-help/devguide-clean/driverAlarm.txt:10`
  - Offnormal and normal are two states of ONE alarm: without `AlarmSupport` you transition each offnormal
    record to normal yourself. [CERT-doc] `driverAlarm.txt:12`
  - `BDevice.alarmSourceInfo` / `ackAlarm` are the ping-health alarms, not this ext. [CERT] `D/driver-rt/javax/baja/driver/BDevice.java:590-593,689-696`
  - Catch `ServiceNotFoundException` when the station has no alarm service. [CERT] `BBacnetAlarmDeviceExt.java:163-165`
  - Sending station alarms TO a device is a `BAlarmRecipient` whose `handleAlarm` works off the engine
    thread and skips down/disabled devices. [CERT-doc] `driverAlarm.txt:16`

## 8 · Transport-specific twists — KNX and Z-Wave

### 8.1 · KNX (`eibnetIp` driver)

```java
public class BEibnetIpNetwork extends BLoadableNetwork implements EibnetConst, BIService
```

KNX addressing is by **group address**, not by object/property (unlike BACnet). Each proxy ext holds
its group addresses in a `groupAddresses` slot, a `BGroupAddressFolder` (a `BVector`, eibnetIp-rt `com/tridium/eibnetIp/point/BKnxProxyExt.java:54`). The network owns two stack layers as
`@NiagaraProperty` children: `linkLayer` (`BEibnetIpLinkLayer`) for UDP multicast/unicast
EIBnet/IP discovery, and `tunnelLayer` (`BEibnetIpTunnelLayer`) for the tunneled connection.
Per-datatype proxy families: `BKnxBooleanProxyExt`, `BKnxEnumProxyExt`, `BKnxStringProxyExt` —
pick by KNX DPT, not by a single generic proxy ext. Point discovery reads an ETS export (`.esf` session file)
chosen in the point manager's Discover (eibnetIp-wb `com/tridium/eibnetIp/ui/BKnxPointManager.java:367-379`); `BKnxDiscoverDevicesJob`
is a KNXnet/IP device search. [ev: code eibnetIp-rt vineflower BEibnetIpNetwork.java:28-35; corpus B1357]

### 8.2 · Z-Wave (`zwave` driver)

```java
public class BZWaveNetwork extends BBasicNetwork implements ZWaveMessageConst
```

Z-Wave device discovery ("inclusion") is not a one-shot job — it is a persistent lifecycle.
Each `BZWaveDevice` adds a `BInclusionMonitor extends BComponent` as a hidden, transient `inclusionMonitor`
slot (flags 71, `BZWaveDevice.java:1783-1786`), so it is not persisted; it tracks completion flags
(`commandClassParsingComplete`, `wakeUpDataRetrievalComplete`, `associationsRetrevalComplete`,
etc.) as readonly properties. Inclusion succeeds only when all complete and every command class is processed
(`getIsInclusionSucessful`, `BInclusionMonitor.java:477-479`). [ev: code BInclusionMonitor.java:26-52; corpus B1384]

Command classes: the network maps each class id to a registered `BCmdClassObject` subtype
(`getCommandClassInstance`, `BZWaveNetwork.java:2292-2299`), whose `addCmdClass` adds the matching proxy points
under the device (`BZWaveDevice.java:1081-1084`); `BZWaveCommandClass` is a frozen enum of class IDs. Firmware
upgrade is for the Z-Wave option card: the `.hex` image ships in the jar and is copied to `/tmp` before upload
(`comm/serial/fw/BZWaveFirmwareUpgrade.java:41-54`); device templates (`.info`) are read from the station's
`^zwaveDevTemplates/` first, then from the jar (`BZWaveDevice.java:588-600`). [ev: corpus B1102] [ev: corpus B1384]

---

### `getDeviceManagerSubscribeDepth()` — deep slot-tree discovery `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ20]`

**Override `getDeviceManagerSubscribeDepth()` in a `BNNetwork` subclass to return >1 for deep slot trees (PD-20):** the default implementation is hardcoded to return `1` (`BNNetwork.java:174-183`, docSource). It is the subscribe depth of the auto device manager (`NDeviceModel.getSubscribeDepth()`, ndriver-wb `device/NDeviceModel.java:84-87`, docSource; also sent to the ux manager as `sd`). When the manager's columns must show values deeper in the device tree, return that depth (e.g. `3`). It does not drive discovery or polling: proxy points subscribe through their own exts `[INFER]`.

```java
@Override
public int getDeviceManagerSubscribeDepth() {
    return 3;  // levels below the network the device manager subscribes
}
```

`[ev: corpus B1072]`

**`NMgrControllerUtil.makeCommands()` turns the network's agents into device-manager commands — the NATIVE device-manager-agent extension point (PD-14):** it filters `network.getAgents()` with `AgentFilter.is(BINDeviceMgrAgent.TYPE)` and wraps each agent in an `NDeviceMgrAgentCommand`; the point-manager variant filters the device and point-ext agents for `BINPointMgrAgent` (ndriver-wb `ui/NMgrControllerUtil.java:92-98,121-141`, docSource). This is NOT CCN-specific — any `BNNetwork` subclass may override `getAgents()` to add or drop agents, or register a `BINDeviceMgrAgent` on the network type through the standard agent registry `[INFER]`. `[ev: retro wb-vendor-ux-rt-wb-pattern-deltas Δ14]` `[ev: corpus B1066]`

---

## 9 · Vendor/platform driver reference patterns (wave-3 survey)

Reference archetypes from obixDriver, lonworks, honBACnetUtilities, niagaraDriver, mbus, and opc modules. Our own modules stay at rung 0–1; these sections exist for scope discussions and as copy references only.

### 9.1 · oBIX driver vs. servlet — local-export-discover idiom [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ1]

**Driver vs. servlet decision:** use the Network/Device/ProxyExt driver pattern for bidirectional telemetry where the external system initiates reads/writes and the driver provides subscription + polling. Use a `BWebServlet` / SPA when you need a browser-facing UI on top of data that already lives in the station.

**Local-export-discover idiom** (pushing local N4 data outward, e.g. oBIX export): resolve the `slot:/` ORD to the station component space, then filter children by `BIWritablePoint` to discover exportable points. Iterate the filtered set to build the export payload; no separate Network subclass is needed for a pure-export path `[INFER]`.

```java
// Resolve the local station root and collect its writable points for export
BOrd root = BOrd.make("slot:/");
BComponent space = (BComponent) root.get(base, cx);   // base: any mounted component
for (BComponent child : space.getChildComponents()) {
    if (child instanceof BIWritablePoint) {
        BControlPoint wp = (BControlPoint) child;
        // push wp.getOutStatusValue() to the external system
    }
}
```

Separate the discovery walk from the push loop — discovery is a one-time enumeration; push is per-cycle. [ev: corpus B1094]

### 9.2 · LonWorks — NV-binding UX is not a point list [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ2]

LonWorks commissioning requires three **distinct** manager patterns — do NOT collapse them into a single point-list table:

| Phase | Pattern |
|-------|---------|
| **XIF/LNML typing** | The `.xif` / `.lnml` interface file types each network variable (NV): `BLocalNvManager` keeps the LNML ORD and the parsed type as slot facets (`LnmlOrd`, `XType`), and a device's NV table is learned when the device is added (`learnNv()`), not by a separate scan. A manager that shows raw values before typing misleads `[INFER]`. |
| **Changeable-NV type editing** | NVs whose SNVT type is declared changeable get their own panel, `BChangeableNvManager` (a `BWbComponentView`, not a table manager), which changes the NV type and offers to delete the links and proxy points the change breaks. NV bindings live in `BLonLinkManager`. |
| **Service-pin commissioning** | Pressing the device's service pin raises `BLonNetmgmt.servicePinReceived`; the device manager matches the Neuron ID to a table row and selects it, and `doCommission()` then maps that Neuron ID to a subnet/node. |

NV-binding UX is a separate workflow from point-list monitoring. Merging them into one manager produces a table that is either incomplete (pre-typing) or misleading (post-typing polluted by unbound NVs) `[INFER]`. [ev: corpus B1095]

### 9.3 · OEM-on-stock-driver patterns (honBACnetUtilities) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ3]

When building an OEM module on top of a stock Tridium driver (e.g. the stock BACnet driver), four patterns avoid duplicating the stock driver's architecture:

| Pattern | Mechanism |
|---------|-----------|
| **Type-float slot** | The OEM device manager (`BHonBacnetDeviceManager extends BBacnetDeviceManager`) floats the OEM device type to index [0] in `getNewTypes()`, `doAdd()` and `doNew()`, so it is the default choice — no Manager subclass per device family. |
| **Two-anchor manager mount** | One manager is `@AgentOn` two tree locations (`BHonBacnetAlarmManager` on `HonBacnetAlarmDeviceExt` and `HonBacnetNotificationClassFolder`) and dispatches on `instanceof` in `doLoadValue()`. |
| **Dual FE registration** | A thin subclass of the stock FE is `@AgentOn` the stock type (`BHonBacnetBitStringEditor extends BBacnetBitStringFE`, WB render path only) next to a fresh `BWbFieldEditor` `@AgentOn` the OEM wrapper type (`BHonBacnetBitStringFE`); each FE is registered on its own type, so they coexist. |
| **ORD-carrier navigation** | rt point types carry an ORD (`getAlarmViewHyperlink()`); a WB dialog reads it per point and `BHonWbService.hyperlink(ord)` syncs the space on a background thread, then hyperlinks. |

See also `types/wb-widgets.md §Plugin-extensible device-type manager via SPI` for the complementary WB-side pattern. [ev: corpus B1096]

### 9.4 · Command-driven manager SPI — BStationMgrCommand registry discovery [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ4]

`BStationMgrCommand` is a registry-discovered command SPI for adding toolbar commands to the Niagara-driver station manager (`BStationManager`, `@AgentOn niagaraDriver:NiagaraNetwork`). Key points:

- Implementers register via a plain `<type>` in `module.xml`; `makeCommands()` discovers them via `Sys.getRegistry().getConcreteTypes(BStationMgrCommand.TYPE.getTypeInfo())` and keeps those whose `isCommandActive(mgr)` is true (niagaraDriver-wb vineflower `BStationManager.java:302-311`) — no static registration call required.
- **Session-keyed learn state:** `BNiagaraFileManager`'s `State` saves the Fox session id it learned under and drops the cached learn roots when the id changes on restore (`file/BNiagaraFileManager.java:238-266`), so a reconnect never shows a stale learn table. Do NOT keep session learn state in a static map.
- **Offline `.bog` guard:** `BStationMgrTable.reload()` checks `getSpace().getType().is(BBogSpace.TYPE)` and never subscribes `FoxService.serverConnections` against an offline `.bog` (`BStationMgrTable.java:43-48`); gate live-only work the same way.
- **`CredentialsColumn` lease + `newCopy`:** the column's `load()` leases the `BClientCredentials` and returns `val.newCopy()` with the `BPassword` copied in; `save()` copies the same way (`BStationManager.java:330-370`). Never hand the live `BCredentials` to the edit row.
- **Shorthand `HistoryId`:** match an existing import with `id.equals(impId) || impId.equals(id.toShorthand(stationName))` (`history/NiagaraImportLearn.java:85-86`); `BHistoryId.make(deviceName, historyName)` builds an id (history-rt `BHistoryId.java:54`).

[ev: corpus B1097]

### 9.5 · Field-bus protocol driver archetype — dual-addressing discovery wizard (M-Bus) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ7]

M-Bus and similar field-bus drivers require a **dual-addressing discovery wizard**: `BMbusDeviceManager.doDiscover()` offers five discovery modes (database scan, single device, primary-address range, full secondary scan, secondary-address scan) across both address spaces (B1100); the scan shapes below are protocol background `[INFER]`:

| Phase | Address type | Range | Manager pattern |
|-------|-------------|-------|----------------|
| **Primary scan** | Primary address (1 byte, 0–250) | Linear; one request per address | Progress bar; abort on first valid response per address |
| **Secondary scan** | Secondary address (8 bytes: manufacturer + ident + medium + version) | 2³² filtered bitmask walk | Wildcarded broadcast; narrows on each confirmed response |

Additional patterns for field-bus protocol drivers:
- **Multi-baud scan:** the wizard must iterate across supported baud rates (300 to 38400, mbus-rt `BMbusBaudRate.java:47-54`) if the device baud rate is unknown at commissioning; cache the confirmed baud on the device component.
- **Manufacturer-specific data model:** `DataRecord` (DIF + VIF + data) is decoded differently per manufacturer extension code; the driver keeps manufacturer-specific layouts as station data: a `ManuSpecDataNetworkExt` holds `ManuSpecDef` components whose elements give `offset`/`length`/`byteMask`/`bitMask`/`orFlag`, edited by `BManuSpecDefManager`/`BManuSpecDefElementManager` and referenced via `manuSpecDefElementOrd`, so a new manufacturer needs no code change.
- **Hardcoded-facet lint candidate:** when a history import assigns `facet precision=4` independently of the proxy point's `-exponent` facet, the two can diverge silently. Flag this mismatch in import-learn flows (see `types/issues-and-gotchas.md §H1` for the proposed lint). [ev: corpus B1100]

[ev: corpus B1100]

### 9.6 · Protocol-adapter manager — action-slot bridge and structured error decode (OPC) [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ10]

This section is about the classic OPC DA (COM) manager. For OPC UA client/server and for choosing a data-egress channel see § 10. `[ev: retro t9c-reference-gaps Δ7]`

OPC and similar protocol-adapter drivers expose three recurring WB patterns that differ from standard BACnet/Modbus drivers:

**Action-slot bridge:** OPC items are proxy points (`OpcPointModel.newInstance()` auto-attaches a `BOpcProxyExt`); the WB→rt→COM bridge is action-slot invocation (`submitDeviceDiscoveryJob`, `discover()`, `invoke(BOpcDaClient.setSecurity)`). Keep the COM/native error handling inside the driver layer: the JNI bridge throws `HresultException` (an `OpcException`) whose `toString()` decodes the HRESULT with `OpcEnv.getDescription` (opc-rt vineflower `jni/HresultException.java:6-24`), and the point device ext turns a failure into `writeFail(msg)` on the points (`client/point/BOpcPointDeviceExt.java:668-681`).

```java
// Inside the driver comm layer: a failed COM call through the JNI bridge
try {
    syncIo = peer.getSyncIo();
} catch (Exception e) {                                   // HresultException: "[hex - description] msg"
    for (BOpcProxyExt ext : points) ext.writeFail(e.getMessage());
    return;
}
// The WB view never sees the exception; it reads the point's fault cause
```

**Lazy hierarchical browse:** OPC address spaces can contain millions of nodes; do NOT eagerly expand the full tree on manager open. `OpcPointLearn.getChildren()` checks a lazy flag (`getKidCheck()`) and only then calls `r.discover()` (COM `IOPCBrowse.Browse()`) under a 10 s lease; the manager clears the learn table on focus because COM browse sessions expire.

**Security-gated state:** OPC security (DCOM, certificate-based OPC-UA) may leave the connection in a partially authenticated state. `BOpcPointManager.isFocusTraversable()` reads `BOpcDaClient.getServerSecured`; if it is set, it clears it and marks `IsUserLoggedIn`, and saved learn state is never restored while `IsUserLoggedIn` is set. There is no `SECURITY_FAULT` status in `BStatus`; security edits go through `BOpcDASecurityView`. [ev: corpus B1103]

## 10 · OPC UA and the data-egress choice `[ev: retro t9c-reference-gaps Δ7]`

Full guides (read them, do not re-derive): niagara-research `docs/opc-n4-guide.md` (recipes §2-§4, type and
status mapping §5, pitfalls §8, 4.15 differences §11) and `docs/n4-data-access-guide.md` (decision table §1,
channel families §4, security baseline §5). Corpus blocks B1222-B1281. Code paths below are under
niagara-research `organized/`; `C=opcUaClient/opcUaClient-rt/vineflower/com/tridium/opcUaClient`,
`S=opcUaServer/opcUaServer-rt/vineflower/com/tridium/opcUaServer`.

- **Two stacks.** OPC DA = module `opc` (COM/DCOM, Windows only, licence `tridium:opc`; § 9.6). OPC UA =
  `opcUaCore` + `opcUaClient` + `opcUaServer` (`opc.tcp`, JACE and Windows). [CERT]
  `opc/opc-rt/vineflower/com/tridium/opc/client/BOpcNetwork.java:323-324`; `docs/opc-n4-guide.md` §7; `[ev: corpus B1222]`
- **Client** is an ndriver stack: `BOpcUaNetwork extends BNNetwork` (licence `tridium:opcUaClient`),
  `BOpcUaDevice`, `BOpcUaClientPointDeviceExt`, `BOpcUaClientProxyExt`. Reads are subscriptions only
  (`publishInterval`), no poll. [CERT] `C/BOpcUaNetwork.java:42,70-71`; `docs/opc-n4-guide.md` §3, §7
- **Server** is a driver network too, not a service: `BOpcUaServer extends BNNetwork` (licence
  `tridium:opcUaServer`) → `BOpcUaNamespace extends BNDevice` → server proxy exts. It exposes ONLY the points
  you add; each is Export (links the source `out`, read-only node) or Import (links into an `inN`, offered
  only for writable sources). [CERT] `S/BOpcUaServer.java:175,344-345`; `S/BOpcUaNamespace.java:121`;
  `S/point/BOpcUaServerProxyExt.java:148-152`; `S/point/BOpcUaServerLearnPointEntry.java:121-128`
- **Namespace-index drift is handled for discovered points:** on subscribe the client rewrites `ns=` from
  the stored `nameSpaceUri`; a hand-made point with an empty `nameSpaceUri` is not remapped. [CERT]
  `C/point/PointSubscribeCmd.java:34-40`
- **Certificates** live in the station's CertManagerService key/trust stores; approve the peer per
  `host:port` in Allowed Hosts, again after a cert is regenerated; use Basic256Sha256 + SignEncrypt and turn
  off Basic128Rsa15 and anonymous tokens. `docs/opc-n4-guide.md` §3, §4, §8 `[ev: corpus B1244]`
- **Server status and authorization differ by release:** 4.14 publishes every value as Good and has no
  per-user authorization; 4.15.3.28+ sends Bad/Good_NoData and has RBAC for user-name sessions only.
  `docs/opc-n4-guide.md` §5, §11 `[ev: corpus B1280]`

**Egress decision** — the full per-job table is `docs/n4-data-access-guide.md` §1; the module builder's cut:

| Consumer | Channel | Who initiates | Note |
|---|---|---|---|
| SCADA / third-party read-only | OPC UA server, curated namespace | client pulls/subscribes | UserName + SignEncrypt; RBAC only on 4.15.3+ |
| Off-station app, BI, alarm feed to systems | oBIX server `/obix` | client pulls (Watch, history query, `alarmFeed`) | a read user sees the whole tree; needs the oBIX export licence |
| Cloud, outbound only | MQTT (`types/cloud-connector.md`) or an edge poller | the station pushes | strings only; publishes drop while disconnected |
| Another BMS | BACnet/IP export | peer reads/writes | no auth on BACnet/IP; writes run as station user `BACnet` |
| On-site HMI, narrow write API | custom `BWebServlet` (`dashboard.md`) | browser pulls JSON | write with the real user Context; CSRF is yours |
| Station-served page, few live values | BajaScript (`dashboard.md` § Reading the station directly) | browser subscribes | the user's own permissions |

## 11 · Field-bus rules — BACnet export, MS/TP and Modbus

Rules that decide field behaviour when our module's points sit on, or are exposed through, the stock BACnet and
Modbus drivers. Guide paths are `niagara-help/versions/4.15.6.20/guides-clean/`.

### 11.1 · Exposing values over BACnet `[ev: retro devguide415-r2-drivers-protocols Δ5]`

- **Never enable an export priority our module drives.** Each priority level ticked in a writable export
  descriptor's `bacnetWritable` links that `inN` of the source point to the descriptor; the guide warns "Do not
  select any priority level already linked or in use by the system, otherwise control contention occurs"
  (`Bacnet/AddandEditExportObjectsWindow-540E5726.txt:27`). Reserve our level (`in8`, `types/logic-authoring.md`
  § Authoring a driver) and list the levels left to remote writers in the module README.
- **Remote writes run as the station user `BACnet`,** which has no permissions until granted: give it Operator
  write on the category that holds the exported points and a non-blank password
  (`Bacnet/ConfiguringExportedObjectsForWriteB-5410ADDB.txt:7-29`).
- **Exported enums need ordinals 1..n with no gaps.** An `EnumPoint`/`EnumWritable` whose range starts at 0 or
  has gaps "does not export to BACnet", with no error (`Bacnet/TypesOfObjectsToExportBACnet-540D731F.txt:17`).
  Import-side non-contiguous ranges: `types/logic-authoring.md` § BEnumRange authoring grammar (its Non-contiguous-ordinal gotcha). Gotcha H5.
- **The N4 level is the BACnet priority.** A proxy write goes out as `WriteProperty` at the winning N4 level and
  relinquishes the old level (vineflower bacnet-rt 4.14.0.162 `javax/baja/bacnet/point/BBacnetProxyExt.java:528-547`;
  corpus B544 §544.5), so a device program writing at priority 1-7 beats our `in8` in the device's own array.
  A Px virtual-point write goes out at priority 16 unless the ORD carries `;priority=N`
  (`Bacnet/ConfiguringVirtualPointsInPxViewsBA-5398ED55.txt:29`, `Bacnet/VirtualComponentsBACnet-53942D00.txt:59-60`). `[ev: corpus B1327 rows 4-7]`
- Owed Go lints G85 `bacnet-export-inN-contention` and G86 `bacnet-export-enum-range` (ledger).

### 11.2 · MS/TP and COV `[ev: retro devguide415-r2-drivers-protocols Δ6]`

- **COV is a tuning-policy setting:** `Use Cov`, `Cov Subscription Lifetime` (default 15 min) and `Accept
  Unsolicited Cov` (default false) live on the BACnet tuning policy (`Bacnet/bacnet-BacnetTuningPolicy.txt:24-29`);
  bind MS/TP and IP devices to separate policies (`Bacnet/AddingATuningPolicyBACnet-4FD602EE.txt:11`).
- **Large MS/TP trunks:** a Boolean `skipUpload=true` slot on a `BacnetDevice` skips the device-object upload
  at station start (`Bacnet/TroubleshootingBACnet-4FF09FBA.txt:27,57-58`); a duplicate Device ID faults the
  device (`:17`).
- **Worst-case wait per request = `apduTimeout` × (`numberOfApduRetries` + 1)** — defaults 3000 ms and 3 on
  `BLocalBacnetDevice`; ping and poll threads queue behind it (corpus B1190 §1190.2, live on the PANCCADIA TC500:
  10 s × 4 = 40 s). Raise the timeout only with the retry count in view.

### 11.3 · Modbus data `[ev: retro devguide415-r2-drivers-protocols Δ6]`

- Integer is unsigned 16-bit; Long/Float take two registers; Double and Long 64-bit take four (new in 4.15)
  (`Modbus/ModbusData-5FD097AE.txt:18,23,28,34,37`).
- Byte order for 32/64-bit values is set on the network or the device, default `Order3210` (big-endian)
  (`Modbus/modbusAsync-ModbusAsyncNetwork.txt:25-28`). Modbus carries no type metadata, so a wrong order still
  reads `{ok}` with a zero or absurd value `[INFER]` — check one known value at commissioning. Gotcha H4.
- Writes are rounded and clamped by data type (Integer 0..65535, Signed Integer -32768..32767; Float and Double
  not rounded) (`Modbus/ModbusData-5FD097AE.txt:64-79`).
- Wire addresses are zero-based: holding register 108 in a vendor sheet is address 107
  (`Modbus/ModbusRegisters-5E427D5B.txt:33`).
- Modbus has no priority array: the 16 N4 levels resolve to one value before the wire (corpus B544 §544.5).
  `[ev: corpus B1327 row 10]`

### 11.4 · Weak peers — capability overrides and queue depth `[ev: retro devguide415-r2-drivers-protocols Δ10]`

- **Capability overrides are a last resort.** For a BACnet device that claims what it cannot deliver, the
  bacnetUtil children `RpmOverride`, `SegmentationOverride`, `ApduSizeOverride` and `ServicesSupportedOverride`
  correct the claim; the guide calls them "features of last resort", temporary until firmware or network fixes
  (guide 4.15.6.20 `EngNotes/BACnetOverridingADevice.txt:19`). Record each one in the site runbook.
- **Expose bounded queues as diagnostics.** Stock MS/TP diagnostics count queue-full events (`rxQ_full`,
  `apptxq_overflow`; `EngNotes/bacnetUtil-Tokens.txt:18-20,59-60`). A custom driver should show its own comm or
  write queue depth and overflow count the same way (the spy page in `types/observability.md` § 2, plus a
  read-only transient slot when a dashboard or alarm must see it `[INFER]`) and baseline it per trunk.
  `[ev: corpus B1336 rows 4,6]` Watch-client leases and 503 back-off: `types/cloud-connector.md` § 10.

---

**See also:** `types/logic.md` (BStatus bits, BControlPoint), `types/logic-authoring.md`
(§Authoring a driver — write-path safety checklist, writeOnUp/fallback lints).
