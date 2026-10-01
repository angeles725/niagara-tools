---
name: mcp-n4
description: "Trigger: read or write a live Niagara N4 station through the mcp_n4 MCP server (BOX JSON) — navigate the component tree, read slots, create components, set values, link, invoke actions, remove, rollback, save. Enforces the dry-run, confirmation-token, write-scope and read-back workflow and the route ladder."
license: Apache-2.0
metadata:
  author: angeles725
  version: "0.1"
---

Thin launcher. The real content lives in an EXTERNAL kit (METHODOLOGY, README, server code) — read it, don't restate it from memory.

## Resolve the kit (do this first)

`KIT` = the directory holding `METHODOLOGY.md` and `mcp_n4/server.py`. Resolve once:
1. `$MCP_N4_KIT` if set and it contains `METHODOLOGY.md`.
2. Else the default: `/home/cristian/modulos_niagara_n4/niagara-tools/mcp-n4-kit` (confirm `METHODOLOGY.md` exists).
3. Else locate it: `fd -t f server.py` under the user's module dirs and keep a hit whose parent is `mcp_n4/` next to a `METHODOLOGY.md`. Never search `$HOME` or `/` wholesale; if the result is missing or ambiguous, ask.

Then read `$KIT/METHODOLOGY.md` FIRST and follow its session checklist.

## Register the server

Follow `$KIT/README.md` ("Running the server" and the `.mcp.json` example). The operator, not the model, chooses stations, credential env prefix and TLS policy (`--station NAME=https://…`, `--credential-env`, `--insecure-tls NAME`). Writes need `--allow-writes` plus at least one `--write-scope`; the default is read-only. If no station is configured, stop and tell the operator which flag is missing.

## Session checklist (full detail in METHODOLOGY section 4)

1. `n4_describe_session`, `n4_connect` to a configured station; confirm the real station name and the version tier.
2. `n4_navigate` / `n4_read_slots` the subtree.
3. Write tools with `dry_run=true` (the default); show the plan to the human.
4. After approval, repeat with `dry_run=false` and the `confirmation_token`.
5. Read the verdict; `n4_find_dangling_outputs`; `n4_save_station` with persistence evidence.
6. Keep every `batch_id`; close with the session retro.

## Hard Rules (non-negotiable)

- Never write without showing the dry-run plan to the human first.
- No credentials in tool arguments or chat; they come from the operator's environment.
- Never retry a failed login: 5 failures in 30 s lock the account.
- Report `mismatch`, `failed`, `unverified`, `in-doubt` and `persisted: unknown` exactly as returned; never call them success.
- Never invent a bare "cannot": give the route ladder from METHODOLOGY section 1 (cost, needs, next step).
- Never contact a station the operator did not configure and authorize.
- Close with the session retro: proposed kit deltas, never applied unprompted.

## Output Contract

Report: kit path, station name and tier, each write as `requested / observed / verdict` with its `batch_id`, dangling-output result, save evidence, and the retro line.

## References

- `$KIT/METHODOLOGY.md`, `$KIT/README.md`, `$KIT/mcp_n4/`.
- Install this launcher: `niagara-tools/scripts/install-skill.sh --skill mcp-n4`.
