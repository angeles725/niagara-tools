### Added — lint-servlet handler-size

- `lint-servlet` reports a method of a servlet file that spans more than 80 lines (its name line to its closing
  brace) as `handler-size`: WARN with no module map or a `modularity: legacy` map (FAIL under `--strict`), FAIL in
  a `modularity: strict` module. The servlet stays a router; each route group's handler is its own
  `ux/route/<Route>Handler` class (`types/dashboard.md` DUX1).
