<!-- review-status: pending -->
# Kit-delta apply plan — module corpus (2026-09-20)

This document is a single prioritized worklist across all pending propose-never-apply
delta retros from the 2026-09-19/20 module-corpus research campaign. Nothing here edits
the kit; the plan ranks the order in which a reviewer should evaluate and fold each Δ
token into the kit core. Deltas are sorted by safety tier (A → D) so the highest-risk
items are reviewed first. All 9 pending delta retros (including the module-hardening
failure-modes retro) are complete and committed; their Δ tokens are categorized below.

---

## 1. Inventory

| Retro file | Module / focus | Δ count | review-status |
|---|---|---|---|
| 2026-09-19-apillm-headless-servlet-rt-4.14-deltas.md | kit / Apillm servlet-in-rt | 24 | pending |
| 2026-09-20-wb-vendor-ux-rt-wb-pattern-deltas.md | kit / wb-vendor-ux wave-1/2 | 25 | pending |
| 2026-09-20-honeywell-wb-rt-wb-deltas.md | kit / Honeywell WB patterns | 8 | pending |
| 2026-09-20-wb-manager-framework-deltas.md | kit / BAbstractManager framework | 4 | pending |
| 2026-09-20-wb-field-editors-deltas.md | kit / BWbFieldEditor framework | 3 | pending |
| 2026-09-20-our-dashboard-audit-deltas.md | kit / DashboardPan ODA audit | 5 | pending |
| 2026-09-20-wb-vendor-ux-wave3-vendor-drivers-deltas.md | kit / wb-vendor-ux wave-3 | 14 | pending |
| 2026-09-20-module-hardening-reference-cards-deltas.md | kit / hardening REF cluster | 8 | pending |
| **2026-09-20-module-hardening-failure-modes-deltas.md** | kit / hardening failure-modes | **16** | **pending review (retro COMPLETE + committed)** |
| **TOTAL (8 retros read + 1 placeholder)** | | **107** | |

---

## 2. Prioritized worklist

Tiers: **(A) SECURITY · (B) SILENT-CORRUPTION/DATA-LOSS · (C) ACTIONABLE-CORRECTNESS · (D) REFERENCE/DOC-ONLY**

### Tier A — SECURITY (7 items)

