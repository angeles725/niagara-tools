### Fixed — mcp-n4: history summaries leave {null}-status records out of min/max (retro row M16)

- The summary of `n4_read_history` `summary: true` and of `n4_event_report` counted the `0.0 {null}` records
  a new COV history writes at start in min and max, so min read `0.0` (live on a lab history). A record whose
  status has the `null` flag holds no value (`BStatusValue` renders it `-`), so it is now counted in `count`
  and `status_counts` but left out of `min`, `max`, `first` and `last`, as `nan`/`inf` records already were
  for min and max. `excluded_from_extremes {null, nan_inf}` counts both causes, a `note` says so, and
  `first_record` / `last_record` keep the raw ends when they differ. Other status flags keep their value.
