### Added — mcp-n4: whole-range history summary and alarm state counts (retro row live M2)

- `n4_read_history` `summary: true` returns count, min and max with their ISO timestamps, first, last and
  per-status counts over the whole range, whatever `limit` cut the rows, from one read of every record with
  machine-encoded time and value; a range above `summary_max_rows` (default 100000) says so and adds the
  station's whole-range `select min(value), max(value)`. `start`/`end` also take a span back from now (`30m`,
  `24h`, `7d`), and a bare history name is a history of the connected station.
- `n4_read_alarms` returns `state_counts` (every matching record per source and ack state), `timestamp_iso`,
  `normal_time` and `ack_time` per row, a `source_glob` like-pattern filter and relative `since`/`until`.
  "A cuanto llego" was six manual steps in the 2026-11-16 client A over-amps retro; it is now one call.
