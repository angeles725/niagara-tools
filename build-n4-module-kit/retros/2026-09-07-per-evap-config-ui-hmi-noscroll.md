<!-- review-status: pending -->
# 2026-09-07 · DashboardPan · per-evap-config-ui-hmi-noscroll

**Session**: PANCCADIA PR2 — per-evaporator Configuración; fit the 10" HMI (1280×800, no scroll).
**Delta count**: 1

## What happened
Configuración moved from one setpoint/room to PER-EVAPORATOR: 24 facade slots (`evapMSetpoint`/diffs/coolFault + defrost, M=1..3) + `evapMStartDelay`, and an SPA `evapGroups()` mirroring `freezeGroups` (filter `SENSORS.rol==="evap"`, so no room ever shows an evaporator it lacks). Defrost fields are shown BY TYPE (air evaps: interval+duration; resistance evaps: + terminate/umbral). Cuarto 3 (resistance, 9 fields) OVERFLOWED the fixed 800px height without scroll — split each resistance evaporator into "Evap M" (general) + "Deshielo M" sub-tabs so each fits.

## Evidence
- overflow observed on-HMI: Cuarto 3 Evap tab cut off "Umbral temp. resistencia" (5 rows > viewport) [ev: operator screenshot 2026-09-07]
- air evap = 7 fields (4 rows) fits; resistance = 9 fields (5 rows) doesn't → defrost split into its own sub-tab (`hasResist` gate) [ev: index.html evapGroups]
- servlet unchanged (generic ord write); reader arrays extended; per rebuild the ux jar packages the current index.html; `2.4.0/2.4.1` [ev: build.sh DashboardPan]

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | HMI-fit budget: a 2-column config sub-tab on the WEB-HMI10 (1280×800, NO scroll) holds ~7 fields (4 rows) max after the page chrome. `rc-scan`/checklist should warn when a single config group exceeds that budget → split into sub-tabs. Record 7-fields/4-rows as the per-tab ceiling. | toolbelt/rc-scan.sh; types/dashboard.md §hmi-fit | [ev: operator screenshot] |

## Lessons
- The WEB-HMI10 is 1280×800, 16:10, capacitive — NO scroll is a hard design constraint; verify layouts in `/hmi` at that size.
- Drive per-evaporator sub-tabs from `SENSORS.rol` (mirror `freezeGroups`) so the UI never invents an evaporator a room lacks.
- Show config fields BY hardware type (air vs resistance defrost) via a data-driven flag (`some(rol==="resist")`), not by hardcoding room numbers.
- Editing index.html requires a `-ux` REBUILD to repackage it into the jar; the preview serves the live file but the deployable jar does not until rebuilt.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-per-evap-config-ui-hmi-noscroll.md | DashboardPan | 2026-09-07 | pending | 1 |`
