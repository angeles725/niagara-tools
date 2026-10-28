### Added — report-module runs lint-module-map and prints a modularity row

- `report-module` runs `lint-module-map` once per module after `lint-structure`: its FAIL rows fail the report,
  its WARN rows (MM1 "no MODULE-MAP.md" on every legacy module) do not.
- One uncounted `(module)  INFO  modularity  mode=<m> concerns=<n> unlisted=<n> largest=<path> (<n> hand-written
  lines)` row per module: the module map mode, its concern rows, the part sources in no concern and the largest
  part class by hand-written lines. For a module with no map or a clean map the verdict and exit code are
  unchanged; an invalid or FAILing map (MM2-MM5 FAIL rows) now fails the report (exit 1), and an unreadable map or
  part directory is an env fault (exit 3). The row prints `?` for the counts and largest class in those cases.
