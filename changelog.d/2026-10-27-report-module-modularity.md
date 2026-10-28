### Added — report-module runs lint-module-map and prints a modularity row

- `report-module` runs `lint-module-map` once per module after `lint-structure`: its FAIL rows fail the report,
  its WARN rows (MM1 "no MODULE-MAP.md" on every legacy module) do not.
- One uncounted `(module)  INFO  modularity  mode=<m> concerns=<n> unlisted=<n> largest=<path> (<n> hand-written
  lines)` row per module: the module map mode, its concern rows, the part sources in no concern and the largest
  part class by hand-written lines. The verdict and exit code are unchanged.
