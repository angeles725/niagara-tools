<!-- review-status: pending -->
<!-- Marker lifecycle: the maintainer flips 'pending' above to 'folded <date> · kit <sha>' once the proposed deltas are reviewed and applied (or dismissed) in the kit. This retro only PROPOSES; kit changes are human-reviewed and human-committed. -->
# Retro — Prueba_4_15 · 2026-11-02 · mcp-n4 session: live write-path check of a client dashboard module

> Seeded by `n4_session_retro_draft` (audit.jsonl + journal.jsonl), then reviewed by hand. Client anonymized as "client E".

## Session summary

- Station: Prueba_4_15, Niagara 4.15.6.20, tier B (writes allowed by operator opt-in after a scratch PoC: create folder, create writables, `set`, read back — all `verified`).
- Purpose: build a test fixture (writable points + links into the dashboard's facade panels) so the module's setpoint and reset-pulse paths could be checked live; tear it down after.
- Audit: 150+ entries (executed 69, planned 72, refused 6, error 3). Journal: 71+ batches (verified 68, in-doubt 2, unverified 1).
- Driver: no MCP client registration was possible mid-session, so a stdio driver kept one server process (one connect session) alive behind a unix socket so dry-run tokens survive to the execute call.

## Proposed kit deltas

| # | Proposed change | Target (file · section) | Evidence (batch_id / tool call / audit line) | Type | Priority |
|---|---|---|---|---|---|
| 1 | Ship a persistent stdio driver (`tools/drive.py`): env-file credentials, one long-lived server so the dry-run token and the execute share a session; relative socket path (AF_UNIX 108-byte limit) | `README.md · Running the server` | hand-written driver used for the whole session; first bind failed "AF_UNIX path too long" | tool | high |
| 2 | Document the operator credential pattern: `~/.config/mcp-n4/<station>.env` (0600) created by the operator with `! printf … > file`, loaded by the client; the hard rule "no creds in tool args" has no recipe today | `README.md · Running the server` | session setup | doc | high |
| 3 | Ship a tier-B PoC script (scratch folder → writable → `set` → read back → remove) that records a PoC file the `--allow-tier-b` opt-in can point to | `METHODOLOGY.md · section 5` | batches bbccfadb…, 2d2ed586…, 6d0be14c… (PoC path) | tool | medium |
| 4 | `active`/`inactive`/`auto` on `control:BooleanWritable` need a BOverride argument; the kit sends none → "station error", batch journaled in-doubt. Send a default BOverride or drop them from the allowlist | `mcp_n4/tools_write.py · invoke allowlist` | batch 99aa4125c6b94ac5b6742df00d155c11 (in-doubt), 979f013a… (`auto` unverified) | bug | high |
| 5 | A definitive station refusal ("Could not create link: Target property is readonly." on `in8`) is journaled in-doubt and returned as plain text, not JSON. Classify `Could not …` refusals as failed and keep JSON replies; optionally pre-check READONLY link targets (in1/in8, in6 on BooleanWritable) | `mcp_n4/tools_write.py · error path` | batch 6833bf9494284e29aa4f71e7cef3d95d | bug | high |
| 6 | The BOX server session expires after a short idle ("Could not find Server Session" as plain text). Reconnect once automatically, or return a JSON error naming `n4_connect` | `mcp_n4/box.py · session` | n4_rollback and n4_read_slots after ~90 s idle | bug | high |
| 7 | Kit reads after idle are slow, which made a late reset release (7–8.5 s) look like an observer artefact; per-request timestamps through an independent observer later showed the late releases were real (4 of 20 at 7.5–8.4 s). METHODOLOGY must say: timing claims need an independent observer with per-request timestamps, never kit reads after idle | `METHODOLOGY.md · section 3 live rules` | 20 API-observed pulses, two observers | doc | medium |
| 8 | A top-level scratch folder cannot be created with a narrow scope (the parent `station:|slot:/` must be in scope). Add a create-only-child scope form, or document using an existing child folder | `README.md · write scope` | audit 05:50 n4_create_component refused | doc | low |
| 9 | `n4_set_slot` cannot write `baja:Facets` (EnumWritable `range`) or `baja:RelTime` (minActiveTime): enum write paths and min-on/min-off interplay cannot be live-tested through the kit | `mcp_n4/tools_write.py · value types` | enum writes refused `limits-unknown` until facets exist | feature | medium |
| 10 | Add an in-doubt recovery step: read the target slot, decide retry or rollback, record the decision | `METHODOLOGY.md · section 4 step 6` | batches 99aa4125…, 6833bf94… | doc | medium |
| 11 | `claude mcp list` hangs > 120 s when another configured MCP server times out; never use it to probe registration | `README.md · registering` | session start | doc | low |

| 12 | The BOX session ends ("Could not find Server Session") after a burst of HTTP Basic-auth requests by the same user, and after every station restart; every later call fails until `n4_connect`. Detect that reply and say "session lost — call n4_connect" (never an automatic re-login loop: lockout) | `mcp_n4/session` + `METHODOLOGY.md · section 4` | live test 1.4.4, 03:39 | code | medium |
| 13 | `n4_read_slots` returns only slots that differ from their default (AHU_ALM_RST: proxyExt, out, fallback; never in1..in16, facets, minActiveTime; the service omitted autoSync=true): a default-valued slot cannot be read, so a commissioning check ("is autoSync on?", "are the facets empty?") is impossible through the kit. Add `include_defaults` (or a `slots` filter that always answers each named slot) | `go/internal/mcpn4` read_slots | 0.9.5 tour M4, oBIX autoSync val=true | code | high |
| 14 | `n4_save_station` with persistence evidence is the required step before any forced station stop (no daemon, no console); the verdict `verified` + config.bog sha change is the proof | `METHODOLOGY.md · section 4 step 5` | batch e32a5330… | doc | low |

| 15 | The default niagara-tools checkout sat on an old feature branch 2159 commits behind origin/main and ran the retired Python server (v0.7.4), and the installed launcher copy under `~/.claude/skills/` still named `mcp_n4/server.py`; the skill launcher must resolve the server binary like kit-resolve does (origin/main, clean) and refuse a stale one | `skill/SKILL.md` resolve step; `README.md` running the server | `git rev-list --count HEAD..origin/main` = 2159 | code | high |
| 16 | The Go server refuses a group-writable `--state-dir` (mode 775) with a clear `chmod 700` hint — correct; document it next to `--state-dir` because a fresh `mkdir` under a 002 umask always hits it | `README.md` flags | first start exit 1 | doc | low |
| 17 | `n4_list_links` rejects `depth: 0` ("must be >= 1") while `n4_navigate` treats depth as "levels below"; say in the schema description that 1 = the component itself | tool schema | tour M7 | doc | low |
| 18 | `n4_set_slot` on a point created in the same batch run returned no verdict (W4) while the same call on an existing point plans fine; reproduce with create → set_slot fallback and report the reply | `go/internal/mcpn4` set_slot | tour W4 | probe | medium |
| 19 | Live tour 0.9.5 (Go) on 4.15.6.20: 21/23 PASS — connect, describe, navigate (+type filter), BQL, inventory, list_links, px views, create folder/points with wire-sheet cell, invoke set (observed fallback 17.25), link out→in10 with propagation, rollback of the link batch, scope refusal, remove, dangling scan, list_batches | `tests/` live smoke | mcptour-results.json | evidence | — |
| 20 | The live tour needed a hand-written Python stdio driver (a socket wrapper around the server) to call tools outside the MCP client, which breaks the "new tools in Go" rule. Add a Go subcommand `mcp-n4 call <tool> '<json>'` (one-shot, with the same flags as the server) and `mcp-n4 tour` (the live smoke tour, read-only plus scope-limited writes), so no live check needs a scratch driver | `go/cmd/mcp-n4` | gdrive.py, mcptour.py this session | code | high |
| 21 | Fail closed on a retired server: the server and the launcher should refuse a `python3 -m mcp_n4.server` launch path. Example: an `mcp_n4/__main__.py` stub on old branches is not possible, so put a version floor (`serverInfo.version` >= launcher `metadata.version`) in the SKILL resolve step (done in this PR) plus `install-skill.sh --check` drift detection (build-n4 retro Δ35) | `skill/SKILL.md`, `scripts/install-skill.sh` | row 15 | code | high |

## Already covered

- Dry run default, single-use token, scope, journal and rollback worked as documented: every rollback of a test link was `verified` and the end-state check passed.

## Lessons

- A live check that adds temporary links (masks on in10) removes them in a `finally` block; one crash left a link behind until a manual rollback.
- Put the station in a known state through the kit before an API suite and verify the same state at the end.
