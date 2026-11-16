### Changed — n4call refuses destructive writes without --allow-destructive

- `n4call` refuses, with exit 2 and before the server starts, a calls file whose write calls a
  destructive tool (`n4_remove_component`, `n4_rollback`, `n4_save_station`, taken from the
  server's `destructiveHint`) unless `--allow-destructive` or `--dry-run-only` is given; the
  refusal names the line and the tool. A run with writes says on stderr whether they execute.
  A wrapper that dropped `--dry-run-only` could otherwise execute a rollback planned as a dry run.
