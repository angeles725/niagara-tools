### Added — mcp-n4: behaviour snapshot and diff around a deploy (O6)
- `n4_behavior_snapshot(name, scope)` stores the out value and status bits of every control point below a scope in `<state-dir>/snapshots/<name>.json` (0600, never overwritten), with one BQL read.
- `n4_behavior_diff(before, after?, tolerance, limit)` compares it with a second snapshot or the live station: `status_changed` (flags by name), `value_changed` (numeric tolerance reads a locale decimal comma and a trailing unit), `missing`, `new`, `summary`. A truncated side leaves missing/new `not_checked`, never zero.
