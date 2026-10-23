<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-23 · kit · go-lint-delays

**Session**: niagara-research quality round 8 unit E3a, branch `r8/e3-lint-delays-go` from origin/main 63c5b3b
**Delta count**: 2

## What happened
Decision first: **TreeRule, not Rule.** The bash lint ran one multi-file `xargs awk` over every file to build the
D2b registry of `static long` helpers proven to floor their result, then one awk per file that resolved
`BRelTime.make(Class.fn(..))` against that registry: a verdict in one file depends on another file, which a
per-file `lint.Rule` cannot see. `CheckTree` builds the registry over all files first and then reports file by
file, so rows keep the bash walk order. `toolbelt/lint-delays.sh` (509 lines of bash/awk) became the rule
package `go/internal/lint/rules/delays` and a thin `go_tool_bin` wrapper exec'ing `n4lint lint-delays`. Its
private awk `strip_comment()` (copies remain in `lint-arbitrary-ord.sh` and `lint-write-path.sh`) is now the
shared `javasrc.CutComments`, with table cases taken from the awk function's own output. The check column is
the bash `printf "%-14s"` of the lint name, carried as `Finding.Check`.

Byte identity first (commit ba3b78e), then three truth fixes, each with a RED seeded golden case:
- **T4** (truth audit 2026-10-22): the call site was matched on the raw line, so a `Clock.schedule(` in a
  comment, inside a block comment opened on an earlier line, or in a string literal FAILed; a string holding
  `"Clock.schedule("` before the real call on the same line hid that call behind a false `unclosed-call` row.
  Calls are now read from `javasrc.Strip` lines, skipping a match inside a literal.
- **Guard expression as text**: `find_guard` spliced the delay expression into a dynamic regexp. For
  `BRelTime.make((long) x)` gawk died on the file (`fatal: invalid regexp: Unmatched (`), the rest of the file
  was never checked and the lint still exited 0/1 — a fail-open; `x$y` never matched its own guard. The
  byte-identity commit failed closed (exit 3) as an interim declared difference; the fix quotes the expression.
- **Whole name in code**: the binding scan took the first raw line holding the variable name as a substring,
  comments included (`z` bound to a `// zero` comment: false literal-zero; `d` to a commented-out
  `Math.max(d, 1L)`: false clean), and `delayw > 0` guarded `w` (false PASS).

## Evidence
- RED/GREEN: 45 golden cases (23 bats runs of `tests/lint-delays.bats` — the real-tree LD5 and the chmod LD-locked
  excluded — 19 edge trees, 3 truth seeds); each truth seed failed before its fix `[ev: go/internal/lint/rules/delays/testdata]`
- Byte compare old vs new (stdout, stderr, exit): every bats fixture alone and together, the report-module
  fixture, 14 edge trees, 10 argument shapes (none, `-h`, `--`, missing, file root, empty string, `--strict`,
  two roots, `dir/.`, trailing slash), a symlinked root, every `Cliente` root, all of `Cliente`, every
  `Cliente/*-worktrees/*` checkout, `niagara-research/sources`, `modules-src` and `niagara5-research` (1312 rows):
  identical except the declared differences below `[ev: r8 e3 lint-delays differential]`
- Declared differences of the port: an unreadable `.java` is an env error (exit 3; bash printed awk's fatal and
  exited 0); a path with whitespace no longer loses its helpers to `xargs` word splitting (5 false FAILs in the
  test tree); find's quotes are LC_ALL=C `'..'` (engine-wide); under a UTF-8 locale gawk refused invalid bytes in
  bracket expressions and warned on stderr — the port reads bytes as LC_ALL=C gawk does (identical there)
- Real-tree row diff of the three truth fixes on all of `Cliente`, `niagara-research/sources` and
  `niagara5-research`: no row changes; the fixes show only on the seeded and edge trees `[ev: r8 e3 truth diff]`
- Empty root (after merging origin/main #351): the rule keeps the bash K20 row of `scan_files_nonempty_or_exit`
  through `Spec.EmptyError`, byte-identical to main's bash lint on an empty and a no-Java root; `edge-no-java`
  re-recorded (the first recording predates #351) and pinned by bats LD-empty `[ev: tests/lint-delays.bats]`
- Mutation LD1 in Go (`n <= 0` → `n < 0` for the Math.max floor) fails bats LD1, LD6, LD17; restored, 25/25
- Timing: all of `Cliente` bash 1.6-2.2 s → Go 0.04-0.11 s; `niagara5-research` bash ~13 min → Go 19.6 s

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Audit the awk lints for dynamic regexps built from source text** (`ln ~ expr "..."`): an expression that is not a valid ERE kills gawk on that file, the lint keeps its exit 0/1 and the rest of the file is never checked. Quote the text or match it with `index()`. | `toolbelt/lint-*.sh` awk bodies still in bash; `go/README.md` § Adding a lint rule | `[ev: retro go-lint-delays]` |
| Δ2 | **Port `lint-arbitrary-ord.sh` and `lint-write-path.sh` onto `javasrc.CutComments`** instead of their private `strip_comment` copies when they move to Go. | `go/internal/javasrc/cut.go` users | `[ev: retro go-lint-delays]` |

## Lessons
- A lint whose verdict in one file reads a registry built from every file is a TreeRule even when its report is
  ordered by file.
- A byte-identical port can surface a fail-open the bash lint hid (gawk dying mid-file with exit 0/1); fail closed
  in the identity commit and fix it in its own RED-first commit.
- The golden format cannot express an empty directory; an empty-root case needs a non-Java file in the tree.

---
**Status**: PENDING — INDEX row generated from the header: `| 2026-10-23-go-lint-delays.md | kit | 2026-10-23 | pending | 2 |`
