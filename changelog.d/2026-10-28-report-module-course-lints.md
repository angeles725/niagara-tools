### Changed — report-module runs the course-coverage lints

- `report-module` runs `lint-charset-default` (WARN rows) and `lint-slot-default-type` (FAIL rows) on every artifact's
  `src/`, after `lint-bformat-call`.
