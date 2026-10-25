<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-25 · kit · r10-changelog-gate

**Session**: niagara-research quality round 10, branch `r10/changelog-gate` from origin/main
**Delta count**: 1

## What happened
The v0.33.0 release found `CHANGELOG.md` `## [Unreleased]` empty: no PR since v0.32.1 had added an entry, and the
notes were rebuilt from the merge log. Nothing checked for an entry. The new `scripts/changelog-gate.sh`
(`go/internal/changelog`) fails a range that changes kit behaviour (`go/`, `build-n4-module-kit/`, `tools/`, `scripts/`, minus
the retro gate's own bookkeeping exclusions) unless it adds a `changelog.d/` fragment, adds a non-blank line inside
`## [Unreleased]`, or carries a `Changelog: none (<reason>)` trailer. Fragments follow the `ledger.d/` precedent:
parallel PRs each add their own file instead of editing the same section. It runs in `.githooks/pre-push` and, unlike
the retro half of the hook, in CI too (a new `changelog` job over `origin/main..HEAD` with full history). The
release folds the fragments into `## [Unreleased]` with `scripts/gen-changelog.sh --write` and deletes them; CI runs
its `--check`, so a fragment the fold would refuse never merges and no fragment is left out of a release.

## Evidence
- RED: all 17 cases of `tests/changelog-gate.bats` failed on origin/main 6de4a657 (no gate script)
  `[ev: tests/changelog-gate.bats]`
- Mutation CG-c: accepting any `CHANGELOG.md` change fails CG-c (an older-section edit passed)
  `[ev: tests/changelog-gate.bats]`
- RED: all 9 cases of `tests/gen-changelog.bats` failed before `scripts/gen-changelog.sh` existed; mutation GC1
  (fold right after the header) fails GC1 `[ev: tests/gen-changelog.bats]`
- The retro-gate hook tests fake `git log`; their fake now also prints a `Changelog: none` trailer so each test
  still reaches the retro logic it pins `[ev: tests/build-retro-sync.bats]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A kit behaviour change carries its CHANGELOG entry in the same range, as a `changelog.d/` fragment, an `[Unreleased]` line or a reasoned `Changelog: none` trailer**, checked by the pre-push hook and by CI, so release notes are never rebuilt from the merge log. | `CONTRIBUTING.md` § 6.2 | CG-a..CG-k |

## Follow-ups
- `tools/` does not exist at the repository root today; the gate counts it so a future top-level `tools/` is covered.
- `scripts/` counts as a behaviour root too, for parity with the retro gate, which treats it as build-relevant
  (CG-f4).
