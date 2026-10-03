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
v0.4.x: rollback fidelity (`partial` verdict, frozen-child and link-input reports).
v0.5.0: automatic version-tier write gate. v0.5.1-v0.5.4 (audit 2026-10-03): tokens bound to the
station session, plain-ORD write scope, authentication-failure latch (`--auth-cooldown`), nested
`n4_set_slot`, compare-before-restore rollback, write-branch tests.
v0.6.0: `n4_bql_query` `output_file`, duplicate and localized CSV headers.
v0.7.0: `n4_list_batches`, `snapshot_truncated`, `--load-wait` / `--http-timeout`.
v0.7.1: the write budget is per server process; docs drift fixed.
v0.7.2 (retro 2026-10-03): `tools/mutation_check.py` (mutation checks without stale bytecode), a
stdlib untested-branch audit, explicit per-PR review of medium units.

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
- `mcp_n4/tiers.py`: version detection (`/obix/about/` productVersion) and the tier write gate.
- `mcp_n4/retro.py` and `mcp_n4/templates/retro.template.md`: the session retro draft
  (`n4_session_retro_draft`).
- `tools/new_retro.py`: writes the session retro file and its `pending` row in `retros/INDEX.md`.
- `retros/`: session retros and `INDEX.md`.
- `tests/test_server.py`: protocol, tool and end-to-end stdio tests.
- `tools/live_smoke.py`: the live smoke runner (see Live smoke test).
- `tools/mutation_check.py`: runs a mutation-check command with no stale bytecode (see Run
  the tests).
- `tests/fake_station.py`: fake BOX station on `127.0.0.1` (plain HTTP, tests only).
- `tests/test_box.py`: unit tests.

Security defaults: `http://` is refused unless `allow_http=True` (tests only),
the password is never in `repr`, and HTTP 401/403 raises `AuthError` with no
retry (5 failures in 30 s lock the account). A 401 names what to check: the
user's Authentication Scheme Name must be `HTTPBasicScheme`, then the password.
The server also latches on any `AuthError` (login, `/obix/about/`, `/ord` GET, BOX call):
every station call is refused for `--auth-cooldown` seconds (default and minimum 30), with
the reason and the seconds left; `n4_describe_session` shows `auth_paused`. Only an
`n4_connect` with other credentials may try during the cooldown; a good login clears it.

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
- `--write-scope ORD_PREFIX` (repeatable): ORD prefixes writes may touch; each must be a plain
  slot ORD (`station:|slot:/Path`, nothing chained after it) or the server refuses to start.
- `--state-dir DIR`: journal and audit directory (default `~/.local/state/mcp-n4`).
- `--token-ttl SECONDS`: confirmation token lifetime (default 300; must be >= 1).
- `--auth-cooldown SECONDS`: after an authentication failure, refuse station calls this long
  (default and minimum 30, the lock-out window).
- `--max-writes N`: executed writes allowed per server process (default 200; must be >= 1);
  a reconnect does not reset it.
- `--station-home NAME=PATH` (repeatable): directory holding the `config.bog` of configured
  station NAME; lets `n4_save_station` prove persistence (see Destructive tools).
- `--allow-tier-b NAME` (repeatable): let writes run on that configured station although
  its version is tier B (4.15, 4.3). Only after a PoC matched that build.
- `--allow-tier-c NAME` (repeatable): the same for tier C (any other or undetected
  version). Only after `reg.loadContract`, `loadRoot` and a harmless scratch write
  succeeded on that build (METHODOLOGY section 5).
- `--load-wait SECONDS`: total wait for one component load (default 3, range 1-60); raise
  it for a slow or remote station.
- `--http-timeout SECONDS`: HTTP timeout of one station request (default 20, range 5-300).
- `--progress-file PATH`: append JSON progress lines of long reads (`n4_inventory`) to PATH.
- `--allow-http-for-tests`: permits `http://` base URLs; for the fake station only.

### Upgrading to v0.5.0

Before v0.5.0 the version-tier gate was manual, so a server started with `--allow-writes`
against a tier B (4.15, 4.3) or tier C station could execute writes. From v0.5.0 the same
command line refuses every execution on that station (dry runs still work) until it is
opted in. When upgrading:

1. List the tier B and tier C stations that this deployment already writes to
   (`n4_connect` reports `tier` and `tier_writes`).
2. Add `--allow-tier-b NAME` only for a tier B build where a PoC already matched; add
   `--allow-tier-c NAME` only for a tier C build where `reg.loadContract`, `loadRoot` and a
   harmless scratch write already succeeded (METHODOLOGY section 5).
3. Otherwise run the PoC or probe first, then restart with the opt-in.

