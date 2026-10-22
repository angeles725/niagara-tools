<!-- review-status: folded -->
# 2026-10-22 · kit · t9c-reference-gaps

**Session**: niagara-research quality round 8 (`odd/tasks/quality-round8.md`, task T9c), branch
`docs/r8-t9c-refs` from bb85a29; one commit per topic
**Delta count**: 7

## What happened
The devguide coverage audit (niagara-research `docs/n4-devguide-coverage-gaps.md`, DG3-DG10, and the OPC /
data-egress guides) listed Developer Guide areas that the kit never taught: the auto-logoff activity monitor,
driver alarm extensions, the UI changelog and Px media migration, browser-side managers, direct BajaScript
reads, series transforms, and OPC UA plus the data-egress choice. Each topic was verified against the three
sources (corpus blocks, niagara-help, extracted code in `organized/`) and folded as one short section that
links to the corpus instead of copying it. This retro records the set; each Δ is folded by its own commit.

## Evidence
- Gap list and reasons: niagara-research `docs/n4-devguide-coverage-gaps.md` DG3, DG4, DG6, DG7, DG9, DG10 `[ev: corpus B1303]`
- Kit grep before the change: `rg -i 'autoLogoff|AlarmDeviceExt|DeviceMgrModel|uiChangelog|PxMediaMigration|SeriesSchema|opcUa' types/` → 0 hits outside the OPC DA manager stub (`types/driver-authoring.md` § 9.6)
- Citations per section: `[CERT]` file:line under niagara-research `organized/`, `[CERT-doc]` under `niagara-help/`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Auto-logoff model: server-side idle timer, per-user period, only user input counts, the always-on dashboard recipe, 4.15 absolute logoff | `types/dashboard.md` § Session lifetime — the auto-logoff activity monitor | `[ev: corpus B1344]` |
| Δ2 | `BAlarmDeviceExt` recipe: source of incoming alarms, `doRouteAlarm`/`doAckAlarm`, ack round trip, pitfalls | `types/driver-authoring.md` § 7.5 | `[ev: corpus B1226]` |
| Δ3 | UI changelog 4.13 → 4.14 → 4.15 breaking items for `-ux` code, and what `PxMediaMigrator` does | `types/frontend-standard.md` § UI API changes across NRE versions | `[ev: corpus B1315]` |
| Δ4 | Browser-side managers: AMD ids, extend `DeviceMgr`/`PointMgr`, model/column/learn hooks, Java `@AgentOn` registration | `types/driver-authoring.md` § 1.5 | `[ev: corpus B752]` |
| Δ5 | Direct BajaScript reads (ORD, subscriber, BQL cursor, nav, tags/relations) and the BajaScript-vs-servlet decision | `types/dashboard.md` § Reading the station directly with BajaScript | `[ev: corpus B1306]` |
| Δ6 | Series transforms guide: graph, nodes, `BSeriesSchema`, `BTransformFunction`, `transform:` ORD, pitfalls | `types/history-transforms.md` (new) | `[ev: corpus B1339]` |
| Δ7 | OPC UA client/server summary and the data-access/egress decision table | `types/driver-authoring.md` § 10, `types/cloud-connector.md` pointer | `[ev: corpus B1222]` |

Each Δ was folded by its own commit on this branch; the last commit flipped this retro and its INDEX row.

## Lessons
- A coverage audit against the official devguide finds whole subsystems the kit never named; link the
  derivation (B<n>) and keep only the recipe, the decision and the pitfalls in the kit.
- Our own blocks can be wrong about version scope (B1315 called `ui/uxMediaMigration` 4.15-only; the 4.13.2.18
  docs ship it); re-check version claims against the versioned doc trees before folding them.
- Release notes in the devguide mix AX and N4 history; check the "Resolved In" build before using one as an
  N4 version fact.

---
**Status**: FOLDED 2026-10-22 — Δ1-Δ7 each folded by its own commit on `docs/r8-t9c-refs`; INDEX row flipped to folded.
