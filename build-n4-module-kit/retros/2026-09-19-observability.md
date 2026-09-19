<!-- review-status: folded -->
# 2026-09-19 · kit · observability

**Session**: 2026-09-19 kit-improvement campaign (branch odd/apply-kit-candidates-2026-09-19) — WU folding the observability cluster.
**Delta count**: 7

## What happened
The kit had one logging line and no spy/fault/audit-vs-log guidance. Cluster OBS-1..OBS-7 in the
master register, from real driver/service code.

## Evidence
- java.util.logging preferred (javax.baja.log.Log @Deprecated): `[ev: corpus B20]`
- spy(SpyWriter) recipe + Spy.ROOT.add: `[ev: code BModbusNetwork.java]`
- System.out bypasses BLogHistoryService: `[ev: corpus B92]`; audit synchronous, user-initiated only `[ev: corpus B33]`

## Proposed kit deltas (folded in this PR)
| Δ | Delta | Target | Token |
|---|---|---|---|
| Δ1 | Logger convention (short-name, level map, isLoggable guard) | `types/observability.md` §1 | `[ev: corpus B20]` |
| Δ2 | System.out anti-pattern + LoggingPermission | `types/observability.md` §1 | `[ev: corpus B92]` |
| Δ3 | Spy page registration + SpyDir/SpyWriter recipe | `types/observability.md` §2 | `[ev: code BModbusNetwork.java]` |
| Δ4 | HogsPage + diagnostics surfaces | `types/observability.md` §2 | `[ev: corpus B33]` |
| Δ5 | appFail/configFail/configFatal + faultCause | `types/observability.md` §3 | `[ev: corpus B20]` |
| Δ6 | audit-vs-log decision table + fire-and-forget AuditEvent | `types/observability.md` §4 | `[ev: corpus B33]` |
| Δ7 | runtime-state exposure priority | `types/observability.md` §5 | `[ev: corpus B33]` |

## Lessons
- Prefer java.util.logging; name the logger by module short-name so it appears in Logger Config.
- Audit only user-initiated out-of-band writes, fire-and-forget; never audit autonomous logic.

---
**Status**: FOLDED into `types/observability.md` (2026-09-19).
