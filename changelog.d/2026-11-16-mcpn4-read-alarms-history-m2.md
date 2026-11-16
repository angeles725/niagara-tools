### Added — mcp-n4: whole-range history summary and alarm state counts (retro row live M2)

- `n4_read_history` `summary: true` returns count, min and max with their ISO timestamps, first, last and
  per-status counts over the whole range, whatever `limit` cut the rows, from one read of every record with
  machine-encoded time and value; a range above `summary_max_rows` (default 100000) says so and adds the
  station's whole-range `select min(value), max(value)`. `start`/`end` also take a span back from now (`30m`,
  `24h`, `7d`), and a bare history name is a history of the connected station.
- `n4_read_alarms` `state_counts: true` returns every matching record per source and ack state; rows add `timestamp_iso`,
  `normal_time` and `ack_time`; a `source_glob` like-pattern filter and relative `since`/`until`.
  "A cuanto llego" was six manual steps in the 2026-11-16 client A over-amps retro; it is now one call.

### Security — mcp-n4: BQL values are checked as BQL decodes them

- Niagara decodes `$xx` / `$uxxxx` escapes inside a BQL body (`BqlQuery.make`), so `n4_read_alarms`
  `source` and `alarm_class` holding `$27` could close the quoted literal (live: `alarm_class` matched 27 of 27
  alarm records), and `n4_bql_query` / `n4_neql_query` selects could pass a local check through an escape
  (`select uuid$20from$20control:NumericPoint` on `alarm:` passed the no-from rule). `bql.DecodeForBql` (one pass
  of `EscUtil.unescape`, a malformed escape refused) now feeds every one of those checks. ORD paths and history
  ids are matched escaped by the station and are never decoded. `source_glob` takes no `$`.
