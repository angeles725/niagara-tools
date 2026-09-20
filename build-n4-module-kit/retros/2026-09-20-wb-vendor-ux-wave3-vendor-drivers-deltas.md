<!-- review-status: pending -->
# 2026-09-20 · kit · wb-vendor-ux-wave3-vendor-drivers-deltas

**Session**: niagara-research focus `wb-vendor-ux` wave-3 (B1094–B1105) — survey of 11 vendor/platform `-wb` modules not covered by wave-1/2 (obixDriver, lonworks, honBACnetUtilities, niagaraDriver, ace, platPower, mbus, nSnmp, zwave, opc, honAdvWirelessCfg) plus the B1105 synthesis that emitted the consolidated PD-WV3-* table and the WB target taxonomy. Commits 6da8005..e9bc8b2 on main.

**Delta count**: 14

## What happened
Wave-3 surveyed 11 vendor/OEM/platform `-wb` archetypes, extending the cross-vendor catalog that waves 1–2 and the honeywell-wb retro started. B1105 synthesised the findings into a PD-WV3-* table, extended the WB rung ladder (B751) with a second "target" axis (station-component vs platform-service-plugin vs platform-daemon-file), and flagged two genuinely actionable deltas: a stronger-than-MD5 security recommendation and a precision-consistency lint for import-learn flows. The remaining 11 PD-WV3-* entries are vendor-WB reference patterns — our own modules stay at rung 0–1. This retro proposes; it does not edit the kit.

