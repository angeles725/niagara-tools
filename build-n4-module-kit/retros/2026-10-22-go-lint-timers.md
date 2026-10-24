<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-lint-timers

**Session**: niagara-research quality round 8, work unit T8k-3, branch `r8/t8k3-lint-timers` on the engine commits 7410424/4e20152/70d15be
**Delta count**: 2

## What happened
`toolbelt/lint-timers.sh` was the slowest kit lint: 615 lines of bash and awk that ran two `printf | grep` per Java
line and four to five awk programs per file, each pass over every file. It is now the rule package
`go/internal/lint/rules/timers`, a `lint.TreeRule` (every check runs over every file in walk order, so rows stay
grouped by check), and the script is a wrapper that execs `n4lint lint-timers`. The port keeps the awk programs'
gawk semantics on purpose: POSIX leftmost-longest matching, `while (match(t, re))` loops over a shrinking
remainder, grep's UTF-8 `.`, bash `read` skipping an unterminated last line and splitting a path with spaces,
and gawk's `for (k in a)` hash order for the orphan-flag and release-gate-parity rows (`gawkOrder`, a port of
gawk 5 `str_array.c` checked against gawk itself). R8-D7 needed no change: the bash already FAILed jdk-thread only
for a thread modification without MANAGE_EXECUTION.

## Evidence
- RED: the 62 golden cases (every tests/lint-timers.bats case recorded from the old script through a recording
  wrapper, plus 3 edge trees) failed against a stubbed CheckTree (54 of 62; the 8 others expect no rows), as did the
  unterminated-line and unreadable-permissions tests `[ev: go/internal/lint/rules/timers/timers_test.go]`
- Byte identity, old script (git archive 70d15be) vs wrapper, stdout + stderr + exit: same on tests/fixtures/lint-timers,
  Cliente/Honeywell, Cliente/panccadia-leon, Cliente/Leon-Guanjuato, no args, `-h`, a missing root, a file root and
  two arguments. A 600-file sample of decompiled Niagara sources that use Clock, executors, jobs or atSteadyState
  gives the same 423 rows and exit; its stderr differs only by gawk's two "Invalid multibyte data detected" warnings
  for one non-UTF-8 file, which the Go port does not print `[ev: go/internal/lint/rules/timers/timers.go]`
- Timing, one run each: Honeywell 65.9 s → 0.25 s, panccadia-leon 99.4 s → 0.37 s, Leon-Guanjuato 22.9 s → 0.11 s,
  the 600-file sample 883 s → 5.6 s
- Intended difference: an unreadable module-permissions.xml now exits 3 and stops the run, as the header documented;
  the bash `exit 3` ran inside the `$(perms_grant)` subshell, so the old run went on and could exit 0
- Engine-owned differences (engine frozen, reported): after a walk error the engine still scans the listed files,
  so an unreadable sub-directory prints rows the bash never printed (same stderr line and exit 3); the walk message
  quotes the directory as find does in the C locale (`'dir'`), while GNU find under C.UTF-8 prints `‘dir’`
- Mutation spot checks: insertion order instead of gawk order, dropping the grep UTF-8 rule, counting the
  unterminated last line, not splitting the permissions path and scanning fields at any depth each fail a golden case

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Record a lint's golden cases from the old script before porting it.** Point the bats suite's `LINT` at a wrapper that copies `$BATS_TEST_TMPDIR` and execs the old script, then store tree + stdout + exit per case as testdata. Why: hand-written expectations miss the incidental behaviour (row order, hash order, read/grep quirks) that byte identity needs. Reviewer check: a port whose Go tests restate the bats assertions as substrings instead of exact recorded rows is a finding. | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/timers/timers_test.go]` |
| Δ2 | **A bash `exit` inside `$(...)` does not stop the script.** An env check in a function called through command substitution must return a status the caller tests, or the lint fails open. Why: perms_grant printed the error and exited 3 in the subshell, and lint-timers went on to exit 0. Reviewer check: `exit` inside a function whose output is captured with `$(...)` is a finding. | `BUILD-LOOP.md` § lint authoring | `[ev: build-n4-module-kit/toolbelt/lint-timers.sh]` |

## Lessons
- gawk's `for (k in a)` order is deterministic (awk_hash, 13/127/1021 buckets) and is part of a ported lint's output.
- Error texts and stop points belong to the output contract; list every one a port changes.
- A lint that runs one process per line is the first one to port: 66-99 s became under 0.4 s.

---
**Status**: FOLDED (quality round 8 pointer unit, `docs/r8-pointer-lines`) — Δ1 `go/README.md:86-94` § Adding a lint rule step 3, Δ2 `CONTRIBUTING.md:225-229` § 6; tokens in `ledger.d/2026-10-23-r8-pointer-lines.md`. INDEX row generated from the header: `| 2026-10-22-go-lint-timers.md | kit | 2026-10-22 | folded | 2 |`
