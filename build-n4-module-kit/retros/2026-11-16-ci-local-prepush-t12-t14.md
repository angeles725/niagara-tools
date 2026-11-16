<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 3 -->
# 2026-11-16 · kit · ci-local-prepush-t12-t14

**Session**: kit rows T12, T13 and T14 of the 2026-11-15 coordination-stretch retro: `ci-local.sh --only` of the
Go step, the pre-push and host-budget headers, and a cheaper shellcheck set for the pre-push CI lints.
**Delta count**: 3

## What happened
#696 shipped the pre-push CI-lints step (`scripts/ci-local.sh --changed --range`) with one known cost: any
changed shell file shellchecked CI's whole set (~285 files, ~30-43 s), because a script checked alone loses the
context of the lib it sources (a lone `ci-local.sh` read `CACHE` as unused, SC2034). shellcheck pairs a source
line such as `. "$ROOT/scripts/lib/x.sh"` with the command-line file whose path ends in the literal tail
`scripts/lib/x.sh`, so the change only needs its sourced and sourcing files beside it. Separately,
`--only "Go gofmt"` kept the Go step's `export KIT_REQUIRE_JS_TOOLS=1` without the npm step that installs acorn,
and the two headers still described the pre-#696 behaviour.

## Evidence
- RED before GREEN per row: CL23 (T12), PCL4 and HB8 (T13), and the cisteps closure and `changed` tests (T14)
  fail on the base. `[ev: tests/ci-local-pins.bats]` `[ev: go/internal/cisteps/sourced_test.go]`
- T14, measured with the pinned shellcheck 0.10.0: the closure of each of the 288 files of CI's set alone, one
  shellcheck run per closure, gives zero findings (no false SC2034), with no whole-set fallback; closures hold 1-6
  files except `go/go-tool.sh` (130, its sourcers). The whole set took 32.7 s on the same host.
  `[ev: go/internal/cisteps/sourced.go]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. T12: a Go step replayed without CI's npm step on a host where acorn cannot be resolved runs with `KIT_REQUIRE_JS_TOOLS` unset and a NOTE; with acorn, or the npm step selected, CI's setting stays | `scripts/ci-local.sh` header · `scripts/lib/ci-local-js.sh` | `[ev: scripts/lib/ci-local-js.sh]` |
| Δ2 | APPLIED 2026-11-16. T13: the `.githooks/pre-push` header lists the CI-lints step and the retro refusal's directories; `toolbelt/host-budget.sh` documents `--disk-pct`, `--disk-min-free` and the disk row; PCL4 and HB8 grep them | `.githooks/pre-push` · `toolbelt/host-budget.sh` | `[ev: tests/host-budget.bats]` |
| Δ3 | APPLIED 2026-11-16. T14: `ci-steps changed` shellchecks the changed files with the files they source (downward) and the files that source them (upward), transitively; CI's whole set only when a source line names no file of the set, with a stderr line naming it | Go `internal/cisteps` `sourced.go`, `subcommands.go` · `scripts/lib/ci-local-changed.sh` | `[ev: go/internal/cisteps/sourced.go]` |
