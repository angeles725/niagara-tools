<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-full-stderr

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run` (D6), the failed-run log keeps
test.exe's whole stderr, branch `r10/rnt-full-stderr` from 827b20e5 (origin/main)
**Delta count**: 1

## What happened
On a real failure the run log's 200-line stderr tail cut the station-log lines around the failing class: test.exe
writes the station log to stderr, and the tail kept only its end. #572 already kept testng-results.xml and the
traces beside the log; the stderr file went with the scratch dir. Each rule got a fake-runner RED first on generic
`ModuleA` fixtures; no station boot was run:
1. Cleanup reads `out/stderr.txt` with the rest of the evidence, before the delete: at most 4 MiB plus one byte,
   no buffer sized from the file.
2. On a failed run the log dir gets `<stem>.stderr.txt` beside `<stem>.log` (0600, created exclusively), named in
   the `kept evidence` section; a green run keeps none.
3. It reuses the #572 helpers: the log's own redactor (`redactorFor`), the line-end cap `capAtLine` with its
   `(truncated: ...)` row, and `writeExclusive`. No second redactor or capper.
4. A missing or unreadable stderr (a directory in its place included) is a `test.exe stderr unavailable: <why>`
   row; the log is still written and the run never crashes (it ends exit 3, as the console cross-check cannot read
   the file either).

## Evidence
- the REDs, then green: `TestRunFailKeepsFullStderr`, `TestRunFailFullStderrCapped`, `TestRunFailFullStderrUnavailable`, the updated sibling counts in `TestRunFailKeepsTestNGEvidence` and `TestRunFailTestNGUnavailable`, and the guard `TestRunPassKeepsNoTestNGEvidence` `[ev: go/internal/niagaratest/run_failevidence_test.go]`
- the read and the write `[ev: go/internal/niagaratest/failevidence.go]`
- mutation: dropping the `keepStderr` call from `writeRunLog` turned all five FAIL-run evidence tests red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A failed `run-niagara-test run` also keeps a redacted `<stem>.stderr.txt` (test.exe's whole stderr, the station log) beside its run log, cut at a line end under 4 MiB with a marker row; an unreadable stderr is a row in the log | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/failevidence.go]` |

## Lessons
- A line tail sized for the console is too short for the station log it carries: keep the whole stream, byte-capped, beside the log.
- A new kept-evidence file reuses the existing cap, redactor and exclusive write, so every kept file is cut and redacted the same way.

## Fold
- The stderr keep is code in the branch that wrote this retro (`go/internal/niagaratest/failevidence.go`).
- Δ1 is cited in `types/moduleTest.md` § Running from WSL (the full-stderr bullet). Owed: a live failed-run comparison.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-full-stderr`; log in `ledger.d/2026-10-25-r10-rnt-full-stderr.md`) — Δ1 in `types/moduleTest.md` § Running from WSL.
