### Removed — dead bash lint libs

- `toolbelt/lib/awk-lint.sh` and `toolbelt/lib/scan-files.sh` are deleted with their unit tests
  (`tests/awk-lint.bats`, `tests/scan-files.bats`, `tests/lint-empty-root.bats` ER3/ER4): every lint that sourced
  them is an `n4lint` wrapper, so no script called them. The walk, the argument modes and the empty-root row live
  in `go/internal/lint`.
