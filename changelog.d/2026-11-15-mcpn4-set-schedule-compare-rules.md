### Fixed — mcp-n4: n4_set_schedule and n4_rollback compare event and date rules
- A wrong-type or null value in a special event's ranges is named as a fault instead of an internal error, with every other fault kept.
- The write read-back compares each special event's rule, not only its name and ranges.
- `n4_rollback` compares the specialEvents list (names, order, rules, ranges) and each calendar date part (`yearSchedule`, `monthSchedule`, ...) by content: a list changed since the batch is not overwritten, and a list or date the station did not restore reads as a mismatch.
- `remove_special_event` and `remove_date` refuse a name given twice; a `reference` event must name an existing `schedule:CalendarSchedule`.
