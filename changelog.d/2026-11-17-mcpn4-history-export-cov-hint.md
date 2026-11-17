### Added — mcp-n4: `n4_export_history`, histories to one CSV for client reports (station-audits S2)

- `n4_export_history(history_ids, period | start/end, max_rows=100000)` writes 1 to 20 histories to ONE CSV
  file, `<state-dir>/exports/<station>-<UTC time>.csv` (directory 0700, file 0600, never overwritten; the
  server names it, the caller never passes a path). Columns `history_id,timestamp,value,status`, oldest first,
  with machine-encoded time and value (ISO-8601 with the station's offset, `21.0` not `21,0`). The reply is
  `{path, rows_total, per_history [{history_id, rows, truncated}], bytes}`. Ids are checked like
  `n4_read_history`'s before any request; the station is only read.
