<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 8 -->
# 2026-10-16 · kit · n4-course-handson-ndriver

**Session**: hands-on Lab 11 (`trafficLightDriver` NDriver against the course `trafficLightServer`), Certification Exam P3 (`websiteDriver`, TCP reachability ping) and the Bonus Lab (device/point managers and discovery), all generated with the New Driver Wizard and verified live on 4.15.6.20.
**Delta count**: 8

## What happened
Two NDriver modules were generated with the New Driver Wizard, completed, built and run against live targets:
the course traffic-light server over TCP and a public web host. The live run showed that the kit's own driver
page teaches a `doPing` signature that does not exist in 4.15, that the wizard's ping stub reports success with no
I/O, that a fresh TCP comm config binds the server port, and how the manager agent, columns, New-types filter and
discovery job are actually wired. It also measured how often wizard output uses `com.tridium.*` classes, which
matters for the private-API lint proposed on 2026-10-15.

## Evidence
- [CERT] `javap -cp driver-rt.jar javax.baja.driver.BDevice` (4.15.6.20): `public abstract void doPing() throws java.lang.Exception;`,
  `public void pingOk();`, `public void pingFail(java.lang.String);`. The kit shows `doPing(cx)` in the class ladder
  (`types/driver-authoring.md:15`) and `public BObject doPing(Context cx)` with `return pingOk()` / `return pingFail(…)`
  in § 5 (`types/driver-authoring.md:145-153`).
- [CERT-live] Exam P3 `BWebsiteDriverDevice.doPing()` (`CertificationExam/websiteDriver/…/BWebsiteDriverDevice.java`):
  `try (Socket socket = new Socket()) { socket.connect(new InetSocketAddress(getWebsite(), getPort()), 2000); pingOk(); } catch (IOException ex) { pingFail(ex.getLocalizedMessage()); }`
  compiled against the javap signatures and pinged `{ok}` against `www.google.com:443` (live 2026-10-16) `[ev: engram obs 11009]`.
- [CERT] Wizard template `devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:104-111`: `public void doPing() { // TODO - add ping implementation // if() pingOk(); // else //  pingFail(…); }`
  — an unconditional `pingOk()`. The other wizard stubs left in the exam tree: `// TODO add poll support`
  (`BWebsiteDriverDevice.java:231`), `// TODO - convert linkMessage driver specific NMessage` + `return null;`
  (`WebsiteDriverMessageFactory.java:27-28`), `// TODO add code to detect beginning and end of message`
  (`WebsiteDriverLinkMessage.java:32`).
- [CERT-live] Lab 11: a fresh `TrafficLightDriverNetwork` came up with Tcp Config `Server Enabled = true`; the comm
  tried to bind port 23, already held by `trafficLightServer` → Tcp Config fault `Address already in use: JVM_Bind`,
  Health `Can not connect to local:0`, Status `{down}`; `Server Enabled = false` + Save + Ping fixed it
  `[ev: engram obs 11476]`. The lab text never mentions the field (grep `server enabled` in
  `sources/n4-dev-course/text/Lab Instructions.txt`: 0 hits); only the p.107 screenshot shows it. `javap` lists
  `serverEnabled` as a `Property` of `com.tridium.ndriver.datatypes.BTcpCommConfig`.
- [CERT-live] Lab 11 enum points: the facets dialog needed "Use Frozen Enum in Range" with
  `trafficLightDriver:TrafficLightState` to show light states `[ev: engram obs 11477]`.
- [CERT-live] Bonus Lab, verified live `[ev: engram obs 11479]`: Part 1 — device column via a facet on the device
  property, `facets = @Facet("SfUtil.incl(SfUtil.MGR_EDIT)")` (`BTrafficLightDriverDevice.java:65-69`), shown and
  editable in DeviceManager and DeviceUxManager. Part 2 — `BTrafficLightDriverPointManager extends BNPointManager`
  with `Model extends NPointModel`, `Controller extends NPointController`, column
  `new MgrColumn.PropPath(new Property[] {BControlPoint.proxyExt, BTrafficLightDriverProxyExt.lightId}, MgrColumn.EDITABLE)`,
  and `getNewTypes()` built from `addEnumPointTypes` / `addStringPointTypes` / `addNumericPointTypes` so New offers
  only Enum/String/Numeric. Parts 3-4 — device discovery listed intersections, point discovery listed lights.
- [CERT] Discovery submit: the wizard emits `doSubmitDiscoveryJob(BNDiscoveryPreferences preferences)` ending in
  `return job.submit(null);` (`devkit-wb.jar!gradle/ndriver/BNfooNetwork.java.vm:195-205`); the course reference
  used live takes `(BNDiscoveryPreferences preferences, Context cx)` and returns `job.submit(cx)`
  (`BTrafficLightDriverNetwork.java:290-299`).
