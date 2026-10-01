# mcp-n4-kit

An MCP server plus a Claude Code skill that let an AI agent read and write
Niagara N4 stations over the BOX JSON protocol (route R-A). No station module
is needed: the client speaks to `POST <station>/box/` with HTTP Basic auth.

## Status

T1: the stdlib-only BOX client library (`mcp_n4/box.py`) and its tests against
an in-memory fake station. T2: the stdio MCP server with read-only station tools.
Write tools and the skill come in later tasks.

## Layout

- `mcp_n4/box.py`: BOX frame, session, `syncTo`, `checkLinks`, `invokeAction`,
  `pollchgs`, tree loading and status helpers. No side effects at import.
- `mcp_n4/server.py`: stdio JSON-RPC 2.0 protocol layer and `main()`.
- `mcp_n4/tools_read.py`: the read tool implementations and the session state.
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

- `--allow-writes`: records the writes-allowed mode (write tools arrive in T3);
  the default is read-only.
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
