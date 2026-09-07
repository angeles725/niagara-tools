<!-- review-status: pending -->
# 2026-09-07 · kit · live-commissioning-verification-gaps

**Session**: PANCCADIA León live commissioning of the per-evaporator rearchitect (ColdRoomPan v2.1.x + DashboardPan v2.4.x) on the JACE-9000, driven over oBIX with the operator (Cristian). All modules had already PASSED every kit gate.
**Delta count**: 7

## What happened

The per-evap rearchitect passed the WHOLE kit gate — build 52 + signed, 64 pure tests, the full lint suite, `verify-module.sh`, `schema-risk.sh` SAFE — and was handed off as "built + verified". At the LIVE station it was **non-functional until ~50 links were hand-wired, and carried defects the gate could not see**. Every real problem surfaced only during live oBIX commissioning, and **the operator had to act as the watchdog** — directing each check ("check the intervals", "check all the configs", "why doesn't the defrost finish") — instead of the kit carrying that vigilance. The kit's verify gate is CODE-level and BLIND to commissioning correctness (facade↔rt links, config values, control/status parity on a running station). Three classes of defect got through: (1) a code recovery-gap bug that stranded a resistance output ON; (2) a design incoherence — per-evaporator control/config but per-ROOM status on the facade (3 defrost controllers, 1 dashboard status); (3) silent misconfigurations the module accepts without any signal (interval ≤ duration, `hasDefrost=false` while `airDefrost=true`, setpoint 0 = won't cool, duration = 1 s).

## Evidence
- **Stranded `inDefrost` → `resistanceOut` stuck ON, no recovery path**: `BEvaporatorUnit.java:1224` (`applyHoaOutputs` `if(inDefrost)return`), `:1125`/`:1174` (applyRunCmd/applyFanRunMode same guard), `:1053-1056` (`stopped()` clears `inDefrost` but deliberately does NOT write `resistanceOut`), `computeEvapCall→applyRunCmd` only (never writes `resistanceOut`). Live: `Programacion/ColdRoom_1/EvaporatorUnit_1` `resistanceOut=true` with BOTH controllers `defrostActive=false`; disable/enable did NOT clear it. `[ev: engram #8443]` `[ev: live oBIX PANCCADIA JACE 2026-09-07]`
- **Control/status parity incoherence**: `BRoomPanel` has per-evap `evapMDefrostInterval`/`Duration` (config ×3) but only ONE room-level `defrostActive`/`defrostStart`/`nextDefrostTime`/`coolingSince`. 3 `BDefrostController`s per Cuarto1, 1 status slot on the facade → the dashboard can only ever show one evaporator's defrost. `[ev: live oBIX Services/DashboardService/Cuarto1 slot dump]`
- **Silent misconfig, no lint**: `enterDefrost()` `BEvaporatorUnit.java:1398` early-returns on `!hasDefrost`, so `hasDefrost=false`+`airDefrost=true` silently disables air defrost; the "air units need hasDefrost=true AND airDefrost=true" rule lives ONLY in a code comment `:182`. Live examples: Cuarto2/Cuarto4 `hasDefrost=false`; Cuarto1-U1 `DefrostController.duration=PT1S`; Cuarto2 `interval=PT6M ≤ duration=PT20M` (defrosts non-stop); every `evapSetpoint` starts 0 (won't cool). `[ev: live oBIX config audit]`
- **Error-prone commissioning, no aid**: Cuarto1 mapping is CROSSED (dashboard `evap1` ↔ physical `EvaporatorUnit_3`); ~50 facade↔rt links to hand-wire across 5 rooms with the crossing, and setpoint/diff links were wired straight-instead-of-crossed on the first pass. No wiring map or verification shipped. `[ev: live oBIX link audit]`
- **No live-verification tool in the kit**: had to build `niagara-research/tools/obix-nav.py` (oBIX batch facade↔rt link + config auditor) on the spot to see any of the above. `[ev: obix-nav.py]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add a **commissioning-verify gate**: an oBIX facade↔rt LINK + CONFIG-VALUE auditor run against the live station before a rewiring-required module is called "done"; fold `obix-nav.py` into the toolbelt as its seed. State that build+unit-verify ≠ commissioned. | `BUILD-LOOP.md` § step 6 (post-reload) + `toolbelt/` | `[ev: obix-nav.py]` |
| Δ2 | New lint `lint-status-parity.sh`: FAIL/WARN when a per-instance control or config surface (N children/slots) has a SCALAR (single) status/telemetry mirror on the facade — N configs but 1 status. | `toolbelt/` + `BUILD-LOOP.md` § step 5 | `[ev: live oBIX Cuarto1 dump]` |
| Δ3 | New lint `lint-config-sanity.sh`: FAIL on `interval ≤ duration`, duration/interval below a floor, OPERATOR cooling-setpoint defaulting to 0, and per-instance flag combos that silently disable a function. | `toolbelt/` + `BUILD-LOOP.md` § step 5 | `[ev: live oBIX config audit]` |
| Δ4 | Code rule: a cross-field invariant (e.g. `hasDefrost`/`airDefrost`) must be ENFORCED in code (derive one from the other, or reject the bad combo) or checked by a lint — never left in a comment only. | `types/logic-authoring.md` | `[ev: BEvaporatorUnit.java:182]` |
| Δ5 | Code rule: any transient mode flag that gates ALL output writers must have a guaranteed recovery path — clear protection/heat outputs on `started()`/enable regardless of the flag; never let the ONLY output-clearing path sit behind the flag. Add `lint-recovery-path`. | `types/logic-authoring.md` + `toolbelt/` | `[ev: engram #8443]` |
| Δ6 | Any per-instance rearchitect must SHIP a commissioning aid: a generated wiring map (facade slot → rt slot, with the physical mapping/crossing) + a checkable per-instance checklist. Operator must not hand-derive the crossing. | `types/*dashboard*` guide + `toolbelt/` (wiring-map generator) | `[ev: live oBIX link audit]` |
| Δ7 | State explicitly that the verify gate is CODE-level and BLIND to live commissioning; the kit — not the operator — owns the watchdog role. Require a commissioning-verify pass (Δ1) for any module needing station rewiring before hand-off. | `BUILD-LOOP.md` § hand-off + `METHODOLOGY.md` | `[ev: obix-nav.py]` |

## Lessons
- **"Built + unit-verified" ≠ "commissioned + correct."** A rearchitect that needs station rewiring is not done until a LIVE facade↔rt link + config audit passes; the kit gate never touched the JACE and every real defect hid past it.
- **Per-instance control demands per-instance STATUS.** Splitting control/config N-ways while leaving status/telemetry 1-way is an incoherence design review must catch — 3 defrosts, 1 dashboard status.
- **A transient mode flag that gates output writers needs a guaranteed recovery path.** Never let the only output-clearing path live behind the flag; clear protection outputs on start/enable unconditionally.
- **Cross-field invariants belong in code or a lint, never only in a comment.** `hasDefrost=false`+`airDefrost=true` disabled defrost with zero signal because the rule was a comment.
- **The kit must be the watchdog, not the operator.** This session only worked because the human directed each check. Config-sanity + commissioning-verify + a shipped wiring map move that vigilance into the kit so the next module surfaces its own defects.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-live-commissioning-verification-gaps.md | kit | 2026-09-07 | pending | 7 |`
