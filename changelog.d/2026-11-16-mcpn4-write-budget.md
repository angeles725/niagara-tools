### Changed — mcp-n4: the write budget never blocks the save

- `n4_save_station` neither needs nor spends a `--max-writes` write, so a bulk job that spent the
  budget can still persist its work; the budget refusal says so.
- Every write reply (dry run and execution) carries `write_budget` `{executed, max, remaining,
  save_counts}`.
