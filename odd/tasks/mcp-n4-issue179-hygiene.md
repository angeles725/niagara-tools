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
- [x] **T2 — load counter (MWU2)**: `_loads_outstanding` is decremented when `ssc` raises (try/except) and reset
  to 0 when a load is answered (replies in request order), with RED tests.
- [x] **T3 — navigate `types` filter + progress/permission doctrine**: optional `types` filter on `n4_navigate`;
  METHODOLOGY rules: progress to a file for long clients; surface a permission-classifier block to the operator.
- [x] **T4 — rollback fidelity (MWU3)**: frozen-child config restore or explicit verdict downgrade; keep
  link-targeted slot values in `_snapshot`; type-contract lookup instead of the heuristic; the six #184
  advisory findings. Separate PR. Route: delegated (writer). Release: MINOR (new verdict `partial`).

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
- Release: VERSION 0.27.0, `mcp_n4.__version__` 0.3.0, CHANGELOG `[v0.27.0]`. RDD outcome: assessed high (dangerous_sink, process_boundary); consent granted (operator pre-authorized); 4-lens review APPROVED and acknowledged (lineage review-3caa2e8cefaeef6e). START returned a false failure (gentle-ai 4.0.0 defect, reported as Gentleman-Programming/gentle-ai#5195); bound STATUS recovered the minted lineage.
- Advisory (non-blocking) findings carried to T4: R2-001 `_types_arg` docstring says "set" but returns the list; R2-002 T2 task line says try/finally (code uses try/except + reset-on-answer); R3-001 `ssc` raising after the request was sent would under-count; R3-002 umask read is process-global; R3-003 `has_children` still counts filtered-out descendants; R4-001 reset-to-0 relies on in-order replies.

- T4 (route: delegated writer, branch `feat/mcp-n4-rollback-fidelity`): RED first — 7 tests failed before the
  fix (`COMPONENT_TYPES` missing; `KeyError: 'link_inputs_not_restored'` x3 incl. the remove-plan `prior`;
  frozen-config/type/descendant verdicts still `verified`); the "matching frozen config stays verified" test was
  a green regression pin. GREEN after the fix. Decisions:
  - G1/R4-002: verdict downgrade, not a restore. The read-back compares each frozen child with the snapshot
    (type + captured slots, omitted = type default) and reports `frozen_config_not_restored`; verdict
    `partial`. Not restored because a fresh frozen child can be of another type (NullProxyExt vs a driver
    proxyExt) that a set op cannot change, and set ops on frozen children are not live-certified.
  - G2/R3-002: `_snapshot` keeps a link-target's value on its link record (`prior`); external-source inputs
    become `unlinked_input` inverse entries. Rollback reports `link_inputs_not_restored` (skipped relink stays
    `mismatch`; external link -> `partial`). Reported, not written (live finding 3 rejected these values).
  - G3: no contract/catalog in the kit; explicit `box.COMPONENT_TYPES` table (7 entries from N4 `extends`
    declarations / live loads), heuristic fallback; test shows the table wins both ways.
  - Advisory: R2-001 docstring, R2-002 this task line, R3-003 `has_children` semantics in the tool
    description, R3-001/R4-001 comments, R3-002 umask read from `/proc/self/status` (fallback set-and-restore)
    with a test.
- After T4: `python3 -m unittest discover -s mcp-n4-kit/tests -v` -> 452 tests OK (443 before; +9).
- Release: VERSION 0.28.0, `mcp_n4.__version__` 0.4.0, CHANGELOG `[v0.28.0]`.
- Open (not in T4): writing frozen-child config back through `s` ops (the restore half of B1200-G1) and a live
  `reg.loadContract` lookup (B1200-G3) both need a station run to certify; the rollback verdict now discloses
  the gap instead.

## Next step
- Open the T4 PR, run native review if due, merge after CI green, then close issue #179 (remaining live-only
  follow-ups noted above).
