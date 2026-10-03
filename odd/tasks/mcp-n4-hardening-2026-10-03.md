# mcp-n4-hardening-2026-10-03 — close the 2026-10-03 audit findings of mcp-n4-kit

## Objective
Close the findings F1-F17 of the 2026-10-03 audit of `mcp-n4-kit` (safety binding, auth lock-out,
write/rollback correctness, test debt, BQL robustness, ergonomics, budget scope and docs drift), one
reviewable work unit per PR.

## Why
- Two high-severity safety gaps: a confirmation token is not bound to the station/session it was issued on
  (F1), and the write scope is a string-prefix check that a chained ORD (`|h:`, `|slot:..`) passes (F2).
- Repeated calls after an authentication failure can lock the station account (F4).
- Wrong rollback data or silent overwrites: nested Status previous values from type defaults (F3), rollback
  without compare-before-restore (F6), undetected snapshot truncation (F8).
- BQL parsing loses duplicate columns and depends on English headers (F7, F12); docs drifted (F15).

## Authorization
- The operator pre-authorized the whole feature: ODD + RDD (consent `granted` for every candidate), commit,
  push, PR, merge and issue comments, chained and automatic (one PR per work unit, each from the latest
  `origin/main` after the previous one merged).
- Allowed edit surfaces: `mcp-n4-kit/**` and this document. Root `VERSION`/`CHANGELOG.md` and
  `build-n4-module-kit/` belong to another release owner: each PR body carries its CHANGELOG entry text
  (README "Versioning").
- The repository is PUBLIC: no customer, station, host, IP or user names.

