---
name: mcp-n4
description: "Trigger: read or write a live Niagara N4 station through the mcp_n4 MCP server (BOX JSON) — navigate the component tree, read slots, create components, set values, link, invoke actions, remove, rollback, save. Enforces the dry-run, confirmation-token, write-scope and read-back workflow and the route ladder."
license: Apache-2.0
metadata:
  author: angeles725
  version: "0.7.1"
---

Thin launcher. The real content lives in an EXTERNAL kit (METHODOLOGY, README, server code) — read it, don't restate it from memory.

## Resolve the kit (do this first)

`KIT` = the directory holding `METHODOLOGY.md` and `mcp_n4/server.py`. Resolve once:
1. `$MCP_N4_KIT` if set and it contains `METHODOLOGY.md`.
2. Else the default: `/home/cristian/modulos_niagara_n4/niagara-tools/mcp-n4-kit`, the author's checkout on the primary machine (confirm `METHODOLOGY.md` exists). The installed launcher is a copy under `~/.claude/skills/`, so its own location says nothing about the kit; on any other machine or checkout set `MCP_N4_KIT` to the `mcp-n4-kit` directory.
3. Else STOP searching and say so plainly: "the mcp-n4 kit is not on disk at <path checked>". A checkout on another branch may lack `mcp-n4-kit/`; never switch a shared checkout's branch to get it. Offer the operator one of:
   - a worktree of `origin/main`: `git -C <niagara-tools checkout> fetch origin && git -C <niagara-tools checkout> worktree add <path> origin/main`, then `MCP_N4_KIT=<path>/mcp-n4-kit`;
   - an installed copy they already have (set `MCP_N4_KIT` to it).
   Ask before creating the worktree.

Then read `$KIT/METHODOLOGY.md` FIRST and follow its session checklist.

## Register the server

Follow `$KIT/README.md` ("Running the server", "Getting the kit and registering it per project"). Register the server once in the project's `.mcp.json` rather than driving stdio by hand. The operator, not the model, chooses stations, credential env prefix, TLS policy and timing (`--station NAME=https://…`, `--credential-env`, `--insecure-tls NAME`, `--load-wait`, `--http-timeout`, `--auth-cooldown`), and where each station keeps its `config.bog` (`--station-home NAME=PATH`, so `n4_save_station` can prove persistence). Writes need `--allow-writes` plus at least one `--write-scope`; the default is read-only. A tier B or C station (METHODOLOGY section 5) also needs the operator's `--allow-tier-b NAME` / `--allow-tier-c NAME`, given only after the PoC or probe that tier requires. If no station is configured, stop and tell the operator which flag is missing.

## Session checklist (full detail in METHODOLOGY section 4)

1. `n4_describe_session`, `n4_connect` to a configured station; confirm the real station name, `version` and `tier`. With `tier_writes: refused`, plan with dry runs only (they carry `tier_gate`) and give the operator the refusal's routes.
2. Inventory or wide reads: `n4_inventory` / `n4_bql_query` FIRST (METHODOLOGY section 7); `n4_navigate` (optional `types` filter) / `n4_read_slots` only for targeted components. A read or client expected to exceed ~1 min writes progress to a file and reports counts; never rely on stdout piped through `tail`.
3. Write tools with `dry_run=true` (the default); show the plan to the human.
4. After approval, repeat with `dry_run=false` and the `confirmation_token`.
5. Read the verdict; `n4_find_dangling_outputs`; `n4_save_station` with persistence evidence.
6. Keep every `batch_id` (`n4_list_batches` lists them, newest first).
7. Close with the session retro: call `n4_session_retro_draft` (or run `$KIT/tools/new_retro.py --station NAME --state-dir DIR`), review the candidate deltas, keep or edit them (or leave the honesty line `no new deltas`). Propose, never apply; stage issues only by hand.

## Hard Rules (non-negotiable)

- Never write without showing the dry-run plan to the human first.
- No credentials in tool arguments or chat; they come from the operator's environment.
- Never retry a failed login: 5 failures in 30 s lock the account. On a 401, have the operator check the user's Authentication Scheme Name = `HTTPBasicScheme`, then the password.
- Report `partial`, `mismatch`, `failed`, `unverified`, `in-doubt` and `persisted: unknown` exactly as returned; never call them success. A `partial` rollback lists `frozen_config_not_restored` / `link_inputs_not_restored`: show them to the operator. A frozen child whose read-back failed carries `readback_error` and makes the verdict `unverified`, not `partial`: report it as "could not check", never as "missing".
- Never ask for or suggest a tier opt-in to get past a refused write without the PoC or probe that tier requires; the opt-in is the operator's record that it ran.
- Never invent a bare "cannot": give the route ladder from METHODOLOGY section 1 (cost, needs, next step).
- Never contact a station the operator did not configure and authorize.
- If the harness permission classifier blocks a live write, surface it to the operator and stop; never route around it (no other tool, subagent or peer session).
- Close with the session retro (`n4_session_retro_draft` / `tools/new_retro.py`): proposed kit deltas, never applied unprompted; never stage issues automatically.

## Output Contract

Report: kit path, station name and tier, each write as `requested / observed / verdict` with its `batch_id`, dangling-output result, save evidence, and the retro line.

## References

- `$KIT/METHODOLOGY.md`, `$KIT/README.md`, `$KIT/mcp_n4/`.
- Install this launcher: `niagara-tools/scripts/install-skill.sh --skill mcp-n4`.
