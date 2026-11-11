### Added — mcp-n4 history extensions in one batch
- `n4_add_history` adds one interval history extension per point (1 to 50), already configured (`historyName`, `interval`, `enabled`) under one dry run and one confirmation token; it refuses an existing child of the same name and a history name that would repeat across the batch.

### Changed — mcp-n4 n4_set_slot slot types
- `n4_set_slot` types a slot the station omits at its default from the type definition (registry channel) and accepts `baja:Format` and `baja:RelTime`, so a history extension's `enabled`, `historyName` and `interval` can be set.