- [CERT-live] New Driver Wizard UI (2026-10-16, reported by the session): unchecking Point Device Ext leaves
  "PointManager supports discovery" checked but disabled (no effect on output); base package and vendor are taken
  from the root project.
- [CERT] Course reference device keeps `private static final Set<String> subscribedPoints = new HashSet<>();`
  (`BTrafficLightDriverDevice.java:414`) — one set shared by every device instance `[ev: engram obs 11477]`.
- [CERT] Private-API exposure of wizard output: 79 `import com.tridium.ndriver…` lines in 32 of 36 `.java` files of
  the two driver modules, plus one `import com.tridium.driver.util.DrUtil` (`BTrafficLightDriverProxyExt.java:21`);
  `BNDevice`, `SfUtil`, `BNPointManager` all live in `com.tridium.ndriver.*`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | CORRECT the ping API: class-ladder row → `doPing()`; § 5 snippet → `public void doPing() throws Exception { … if (ok) pingOk(); else pingFail("no response"); }` with `pingOk()`/`pingFail(String)` as void calls; keep the DOWN-propagation text. Lint: YES — a kit self-check (`kit-links.bats`-style grep) FAILs on `BObject doPing` or `return pingOk` in `types/*.md`. Acceptance: the snippet matches the `javap` signatures above and compiles in a wizard-generated `BNDevice` subclass. | `types/driver-authoring.md:15` + § 5 Health (:142-153) | `[ev: code javap driver-rt.jar BDevice 4.15.6.20]` |
