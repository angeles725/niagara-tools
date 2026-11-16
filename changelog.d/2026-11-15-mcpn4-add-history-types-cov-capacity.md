### Changed — n4_add_history picks the extension from the point type and refuses a repeated history id

- `n4_add_history` reads each point's type from the station's type registry and creates the extension that fits it:
  a numeric, boolean, enum or string control point gets `NumericInterval`, `BooleanInterval`, `EnumInterval` or
  `StringInterval`, named after that type unless `ext_name` is given. An `ext_type` that does not fit the point, or a
  point that is not one of those, is refused at the dry run naming the type that fits. Before, a boolean point got a
  `NumericInterval` that failed only at execute.
- The dry run predicts each history id (`/<station>/<history name>`) and refuses one that repeats in the batch or that
  an extension on the station already holds, naming that extension (one BQL read of every `history:HistoryExt`).
  Before, the extension was created in fault "Duplicate history id" and reported `verified`.
- The read-back compares the history id with the predicted one and the fault cause with an empty one, so an extension
  in fault reads `mismatch`. The capacity note computes the duration from the requested interval (about 50 s at
  100 ms), where it always said "about 41 h at 5 min". Bench retro 2026-11-15, row 7.

### Added — n4_add_history change-of-value extensions and a capacity argument

- `mode=cov` adds a change-of-value extension (`history:NumericCovHistoryExt`, `BooleanCov`, `EnumCov`, `StringCov`)
  per point in the same one-token batch: it records each change of the point value, has no interval, and refuses
  `interval_ms`; `interval_ms` is now required only for the default `mode=interval`. Before, a COV extension took 3
  calls per point by hand. Bench retro 2026-11-15, row 8.
- `capacity` (records, 1 to 2147483647, or `"unlimited"`) is set by a set op on
  `<ext_name>/historyConfig/capacity` with a `history:Capacity` value `1:<records>` (or `0:0`), right after each add in
  the same batch, since `historyConfig` is a component a station refuses inside an add body. The read-back checks it
  and the note says how long it lasts at the interval (20160 records at 5 min: about 70 d). Before, capacity stayed at
  500 records, about 41.7 h at 5 min, too short for a dashboard's 7 d / 30 d views. Client-A history rollout retro,
  row 2.
