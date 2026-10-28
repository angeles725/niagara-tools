### Added — report-module runs the run-3 module lints

- `report-module` now runs `lint-permissions-xml` (its FAIL rows fail the report; PASS `clean` otherwise),
  `lint-structural-tests`, `lint-logger-name` and `lint-adapter-cfg-default` (WARN rows relayed) once per module. A
  module without their sources gets a SKIP row instead of an env fault.
