<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-lint-engine-bool-flags

**Session**: niagara-research quality round 10, branch `r10/lint-engine-bool-flags` from origin/main
**Delta count**: 2

## What happened
Mode Valued took value options only, so `lint-private-api-import.sh` (G65) translated `--no-default-allow` into the
engine option `--default-allow off` with a bash loop that had to skip option values
`[ev: retro r10-dr3-driver-lints Δ1]`. The engine now declares boolean flags (`Spec.Flags`, `Args.Flags`), so the
wrapper is the plain `exec n4lint` its siblings are and `--default-allow` is gone (only the G65 goldens used it).

## Deltas
| Δ | Change | Where | Evidence |
|---|---|---|---|
| Δ1 | Mode Valued boolean flags: anywhere in argv, repeatable, read through `Configure` as `Args.Flags`; `<flag>=<v>`, an unknown flag and `--` stay usage errors; a flag in a value position is that option's value. Existing rules unchanged (goldens and bats untouched). | `go/internal/lint/args.go`, `lint.go`, `run.go`; `go/README.md` | `TestParseValuedFlags` |
| Δ2 | G65 declares `--no-default-allow` natively; the wrapper's translation loop and the `--default-allow on\|off` option are removed; usage `[--strict] [--allow <glob>]... [--no-default-allow] <module-root>`. | `go/internal/lint/rules/privateapiimport`, `toolbelt/lint-private-api-import.sh` | goldens `no-default-allow`, `flag-value`, `usage`; bats PAI-flag |

**Status**: FOLDED 2026-10-24 — Δ1 in `go/README.md` (Mode Valued), Δ2 in the wrapper header (its user-facing contract).
