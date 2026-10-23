<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · line-loop-lint-forks

**Session**: niagara-research quality round 8, work unit T8l, branch `r8/t8l-line-loop-perf` from origin/main 1879746
**Delta count**: 1

## What happened
The line-loop lints moved onto `lib/awk-lint.sh` by retro `2026-10-22-awk-lint-line-loops` still ran a bash
`while read` over every source line, and several forked a `printf | grep` (and a `$(... | sed)` per hit) on each
line: two clones and one exec per source line. On a 33k-line client tree one lint took about a minute.
`lib/awk-lint.sh` gains `awk_lint_each '<awk program>'`: one awk per listed file, the file on stdin, its path
in `ENVIRON["AL_FILE"]`, rows printed, `AL_ROWS=1` when a row printed; an unreadable file prints
`<name>: cannot scan <file>: <reason>` and sets `AL_ERR`, so `awk_lint_exit` exits 3. Six lints use it:
agent-on-shape, se-display, clock-zero-floor, bql-string-concat, no-system-out and arbitrary-ord (its
string-aware `code_part` and `marker_state` ported to awk functions). Their matching semantics are unchanged,
including the `//`-inside-a-string quirk of the five simple strippers. Intended behaviour change: an unreadable
source file is exit 3, not bash's "Permission denied" plus exit 0 (clock-zero-floor skipped it silently). The six
lints also source the lib through `dirname`, so `bash lint-x.sh <root>` works from inside `toolbelt/`.
Left as is: lint-uberjar-api-conflict (a few greps per file, no per-line loop), lint-bundled-jar-class-version
(per jar entry, at most 20 per jar, binary bytes awk cannot read), and lint-null-context-write, which another
session is rewriting in parallel; it is re-measured after that work merges.

## Evidence
- RED: 9 new cases failing before the change (AL21-AL23, AOS/SED/CZF/BSC/NSO/AO-unreadable); GREEN
  `bats -j 3 tests/lint-*.bats tests/awk-lint.bats` 509 ok `[ev: tests/awk-lint.bats]`
- Relative invocation before: `cd toolbelt && bash lint-no-system-out.sh ../types` printed `cd: ... Not a directory`
  and `awk_lint_parse: command not found`, exit 1 (not even the env exit 3) `[ev: tests/awk-lint.bats]`
- Timings old → new (one run at a time), Cliente/Honeywell (23k lines) / Cliente/panccadia-leon (33k lines):
  bql-string-concat 38.17 → 0.16 s / 55.95 → 0.20 s; arbitrary-ord 42.75 → 0.21 s / 49.19 → 0.19 s;
  clock-zero-floor 0.09 → 0.21 s (no file there names Clock.schedule, so the old grep pre-filter skipped every
  file) / 13.51 → 0.23 s; se-display 0.28 → 0.23 / 0.56 → 0.27 s; no-system-out 0.24 → 0.16 / 0.43 → 0.17 s;
  agent-on-shape 0.05 → 0.02 / 0.04 → 0.02 s `[ev: toolbelt/lib/awk-lint.sh]`
- Byte-identity: stdout and exit code equal, origin/main copy vs branch, for the six lints on both client trees,
  the repo root and a crafted edge tree (CRLF, no final newline, empty and unclosed `type=`, several `<on` per
  line, `>` inside the value, `//` in strings and chars, `/* it's */`, reviewed markers with and without reason)
  `[ev: toolbelt/lib/awk-lint.sh]`
- Scripts net: the six lints +131/-149, lib +30/-2 (helper and its doc) `[ev: toolbelt/lib/awk-lint.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A lint that inspects source line by line runs one awk per file (`awk_lint_each`), never a shell loop that forks per line.** Why: a `printf \| grep` per line costs two clones and one exec per source line, about a minute on a 33k-line tree against 0.2 s. Reviewer check: a `while read` over a source file whose body runs `grep`, `sed` or a `$(...)` is a finding. | `BUILD-LOOP.md` § Lint doctrine (with retro awk-lint-line-loops Δ1) | `[ev: toolbelt/lib/awk-lint.sh]` |

## Lessons
- Measure per lint, not per pattern: the `case`-only loops (se-display, no-system-out) were already fast; the time
  went to the loops that forked per line.
- A grep pre-filter per file can beat a full awk pass on a tree where it rejects every file; the trade is fine at
  0.2 s, but the benchmark has to say so.
- Pass a path to awk through `ENVIRON`, never `-v`, which escape-processes backslashes in the path.
- `${BASH_SOURCE[0]%/*}` is not a directory when the script is run by bare name; use `dirname`.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `BUILD-LOOP.md:226-232` § Kit tooling doctrine "No fork per input line" + `tests/lint-fork-per-line.bats`. INDEX row generated from the header: `| 2026-10-22-line-loop-lint-forks.md | kit | 2026-10-22 | folded | 1 |`
