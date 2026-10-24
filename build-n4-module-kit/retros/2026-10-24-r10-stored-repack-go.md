<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-stored-repack-go

**Session**: niagara-research quality round 10, branch `r10/stored-repack-go` from origin/main 91b9e3d6
**Delta count**: 2

## What happened
`toolbelt/stored-repack.sh` (45 lines: unzip, then `zip -0` in a set order) was ported to Go as
`go/internal/storedrepack` and `go/cmd/stored-repack`. The script is now the `go_tool_bin` wrapper; its header
(lines 2-15, still the usage text) stays. The entries stream from the input zip into a temporary file beside
the output: `CreateRaw` gets the input's CRC and size, so each STORED entry has its sizes in the local header
and no data descriptor, and `zip.File.Open` checks the bytes against those headers at EOF. The result is
reopened and counted, then linked into place. No extraction, no zip/unzip tools, and never a partial output.

- **Contract.** It is semantic, not byte identity, because zip(1) and archive/zip write different timestamps,
  attributes and header layouts. The contract covers the exact entry names and order, every entry STORED
  with the input's bytes, no directory entries, and the exact stdout, stderr and exit codes. "Everything else"
  keeps its input order. The bash wrote those entries in the directory-walk order of its extraction dir, which
  is a filesystem property, so the goldens compare that part sorted.
- **Goldens.** `record.sh` recorded 22 cases from the bash at 91b9e3d6, with a `testdata/setup.sh` that
  builds each input jar from a `.spec` and lists the output through `jarlist` (names, order, method, size,
  content hash, data-descriptor flag). They cover the four bats runs plus edge trees: several aliases and
  blocks, an orphan block, nested and lowercase `.SF`, stored input, directory entries, dotfiles, a manifest
  last or as a directory, no manifest, an existing output, and the input named as output. 22/22 pass on Go.
- **Declared differences** (truth-* cases in the port commit):
  - No zip/unzip on PATH is no longer exit 2.
  - A corrupt zip, a CRC mismatch, a `..` entry name and a duplicate name fail as before (exit 1,
    `cannot unzip <in>`), with one Go reason line in place of unzip's report.
- **Truth fixes.** One commit each, RED quoted:
  - A dangling-symlink output is refused. The bash's `[ -e ]` let zip write through it.
  - A missing output directory is exit 1 `cannot write`. The bash's `cd` failed inside `OUT_ABS`, so it
    tried `/out.jar` and exited 15.
  - A manifest-only jar is exit 0. The bash hit `zip error: Nothing to do!`, exited 12 and left a partial jar.

## Evidence
- `go test ./internal/storedrepack/...` (30 goldens: 22 recorded, 8 truth; order, STORED headers, content identity,
  exits, usage-is-header, bad-CRC-leaves-nothing, dangling symlink) green `[ev: go/internal/storedrepack/testdata]`
- bats S1-S4 unchanged on the wrapper; S5 multi-alias with verify-module `--stored`, S6 existing output, S7-S9 the
  truth fixes `[ev: tests/stored-repack.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A port whose output is a binary archive pins its observable contract: setup.sh builds inputs from text specs and a run wrapper writes a listing both sides compare | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/storedrepack/testdata/setup.sh]` |
| Δ2 | A STORED zip entry from Go goes through `CreateRaw` with the source CRC and size, streamed from `File.Open`; `CreateHeader` sets the data-descriptor flag `ZipInputStream` rejects on STORED | `go/README.md` § Adding a lint rule (after the shared-code rule) | `[ev: go/internal/storedrepack/storedrepack.go]` |

## Lessons
- record.sh cannot spell a jar, but its `setup_case` and `RUN` hooks can build one and post-process the run.
  A binary port needs no recorder of its own.
- In bats, `[ a ] && [ b ]` on a line that is not the test's last never fails the test. A wrong entry count
  passed until the assertions were split onto their own lines.
