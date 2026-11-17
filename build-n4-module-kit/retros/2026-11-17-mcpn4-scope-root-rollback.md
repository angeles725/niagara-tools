<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-17 · kit · mcpn4-scope-root-rollback

**Session**: the lab live check of the n4_ack_alarm selection (#808) on the 4.15 lab station.
**Delta count**: 2

## What happened
The check used `--write-scope station:|slot:/AckLab`. `n4_create_component` created `AckLab` under the
station root (a create may make the scope root itself), but `n4_rollback` of that batch was refused as
outside the write scope: the rollback checked the parent its inverse removes from, not the created path the
create was checked on. Undoing the lab setup needed a wider scope, so the one write a scope could make was
the one it could not undo.

## Evidence
- Lab log `odd/evidence/2026-11-17-ack-select-live.log` in the research repository (rollback refused under
  scope AckLab, accepted under scope `/`).
- `go/internal/mcpn4/writeops/rollback.go` checked every recorded target with `Scope.Check`.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-17. A rollback that only removes the write-scope root is checked on the created path, as its create was, so the same scope undoes its own root create; later work inside the root still refuses it, and `n4_remove_component` of the root stays refused (a scope never deletes itself and its contents in one call). `TestTheCreateOfTheScopeRootRollsBackUnderTheSameScope`, `TestTheScopeRootRollbackStillNeedsTheScope`, `TestRemovingTheScopeRootIsStillRefused`, blackbox `TestTheSameScopeRollsBackItsRootCreateOnlyWhileNothingWasWrittenInside` | `go/internal/mcpn4/writeops/rollback_scope_root.go`; `mcp-n4-kit/README.md` write scope; `mcp-n4-kit/METHODOLOGY.md` L5 | `[ev: code go/internal/mcpn4/writeops/rollback_scope_root.go]` |
| Δ2 | APPLIED 2026-11-17. Correction after the independent verifier of #809 (BLOCKER P2/P2b): the exception looked only at the shape of the inverse op, so the rollback of an `n4_copy_component` (or replicate/deploy) batch that named its copy after the scope root was admitted under that scope and deleted the root with hand-made and later children inside (those batch kinds never run `laterWork`). The exception is now limited to an `n4_create_component` batch with its single remove op, the one kind `laterWork` protects. `TestACopyRollbackUnderTheRootScopeIsRefused`, `TestACopyRollbackUnderTheRootScopeIsRefusedAfterLaterWork`, `TestACreateRollbackUnderTheRootScopeKeepsAHandMadeChild` | `go/internal/mcpn4/writeops/rollback_scope_root.go` | `[ev: code go/internal/mcpn4/writeops/rollback_scope_root.go]` |

## Lessons
- A scope rule for a forward write and the rule for its inverse must check the same path, or the one
  write a scope admits becomes the one it cannot undo.

- A guard exception keyed on the shape of an op must also name the tool that wrote it: other tools record
  the same op shape without the same protections.
