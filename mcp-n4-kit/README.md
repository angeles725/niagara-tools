# mcp-n4-kit

An MCP server plus a Claude Code skill that let an AI agent read and write
Niagara N4 stations over the BOX JSON protocol (route R-A). No station module
is needed: the client speaks to `POST <station>/box/` with HTTP Basic auth.

## Status

T1: the stdlib-only BOX client library (`mcp_n4/box.py`) and its tests against
an in-memory fake station. T2: the stdio MCP server with read-only station tools.
T3: guarded write tools (`--allow-writes`). T4: destructive tools, rollback and save. T5: METHODOLOGY and the skill launcher.
v0.2.0 (retro 2026-10-02): bulk reads with `n4_bql_query` / `n4_inventory`, adaptive load
polling with `elapsed_ms`, display strings for complex slots, an actionable 401.
v0.3.0 (issue #179): `n4_navigate` `types` filter, load-counter recovery after a timed-out load,
retro files keep their mode.

## Methodology and skill

- [METHODOLOGY.md](METHODOLOGY.md): route ladder, safety layers mapped to code and tests,
  live rules, the mandatory session checklist and version tiers. Read it before writing.
- [skill/SKILL.md](skill/SKILL.md): the Claude Code skill launcher. Install it with
  (from the repository root; add `--dry-run` to preview, `--force` to overwrite a diverged copy):

```
scripts/install-skill.sh --skill mcp-n4
```

## Layout

- `METHODOLOGY.md`, `skill/SKILL.md`: see above.
- `mcp_n4/box.py`: BOX frame, session, `syncTo`, `checkLinks`, `invokeAction`,
  `pollchgs`, tree loading and status helpers. No side effects at import.
- `mcp_n4/server.py`: stdio JSON-RPC 2.0 protocol layer and `main()`.
- `mcp_n4/tools_read.py`: the read tool implementations and the session state.
- `mcp_n4/bql.py`: pure BQL helpers (query guard, ORD composition, CSV parsing, `$xx`
  decoding, inventory summary).
- `mcp_n4/safety.py`: confirmation tokens, write scope, journal and audit log.
- `mcp_n4/tools_write.py`: the write tools and their guard pipeline.
- `tests/test_server.py`: protocol, tool and end-to-end stdio tests.
- `tools/live_smoke.py`: the live smoke runner (see Live smoke test).
- `tests/fake_station.py`: fake BOX station on `127.0.0.1` (plain HTTP, tests only).
- `tests/test_box.py`: unit tests.

Security defaults: `http://` is refused unless `allow_http=True` (tests only),
the password is never in `repr`, and HTTP 401/403 raises `AuthError` with no
retry (5 failures in 30 s lock the account). A 401 names what to check: the
user's Authentication Scheme Name must be `HTTPBasicScheme`, then the password.

## Running the server

From `mcp-n4-kit/`:

```
MCP_N4_USER=<user> MCP_N4_PASSWORD=<secret> \
  python3 -m mcp_n4.server --station MyStation=https://10.0.0.5
```

The server speaks newline-delimited JSON-RPC 2.0 on stdin/stdout. Credentials are
read from the environment only (`<prefix>_USER` / `<prefix>_PASSWORD`) and never
appear in output.

Connection policy belongs to the operator, not to the model. The model can only
call `n4_connect station=<NAME>` with a NAME you configured at start; it cannot
choose a URL, a credential prefix or TLS verification. Without any `--station`,
`n4_connect` refuses.

Flags:

- `--station NAME=URL` (repeatable): the only stations `n4_connect` may open. `URL`
  must be `https://`. Set NAME to the station's real `stationName`: `n4_connect`
  checks it by default (override per call with `expected_station`).
- `--credential-env PREFIX`: env var prefix for `<PREFIX>_USER` / `<PREFIX>_PASSWORD`
  (default `MCP_N4`).
- `--insecure-tls NAME` (repeatable): skip certificate verification for that
  configured station (self-signed certificates). Without it TLS is always verified.

- `--allow-writes`: registers the write tools; the default is read-only.
- `--write-scope ORD_PREFIX` (repeatable): ORD prefixes writes may touch.
- `--state-dir DIR`: journal and audit directory (default `~/.local/state/mcp-n4`).
- `--token-ttl SECONDS`: confirmation token lifetime (default 300).
- `--max-writes N`: executed writes allowed per session (default 200).
- `--progress-file PATH`: append JSON progress lines of long reads (`n4_inventory`) to PATH.
- `--allow-http-for-tests`: permits `http://` base URLs; for the fake station only.

### Getting the kit and registering it per project

The kit must exist on disk: a checkout on another branch may not contain `mcp-n4-kit/`. When
it is missing, add a worktree of `origin/main` instead of switching the shared checkout:

```
git -C /path/to/niagara-tools fetch origin
git -C /path/to/niagara-tools worktree add /path/to/niagara-tools-main origin/main
export MCP_N4_KIT=/path/to/niagara-tools-main/mcp-n4-kit
```

Register the server once per project, in that project's `.mcp.json` (Claude Code reads it from
the project root). Use an absolute `cwd` to the kit, and keep the credentials in the
environment that launches the client: never commit them in `.mcp.json`.

