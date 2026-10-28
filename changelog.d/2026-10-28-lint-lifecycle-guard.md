### Added — lint-lifecycle-guard

- New `toolbelt/lint-lifecycle-guard.sh [--strict] <src-root>` (WARN, advisory): LG1 a `changed`/`added`/`removed`
  callback whose first statement after `super` is not an `isRunning()` (or `Sys.atSteadyState()`) guard; LG2 a
  `stationStarted()` doing startup work in a class with no `started()`, `serviceStarted()` or `descendantsStarted()`.
  Not yet run by `report-module`.
