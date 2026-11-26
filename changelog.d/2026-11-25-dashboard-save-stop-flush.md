### Fixed

- `types/dashboard.md` § Write path (#1118): the station-save exemplar no longer drops a pending save in
  `stopped()`. With the 5 s settle and one save per 60 s, a write made 5-60 s before a restart or a servlet stop
  was never saved and the equipment went back to the old value. `stopped()` now reads `!saveTicket.isExpired()`
  before `cancel()`, then calls `station.save()` once (`BStation.doSave` → `Station.saveAsync`, it does not
  block) inside a try/catch that logs and never throws.
- `lint-dashboard-write-save` (#1118): two new WARN rows (exit 0), waived like the FAIL by a non-empty
  `persistence:` line in `MODULE-MAP.md`. `stop-drops-save`: a `stopped()` that cancels the save ticket without
  a flush. `write-path-no-save`: a `-ux` handler that writes with the request `Context` and has no path to a
  save, directly or through a helper, in a module that saves. The FAIL on a module with no save is unchanged.
