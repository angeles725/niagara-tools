<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-run-niagara-test-go

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test` phase 1 (the verdict parser), branch
`r10/run-niagara-test-go` from origin/main bc8c5c09
**Delta count**: 2

## What happened
Spec C40 asks for a test runner that refuses silent passes: `test.exe` and gradle `niagaraTest` both exit 0 with
BUILD SUCCESSFUL on a run that ran no test or had its test module rejected. Phase 1 is the pure half, the verdict
on a captured log. It lives in `go/internal/niagaratest` (`Verdict`) and `go/cmd/run-niagara-test`
(`run-niagara-test verdict [<log>|-]`), and `toolbelt/run-niagara-test.sh` is the `go_tool_bin` wrapper.

- **Rule.** PASS only when the log holds at least one `Total tests run: N` line and every one has N > 0 and
  `Failures: 0`. A summary with 0 tests, failures or no Failures count fails. So do a `Failures: N>0` line outside
  a summary, `invalid test target(s)` and `failed signature validation` (matched in any case), a malformed number
  (`six`, `-1`, `6x`, past int32), no summary at all, and a line longer than 1 MiB. Skips are reported and never
  fail a run.
- **Bounded memory.** `bufio.Scanner` with `MaxLine` (1 MiB) as its token cap; nothing is sized from the input.
  A line past the cap stops the read, so it is a FAIL reason: a failure after it would otherwise go unseen.
- **Line shapes.** Taken from real runs (retros `testing-parity-btestng`, `r10-dr5-driver-exemplar`). The fixtures
  are synthetic logs built from those shapes; real anonymized logs are owed.
- **Tests.** One parser table (14 shapes and edges: CRLF, no Skips, a timestamp prefix, malformed numbers, a huge
  line) plus a read error. Nine CLI goldens through `golden.Runner` pin rows, usage, exits and unreadable logs,
  with no shape the table already holds, plus one stdin unit test. Two bats smoke runs prove the wrapper's exit is
  the verdict. RED was observed on a stub `Verdict` (14/14 table cases) and a stub `Main` (9/9 goldens, stdin).
- **Not done here.** `toolbelt/INDEX.md` has no row: it is generated from `lint-*.sh` headers only, and this is not
  a lint. Phase 2, the runner (scratch copy, `:clean`, `test.exe`, jar backup and restore), is owed.

## Evidence
- `go test ./internal/niagaratest` green: table, read error, 9 goldens, stdin `[ev: go/internal/niagaratest/verdict_test.go]`
- bats RNT1/RNT2 green; RNT2 fails when the wrapper drops the tool's exit code `[ev: tests/run-niagara-test.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A new Go tool with no bash source still uses golden.Runner: hand-written cases, no record.env, each `@kind` a `truth:` line saying so, and a testdata/README.md that names where the fixtures' shapes come from | `go/README.md` § Adding a lint rule (step 3) | `[ev: go/internal/niagaratest/testdata/README.md]` |
| Δ2 | A tool that judges a log fails closed on `bufio.ErrTooLong`: the line past the cap is a FAIL reason at its line number, never a silent end of input | `go/README.md` § Adding a lint rule (after the shared-code rule) | `[ev: go/internal/niagaratest/verdict.go]` |

## Lessons
- A verdict tool needs its own words for every way a log can be incomplete (no summary, a cut read, a malformed
  count); each one defaults to FAIL, because the tool exists to stop a silent pass.
