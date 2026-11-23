<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-19 · kit · mcpn4-status-isok

**Session**: a field JACE (4.14.0.162, client B), first live use of the `n4_read_history` status filter shipped the same day (#829).
**Delta count**: 2

## What happened
`status: fault` on a COV setpoint history returned 29 fault records over 7 days and `history_type` named it COV. `status: not-ok` returned zero rows, and its record count "answered no whole number". The filter was `status.ok = false`, which BQL does not resolve: it matches nothing and returns an empty result with no error. The fake station accepted the string, so every test was green.

## Evidence
- Same history, same session: `where status.ok = false` → 0 rows; `where status.isOk = false` → `{down,stale}`, `{stale}`, `{fault}` records; `where not status.isOk` → the same; `where status.fault = true` → 29 records.
- Fix: `HistoryStatusWhere` ok / not-ok use `status.isOk`; the blackbox test pins the request strings; tool description updated.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. ok / not-ok filters use `status.isOk` (live-checked) | `go/internal/mcpn4/bql/history_audit.go` | `[ev: retro mcpn4-status-isok Δ1]` |
| Δ2 | OPEN. A BQL where-clause that the fake station accepts but a station silently resolves to nothing cannot be caught offline: every new BQL predicate shipped by a tool carries a live-check note (station version + date) in its source comment, and the retro of the feature lists the predicates still unchecked as an open delta | `go/internal/mcpn4/bql/` doc comments + BUILD-LOOP §5 kit tooling doctrine | `[ev: retro mcpn4-status-isok Δ2]` |

## Lessons
- A BQL field that does not exist is not an error: it is an empty answer. The M1 warning covers projected columns; a where-clause on a missing field still returns zero rows silently.
