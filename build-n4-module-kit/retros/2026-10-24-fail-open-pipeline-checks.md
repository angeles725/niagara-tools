<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · fail-open-pipeline-checks

**Session**: niagara-research quality round 10 (Ayudante2, Go lint-engine owner), branch `r10/slotomatic-markers-go` from origin/main 45ddb3b5
**Delta count**: 2

## What happened
The round's lander (the parent session's own pre-push script, outside this repo) gated every candidate with two
checks that cannot fail when their producer does not run:

- `(cd go && go build ./... && go vet ./... && go test ./... 2>&1 | grep -E '^(FAIL|---)') && { echo FAIL; exit 1; }`
  — the group's status is grep's. When `go build` or `go vet` fails, `go test` never runs, grep reads nothing and
  exits 1, so the group "fails" and the FAIL branch is skipped: a broken build passes the gate.
- `[ -z "$(gofmt -l go)" ]` — on a syntax error gofmt prints the error on stderr and lists nothing on stdout, so
  the test passes.

PR #409 was pushed through both with a broken `go/internal/lint/lint_test.go`: an automatic keep-both conflict
resolution had cut two test functions, leaving a file that did not compile.

The fix idioms derive the verdict from the producer's exit status:
`(cd go && go build ./... && go vet ./... && go test ./... >"$log" 2>&1) || fail` and
`fmt=$(gofmt -l go 2>&1) && [ -z "$fmt" ] || fail`.

A ratchet landed with this retro: `tests/fail-open-idioms.bats` scans `scripts/`, `.githooks/` and
`build-n4-module-kit/toolbelt/**/*.sh` for both shapes (GREP-GATE: a parenthesized group ending in `| grep …`
whose status is used with `&& … exit|fail|return`, quoted text blanked; FMT-STDERR: `$(gofmt|shfmt -l …)` with no
`2>` inside), against `tests/fixtures/fail-open-idioms/baseline.txt` (two-way, empty when it landed).

## Evidence
- The lander commands and the PR #409 breakage are the parent session's report of its own lander (not in this
  repository) `[ev: retro fail-open-pipeline-checks]`
- Seeded fixture `tests/fixtures/fail-open-idioms/planted.sh` (both lander lines plus two variants): 4 rows;
  `fixed.sh` (the fix idioms, a printf-fed grep, the shape inside a string and a comment): 0 rows (FOI1)
  `[ev: tests/fail-open-idioms.bats]`
- The kit tree at 45ddb3b5: 89 files scanned, 0 hits, so the baseline is empty. Observed mutations: treating
  every `$(gofmt -l …)` as unsafe or not blanking double-quoted text fails FOI1; appending
  `[ -z "$(gofmt -l go)" ] || exit 1` to `toolbelt/lint-size.sh` fails FOI2 (`FMT-STDERR …/lint-size.sh:129`)
- Out of the scan: `.github/workflows/ci.yml:77` `unformatted="$(gofmt -l .)"` runs under the Actions default
  `bash -e -o pipefail`, where the assignment carries gofmt's exit 2 and fails the step, so it is safe as written

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A check derives pass/fail from the exit status of the producer, never from grepping a pipeline whose producer may not run, and captures a formatter's stderr with its list (`2>&1`); `tests/fail-open-idioms.bats` is the ratchet | `METHODOLOGY.md` (new K rule beside K20) | `[ev: tests/fail-open-idioms.bats]` |
| Δ2 | A keep-both conflict resolution applies only to hunks where both sides add whole lines; a test file or an edited line needs a human merge, then a compile and test run before the commit | `ORCHESTRATION.md` § merge rules (beside the fragment-merge rule) | `[ev: retro fail-open-pipeline-checks]` |

## Fold
- Δ1 → `METHODOLOGY.md` K20 sub-rule (producer exit); `tests/fail-open-idioms.bats` landed with this retro.
- Δ2 → `ORCHESTRATION.md`, beside the fragment-merge rule (keep-both only for whole-line additions).

## Lessons
- A gate is proven only by watching it fail on the input it exists for: neither lander line was ever run against a
  broken build before PR #409 went out.
- grep's status answers "was the text printed", not "did the step succeed"; the two coincide only when the
  producer is guaranteed to run and to print on failure.

---
**Status**: FOLDED 2026-10-24 (branch `docs/r10-fold-meta`) — INDEX row: `| 2026-10-24-fail-open-pipeline-checks.md | kit | 2026-10-24 | folded | 2 |`
