### Changed — VERSION-LEDGER records the Niagara version of each build; preflight `ledger-niagara` (#1026)

- `templates/VERSION-LEDGER.md`: an optional `Niagara` column after `Version` — the version of the `niagara_home`
  the build used (e.g. `4.14.0.162`, the `vendorVersion` of its `modules/baja.jar`; `—` when unknown). The scaffold
  emits the new header (it drops the sample row, as before).
- `toolbelt/ledger-check.sh` (Go: `go/internal/ledgercheck`): reads a ledger with or without the column (the header
  of either shape, rows of its cell count); new check LC10 FAILs a Niagara cell that is not empty, `—` or a version
  `X.Y.Z[.W]`. The reader of the Versions table moved to `table.go`, the column's code is in `niagara.go`.
- `toolbelt/preflight.sh` (Go: `go/internal/preflight/ledgerniagara.go`): WARN `ledger-niagara` when the latest row
  of `<gradle-root>/odd/VERSION-LEDGER.md` (else the parent's) names a Niagara version other than `<niagara_home>`'s
  (`modules/baja.jar` vendorVersion, else the last `X.Y.Z` of its path; equal up to the shorter's length). No
  row for a ledger without the column or an empty, `—` or malformed cell; the exit code is unchanged. This is the
  ledger half left open by `sibling-home` (#1024).
- Tests: Go cases for LC10, both header shapes and `LatestRow`; four preflight goldens and unit cases (another
  version, the same, empty, old shape, parent ledger, `baja.jar` over the directory name); bats `LCB5` and two
  `ledger-niagara` cases.
