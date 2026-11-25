### Fixed — mcp-n4 `SelectedSlots` no longer reads `TOP n` / `DISTINCT` / `ALL` as part of the first column

- `select top 10 slotPath, name from ...` named its first column `top 10 slotPath`, and `select distinct a from ...` named
  `distinct a`; the `name` column was then not decoded by position and `NullColumns` warned about a slot nobody queried.
  The select list now drops a leading `TOP <n>` and then `DISTINCT` or `ALL` (the order BQL's `SelectParser` reads them,
  case-insensitive) before splitting; a column named `top`, `topSpeed` or `distinctCount` is untouched.
- `ValidateQuery` already accepted these selects, so they did reach `SelectedSlots`; no recorded differential case used a modifier.
