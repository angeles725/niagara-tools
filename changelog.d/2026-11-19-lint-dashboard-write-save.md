### Added — lint-dashboard-write-save; lint-delays reads a Math.max floor in either argument

- `go/internal/lint/rules/dashboardsave` (rule `lint-dashboard-write-save`, over a module root): FAILs every write
  line of a `-ux` part (`.set(`, `.add(`, `.remove(`, `.invoke(` whose last argument is the request
  `Context`/`WebOp`, and `addSpecialEvent(`) when nothing in the module requests a station save
  (`BStation.save`, `Sys.getStation().save()`, `Station.save*(`) and `MODULE-MAP.md` declares no
  `persistence:` line. Issue #821: a dashboard lost its writes on restart and a driver re-sent a stale value.
  Not registered in `n4lint` yet: the registration lands with its report-module member (retro Δ4).
- `build-n4-module-kit/types/dashboard.md` § Write path: the debounced station-save exemplar and the
  `persistence:` declaration; BUILD-LOOP §6.b item 6 links the rule.
- `lint-delays`: `Math.max(1L, x)` is a floor like `Math.max(x, 1L)` (inline, local binding and helper body);
  it used to FAIL `zero-floor cannot determine Math.max floor`.
