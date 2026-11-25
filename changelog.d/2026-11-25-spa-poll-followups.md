### Changed

- `lint-spa-poll-no-recovery` (`go/internal/lint/rules/spapoll`, `handler.go`): the swallowed-failure FAIL reads the
  catch handler at its own level. `.catch(e => Promise.reject(e))` and `.catch(e => { return Promise.reject(e) })`
  rethrow, so they are no longer a false FAIL (which hard-fails builds). A `throw` or `return` inside a callback the
  handler passes on (`list.forEach(i => { return; })`, `later(function () { throw e; })`) does not exit the catch, so
  that swallow is now a FAIL (it was missed). A conditional exit in the handler (`if (x) { return; }`) still counts as
  an exit; a named handler (`.catch(onError)`) shows none; a comment or a string is not code. Issue #966 item 3.
- Pins for the rest of issue #966: goldens `truth-swallow-clean-1` (a stamp behind `if`/`else`/`?:`/`&&`/`||` or at
  another brace depth, and the rethrow shapes) and `truth-swallow-fail-2`, bats `SPR18`/`SPR19`, unit tests
  `TestExitsCatch` and a `report-module.bats` case `RM17b` (fixture `tests/fixtures/lint-spa-poll-report`) that a FAIL
  row of this lint shows in `report-module` and counts. The member keeps no verdict row (as `lint-delays`).
- Readability: the doctrine for the FAIL lives in `types/dashboard.md` § Poll loop reliability; the
  `types/frontend-standard.md` paragraph, the wrapper header and the package comment point at it.
