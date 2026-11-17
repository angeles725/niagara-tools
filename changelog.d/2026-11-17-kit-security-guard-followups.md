### Fixed — client guard at commit time and lint-servlet-write-guards: the verifier's follow-ups

- `.githooks/commit-msg` (via `.githooks/pre-commit --message`): comment lines are guarded too (`-m`, `-F` and
  `--cleanup=verbatim` keep them; a comment naming a client is now refused even where git would strip it), and a
  scissors-shaped line in a `-m`/`-F` message no longer hides the lines below it: below a `commit -v` cut only the
  lines of the diff git appends are dropped.
- `lint-servlet-write-guards`: FAIL `self-permission` also for `super.getPermissions(cx)` and for a local bound to
  the user or the servlet (`BUser caller = op.getUser(); caller.getPermissions(op)`, `self = this`);
  `no-target-permission` WARNs unless the check names the written receiver and its result is used, counts
  `invoke(…)` as a write, and accepts a `requireTargetWrite` declared in the file only when it checks its first
  parameter; `no-csrf` needs a `verifyCsrfToken(` call or `CsrfProtectedFilter`, not a variable named `csrfToken`.
- `types/servlet-write.md`: fact #3 names the cached-permissions early return of `BComponent.getPermissions`
  (proxies only).