| Rank | Delta (PD token) | Source retro | Target kit file / § | Why this tier |
|---|---|---|---|---|
| A1 | apillm Δ13 | apillm-headless-servlet-rt-4.14-deltas | `toolbelt/lint-servlet.sh`; `types/security.md` | Every BWebServlet response path must set `X-Content-Type-Options: nosniff`; missing from current DashboardPan `setApiHeaders()` and the scaffold default; lint gate absent |
| A2 | PD-ODA1 | our-dashboard-audit-deltas | `docs/module-best-practices.md` MBP2; `types/security.md`; lint-servlet | Same header-hygiene gap from the ODA2 security audit; adds `Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'` for HTML paths; propose as lint check; **pair with A1** |
| A3 | PD-23 | wb-vendor-ux-rt-wb-pattern-deltas | `types/security.md` | Outbound vs inbound security are separate axes: `enableNonDriverClients` gate governs outbound HTTP; `BAuthenticationScheme` SPI governs inbound auth — conflating them produces wrong trust models |
| A4 | honeywell Δ3 | honeywell-wb-rt-wb-deltas | `types/security.md`; `types/wb-widgets.md` | Fail-closed authorization default: missing authorization mixin/PIN → control DISABLED + HIDDEN (never open-by-default); authoring rule absent from kit |
| A5 | honeywell Δ7 | honeywell-wb-rt-wb-deltas | `types/security.md`; `types/observability.md` | Redact `Authorization`/credential headers in structured logging even at FINEST — cloud/http connector hardening |
| A6 | PD-WV3-security (Δ13) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/security.md`; `lint/` new: `no-md5-credential-digest` | Prefer SHA-256 or stronger over MD5 when storing credentials inside a Niagara module; MD5 observed in honAdvWirelessCfg credential digest (corpus B1104); lint candidate |
| A7 | PD-MH-REF5 (Δ7) | module-hardening-reference-cards-deltas | `types/logic.md`; `docs/module-best-practices.md` MBP1; `lint/` candidate: `no-bformat-untrusted-input` | A `BFormat` pattern constructed from untrusted input is reflective code execution; `FormatDenylist` exists for this reason; authoring rule + lint candidate absent |

---

### Tier B — SILENT-CORRUPTION / DATA-LOSS (5 items + failure-modes pending)

| Rank | Delta (PD token) | Source retro | Target kit file / § | Why this tier |
|---|---|---|---|---|
| B1 | PD-MH-REF2 (Δ2) | module-hardening-reference-cards-deltas | `types/logic.md` §RT-control-logic; `docs/module-best-practices.md` MBP1 | `isValid()` not `isOk()` for control math: `isOk()` returns false on alarm/override, silently dropping control output even when the sensor value is usable — fail-to-danger; highest-priority actionable delta in the REF cluster |
| B2 | PD-MH-RUN1 (Δ8) | module-hardening-reference-cards-deltas | `types/logic.md` §RT-control-logic; `docs/module-best-practices.md` MBP1; `lint/` candidate: `timer-rearm-not-in-finally` | `changed()` / timer / lifecycle callbacks are BARE (no framework wrapper); a self-rescheduling timer that throws before re-arm silently stops; wrap callback bodies in try/catch and re-arm timers in finally |
| B3 | PD-MH-REF7 (Δ5) | module-hardening-reference-cards-deltas | `docs/how-to-create-an-n4-module.md` §slot-authoring; `docs/module-best-practices.md` MBP1 | Always call `SlotPath.escape()` on names from external data (BACnet device names, meter IDs, spreadsheet rows); leading digits and spaces fail silently — no compile-time error |
| B4 | PD-MH-REF6 (Δ6) | module-hardening-reference-cards-deltas | `types/logic.md` §RT-control-logic; `docs/module-best-practices.md` MBP1 | `BRelTime.encodeToString()` emits raw milliseconds as a decimal string, NOT ISO-8601; external clients (oBIX/REST/JSON) silently receive an uninterpretable value unless explicitly converted |
| B5 | apillm Δ10 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §json; JSON helper | Serialize an ABSENT facet as JSON `null` (or omit the key), never the literal string `"null"`; observed live: `"unit":"null"` when `getFacets().gets(UNITS,"")` fed a null through `Json.q` — corrupts API consumers that test `!= null` |

#### Failure-modes deltas (retro COMPLETE — from B1113–B1132)

The `2026-09-20-module-hardening-failure-modes-deltas.md` retro is COMPLETE and committed
(16 deltas PD-MH-RUN1/2/3/4/6/7 · PER2/3/5/6 · BLD1–6 · UXS6). Its deltas map to the
tiers below — see that retro for each Δ's full target file/§ and evidence [ev: corpus B1113–B1132]:

| Placeholder category | Block range | Tier | Notes |
|---|---|---|---|
| SECURITY — `RUN6` post()-drops-Context (no RBAC) | B1129 | **A** | `invoke()` enforces checkInvoke; `post()` drops Context entirely — any action dispatched via `post()` runs with no security context |
| SILENT-CORRUPTION — `PER2`/`PER3`/`PER5`/`RUN2` | B1114–B1132 | **B** | Stale-slot / silently-wrong persistence and scheduling failure modes (BConversionLink stale converter after source retype confirmed at B1131) |
| RESOURCE-LEAK — `RUN4`/`RUN7` | B1130 (RUN7 ForkJoinPool) | **B/C** | ForkJoinPool sizing, `submit`=`invoke`, `RejectedExecutionException` callback-death; resource-leak patterns |
| BUILD-TRIAGE — `BLD1`–`BLD6` | B1113–B1132 | **C** | Build-time triage patterns; actionable-correctness, no runtime data loss |
| LIFECYCLE — `RUN1`/`RUN3` | B1113–B1132 | **B/C** | Lifecycle seam issues; rank alongside B2 (RUN1 timer) once full text is available |

---

### Tier C — ACTIONABLE-CORRECTNESS (43 items)

Ordered roughly by risk (build/compile breaks first, behavioral correctness second, lint/rule additions last).

| Rank | Delta (PD token) | Source retro | Target kit file / § | Why this tier |
|---|---|---|---|---|
| C1 | apillm Δ1 | apillm-headless-servlet-rt-4.14-deltas | `fixtures/MinimalDash/…/B*Servlet.java` | Fix MinimalDash fixture: override `doGet(WebOp)` not `doService(WebOp)` (private in 4.14); derive `getServletName()` from module name, not hardcoded `"minimaldash"` — **build break** |
| C2 | apillm Δ2 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §servlet; scaffold rt gradle | `BWebServlet` in `-rt` profile needs `compileOnly("javax.servlet:javax.servlet-api:3.1.0")` — `-ux` gets it transitively but `-rt` does not; **build break** |
| C3 | apillm Δ3 | apillm-headless-servlet-rt-4.14-deltas | `fixtures/*/-wb` gradle; `types/wb-widgets.md` | `-wb` gradle must include `com.tridium.convention.niagara-home-repositories`; omitting it causes unresolved `:baja`/`:web` — **build break** |
| C4 | apillm Δ16 | apillm-headless-servlet-rt-4.14-deltas | `scaffold-module.sh`; `toolbelt/lint-structure.sh` | `gradle.properties` must carry the JDK-pinning block; scaffold should emit it; `lint-structure` FAIL on every build until fixed |
| C5 | apillm Δ4 | apillm-headless-servlet-rt-4.14-deltas | `scaffold-module.sh`; `fixtures/…/module.palette` | Scaffold palette root should emit `t="b:UnrestrictedFolder"` — kills standing `palette-root` WARN (B746 §746.1) |
| C6 | apillm Δ17 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | `-wb` manager view (`BWbComponentView`) must REFRESH ON ACTIVATION and/or subscribe to its component (Subscriber / topic-fire); `doLoadValue()`-only leaves table stale after station restart or external change |
| C7 | apillm Δ8 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` / UX rule | `-wb` station point/folder picker must be scoped to the STATION COMPONENT SPACE (`slot:`), not `BWbFieldEditor.dialog(this,title,BOrd.NULL)` which defaults to `file:` (browses C:/) |
| C8 | apillm Δ9 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §servlet-data | Read-only export servlet must DEDUP by resolved handle/ord — overlapping exports double-emit; observed live: 22 rows for 11 unique points |
| C9 | apillm Δ18 | apillm-headless-servlet-rt-4.14-deltas | `types/logic-authoring.md`; `types/observability.md` | Gate outbound poll on `enabled` AND non-empty `url`; when blank, skip and log `lastPollStatus=unconfigured` instead of `poll error: null` |
| C10 | apillm Δ15 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | After mutating a table model in place, `table.setModel(sameInstance)` is a NO-OP in some BJ builds; call `table.relayout()` to force repaint |
| C11 | PD-18 | wb-vendor-ux-rt-wb-pattern-deltas | `types/cloud-connector.md`; `types/issues-and-gotchas.md` | **GOTCHA**: `httpClient-rt BHttpClientService.enableNonDriverClients=false` blocks non-driver module outbound HTTP by default; a module using `BHttpClient` must opt in; raw `HttpURLConnection`/OkHttp bypass the gate |
| C12 | PD-06 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Every `BOrd` ref slot needs a station-scoped WB picker (never the file-space default) — vendor-grade UX rule |
| C13 | apillm Δ5 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | wb-agent registration contract: `<agent>` in `module-include.xml` AND inline `@AgentOn` in `@NiagaraType`; inline-only does not emit the agent |
| C14 | apillm Δ6 | apillm-headless-servlet-rt-4.14-deltas | `types/logic.md` §timers; lint-delays doc | `Clock.schedule*` delay floor must be INLINE (`Math.max(ms,floor)` in the call), not via a reassigned variable — `lint-delays` traces only inline floors and FAILs a variable one |
| C15 | PD-FE1 (apillm Δ20) | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md`; `types/value-types.md` §facets | Add `@BFacets(values={"targetType=baja:Component"})` to a `BOrd` `@NiagaraProperty` → `BOrdFE` selects `BComponentChooser` (station tree) even for null ord; zero `-wb` code needed |
| C16 | PD-FE1 | wb-field-editors-deltas | `docs/how-to-create-an-n4-module.md` §-wb; `types/wb-widgets.md` | Document the `targetType` facet pattern in the how-to guide (same mechanism as C15, how-to angle) |
| C17 | PD-FE2 (apillm Δ21) | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | `targetType=baja:ControlPoint` for point-picker slots — constrains `BOrdFE` chooser to ControlPoint subtypes |
| C18 | PD-FE2 | wb-field-editors-deltas | `types/wb-widgets.md` | Document point-creation-from-WB flow for importer-style managers: `getNewTypes()` + `MgrController.promptForNew()` → `MgrEdit.commit()` → `Mark.moveTo(container)` — do NOT call `BComponent.add()` manually |
| C19 | apillm Δ23 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | Rule: one-step "create referenced component" in a manager — type dropdown + name + folder → `new B*Writable()` + `folder.add()` + wire ref ord; never force pre-create + hand-pick ORDs |
| C20 | apillm Δ24 | apillm-headless-servlet-rt-4.14-deltas | `types/structure.md`; `BUILD-LOOP.md` §slotomatic | Slotomatic checksum: when hand-adding `@NiagaraProperty` set auto-region checksum to `0` (integer), NOT `(auto)` placeholder — `(auto)` triggers a parse error (build passes but noisy) |
| C21 | PD-01 | wb-vendor-ux-rt-wb-pattern-deltas | `types/driver-authoring.md`; `types/wb-widgets.md` | WB-presence rule: every `BBasicNetwork` ships a `@AgentOn` WB device manager |
| C22 | PD-02 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | One `MgrColumn.Prop` per key `@NiagaraProperty` on the proxy ext (column = proxy slot) |
| C23 | PD-03 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | `discovery/toRow()` result objects must carry ALL config slots so the add-dialog is pre-populated |
| C24 | PD-04 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md`; `types/logic-authoring.md` | Every long RT op is a `BOrd` action on a job-bar (`submit→sync→resolve→registerForEvents→jobBar.load`) |
| C25 | PD-05 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Prune inapplicable inherited commands in `makeCommands()`/`getAgents()` |
| C26 | PD-07 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Dialogs accept live RT objects for pre-population, not a re-fetch |
| C27 | PD-09 | wb-vendor-ux-rt-wb-pattern-deltas | `types/logic-authoring.md`; `types/observability.md` | Live RT state changes fire typed `@NiagaraTopic` payloads the WB subscribes to (topic-fire) |
| C28 | PD-15 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Facet-driven field-editor selection — pick the FE from the slot's facets (e.g. `TimeZoneSelectionFE`) instead of a fixed editor |
| C29 | PD-16 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Manager column template hook: `addDefault*Columns()` + `addCustom*Columns()` so subclasses extend the column set cleanly |
| C30 | PD-17 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | `isDefault` blank-suppression in polymorphic manager columns (hide cells that don't apply to a row's type) |
| C31 | PD-19 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Unified multi-part WB view recipe: `BTabbedPane`+`BLabelPane` (Baja native, NOT `JTabbedPane`), one `createTPage*()` per tab, `BJobBar`(NORTH)+`BTable`(CENTER) per status tab, `doLoad`/`doSave` symmetry |
| C32 | PD-20 | wb-vendor-ux-rt-wb-pattern-deltas | `types/driver-authoring.md` | Override `getDeviceManagerSubscribeDepth()` in `BNNetwork` subclass to return >1 for deep slot trees (default hardcoded 1) |
| C33 | PD-24 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | **GOTCHA**: `BTabbedPane.selection` is TRANSIENT — persist the selected tab by label TEXT across `doLoadValue`/`doSaveValue` |
| C34 | PD-WMF4 (Δ4) | wb-manager-framework-deltas | `docs/how-to-create-an-n4-module.md` §-wb | Add minimal non-driver custom-manager template (`@AgentOn` a service/container, `MgrColumn.Prop` only, no discover) to how-to guide's `-wb` section |
| C35 | honeywell Δ6 | honeywell-wb-rt-wb-deltas | `types/logic-authoring.md`; `types/observability.md` | `AtomicBoolean` single-run guard on async background jobs triggered from the UI — prevents duplicate submissions |
| C36 | honeywell Δ8 | honeywell-wb-rt-wb-deltas | `types/wb-widgets.md` | `static State extends DeviceState` (class-scoped) to survive manager view close/reopen without losing subscription/discovery state |
| C37 | PD-WV3-snmp (Δ8) | wb-vendor-ux-wave3-vendor-drivers-deltas | `docs/how-to-create-an-n4-module.md` §-wb; `types/wb-widgets.md` | Thin non-driver table-manager CRUD template (~50–70 lines: `@AgentOn` a service, `MgrColumn.Prop` only, no discover) + credential variant (`BPasswordStrength.DEFAULT` in `MgrEdit` validate) |
| C38 | PD-WV3-lint (Δ14) | wb-vendor-ux-wave3-vendor-drivers-deltas | `lint/` new: `precision-facet-learn-mismatch` | Precision-consistency lint: flag when a history record's `facet precision` differs from the proxy point's `exponent` facet (mbus corpus B1100 observed mismatch) |
| C39 | PD-ODA2 (Δ2) | our-dashboard-audit-deltas | `docs/how-to-create-an-n4-module.md` §dashboard; `types/dashboard.md` | `SERVICE_ORD` and room/slot arrays must be runtime-resolvable (BQL introspection or `@NiagaraProperty`); hardcoding requires recompile per site — document portability tradeoff and mitigation |
| C40 | PD-ODA3 (Δ3) | our-dashboard-audit-deltas | `docs/module-best-practices.md` MBP3; `types/dashboard.md` | Strengthen empty-palette rule: a dashboard module's `module.palette` MUST include the primary rt service type and facade container as draggable entries — empty palette forces Config > New Component every deploy (MBP3 violation) |
| C41 | PD-ODA4 (Δ4) | our-dashboard-audit-deltas | `docs/module-best-practices.md` MBP2; `types/dashboard.md` | Any rt class with a pure-String static helper annotated "for WSL unit testing" MUST have a corresponding JUnit test in the rt-test profile — the comment is the author's intent contract; absence detectable by lint sweep of `srcTest` |
| C42 | PD-ODA5 (Δ5) | our-dashboard-audit-deltas | `docs/module-best-practices.md`; `commissioning-verify.sh` | After deploying a `BWebServlet`-based module, probe the live endpoint with `curl` that checks for `X-Content-Type-Options` and `X-Frame-Options`; add to `commissioning-verify.sh` post-deploy checklist |
| C43 | PD-MH-REF4 (Δ4) | module-hardening-reference-cards-deltas | `docs/how-to-create-an-n4-module.md` §slot-authoring | BEnumRange authoring guidance: BNF grammar for inline facets, `make()` overload catalogue, and non-contiguous-ordinal warning — `make(String[])` forces 0..n-1; protocol enums with gaps need `make(int[], String[])` |

---

### Tier D — REFERENCE / DOC-ONLY (36 items)

| Rank | Delta (PD token) | Source retro | Target kit file / § | Why this tier |
|---|---|---|---|---|
| D1 | apillm Δ7 | apillm-headless-servlet-rt-4.14-deltas | `METHODOLOGY.md` (new UX rule); `types/wb-widgets.md`; `types/dashboard.md` | Standing user directive: all user-facing module UI must be intuitive — pickers/choosers + drag-drop + status tables; never a blank-entry "Add" that dumps user into raw property sheet |
| D2 | apillm Δ11 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §json | Apply point's PRECISION facet to numeric values in display JSON; document as a deliberate choice |
| D3 | apillm Δ12 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §data-refresh | Dashboard/gateway live view = REST-poll on ~5s interval (not WebSocket/Fox-subscription unless sub-second latency or high point-count required) — confirmed DashboardPan design pattern |
| D4 | apillm Δ14 | apillm-headless-servlet-rt-4.14-deltas | `types/dashboard.md` §servlet-data | Live-view servlet dual-path recipe: root (`/`) → self-contained HTML polling its own JSON via ES5 XHR (`withCredentials=true`); `/out` → JSON API — no CDN, no `-ux` module |
| D5 | apillm Δ19 | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | **(SUPERSEDED by C15 / PD-FE1)** Custom `BWbFieldEditor` for `BOrd` station-component picker — the `targetType` facet fix is zero-code and eliminates this; note the supersession |
| D6 | apillm Δ22 (PD-FE3) | apillm-headless-servlet-rt-4.14-deltas | `types/wb-widgets.md` | Filtered station-subtree chooser with custom `BComponentChooser` subclass + `selectFilter` lambda + `@AgentOn` on a custom ord type — reference recipe for advanced use |
| D7 | honeywell Δ1 | honeywell-wb-rt-wb-deltas | `types/driver-authoring.md`; `types/wb-widgets.md` | SPI-extensible multi-device-type manager via `BIHonDeviceModel` + `NiagaraRegistryUtil.getImplementersOfTypeSpec()` — one manager base, many device types register |
| D8 | honeywell Δ2 | honeywell-wb-rt-wb-deltas | `types/wb-widgets.md` | Persist WB user preferences as per-user JSON under `userHome/` (no rt slots needed) |
| D9 | honeywell Δ4 | honeywell-wb-rt-wb-deltas | `types/wb-widgets.md` | Multi-level user authorization on UI bindings via integer `visibilityPin`/`actionPin` slots (default -1) |
| D10 | honeywell Δ5 | honeywell-wb-rt-wb-deltas | `types/logic-authoring.md`; `types/distribution.md` | Chunked Base64 transfer (~5000-byte segments) for firmware/file OTA when rt actions accept only small strings |
| D11 | PD-WMF1 (Δ1) | wb-manager-framework-deltas | `docs/module-best-practices.md` MBP3; `types/wb-widgets.md` | `BAbstractManager` 6-object anatomy + 3 boolean gates (`isLearnable`/`isTaggable`/`isTemplatable`) reference section |
| D12 | PD-WMF2 (Δ2) | wb-manager-framework-deltas | `docs/module-best-practices.md` MBP3; `types/wb-widgets.md` | `MgrColumn` taxonomy reference card: Name/Type/Prop/PropPath — when to use each |
| D13 | PD-WMF3 (Δ3) | wb-manager-framework-deltas | `docs/module-best-practices.md` MBP3 | `BAbstractManager` vs `BWbComponentView` decision table |
| D14 | PD-08 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Use `BWbComponentView` (not a manager) for non-table specialized UIs (terminals, schedule editors) |
| D15 | PD-10 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Cross-cutting WB concerns use a `BWbService`, not per-component agents |
| D16 | PD-11 | wb-vendor-ux-rt-wb-pattern-deltas | `METHODOLOGY.md` decision tree | Add WB archetypes (pure-resource palette / driver-managers / specialized-view / rt-only negative) to the module decision tree |
| D17 | PD-12 | wb-vendor-ux-rt-wb-pattern-deltas | `types/dashboard.md` | When a module ships `-ux`, document the RT↔UX↔WB triangle (same rt data, three surfaces) |
| D18 | PD-13 | wb-vendor-ux-rt-wb-pattern-deltas | `types/driver-authoring.md` | Driver modules with NO Hx/UX intent must state that explicitly (rely on base Niagara hx-wb) |
| D19 | PD-14 | wb-vendor-ux-rt-wb-pattern-deltas | `types/driver-authoring.md` | `NMgrControllerUtil.network.getAgents().filter()` is the NATIVE device-manager-agent extension point (`BINDeviceMgrAgent` recipe) |
| D20 | PD-21 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | COV/capability-bit `toRow()` pre-population — decode a `servicesSupported`/capability bitfield into add-row pre-filled fields |
| D21 | PD-22 | wb-vendor-ux-rt-wb-pattern-deltas | `types/wb-widgets.md` | Boolean-flag column-switch template: two column arrays, dispatch in `makeColumns()` on an `isX()` hook |
| D22 | PD-25 | wb-vendor-ux-rt-wb-pattern-deltas | `types/cloud-connector.md` | Cloud integration entry point: `BICloudConnector` (CompletableFuture API), `BConnectorImpl`, `nCloudDriver` bridges cloud service to device/point tree |
| D23 | PD-WV3-obix (Δ1) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md`; `docs/module-best-practices.md` MBP3 | Driver-vs-servlet decision note + local-export-discover idiom (`slot:/` resolution + `BIWritablePoint` filter) |
| D24 | PD-WV3-lon (Δ2) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md` | NV-binding UX is NOT a point list: XIF/LNML typing, changeable-NV discovery, service-pin commissioning each require a distinct manager pattern |
| D25 | PD-WV3-honbacnet (Δ3) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md`; `types/wb-widgets.md` | OEM-on-stock-driver patterns: type-float slot, two-anchor manager mount, dual-FE registration, ORD-carrier navigation |
| D26 | PD-WV3-nd (Δ4) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md`; `types/wb-widgets.md` | `BStationMgrCommand` registry-discovered command SPI, session-keyed learn state, offline `.bog` guard, `CredentialsColumn` lease+newCopy, shorthand `HistoryId` |
| D27 | PD-WV3-ace (Δ5) | wb-vendor-ux-wave3-vendor-drivers-deltas | `docs/module-best-practices.md` MBP3; `types/wb-widgets.md` | "Vendor programming environment" WB tier as the ceiling of the rung ladder — mark reference-not-imitation; our modules stay at rung 0–1 |
| D28 | PD-WV3-plat (Δ6) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/wb-widgets.md` | `PlatformServicePlugin` as a named WB view-target: `@AgentOn platform:*Service`, `poll`/`lease`/`savePlatformServiceProperties` contract |
| D29 | PD-WV3-mbus (Δ7) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md` | Dual-addressing discovery wizard, multi-baud scan, manufacturer-specific data model — field-bus protocol driver archetype |
| D30 | PD-WV3-zwave (Δ9) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/wb-widgets.md` | Multi-perspective managers over one object graph, firmware-capability-gated UI, raw-payload byte FE, power-state column |
| D31 | PD-WV3-opc (Δ10) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/driver-authoring.md`; `types/wb-widgets.md` | Action-slot bridge with structured error decode (keep native/COM error out of WB view), lazy hierarchical browse, security-gated state |
| D32 | PD-WV3-beats (Δ11) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/wb-widgets.md` | Host-OS config from WB Platform tab (`BDaemonSessionView` + `DaemonFileUtil` file-push); NRE-version-gated feature tab pattern; platform-daemon-file WB target |
| D33 | PD-WV3-taxonomy (Δ12) | wb-vendor-ux-wave3-vendor-drivers-deltas | `types/wb-widgets.md` §wb-view-targets | WB target taxonomy: three axes (station-component/driver, platform-service-plugin, platform-daemon-file); extends B751 rung ladder with a second axis |
| D34 | PD-FE3 | wb-field-editors-deltas | `types/wb-widgets.md` | Filtered station-subtree chooser: subclass `BComponentChooser`, override constructor with type-checking `selectFilter` lambda, register via `@AgentOn` on a custom ord type — NEVER override global `baja:Ord` agent |
| D35 | PD-MH-REF3 (Δ1) | module-hardening-reference-cards-deltas | `docs/how-to-create-an-n4-module.md` §slot-authoring; `docs/module-best-practices.md` MBP1 | BFacets key reference card: all 31 keys (radix, showSeconds, fieldWidth, showUnits, unitConversion, maxOverrideDuration, targetType, the 5 editor keys, etc.) with string literal, purpose, and factory method |
| D36 | PD-MH-REF1 (Δ3) | module-hardening-reference-cards-deltas | `docs/how-to-create-an-n4-module.md` §-rt; `types/logic.md` | BQL function reference card (5 aggregates + 7 expression scalars + 24 BBqlTime functions) + custom-fn registration how-to: public static method on a `BObject` library class, registered via `moduleSpec::fn()` |

---

## 3. Target-file rollup

Group by kit file so a reviewer can apply all deltas to one file in a single pass.

| Kit file / location | Delta tokens (by rank) |
|---|---|
| `docs/module-best-practices.md` (MBP1/2/3) | A2(PD-ODA1), A7(PD-MH-REF5), B1(PD-MH-REF2), B2(PD-MH-RUN1), B3(PD-MH-REF7), B4(PD-MH-REF6), C34(PD-WMF4), C40(PD-ODA3), C41(PD-ODA4), C42(PD-ODA5), D11(PD-WMF1), D12(PD-WMF2), D13(PD-WMF3), D23(PD-WV3-obix), D27(PD-WV3-ace), D35(PD-MH-REF3) |
| `docs/how-to-create-an-n4-module.md` (§-wb, §dashboard, §slot-authoring, §-rt) | B3(PD-MH-REF7), C16(PD-FE1), C18(PD-FE2), C34(PD-WMF4), C37(PD-WV3-snmp), C39(PD-ODA2), C43(PD-MH-REF4), D36(PD-MH-REF1), D35(PD-MH-REF3) |
| `types/wb-widgets.md` | A4(honeywell Δ3), C7(apillm Δ8), C10(apillm Δ15), C12(PD-06), C13(apillm Δ5), C15(PD-FE1/apillm Δ20), C17(PD-FE2/apillm Δ21), C19(apillm Δ23), C21(PD-01), C22(PD-02), C23(PD-03), C24(PD-04), C25(PD-05), C26(PD-07), C27(PD-09), C28(PD-15), C29(PD-16), C30(PD-17), C31(PD-19), C33(PD-24), C35(honeywell Δ6), C36(honeywell Δ8), D6(PD-FE3), D7, D8, D9, D14(PD-08), D15(PD-10), D20(PD-21), D21(PD-22), D25(PD-WV3-honbacnet), D26(PD-WV3-nd), D28(PD-WV3-plat), D30(PD-WV3-zwave), D32(PD-WV3-beats), D33(PD-WV3-taxonomy), D34(PD-FE3) |
| `types/security.md` | A1(apillm Δ13), A2(PD-ODA1), A3(PD-23), A4(honeywell Δ3), A5(honeywell Δ7), A6(PD-WV3-security) |
| `types/logic.md` (§RT-control-logic, §timers) | A7(PD-MH-REF5), B1(PD-MH-REF2), B2(PD-MH-RUN1), B4(PD-MH-REF6), C14(apillm Δ6), D36(PD-MH-REF1) |
| `types/logic-authoring.md` | C9(apillm Δ18), C24(PD-04), C27(PD-09), C35(honeywell Δ6), D10(honeywell Δ5) |
| `types/driver-authoring.md` | C20(PD-20), C21(PD-01), C32(PD-20), D18(PD-13), D19(PD-14), D23(PD-WV3-obix), D24(PD-WV3-lon), D25(PD-WV3-honbacnet), D26(PD-WV3-nd), D29(PD-WV3-mbus), D31(PD-WV3-opc) |
| `types/dashboard.md` | B5(apillm Δ10), C8(apillm Δ9), C39(PD-ODA2), C40(PD-ODA3), C41(PD-ODA4), D2(apillm Δ11), D3(apillm Δ12), D4(apillm Δ14), D17(PD-12) |
| `types/observability.md` | A5(honeywell Δ7), C9(apillm Δ18), C27(PD-09), C35(honeywell Δ6) |
| `types/cloud-connector.md` | C11(PD-18), D22(PD-25) |
| `types/issues-and-gotchas.md` | C11(PD-18) |
| `types/distribution.md` | D10(honeywell Δ5) |
| `types/value-types.md` §facets | C15(PD-FE1/apillm Δ20) |
| `types/structure.md` | C20(apillm Δ24) |
| `METHODOLOGY.md` | D1(apillm Δ7), D11(PD-WMF1 via MBP3), D16(PD-11) |
| `toolbelt/lint-servlet.sh` | A1(apillm Δ13), A2(PD-ODA1) |
| `toolbelt/lint-structure.sh` | C4(apillm Δ16) |
| `commissioning-verify.sh` | C42(PD-ODA5) |
| `scaffold-module.sh` / `fixtures/MinimalDash/` | C1(apillm Δ1), C2(apillm Δ2), C3(apillm Δ3), C5(apillm Δ4) |
| `fixtures/*/-wb` gradle | C3(apillm Δ3) |
| `BUILD-LOOP.md` §slotomatic | C20(apillm Δ24) |
| `lint/` (new lints) | A6(no-md5-credential-digest), A7(no-bformat-untrusted-input), B2(timer-rearm-not-in-finally), C38(precision-facet-learn-mismatch) |

---

## 4. Notes

- **This document is propose-never-apply.** Nothing in this file edits, modifies, or
  creates any kit file. It is a ranked reading list for a human reviewer.
- **Failure-modes retro COMPLETE + committed.** The
  `2026-09-20-module-hardening-failure-modes-deltas.md` retro (16 deltas) is finished and
  in the INDEX; its deltas are categorized by tier in the failure-modes subsection above
  (SECURITY/RUN6 → A · SILENT-CORRUPTION/PER2,PER3,PER5,RUN2 → B · RESOURCE-LEAK/RUN4,RUN7 → B/C ·
  BUILD-TRIAGE/BLD1–6 → C · LIFECYCLE/RUN1,RUN3 → B/C). Per-Δ target files are in that retro.
- **INDEX.md was read only.** This file does not add a row to `retros/INDEX.md`; it is
  a standalone plan, not a delta retro, and the INDEX sync-gate does not apply to it.
- **Tier-A items (A1 + A2 + C42) form a coherent servlet-header mini-campaign** that
  can be applied atomically: one lint rule, one guide section, one commissioning probe.
- **Tier-B items B1 (isValid) and B2 (timer-rearm) are the highest-priority single-file
  edits** — both touch `types/logic.md` and `docs/module-best-practices.md` MBP1, and
  both have lint candidates that prevent silent failure in future modules.
- **Superseded delta**: apillm Δ19 (D5) is superseded by apillm Δ20 / PD-FE1 (C15).
  Fold it with a forward-reference note, do not implement it.
