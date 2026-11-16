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
