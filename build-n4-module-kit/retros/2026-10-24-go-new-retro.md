<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-24 · kit · go-new-retro

**Session**: niagara-research quality round 10, branch `r10/new-retro-go` from origin/main 15e792d1
**Delta count**: 6

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
  `[ev: tests/new-retro-truth.bats NRT1]` `[ev: tests/new-retro-truth.bats NRT2]`
- Review fixes of e8406d0 (Δ4-Δ6): atomic envelope + rollback, no PASS without a flip, --deltas capped at 99;
  6 recorded cases became truth cases in all, so 33 stay recorded and 11 are hand-written
  `[ev: go/internal/newretro/command_test.go TestEnvelopeWriteFailureLandsNothing]` `[ev: tests/new-retro-truth.bats NRT3]`
- A draft golden with `--deltas 99999999999999999999` made the package test reach 37 GB RSS and the host OOM-killed
  every session: `strconv.Atoi` returns the clamped maximum WITH an error, the error was dropped, and the count
  sized the stub's row loop. After the fix the package test binary peaks at about 10 MB RSS
  `[ev: go/internal/newretro/command_test.go TestDeltasRefusedBeforeRender]`
- The index half of the recorded goldens is not bash parity: the recorder ran this checkout's gen-retro-index
  (Go retroindex since r8), and each @kind says so

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | DONE (T1, this branch): the bash slug glob checked only the first 6 chars, so `abcdef ghi` wrote a retro with a space in its name and `abcdef/../../x` reached cp; the whole slug is now checked | `go/internal/newretro/newretro.go` slugOK | `[ev: tests/new-retro-truth.bats NRT1]` |
| Δ2 | DONE (T2, this branch): `--date ../x` wrote the stub outside retros/ (missing from the index) yet flipped the envelope, `--date 2026-13-01` was accepted, `--deltas -1` / `x` failed only in the index generator after the stub was written; both values are now validated first (exit 3) | `go/internal/newretro/newretro.go` parse | `[ev: tests/new-retro-truth.bats NRT2]` |
| Δ3 | A port's golden case should pin every file the run leaves (the whole tree after the run, not a chosen file list): a stub left behind or a write outside the target dir fails the case | `go/README.md` § `Adding a lint rule` step 3 | `[ev: go/internal/newretro/golden_test.go TestGolden]` |
| Δ4 | DONE (review W1/W3): BUILD-STATE.src.md was rewritten with a truncating write after the stub and INDEX.md had landed; it now goes through a temporary file + rename, and a failure removes the stub and restores INDEX.md (a stub whose write fails is removed too) | `go/internal/newretro/newretro.go` write | `[ev: go/internal/newretro/command_test.go TestEnvelopeWriteFailureLandsNothing]` |
| Δ5 | DONE (review W2): no section for the module (a typo, an empty file) or a section without retro_pending exits 3 before anything is written; an already-true envelope is reported as `retro_pending: already true`, never as a flip | `go/internal/newretro/flip.go` Flip | `[ev: tests/new-retro-truth.bats NRT3]` |
| Δ6 | DONE (review S5): --deltas is 0-99, checked in parse with the Atoi error checked, before the renderer can size a loop from it | `go/internal/newretro/newretro.go` count | `[ev: go/internal/newretro/command_test.go TestDeltasRefusedBeforeRender]` |

## Follow-ups (not done here)
- W4: two concurrent runs both read BUILD-STATE.src.md before either writes it, so the later rename drops the
  other's flip (and each regenerates INDEX.md from its own scan). The kit runs new-retro one at a time today;
  a lock file in retros/ would make it safe.
- S6: a module name is matched byte for byte, so `a\b` matches `module: a\b` literally (the awk `-v` assignment
  of the bash unescaped backslashes first). No real module name holds one.
- S7: the recorded goldens prove the stub and envelope halves against the bash; the INDEX.md half is the Go
  retroindex the recorder's checkout already ran.

## Lessons
- Never size a loop, slice or builder from an input whose parse error was dropped: `strconv.Atoi` on an overflow returns the clamped maximum together with the error, and one such golden case took the whole host down.
- A write-side tool's goldens need the tree after the run as well as stdout/stderr/exit; the bash's out-of-dir write (Δ2) only showed up as an extra `@expect` file.
- Inject the clock into the command (`Command(today)`), not the environment: the recorder fakes `date`, the Go test passes the same date, and a wrapper check maps the real date once.
- Keep a wrapper's DEFAULT root on the path the old script used (unresolved), even when `here=` must resolve for `go/`; they answer different questions.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-24-go-new-retro.md | kit | 2026-10-24 | pending | 3 |`
