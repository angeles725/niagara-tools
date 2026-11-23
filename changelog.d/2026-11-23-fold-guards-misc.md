### Changed — lint-servlet-write-guards scope, interrupt cleanup, preflight delivery row, contrast rules

- `lint-servlet-write-guards` reads the servlet's dispatch class: a servlet that names a type of its own directory
  that verifies the CSRF token (types/dashboard.md DUX1) no longer gets the `no-csrf` WARN. It also narrows what a
  write is: a one-argument `set(v)`, a numeric first argument (JDBC `setInt(1, v)`, `List.set(0, x)`) and a variable
  declared only as a `PreparedStatement`, `List`, `Atomic*`, `Calendar` or `ThreadLocal` are no longer
  `no-target-permission` or `no-csrf` WARNs.
- `build.sh`: an interrupt ends what gradle itself started (a single-use daemon forked by the wrapper's client)
  after gradle ends and before `modules/` is put back, so it cannot install a jar after the restore.
- `preflight.sh`: a PASS `delivery` row in a kit checkout names what pre-push asks of a kit change (changelog
  fragment, ledger fragment, pending retro, or the trailers).
- `types/frontend-standard.md` § 10: contrast is checked per token pair (WCAG AA check beside the Node tests), a
  theme change re-maps every colour token, and faded state is a muted colour, never `opacity` on text.
