# mcp-n4-issue179-hygiene — close issue #179 and the post-v0.26.0 usage friction in mcp-n4-kit

## Objective
Close the open findings of GitHub issue #179 (T6e+T8b review advisory + B1200 gaps) and the friction
reported by the 2026-10-02 remote inventory session that v0.26.0 did not encode yet.

## Problem / Why
- `write.check_state_files()` runs twice (R1-001/R2-001/R3-001/R4-003).
- `tools/new_retro.py` rewrites the retro file and INDEX.md through `mkstemp` and leaves them mode 0600 (R3-003/R4-004).
- `box.py` has an ambiguous `or [] if isinstance` expression (R2-002); `retro.py` builds `by_batch` twice (R2-003).
- `BoxClient._loads_outstanding` never decrements when a load times out or errors, so a session ends "untrusted" (R4-001).
- Rollback does not restore frozen-child config (proxyExt) and `_snapshot` drops link-targeted slots (R4-002, R3-002, B1200-G1/G2/G3).
- Usage friction: `n4_navigate` has no `types` filter; long-running clients must report progress to a file;
  the auto-mode permission classifier can block live writes and agents must surface that, never route around it.

## Scope
- In scope: `mcp-n4-kit/**`, `CHANGELOG.md`, `VERSION`, this document.
- Out of scope: build-n4-module-kit, installed skill copies under `~/.claude/skills`, any write to a live station.
- The repository is PUBLIC: no customer, station, host, IP or user names.

## Constraints
- Python 3.10+, stdlib only, `unittest`. Test-first (RED then GREEN) wherever a deterministic test exists.
- Runner: `python3 -m unittest discover -s mcp-n4-kit/tests -v` from the repo root.
- Delivery strategy: `ask-on-risk` default; each work unit is one PR, merged after CI green.

## Tasks
- [x] **T1 — hygiene (MWU1)**: drop the duplicate `check_state_files()` call; preserve file mode in `new_retro.py`;
  parenthesize the `box.py` expression; build `by_batch` once. Route: delegated (writer). Release: PATCH.
- [x] **T2 — load counter (MWU2)**: `_loads_outstanding` decrements on timeout/error (try/finally) with a RED test.
- [x] **T3 — navigate `types` filter + progress/permission doctrine**: optional `types` filter on `n4_navigate`;
  METHODOLOGY rules: progress to a file for long clients; surface a permission-classifier block to the operator.
- [ ] **T4 — rollback fidelity (MWU3)**: frozen-child config restore or explicit verdict downgrade; keep
  link-targeted slot values in `_snapshot`; type-contract lookup instead of the heuristic. Separate PR.

## Acceptance
- All mcp-n4-kit unit tests green; new tests fail before the fix.
- CHANGELOG entry + VERSION bump per PR; issue #179 closed when T1-T4 land (or re-scoped with the remainder).

## Progress / evidence
- Baseline (origin/main 3c0ce5c): `python3 -m unittest discover -s mcp-n4-kit/tests -v` -> 433 tests OK.
- T1 (route: delegated writer): duplicate `check_state_files()` dropped; `new_retro._atomic_write` chmods the temp
  file to the destination's mode (new file: 0644 & ~umask); `box.py` drain iterable parenthesized; `retro._window`
  returns `by_batch` (built once). New test `test_keeps_the_index_mode_and_gives_a_new_retro_0644_under_the_umask`:
  RED on the old `new_retro.py` (`AssertionError: 384 != 436`, i.e. 0600), GREEN after.
- T2 (route: delegated writer): RED first — `test_an_answered_load_clears_the_count_left_by_a_timed_out_one` and
  `test_a_cached_handle_regains_the_early_abort_after_a_timed_out_load` FAILED (counter stuck at 1).
  `test_a_rejected_load_request_is_not_left_outstanding` was already green (regression pin). Fix: the counter is
  incremented before `ssc` and decremented if `ssc` raises; an answered load resets it to 0 (replies come in
  request order). A pure timeout still keeps the request counted, because a late reply stays possible
  (`test_an_unanswered_earlier_load_disables_the_stale_handle_early_abort` still holds). GREEN.
- T3 (route: delegated writer): RED first — 6 new `TestNavigate` tests, 5 failed/errored (`KeyError: 'matched'`,
  no -32602 for a non-array `types`). Implemented `types` (array of strings) on `n4_navigate`: kept entries carry
  `matched`; non-matching ancestors stay only as the path to a match; absent/empty = unchanged output. METHODOLOGY
  section 3 (permission-classifier block) and section 7 (progress file, never stdout through `tail`); README and
  SKILL updated. GREEN.
- After T1-T3: `python3 -m unittest discover -s mcp-n4-kit/tests -v` -> 443 tests OK.
- Release: VERSION 0.27.0, `mcp_n4.__version__` 0.3.0, CHANGELOG `[v0.27.0]`. RDD outcome: pending (parent).

## Next step
- Parent: RDD assess/review of the T1-T3 commits, push + PR (human decision). Then T4 (rollback fidelity) as a separate PR.
