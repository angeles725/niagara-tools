<!-- review-status: folded -->
# 2026-09-20 · kit · wb-vendor-ux-rt-wb-pattern-deltas

**Session**: niagara-research focus `wb-vendor-ux` (B1054–B1061) — cross-vendor survey of how Honeywell/Tridium/Distech/OEM build their `-wb`, and the RT↔WB correlation. WV8 synthesis (B1061) emitted 13 proposed deltas (PD-01..PD-13) for this kit so our own modules match vendor-grade WB UX.

**Delta count**: 25

## What happened
Surveyed 8 vendor `-wb` archetypes (Centraline pure-resource palette, andoverAC256 driver managers, axvelocity Velocity templates, ccn 9-agent driver, andoverInfinity VT100 terminal, lonSchneider/lonDistech negative, clCBus nDriver). Extracted 10 repeatable cross-vendor UX patterns and the RT↔WB coupling rules. These become proposed authoring rules so a module's `-wb` feels vendor-grade instead of a bare property sheet. Source blocks carry the [CERT] evidence; this retro proposes, it does not edit the kit.

## Evidence
- B1057 ccn-wb `getAgents()` removes PropertySheet, promotes DeviceManager as default view; B1055 andover PointManager column→PropPath + `toRow()` pre-population; B1060 clCBus job-bar schedule editor; B1058 andoverInfinity topic-fire VT100; B1059 negative (lonSchneider/lonDistech zero-Java .lnml). `[ev: corpus B1055–B1061]`
- Prior wb mechanism cited, not re-derived: B9/B12/B15§15.8/B705-710/px-editor/CS7. `[ev: corpus B1061]`

