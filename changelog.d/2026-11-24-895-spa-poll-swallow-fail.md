### Changed — lint-spa-poll-no-recovery FAILs a swallowed poll failure (issue #895)

- `lint-spa-poll-no-recovery`: a poll fn whose catch path (a `catch (e) {..}` block or a `.catch(handler)`) neither
  throws nor returns, followed in the same block by an unconditional `lastOk*` clock stamp, is now a `FAIL` row (exit 1
  with or without `--strict`): the failed poll is stamped as a success, so the success-time watchdog never sees it. A
  catch that rethrows or returns, a stamp inside the try or the then-handler, and a stamp behind an
  `if`/`?:`/`&&`/`||` stay clean. The existing WARN rows are unchanged. Not seen: a tick that swallows in another
  file or function the poll calls; make that tick throw on failure. Doctrine in `types/frontend-standard.md` and
  `types/dashboard.md`.
