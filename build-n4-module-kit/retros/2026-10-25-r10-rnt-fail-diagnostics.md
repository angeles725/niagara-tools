<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-rnt-fail-diagnostics

**Session**: niagara-tools quality round 10, kit tool G33 `run-niagara-test run` (D6), a redacted run log kept on a
failed run, branch `r10/rnt-fail-diagnostics` from ba3b7aa4 (origin/main)
**Delta count**: 1

## What happened
A failed `run` without `--keep-home` deleted the scratch dir, and with it test.exe's `out\stdout.txt`,
`out\stderr.txt` and its `-output:` dir; a rejection such as `invalid test target(s)` could not be explained after
the run, so a one-boot budget bought a verdict and no diagnosis. Each rule got a fake-runner RED first; no Windows
boot was run:
1. Cleanup reads the evidence before it deletes the dir: the last 200 stderr and 50 stdout lines of test.exe
   through a ring of lines (each cut at 4 KiB, a file read from at most its last 8 MiB, never a buffer sized from a
   file) and a listing of the `-output:` dir (names and sizes, 200 files at most).
2. The run log is written last, so it holds every row (the last 500, cleanup rows included) and the final exit
   status: `<UTC timestamp>-<module>.log`, mode 0600, under `--log-dir` (default `$XDG_STATE_HOME/run-niagara-test`,
   else `~/.local/state/run-niagara-test`, made 0700), and the row `FAIL  run-niagara-test  diagnostics kept:
   <path>`. Exit 1, 3 and 4 keep one; a green run only with an explicit `--log-dir` or `--keep-home` (`run log:
   <path>`). A log that was due and could not be written makes a green run exit 3.
3. Redaction before the write: the home copy reads `<copy>`, the scratch dir `<scratch>`, `%USERPROFILE%` and the
   WSL `$HOME` `<you>`, in Windows, forward-slash and `/mnt` forms, case-insensitively and only as a whole path
   component (a longer name sharing the prefix stays). Install paths stay: the user's own, needed to diagnose.

## Evidence
- the REDs, then green: `TestRunFailKeepsDiagnostics`, `TestRunFailExitsKeepDiagnostics`, `TestRunPassLog`, `TestRunDiagnosticsBoundLongLine`, `TestRedactMixedPathForms` `[ev: go/internal/niagaratest/run_diagnostics_test.go]`
- the line ring, the evidence read, the redactor and the log write `[ev: go/internal/niagaratest/diagnostics.go]`
- guard-drop mutations: no `(?i)` on the redactor, no forward-slash Windows form, and no path-component boundary each turned `TestRedactMixedPathForms` red; no per-line cap turned `TestRunDiagnosticsBoundLongLine` red

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A failed `run-niagara-test run` keeps a redacted run log (rows, stderr/stdout tails, `-output:` listing, exit status) under `--log-dir` before cleanup deletes the scratch dir; attach the log, not the home | `types/moduleTest.md` § Running from WSL | `[ev: go/internal/niagaratest/diagnostics.go]` |

## Lessons
- A cleanup that deletes the scratch dir must first copy out what explains a failure; otherwise the verdict survives and its cause does not.
- Write the diagnostic log last, from evidence read before the delete, so it carries the final exit code and every cleanup row.
