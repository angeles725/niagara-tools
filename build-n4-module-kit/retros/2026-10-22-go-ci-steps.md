<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-ci-steps

**Session**: niagara-research quality round 8 work unit T8j-2, branch `r8/t8j2-ci-steps-go` from origin/main 657ae5b
**Delta count**: 2

## What happened
The standing rule of 2026-10-22 (new tools and ports in Go when possible) reached the second tool:
`scripts/lib/ci_steps.py`, the workflow-step parser behind `scripts/ci-local.sh`, became `go/cmd/ci-steps`
(`go/internal/cisteps`) and the Python file was deleted. A second wrapper needed the same build-on-demand logic
as `toolbelt/gen-retro-index.sh`, so that logic moved once into `go/go-tool.sh` (`go_tool_bin`), now used by both
and pinned by `tests/go-tool.bats`. The same unit fixed the RDD advisories left on #288: `retroindex` swallowed
`ReadDir` and stat errors and its exported `ParseHeader` dropped read errors (now fail closed, exit 3, nothing
rendered), the wrapper rebuild paths had no tests, and the CI `gofmt -l` step could mask a gofmt failure.

## Evidence
- Byte-identity: the origin/main `ci_steps.py` and the new binary over the 6 `tests/fixtures/ci-local/*.yml`, the new
  and the origin/main `ci.yml`, a missing file and a directory give the same stdout, stderr and exit (10/10); the
  origin/main `ci-local.sh --list` and the new one match for all of them and for the default `ci.yml` (24 lines);
  29 table cases + 6 edge shapes differ only for invalid UTF-8 (Python's codec message is not reproduced)
  `[ev: T8j-2 byte-identity harness]`
- `go/internal/cisteps` table tests port every bats fixture shape plus the error forms (folded `>`, `|+`,
  unterminated quote, `run:`/`uses:` outside a step, workflow/job-level `env:`/`defaults:`)
  `[ev: go/internal/cisteps/cisteps_test.go TestParse]`
- `tests/go-tool.bats` GT1-GT8 RED 0/8 → GREEN 8/8; mutation GT3 (test only `[ -x bin ]`) fails GT3/GT4/GT5/GT7
  `[ev: tests/go-tool.bats]`
- `TestScanReportsAnUnstatableRetroAsIO` fails when a stat error is skipped again; `TestMainUnlistableRetrosDirExits3`
  pins exit 3 with `index not written` `[ev: go/internal/retroindex/retroindex_test.go]`
- `tests/build-retro-sync.bats` `_fixture_top` symlinks only `go/` next to a copied wrapper, so a helper under
  `scripts/lib/` would have broken H11/H12; the helper lives in `go/` `[ev: retro go-module-gen-retro-index Δ2]`
- A step-level `working-directory:` stays UNCOVERED: supporting it changes the record format, so the CI Go steps
  keep `cd go` inside `run:` `[ev: go/internal/cisteps/cisteps.go stepKeys]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A Go wrapper gets its binary from `go/go-tool.sh` `go_tool_bin`, never from its own rebuild code.** The helper rebuilds when the binary is missing or older than any `*.go`/`go.mod`, returns 3 with a `<tool>:` message when the module, the toolchain or the build is missing, and leaves the old binary and no temporary file behind. Why: two copies of the rebuild check drift, and the first copy shipped untested. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-ci-steps]` |
| Δ2 | **A generator fails closed on I/O: an unlistable directory or an unreadable input is exit 3 and nothing is rendered.** Only a vanished entry or a dangling symlink may be skipped. Why: a skipped unreadable retro produced an index that looked complete. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-ci-steps]` |

Both are applied in this unit (the helper and the fail-closed `retroindex`; § 2.2 documents Δ1); the retro stays
pending until a fold review decides them.

## Lessons
- When a second wrapper needs logic, extract it before the copy lands, and put the helper where every fixture that
  copies a wrapper already mirrors the tree (here `go/`).
- A port that must match Python byte for byte needs Python's semantics spelled out: Unicode `\w`, text-mode newline
  translation, `str.isspace` and the `OSError` message format.
- `test -z "$(cmd)"` ignores the exit status of `cmd`; assign first (`out="$(cmd)"`) so `bash -e` stops on its failure.

---
**Status**: PENDING (Δ1/Δ2 applied in T8j-2; fold review pending) — INDEX row generated from the header: `| 2026-10-22-go-ci-steps.md | kit | 2026-10-22 | pending | 2 |`
