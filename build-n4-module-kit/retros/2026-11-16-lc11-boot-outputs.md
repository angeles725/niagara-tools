<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · lc11-boot-outputs

**Session**: rows LC11 and LP1 of a client A retro: a control cycle run before steady state drives outputs past
the boot safety holds and bumps a revision on every restart; the lab restart proof that found both.
**Delta count**: 2

## What happened
After the LC10 fix, a second lab restart proof of client A's compressor control module set non-default timers
(power-on delay 60 s, minimum off 2 min), linked every input that gates demand and toggled the inputs between
restarts. Read-backs started before `Station Started` showed a compressor commanded ON seconds after the station
started, OFF when the power-on stagger armed in `atSteadyState()`, and ON again when the stagger expired: the
module armed the stagger and its minimum-off-across-restart seed only at steady state, and an input change ran the
control cycle first. The same reads showed the persisted staging-config revision going 1 → 2 → 3 → 4 over three
restarts: the model's "last valid" config was not seeded, so the first cycle of every boot saw a change. A first
proof round with default timers and an unlinked suction input (which read `0.0 {ok}` and won the lower-of-two
pressure choice, so there was never demand) had shown neither defect. The fix is in progress in the client module.

## Evidence
- Niagara lifecycle: links propagate from Component Start; steady state exists to delay control commands to
  external devices. `niagara-help/versions/4.15.6.20/devguide-clean/station.txt:26`,
  `niagara-help/versions/4.15.6.20/devguide-clean/station.txt:30`.
- Tridium's staging block arms in `started()` and forces outputs off before its first calculation:
  `niagara-help/versions/4.15.6.20/source/kitControl-rt/com/tridium/kitControl/hvac/BSequence.java:805-818`.
- Live (client A, lab station, Niagara 4.15.6.20): restart C of lab proof round 2 (output ON-OFF-ON at boot) and
  restarts A to C (revision 1 → 4); the proof log keeps every timestamped read and the station log lines.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. LC11: arm boot output holds (power-on stagger, min off across restart, anti-short-cycle) first in the cycle, write OFF while held, TRANSIENT outputs, seed derived "last valid" state with LC10's seed; source-structural test | `types/logic.md` LC11 | `[ev: retro lc11-boot-outputs Δ1]` |
| Δ2 | APPLIED 2026-11-16. LP1: lab restart proof recipe (non-default timers, every demand input linked, inputs written during boot, reads from before `Station Started` with the clock offset, per-read checks, each backup missing once, proven cleanup) | `build-verify.md` § Lab restart proof (LP1) | `[ev: retro lc11-boot-outputs Δ2]` |
