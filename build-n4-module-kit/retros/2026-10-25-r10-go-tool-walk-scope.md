<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-25 · kit · r10-go-tool-walk-scope

**Session**: niagara-tools quality round 10 (Ayudante1), CI wall time, branch `perf/go-tool-prebuild-stamp`
from origin/main 08bfe226
**Delta count**: 2

## What happened
`go/go-tool.sh` `go_tool_bin` walked the whole `go/` tree on every call, about 3,928 times in
`tests/report-module.bats` alone, which sets the floor of the CI bats step. The first design was a prebuilt
stamp: CI's pre-build would record the tree and later calls would prove a binary current from the stamp.
That design was dropped before commit after measurement. To stay fail-safe, a check must still stat every
source and directory on each call to see a later edit. That is about as many stat calls as the walk (about
1,400 `newfstatat`), so the best stamp check measured 12.5-16 ms against a 16-25 ms walk, and the first
version was slower (40 ms). A content hash per call cost 80 ms.

The walk was made cheaper instead. A source is now every file under `go/` except `bin/`, `testdata/` and
`*_test.go`, none of which `go build` compiles into a binary: 305 test files are skipped and the walk
reads 155 directories instead of 359. The new scope also closes a staleness gap that already existed: the
old walk watched only `*.go` and `go.mod`, so an edit to one of the five `//go:embed` assets
(`registry.json`, `retro.template.md`, `unicode.txt`, `usage.txt`, `b795-795.4.csv`) left the old binary
running with the old asset. Non-.go files now count, so such an edit triggers a rebuild.

## Evidence
- RED on the old walk, then GREEN: GT10 (edited `*_test.go`, no rebuild), GT11 (edited `testdata/`, no
  rebuild), GT12 (edited embedded asset, rebuild). GT13 (non-test `.go` beside a test file, rebuild) and
  GT14 (no non-test file embeds from `testdata/`) pin the rest `[ev: tests/go-tool.bats]`
- Mutations, each RED: GT9 `find | head` -> GT9; `*_test.go` exclusion dropped -> GT10; `testdata` prune
  dropped -> GT11; old `*.go`/`go.mod` name filter -> GT12 `[ev: go/go-tool.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A build-on-demand staleness check watches what the build reads: every file of the module except `bin/`, `testdata/` and `*_test.go`, embedded assets included; a check that watches only `*.go`/`go.mod` serves a stale `//go:embed` asset | `CONTRIBUTING.md` § 2.2 Go module | `[ev: go/go-tool.sh]` |
| Δ2 | Before a cache or stamp replaces a filesystem walk, measure the walk's syscalls: a check that is fail-safe against later edits stats the same files, so it cannot be much cheaper; narrow the set the walk reads instead | `CONTRIBUTING.md` § 2.2 Go module | `[ev: tests/go-tool.bats]` |

## Lessons
- A fail-safe per-call freshness check costs one stat per watched file, the same as the walk it replaces.
- Narrow a watched set to what the build reads, and check that every input the build reads, embedded
  assets included, is still in it.
- A prune is safe only while a test pins its assumption: here, no non-test file embeds from `testdata/`.

## Fold
- The narrowed walk is code already shipped in the PR that wrote this retro (`go/go-tool.sh` `go_tool_bin`, `tests/go-tool.bats` GT10-GT14); not re-implemented.
- Δ1 and Δ2 → `BUILD-LOOP.md` § 5 Go-first bullet, not `CONTRIBUTING.md` § 2.2: that file is outside the kit and its citations are not counted by the fold audit; the Go-first bullet links to it for the layout.
- Lesson 3 (a prune is safe only while a test pins its assumption) is GT14, cited with Δ1. Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `docs/r10-fold-tail-4`; log in `ledger.d/2026-10-25-r10-fold-tail4.md`) — Δ1, Δ2 in the `BUILD-LOOP.md` § 5 Go-first bullet.
