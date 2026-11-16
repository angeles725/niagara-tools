### Added — n4_add_alarm creates configured alarm extensions in one batch

- `n4_add_alarm` adds one alarm extension per point (1 to 50) under one confirmation token: an
  `alarm:AlarmSourceExt` whose `offnormalAlgorithm` and `faultAlgorithm` follow the kind (`out_of_range`,
  `numeric_change_of_state`, `boolean_change_of_state`, `boolean_command_failure`, `enum_change_of_state`,
  `enum_command_failure`, `string_change_of_value`, `status`), picked from the point type unless given. It sets the
  alarm class, the offnormal, fault and normal texts, the source name, the delays, the enabled transitions (hex bit
  mask), inhibit, the hyperlink, and the algorithm's limits (enabling each limit given), deadband, limit texts or
  alarm values; for `out_of_range`, `fault_high_limit`, `fault_low_limit` and `fault_deadband` set the same slots
  on its fault algorithm, beyond which the point goes to fault with `toFaultText`. The add holds the plain slots; set ops in the same batch swap both algorithms by type and set the
  rest by path, since a station refuses a component inside an add body.
- The dry run refuses a kind that does not fit a point (naming the kinds that do), an argument the kind does not
  take, an alarm class the station does not hold (one BQL read, the refusal lists the classes) and a second
  extension of the same kind on a point unless `allow_second`. The read-back reads every slot it set and lists any
  value the station stored differently under `coerced`. The README gains the alarm extension recipe: kinds, slots,
  encodings and the text syntax. Before, an alarm took a create, two algorithm swaps and one set per slot by hand.
  Bench retro 2026-11-15, row 10.
