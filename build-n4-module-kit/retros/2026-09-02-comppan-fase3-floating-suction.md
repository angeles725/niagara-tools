# Retro — CompPan Fase 3 (floating suction, mechanism-only) · 2026-09-02

Module: CompPan-rt. Additive on top of Fase 2. Built by session "Ayudante". The R404A curve
depends on technician-validated field data that had NOT arrived, so this build ships the full
MECHANISM with the data function stubbed. PROPOSED kit deltas (propose-never-apply).

## What this build PROVED

1. **"Build the mechanism, stub the unvalidated safety-critical data" is a shippable state.**
   When a control feature needs field data that is not yet validated (here: the R404A P-T table
   + gauge/absolute confirmation, which sets the suction pressure that drives compressors on a
   live rack), do NOT invent placeholder numbers. Ship the ENTIRE mechanism (slots, selection
   logic, status, fallback, tests) with the data-dependent function as a STUB returning NaN. The
   feature stays provably INERT (NaN → adapter keeps the fixed setpoint), zero invented numbers
   reach the plant, and activation later is a one-function fill-in. The operator can even enable
   the feature slot and nothing happens until the curve is filled. [ev: r404aDewPressurePsia stub
   returns NaN; test floatingSetpoint_inertUntilCurveFilled asserts inert with an active zone]

2. **A phase that only changes an INPUT to the previous phase needs no change to the core.**
   Fase 3 only changes HOW `suctionSetpoint` is computed (adapter reads room setpoints + calls,
   calls a pure helper, overrides `cfg.suctionSetpoint`); the pure `step()` staging logic is
   untouched. Result: all Fase 1 + Fase 2 tests stay green unchanged, blast radius = the new
   helper + adapter wiring. Push new logic to the boundary that feeds the stable core.

3. **Store refrigerant P-T tables in ABSOLUTE (psia); convert to the gauge sensor with a site
   atmosphere slot — never bake 14.7.** Absolute is altitude-independent, so one table is correct
   anywhere; `psig = psia − localAtmPsi`. León, Gto (~1,800 m) ≈ 11.8 psia, ~3 psi off sea level
   — not negligible for a 5-psi control band. Exposed `localAtmPsi` (default 11.8) as an operator
   slot; vented-gauge vs sealed-gauge is then just this one number, confirmed in the field.
   [ev: floatingSetpoint returns psia − localAtmPsi]

4. **A floating reference keys off the coldest ACTIVE zone, not the min of all setpoints.**
   `SSTreq_k = setpoint_k − coilTD` only for rooms with `calling==true` and a valid setpoint;
   target = min over those. A satisfied (not-calling) zone must not drag suction down — that is
   the efficiency win. Needs per-room call + setpoint, which the module already had. [ev: tests
   minActiveSst_satisfiedColdZoneDoesNotDragItDown / _invalidSetpointExcluded / _noActiveZone]

5. **coilTD is a property of the EVAPORATOR, not of the condensing unit.** The Danfoss Optyma
   OP-HGZ100D39Q datasheet has no coilTD; it is `T_return_air − SST_evap` at the coil (~8 K design
   for frozen, per Danfoss DT1 min). The rack's common-header LP sensor sees header SST, not each
   coil's, so the rack-referenced value is an ESTIMATE (name it so). Keep coilTD an operator slot
   to validate/measure, never a baked constant. [ev: operator + Acompanante field analysis]

## Build facts
- Tests standalone (WSL): 22/22 (17 Fase 1+2, 5 Fase 3 selection/inert). No pressure value asserted.
- build.sh --plugin-version 7.6.17; gate ALL PASS (major 52, signed); STORED 0 deflated, baja 4.14.
- Sensors (site): Johnson Controls P499 gauge (psig), 0–10 VDC. Suction = 0–100 psig (100 mV/psi,
  good resolution for ~15–75 psig); discharge = 0–500 psig. Scaling is a driver/point concern; the
  curve is range-agnostic once the point is scaled. HAZARD for the doc: never a 0–100 on discharge.

## UPDATE — curve filled (same day, operator data arrived)
The operator provided the validated R404A dew table (ABSOLUTE psia) + coilTD 8 K, and confirmed the
P499 is VENTED gauge (localAtmPsi ≈ 11.82 for León). `r404aDewPressurePsia` is now filled (12-anchor
table, linear interp, clamp to [-30,+10]); localAtmPsi default = 11.82; curve + full-chain tests added
(26/26). The "mechanism-stub first" lesson (§1) still stands as the general pattern — it let the module
build, test, and pass the gate for a full round while the field data was still pending, then absorb the
data as exactly the one-function fill-in it was designed to be, with no rework to the mechanism.

Fase 3 stays DORMANT by default (floatingSuction=false; no effect if suctionBand=0). To ACTIVATE:
operator measures the exact local atmosphere on site (refine localAtmPsi), confirms coilTD per
evaporator, then sets suctionBand>0 and floatingSuction=true.
