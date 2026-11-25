### Added — preflight `sibling-home`: a sibling module root pinning another niagara_home (F24)

- `toolbelt/preflight.sh` (Go: `go/internal/preflight/siblinghome.go`): WARN `sibling-home` when an immediate
  sibling of the gradle root's directory that is a module root (`settings.gradle.kts` or `settings.gradle`) pins a
  different `niagara_home` than the `<niagara_home>` argument, in its `gradle.properties` (the only form the kit's
  templates write; a comment, a blank value, a non-module directory and the gradle root through a symlink are
  skipped; a `C:\\…` pin is read as `/mnt/c/…`). The row names the sibling and both values; the exit code is
  unchanged. Evidence: a client repository held three roots pinned to 4.13, 4.14 and 4.15 and a module was built
  against 4.13 while the JACE ran 4.15 (dev-course-material-vs-kit 2026-11-24, F24).
- Not done: the second half (the `<niagara_home>` version against the deploy target in `VERSION-LEDGER.md`) —
  `templates/VERSION-LEDGER.md` has no Niagara deploy-target field (its rows are module versions), so there is
  nothing to compare; add the field first.
- `BUILD-LOOP.md` §0: one line on the pin check.
- The pin and the target are compared as resolved paths (symlinks resolved when the path exists, the cleaned string
  otherwise; a relative pin is read against its sibling root), so a symlinked install or a relative pin naming the
  same directory is no mismatch. Goldens pin the symlinked install, the symlinked gradle root, two mismatching
  siblings in sorted order and a drive-letter pin.
