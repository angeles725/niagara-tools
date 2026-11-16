### Added — a size rule for the Go tree

- `go test ./...` now fails when a non-test Go file exceeds 200 lines, unless `go/internal/filesize/baseline.txt`
  lists it. The baseline holds the 98 files that were already larger, each with its line count: a listed file may
  shrink but not grow, no file may be added, and a stale line is itself a failure. Split a file by
  responsibility, then lower or delete its line.
