### Changed — the Go file-size gate is research-sdd's debt list (issue #969)

- `go/internal/filesize`: replaces the WARN 201-300 / FAIL over 300 band and the shrink-only `baseline.txt` with the
  debt list `modularity-debt.txt` (`<path> <lines> <legacy|accepted> <reason of at least 10 characters>`, sorted by path).
  A production Go file (not a test, `testdata/`, `bin/`, an ignored path or a `// Code generated ... DO NOT EDIT.` file)
  over 200 lines needs a line. A count of 400 or less is a ceiling: the file may shrink freely and may grow up to 400
  only with its line raised in the same change; a count over 400 is exact (it may not grow, its line is lowered when it
  shrinks); an `accepted` line is never above 400; a listed file at 200 lines or less must delete its line. An unlisted
  file over 200 lines fails with the exact line to add (`<path> <n> accepted <reason>`). The 68 baseline files are the
  `legacy` lines (reason `recorded when the ratchet was created (#969)`, counts as they stand today). There is no WARN
  band any more. `NT_FILESIZE_UPDATE=1 go test -run TestTheGoTreeKeepsItsFilesSmall ./internal/filesize` lowers counts
  and deletes the lines of files that are gone or at 200 lines or less (`go/README.md`).
- `scripts/ci-local.sh --changed` still runs the gate for a changed `go/**/*.go`; it now prints the failing test's
  messages (the line to add, raise, lower or delete) and no WARN lines. Pinned by `tests/ci-local-filesize.bats`.
