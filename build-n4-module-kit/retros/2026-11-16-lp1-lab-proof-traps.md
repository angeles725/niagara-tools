<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · lp1-lab-proof-traps

**Session**: row LP1b of a client A retro: a third lab restart proof of the client's compressor control module
and the four ways it could have passed without testing what it claimed.
**Delta count**: 2

## What happened
Lab proof 3 of client A's compressor control module (lab station, Niagara 4.15.6.20) re-ran the LP1 recipe after
the LC11 fix. Four traps showed up: the cleanup grep used the module name, while `config.bog` writes the module's
type prefix, so it could never match; the minimum-off hold and the stage delay plus power-on delay were close, so
the proof could not say which hold kept the output OFF; removing the proof component and restoring it with
`n4_rollback` built a new instance, so `stopped()` then `started()` never ran on the same object; and a jar swapped
under the same file name left the registry "Up-to-date" until the jar was touched (mcp-n4 METHODOLOGY M14).

## Evidence
- Lab proof 3 log (client A, lab station, Niagara 4.15.6.20): cleanup grep, hold timings, rollback instance and the
  `Module changed … Rebuilding registry` start log; kept in the research repository's evidence folder.
- `mcp-n4-kit/METHODOLOGY.md` live rule "a module swapped under the same jar name must change the jar before the
  start" (M14).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. LP1 step 1: set two holds far apart (e.g. minimum off 5 min against seconds of stage plus power-on delay) so the proof tells them apart | `build-verify.md` § Lab restart proof (LP1) | `[ev: retro lp1-lab-proof-traps Δ1]` |
| Δ2 | APPLIED 2026-11-16. LP1 step 7 and traps: grep the config.bog type prefix, a rollback is not a restart, touch a jar swapped under the same name | `build-verify.md` § Lab restart proof (LP1) | `[ev: retro lp1-lab-proof-traps Δ2]` |
