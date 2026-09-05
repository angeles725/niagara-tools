<!-- review-status: folded -->
<!-- kit-retro -->
# run-pure-test.sh: `set -e` + a pipe aborts the empty-cache error path before it can report

Date: 2026-09-04 · Module: kit (toolbelt) · Tool: `toolbelt/run-pure-test.sh` (added PR #14, v0.8.0)

`run-pure-test.sh` wraps the WSL pure-JUnit recipe (resolve cached JUnit/Hamcrest via `find ~/.gradle`,
compile the pure class + test with Java 8, run `JUnitCore`). It is the executable coverage layer of the
4-layer QA stack now documented in `build-verify.md`.

## The lesson (proven RED-first by QA, then fixed)

With `set -e` (or `set -euo pipefail`) active, the empty-cache branch —
`JU=$(find ~/.gradle -name 'junit-4.13.2.jar' | head -1)` then `[ -z "$JU" ] && die 3 "..."` — aborted
the script BEFORE the intended `die 3` fetch-hint could print: the `find | head` pipeline and the failing
test guard tripped `set -e`, so the operator got a bare non-zero exit with no actionable message. QA's
red-first bats (`assert run-pure-test empty-cache RED path exits 3`) caught it; the fix makes the
empty-cache path report and exit `3` deterministically instead of letting `set -e` abort first.

## Proposed kit deltas (already applied in PR #14 + PR3)

- The tool exists at `toolbelt/run-pure-test.sh` and is pointed at from `build-verify.md` §Unit tests in
  WSL and the 4-layer assurance stack. (folded)
- Rule to keep: when a script uses `set -e` AND intends to REPORT a specific exit code on a guarded
  failure, isolate the probe from `set -e` (capture into a var without a failing pipe, or guard with
  `|| true`) so the human-facing `die <code> "<hint>"` runs instead of a bare abort.

## Self-verify

| Claim | Marker | Evidence |
|---|---|---|
| `set -e` aborted the empty-cache path before `die 3` | [CERT] | PR #14 commit `c1aecc3` "run-pure-test reports die 3 on empty cache (set -e was aborting first)" |
| The RED path is asserted | [CERT] | bats "assert run-pure-test empty-cache RED path exits 3 (P6)" |

Connections: [[2026-09-04-kit-continuity-and-retro-gate-campaign]]; `build-verify.md` §How you know it's good.
