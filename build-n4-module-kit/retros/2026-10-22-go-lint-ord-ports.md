<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · go-lint-ord-ports

**Session**: niagara-research quality round 8, work unit T8k-4, branch `r8/t8k4-set-null-ord` on the engine commit 0dd2ffa (stop on walk error by default)
**Delta count**: 2

## What happened
`toolbelt/lint-set-null-ord.sh` and `toolbelt/lint-wb-external-ord-value.sh` were `awk_lint_run` lints: one
awk process per Java file over the awk method-boundary lib. They are now the per-file rule packages
`go/internal/lint/rules/setnullord` and `go/internal/lint/rules/extordvalue` on `go/internal/javasrc`
(the Go port of mb_strip/mb_parse), and the scripts are wrappers that exec `n4lint <rule>`. Both rules set
`Spec.ContinueOnWalkError` because `awk_lint_run` scans the files it did list after a walk error. The two lints share
no check logic, so no rule helper was added; the golden-case runner became the test helper
`go/internal/lint/rules/golden` (`Run` for the recorded cases, `Unreadable` for the awk_lint_run failure contract).

## Evidence
- RED: 48 golden cases (32 for lint-set-null-ord, 16 for lint-wb-external-ord-value: every bats run recorded from the
  old script through a recording wrapper, plus 22 edge trees) failed 26 of 48 against a stub Check; the 22 others expect
  no rows or test the argument contract `[ev: go/internal/lint/rules/setnullord/setnullord_test.go]`
- Byte identity, old script vs wrapper, stdout + stderr + exit: same on tests/fixtures, Cliente/Honeywell,
  Cliente/panccadia-leon, Cliente/Leon-Guanjuato, Cliente/LLM, Cliente/ford, an 81-file sample of decompiled Niagara
  and Optimizer sources that call getSlotPathOrd() or resolve ORDs (17 and 3 rows), an unreadable sub-directory plus an
  unreadable file (plain and newline names), a path with a space, a non-UTF-8 file, no args, `-h`, a missing root, a file
  root, two roots, `--strict` and `--`
- Timing, one run each, lint-set-null-ord / lint-wb-external-ord-value: Honeywell 1.24 s → 0.11 s / 1.20 s → 0.10 s,
  panccadia-leon 4.48 s → 0.18 s / 1.82 s → 0.10 s, Leon-Guanjuato 0.74 s → 0.06 s / 0.40 s → 0.03 s, the 81-file sample
  6.27 s → 0.78 s / 6.11 s → 0.49 s
- Intended difference: a path holding a backslash. `awk -v FILE="$f"` escape-processes the value, so the bash rows
  printed `a<TAB>b/A.java` for a directory named `a\tb`; the rules print the real path
- Engine-owned difference (reported): with two unreadable sub-directories the env line names the first in sorted order,
  where find names the first in directory-read order; one unreadable directory is byte-identical
- Mutation spot checks: a receiver-agnostic guard, tracking the assigned local on its own line, dropping the
  registerForComponentEvents( check, an unanchored `ord` suffix and dropping ContinueOnWalkError each fail a test;
  dropping `Longest()` fails none (an equivalent mutant for these patterns, kept for gawk fidelity)

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **Pass a file path to awk through ENVIRON, never `-v`, in `awk_lint_run`.** `-v FILE="$f"` escape-processes the value, so a path with a backslash prints wrong in every row; `awk_lint_each` already uses `ENVIRON["AL_FILE"]`. Why: the bash lints still on `awk_lint_run` print a corrupted path for a directory named `a\tb`. Reviewer check: a new `awk -v` carrying a path is a finding. | `toolbelt/lib/awk-lint.sh` § awk_lint_run | `[ev: build-n4-module-kit/toolbelt/lib/awk-lint.sh]` |
| Δ2 | **Port a lint's golden cases with `rules/golden`.** Record every bats run of the old script (args, exit, stdout, stderr, tree, `(noeol)` files) and call `golden.Run`; an `awk_lint_run` port also calls `golden.Unreadable`. Why: two ports reached byte identity on the first full run with recorded rows, and the shared runner removed a per-rule copy of the parser. Reviewer check: a port that restates bats assertions as substrings, or copies the golden parser into its package, is a finding. | `go/README.md` § Adding a lint rule | `[ev: go/internal/lint/rules/golden/golden.go]` |

## Lessons
- An `awk_lint_run` port must set `ContinueOnWalkError`; a `scan_files_or_exit` port keeps the default stop.
- `awk -v` rewrites backslashes in the value: compare paths with a backslash when porting an awk lint.
- An equivalent mutant (no observable change) is a fact to state, not a missing test.
- lint-guard-pins reads only header lines 1-60 (`guardpins.HeaderLines`): a `# Mutation:` line pushed past line 60
  by a longer header is dropped without a finding (WEO-finderr was, until the wrapper paragraph was shortened).

---
**Status**: FOLDED (quality round 8 pointer unit, `docs/r8-pointer-lines`) — Δ1 ab15c4c (PR #354) `toolbelt/lib/awk-lint.sh` awk_lint_run via ENVIRON + `tests/awk-lint.bats` AL24, Δ2 `go/README.md:86-94` § Adding a lint rule step 3; tokens in `ledger.d/2026-10-23-r8-pointer-lines.md`. INDEX row generated from the header: `| 2026-10-22-go-lint-ord-ports.md | kit | 2026-10-22 | folded | 2 |`
