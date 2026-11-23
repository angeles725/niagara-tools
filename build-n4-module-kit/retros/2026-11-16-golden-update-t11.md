<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · golden-update-t11

**Session**: row T11 of the 2026-11-15 coordination-stretch retro (kit retro ntpl-lint-walk-k-rows Δ6): a re-record mode for the shared golden runner.
**Delta count**: 2

## What happened
`golden.Runner` had no re-record mode: the 116 report-module goldens of niagara-tools #697 were re-recorded with a
temporary dump hook removed before commit, and a `CHDIR=1` case needed an absolute testdata path in that hook. The
runner (442 lines, on the size baseline) was split by responsibility first, then `-update` landed in its own file.

## Evidence
- Refactor: every code line of the old `runner.go` is in `runner.go`, `casesections.go`, `replay.go` or `tree.go`
  (sorted line sets compared); the package tests and the whole Go suite pass. `[ev: go/internal/lint/rules/golden/replay.go]`
- RED before GREEN: `TestUpdateRewritesTheCase` and `TestRewriteSpellsSections` failed before `-update`; with the
  case directory left relative, the `CHDIR=1` case fails with "stat cases/c.txt: no such file or directory"; with
  the flag check dropped, `TestNoUpdateWritesNothing` fails. `[ev: go/internal/lint/rules/golden/update_test.go]`
- `go test ./internal/reportmodule -update` leaves every report-module case byte-identical (they all pass).
  `[ev: go/internal/reportmodule/testdata/README.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. `golden.Runner` split by responsibility (`runner.go`, `casesections.go`, `replay.go`, `tree.go`); `runner.go` leaves the size baseline | Go `internal/lint/rules/golden` · `internal/filesize/baseline.txt` | `[ev: go/internal/lint/rules/golden/tree.go]` |
| Δ2 | APPLIED 2026-11-16. T11 (closes ntpl-lint-walk-k-rows Δ6): `-update` rewrites the `want` and `stderr` sections of each case that differs, keeps every other byte, resolves the case dir to an absolute path (CHDIR=1), writes nothing without the flag | `record/README.md` (Re-recording through the Go port) · `internal/reportmodule/testdata/README.md` · Go `internal/lint/rules/golden` `update.go` | `[ev: go/internal/lint/rules/golden/update.go]` |
