### Fixed — run-3 review fix-forward: env-fault verdict, repeated ledger header, KCRD2 teeth

- `report-module`: a member env fault (an `ERROR` row, exit 3) with no FAIL now reads verdict `INCOMPLETE`
  instead of `CLEAN`; the exit stays 3 and a FAIL still reads `ISSUES`.
- `ledger-check`: a header row repeated in the Versions section (for example after a blank line) is no longer
  checked as a data row (it FAILed LC5/LC6/LC7 on the column names).
- `tests/kit-cite-resolve-deferred.bats` KCRD2 asserts exit 0 and the resolved `[deferred]` pointer row; it
  passed before on a missing resolver.
