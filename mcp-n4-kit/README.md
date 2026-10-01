# mcp-n4-kit

An MCP server plus a Claude Code skill that let an AI agent read and write
Niagara N4 stations over the BOX JSON protocol (route R-A). No station module
is needed: the client speaks to `POST <station>/box/` with HTTP Basic auth.

## Status

T1 only: the stdlib-only BOX client library (`mcp_n4/box.py`) and its tests
against an in-memory fake station. The MCP server, the tools and the skill
come in later tasks.

## Layout

- `mcp_n4/box.py`: BOX frame, session, `syncTo`, `checkLinks`, `invokeAction`,
  `pollchgs`, tree loading and status helpers. No side effects at import.
- `tests/fake_station.py`: fake BOX station on `127.0.0.1` (plain HTTP, tests only).
- `tests/test_box.py`: unit tests.

Security defaults: `http://` is refused unless `allow_http=True` (tests only),
the password is never in `repr`, and HTTP 401/403 raises `AuthError` with no
retry (5 failures in 30 s lock the account).

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
