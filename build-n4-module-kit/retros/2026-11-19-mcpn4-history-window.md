<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-19 · kit · mcpn4-history-window

**Session**: kit branch feat/mcpn4-history-window — mcp-n4 deltas M2 and M3 (a client 2.3.0 upgrade retro)
**Delta count**: 3

## What happened
A client station's COV setpoint history held `{fault}` / `{ok}` record pairs 5 s apart on every comm drop.
`n4_read_history` returned the newest records with no status filter and did not say the history was COV, so
both facts were read off the timestamps by hand. Fifteen room temperature and humidity histories sat at 0.0
`{ok}` for a week (the device sends 0.0 for "no sensor") and `n4_histories_audit` said nothing about them.

## Fix
- `n4_read_history`: `from` / `to` name the window as `start` / `end` do (one pair, never both; a bad bound
  is named as the caller wrote it). `status` adds a BQL `where` to the records read and to the count, so
  `limit`, `truncated` and `total_records` count only the matching records. `history_type=true` reads the
  history extensions once (`select slotPath, historyConfig.id, type, historyConfig.interval.encodeToString`)
  and reports `cov`, or `interval` with its period, reusing the audit's `isCov` and `spanText`.
- `n4_histories_audit` `flat=true`: per distinct enabled history that recorded within the window, the
  live-checked `select min(value), max(value) where status.null = false` over `[now - flat_days, now]`; when
  min equals max, one `select count(*)` before the window, so a history younger than the window is never
  flagged. A history with no record in the window stays the stale list's business.
- Both are opt-in: without the new arguments the requests and the reply are unchanged (pinned by tests).

## Evidence
- RED (6 failing): from/to sent no range, status sent no `where`, `history_type` absent, `flat` absent and no
  window read. GREEN after the fix, with `TestReadHistoryWithoutTheNewArgumentsAnswersAsBefore` and
  `TestHistoriesAuditWithoutFlatAnswersAsBefore` green before and after.
  `[ev: go/internal/mcpn4/blackbox/history_window_test.go]` `[ev: go/internal/mcpn4/blackbox/histories_audit_flat_test.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | Applied: `n4_read_history` `from`/`to`, `status` filter, `history_type` | `go/internal/mcpn4/readtools/history_window.go`, `history_type.go` | `[ev: retro client-2.3.0-mcp-n4 M2]` |
| 2 | Applied: `n4_histories_audit` `flat` / `flat_days` | `go/internal/mcpn4/readtools/histories_audit_flat.go` | `[ev: retro client-2.3.0-mcp-n4 M3]` |
| 3 | Open: live-check the `status.ok` / `status.fault` ... where forms on a 4.15 station (only `status.null` is live-checked) and the `history_type` lookup's cost on a large station | live probe | `[ev: go/internal/mcpn4/bql/history_audit.go HistoryStatusWhere]` |

## Lessons
- A history's type and its status pattern are facts the station already holds; a read tool that leaves them
  to timestamp arithmetic hands the operator a job the tool can do in one query.
- A flat value with `{ok}` status is invisible to staleness and status checks: it needs its own rule.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-11-19-mcpn4-history-window.md | kit | 2026-11-19 | pending | 3 |`
