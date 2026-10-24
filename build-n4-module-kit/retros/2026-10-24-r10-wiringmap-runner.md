<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-wiringmap-runner

**Session**: niagara-research quality round 10, R10-D14 follow-up, branch `r10/wiringmap-runner` from origin/main 70af46b2
**Delta count**: 2

## What happened
`go/internal/wiringmap` was the last Go port with a private golden loop (whole post-run tree, `@expect` sections)
and a private recorder (`testdata/record/record.sh` + `cases.sh`). Its 65 cases were converted to the
`golden.Runner` format (`@expect` to `@after`, `(locked)`/`(unreadable)` to `(mode 000)`, `@tools` folded into
`@kind`), the 58 recorded ones re-recorded with `golden/record/record.sh` from the bash at 15e792d1, and the
private loop and recorder deleted. `TestGolden` is now one `golden.Runner` literal (exact count 65, `CHDIR=1`).
The `toolbelt/new-retro.sh` header no longer names the deleted newretro recorder.

## Evidence
- Re-recording: 58/58 recorded cases byte-identical to the converted ones once the `@kind` lines are dropped;
  the script and `lib/scan-files.sh` hashed to the pins `[ev: go/internal/wiringmap/testdata/record.env]`
- RED: a planted `# Wiring Mop` in the `@after wiring-map.md` section of `bats-output-file` failed
  `TestGolden/bats-output-file` with `changed wiring-map.md after the run` `[ev: go/internal/wiringmap/golden_test.go]`
- The first converter run crashed on `edge-lexicon-bytes` (a 0xe9 byte that is not UTF-8) after writing 18 cases
  `[ev: go/internal/wiringmap/testdata/edge-lexicon-bytes.txt]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A golden-case converter reads bytes, not text**: `newline=''` keeps CRLF but a UTF-8 decode still fails (or mangles) a case that pins non-UTF-8 input; open with a byte-preserving codec (latin-1) or in binary mode, and split on `\n` only. | `BUILD-LOOP.md` § 5 golden recordings | `[ev: go/internal/wiringmap/testdata/edge-lexicon-bytes.txt]` |
| Δ2 | **Retiring a recorder greps every header that names it**: the newretro recorder was deleted but `toolbelt/new-retro.sh` still pointed at it; `toolbelt/generate-wiring-map.sh` line 45 still names the wiringmap one. | `BUILD-LOOP.md` § 5 What to port | `[ev: build-n4-module-kit/toolbelt/new-retro.sh]` |

## Follow-ups
- `toolbelt/generate-wiring-map.sh` header (line 45) still names `go/internal/wiringmap/testdata/record/record.sh`
  (outside this change's edit surface).

## Lessons
- The shared recorder reproduced the private recorder's bytes exactly once the default `HOME`/`LC_ALL` were added:
  the bash reads neither, so no case declares them.
