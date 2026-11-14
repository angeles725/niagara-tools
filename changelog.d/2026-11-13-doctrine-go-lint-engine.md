### Changed — lint doctrine points at the Go engine

- `BUILD-LOOP.md` § 5 (shared helpers, no fork per input line, fail-closed input listing) and `skill/SKILL.md` no
  longer tell an author to use the removed `toolbelt/lib/awk-lint.sh` / `lib/scan-files.sh`: a new lint is a Go
  rule on `go/internal/lint` (`lint.Walk`, the shared runner). The owed NUL-separated listing is closed by the
  removal of the bash lister.
