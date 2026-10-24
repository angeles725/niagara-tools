<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-new-retro-flip-lock

**Session**: niagara-research quality round 10, branch `fix/newretro-flip-lock` from origin/main 70af46b2 (follow-up W4 of retro `2026-10-24-go-new-retro`)
**Delta count**: 1

## What happened
Two `new-retro` runs at once both read BUILD-STATE.src.md before either wrote it, so the later rename dropped the
other run's `retro_pending` flip, and each regenerated retros/INDEX.md from its own scan (a row could go missing, and
the shared temporary file made one run fail with exit 1). `write` now holds an exclusive `flock(2)` on the `retros/`
directory itself from the existence checks through the stub, the INDEX.md regeneration and the envelope rename.

## Evidence
- RED: `TestConcurrentRunsKeepEveryFlip` (4 runs, one module each) failed 20/20 under `go test -race -count=20`: 3 of 4
  flips lost, missing stubs and INDEX rows, runs exiting 1 `[ev: go/internal/newretro/concurrent_test.go]`
- The interleaving is forced, not timed: the `afterRead` test seam is a barrier that holds each run just after it
  reads the envelope until every run has read it (a 100 ms timeout releases it once runs are serialised and the
  others cannot arrive)
- GREEN: the same 20 runs pass with the lock `[ev: go/internal/newretro/lock.go]`
- The directory, not a lock file, is locked: nothing is left in the tree (goldens and `TestNoWriteOutsideRetros`
  unchanged, no .gitignore entry), the directory is never replaced (BUILD-STATE.src.md and INDEX.md are, by rename,
  so a lock on them could land on a stale inode), and the kernel drops a flock when its last descriptor closes,
  process death included, so a killed run never blocks the next one

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A kit tool that read-modify-writes a committed file holds one exclusive `flock` on a directory that is never replaced, from its first read through its last rename (generators included); a lock on a file the tool replaces by rename does not exclude. | `go/README.md` § `Adding a lint rule` | `[ev: go/internal/newretro/lock.go]` |

## Lessons
- A race test needs a barrier at the read, not a sleep: hold every run after its read until all have read, with a
  short timeout so the serialised (fixed) code still finishes.
- flock is per open file description, so goroutines that each open the path exclude each other in one process; the
  test exercises the real lock without subprocesses.
