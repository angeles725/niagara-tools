<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-11-16 · kit · kitlinks-prepush-t25

**Session**: row T25 of the 2026-11-15 coordination-stretch retro: niagara-tools #706 failed CI on kit-links L1
("dangling reference" to a bare doc name a ledger.d fragment wrote in prose), a failure the pre-push replay could not see.
**Delta count**: 2

## What happened
A ledger.d fragment is copied into the generated BUILD-STATE.md, and kit-links L1 scans the kit's root docs, so a bare
`<name>.md` in a fragment's prose is a dangling reference. `scripts/ci-local.sh --changed`, which `.githooks/pre-push`
runs, ran shellcheck, the bats lints and gofmt only, so the writer met L1 first in CI. `--changed` now runs
`tests/kit-links.bats` whenever the change touches a kit doc, a ledger.d or a changelog.d file, on a snapshot of the
change (the pushed commit, or the work tree) with the generated state rebuilt there; the checkout is never written.
To keep `scripts/ci-local.sh` under ~200 lines, its host-budget gate moved to `scripts/lib/ci-local-host.sh` in a
refactor-only commit.

## Evidence
- RED before GREEN: CL30 (a fragment naming a bare missing doc) exited 0 from `--changed` on origin/main.
  `[ev: tests/ci-local-kitlinks.bats]`
- Mutations in a scratch copy, not committed: dropping the `run_kit_links` call (CL30), rebuilding the state in the
  checkout instead of the snapshot (CL31) and running kit-links on every change (CL32) each turn their test red.
  `[ev: tests/ci-local-kitlinks.bats]`
- The module first went to a new `scripts/ci-local/` folder: CI's shellcheck globs do not cover it and the pre-push
  test fixtures copy only `scripts/lib/*.sh`, so it lives in `scripts/lib/`. `[ev: .github/workflows/ci.yml]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-11-16. `ci-local.sh --changed` runs kit-links on a snapshot of a change that touches a kit doc, a ledger.d or a changelog.d file | `scripts/lib/ci-local-kitlinks.sh`, `scripts/ci-local.sh` | `[ev: scripts/lib/ci-local-kitlinks.sh]` |
| Δ2 | APPLIED 2026-11-16. The host-budget gate of `ci-local.sh` lives in its own module (refactor only) | `scripts/lib/ci-local-host.sh` | `[ev: scripts/lib/ci-local-host.sh]` |
