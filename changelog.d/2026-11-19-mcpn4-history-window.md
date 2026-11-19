### Added — mcp-n4: `n4_read_history` window, status filter and history type; `n4_histories_audit` flat histories

- `n4_read_history` takes `from` / `to` (the same window as `start` / `end`, one pair or the other) and
  `status` (`not-ok`, `ok`, or one flag such as `fault`), filtered by the station so `limit`, `truncated` and
  `total_records` count only the matching records: the `{fault}` / `{ok}` pairs a comm drop leaves in a COV
  history are one call, not a hand scan of timestamps (delta M2).
- `history_type=true` states whether the history records on change of value (`cov`) or every interval (with
  its period, e.g. `5m`), from the collection interval of the extension that records into it.
- `n4_histories_audit` `flat=true` lists the histories holding one value for more than `flat_days` days
  (default 2) as a possible device placeholder, with the point and the value: a device that sends 0.0 for
  "no sensor" no longer hides behind `{ok}` records (delta M3).
- Without the new arguments both tools send the same requests and answer the same reply as before.
