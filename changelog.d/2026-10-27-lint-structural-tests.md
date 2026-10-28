### Added — lint-structural-tests

- New `toolbelt/lint-structural-tests.sh <module-root>...` (G34): WARNs (FAIL under `--strict`) a `srcTest` test
  that reads a `.java` source as text (`Files.readAllBytes`/`readAllLines`/`readString`/`lines`, `Paths.get`/`Path.of`
  with a `.java` literal) when its part's `srcTest` has no `BTestNg`/`BTestNgStation` class. Not yet run by
  `report-module`.