## Anti-cascade policy
Only a fail-open or a regression introduced by this feature becomes a sub-task. Every other non-blocking review
advisory is parked in the "Parked advisories" section below and in a comment on the tracking issue (#200).

## Constraints
- Python 3.10+, stdlib only, `unittest`. Test-first per behavior (RED observed, then GREEN); every new safety
  guard gets one observed mutation (break it, a test fails, restore).
- Test isolation: `DestructiveCase.patch` / `addCleanup` (README "Test isolation rules"); test-heavy units run a
  shuffled-order pass (seeds 1, 7, 42) before merge.
- `mcp_n4.__version__` bumps per unit (patch for fixes, minor for new tools); the skill metadata version matches.
- Delivery strategy: `single-pr` per work unit (each unit is already <= ~400 authored lines, advisory).

## Tasks
- [x] **H1 — safety binding (F1, F2, F16)**: the plan (so the token MAC) binds station name, base URL and a
  per-connect session id; `WriteScope` refuses any `|` after `station:|slot:` (incl. `|h:`), and prefixes are
  validated at startup; `--token-ttl`/`--max-writes` < 1 are refused at startup. Route: inline (parent
  writer, 3 small files + tests). Release: PATCH 0.5.1.
- [x] **H2 — auth latch (F4)**: `Context.auth_latch` latch set by any `AuthError` (connect, about, get_ord,
  BOX); station calls refused until a cooldown (>= 30 s, configurable) or a reconnect; message gives reason
  and remaining cooldown. Release: PATCH.
- [x] **H2a — fallback latch (H2 review fail-open, anti-cascade sub-task)**: the `Server._call` fallback
  latches every propagated `AuthError`, also while a latch for other credentials is held; the active
  credentials are recorded through `Context.use_credentials` (the latch itself is set only by
  `note_auth_failure`). Shipped in the H3 PR.
- [x] **H3 — set_slot / rollback correctness (F3, F6, F10)**: nested slot load depth; compare-before-restore on
  rollback of set/fallback; clearer "slot not found" for a slot at its type default. Release: PATCH.
- [x] **H4 — test debt (F13)**: tests for the untested write branches. Release: PATCH.
- [ ] **H5 — BQL hardening (F7, F12, output_file)**: duplicate headers kept (`Type`, `Type#2`); inventory
  resolves columns by queried slot / position; optional `output_file` for `n4_bql_query` under the state dir.
  Release: MINOR.
- [ ] **H6 — ergonomics (n4_list_batches, F8, F11)**: read-only `n4_list_batches`; remove-snapshot truncation
  warning; `--load-wait` / `--http-timeout`. Release: MINOR.
- [ ] **H7 — budget scope + docs (F5, F15, F17)**: process-scoped write budget; README/SKILL/METHODOLOGY drift;
  FlexAddress/BacnetAddress value types; skill reinstall + drift check; feature retro; this doc COMPLETE.

## Acceptance
- `python3 -m unittest discover -s mcp-n4-kit/tests` green on every PR; each new test failed before its fix.
- Every new safety guard has an observed mutation recorded below.
- Each PR merged after CI green with its CHANGELOG entry text in the PR body.

## Checks
- `python3 -m unittest discover -s mcp-n4-kit/tests` (repo root).
- Shuffled order (seeds 1, 7, 42) for test-heavy units.
- CI green; native RDD review when due (`gentle-ai review assess ... --committed-only`).

## Live-gated (not certifiable on the fake station; listed on #200)
- F9: inverse of an override action (`emergencyOverride`/`override`) needs a live station.
- Live certification of F2 (a station's answer to a chained ORD), F3 (nested Status load depth), F10 (slot
  at type default omitted) and F12 (localized BQL headers).

## Progress / evidence
- Baseline (origin/main 255f2d9): `python3 -m unittest discover -s mcp-n4-kit/tests` -> 484 tests OK.
- H1 (route: inline, branch `fix/mcp-n4-h1-safety-binding`; 4 source files with small edits, no research needed).
  RED first — 7 new tests: `TestTokenStationBinding` x3 errored (a token from station A EXECUTED on station B
  and after a reconnect: `KeyError: 'isError'`; no `plan["station"]`), `TestWriteScope` chained-ORD and
  startup-prefix tests failed (`station:|slot:/A|h:1f` passed the scope), `TestLimitsStartup` x2 failed (0 and
  -5 accepted). GREEN: 491 tests OK. Decisions:
  - F1: binding lives in the plan (`station: {name, base_url, session_id}`), which the MAC already covers via
    the plan hash; `Session.session_id` is a fresh uuid4 per `n4_connect`. Shown in the dry run on purpose.
  - F2: `safety._plain_slot_ord` — only `station:|slot:/<path>` with no `|` after the slot path and no `..`
    segment; the same rule validates every `--write-scope` prefix at startup (bad prefix -> exit 2).
  - F16: argparse `_positive_int` for `--token-ttl`/`--max-writes`, and `WriteState` refuses < 1 for
    programmatic callers.
  - Observed mutations (each restored): M1 plan `session_id: None` -> 2 binding tests fail; M2 drop the `|`
    check -> 2 scope tests fail; M3 `_positive_int` floor -100 -> parser test fails.
  - Release: `mcp_n4.__version__` 0.5.1, skill metadata 0.5.1.
- H2 (route: inline, branch `fix/mcp-n4-h2-auth-latch`; 3 source files + one new test file). RED first — new
  `tests/test_auth_latch.py` (9 tests): 5 failed and 2 errored (a second `n4_connect` with the rejected password
  reached the station: `2 != 1` requests; reads after a 401 went to the station again; no `auth_cooldown`).
  GREEN: 500 tests OK. Decisions:
  - The latch lives on `Context` (process scope, survives reconnects). `BoxClient._auth_error` calls an
    `on_auth_error` hook before raising, so a failure the caller catches (the version probe) still latches;
    `Server._call` also latches on a propagated `AuthError` (clients without the hook).
  - Gate: every `needs_session` tool and every write tool, checked in `Server._call` before the handler.
    `n4_connect` checks a keyed credential fingerprint: the same credentials wait, other ones may try; a
    good login clears the latch (before the version probe, which may set it again).
  - `--auth-cooldown` default and minimum 30 s (the lock-out window); `n4_describe_session` and
    `n4_connect` report `auth_paused: {reason, retry_in_s}`.
  - Observed mutations (each restored): no hook -> the about test fails; gate off -> 4 tests fail; connect
    check off -> 3 tests fail.
  - Release: `mcp_n4.__version__` 0.5.2, skill metadata 0.5.2.
- H3 (route: inline, branch `fix/mcp-n4-h3-setslot-rollback`; 3 source files, focused edits). RED first — 7
  tests: hookless client let a second rejected credential retry (`'authentication failure' not found`);
  nested `grp/sp` planned previous `0.0` instead of `7.5`; the "not found" text had no explanation; rollback
  overwrote a slot / fallback changed after the batch (x3, incl. a change between dry run and confirm,
  `KeyError: 'isError'` = executed); nested rollback read-back was `mismatch`. GREEN: 506 tests OK. Decisions:
  - H2a: unconditional `note_auth_failure` in the `Server._call` fallback (re-latching restarts the
    cooldown with the failing credentials; the hook path sets the same values).
  - F3: `_slot_depth(slot) = segments + 1` for the set plan, its read-back and the rollback read-back of `s` ops.
  - F6: `_unchanged_since` runs in the rollback plan (so again at confirm): every `s` inverse whose slot
    the batch wrote (`s` op, or `invokeAction set` -> fallback value only) is read and compared; a
    difference refuses with both values. No override argument: like an in-doubt batch, a changed slot is
    the operator's decision, restored by hand with `n4_set_slot` (its own dry run + token).
  - F10: the refusal explains type-default omission and routes to `n4_read_slots` / Workbench / an action.
  - Observed mutations (each restored): compare call removed -> 3 tests fail; depth fixed at 2 -> 2 fail;
    conditional fallback latch -> the hookless test fails.
  - Release: `mcp_n4.__version__` 0.5.3, skill metadata 0.5.3.
- H4 (route: inline, branch `test/mcp-n4-h4-write-branches`; tests + one small fix). Characterization tests for
  the F13 branches (code already existed, so the proof is a mutation per branch, not a RED): status on a plain
  type, rollback without recorded targets, component-intent and relink-intent journal failures, malformed
  relink record, config.bog unreadable after the save; the `readback_failed` retry refusal was already pinned
  by `test_a_rollback_accepted_but_unverified_cannot_be_retried` (mutation confirms). Plus the H1 parked test
  advisories: `server.main` exits 2 for a chained / `..` `--write-scope`, and the `..` prefix case.
  - Found and fixed (RED observed): the component-intent failure text was swallowed — `_in_doubt_with_created`
    wrapped the reason in a ToolError and `_in_doubt` printed only `ToolError`, so the operator never learnt
    the journal write failed. `_in_doubt` now shows a kit-written reason text as is.
  - Observed mutations (each restored): each of the 7 branches disabled -> exactly its test fails.
  - `test_auth_latch` now patches `BoxClient.about` with `mock.patch.object` (README isolation rule).
  - Shuffled order seeds 1, 7, 42: OK. Full suite 513 tests OK.
  - Release: `mcp_n4.__version__` 0.5.4, skill metadata 0.5.4.

## Parked advisories
- H1 review (non-blocking, reliability lens): R3-startup-exit-unproved — no test runs `server.main` with a bad
  `--write-scope` to prove exit 2 (code path exists: `main` catches `SafetyError`); R3-dotdot-prefix-untested —
  the startup prefix test has no `..` case. Candidates for H4 (test debt).
- H2 review (non-blocking): R3/R4 active-credentials-set-before-login — moot in practice (`n4_connect` closes
  the old session first, so no older hooked client outlives a failed connect); the attribute is now set via
  `Context.use_credentials` (R2-private-attr-crossing). R2-task-doc-stale-name fixed (`auth_latch`). The
  fallback-latch findings (R2/R3/R4) were a fail-open -> H2a.
- H3 review (non-blocking): R4 compare-before-restore compares a whole Status (value + status), so a
  station-driven status change also refuses (fail-closed, by design); R4 a `KeyError`/`ValueError` from
  `_observe` is reported as "changed" without its cause; R2 the dense `seen` expression; R2 the H2a doc
  wording (fixed in H4). R3 invoke-set unchanged path: already covered by
  `test_rollback_of_invoke_set_restores_the_fallback`.

## Delivery record
| Unit | PR | Merge | Version | Review |
| --- | --- | --- | --- | --- |
| H1 | #229 | 0487970 | 0.5.1 | medium; 1 lens APPROVED + acknowledged (review-b8c070f2f4a8c38e); 2 advisories parked |
| H2 | #231 | 0b7c3dd | 0.5.2 | high; 4 lenses APPROVED + acknowledged (review-ce356fee111e83c7); 1 fail-open -> H2a, rest parked |
| H3 | #232 | 81cdb3b | 0.5.3 | high; 4 lenses APPROVED + acknowledged (review-3bbf4aa80a638470); 4 advisories parked |

## Next step
- H5 (BQL hardening).
