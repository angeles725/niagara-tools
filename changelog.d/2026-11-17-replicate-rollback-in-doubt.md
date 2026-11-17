### Changed — mcp-n4: rollback outside the write budget, in-doubt replication rollback, held-register note (retro mcpn4-replicate-points Δ5, Δ6)

- **Every `n4_rollback` is outside `--max-writes`**, like `n4_save_station`: it runs with the budget spent and spends
  no write, nested components and links included. `write_budget` gains `rollback_counts: false`; the budget refusals
  say `n4_save_station and n4_rollback still run` / `are outside the budget`. A replication of N adds now needs N
  writes (was N + 1: #777's write kept for the rollback is gone).
- `n4_rollback` of an in-doubt `n4_replicate_points` batch whose failed add is journaled removes only the copies the
  station accepted (by the name its reply assigned) that the read-back finds with the type the batch added; the
  failed add, the adds never sent changed copies and copies a later batch wrote under are kept, each with a verdict in the plan notes and in
  `observed.copies`; a rollback that keeps a copy is `partial`.
- The replication dry run reads each target four levels deep and notes a copy at the register a writable Modbus
  point already on that device holds.
