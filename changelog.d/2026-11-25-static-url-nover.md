### Changed

- `toolbelt/rc-scan.sh` (Go: `go/internal/rcscan`): new advisory check `static-url-nover` (WARN under every mode,
  `--strict` included). An `rc/` HTML, CSS or JS reference to a static asset with no `?v=` is a row: a `/module/…`
  or `/file/…` path, or a relative `img/`, `css/` or `js/` path, written as a `src=`/`href=` attribute, a CSS
  `url(…)` or `@import`, the string argument of `fetch(`, `new URL(` or `import`, a `.src =` / `.href =`
  assignment, or a `src=`/`href=`/`url(…)` inside a JS string. A browser keeps such an asset after a module update
  until its cache entry expires; `lint-ux-version` (#1007) reads only the dashboard entry pages. Not references:
  `require(`/`define(` strings (module ids), absolute URLs, `data:`, anchors, comments, templates, a string that
  goes on (`"img/" + n`), `rc/vendor`, `rc/ext` and `*.min.js`. A `?v=` (or `ver`, `version`) at any position of the
  query counts. Waiver: `rc-scan: allow static-url-nover <reason>` in a comment of the file's syntax; a marker with
  no reason waives nothing.
- `toolbelt/lint-servlet.sh` `cache-nofinger` also reads the `rc/` pages (`go/internal/lint/rules/servlet`
  `rcrefs.go`, over `rcscan.Unfingerprinted`): the nearest `rc/` or `src/rc/` at or above the servlet is read for
  references to the servlet's own static paths (relative `img/`, `css/`, `js/`, or under `/<getServletName()>/`)
  with no `?v=`. An unfingerprinted `max-age` row names the first one; a `max-age` gated on a `?v=` request, clean
  before, WARNs when such a reference exists (its requests never take the cached branch). A module with no `rc/`
  above the servlet, and one whose references all carry `?v=`, give the rows they gave before.
- `types/frontend-standard.md` § Enforcement and `types/data-access.md` document both; goldens
  (`truth-static-url-nover-*`, `truth-cache-rcrefs-*`) and bats `RC38`–`RC41`, `LSV8`–`LSV10` pin them; the kit's
  MinimalDash fixture stays clean under both. Issue #1035.
- Review follow-up (R3): the `cache-nofinger` `rc/` climb never reads another module's `rc/`. It ends at a directory
  with a gradle script (`build.gradle`, `build.gradle.kts`, `settings.gradle`, `settings.gradle.kts`), a
  `module-include.xml`, a `<part>.gradle(.kts)` or a name ending `-rt`/`-ux`, after looking at that directory's own
  `rc/` and `src/rc/`; the 10-level limit stays. An `rc/` file it cannot read is no longer silence: the servlet
  file gets the engine's env error (`cannot scan <file>: lint-servlet: rc/ pages could not be read: <why>`, exit 3,
  its other rows still printed), as an unreadable source file does. `rcscan.Unfingerprinted` reports a URL once per
  file and line, as the rc-scan rows do. Changed behaviour: an empty `?v=` (no value) is no fingerprint in both
  readers (it was one); `TestStaticNoVersion` pinned the old reading and now pins this one. Unit tests
  (`TestRCDirAbove`, `TestRCReadError`, `TestUnfingerprintedDedup`), goldens `truth-cache-rcrefs-4`/`-5` and bats
  `RC42`, `LSV11`, `LSV12` pin them.
