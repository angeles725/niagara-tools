### Added — mcp-n4: `n4_export_history`, histories to one CSV for client reports (station-audits S2)

- `n4_export_history(history_ids, period | start/end, max_rows=100000)` writes 1 to 20 histories to ONE CSV
  file, `<state-dir>/exports/<station>-<UTC time>.csv` (directory 0700, file 0600, never overwritten; the
  server names it, the caller never passes a path). Columns `history_id,timestamp,value,status`, oldest first,
  with machine-encoded time and value (ISO-8601 with the station's offset, `21.0` not `21,0`). The reply is
  `{path, rows_total, per_history [{history_id, rows, truncated}], bytes}`. Ids are checked like
  `n4_read_history`'s before any request; the station is only read.

### Added — mcp-n4: `n4_read_history` `cov_hint`, a COV tolerance suggested from real data (station-audits S3)

- `cov_hint: true` suggests a COV `changeTolerance` from the same machine-encoded read the summary uses
  (both together still read once); report only, nothing is written. It returns the |delta| distribution of
  successive records (`median`, `p90`, `p99`, `noise_floor`), the tolerances tried and
  `suggested_tolerance`: the smallest 1-2-5 value at or above the median non-zero |delta| for which a
  simulated COV (`|value - last kept| > tolerance`) keeps at most half the records, with `reduction_estimate`.
  A non-numeric history is `cov_hint: null` with `cov_hint_note`. The reply without `cov_hint` is unchanged.
