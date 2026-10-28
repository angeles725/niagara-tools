### Fixed — report-module modularity row and the strict size relay

- The `(module) INFO modularity` row reads the mode the per-file rules apply (`lint.ModuleMode`): a strict header
  over a broken table reads `mode=strict`, not `none`. Its largest class is taken over the same files
  `lint-module-map` walks (a symlinked part included), and with a map at the root a module with no part, or an
  unreadable map or part, prints `?` instead of counts of a partial walk.
- Only `lint-size` (the mode-aware advisory member) has an exit 1 with FAIL rows relayed as FAIL; any other
  advisory member exiting 1 is an env fault (ERROR, exit 3) as before 2e261afb.