Never add an opt-in just to clear a refusal: the opt-in records that the PoC or probe ran.

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
`n4_inventory`, `n4_list_batches`, `n4_session_retro_draft`.
`n4_list_batches` lists the journaled write batches newest first (`batch_id`, `tool`,
`station_name`, `ts`, `state`, `verdict`, `rollback_of`, `rolled_back_by`): the way to find a
`batch_id` for `n4_rollback`. It reads the journal only and needs no session.

For an inventory, use `n4_inventory` (networks, devices and points under `/Drivers`, with the
network's built-in `localDevice` flagged `local` and counted apart) or `n4_bql_query` first.
Both send `GET /ord/<url-encoded station:|slot:<base>|bql:select ...|view:file:ITableToCsv>`.
The query must be one `select`, `|` is refused, and rows are capped (default 5000).
CSV headers are the station's display names, which a localized station translates: the
queried `name` column is decoded by its position in the select list, `n4_inventory` reads its
columns by position, and a repeated header is kept as `Type#2`, `Type#3`. With
`output_file: "<name>.csv|.json"`, `n4_bql_query` writes the rows to `<state-dir>/bql/<name>`
(directory 0700, file 0600, never overwritten) and replies without `rows`.
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
2. **Version tier.** `n4_connect` detects the version (oBIX `productVersion`) and its
   tier (METHODOLOGY section 5). On tier B or C an execution is refused unless the
   operator opted the station in with `--allow-tier-b` / `--allow-tier-c`; a dry run
   still works and carries `tier_gate` saying the execution would be refused. An
   undetected version is tier C. Tier A (4.13, 4.14) is unaffected.
3. **Budget.** At most `--max-writes` executed writes per server process (a reconnect does
   not reset it).
4. **Scope.** Every target ORD must sit under a `--write-scope` prefix, matched on
   slot boundaries (`/A` does not cover `/AB`). With no prefix, every write is refused.
   Only plain slot ORDs pass: a `|` after `station:|slot:` (`|h:…`, `|slot:../…`) is
   refused, because a chained ORD resolves past the path the prefix was matched on.
5. **Dry run.** `dry_run` defaults to true: the reply is the exact BOX ops and their
   inverse, a `plan_hash` and a `confirmation_token`; nothing is sent to the station.
6. **Token.** To execute, repeat the same call with `dry_run=false` and the token.
   It is single use, expires after `--token-ttl`, and is an HMAC under a per-process
   random key bound to the tool, the arguments, the plan and the expiry. If the
   station changed since the dry run the plan hash differs and the token is refused.
   The plan names the station (`station: {name, base_url, session_id}`, a fresh id per
   `n4_connect`), so a token never executes after a reconnect or on another station.
7. **Write-ahead journal.** Before any op is sent, an `intent` record
   `{batch_id, ts, tool, ops, inverse_plan, station_name, phase}` is appended to
   `<state-dir>/journal.jsonl`; if that fails nothing is sent (the token stays
   spent). After the reply a `result` record `{batch_id, phase, accepted, inverse,
   verdict}` follows, holding the real inverse (for example the server-assigned
   name). An intent without a result, or a result flagged `in_doubt` (the station
   reply was ambiguous or malformed), is an `in-doubt` batch: the station may or
   may not have applied it, so inspect it before retrying.
8. **Audit.** Every write call, dry, executed or refused, appends a line to
   `<state-dir>/audit.jsonl`; arguments whose key contains `pass`, `secret`, `token`
   or `credential` are replaced by `***`.
9. **Read-back.** After executing, the node is reloaded and the reply carries
   `{requested, accepted, observed, verdict}`: `verified`, `partial` (a rollback that
   did not bring back every piece of configuration, listed in the reply), `mismatch`
   (reported, not raised), `failed` (including any read-back error, reported as `readback_error`
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
  ending in `/value` or `/status`. A nested slot (`grp/sp`) is loaded one level deeper
  so its previous value is the real one. The slot must already be listed by the station;
  a plain slot at its type default is omitted by the station and is refused (the
  refusal says so and routes to `n4_read_slots`, or to setting it once in Workbench).
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
  The plan loads one level deeper to see what the cut drops: anything there is listed in
  `snapshot_truncated: {depth, paths}` (first 20 paths, with the exact count in the note),
  also in the execution reply.
- `n4_rollback(batch_id)` plans the inverse recorded in the journal as a new batch
  (`rollback_of` points to the original). It refuses an unknown batch, one already
  rolled back, a rollback itself, an `in-doubt` batch (it shows the journaled
  intent so a human decides) and a batch from another station. The current
  `--write-scope` is enforced on every ORD the batch touched, and each recorded
  handle must still belong to its ORD. Before restoring a slot or fallback it reads
  the slot and refuses when it no longer holds what the batch wrote (a later change is
  never overwritten; restore it by hand with `n4_set_slot`). Inverses: remove a created
  component or link, restore a slot or fallback, re-create a removed component from its
  snapshot (links only when both ends exist; the reply reports
  `relinks: {restored, skipped}`). A frozen child (e.g. a writable's `proxyExt`) is
  never re-added: the read-back compares it with the snapshot and lists any
  difference in `frozen_config_not_restored`. A link-driven input whose link is not
  re-created (an end is missing, or its source was outside the removed subtree) is
  listed in `link_inputs_not_restored` with the value captured when the confirmed
  remove ran (the dry run previews it as `link_input_values`, outside the
  confirmation hash, because a link keeps changing it). Either
  list makes the verdict `partial`, never `verified`. A frozen child the read-back
  could not load is listed with `readback_error` (no `missing`) and makes the verdict
  `unverified`: a read error proves neither loss nor restore.
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

Test isolation rules:

- Replace a module or object attribute only through `mock.patch` (or `mock.patch.object`)
  with `addCleanup(patcher.stop)`, or inside a `with` block / context manager that restores
  it. Never assign the attribute directly: the replacement leaks into later tests and stays
  hidden while the suite runs in file order. `DestructiveCase.patch(obj, attr, new)` in
  `tests/test_tools_destructive.py` is the reference helper.
- Before merging a test-heavy change, also run the suite in a shuffled order with a few
  seeds (for example 1, 7 and 42), from the repo root:

```
python3 - 42 <<'EOF'
import random, sys, unittest
def flat(s):
    for t in s:
        yield from (flat(t) if isinstance(t, unittest.TestSuite) else [t])
tests = list(flat(unittest.defaultTestLoader.discover("mcp-n4-kit/tests")))
random.Random(int(sys.argv[1])).shuffle(tests)
sys.exit(not unittest.TextTestRunner().run(unittest.TestSuite(tests)).wasSuccessful())
EOF
```
- Run mutation checks without bytecode. A mutation restored with a plain copy within the
  same second, with the same file size, can leave a `__pycache__` `.pyc` that still matches
  the source's recorded mtime and size: the "restored" code keeps running the mutant
  (retro 2026-10-03 D1). Run both the mutated and the restored run through the helper, which
  removes every `__pycache__` under the kit and sets `PYTHONDONTWRITEBYTECODE=1`:

```
python3 mcp-n4-kit/tools/mutation_check.py -- python3 -m unittest discover -s mcp-n4-kit/tests
```

  By hand: `find mcp-n4-kit -name __pycache__ -prune -exec rm -rf {} +` after each restore
  and `PYTHONDONTWRITEBYTECODE=1` on each run. Restore with `cp -p` only when the saved
  copy's mtime differs from the mutant's.

Untested-branch audit (retro 2026-10-03 D2). Before each minor release (`0.X.0`), list the
lines the suite never runs, with the stdlib `trace` module (no `coverage` dependency; CI has
none). From the repo root, about one minute:

```
PYTHONDONTWRITEBYTECODE=1 python3 -m trace --count --missing --summary \
  --coverdir="$(mktemp -d)" --module unittest discover -s mcp-n4-kit/tests
```

The summary prints a line percentage per `mcp_n4.*` module; in `<coverdir>/mcp_n4.<module>.cover`
each never-run line is marked `>>>>>>`. Review the marked `raise`, `except` and refusal
branches first, starting with `tools_write.py` and `safety.py`: each one either gets a
characterization test or a one-line reason it stays untested in the release PR. The
percentage is a pointer, not a gate. Delete the coverdir afterwards.

## Versioning

- The kit version lives in `mcp_n4/__init__.py` (`__version__`).
- There is no kit-local changelog. Entries go in the root `CHANGELOG.md` under a heading of
  the form ``### <kind> — `mcp-n4-kit` vX.Y.Z: <summary>`` (kind: Added, Changed, Fixed, ...).
- When another release is in flight (another writer owns the root `CHANGELOG.md` or
  `VERSION`), put the entry text in the PR body for the release owner to fold.
- Each work unit merges as its own PR; a medium-risk unit is reviewed at its PR, not left
  to the slice budget (METHODOLOGY section 8).

## Evidence (niagara-research)

- B1177: BOX recipe.
- B1192: decode and Status.
- B1179: auth and lock-out.
- B1197: MCP synthesis.
- B1199: live PoC on station `LLM`.

Note: this kit is not covered by the repo pre-push retro gate yet.
