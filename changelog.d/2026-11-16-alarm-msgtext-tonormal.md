### Added — alarm guidance: the return to normal rewrites msgText

- `build-n4-module-kit/types/logic.md` LC9: the toNormal transition rewrites the same alarm record's `msgText`
  with `toNormalText` (empty by default), and alarm texts resolve against the record; a value that must outlive
  the event lives in a persisted read-only slot, reset by acknowledge and quoted by both texts through an
  `alarmData` key. `mcp-n4-kit/METHODOLOGY.md` § 3 carries the same rule as one live-rules row.
