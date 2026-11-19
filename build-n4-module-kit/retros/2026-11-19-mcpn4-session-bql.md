<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-19 · kit · mcpn4-session-bql

**Session**: a field station (4.14) behind a tunnel; mcp-n4 reads during a station restart and a BQL override-limit audit.
**Delta count**: 3

## What happened
The station restarted at 04:40. mcp-n4 calls at 04:43 were answered `com.tridium.box.ServerSessionException: Could not find Server Session` (BBoxService.validateFrameServerSession). Separately, `select maxOverrideDuration from control:NumericWritable` returned 135 rows of `"NULL"`: maxOverrideDuration is a ControlPoint facet key, not a slot, and BQL answers NULL for a name it cannot resolve, so the reply read as "not set".

## Evidence
- `jsonrpc/session_loss.go` already reconnected once with the last `n4_connect` and retried a read or dry run once, and refused to retry an execution. Missing: a second loss after the retry returned the raw BOX text, and the execution message did not start with "session lost".
- `readtools/bql.go` returned the rows with no check of the projected names.
- RED through the fake station (`blackbox/session_loss_test.go`, `blackbox/bql_unknown_column_test.go`): `TestASecondLossInARowIsReportedNotLooped`, `TestAWriteExecutionThatLosesItsSessionIsNotRetried`, `TestAColumnNullOnEveryRowThatIsNoSlotIsWarned` failed before the fix; `TestAReadThatLosesItsSessionReconnectsOnceAndSucceeds`, `TestAColumnNullOnSomeRowsOnlyIsNotWarned` and `TestANormalBqlReplyIsUnchanged` guard the unchanged paths.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-19. A second session loss in a row after the one reconnect is a named error (never looped, never raw BOX text); an execution answers "session lost ... redo the dry run" and is not retried | `go/internal/mcpn4/jsonrpc/session_loss.go` | `[ev: retro mcpn4-session-bql Δ1]` |
| Δ2 | APPLIED 2026-11-19. `n4_bql_query` warns on a column NULL on every row whose name is no slot of the queried type (type contract via the session registry cache); unchecked contract gives the weaker warning | `go/internal/mcpn4/bql/nullcolumns.go`, `readtools/bql_columns.go` | `[ev: retro mcpn4-session-bql Δ2]` |
| Δ3 | OPEN. State both behaviours in the `n4_connect` and `n4_bql_query` descriptions (registry.json and the mirrored `testdata/tools/*.json` goldens) | `go/internal/mcpn4/registry/registry.json`, `go/internal/mcpn4/testdata/tools/` | `[ev: retro mcpn4-session-bql Δ3]` |

## Lessons
- A BQL `NULL` is not evidence: check the column names a slot before reading it as "not set".
- A reconnect-and-retry needs a terminal answer for the retry's own failure, or the operator sees the raw station text again.
