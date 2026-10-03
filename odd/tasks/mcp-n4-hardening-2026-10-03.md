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
- [ ] **H2 — auth latch (F4)**: `Context.auth_failed` latch set by any `AuthError` (connect, about, get_ord,
  BOX); station calls refused until a cooldown (>= 30 s, configurable) or a reconnect; message gives reason
  and remaining cooldown. Release: PATCH.
- [ ] **H3 — set_slot / rollback correctness (F3, F6, F10)**: nested slot load depth; compare-before-restore on
  rollback of set/fallback; clearer "slot not found" for a slot at its type default. Release: PATCH.
- [ ] **H4 — test debt (F13)**: tests for the untested write branches. Release: PATCH.
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

## Parked advisories

## Delivery record

## Next step
- H2 (auth latch).
