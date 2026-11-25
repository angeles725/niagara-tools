### Changed

- `toolbelt/report-module.sh` (Go: `go/internal/reportmodule`): the ERROR row of a lint that could not run (exit 3)
  now carries the lint's own first reason line, e.g. `ERROR  lint-vacuous-tests  env fault (exit 3):
  lint-vacuous-tests: go not found on PATH`, so the punch-list names the missing tool (it was the bare
  `env fault (exit 3)` for the common lints; the advisory lints already did this). The verdict is unchanged: a
  member env fault with no FAIL reads `INCOMPLETE` (exit 3), never `CLEAN`, and the other members' rows stay a
  usable result. A member's own `ERROR  <check>  ...` row is not used as the reason. (retro
  2026-11-02-client-e-dashboard-live-check Δ31, #986)
- `n4_list_links` (`go/internal/mcpn4/readtools/links.go`): the `depth` semantics are documented and pinned by a
  test: ord is level 0 and always scanned, so `depth` 1 is ord and its direct children; `depth` 0 stays refused by
  the argument validation (`argument depth must be >= 1`). The tool-schema description of `depth` now says so. (retro 2026-11-02-dashboard-live-check-4-15 Δ17, #985)
