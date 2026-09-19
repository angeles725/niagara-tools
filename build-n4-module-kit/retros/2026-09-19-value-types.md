<!-- review-status: folded -->
# 2026-09-19 · kit · value-types

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the value-types cluster.
**Delta count**: 8

## What happened
The kit covered BStatus/slots but had no doc on authoring your OWN value types. Cluster TYP-G1..G8
in the master register, from real code (BSampleRate, BHistoryId, BAlarmTimestamps, aaphp enums).

## Evidence
- BSimple four I/O methods + intern(): `[ev: code BSampleRate.java]` `[ev: code BHistoryId.java]`
- @NiagaraEnum getRange().get() (not switch): `[ev: corpus B4]`; BStruct as action/learn bag `[ev: code BAlarmTimestamps.java]`
- BFacets factories + keys: `[ev: code BBacnetActionCommand.java]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | BFrozenEnum @NiagaraEnum + getRange().get() make() | `types/value-types.md` §1 | `[ev: corpus B4]` |
| Δ2 | BDynamicEnum + BEnumRange.make | `types/value-types.md` §2 | `[ev: code AaPhpAttributeConversion]` |
| Δ3 | BSimple 4 I/O methods + intern() | `types/value-types.md` §3 | `[ev: code BSampleRate.java]` |
| Δ4 | BStruct composite + BStruct-vs-BComponent rule | `types/value-types.md` §4 | `[ev: code BAlarmTimestamps.java]` |
| Δ5 | BFacets factories (make/merge/makeEnum/makeRemove) | `types/value-types.md` §5 | `[ev: code BBacnetActionCommand.java]` |
| Δ6 | BFacets keys + facets-are-UI-hints | `types/value-types.md` §5 | `[ev: corpus B4]` |
| Δ7 | BUnit getUnit/NULL/make/convertTo | `types/value-types.md` §6 | `[ev: corpus B745]` |
| Δ8 | Persistence: BOG v= round-trip, newCopy, REMOVE_ON_CLONE | `types/value-types.md` §7 | `[ev: corpus B5]` |

## Lessons
- BSimple MUST implement all four encode/decode + hashCode; intern() gives == identity for equal values.
- Facets are UI hints — the server never validates min/max on a write.

---
**Status**: FOLDED into `types/value-types.md` (2026-09-19).
