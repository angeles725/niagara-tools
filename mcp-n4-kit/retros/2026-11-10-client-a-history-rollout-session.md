<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed and human-committed. -->
# Retro — client-a-jace · 2026-11-10 · mcp-n4 session retrospective (history rollout)

> Drafted with `n4_session_retro_draft` (1 mechanical candidate, kept as Δ1) and completed by hand from the
> session: 35 interval history extensions on a client A JACE (N4 4.14, tier A) over a metered tunnel.
> The reviewer keeps, edits or drops each delta. Stage GitHub issues only by hand.

## Session summary

- Window 1 (old binary): 10 extensions configured with `n4_set_slot` (create + historyName + interval +
  enabled, each a dry run plus an execute: about 8 calls per point). The tunnel dropped mid-batch
  (`Connection refused`); the next `n4_connect` opened a new server session and every pending token was dead.
- Window 2 (new binary, `n4_add_history`): 15 extensions in one dry run + one execute (batch 7388b038),
  10 half-configured extensions removed one by one (10 dry runs + 10 executes) and re-added in one batch
  (dfba1698). BQL over `history:HistoryExt`: 35 rows, 5 minutes, enabled, `{ok}`.
- Station saved twice: verdict `unverified`, `persisted: unknown` (remote JACE, no `--station-home`).
- Audit (window 2): 27 entries, executed 13, planned 13, read 1; journaled batches 13 (verified 12,
  unverified 1: the save).

## Proposed kit deltas

| # | Proposed change | Target (file · section) | Evidence (batch_id / tool call / audit line) | Type | Priority |
|---|---|---|---|---|---|
| 1 | Prove a save on a REMOTE station without `--station-home`: read the station's `config.bog` file metadata (size, modified time) through the file ORD before and after the save, and report `persisted: changed` / `unchanged` | `METHODOLOGY.md · section 3` | batch c72b607ff81d41d58bdf125b7baaf475 (n4_save_station, verdict unverified); batch 39613c4dee0249b9b64313561e568b95 | new | HIGH |
| 2 | `n4_add_history` takes an optional `capacity` (records) so a dashboard's 7 d / 30 d views are covered; today the default 500 records is about 41 h at 5 min | `README.md · Writing safely` | batch 7388b038b65c4e7f9e8c77e7bd1abde3 note "Capacity stays at the extension default" | new | HIGH |
| 3 | A batch remove (`n4_remove_components`, many children under one token, one snapshot per child) for clean-up after a partial rollout | `README.md · Writing safely` | ten single removes 4fed6c3b..4660e346 (20 tool calls) before re-adding with one `n4_add_history` | new | MEDIUM |
| 4 | `n4_read_history` (last N records of a history, by history id or by the extension ORD) so a rollout proves records arrive, not only configuration | `METHODOLOGY.md · section 4` | BQL row `{ok}` proves the extension runs, not that records exist | new | MEDIUM |
| 5 | After a server-session loss, `n4_connect` lists the dry runs that died (tool + args digest) and offers to re-plan them; never re-sends a token | `METHODOLOGY.md · section 3` | 8 dry runs issued, then `Connection refused`; the reconnect made them unusable | new | MEDIUM |
| 6 | Retry a READ-ONLY call (dry run planning, read_slots, BQL) once on a transport timeout; never an execute | `METHODOLOGY.md · section 2` | dry run of `$32005/.../RA_T` enabled: `transport failure: urlopen error timed out`, the identical retry succeeded | new | LOW |
| 7 | `n4_bql_query` refusal for a `|` in the query names the fix: put the subtree in `base` | `README.md · Reading` | refusal "query must not contain '|'" on `station:|slot:/Drivers/...|bql:select ...` | new | LOW |
| 8 | A generic `n4_set_slots` (same slot/value on many ORDs, one token) for recipes that are not history extensions | `README.md · Writing safely` | 30 `n4_set_slot` calls for historyName/enabled on 10 extensions in window 1 | new | LOW |

## Already covered (dedupe)

- Tokens bound to the server session refused every stale token after the reconnect (METHODOLOGY section 2,
  audit F1): worked as designed; the SKILL already carries the "redo the dry runs" rule.
- The read-back reported `failed` + `readback_failed` (not success) when the tunnel dropped after an accepted
  write; a later BQL showed the write had landed. Behaviour matches METHODOLOGY section 3.
- The write-scope and tier gates allowed the session (tier A) without extra flags.

## Honest verdict

Two deltas are worth doing first: Δ1 (a save is the only write the kit cannot prove on a remote station) and
Δ2 (without capacity the rollout looks done but silently drops history after 41 h). Δ3-Δ8 are ergonomics
measured in tool calls and data volume over a metered link; keep or drop them by cost.
