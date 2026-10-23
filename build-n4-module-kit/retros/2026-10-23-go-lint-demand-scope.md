<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-23 · kit · go-lint-demand-scope

**Session**: niagara-research quality round 8 unit E3c, branch `r8/e3-lint-demand-scope-go` from origin/main f56db17
**Delta count**: 1

## What happened
`toolbelt/lint-demand-scope.sh` (196 lines, one awk fork per file) is now the rule package
`go/internal/lint/rules/demandscope` and a thin wrapper that execs `n4lint lint-demand-scope`. Decision: **Rule,
not TreeRule.** A file's rows depend on that file alone (its class-level fields and its methods), and the bash
lint already reported file by file in the sorted walk order. The bash argument contract (`[ "${1:-}" =
"--strict" ] && shift`: one optional leading flag, then raw roots) matched no engine mode, so the engine gained
`lint.Leading` in its own commit with a Parse table test. The port went byte-identical first, keeping the
naive first-`//` cut and the inert `\b` checks; then two false-result classes were fixed, each with a RED test,
and the header was corrected where it claimed more than the code did.

## Evidence
- RED: 37 golden cases (every tests/demand-in-scope.bats run on a tree except DS-smoke, which copies a client
  file, plus edge trees: 17 argument shapes, comments and literals, signature shapes, field spellings, CRLF,
  no final newline, non-ASCII bytes, walk order and pruning, unbalanced braces, process-variable spellings),
  recorded from origin/main's bash through a recording wrapper; the package did not exist, so all failed
  `[ev: go/internal/lint/rules/demandscope/demandscope_test.go]`
- Byte identity, old script (origin/main f56db17) vs the wrapper, stdout + stderr + exit: same on 9 argument
  shapes (no args, `-h`, `--`, `--strict --strict`, a trailing `--strict`, an extra root, a missing root, a file
  root, a relative root) and on Honeywell (±`--strict`), panccadia-leon, Leon-Guanjuato, all of Cliente
  (±`--strict`) and sources/decompiled `[ev: toolbelt/lint-demand-scope.sh]`
- Timing, one run each on a shared host (load 12-20): Cliente 2.5 s → 0.10 s, panccadia-leon 0.99 s → 0.06 s,
  Honeywell 0.71 s → 0.04 s, Leon-Guanjuato 0.29 s → 0.03 s, sources/decompiled 29.6 s → 4.1 s
- Declared difference (port): an unreadable .java file exits 3 with "cannot scan" on stderr (the bash lint
  printed awk's error and exited 0, or 1 under `--strict` with another WARN) `[ev: go/internal/lint/rules/demandscope/demandscope_test.go]`
- Fix 1 (TestCommentsAndLiterals RED 7/9 → GREEN): comments and literal contents are no longer code. A demand
  word in a block comment cleared the whole file, a brace in a literal shifted the depth, and a `"<%s>
  attempting"` format string was a temp comparison. Real trees: Cliente unchanged (0 rows), decompiled 6 → 5
  rows (fox-rt Tuner.java:379 `run reads temp`, a false row)
- Fix 2 (TestKeywords RED 3/5 → GREEN): gawk reads `\b` in a regex as a backspace, so `/\bclass\b/`,
  `/\binterface\b/`, `/\benum\b/` never matched (a nested `class Demand` cleared the file; a method on an enum's
  line was checked). They are whole-word matches now; no real-tree row changes. The fourth, `/\breturn[^;]*[<>]/`,
  is deleted rather than revived: as a word boundary it added 13 rows on the real trees, all false
- Header: it claimed rows name the demand term that cleared a method (rows only print an absence); it now lists
  the matched writes and names as the code matches them and the unchecked method shapes (multi-line parameter
  list, a later-line `{`, nested classes). No suppression marker is documented, so none was removed

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A gawk `\b` in a bash lint is a backspace, never a word boundary: read it as inert when porting.** gawk's word boundary is `\y`; `\b` in a regex is the backspace byte, so a check written `/\bword\b/` never matched and every recorded golden has it off. Keep it as the byte in the byte-identical port, then decide per check, with real-tree row counts, whether the intended word-boundary check is truer (here: yes for class/interface/enum, no for the return-comparison clause, 13 false rows). Reviewer check: a port that silently turns an awk `\b` into a Go `\b` changes output without a declared difference. | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/demandscope/demandscope.go]` |

## Lessons
- A dead clause found during a port is a decision, not a revival: measure what turning it on prints before
  making the code match its comment; deleting it can be the truthful fix.
- A bash lint that parses only `$1 = --strict` has its own argument contract (`--` and `-h` are roots); an
  engine mode that names it keeps byte identity without per-rule argument code.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-go-lint-demand-scope.md | kit | 2026-10-23 | pending | 1 |`
