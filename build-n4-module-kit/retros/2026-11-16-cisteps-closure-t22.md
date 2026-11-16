<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 4 -->
# 2026-11-16 · kit · cisteps-closure-t22

**Session**: kit row T22 of the 2026-11-15 coordination-stretch retro: the native-review advisories of #701 (the
pre-push shellcheck closure and the CL23 test).
**Delta count**: 4

## What happened
#701 replaced the pre-push whole-set shellcheck (40 s) with the closure of the changed files (4.5 s). Its review
approved it with advisories. The parser read a `.`/`source` command only at the start of a line, so a guarded
`[ -f x ] && . x` was invisible. The whole-set fallback looked only at lines inside the closure, yet an unresolved
line anywhere might be the only sourcer of a changed lib. A comment between a `# shellcheck source=` directive
and its command dropped the directive. A changed path the set does not hold was seeded into the closure. And
CL23 read the host's `KIT_ACORN`/`KIT_NODE`. CI still shellchecks the whole set, so these affected only the local
pre-push step.

## Evidence
- RED before GREEN for each item: the compound, directive, set-wide-fallback and deleted-path cases of
  `sourced_t22_test.go`, and CL23 run with `KIT_ACORN` and `KIT_NODE` exported. `[ev: go/internal/cisteps/sourced_t22_test.go]`
- Mutation: disabling the set-wide fallback turns three closure tests red; HB9 turns red when the `--disk-pct`
  default moves from 90. `[ev: go/internal/cisteps/sourced.go]`
- The real set, with the pinned shellcheck 0.10.0: 130 source references, no fallback, and the closure of each of the
  288 files alone gives zero findings (go/go-tool.sh's closure now holds 131 files: a compound sourcer was found).

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. A `.`/`source` command after a separator or keyword is a source line; a source line anywhere in CI's shellcheck set that names no file of the set checks the whole set | Go `internal/cisteps` `sourceparse.go`, `sourced.go` | `[ev: go/internal/cisteps/sourceparse.go]` |
| Δ2 | APPLIED 2026-11-16. A `# shellcheck source=` directive holds over comment and blank lines; a changed path the set does not hold is left out of the closure | Go `internal/cisteps` `sourceparse.go`, `sourced.go` | `[ev: go/internal/cisteps/sourced.go]` |
| Δ3 | APPLIED 2026-11-16. CL23 runs the replay with `KIT_ACORN`, `KIT_NODE` and `KIT_REQUIRE_JS_TOOLS` cleared and asserts the full replay's status; HB9 pins the `--disk-pct` default on the command line | `tests/ci-local-pins.bats` · `tests/host-budget.bats` | `[ev: tests/ci-local-pins.bats]` |
| Δ4 | DECLINED 2026-11-16. A behavioural test behind PCL4: the skip line it documents is already asserted by `go/internal/prepush/cilints_test.go:58`. Also declined: reading the set through one `git cat-file --batch` (the reads cost about 2 s, under the 10 s target) and a NOTE repeated in ci-local's summary (outside this row's files) | `tests/pre-push-ci-lints.bats` | `[ev: go/internal/prepush/cilints_test.go]` |
