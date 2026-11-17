### Added — mcp-n4: `n4_read_audit`, who changed what from the station's AuditHistory (station-audits S7)

- `n4_read_audit(target | target_glob, slot, user, operation, period | start/end, limit=100)` reads
  `history:/<station>/AuditHistory` and returns its records newest first, each
  `{timestamp, operation, target, slot, old_value, value, user}` (timestamp ISO-8601 with the station's offset), with
  `truncated` and `total_records`. `target` matches a component and everything below it; `slot` and `user` are exact;
  `operation` is one of the AuditEvent operations the audit history keeps. Every filter value is checked on its
  decoded form before any request. Read-only.
