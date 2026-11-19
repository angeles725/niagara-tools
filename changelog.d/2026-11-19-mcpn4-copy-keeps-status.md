### Fixed — mcp-n4: `n4_copy_component` keeps the status of Status values

- A copy carries the status of a Status value (StatusBoolean, StatusNumeric, StatusEnum, StatusString)
  whole with its value whenever it is off its declared default, as `n4_set_slot` writes a Status slot.
  The status child is declared readonly, so it was left out and every copy fell back to `{ok}`: a schedule
  whose `in` and `defaultOutput` held `{null}` was copied with a live `out` (false / 0.0) that would have
  switched units OFF and set setpoints to 0 once linked (delta P2, live near-miss 2026-11-19).
- The read-back compares that status on each copy: a copy whose status differs from the source is a
  `mismatch`, never `verified`. `n4_replicate_points` shares the same capture and gains both.
- A status at its default (`{ok}`) is still left out, so a plain copy plans the same ops as before.
