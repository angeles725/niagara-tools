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
