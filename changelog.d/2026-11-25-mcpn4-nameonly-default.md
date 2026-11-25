### Fixed

- mcp-n4: a frozen property set back to its declared default while a link targets it (for example a writable's `in16` fed by a schedule) is listed by the station by name only. `n4_read_slots` now lists its declared default (`default: true`) instead of type and value null, `n4_set_slot`'s read-back verifies it instead of failing with "is not a StatusBoolean/StatusNumeric", and `n4_rollback` no longer refuses that batch as "changed since". (#890)
