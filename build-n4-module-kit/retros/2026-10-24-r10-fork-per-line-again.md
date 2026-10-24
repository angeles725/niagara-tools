<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-fork-per-line-again

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`), issue #432,
branch `fix/r10-preflight-jarlock-single-pass` from origin/main 0bd8728b
**Delta count**: 1

## What happened
`preflight.sh` Check 4 (jar-lock) hung for more than 10 minutes on a busy host with 410 module jars
(2026-10-24, kit 780c7095, PANCCADIA ColdRoomPan build). The single-`lsof` fix from retro
panccadia-defrost-sequencing-hmi-reload-deltas Δ8 had removed one `lsof` fork per jar, but kept one
`printf '%s\n' "$LSOF_OUT" | grep -qxF "n$jar"` per jar: two forks per jar, each copying and scanning the whole
captured listing. Cost grew with jars times listing size. The fork-per-line guard
(`tests/lint-fork-per-line.bats`) did not see it, for two reasons: it scans only `toolbelt/lint-*.sh`, and it
reports a fork fed by a `while read` variable, while this loop was a `for` over a glob that piped a captured
listing.

## Evidence
- Before (origin/main 0bd8728b): `tests/preflight.bats` PF-jarlock-bulk-timeout, 400 jars against a
  60000-line fake listing, exceeded `timeout 10` (exit 124); the three 400-jar cases took 2 min 50 s together
- After: one preflight run over the same fixture takes 154 ms; the 12 jar-lock cases run in 2.5 s
- Design: the jar set goes into one awk as a lookup table, the listing is read once, and matches print in
  glob order. Same rows, same SKIP paths (9p/drvfs, lsof absent, empty listing), whole-line `n<path>` match
- Pins: PF-jarlock-bulk-order, -bulk-timeout, -bulk-pass, -bulk-empty, -whole-line, -regular-only

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **No fork per item over a captured listing.** A loop that runs `printf "$BIG" \| grep` (or any fork) once per item, against a listing captured once, is the same fork-per-line class: the work grows with items times listing size. Load the item set and the listing into one awk (set lookup, one pass) instead. The fork-per-line guard covers only `while read` loops in `lint-*.sh`, so a bounded-time bats case with a realistic item count (hundreds) is the pin for any other script. | `BUILD-LOOP.md` § 5 kit tooling doctrine, "No fork per input line" | `[ev: tests/preflight.bats PF-jarlock-bulk-timeout]` |

## Lessons
- A perf fix that removes the expensive fork can leave a cheaper fork per item; time the fix against the real
  item count, not the count of the original complaint.
- A structural guard proves only its scope. Name what it does not cover where the rule lives.

---
**Status**: FOLDED 2026-10-24 (branch `fix/r10-preflight-jarlock-single-pass`) — Δ1 `BUILD-LOOP.md` § 5
"No fork per input line" extended to per-item forks over a captured listing; `toolbelt/preflight.sh` Check 4
ported to one awk pass.
