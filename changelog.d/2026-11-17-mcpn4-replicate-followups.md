### Fixed — n4_replicate_points keeps a write for its rollback and folds a trailing-slash target

- `n4_replicate_points` refuses a replication that would leave no write of `--max-writes` for its own
  `n4_rollback` (one that used the last write could be rolled back only after a server restart).
- Two targets that differ only by a trailing `/` are one target and are refused as a duplicate (they put two
  copies under one parent).
