# Retro — ColdRoomPan fan-mode + defrost-entry sequence (additive) · 2026-09-02

Module: ColdRoomPan-rt. Additive changes on top of the hardened ColdRoomPan, built by session
"Acompañante" at operator request: per-evaporator fan mode, valve-first fall + run-on, defrost-entry
delay, and the removal of a dead `@NiagaraType` registration. PROPOSED kit deltas (propose-never-apply).

## What this build PROVED

1. **An additive slot change to a LIVE module preserves station instances + links — but the JACE
   will not auto-install it without a version bump.** New frozen slots load at their default on the
   existing bog; instances and link-marks are keyed by name, so nothing is re-placed or re-linked
   ([CERT-doc], buildingComplexes: every frozen property carries a compile-time default). BUT Software
   Manager compares versions: same version → shows "Up to Date", never offered for upgrade. Either bump
   the module version, or force "Re-Install <same version>" by hand. → Methodology delta: an "additive
   upgrade" note — instances/links survive, but say version-bump-or-Re-Install explicitly.

2. **A discrete mode selector used ONLY as an internal slot type is a frozen enum, not a double.**
   The plain-double rule (logic.md) exists only for values LINKED ACROSS custom modules. `BFanMode`
   (withCall/runOnDelay/continuous) is internal config → authored as a `BFrozenEnum` like BStagingMode,
   registered in module-include.xml. Reinforces logic.md line 23: the double is for cross-module links,
   the frozen enum for local behavior selectors.

3. **Every actuation timer must be cancelled on the shared cancel path, or a stale fire flips an
   output after the command already changed.** The unit went from one `startDelayTicket` to three
   (`startDelay`, `stopDelay` run-on, `defrostFanOffDelay`). `cancelTicket()` must cancel ALL of them,
   and it must be called at the top of BOTH edges of `applyRunCmd` and at defrost enter/exit. A run-on
   timer left armed when a new call arrives would turn the fan off mid-cooling.

4. **A mode that holds an output OUTSIDE the command edges needs its own apply method, wired into
   atSteadyState + changed(modeSlot) + defrost-exit — the edge handler alone is not enough.** The
   `continuous` fan must run even with no call (no edge ever fires). `applyFanRunMode()` forces the fan
   on outside defrost; `applyHoaOutputs()` also had to honor continuous (it was setting evapOut=lastCmd,
   which would drop a satisfied-room continuous fan). Lesson: when a feature decouples an output from the
   command edge, audit every place that sets that output.

5. **The verify-gate `types` check catches a dead `@NiagaraType` registration — but only on rebuild.**
   module-include.xml still registered `BHoaMode` (class deleted when HOA became a plain double); the jar
   shipped that dead type and the JACE logged `Missing class ColdRoomPan:HoaMode`. The gate's `types`
   check FAILED on the first rebuild and surfaced it, which is exactly the safety it is for. Fix = remove
   the line, rebuild → 6 declared types == 6 classes. Lesson to keep in the checklist: when deleting a
   `@NiagaraType` class, delete its module-include.xml line in the same change; the gate will catch it if
   you actually rebuild (a stale committed jar hides it).

## Build

Java 8 + slotomatic, target PowerB 4.15.3.28 (plugin 7.6.22, its own niagara_home). Verify gate ALL
PASS (bytecode 52, signed, 6 types resolve). Backward-compatible defaults (fanRunMode=withCall,
stopDelay=0) so a live room's behavior does not change on upgrade until the operator opts in.
