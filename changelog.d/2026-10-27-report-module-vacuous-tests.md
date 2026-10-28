### Added — report-module runs lint-vacuous-tests

- `report-module.sh` runs `lint-vacuous-tests` once per module on the module root, after the driver lints: its
  FAIL rows (a srcTest `@Test` that cannot fail or never runs) fail the report; a module with no `srcTest/`
  `*.java` gets a `SKIP  lint-vacuous-tests  no *.java under a srcTest/` row, never an env fault.
