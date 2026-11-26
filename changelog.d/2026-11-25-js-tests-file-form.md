### Added

- `report-module` row `js-tests` (issue #1124): a `-ux` part with a `srcTest/` has its JS tests run as
  `node --test <file>...` in the part's directory, the files (`srcTest/**/*.test.js` and `*.test.cjs`, in byte
  order, none under `node_modules/` or a dot directory) listed by the tool itself, because `node --test <dir>` fails
  on Node 22 (it loads the directory as a module) and only the file form runs. PASS `node --test: N files passed`;
  FAIL on a non-zero exit with the tail of node's output, and on a run past `KIT_JS_TESTS_TIMEOUT` seconds (default
  600); SKIP naming the reason when node is not on PATH (`KIT_NODE` names another) or `srcTest/` holds no such file.
  A part with no `srcTest/` prints no row. A broken JS suite is now a FAIL in the punch-list, not something each
  session has to remember. Tests: `go/internal/reportmodule/jstests_test.go`.
- The file form is written in `types/frontend-standard.md` § 11 and `types/dashboard.md`
  (`node --test <Module>-ux/srcTest/rc/*.test.js`), in `BUILD-LOOP.md`, and in the `scaffold-module.sh --type
  dashboard` help and the `report-module.sh` header.
