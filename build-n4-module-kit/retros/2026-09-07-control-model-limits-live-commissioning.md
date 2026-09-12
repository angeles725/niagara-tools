<!-- review-status: pending -->
# 2026-09-07 · kit · control-model-limits-live-commissioning

**Session**: PANCCADIA León live commissioning over oBIX (Cuarto 5 split unit + valve-hold override). Limits of the ColdRoomPan control model surfaced when the operator's real requirements didn't fit the built-in evaporator logic. Companion to `live-commissioning-verification-gaps`.
**Delta count**: 5

## What happened

The operator's requirements for the split A/C room (Cuarto 5) and for a "keep the valve open" case exposed places where the ColdRoomPan `BEvaporatorUnit` model does not fit, and where a documented workaround has a subtle trap:

1. **Split-unit sequence (fan-first) is the OPPOSITE of the evaporator model.** `BEvaporatorUnit` sequences valve-first-then-fan (refrigeration order); a split A/C needs fans-first → compressors, and compressors-off-before-fans (fan must outlast the compressor). It cannot be done with the unit's slots — it required a full wire-sheet sequencer (delays + gates) built by hand.

2. **Multi-stage by demand needs per-stage thresholds, and there is no "derived differential."** Two units share one setpoint; to stage them the second compressor's differential had to be DERIVED from the first (`Add`/`Subtract` blocks linked into the second unit's diffUp/diffDown) so it tracks operator edits — otherwise editing the differential breaks the staging.

3. **Defrost has hard priority over HOA and there is NO way to disable it.** During (air) defrost the valve is force-closed and HOA is ignored (`applyHoaOutputs` returns while `inDefrost`); `BDefrostMode` only has `interval`/`schedule` (no `off`), and `BDefrostController` has no `enabled` slot. So "keep this valve open" is impossible without preventing the defrost — which today needs a code change or an out-of-band relay OR.

4. **The relay-OR workaround has a numeric→boolean trap.** Feeding `valveMode` (double 0/1/2) straight into a boolean `Or` converts `!=0` → true, so **both Encender(1) AND Apagar(2) forced the valve ON** — Apagar opened the valve. Correct form is `Or(valveOut, Equal(valveMode,1))`.

## Evidence
- Split sequence not native: `BEvaporatorUnit.java:1136-1162` (applyRunCmd rising: valveOut first, evapOut after startDelay); operator sequence built as Programacion/Cuarto5_Secuencia (COMPdelay/FANruns/gates). `[ev: live oBIX PANCCADIA 2026-09-07]`
- Derived differential: stageUp=`Add(U3.evapDifferentialUp,3)`, stageDown=`Subtract(U3.evapDifferentialDown,1)` linked to EvaporatorUnit2 diffs. `[ev: live oBIX]`
- Defrost priority / no disable: `applyHoaOutputs` `if(inDefrost)return` (`:1224`), `enterDefrost` closes valve (`:1407`); `BDefrostMode` enum = interval/schedule only; `BDefrostController` has no `enabled`. `[ev: BDefrostMode.java:22-23]` `[ev: BDefrostController.java]`
- valveMode-as-boolean trap: Or5-9 in `/Drivers/NrioNetwork/io34_5_2/points` initially `Or(valveMode, valveOut)`; valveMode=2 → bool true → valve ON on Apagar; fixed with `Equal(valveMode,1)`. `[ev: live oBIX Or5-9 + Equal0-4]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Document a **split/sequenced-unit pattern** (fan-first, compressors staged, reverse shutdown, fan outlasts compressor) as a reusable wire-sheet recipe OR a `BEvaporatorUnit` "split mode" — the refrigeration model does not fit A/C order. | `types/logic-authoring.md` (+ optional module mode) | `[ev: live oBIX Cuarto5]` |
| Δ2 | Document the **derived-differential staging** pattern (Add/Subtract linked into the second stage's diff) so multi-stage-by-temperature survives operator edits, and warn against hardcoding the second diff. | `types/logic-authoring.md` | `[ev: live oBIX stageUp/stageDown]` |
| Δ3 | Add a **defrost enable/Off** to ColdRoomPan (per-evaporator `defrostEnable`, or a `BDefrostMode.off`, or skip-on-valveMode==HAND) so "keep valve open" doesn't need an out-of-band relay OR; surface it via `defrostSkipped`/`lastSkipReason`. | ColdRoomPan `BDefrostController`/`BEvaporatorUnit` + `types/logic-authoring.md` | `[ev: BDefrostMode.java:22-23]` |
| Δ4 | Kit rule + lint: **never link a multi-state/enum (0/1/2) numeric straight into a boolean gate** — it converts `!=0`→true, silently merging distinct states (HAND vs OFF). Require an explicit `Equal(x,N)`. | `toolbelt/` (lint) + `types/logic-authoring.md` | `[ev: live oBIX Or5-9]` |
| Δ5 | Note that **defrost has priority over HOA and there is no disable** — document it so integrators don't expect HOA to override defrost, and point to the defrost-enable (Δ3) as the intended lever. | `types/logic-authoring.md` § defrost | `[ev: BEvaporatorUnit.java:1224]` |

## Lessons
- **The evaporator model is refrigeration-shaped (valve-first).** A/C split units (fan-first, staged compressors, reverse shutdown) need a sequencer, not the unit's slots — say so up front.
- **Multi-stage by temperature = per-stage thresholds; derive the second from the first** so operator edits don't break staging.
- **Defrost outranks HOA and can't be turned off today.** Any "keep the output on during defrost" requirement needs a defrost-enable (code) or an out-of-band override — decide deliberately.
- **A multi-state numeric into a boolean gate is a trap:** `!=0`→true merges HAND and OFF. Always gate with `Equal(mode, value)`.
- **These only surfaced live.** The build gate never exercised the operator's real control requirements — commissioning did (ties to `live-commissioning-verification-gaps`).

---
**Status**: PENDING — INDEX row appended: `| 2026-09-07-control-model-limits-live-commissioning.md | kit | 2026-09-07 | pending | 5 |`
