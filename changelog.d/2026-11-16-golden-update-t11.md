### Added — `-update` re-records the golden cases of a Go port

- The shared golden runner (`go/internal/lint/rules/golden`) takes `go test ./internal/<pkg> -update`: a case
  whose exit code, stdout or stderr differs gets its `want` and `stderr` sections rewritten instead of failing,
  and every other byte of the case file is kept. The case directory is resolved to an absolute path first, so
  `CHDIR=1` cases (report-module) are rewritten in place. Without the flag nothing is written. Documented in
  `go/internal/lint/rules/golden/record/README.md`.
