<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-10-24 · kit · go-new-retro

**Session**: niagara-research quality round 10, branch `r10/new-retro-go` from origin/main 15e792d1
**Delta count**: 3

## What happened
`toolbelt/new-retro.sh` (bash + an awk flip + a shell-out to `gen-retro-index.sh`) is now the thin `go/go-tool.sh`
wrapper over `go/cmd/new-retro` (`go/internal/newretro`). It is a standalone `cli.Main` command: it writes the retro
stub, regenerates retros/INDEX.md in process with `retroindex.Generate` + `Output` (no second binary, so the bash
"generator not found" exit 3 is unreachable) and flips retro_pending in BUILD-STATE.src.md (`Flip`, the awk it
replaced). Since R10-D1 neither INDEX.md nor BUILD-STATE.md is committed, so the tool writes the committed
BUILD-STATE.src.md and the generated, ignored INDEX.md, as the bash did. The wrapper keeps the bash default kit root:
the UNRESOLVED `../` of the script (a symlinked toolbelt copy names its own parent), while `here=` resolves the real
path for `go/` (GWS3).

## Evidence
- 39 goldens recorded from the bash at 15e792d1 by `go/internal/newretro/testdata/record/record.sh`: the 10
  retro-loop.bats runs plus kit-root, usage, option, slug and flip edge cases; each pins exit, stdout, stderr, the
  environment and every file the run left (the whole tree after the run), with the date injected (a fake `date`
  in the recorder, `Command(today)` in Go) `[ev: go/internal/newretro/golden_test.go TestGolden]`
- RED (package missing) → GREEN, all 39 byte-identical; dropping the stub removal on a generator failure fails
  bats-rl10-generator-fails and edge-deltas-negative
- The recorder pointed at the wrapper reproduces all 39 once today's date maps to the recorded one (the binary
  reads the real clock); retro-loop, build-retro-sync, kit-links, go-wrapper-symlink 85/85 ok after the switch
- Truth fixes T1-T2 (Δ1-Δ2), each one commit with its RED, hand-written truth-* goldens and a bats pin
  `[ev: tests/new-retro-truth.bats NRT1]` `[ev: tests/new-retro-truth.bats NRT2]`; 3 recorded cases became
  truth cases, so 36 stay recorded

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE (T1, this branch): the bash slug glob checked only the first 6 chars, so `abcdef ghi` wrote a retro with a space in its name and `abcdef/../../x` reached cp; the whole slug is now checked | `go/internal/newretro/newretro.go` slugOK | `[ev: tests/new-retro-truth.bats NRT1]` |
| Δ2 | DONE (T2, this branch): `--date ../x` wrote the stub outside retros/ (missing from the index) yet flipped the envelope, `--date 2026-13-01` was accepted, `--deltas -1` / `x` failed only in the index generator after the stub was written; both values are now validated first (exit 3) | `go/internal/newretro/newretro.go` parse | `[ev: tests/new-retro-truth.bats NRT2]` |
| Δ3 | A port's golden case should pin every file the run leaves (the whole tree after the run, not a chosen file list): a stub left behind or a write outside the target dir fails the case | `go/README.md` § `Adding a lint rule` step 3 | `[ev: go/internal/newretro/golden_test.go TestGolden]` |

## Lessons
- A write-side tool's goldens need the tree after the run as well as stdout/stderr/exit; the bash's out-of-dir write (Δ2) only showed up as an extra `@expect` file.
- Inject the clock into the command (`Command(today)`), not the environment: the recorder fakes `date`, the Go test passes the same date, and a wrapper check maps the real date once.
- Keep a wrapper's DEFAULT root on the path the old script used (unresolved), even when `here=` must resolve for `go/`; they answer different questions.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-new-retro.md | kit | 2026-10-24 | pending | 3 |`
