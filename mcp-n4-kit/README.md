# mcp-n4-kit

An MCP server plus a Claude Code skill that let an AI agent read and write
Niagara N4 stations over the BOX JSON protocol (route R-A). No station module
is needed: the client speaks to `POST <station>/box/` with HTTP Basic auth.

## Status

T1: the stdlib-only BOX client library (then `mcp_n4/box.py`; since S17 the Go server's
`go/internal/mcpn4/box`) and its tests against an in-memory fake station. T2: the stdio MCP server with read-only station tools.
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
v0.7.2 (retro 2026-10-03): mutation checks without stale bytecode (a Python helper, retired
with the Python tests; the Go rule is in METHODOLOGY section 8), an untested-branch audit,
explicit per-PR review of medium units.
v0.7.3: the mutation helper exits 3 when its command cannot be launched (helper retired in T9).
v0.8.1: a failed write keeps its batch id, a partial batch reports what was sent and its inverse, BOX replies capped at 32 MB.
v0.9.0: station file reads and Px views: `n4_read_file`, `n4_list_px_views`, `n4_px_summary`.
v0.9.1: read truth fixes: `n4_find_dangling_outputs` resolves slot-path link sources and reports
an output an unresolved source may feed as `unknown`, not dangling; a BQL body cut at the byte
cap keeps whole CSV records only; an HTML, empty or ragged BQL answer is an error, not rows.
v0.9.2: BOX transport truth: an answer that is not HTTP, or a body cut before its
Content-Length, is a `transport failure`, never an internal error or a complete result; a
`pollchgs` or load reply of the wrong shape, or nested deeper than 1000 levels, is a
`malformed BOX reply`.
v0.9.3: a non-UTF-8 credential is refused with a fixed message that no longer reveals a
character position (R8-D19); the Go server serves `n4_navigate`, `n4_read_slots`,
`n4_list_links` and `n4_find_dangling_outputs` (slice S5), after S4's `n4_connect` and
`n4_describe_session`.
v0.9.4 (issue #448): one unreadable `journal.jsonl` line (not UTF-8, or a batch id that cannot
key a batch) no longer fails `n4_list_batches` and every rollback: the readable batches are
listed, each bad line is an `unreadable journal line N: <error class>` row and the reply says
the list is incomplete; `n4_rollback` refuses (`journal line unreadable`) only the batches such
a line may belong to; a batch id with a trailing newline is malformed. Both servers.
v0.9.5 (issue #466): one unreadable `audit.jsonl` line (not UTF-8) no longer fails
`n4_session_retro_draft`: the audit log is split and decoded line by line like the journal, the
bad line is left out, and the draft's session summary says it is incomplete and names each one
(`unreadable audit line N: UnicodeDecodeError`; no byte of the line is echoed). Both servers.
Go port status, at v0.9.5 (docs/go-port-plan.md sections 3 and 20): the Go server
(`go/cmd/mcp-n4`, built to `go/bin/mcp-n4`) serves every read and write tool with the same
flags, `--allow-writes` included (S0-S14), and the #304 in-doubt matrix is proven on both
servers (S15). Since S16 it is the default, and since S17 the only server: the Python server
(`python3 -m mcp_n4.server`) and its `MCP_N4_IMPL=python` switch are removed
(docs/go-port-plan.md section 23).

## Methodology and skill

- [METHODOLOGY.md](METHODOLOGY.md): route ladder, safety layers mapped to code and tests,
  live rules, the mandatory session checklist and version tiers. Read it before writing.
- [docs/go-port-plan.md](docs/go-port-plan.md): the history of porting the server to Go
  (package layout, parity strategy, slices S0-S17, go/no-go criteria).
- [skill/SKILL.md](skill/SKILL.md): the Claude Code skill launcher. Install it with
  (from the repository root; add `--dry-run` to preview, `--force` to overwrite a diverged copy):

```
scripts/install-skill.sh --skill mcp-n4
```

## Layout

- `METHODOLOGY.md`, `skill/SKILL.md`: see above.
- The server is the Go module's `go/cmd/mcp-n4` (built to `go/bin/mcp-n4`) over
  `go/internal/mcpn4/...`: `box` (BOX client), `jsonrpc` and `cli` (stdio protocol and flags),
  `readtools` and `writeops` (the tools), `safety`, `journal`, `tiers`, `bql`, `files`, `px`,
  `retro` (the session retro draft and its embedded template).
- `go/cmd/mcp-n4/retro-draft`: prints the session retro draft, and with `--write` writes the
  session retro file and its `pending` row in `retros/INDEX.md`.
- `retros/`: session retros and `INDEX.md`.
- `go/internal/mcpn4/blackbox`: the Go black-box harness (the real server over stdio against
  the `boxstation` fake) and the protocol, connect, read-tool, tier and end-to-end stdio tests.
- `go/cmd/mcp-n4/live-smoke`: the live smoke runner (see Live smoke test).
- `go/cmd/mcp-n4/n4call`: the bench client (see Bench client).
- `go/internal/mcpn4/readtools/readtoolstest/boxstation`: the fake BOX station on `127.0.0.1`
  (plain HTTP, tests only).
- `tests/fixtures/`: recorded station answers (BQL CSVs, a Px view), a journal the Python server
  wrote, and the frozen answers of the removed Python server (`oracles/`) that Go tests read.
  The kit has no Python code left.

Security defaults: `http://` is refused unless `allow_http=True` (tests only),
the password is never in `repr`, and HTTP 401/403 raises `AuthError` with no
retry (5 failures in 30 s lock the account). A 401 names what to check: the
user's Authentication Scheme Name must be `HTTPBasicScheme`, then the password.
The server also latches on any `AuthError` (login, `/obix/about/`, `/ord` GET, `/file/` GET,
BOX call):
every station call is refused for `--auth-cooldown` seconds (default and minimum 30), with
the reason and the seconds left; `n4_describe_session` shows `auth_paused`. Only an
`n4_connect` with other credentials may try during the cooldown; a good login clears it.

## Running the server

Build the Go server once (from the repository root: `cd go && go build -o bin/ ./cmd/mcp-n4`),
then:

```
MCP_N4_USER=<user> MCP_N4_PASSWORD=<secret> \
  go/bin/mcp-n4 --station MyStation=https://10.0.0.5
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
the project root). Use an absolute path to the Go binary, and keep the credentials in the
environment that launches the client: never commit them in `.mcp.json`.

Example MCP client entry (Claude Code `.mcp.json`):

```json
{
  "mcpServers": {
    "n4": {
      "command": "/path/to/niagara-tools/go/bin/mcp-n4",
      "args": ["--station", "MyStation=https://10.0.0.5", "--insecure-tls", "MyStation"],
      "env": {"MCP_N4_USER": "<user>", "MCP_N4_PASSWORD": "<secret>"}
    }
  }
}
```

Read tools (all `readOnlyHint`): `n4_connect`, `n4_describe_session`, `n4_navigate`,
`n4_read_slots`, `n4_list_links`, `n4_find_dangling_outputs`, `n4_bql_query`,
`n4_inventory`, `n4_list_batches`, `n4_session_retro_draft`, `n4_read_file`,
`n4_list_px_views`, `n4_px_summary`, `n4_spy_page`, `n4_station_health`.
`n4_list_batches` lists the journaled write batches newest first (`batch_id`, `tool`,
`station_name`, `ts`, `state`, `verdict`, `rollback_of`, `rolled_back_by`): the way to find a
`batch_id` for `n4_rollback`. It reads the journal only and needs no session.

For an inventory, use `n4_inventory` (networks, devices and points under `/Drivers`, with the
network's built-in `localDevice` flagged `local` and counted apart) or `n4_bql_query` first.
A network is a row of the networks query (a `driver:DeviceNetwork` of any type), wherever it sits
below `base`: with a base above the driver container it is named by its path below the base
(`Plant/Drivers/BacnetNetwork`), a device or point under no network has `network: null`, and a
plain folder is never listed as a network.
Both send `GET /ord/<url-encoded station:|slot:<base>|bql:select ...|view:file:ITableToCsv>`.
The query must be one `select`, `|` is refused, and rows are capped (default 5000).
A body over 32 MB is cut at its last complete record (a newline inside a quoted cell does not
end one), so a truncated result (`truncated: true`) never carries a torn last row.
An answer that is not a CSV table is an error, never zero rows: an HTML page (by Content-Type
or by its first bytes: a login or error page), an empty body (ITableToCsv writes a header row
even for zero rows; a header-only body is 0 rows), or a record whose cell count differs from
the header's (a ragged row is not padded or trimmed).
CSV headers are the station's display names, which a localized station translates: the
queried `name` column is decoded by its position in the select list, `n4_inventory` reads its
columns by position, and a repeated header is kept as `Type#2`, `Type#3`. With
`output_file: "<name>.csv|.json"`, `n4_bql_query` writes the rows to `<state-dir>/bql/<name>`
(directory 0700, file 0600, never overwritten) and replies without `rows`.
Px views and other station files (METHODOLOGY section 9): `n4_list_px_views(root)` runs
`select slotPath, displayName, pxFile, media from baja:PxView` and returns `display_name`,
`px_file` (e.g. `file:^px/General.px`) and `media` per view; `slot_path` / `owner_path` are
null when the station answers `NULL` (a PxView is a property value, not a component).
`n4_read_file(file_ord, max_bytes=2097152, output_file=None)` sends
`GET /file/<path>` for `file:^px/General.px`, `station:|file:^px/General.px` or
`px/General.px`. It refuses `..`, `|`, `:`, absolute paths, directories, names ending in
`.` or a space (Windows aliases of another name) and `.bog` / `.bog.gz` before any request; returns text for text types and size, `sha256` and `content_type` for
the rest; `status` is `ok`, `forbidden` (403) or `not_found` (404). With
`output_file: "<name>.<ext>"` the complete bytes go to `<state-dir>/files/<name>` (0600,
never overwritten; a truncated read is not saved). `n4_px_summary(file_ord, max_bytes,
max_items=200)` fetches a `.px` and returns its media, imported modules, canvases and
`viewSize`, layers, widget and binding counts, bound ORDs grouped by component, images,
included Px, charts and label texts, without the XML.
Station health in one call: `n4_station_health(base="station:|slot:/", max_not_ok=50)` composes
the reads above into one reply, each section answered on its own. `system` is the station's
SystemService (`/Services/PlatformServices/SystemService`) by its slot names: `currentCpuUsage`
and `overallCpuUsage` in percent, `numCpus`, `totalPhysicalMemory` and `freePhysicalMemory` as the
station reports them (it names no unit; the magnitude on the 4.15 station measured fits
kilobytes) plus the derived
`memory_used_percent`, `niagaraVersion`, `hostId`, `model`, `stationName`, `systemTime`, `osName`,
`javaVmVersion`. `points`, `networks` and `devices` each run one query that selects `status.bits`
(the status as a number, which no locale translates) and report `total`, `ok`, `by_flag` (a count
per flag: `disabled`, `fault`, `down`, `alarm`, `stale`, `overridden`, `null`, `unackedAlarm`; a
row with two flags counts under both), `null_only`, `not_ok_count` and `not_ok`. Every row is one
of three: `ok` (no flag), `null_only` (the flag `null` alone: a point with no value, such as an
unlinked input, which is not a fault and is not listed) or not ok (at least one flag other than
`null`), so `ok + null_only + not_ok_count = total`. `not_ok` lists the not-ok entries up
to `max_not_ok` ( `truncated` says the list was cut, `rows_truncated` that the query hit
`max_rows`, so the counts are lower bounds). Networks and devices also carry `disabled` (rows
whose `enabled` is false) and name, type and `enabled` per entry. A status the station did not
answer as a number counts under `unknown` and as not ok, never as ok. `totals` are the `n4_inventory` totals of
the same rows. `capacity` is the Global Capacity table of the spy metrics page (`spy:/metrics`):
`resources` (`type`, `used`, `limit`, and `unlimited: true` where the station shows no limit) and
`recount` (when the capacity recount last ran or failed). `capacity` is `null` with a
`capacity_note` when the page is unreachable (the user may not read spy pages) or holds no such
table: it is never a guessed count. The capacity counts are the licensing counter's, which counts
by its own rules (it does not decrement on a delete until a recount), so they can differ from
`totals`. A section that fails is `{"error": "..."}` and the others still answer; only an
authentication failure fails the call.
Spy pages (the station's diagnostics tree, the pages of the Workbench Spy view):
`n4_spy_page(path="", max_rows=500, max_bytes=2097152)` sends one `GET /ord?spy:/<path>` with
the session's credentials and returns the page as data: `title`, `tables` (each with its `title`,
`header` and `rows` of cells), `links` (the child pages, each `path` ready to pass back), `actions`
(the POST forms the page offers, by clean spy path without the query or fragment of the form's
action: named, never invoked; the content of a form field, `textarea`, `select`, `button` or an
`input` value, is never returned, wherever it sits) and `text` (what sits outside the tables). An empty `path` is the index. It only reads:
a path is plain names separated by `/`, and a scheme or `:`, `|`, `..`, a query string (`?`, `&`,
`=`, `#`), `%`, `;`, a backslash, any Unicode control character, invalid UTF-8 and the known action pages (`util/gc`,
`util/forceThreadDump`) are refused before any request. `status` is `ok`, `forbidden` (403: the
user may not read spy pages, which need a super user), `not_found` (404, or the station's "Spy Not
Found" page), `not_html` or `parse_timeout` (the page was fetched but not parsed within
`timeout_s`); a 401 is an authentication error like on any other read. `truncated`
is true when a cap cut the reply and `truncated_by` names the caps: `max_bytes`, `max_rows` (rows
across all tables), `tables` (200), `row_cells` (100 per row), `links` (500), `actions` (100),
`name` (a link name or title over 200 characters), `cell`, `text`. The caps bound the reply as a
whole, whatever the page holds. Spy pages are diagnostics, not an API: their layout differs between
Niagara versions (this reader was measured on 4.15), so read the tables as found.
`n4_navigate` and `n4_read_slots` cost one round trip per component; they report `elapsed_ms`.
`n4_navigate` takes an optional `types` list (e.g. `["bacnet:BacnetDevice"]`) that keeps only
children of those types or of their subtypes (`history:HistoryExt` keeps every history extension)
plus the path to them, each kept entry marked `matched`. The ancestry is read from the station's
type registry, one call for the types of the listing; when it cannot be read the filter keeps exact
types only and the reply says so in `types_note`. A common `baja` type is described without its
supertype, so `baja:Component` does not keep a `baja:Folder`.
`n4_read_slots` adds `value_display` (the station's display string) and returns that string as
`value` for complexes it cannot decode (e.g. a Modbus `dataAddress`: `Decimal:302`). A slot that
is itself a `baja:Status` (a proxy extension's `status`, for example) reads `status_flags`, the names of the
flags set (`fault`, `down`, `alarm`, `unackedAlarm`, ...), and `status_ok` false when any is set. One station session
is active per process; `n4_connect` replaces it (the old session is always closed
first, so a failed reconnect leaves no session). `n4_describe_session` lists the
configured station names. `n4_list_links` and
`n4_find_dangling_outputs` scan components up to `depth` levels below `ord`
(the ord itself is level 0). Links are seen when their target is at most one level
below `depth`; an out slot used only by a deeper target, or from outside the
subtree, is reported as dangling. A link source given by handle (`h:`) or by absolute
slot path (`station:|slot:/Folder/A`) is resolved; while any link reading `out` has a
source in another form (listed in `unresolved_links`), every out slot not proven used
is reported under `unknown`, never as dangling.

## Writing safely

Write tools exist only with `--allow-writes` (otherwise `tools/list` omits them
and calling one returns JSON-RPC -32601): `n4_create_component`, `n4_add_history`,
`n4_copy_component`, `n4_set_slot`, `n4_set_flags`, `n4_invoke_action`, `n4_create_link`, plus the destructive class
(`destructiveHint: true`) `n4_remove_component`, `n4_rollback` and
`n4_save_station`. Example:

```
go/bin/mcp-n4 --allow-writes --write-scope 'station:|slot:/Sandbox'
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
   Two tools scope differently. `n4_create_component` may create the scope root itself
   under its parent (scope `…/Sandbox`, `parent_ord` its parent, `name` `Sandbox`), so a
   scope can name a folder that does not exist yet; it is refused when that root already
   exists, since the station would rename the new one outside the scope. The same scope
   cannot remove that root again or roll its create back (`n4_remove_component` and
   `n4_rollback` check `parent_ord`, so a scope never deletes itself in one call): remove
   it under a wider scope or by hand.
   `n4_create_link` is scoped on `target_ord` only: the link is stored on the target and
   the source is only read, so the source may sit outside the scope but must still be a
   plain slot ORD. A refusal names the end (`target_ord:` or `source_ord:`). A rollback
   that re-creates a removed component's links still checks both ends of each link.
5. **Dry run.** `dry_run` defaults to true: the reply is the exact BOX ops and their
   inverse, a `plan_hash` and a `confirmation_token`; nothing is sent to the station.
6. **Token.** To execute, repeat the same call with `dry_run=false` and the token.
   It is single use, expires after `--token-ttl`, and is an HMAC under a per-process
   random key bound to the tool, the arguments, the plan and the expiry. If the
   station changed since the dry run the plan hash differs and the token is refused.
   The plan names the station (`station: {name, base_url, session_id}`, a fresh id per
   `n4_connect`), so a token never executes after a reconnect or on another station.
7. **Write-ahead journal.** Before any op is sent, an `intent` record
   `{batch_id, ts, tool, ops, inverse_plan, station_name, phase, targets}` (a rollback adds
   `rollback_of` and, when present, `relinks` and `components`) is appended to
   `<state-dir>/journal.jsonl`; if that fails nothing is sent (the token stays
   spent). After the reply a `result` record `{batch_id, phase, accepted, inverse,
   verdict}` follows, holding the real inverse (for example the server-assigned
   name). An intent without a result, or a result flagged `in_doubt` (the station
   reply was ambiguous or malformed), is an `in-doubt` batch: the station may or
   may not have applied it, so inspect it before retrying. Any failure after the
   intent is written is reported with its `batch_id` (reply and audit line, outcome
   `error`). When op k of a multi-op batch fails, the error names the ops already
   sent, their replies and their inverse, and an in-doubt `result` record
   (`sent`, `failed_op`, partial `accepted` and `inverse`) is journaled; a rollback
   that stops after re-creating components lists the components that now exist.
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
  so its previous value is the real one. A plain slot at its type default is omitted by
  the station's load; the tool then reads the slot's declared type and default from the
  station's type registry (BOX `reg`/`loadTypes`, walking supertypes), so such a slot
  (e.g. a history extension's `enabled=false`) can be set. A dynamic slot, an action or
  a name no type declares is still refused. Value types: `baja:Double`, `baja:Boolean`,
  `baja:String`, `baja:StatusNumeric`, `baja:StatusBoolean`, `baja:Format` (pattern
  string, e.g. a history ext `historyName`), `baja:RelTime` (whole milliseconds,
  e.g. `300000` for a 5-minute history `interval`), `baja:Integer` (a whole number in
  the Java int range, e.g. a Modbus `deviceAddress`) and `baja:Simple`: any other simple,
  given as its encoded string and sent under the slot's own type (an enum tag such as
  `signedInteger` for `dataType`, a Facets string) or under `simple_type` when the
  value replaces the type (`driver:LinearConversion` with `0.1;0.0` for a proxy
  `conversion`; the rollback restores the old type, and the read-back checks the type).
  A struct is refused as a Simple: set its children, e.g. a Modbus address through
  `proxyExt/dataAddress/addressFormat` (`modbus`) and `proxyExt/dataAddress/address`
  (`40005`: send it as `baja:Simple`, which also takes a whole number, because an MCP
  client may turn the string "40005" into a number). Enum tags come from the type's
  range, e.g. Modbus `DataTypeEnum` is `integerType`, `signedInteger`, `floatType`.
- `n4_set_flags(ord, slot, flags | add/remove)` sets the flags of one slot of a component (a
  property, an action or a topic; a name, not a path) with the station's set-flags
  sync op. `flags` is the complete new set as letters, each once, in any order; `""`
  clears every flag. Instead of `flags`, `add` and/or `remove` name the letters to change
  in the current set and keep the others (for example `add: "s"` makes the slot a summary
  slot without dropping its `r`); a call gives one form, never both, and no letter in both. Letters: `r` readonly, `t` transient, `h` hidden, `s` summary,
  `a` async, `n` no-run, `d` default-on-clone, `c` confirm-required, `o` operator,
  `x` execute-on-change, `f` fan-in, `A` no-audit, `p` composite, `R`
  remove-on-clone, `m` metadata, `N` non-critical, `L` link-target, `1` to `4`
  user-defined. The plan notes show the current and the requested flags with their
  names. The current flags are the ones the station sends on the slot; a frozen slot
  at its declared flags carries none, so they are read from its type contract
  (registry channel). The inverse restores exactly those flags. Refused: an unknown
  or repeated letter, a slot the component does not have, a set equal to the current
  one, and a slot whose current flags cannot be read (no flags on the slot and no
  readable type contract), because no faithful inverse exists. The read-back loads
  the slot again: `verified` only when the flags it holds are the requested set,
  else `mismatch` with `requested` and `observed`. Proven live on 4.15.6.20 for a
  dynamic slot, a frozen property and a frozen action (set, save, `config.bog`,
  rollback); on another build, try it on a scratch slot first.
  The station applies it only for a user with admin write on the component.
- `n4_invoke_action` allows only `set`, `active`, `inactive`, `auto` and `ping`. `set`
  restores the previous `fallback` as its inverse; the others have none and the plan
  says so. `ping` on a driver network, device or point device ext asks the driver to
  check the device and changes no setting.
  `emergency*`, `save` and `restart` are refused; saving has its own tool,
  `n4_save_station`.
- `n4_create_component` follows a station rename on collision (the inverse and the
  read-back use the assigned name). With `proxy_ext` (a type spec such as
  `modbusCore:ModbusClientNumericProxyExt`) the point is created with that driver
  extension in the same add, so a read-only field register is a `control:NumericPoint`
  that never writes; the read-back checks the extension type.
- `n4_add_history(points, history_name, interval_ms)` adds one interval history
  extension per point (1 to 50), created already configured (`historyName`,
  `interval`, `enabled`, default true) in one batch under one confirmation token.
  It refuses a point that already has a child named `ext_name` and, when the
  pattern only uses `%name%` / `%parent...name%` tokens, a history name that would
  repeat across the batch. The inverse removes every extension; the read-back
  compares each one. Capacity stays at the default (500 records, Roll).
- `n4_copy_component(parent_ord, name, new_names)` adds the configuration of one
  child again under each new name (1 to 30) in one batch under one confirmation
  token: type, plain slots and child components up to 6 levels deep (a Modbus
  device down to its point addresses). Runtime slots are left out by their flags, read
  from the station's type definitions through the registry channel: a frozen slot
  declared readonly, transient or default-on-clone (`r`, `t`, `d`) or made transient,
  and a dynamic slot flagged transient or remove-on-clone (`t`, `R`). Flags an
  integrator set otherwise travel with the copy. The plan notes list each slot left
  out with its flags and call out a readonly slot holding a non-default value.
  Links are not copied either; the plan counts the links left behind, and a deeper
  source is refused, as is a station whose type definitions cannot be read. Change
  what differs per copy afterwards (e.g. `deviceAddress` with `n4_set_slot`). The
  inverse removes every copy under its assigned name; the read-back loads each copy
  and compares its whole configured subtree with the plan, listing the paths that differ.

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
  never overwritten; restore it by hand with `n4_set_slot`). The flags an
  `n4_set_flags` batch set are checked the same way and restored by a set-flags op,
  verified by reading the slot back. The rollback of an `n4_create_component` batch
  removes the component with everything under it, so it refuses when that would
  destroy later work: a later batch (not itself rolled back) whose targets are the
  component or under it, or a link or child component on it that its create did not
  make (a Workbench edit no journal records; a frozen slot its type declares does not
  count). The refusal names the component, each later batch (`batch_id (tool)`,
  newest first) and each extra child or link: roll the later batches back first,
  newest first, and remove by hand what no batch made; to drop it all instead,
  `n4_remove_component` it (its snapshot keeps an inverse). The live check reads the
  component's direct children only: a hand edit deeper down is not seen. Inverses: remove a created
  component or link, restore a slot, its flags or a fallback, re-create a removed component from its
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

`go/bin/live-smoke` (built from `go/cmd/mcp-n4/live-smoke`; formerly `tools/live_smoke.py`)
drives the real server over stdio (not the library) through the
public tool interface: every write is dry run, token, execute, verdict. It runs the
kitControl thermostat of B1199 inside a scratch folder `McpSmoke`, saves, removes the
folder, rolls the removal back (a required step: a rollback that does not verify fails the
run), removes it again, saves and checks the folder is gone. Without
`--apply` it only prints the scenario and contacts nothing. It launches the Go server
(`--go-bin PATH`, else `MCP_N4_GO_BIN`, else the `mcp-n4` next to `live-smoke`, that is
`go/bin/mcp-n4`; missing is exit 2). On a station that is not tier A (METHODOLOGY section 5) pass the
operator's `--allow-tier-b NAME` or `--allow-tier-c NAME`: the runner hands it to the server and
never adds one itself. A step that does not verify prints its reason under its line. `--report`
needs an existing directory (exit 2 before anything runs).

```
(cd go && go build -o bin/ ./cmd/mcp-n4 ./cmd/mcp-n4/live-smoke)
MCP_N4_USER=<user> MCP_N4_PASSWORD=<secret> \
  go/bin/live-smoke --apply \
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

## Bench client

`go/bin/n4call` (built from `go/cmd/mcp-n4/n4call`) calls tools on a server it starts over
stdio, for a live probe on a station that is not registered in the agent's MCP configuration:

```
go/bin/n4call --calls calls.jsonl -- go/bin/mcp-n4 --station <NAME>=https://<host> ...
```

Each line of the file is `{"tool": NAME, "args": {...}}`; with `"write": true` the call is sent
with `dry_run=true` and then executed with the `confirmation_token` it returned
(`--dry-run-only` stops after the dry run). Output is one JSON line per request (`tool`,
`phase`, `error`, `result`); credential values are masked; the run stops with exit 1 at the
first tool error. The server command line after `--` is the operator's: scopes, tier opt-ins
and station homes are whatever it names, n4call adds nothing.

## Session retros

Every session ends with a retro that proposes kit deltas (never applies them). The read-only
tool `n4_session_retro_draft` (both modes) drafts it from the server's audit and journal;
`go/bin/retro-draft --write --station NAME --state-dir DIR [--since TS] [--dry-run]` (build it with
`cd go && go build -o bin/ ./cmd/mcp-n4/retro-draft`) writes
`retros/YYYY-MM-DD-<station>-session.md` and a `pending` row in `retros/INDEX.md`, both atomically.
It refuses to overwrite without `--force`; `--out-dir` names another retros directory and
`--date` another day (default today, UTC). Nothing is staged as a GitHub issue automatically.
The draft is the server's own drafting code (`go/internal/mcpn4/retro`).

## Run the tests

```
(cd go && go test -count=1 ./internal/mcpn4/... ./cmd/mcp-n4/...)
```

The whole suite is Go. `go/internal/mcpn4/blackbox` drives the real server as a separate process
over stdio against the `boxstation` fake (the test binary re-executes itself as the server, so
no prebuilt `go/bin` is needed). CI runs it in the `go` job (`go test -count=1 ./...`).

Test isolation rules:

- Restore anything a test replaces (a package variable, an environment variable, a station
  hook) with `t.Cleanup` or `t.Setenv`, never by hand at the end of the test: a failing test
  would leave the replacement in place for the next one.
- Before merging a test-heavy change, also run the suite in a shuffled order with a few seeds,
  for example `go test -count=1 -shuffle=1 ./internal/mcpn4/...` with 1, 7 and 42.
- Mutation checks follow METHODOLOGY section 8 (`go test -count=1` on both runs,
  `git diff --exit-code` on the mutated file before reading GREEN).

Untested-branch audit (retro 2026-10-03 D2). Before each minor release (`0.X.0`), list the
server lines no test runs with `cd go && go test -coverprofile=cover.out ./internal/mcpn4/... &&
go tool cover -func=cover.out` (the black-box suite drives a separate binary, so it is not in
the profile). Review the uncovered `return err` and refusal branches first, in `writeops` and
`safety`: each one either gets a test or a one-line reason it stays untested in the release
PR. The percentage is a pointer, not a gate. Delete `cover.out` afterwards.

## Versioning

- The kit version is the server version in `go/internal/mcpn4/registry/registry.json`
  (`server.version`), which `initialize` and `n4_describe_session` report.
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

Note: `mcp-n4-kit/` is not covered by the repo pre-push retro gate yet; its Go port under `go/`
is, like every `go/**` change (`.githooks/pre-push`, build-relevant paths).
