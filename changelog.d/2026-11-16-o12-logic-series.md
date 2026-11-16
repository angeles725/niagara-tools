### Added — logic series: check a pure control core against scripted inputs offline (O12)

- `build-n4-module-kit/toolbelt/java/n4kit/series` (Java 8 source, no dependency): `LogicSeries` steps a pure model
  through a CSV of timed inputs and expected outputs (`time_ms`, `in:<name>`, `out:<name>[~<tol>]`, `-` skips a
  check) through a small `SeriesModel` adapter, and stops at the first mismatch with its file, line, step, time,
  output, expected value, tolerance and actual value. A malformed series is refused before any step.
- `run-pure-test.sh` puts the kit's Java helper root on the javac source path, so a module test imports
  `n4kit.series.*` without copying it.
- `build-verify.md` § Logic series (recipe, results, how it complements the LP1 lab restart proof) with a worked
  example (`tests/fixtures/logic-series/thermo-rt`) pinned by `tests/logic-series.bats`.
