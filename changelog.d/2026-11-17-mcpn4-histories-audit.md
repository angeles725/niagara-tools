### Added — mcp-n4: `n4_histories_audit`, what is wrong with a station's histories (station-audits S1)

- `n4_histories_audit(base, stale_factor=3, cov_max_age=24h, capacity_warn=0.9, max_histories=500)` lists, in one
  read-only call, the control points with no history extension (`no_history`), the disabled history extensions
  (`disabled`), the stale histories (`stale`: last record older than `stale_factor` x the collection interval, or than
  `cov_max_age` for COV, or never recorded), the record-capped histories close to capacity (`near_capacity`, warning
  for full policy `stop`, info for `roll`) and the history ids several extensions share (`colliding_ids`). Each list
  keeps 100 items; `counts` holds the totals. Two fixed BQL reads plus one `select count(*)` per distinct
  record-capped id that has recorded (`count_reads`); a cut read sets `truncated` and leaves `no_history` unchecked.
