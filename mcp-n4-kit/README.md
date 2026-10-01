# mcp-n4-kit

An MCP server plus a Claude Code skill that let an AI agent read and write
Niagara N4 stations over the BOX JSON protocol (route R-A). No station module
is needed: the client speaks to `POST <station>/box/` with HTTP Basic auth.

## Status

T1: the stdlib-only BOX client library (`mcp_n4/box.py`) and its tests against
an in-memory fake station. T2: the stdio MCP server with read-only station tools.
T3: guarded write tools (`--allow-writes`). The skill comes in a later task.

## Layout

- `mcp_n4/box.py`: BOX frame, session, `syncTo`, `checkLinks`, `invokeAction`,
  `pollchgs`, tree loading and status helpers. No side effects at import.
- `mcp_n4/server.py`: stdio JSON-RPC 2.0 protocol layer and `main()`.
- `mcp_n4/tools_read.py`: the read tool implementations and the session state.
- `mcp_n4/safety.py`: confirmation tokens, write scope, journal and audit log.
- `mcp_n4/tools_write.py`: the write tools and their guard pipeline.
- `tests/test_server.py`: protocol, tool and end-to-end stdio tests.
- `tests/fake_station.py`: fake BOX station on `127.0.0.1` (plain HTTP, tests only).
- `tests/test_box.py`: unit tests.

Security defaults: `http://` is refused unless `allow_http=True` (tests only),
the password is never in `repr`, and HTTP 401/403 raises `AuthError` with no
retry (5 failures in 30 s lock the account).

## Running the server

From `mcp-n4-kit/`:

```
MCP_N4_USER=<user> MCP_N4_PASSWORD=<secret> python3 -m mcp_n4.server
```

The server speaks newline-delimited JSON-RPC 2.0 on stdin/stdout. Credentials are
read from the environment only (`<prefix>_USER` / `<prefix>_PASSWORD`, prefix
`MCP_N4` by default, changeable per `n4_connect` call) and never appear in output.

Flags:

- `--allow-writes`: registers the write tools; the default is read-only.
- `--write-scope ORD_PREFIX` (repeatable): ORD prefixes writes may touch.
- `--state-dir DIR`: journal and audit directory (default `~/.local/state/mcp-n4`).
- `--token-ttl SECONDS`: confirmation token lifetime (default 300).
- `--max-writes N`: executed writes allowed per session (default 200).
- `--allow-http-for-tests`: permits `http://` base URLs; for the fake station only.

Example MCP client entry (Claude Code `.mcp.json`):

```json
{
  "mcpServers": {
    "n4": {
      "command": "python3",
      "args": ["-m", "mcp_n4.server"],
      "cwd": "/path/to/niagara-tools/mcp-n4-kit",
      "env": {"MCP_N4_USER": "<user>", "MCP_N4_PASSWORD": "<secret>"}
    }
  }
}
```

Read tools (all `readOnlyHint`): `n4_connect`, `n4_describe_session`, `n4_navigate`,
`n4_read_slots`, `n4_list_links`, `n4_find_dangling_outputs`. One station session
is active per process; `n4_connect` replaces it. `n4_list_links` and
`n4_find_dangling_outputs` scan components up to `depth` levels below `ord`
(the ord itself is level 0) and only see links inside that subtree.

## Writing safely

Write tools exist only with `--allow-writes` (otherwise `tools/list` omits them
and calling one returns JSON-RPC -32601): `n4_create_component`, `n4_set_slot`,
`n4_invoke_action`, `n4_create_link`. Example:

```
python3 -m mcp_n4.server --allow-writes --write-scope 'station:|slot:/Sandbox'
```

Every write call goes through these layers:

1. **Identity.** Refused unless the session was opened with `expected_station`
   and it matched; reconnect with `n4_connect expected_station=<stationName>`.
2. **Budget.** At most `--max-writes` executed writes per session.
3. **Scope.** Every target ORD must sit under a `--write-scope` prefix, matched on
   slot boundaries (`/A` does not cover `/AB`). With no prefix, every write is refused.
4. **Dry run.** `dry_run` defaults to true: the reply is the exact BOX ops and their
   inverse, a `plan_hash` and a `confirmation_token`; nothing is sent to the station.
5. **Token.** To execute, repeat the same call with `dry_run=false` and the token.
   It is single use, expires after `--token-ttl`, and is an HMAC under a per-process
   random key bound to the tool, the arguments, the plan and the expiry. If the
   station changed since the dry run the plan hash differs and the token is refused.
6. **Journal.** Each executed write appends `{batch_id, ts, tool, ops, inverse,
   station_name}` to `<state-dir>/journal.jsonl`, so it can be undone by hand.
7. **Audit.** Every write call, dry, executed or refused, appends a line to
   `<state-dir>/audit.jsonl`; arguments whose key contains `pass`, `secret`, `token`
   or `credential` are replaced by `***`.
8. **Read-back.** After executing, the node is reloaded and the reply carries
   `{requested, accepted, observed, verdict}`: `verified`, `mismatch` (reported, not
   raised), `failed`, or `unverified` for the state-changing actions (`active`,
   `inactive`, `auto`) that have no slot to read back.

The state directory is created with mode 0700 and the files with 0600.

Tool rules:

- `n4_set_slot` writes Status slots whole (value plus status) and refuses a path
  ending in `/value` or `/status`. The slot must already be listed by the station;
  a plain slot at its type default is omitted by the station and is refused.
- `n4_invoke_action` allows only `set`, `active`, `inactive` and `auto`. `set` restores
  the previous `fallback` as its inverse; the others have none and the plan says so.
  Destructive actions (`emergency*`, `save`, `restart`) are not available yet.
- `n4_create_component` follows a station rename on collision (the inverse and the
  read-back use the assigned name).

## Run the tests

```
python3 -m unittest discover -s mcp-n4-kit/tests -v
```

Python 3.10+, stdlib only (no pytest, no MCP SDK).

## Evidence (niagara-research)

- B1177: BOX recipe.
- B1192: decode and Status.
- B1179: auth and lock-out.
- B1197: MCP synthesis.
- B1199: live PoC on station `LLM`.

Note: this kit is not covered by the repo pre-push retro gate yet.
