<!-- review-status: folded -->
# 2026-09-19 · kit · actions-authoring

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — folding the actions-authoring cluster from the master candidate register.
**Delta count**: 6

## What happened
A multi-source research sweep (project memories, corpus, devguide, decompiled `organized/`)
found the kit documented `@NiagaraAction` only in scattered lines with no dedicated doc. The
action side — dispatch contract, threading, flag semantics, typed args, topics, RPC boundary —
was the missing companion to the well-covered property/slot docs. Registered in
`odd/tasks/kit-improvement-candidates-2026-09-19.md` (cluster AC1-AC7).

## Evidence
- `do`+Capitalize dispatch, both signatures tried: slot-o-matic `getMethod("do"+cap, paramType[, Context])` `[ev: corpus authoring-exemplars]`
- ASYNC/timer share one engine thread: devguide `execution.txt` `[ev: corpus module-mechanics]`
- Flag hex values (HIDDEN 0x04, ASYNC 0x10, CONFIRM_REQUIRED 0x80, OPERATOR 0x100, NO_AUDIT 0x800): `Flags.java` `[ev: corpus slots-flags-status-units-java8]`
- `parameterType`/`returnType`/`defaultValue` (required when parameterType set): devguide `slot-o-matic.txt`; ackAlarm(BAlarmRecord) `[ev: corpus authoring-exemplars]`
- `@NiagaraTopic` newTopic()/fire(): report-rt BReportSource; corpus B538 `[ev: corpus B538]`
- `@NiagaraRpc` vs `@NiagaraAction` surface boundary: `[ev: corpus B507]` and `[ev: corpus B822]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `doX()` dispatch general law + Context overload; wrong handler name = silent no-op | `types/actions.md` §1 | `[ev: corpus B822]` |
| Δ2 | ASYNC vs caller-thread threading model; blocking in ASYNC stalls all timers | `types/actions.md` §2 | `[ev: corpus module-mechanics]` |
| Δ3 | Action flag table incl. CONFIRM_REQUIRED (destructive guard) + NO_AUDIT (callback flood) | `types/actions.md` §3 | `[ev: corpus slots-flags-status-units-java8]` |
| Δ4 | Typed args: parameterType/returnType/defaultValue; missing defaultValue = slotomatic error | `types/actions.md` §4 | `[ev: corpus authoring-exemplars]` |
| Δ5 | Actions vs `@NiagaraTopic` (event source, fire()); when to pick which | `types/actions.md` §5 | `[ev: corpus B538]` |
| Δ6 | `@NiagaraRpc` (bajaux/rpc, CSRF-gated) vs `@NiagaraAction` (oBIX/Fox) serving boundary | `types/actions.md` §6 | `[ev: corpus B507]` |

## Lessons
- The `do`+Capitalize handler rule holds for EVERY action; a mismatched name fails silent.
- `ASYNC` is not "run in the background" — it is "run on the shared engine thread"; block there and every timer stalls.
- `CONFIRM_REQUIRED` and `NO_AUDIT` are cheap, high-value flags the kit never mentioned.
- Topic vs action is an architectural choice (many-listeners vs direct-command), not a style choice.

---
**Status**: FOLDED into `types/actions.md` (2026-09-19).
