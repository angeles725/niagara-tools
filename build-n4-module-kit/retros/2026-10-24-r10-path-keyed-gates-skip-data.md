<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-24 · kit · r10-path-keyed-gates-skip-data

**Session**: niagara-research quality round 10 (`odd/tasks/quality-round10-ayudante1.md`), branch
`docs/r10-gate-data-retro` from origin/main 5415b69
**Delta count**: 2

## What happened
The shared niagara-research pre-commit hook chose what to run from the staged paths. When no staged path matched
its tool/code pattern it exited 0 at once, so a commit that staged only research data (corpus blocks,
`CATALOG.md`) never ran the block lint or `gen-catalog --check`. niagara-research PR #46 fixed it with an
optional data check that fails closed. The same round found that the two downstream copies of that hook had
already drifted, because the pin meant to keep them identical skips in exactly the checkouts where drift
happens.

This retro checked the kit's own gates for the same defect class before writing any delta. The kit pre-push
hook has it: a push that changes only `retros/`, `ledger.d/` or `BUILD-STATE.md` skips the content check that
validates exactly those files. CI runs that check on every push and PR, so nothing merges unchecked; the client
gate only gives late feedback. The kit has no identity pin of the drifting shape.

## Evidence
- niagara-research `tools/git-hooks/pre-commit` before PR #46 (`baa1276^1`): line 53
  `[ "$hits" -gt 0 ] || exit 0`, where `hits` counts staged paths matching
  `GATE_PATHS='^tools/(.+\.(py|sh)|tests/.+|git-hooks/.+)$'`. A data-only commit ran nothing.
  PR #46 (merge baa1276) adds `GATE_DATA_PATHS` + `GATE_DATA_CHECK` (word-split, never eval'd; paths set and
  matched with no check dies with `no data check configured for GATE_DATA_PATHS in <conf>: set
  GATE_DATA_CHECK`) and a thin `tools/gate-data-check.sh` over the existing lint-block and `gen-catalog.py
  --check`. RED first: 6 `git_hooks.bats` + 10 `gate-data-check.bats` cases `[ev: niagara-research PR #46 baa1276]`
- niagara-research `tools/tests/git_hooks_conf.bats:240-250`: the "installed: … runs the same hook" pins
  `cmp` the hook with `$REPO_ROOT/module-navigator/…` and `$REPO_ROOT/niagara-help/…` and `skip` when that
  nested checkout is absent, which is every worktree. Both copies lacked c052779ab (`GATE_JOBS` default 6, not
  4, and no validation) until module-navigator PR #11 (b8b07e0) and niagara-help PR #10 (cef80a4) synced them.
  The new pin compares with the upstream checkout's `origin/main`, not its working tree
  `[ev: module-navigator PR #11 b8b07e0, niagara-help PR #10 cef80a4]`
- Kit `.githooks/pre-push:44-50`: `relevant` drops `BUILD-STATE.md`, `ledger.d/` and `retros/`, then
  `[ -n "$relevant" ] || exit 0` with the comment "nothing build-relevant → nothing to gate". The content half it
  skips, `toolbelt/sweep-build-state.sh` (header lines 24-36), checks INDEX integrity, retro marker vs INDEX row,
  generated-index freshness and ledger freshness: all properties of those three excluded paths
  `[ev: tests/build-retro-sync.bats H6]`
- Observed in a scratch fixture (the `_fixture_top` layout of `tests/build-retro-sync.bats`): a retro flipped
  to `folded` and given a new delta count, INDEX not regenerated, range = that retro only. The hook exited 0;
  `sweep-build-state.sh` on the same tree exited 1 with `retro marker disagrees with INDEX row:
  2026-09-04-demo.md marker='folded' INDEX='pending'` `[ev: tests/build-retro-sync.bats H6]`
- `tests/build-retro-sync.bats:232` H6 "an EXCLUDED kit path (retros/ only) does NOT fire the gate" pins the
  skip as intended; it stages no stale index, so it cannot tell "no check applies" from "a check applies and was
  skipped" `[ev: tests/build-retro-sync.bats H6]`
- Backstop: `.github/workflows/ci.yml:103-106` (sweep-build-state), `:121-122` (gen-retro-index `--check`) and
  `:124-125` (gen-build-state `--check`) run on every push to main and every PR with no path filter.
  `CONTRIBUTING.md` § 6.1 says "CI and the gate's content half (`sweep-build-state.sh`) fail a stale index",
  which is true for CI only when the range holds no build-relevant file `[ev: .github/workflows/ci.yml:103-125]`
- Other kit gates checked, no defect: `scripts/ci-local.sh:44-51` skips CI steps by step NAME with a stated
  local equivalent, and stops on an UNCOVERED step (`:13-14`), never by changed path; `sweep-build-state.sh` and
  `sweep-fold-audit.sh` take no path list; the kit has no pre-commit hook. The pre-push filter is correct for
  `types/*.md` and fixtures, which stay build-relevant (`:45` keeps every `build-n4-module-kit/` path not
  excluded). Kit identity pins (`tests/install-skill.bats` IS1/SK1, `tests/install-client-hooks.bats` ICH*,
  `tests/check-skill-drift.bats` SD1) compare in-repo sources with temp installs, so they never skip on a
  missing copy `[ev: scripts/ci-local.sh:44-51]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| 1 | **A path filter in a gate skips only paths no check applies to.** A gate that picks checks from changed paths states, for each class of path it can see (code, tool, doc, data, generated), which check runs or why none applies. A filter that exits early is proven by a test that changes ONLY a skipped class with a defect the gate's checks would catch, and asserts the outcome. A skip test that changes a clean file proves nothing. A data class that has a check gets one that fails closed: data paths configured with no check is an error, not a pass (niagara-research PR #46). Kit instance: `.githooks/pre-push:44-50` skips `sweep-build-state.sh` for a `retros/`/`ledger.d/`/`BUILD-STATE.md`-only range; CI backstops it; the hook fix is owed. Reviewer check: the PR names the data-only test and shows it red on the old filter. | `BUILD-LOOP.md` § 5 kit tooling doctrine; `CONTRIBUTING.md` § 6.1 | `[ev: tests/build-retro-sync.bats H6]` |
| 2 | **An identity pin between shared copies does not skip where drift happens.** A test that keeps two repositories' copies byte-identical compares with the upstream's committed `origin/main` (`git show origin/main:<path>`), not a sibling working tree, and skips only when the upstream repository is really absent, naming what it looked for. A pin that skips in every worktree or CI checkout protects only the one main checkout. Reviewer check: the pin fails in a scratch layout with an old copy and the drift message names the sync command. | `BUILD-LOOP.md` § 5 kit tooling doctrine | `[ev: module-navigator PR #11 b8b07e0, niagara-help PR #10 cef80a4]` |

## Lessons
- "Nothing to gate" is a claim about every check, not about the pattern the gate was first written for.
  List the checks and the paths each one reads before writing the early exit.
- A test that pins a skip must stage a defect in the skipped class, or it pins the bug.
- Drift is found in worktrees and CI, so a pin that skips there gives no protection where it is needed.

---
**Status**: PENDING 2026-10-24 — 2 deltas proposed; fold in the same branch.
