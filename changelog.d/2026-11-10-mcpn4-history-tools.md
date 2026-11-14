### Added — mcp-n4 history extensions in one batch
- `n4_add_history` adds one interval history extension per point (1 to 50), already configured (`historyName`, `interval`, `enabled`) under one dry run and one confirmation token; it refuses an existing child of the same name and a history name that would repeat across the batch.
