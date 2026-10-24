<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-installed-line-guard

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`), branch
`feat/r10-installed-line-guard` from origin/main fa2b9b71
**Delta count**: 1

## What happened
A PANCCADIA fix branch (`fix/r10-coldroompan-token-leak`) was started from an old commit instead of the installed
line. Building it would have shipped DashboardPan at a LOWER version than the one already on the station, and
nothing in the kit noticed: the build.sh drift gate compares a jar with the deployed baseline only when the
baseline is extracted, and a module-level version bump looks normal in isolation. A session-private prototype
caught it; this promotes it into the kit as `scripts/check-installed-line.sh`, with the source-side read
(`vv_source_version`) and the compare (`vv_cmp`) in the shared `toolbelt/lib/vendor-version.sh`.

## Evidence
- Prototype output against installed commit 17c80d3: `DOWNGRADE Dashboard/build.gradle.kts 2.10.0 -> 2.9.2`
  (Compresores 2.8.0 = 2.8.0, Paccadia 2.4.2 -> 2.4.4)
- The prototype compared with `sort -V` (a second comparator) and passed an incomparable suffix; the kit form
  uses `vv_cmp` and refuses `incomparable` (exit 1, named)
- Pins: `tests/installed-line-guard.bats` ILG-* (the incident shape is ILG-downgrade), `tests/vendor-version.bats`
  VV-source-version*; observed mutations: dropped -1 branch (ILG-downgrade red), incomparable read as ok
  (ILG-incomparable red), groups missing on the branch skipped (ILG-missing red)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A fix branch must start from the installed line.** Before building a client fix branch, run `scripts/check-installed-line.sh <client-repo> <branch> <installed-ref>`; a DOWNGRADE, MISSING or INCOMPARABLE row (exit 1) means rebase onto the installed line, never build. | `BUILD-LOOP.md` §0.b preflight, next to check-client-source | `[ev: tests/installed-line-guard.bats ILG-downgrade]` |

## Lessons
- A version guard that reads only one side (the jar being built) cannot see a branch cut from the wrong base;
  compare the branch with the installed ref at the source.

---
**Status**: FOLDED 2026-10-24 (branch `feat/r10-installed-line-guard`) — Δ1 `BUILD-LOOP.md` §0.b preflight bullet.
