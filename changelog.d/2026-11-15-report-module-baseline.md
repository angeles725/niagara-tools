### Added — report-module separates NEW FAIL rows from a committed baseline

- `report-module.sh` reads `<module-root>/.report-baseline` when the module has one: the FAIL rows are printed in
  two groups, NEW and BASE, the summary reads `f FAIL (n new, b base in <file>)`, and the verdict and the exit
  code read the NEW rows only. A module whose FAIL rows were all there before the change exits 0, so `build.sh`
  no longer reports it as "not hand-off-ready". Before, a client module with 37 FAIL rows, none of them from the
  change, ended every build at exit 50.
- `--write-baseline` writes the file from the run's FAIL rows (one `<artifact>  <check>  <detail>` key per line,
  line numbers and counts masked, sorted); `--no-baseline` ignores it for one run.
- The floor is never baselined: verify-module gate rows and schema-risk rows stay NEW and are left out of a
  written file, and a SKIPped floor check still reads INCOMPLETE. A baseline line no FAIL row matches is a WARN
  `stale baseline line` row.
- Without the file and without the options the report is unchanged; only the usage text (exit 2) is longer.
