<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · ci-local-replay

**Session**: niagara-research quality round 8, task T8g, branch `r8/t8g-ci-local` from origin/main 409fd2e
**Delta count**: 1

## What happened
Local checks and CI disagreed in two ways. The dev WSL has shellcheck 0.11.0 (Homebrew) while `ci.yml` pins
0.10.0, so PR #273 passed locally and failed CI on SC2015, and every session excluded SC2329 (a 0.11-only check) by
hand. A writer also saw a local failure that CI would not have produced, after adding pipefail: GitHub runs a
`run:` step with `bash -e {0}`, which sets errexit but not pipefail. This unit adds `scripts/ci-local.sh`. It reads
the steps from `ci.yml` (`scripts/lib/ci_steps.py`, stdlib only), runs each `run:` step with `bash -e -c` from the
repo root (CI=true, RUNNER_TEMP set), and replaces the install steps with the pinned shellcheck v0.10.0 from a
sha256-checked cache. A step it can neither run nor skip on purpose stops it with exit 2.

## Evidence
- `tests/ci-local.bats` RED 0/9 with no script, GREEN 9/9; mutations of the errexit-only shell (CL6), the cached-hash
  check (CL5), the `run: |` parser (CL1) and the sudo/apt-get guard (CL2) each turn their test red `[ev: tests/ci-local.bats]`
- The first run of the replay's shellcheck step on its own script reported SC2015 at `-j` parsing, the same 0.10.0
  finding that failed PR #273 `[ev: scripts/ci-local.sh]`
- Pinned sha256 `6c881ab0…edf87` computed from a download of the v0.10.0 linux x86_64 release tarball (2,404,716
  bytes); the release has no published digest `[ev: scripts/ci-local.sh]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Before pushing a change to scripts, toolbelt or tests, run `scripts/ci-local.sh -j 3` instead of the host's shellcheck and bats alone; it uses CI's tool pins and `bash -e` step semantics, and `--list` shows which CI steps run locally | `CONTRIBUTING.md` § 6 Pre-commit checklist | `[ev: retro ci-local-replay]` |

## Lessons
- A local gate that re-types the CI commands drifts; derive the steps from `ci.yml` and fail on any step that is
  neither run nor skipped on purpose.
- A tool pin only protects CI. A local run needs the same pinned binary, checked by hash on every use.

---
**Status**: FOLDED (quality round 8 W23 U0, main e66da0a) — Δ1 `CONTRIBUTING.md:193` § 6 Pre-commit checklist; token in `ledger.d/2026-10-23-r8-u0-retro-flips.md`. INDEX row generated from the header: `| 2026-10-22-ci-local-replay.md | kit | 2026-10-22 | folded | 1 |`
