<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-bats-list-hazard

**Session**: niagara-tools quality round 10, find and fix bats assertions errexit cannot see, then lint them; branch
`r10/bats-list-hazard` from 63b1c44f (origin/main)
**Delta count**: 1

## What happened
A mutation run of another round-10 lint stayed green because its bats test asserted
`[ "$status" -eq 1 ] && [[ "$output" == *x* ]]` in the middle of the test body. bats runs a test body under errexit,
and bash ignores errexit for every command of an `&&` list except the last one, and for a command negated with `!`:
such a line never fails its test. Only the status of the test's last command counts.

1. Read-only sweep (a throwaway Go scanner with a small bash lexer and list parser) over all 140 `*.bats` of the
   repository: 47 mid-test `&&` lists of assertions in 17 `tests/*.bats` files, 0 bare mid-test `! cmd` (the one
   `! grep` line, `tests/ng-deploy.bats` T22, is the last command of the test's last `if`, so it counts), 5 guards in
   test bodies and 7 in helper functions whose last element is an action (`[ -n "$p" ] && ln -s "$p" d`, `&& break`,
   `&& printf`), left as they are. `tests/helpers/*.bash` and `tests/lib/*.bash`: no hit.
2. The 47 lists were split mechanically, one assertion per line (commit `test(bats): split mid-test && assertions so
   each can fail`). No split turned a test red: the 17 files gave 489 ok and the same 23 skips before and after.
3. `lint-bats-errexit` (`go/internal/lint/rules/batserrexit`, wrapper `toolbelt/lint-bats-errexit.sh`) FAILs both
   forms in `@test` bodies, and `tests/lint-bats-errexit.bats` BEX-kit runs it over the kit's own `tests/`, so CI's bats
   job fails a new occurrence without a CI edit.

Semantics the rule encodes, each observed with bats 1.14.0 on a 3-line fixture: a mid-test `[ a ] && [ b ]` and a
mid-test `! true` pass; `[ a ] || [ b ]` with `[ b ]` false fails; the last line's `&&` list or `!` fails; the last
`&&` list inside an `if`, a `for` or a `{ }` group in mid-test passes (errexit does not fire on the compound's
status); a standalone `( [ a ] && [ b ] )` fails (the subshell's exit status is a command of its own).

## Evidence
- Sweep: 47 lists before, 0 after; `n4lint lint-bats-errexit` gives the same 47 rows on an origin/main copy of
  `tests/` and none on the branch.
- RED (wrapper absent): `bats tests/lint-bats-errexit.bats` 6 of 7 not ok (BEX0, the hazard proof, is green by design);
  `go test ./internal/lint/rules/batserrexit` failed to build (undefined: Spec). GREEN: 7/7 ok; Go cases ok.
- Mutations: BEX1 (classify accepts `&&` lists) -> not ok BEX1 and Go `TestCases` 8 cases FAIL; BEX2 (accepts a bare
  negation) -> not ok BEX2; BEX3 (flags the last command and `||` tails) -> not ok BEX3 and BEX-kit.
- Unverified: `tests/c8-close.bats:141`, split in the C8_CLOSE-gated SC16 smoke; that test needs a pinned client
  station bog and fails at line 137, before the split, on a newer station bog. `tests/lint-vendor-floor.bats` (four
  splits in acorn-gated tests) is 11/11 ok with `KIT_ACORN` set.

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A bats assertion is one command on its own line, never an earlier `&&` element or a bare `! cmd` (errexit sees neither); `toolbelt/lint-bats-errexit.sh` FAILs both and BEX-kit runs it on the kit's `tests/` | `build-verify.md` § Toolbelt authoring; `skill/SKILL.md` Kit tooling | `[ev: code go/internal/lint/rules/batserrexit/batserrexit.go]` |

## Lessons
- A test that never fails is found by making it fail: a guard-drop mutation that leaves the suite green is the signal,
  and the cause can be the assertion's shape, not the fixture.
- Helper functions are out of the lint's scope: a mid-function `&&` list of assertions is the same hazard when a test
  calls the helper as a plain command, and the whole helper runs with errexit off when it is called from a condition.
  The sweep found none today.

## Fold
- Δ1 is cited in `build-verify.md` § Toolbelt authoring (the bats-assertion bullet) and routed in `skill/SKILL.md`.
- Follow-up, not done here: a named `lint-bats-errexit` step in `.github/workflows/ci.yml` (the CI lints are separate
  named steps, not a list one line extends; BEX-kit already runs the lint in the bats job).

---
**Status**: FOLDED 2026-10-25 (branch `r10/bats-list-hazard`; log in `ledger.d/2026-10-25-r10-bats-list-hazard.md`) — Δ1 in `build-verify.md` § Toolbelt authoring.