| Δ2 | Add a wizard-stub lint (silent-success trap): FAIL when a `doPing()` body's only executable statement is `pingOk();` or still contains `TODO - add ping implementation`; WARN on the remaining wizard markers in shipped `src/` (`TODO add poll support`, `TODO - convert linkMessage`, a message factory that only `return null;`, `TODO add code to detect beginning and end of message`). Lint: YES. Acceptance: a fixture copied from `BNfooDevice.java.vm:104-111` FAILs; the exam socket ping and the course reference (`pingOk()` under `if (pingResponse.getPingSuccessful())`) PASS; the exam tree WARNs on its three remaining TODO stubs. | new `toolbelt/lint-driver-scaffold.sh` wired into `report-module.sh` + `types/driver-authoring.md` § 5 | `[ev: code devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:104-111]` |
| Δ3 | Add a "reachability-only device" ping recipe (device with no protocol of its own): `Socket.connect(new InetSocketAddress(host, port), <timeoutMs>)` in try-with-resources → `pingOk()`, `IOException` → `pingFail(ex.getLocalizedMessage())`, with the station-scoped `NETWORK_COMMUNICATION` grant. Confirms `n4-course-build-deps-permissions-signing` Δ2 live (the grant is required and is inlined into `module.xml`); its `hosts="*"` + `ports="*"` rule must stay a WARN, not a FAIL, because the exam accepted that grant. Lint: no (Δ2 of that retro covers the permission cross-check). Acceptance: the recipe text equals the exam method and states the 2000 ms timeout bounds the `Ping:` thread. | `types/driver-authoring.md` § 5 Health | `[ev: live 2026-10-16 exam P3 ping {ok} www.google.com:443]` |
| Δ4 | Add the client-driver comm rule: `BTcpCommConfig.serverEnabled` defaults to true on a new network, so a CLIENT driver binds the listening port itself and faults with `Address already in use: JVM_Bind` when the real server runs on the same host; set it false in commissioning, and in a client-only driver override the default in the `B<X>TcpCommConfig` subclass (`@NiagaraProperty(name = "serverEnabled", type = "boolean", defaultValue = "false", override = true)` `[INFER]`, not yet built). Lint: partial — WARN when a `BTcpCommConfig` subclass of a module whose sources never call a server-side API leaves `serverEnabled` at its default `[INFER]`. Acceptance: § text names the fault string and Health `Can not connect to local:0`; the commissioning checklist has the line. | `types/driver-authoring.md` (NDriver section proposed by `n4-course-drivers-bql-ords-files` Δ3) + commissioning checklist | `[ev: engram obs 11476]` |
| Δ5 | Add the enum-point facet step: an enum proxy point shows named states only when its facets use "Use Frozen Enum in Range" with the driver's `module:Type` (`trafficLightDriver:TrafficLightState`); otherwise it shows ordinals. Lint: no. Acceptance: the commissioning checklist and the driver page carry the step with the course example. | `types/driver-authoring.md` § 2 `BProxyExt` point SPI + commissioning checklist | `[ev: engram obs 11477]` |
| Δ6 | Add the NDriver manager + discovery recipe, all verified live: device columns by property facet `SfUtil.incl(SfUtil.MGR_EDIT)` (or `MGR_EDIT_READONLY`); a point manager is `@NiagaraType(agent = @AgentOn(types = {"<mod>:<X>PointDeviceExt", "<mod>:<X>PointFolder"}))` extending `BNPointManager` with `NPointModel`/`NPointController`; extension columns are `MgrColumn.PropPath(new Property[] {BControlPoint.proxyExt, <ProxyExt>.<slot>}, MgrColumn.EDITABLE)`; `getNewTypes()` restricts New with `add<Kind>PointTypes(list, true)`; `doSubmitDiscoveryJob(prefs, Context cx)` submits with `job.submit(cx)`. Confirms `n4-course-drivers-bql-ords-files` Δ3 ("manager columns via `SfUtil.incl(flag)`") and `n4-course-threading-jobs-doctrine` Δ6(b) (submit with the caller's Context). Lint: YES — WARN on `.submit(null)` inside a `doSubmitDiscoveryJob` body. Acceptance: the wizard template shape WARNs; the course reference does not. | `types/driver-authoring.md` (NDriver section) + `types/wb-widgets.md` § Manager recipe | `[ev: engram obs 11479]` |
| Δ7 | Confirms `n4-course-drivers-bql-ords-files` Δ3 static-collection WARN with a real instance: Tridium's own course reference keeps the per-device subscription set `static` (`BTrafficLightDriverDevice.java:414`), so copying a reference is exactly how the defect enters a module; the WARN must also cover `private static final Set/Map/List` initialised with `new`. Lint: YES (that retro's WARN). Acceptance: the course reference device WARNs at :414. | `types/driver-authoring.md` (NDriver defect checklist) + the lint of that Δ3 | `[ev: code N4_Labs/…/BTrafficLightDriverDevice.java:414]` |
| Δ8 | Refine `n4-course-build-deps-permissions-signing` Δ3 (private-API WARN on `import com.tridium.`): the allow-list must include `com.tridium.ndriver.**` (the wizard's framework: 79 imports in 32 of 36 files) and should list `com.tridium.driver.util.DrUtil` as a known reference use; otherwise every NDriver module floods the WARN and it gets ignored. Lint: YES (allow-list). Acceptance: the two course driver trees produce 0 rows with the allow-list and 80 rows without it. | `toolbelt/verify-module.sh --src` (that Δ3) + `types/module-wiring.md` § Private API | `[ev: live 2026-10-16 grep course driver sources]` |

> **Corrected (2026-10-24, quality round 10) — Δ7:** the row and Evidence lines 52-53 put the static set at
> "`BTrafficLightDriverDevice.java:414`" and call that file "Tridium's own course reference". Reading A (this
> retro): the course reference has `private static final Set<String> subscribedPoints = new HashSet<>();` at :414.
> Reading B (both files read): :414 is right in the session's own lab module
> (`C:\N4-Developer-Training\Core-Class\N4_Labs\niagara_dev_home\trafficLightDriver\trafficLightDriver-rt\src\com\tridiumuniversity\trafficLightDriver\BTrafficLightDriverDevice.java`,
> 418 lines, the reference copied whole plus wizard-added facets), which is outside the corpus; the course reference
> in the corpus has the same line at :317 (`sources/n4-dev-course/course/Labs/Lab11 NDriver/4 Reading and Writing
> Points/BTrafficLightDriverDevice.java`, 321 lines; the Bonus Labs device-manager copy has it at :318). Winner: B
> for the reference's line; the static-set finding itself is right in both files. It was half right at the time:
> the line was correct for the file the session had open, only the label "course reference" was loose. The row text
> and this retro's review-status are left as they were.

## Lessons
- Check every API shown in a kit snippet against `javap` of the target `niagara_home`; the driver page's
  `doPing` would not compile.
- A wizard stub that reports success is worse than one that fails: a device stays `{ok}` with no I/O at all.
- Field defaults matter as much as code: one default (`serverEnabled = true`) took the network down.
- Tridium reference code is the source to mirror for shape, not for every detail; the `static` set shows a
  reference can carry a defect the kit lints for.

---
**Status**: FOLDED 2026-10-23 (quality round 8 U5, branch `docs/r8-u5-c59-c62`) — Δ1-Δ4, Δ6 earlier (WU-P0-5,
C83/C88/C89/C91); Δ5 `types/driver-authoring.md` § 2 (enum range step); Δ7 § 1.7 defect checklist, lint LANDED 2026-10-24 as G66;
Δ8 (allow-list `com.tridium.ndriver.**` + `DrUtil`) LANDED 2026-10-24 as the default allow-list of lint G65,
cited in `types/module-wiring.md` § 4.5 (retro r10-dr3-driver-lints).