## Proposed kit deltas (propose-never-apply) — full text + citations in corpus B1061
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | PD-01 every `BBasicNetwork` ships a `@AgentOn` WB device manager (WB-presence rule) | `types/driver-authoring.md` / `types/wb-widgets.md` | `[ev: corpus B1057]` |
| Δ2 | PD-02 one manager column (`MgrColumn.Prop`) per key `@NiagaraProperty` on the proxy ext (column = proxy slot) | `types/wb-widgets.md` | `[ev: corpus B1055]` |
| Δ3 | PD-03 discovery/`toRow()` result objects carry ALL config slots so the add-dialog is pre-populated | `types/wb-widgets.md` | `[ev: corpus B1055]` |
| Δ4 | PD-04 every long RT op is a `BOrd` action on a job-bar (`submit→sync→resolve→registerForEvents→jobBar.load`) | `types/wb-widgets.md` / `types/logic-authoring.md` | `[ev: corpus B1060]` |
| Δ5 | PD-05 prune inapplicable inherited commands in `makeCommands()`/`getAgents()` | `types/wb-widgets.md` | `[ev: corpus B1057]` |
| Δ6 | PD-06 every `BOrd` ref slot needs a station-scoped WB picker (never the file-space default) | `types/wb-widgets.md` | `[ev: corpus B1061]` |
| Δ7 | PD-07 dialogs accept live RT objects, not a re-fetch (dialog pre-population) | `types/wb-widgets.md` | `[ev: corpus B1055]` |
| Δ8 | PD-08 use `BWbComponentView` (not a manager) for non-table specialized UIs (terminals, schedule editors) | `types/wb-widgets.md` | `[ev: corpus B1058]` |
| Δ9 | PD-09 live RT state changes fire typed `@NiagaraTopic` payloads the WB subscribes to (topic-fire) | `types/logic-authoring.md` / `types/observability.md` | `[ev: corpus B1058]` |
| Δ10 | PD-10 cross-cutting WB concerns use a `BWbService`, not per-component agents | `types/wb-widgets.md` | `[ev: corpus B1060]` |
| Δ11 | PD-11 add the WB archetypes (pure-resource palette / driver-managers / specialized-view / rt-only negative) to the module decision tree | `METHODOLOGY.md` decision tree | `[ev: corpus B1054-B1061]` |
| Δ12 | PD-12 when a module ships `-ux`, document the RT↔UX↔WB triangle (same rt data, three surfaces) | `types/dashboard.md` | `[ev: corpus B1061]` |
| Δ13 | PD-13 driver modules with NO Hx/UX intent state that explicitly (rely on base Niagara hx-wb) | `types/driver-authoring.md` | `[ev: corpus B1061]` |
| Δ14 | PD-14 nDriver `NMgrControllerUtil.network.getAgents().filter()` is the NATIVE device-manager-agent extension point (BINDeviceMgrAgent recipe), not CCN-specific | `types/driver-authoring.md` | `[ev: corpus B1066]` |
| Δ15 | PD-15 facet-driven field-editor selection — pick the FE from the slot's facets (e.g. TimeZoneSelectionFE) instead of a fixed editor | `types/wb-widgets.md` | `[ev: corpus B1067]` |
| Δ16 | PD-16 manager column template hook: `addDefault*Columns()` + `addCustom*Columns()` so subclasses extend the column set cleanly | `types/wb-widgets.md` | `[ev: corpus B1068]` |
| Δ17 | PD-17 `isDefault` blank-suppression in polymorphic manager columns (hide cells that don't apply to a row's type) | `types/wb-widgets.md` | `[ev: corpus B1068]` |
| Δ18 | PD-18 GOTCHA: `httpClient-rt BHttpClientService.enableNonDriverClients=false` blocks non-driver modules from outbound HTTP by default — a module using `BHttpClient` must opt in; raw `HttpURLConnection`/OkHttp bypass the gate (our importer uses HttpURLConnection, so it is NOT gated) | `types/cloud-connector.md` / `types/issues-and-gotchas.md` | `[ev: corpus B1069]` |
| Δ19 | PD-19 unified multi-part WB view recipe: `BTabbedPane`+`BLabelPane` (Baja native tabs, NOT JTabbedPane), one `createTPage*()` per tab, conditional tab when the child is absent, doLoad/doSave symmetry, `BJobBar`(NORTH)+`BTable`(CENTER) per status tab | `types/wb-widgets.md` | `[ev: corpus B1070]` |
| Δ20 | PD-20 override `getDeviceManagerSubscribeDepth()` in a `BNNetwork` subclass to return >1 for deep slot trees (default is hardcoded 1) | `types/driver-authoring.md` | `[ev: corpus B1072]` |
| Δ21 | PD-21 COV/capability-bit `toRow()` pre-population — decode a `servicesSupported`/capability bitfield into the add-row's pre-filled fields | `types/wb-widgets.md` | `[ev: corpus B1073]` |
| Δ22 | PD-22 boolean-flag column-switch template: hold two column arrays, dispatch in `makeColumns()` on an `isX()` hook (e.g. client vs slave point manager) | `types/wb-widgets.md` | `[ev: corpus B1074]` |
| Δ23 | PD-23 outbound-vs-inbound security are separate: the `enableNonDriverClients` gate governs OUTBOUND HTTP; `BAuthenticationScheme` SPI governs INBOUND auth — document both, don't conflate | `types/security.md` | `[ev: corpus B1075]` |
| Δ24 | PD-24 `BTabbedPane.selection` is TRANSIENT (no @NiagaraProperty) — persist the selected tab by its label TEXT across `doLoadValue`/`doSaveValue` | `types/wb-widgets.md` | `[ev: corpus B1076]` |
| Δ25 | PD-25 cloud integration entry point: `BICloudConnector` (CompletableFuture API), `BConnectorImpl` (doConnect/registerDevice/doDisconnect/doPing/sendMessage), `nCloudDriver` bridges the cloud service to a standard device/point tree | `types/cloud-connector.md` | `[ev: corpus B1076]` |

## Lessons
- The vendor-grade `-wb` bar is concrete: manager-with-table over property sheet, a picker/field-editor for EVERY rt ref slot, pre-populated dialogs from live state, job-bar for long ops, command pruning, topic-fire for live state. That IS the intuitive-UI rule made specific — pairs with the Apillm retro Δ7/Δ8.
- Strongest RT→WB coupling: `BCcnNetwork.getAgents()` removes the PropertySheet and promotes its DeviceManager (B1057) — fold as a recommendation only after cross-validating vs vendor-specific (B1061-G2).
- Separate, NOT for this kit: B1061 §1061.6 drafts a research-sdd METHODOLOGY §22 "correlation-first exploration" doctrine (four-lens survey + PATTERN/EXCEPTION/DEPENDENCY/SIMILARITY taxonomy) — UNCONFIRMED, human review only, per "todavía no hay que asegurarlo".

---
**Status**: PENDING — 13 proposed deltas; full text + source citations in corpus B1061.
