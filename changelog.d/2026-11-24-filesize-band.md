### Changed — the Go file-size gate is a WARN/FAIL band and runs in `ci-local.sh --changed` (issue #969)

- `go/internal/filesize`: a non-test Go file over 200 lines is a WARN (reported, never failing; `go test -v` prints it as a
  line starting `WARN `), over 300 lines it fails (`split it by responsibility`). The fail threshold moves from 200 to 300 lines: a
  250-line file that failed before now only warns. The shrink-only baseline is
  unchanged: a listed file may not grow and gets no WARN, and a listed file now at or under 200 lines still needs its
  baseline line deleted. Split by responsibility, never only to get under the number (`go/README.md`).
- `scripts/ci-local.sh --changed` (and the pre-push hook, which calls it): when a changed path is a `go/**/*.go`, it runs
  `go test -count=1 -v ./internal/filesize` from `go/`, prints the `WARN ` lines and exits 1 when the gate fails. Before,
  the gate ran only in CI's Go job, so a breach showed up after review and push. The gate reads the whole work tree, also
  with `--range`. Pinned by `tests/ci-local-filesize.bats`.