## Evidence
- B1094 obixDriver: driver-vs-servlet decision, `slot:/` resolution + `BIWritablePoint` filter for local-export-discover. `[ev: corpus B1094]`
- B1095 lonworks: XIF/LNML typing, changeable-NV discovery, service-pin commissioning — NV-binding UX is not a point list. `[ev: corpus B1095]`
- B1096 honBACnetUtilities: OEM-on-stock-driver patterns — type-float, two-anchor manager mount, dual-FE registration, ORD-carrier navigation. `[ev: corpus B1096]`
- B1097 niagaraDriver: `BStationMgrCommand` registry-discovered command SPI, session-keyed learn state, offline `.bog` guard, `CredentialsColumn` lease+newCopy, shorthand `HistoryId`. `[ev: corpus B1097]`
- B1098 ace: vendor programming environment WB tier — own wire sheet + app wizard + catalog palette + opcode expression editor; ceiling of the WB rung ladder. `[ev: corpus B1098]`
- B1099 platPower: `PlatformServicePlugin` SPI as a distinct WB view target (`platform:*Service` agent, `poll`/`lease`/`savePlatformServiceProperties`). `[ev: corpus B1099]`
- B1100 mbus: dual-addressing discovery wizard, multi-baud scan, manufacturer-specific data model; precision-4 history vs point `-exponent` mismatch as a hardcoded-facet lint candidate. `[ev: corpus B1100]`
- B1101 nSnmp: thin table-manager CRUD template (~50–70 lines) + credential variant using `BPasswordStrength.DEFAULT` in inner `MgrEdit` validate. `[ev: corpus B1101]`
- B1102 zwave: multi-perspective managers over one object graph, firmware-capability-gated UI, raw-payload byte FE, device power-state column. `[ev: corpus B1102]`
- B1103 opc: action-slot bridge with structured COM/native error decode (keep native/COM out of the WB view), lazy hierarchical browse, security-gated state. `[ev: corpus B1103]`
- B1104 honAdvWirelessCfg: host-OS config from WB Platform tab via `BDaemonSessionView` + `DaemonFileUtil` file-push; NRE-version-gated feature tabs; MD5 used for credential storage digest. `[ev: corpus B1104]`
- B1105 synthesis: WB target taxonomy (station-component / platform-service-plugin / platform-daemon-file), PD-WV3-* consolidated table, extended negative finding. `[ev: corpus B1105]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | PD-WV3-obix — add driver-vs-servlet decision note + local-export-discover idiom (`slot:/` resolution + `BIWritablePoint` filter to push local data outward) as a reference pattern in the driver-authoring guide | `types/driver-authoring.md` / `docs/module-best-practices.md` MBP3 | `[ev: corpus B1094]` |
| Δ2 | PD-WV3-lon — document that NV-binding UX is NOT a point list: XIF/LNML typing, changeable-NV discovery, and service-pin commissioning each require a distinct manager pattern | `types/driver-authoring.md` | `[ev: corpus B1095]` |
| Δ3 | PD-WV3-honbacnet — document OEM-on-stock-driver patterns: type-float slot, two-anchor manager mount, dual-FE registration, and ORD-carrier navigation as a reference section | `types/driver-authoring.md` / `types/wb-widgets.md` | `[ev: corpus B1096]` |
| Δ4 | PD-WV3-nd — document the `BStationMgrCommand` registry-discovered command SPI, session-keyed learn state, offline `.bog` guard, `CredentialsColumn` lease+newCopy, and shorthand `HistoryId` as a reference pattern for command-driven managers | `types/driver-authoring.md` / `types/wb-widgets.md` | `[ev: corpus B1097]` |
| Δ5 | PD-WV3-ace — add a note naming the "vendor programming environment" WB tier (own wire sheet + app wizard + catalog palette + opcode expression editor) as the ceiling of the rung ladder; mark it reference-not-imitation — our modules stay at rung 0–1 | `docs/module-best-practices.md` MBP3 / `types/wb-widgets.md` | `[ev: corpus B1098]` |
| Δ6 | PD-WV3-plat — add `PlatformServicePlugin` as a named WB view-target in the WB targets catalog: `@AgentOn` a `platform:*Service`, `poll`/`lease`/`savePlatformServiceProperties` contract; distinct from station-component agents | `types/wb-widgets.md` | `[ev: corpus B1099]` |
| Δ7 | PD-WV3-mbus — document the dual-addressing discovery wizard, multi-baud scan, and manufacturer-specific data model as a reference archetype for field-bus protocol drivers | `types/driver-authoring.md` | `[ev: corpus B1100]` |
| Δ8 | PD-WV3-snmp — add a thin non-driver table-manager CRUD template (~50–70 lines: `@AgentOn` a service, `MgrColumn.Prop` only, no discover) + credential variant (`BPasswordStrength.DEFAULT` in `MgrEdit` validate) to the how-to guide's `-wb` section | `docs/how-to-create-an-n4-module.md` §-wb / `types/wb-widgets.md` | `[ev: corpus B1101]` |
| Δ9 | PD-WV3-zwave — document multi-perspective managers over one object graph, firmware-capability-gated UI (show/hide columns per device capability), raw-payload byte FE, and power-state column as a reference pattern | `types/wb-widgets.md` | `[ev: corpus B1102]` |
| Δ10 | PD-WV3-opc — document the action-slot bridge pattern with structured error decode (keep native/COM error out of the WB view layer), lazy hierarchical browse, and security-gated state as a reference for protocol-adapter managers | `types/driver-authoring.md` / `types/wb-widgets.md` | `[ev: corpus B1103]` |
| Δ11 | PD-WV3-beats — document host-OS config from the WB Platform tab (`BDaemonSessionView` + `DaemonFileUtil` file-push) and NRE-version-gated feature tab pattern as a named WB target variant (platform-daemon-file); add to WB targets catalog | `types/wb-widgets.md` | `[ev: corpus B1104]` |
| Δ12 | PD-WV3-taxonomy (cross-cutting) — add the WB target taxonomy (three axes: station-component/driver, platform-service-plugin, platform-daemon-file) to the WB view-targets catalog; extends the B751 rung ladder with a second axis | `types/wb-widgets.md` §wb-view-targets | `[ev: corpus B1105]` |
| Δ13 | SECURITY (actionable) — add to security best-practices: when storing credentials inside a Niagara module, prefer SHA-256 or stronger over MD5; flag MD5 in credential storage as a security lint candidate | `types/security.md` / `lint/` (new lint: `no-md5-credential-digest`) | `[ev: corpus B1104]` |
| Δ14 | LINT (actionable) — add a precision-consistency lint for import-learn flows: flag when a history record's `facet precision` differs from the proxy point's `exponent` facet (B1100 mbus: precision-4 history vs point `-exponent` mismatch) | `lint/` (new lint: `precision-facet-learn-mismatch`) | `[ev: corpus B1100]` |

## Lessons
- Wave-3 confirms the vendor-WB bar is entirely about platform extensions and driver-specific archetypes; our own modules stay at rung 0–1 (palette + FieldEditor). All PD-WV3-* except Δ13 and Δ14 are reference-only.
- Two deltas ARE genuinely actionable for our codebase: the MD5 → stronger-digest security recommendation (Δ13) and the import-learn precision-consistency lint (Δ14). These should be prioritised in the next lint campaign.
- The WB "target" taxonomy (Δ12) is new conceptual vocabulary that closes a gap in the kit's WB catalog — it costs nothing to document and prevents future confusion when any Honeywell module needs Platform-tab integration.
- The ace "vendor programming environment" tier (Δ5) serves as a concrete upper bound in code-review or scope discussions; knowing the ceiling is useful even if we never build to it.

---
**Status**: PENDING — 14 proposed deltas; full text + source citations in corpus B1094–B1105.