Example MCP client entry (Claude Code `.mcp.json`):

```json
{
  "mcpServers": {
    "n4": {
      "command": "python3",
      "args": ["-m", "mcp_n4.server", "--station", "MyStation=https://10.0.0.5",
               "--insecure-tls", "MyStation"],
      "cwd": "/path/to/niagara-tools/mcp-n4-kit",
      "env": {"MCP_N4_USER": "<user>", "MCP_N4_PASSWORD": "<secret>"}
    }
  }
}
```

Read tools (all `readOnlyHint`): `n4_connect`, `n4_describe_session`, `n4_navigate`,
`n4_read_slots`, `n4_list_links`, `n4_find_dangling_outputs`, `n4_bql_query`,
`n4_inventory`, `n4_session_retro_draft`.

For an inventory, use `n4_inventory` (networks, devices and points under `/Drivers`, with the
network's built-in `localDevice` flagged `local` and counted apart) or `n4_bql_query` first.
Both send `GET /ord/<url-encoded station:|slot:<base>|bql:select ...|view:file:ITableToCsv>`.
The query must be one `select`, `|` is refused, and rows are capped (default 5000).
`n4_navigate` and `n4_read_slots` cost one round trip per component; they report `elapsed_ms`.
`n4_navigate` takes an optional `types` list (e.g. `["bacnet:BacnetDevice"]`) that keeps only
children of those types plus the path to them, each kept entry marked `matched`.
`n4_read_slots` adds `value_display` (the station's display string) and returns that string as
`value` for complexes it cannot decode (e.g. a Modbus `dataAddress`: `Decimal:302`). One station session
is active per process; `n4_connect` replaces it (the old session is always closed
first, so a failed reconnect leaves no session). `n4_describe_session` lists the
configured station names. `n4_list_links` and
`n4_find_dangling_outputs` scan components up to `depth` levels below `ord`
(the ord itself is level 0). Links are seen when their target is at most one level
below `depth`; an out slot used only by a deeper target, or from outside the
subtree, is reported as dangling.

## Writing safely

Write tools exist only with `--allow-writes` (otherwise `tools/list` omits them
and calling one returns JSON-RPC -32601): `n4_create_component`, `n4_set_slot`,
`n4_invoke_action`, `n4_create_link`, plus the destructive class
(`destructiveHint: true`) `n4_remove_component`, `n4_rollback` and
`n4_save_station`. Example:

```
python3 -m mcp_n4.server --allow-writes --write-scope 'station:|slot:/Sandbox'
```

Every write call goes through these layers:

1. **Identity.** Refused unless the session's real `stationName` matched
   `expected_station` (default: the configured station NAME).
2. **Budget.** At most `--max-writes` executed writes per session.
3. **Scope.** Every target ORD must sit under a `--write-scope` prefix, matched on
   slot boundaries (`/A` does not cover `/AB`). With no prefix, every write is refused.
4. **Dry run.** `dry_run` defaults to true: the reply is the exact BOX ops and their
   inverse, a `plan_hash` and a `confirmation_token`; nothing is sent to the station.
5. **Token.** To execute, repeat the same call with `dry_run=false` and the token.
   It is single use, expires after `--token-ttl`, and is an HMAC under a per-process
   random key bound to the tool, the arguments, the plan and the expiry. If the
   station changed since the dry run the plan hash differs and the token is refused.
6. **Write-ahead journal.** Before any op is sent, an `intent` record
   `{batch_id, ts, tool, ops, inverse_plan, station_name, phase}` is appended to
   `<state-dir>/journal.jsonl`; if that fails nothing is sent (the token stays
   spent). After the reply a `result` record `{batch_id, phase, accepted, inverse,
   verdict}` follows, holding the real inverse (for example the server-assigned
   name). An intent without a result, or a result flagged `in_doubt` (the station
   reply was ambiguous or malformed), is an `in-doubt` batch: the station may or
   may not have applied it, so inspect it before retrying.
7. **Audit.** Every write call, dry, executed or refused, appends a line to
   `<state-dir>/audit.jsonl`; arguments whose key contains `pass`, `secret`, `token`
   or `credential` are replaced by `***`.
8. **Read-back.** After executing, the node is reloaded and the reply carries
   `{requested, accepted, observed, verdict}`: `verified`, `mismatch` (reported, not
   raised), `failed` (including any read-back error, reported as `readback_error`
   next to the `batch_id`), or `unverified` for the state-changing actions
   (`active`, `inactive`, `auto`) that have no slot to read back and for an
   ambiguous `checkLinks` reply (`in_doubt: true`; the new link is searched by its
   source and target slots).

The state directory is created with mode 0700 and the files with 0600, but only
when the server creates them. An existing `--state-dir` that is owned by another
user or is group/world-writable makes write mode refuse to start (exit 2); the
server never chmods a directory it did not create.

Tool rules:

- `n4_set_slot` writes Status slots whole (value plus status) and refuses a path
  ending in `/value` or `/status`. The slot must already be listed by the station;
  a plain slot at its type default is omitted by the station and is refused.
- `n4_invoke_action` allows only `set`, `active`, `inactive` and `auto`. `set` restores
  the previous `fallback` as its inverse; the others have none and the plan says so.
  `emergency*`, `save` and `restart` are refused; saving has its own tool,
  `n4_save_station`.
- `n4_create_component` follows a station rename on collision (the inverse and the
  read-back use the assigned name).

Destructive tools use the same pipeline:

- `n4_remove_component(parent_ord, name)` removes a child and its subtree. The plan
  snapshots the subtree to depth 3 (type, plain slots, `wsAnnotation`, and the links
  whose target is inside it) as data; it is the recorded inverse. Re-creation is
  limited to that snapshot and is not a full restore: deeper levels, runtime state
  and links coming from outside the subtree are not recreated (the plan says so).
- `n4_rollback(batch_id)` plans the inverse recorded in the journal as a new batch
  (`rollback_of` points to the original). It refuses an unknown batch, one already
  rolled back, a rollback itself, an `in-doubt` batch (it shows the journaled
  intent so a human decides) and a batch from another station. The current
  `--write-scope` is enforced on every ORD the batch touched, and each recorded
  handle must still belong to its ORD. Inverses: remove a created component or
  link, restore a slot or fallback, re-create a removed component from its
  snapshot (links only when both ends exist; the reply reports
  `relinks: {restored, skipped}`).
- `n4_save_station` invokes `save` on the root. The BOX reply to `save` is `null`
  and proves nothing, so with `--station-home NAME=PATH` (directory holding
  `config.bog`, repeatable) the tool compares mtime and sha256 before and after,
  polling up to 30 s, and returns `persisted: true|false|unknown` plus the
  `evidence`. Without a station home it returns `unknown` and says how to configure
  it. It needs at least one `--write-scope` like every write.

## Live smoke test

`tools/live_smoke.py` drives the real server over stdio (not the library) through the
public tool interface: every write is dry run, token, execute, verdict. It runs the
kitControl thermostat of B1199 inside a scratch folder `McpSmoke`, saves, removes the
folder, rolls the removal back (probe-only: its verdict is recorded but does not gate
the exit code), removes it again, saves and checks the folder is gone. Without
`--apply` it only prints the scenario and contacts nothing.

```
MCP_N4_USER=<user> MCP_N4_PASSWORD=<secret> \
  python3 mcp-n4-kit/tools/live_smoke.py --apply \
    --station <NAME>=https://<host> --station-home <NAME>=<station home dir> \
    --report smoke-report.json
```

Credentials are read only from the environment variables the server reads
(`<PREFIX>_USER` / `<PREFIX>_PASSWORD`, `--credential-env PREFIX`, default `MCP_N4`) and
their values are scrubbed from everything printed or written. The write scope is the
scratch folder plus the station root (needed to create and remove the folder). The
runner refuses to run if `McpSmoke` already exists. After a failed required step it skips
the rest and removes the folder it created. Exit 0 only when every required step is
`verified`; the JSON report lists each step, its verdict and `batch_id`.

## Session retros

Every session ends with a retro that proposes kit deltas (never applies them). The read-only
tool `n4_session_retro_draft` (both modes) drafts it from the server's audit and journal;
`python3 tools/new_retro.py --station NAME --state-dir DIR [--since TS] [--dry-run]` writes
`retros/YYYY-MM-DD-<station>-session.md` and a `pending` row in `retros/INDEX.md`. It refuses to
overwrite without `--force`. Nothing is staged as a GitHub issue automatically.

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
