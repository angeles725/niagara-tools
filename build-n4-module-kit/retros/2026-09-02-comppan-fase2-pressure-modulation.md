# Retro — CompPan Fase 2 (suction-pressure modulation) · 2026-09-02

Module: CompPan-rt (3-compressor rack, PANCCADIA Leon). Type: pure logic (rt control).
Change: additive Fase 2 on top of the STORED Fase 1 — no re-slot of inputs.
Built by session "Ayudante"; Fase 1 authored by session "Cabeza". PROPOSED kit deltas below
(propose-never-apply — a human folds these into METHODOLOGY.md / types/logic.md).

## What this build PROVED

1. **An additive control phase should ship DORMANT, gated by a config default, not "on".**
   Fase 2 (pressure hysteresis) activates only when `suctionBand > 0`; the slot defaults to
   `0d`, so a freshly-installed jar behaves EXACTLY like Fase 1 (demand-count) until the
   operator commissions the band. This extends the existing type-guide lesson "a sensor-fault
   control posture is an operator config slot, not a baked constant" (logic.md §Safety) to
   whole PHASES: never let a jar upgrade silently change control on a live plant — the new
   behavior waits behind an operator-set gate. [ev: this build; CompressorControl.step L~a
   `if (suctionValid && c.suctionBand > 0d)` else fallback]

2. **Backward-compat of a control rewrite is provable by the OLD tests, for free, when the new
   path is gated.** Because Fase 2 is gated on `suctionBand > 0` and the 11 Fase-1 tests all use
   band==0 (or invalid suction), they stayed green UNCHANGED (17/17 total after adding 6 Fase-2
   tests). Gate the new branch on a config the old tests leave at default → the old suite is your
   regression guard at zero cost. [ev: standalone junit 17/17]

3. **Hysteresis-staging model that reconciles pressure vs demand-count (the §83 sub-staging
   trap):** demand is a GATE + FALLBACK, never an upper cap. `D==0 -> rack off`; both suction
   sensors invalid OR band==0 -> demand-count (Fase 1); otherwise suction band decides target
   (`sp ± band/2`, above=stage up after stageUpMs, below=stage down after stageDownMs, in-band
   HOLD), bounded only by `available` and the safety cotas. Capping target at demandCount would
   re-introduce the exact under-staging the writeup warned about (one room, door open, counts "1"
   but pulls a big load). [ev: test demandGate_zeroDemandForcesOff + invalidSuction_fallsBack]

4. **Reciprocating (Maneurop MTZ) = fixed capacity, no unloading → "pressure modulation" is still
   DISCRETE staging, just triggered by the band instead of the count.** Confirms the Fase-1
   staging skeleton is the right substrate; Fase 2 is a pure target-computation change, no new
   output surface. Tune minOn/minOff/stageDelay to the MTZ starts-per-hour limit at commissioning.

5. **`Clock.millis()` sentinel timers need the MIN_VALUE guard BEFORE the subtraction — again.**
   The two new sustain timers (`aboveSince`/`belowSince`) follow the same `== Long.MIN_VALUE`
   guard the Fase-1 `lastStageMs` fix established; both also reset in `resetTransient()`. Reinforces
   the standing lesson: any monotonic-time delta off a MIN_VALUE sentinel overflows negative for
   real timestamps and must be guarded, or the gate latches. [ev: Fase-1 lastStageMs crit bug]

## Build facts (reproduction)
- Tests: NOT via gradle `niagaraTest` (needs Windows native bin/test + dev license, fails in WSL).
  Run standalone: `javac -source 8 -target 8 -cp junit-4.13.2.jar:hamcrest-core-1.3.jar` then
  `java org.junit.runner.JUnitCore com.angeles.CompPan.CompressorControlTest` → 17/17.
- Build: `toolbelt/build.sh --plugin-version 7.6.17 <root> CompPan <niagara_home>` (4.14 => 7.6.17).
- Gate: bytecode major 52, signed, types resolve; STORED variant via `stored-repack.sh` then
  `verify-module.sh --stored --target-version 4.14` → 0 deflated, baja 4.14 <= 4.14, ALL PASS.
- Adding 5 slots (annotation + a compiling AUTO stub) regenerated cleanly via slotomatic, which
  re-derived the AUTO region and updated the type hash (confirms logic.md §Regenerating slots).
