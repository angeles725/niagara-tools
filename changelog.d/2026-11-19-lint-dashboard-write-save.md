### Added — lint-dashboard-write-save; lint-delays reads a Math.max floor in either argument

- `build-n4-module-kit/toolbelt/lint-dashboard-write-save.sh <module-root>` (Go `go/internal/lint/rules/dashboardsave`,
  a report-module member run once per module): FAILs every write
  line of a `-ux` part (`.set(`, `.add(`, `.remove(`, `.invoke(` whose last argument is the request
  `Context`/`WebOp`, and `addSpecialEvent(`) when nothing in the module requests a station save
  (`BStation.save`, `Sys.getStation().save()`, `Station.save*(`) and `MODULE-MAP.md` declares no
  `persistence:` line. Issue #821: a dashboard lost its writes on restart and a driver re-sent a stale value.
  A module with no `-ux` part is a SKIP row in the report.
- `build-n4-module-kit/types/dashboard.md` § Write path: the debounced station-save exemplar and the
  `persistence:` declaration; BUILD-LOOP §6.b item 6 links the rule.
- `lint-delays`: `Math.max(1L, x)` is a floor like `Math.max(x, 1L)` (inline, local binding and helper body);
  it used to FAIL `zero-floor cannot determine Math.max floor`.
