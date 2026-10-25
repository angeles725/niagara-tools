<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-fail-evidence

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run` (D6), the failed-run log keeps
testng-results.xml and every failed method's whole trace, branch `r10/rnt-fail-evidence` from 9db7f4e9 (origin/main)
**Delta count**: 1

## What happened
A real failed run printed 463 stderr lines; its run log kept the last 200, which started inside one failing
method's stack trace, and testng-results.xml went with the scratch dir. That file already holds every FAIL
`test-method` with its `<exception>`, `<message>` and `<full-stacktrace>`, so the traces come from there. Each rule
got a fake-runner RED first on generic `ModuleA` fixtures; no station boot was run:
1. Cleanup reads `out/testng/testng-results.xml` with the rest of the evidence, before the delete: at most 4 MiB
   plus one byte for the copy, and the traces through a streaming decoder over at most the verdict's 64 MiB, the
   traces text itself capped at 4 MiB. No buffer is sized from the file.
2. On a failed run the log dir gets `<stem>.testng-results.xml` and `<stem>.traces.txt` beside `<stem>.log` (0600,
   created exclusively), named in a `kept evidence` section of the log; a green run keeps neither.
3. Both files go through the log's own redactor (`redactorFor`), not a second one. A cut over the cap lands on a
   line end with a `(truncated: ...)` row, so the redactor never sees half a path.
4. A missing or unreadable results file (a directory in its place included) is a `testng-results.xml unavailable:
   <why>` row; the log is still written and the run never crashes.

## Evidence
- the REDs, then green: `TestRunFailKeepsTestNGEvidence`, `TestRunFailTestNGUnavailable`, `TestRunFailTestNGEvidenceCapped`, and the guard `TestRunPassKeepsNoTestNGEvidence` `[ev: go/internal/niagaratest/run_failevidence_test.go]`
- the read, the traces, the cap and the write `[ev: go/internal/niagaratest/failevidence.go]`
- mutation: dropping the `keepTestNG` call from `writeRunLog` turned the three FAIL-run tests red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A failed `run-niagara-test run` also keeps a redacted `<stem>.testng-results.xml` copy and `<stem>.traces.txt` (every FAIL test-method's whole `<full-stacktrace>`) beside its run log, each cut at a line end under 4 MiB with a marker row; an unreadable results file is a row in the log | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/failevidence.go]` |

## Lessons
- A tail of the console is not the failure: take the traces from the structured results file, which holds each one whole.
- Cap redacted text at a line end, never mid-line: a cut inside a path leaves a prefix the whole-component redactor cannot match.

## Fold
- The evidence keep is code in the branch that wrote this retro (`go/internal/niagaratest/failevidence.go`).
- Δ1 is cited in `types/moduleTest.md` § Running from WSL (the testng-evidence bullet). Nothing owed.

---
**Status**: FOLDED 2026-10-25 (branch `r10/rnt-fail-evidence`; log in `ledger.d/2026-10-25-r10-rnt-fail-evidence.md`) — Δ1 in `types/moduleTest.md` § Running from WSL.
