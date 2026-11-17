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

Every `tools/call` is checked against the tool's `inputSchema` before anything reaches a station,
read and write tools alike. A missing required argument, a wrong type, a number out of its
`minimum`/`maximum`, a value outside its `enum` (`argument full_policy must be one of roll, stop;
got 'drop'`; for a list of enum values the item is named, `sections[1]`) and an argument the tool
does not declare (`unknown argument 'capacity_records' for n4_add_history; did you mean
'capacity'?`) are refused with JSON-RPC -32602: an argument is never silently dropped. The objects
a tool keeps open stay free-form: `alarm_values` (tag -> ordinal), `expect_histories`,
`expect_alarms`, `wire_sheet`, `weekly` and the items of the `n4_set_schedule` and `n4_set_tags`
lists, which their tools check themselves; so do the untyped `value`, `arg`, `default_output` and
`capacity` (a number or `"unlimited"`).

### Install to a stable path and register a station per harness

A harness should never run `go/bin/mcp-n4` from a worktree: removing or rebuilding it changes or
breaks every server registered from it. Install the binary once, then register each station:

```
go/bin/mcp-n4 install                       # copies itself to ~/.local/bin/mcp-n4 (--prefix DIR)
~/.local/bin/mcp-n4 register --station lab --harness claude -- \
  --station LAB=https://10.0.0.5 --credential-env LAB_N4 --load-wait 10
~/.local/bin/mcp-n4 register --station lab --harness pi            # reuses the launcher
~/.local/bin/mcp-n4 register --station lab --harness gentle-shell
```

- `install` writes a temporary file next to the target, fsyncs it, makes it 0755 and renames it
  over `<prefix>/mcp-n4`: a running server keeps the old file, a failure leaves it in place. A
  symlink there (into a checkout, say) is replaced by a regular file, and install says so; temporary
  files of an interrupted install older than an hour are removed.
- `register` writes `~/.config/mcp-n4/<station>-run.sh`, which sources the env file
  (`--env`, default `~/.config/mcp-n4/<station>.env`, holding `<PREFIX>_USER` / `<PREFIX>_PASSWORD`,
  chmod 600) and execs `--bin` (default `~/.local/bin/mcp-n4`) with the server flags given after
  `--`. The flags are checked like a server start; a flag carrying a value of the env file is
  refused, so the launcher holds no credential. Without flags after `--` an existing launcher is
  kept only when its `exec` line runs the installed binary (`--bin`); a launcher that runs another
  binary (a worktree's, as hand-written launchers did) is refused, naming both, unless `--force`:
  pass the server flags after `--` to rewrite it. An explicit `--bin` or `--env` is never ignored
  there: it must match what the launcher execs or sources, or register refuses (even with
  `--force`, which cannot re-render a launcher without its flags).
- The MCP entry is `mcp-n4-<station>`: `claude` adds it to the Claude Code project `--project`
  (default the current directory) in `~/.claude.json`, the local scope `claude mcp add` uses (the
  project must have been opened in Claude Code once); `pi` and `gentle-shell` add it to
  `~/.pi/agent/mcp.json` and `~/.gentle-shell/agent/mcp.json` with `"exposure": "direct"`. These
  files are hand-edited, so only the bytes of the entry added or replaced change (with the comma
  and line break joining it to its neighbours); every other server keeps its bytes exactly.
- The launcher execs the installed binary. A `--bin` inside a git checkout (a `.git` file or
  directory at or above it, after symlinks) is refused unless `--force`: removing or rebuilding a
  worktree would break every server registered from it.
- Re-running with the same arguments changes no byte. A launcher or entry that differs is refused,
  naming the first differing line or the differing fields; `--force` replaces it.
- A config file that is a symlink (kept in a dotfiles repository) stays a link: the file it
  resolves to is replaced. A file that is not exactly one JSON object (trailing data, a second
  document) or has a duplicate key on the entry's path is refused with nothing written; a CRLF
  file gets CRLF in the inserted entry. An entry equal in value in another key order (as
  `claude mcp add` writes it) is unchanged.
- Restart the harness (or reload its MCP servers) to pick up a new entry.
- Limits: the credential check refuses a flag holding an env value of 4 bytes or more (a
  shorter one only as a whole flag); a HOME that is itself a git checkout makes the default
  `--bin` look like a checkout (use `--force`); Claude Code rewrites `~/.claude.json` while it
  runs, so register with Claude Code closed or the entry may be overwritten.

The server speaks newline-delimited JSON-RPC 2.0 on stdin/stdout. Credentials are
read from the environment only (`<prefix>_USER` / `<prefix>_PASSWORD`) and never
appear in output.

Connection policy belongs to the operator, not to the model. The model can only
call `n4_connect station=<NAME>` with a NAME you configured at start; it cannot
choose a URL, a credential prefix or TLS verification. Without any `--station`,
`n4_connect` refuses.

Flags:

- `--version`: prints `mcp-n4 <version> (git <sha>)` and exits 0: the `server_version` and
  `server.build` that `n4_describe_session` reports (`<sha>` ends in `+modified` for a dirty tree,
  `unknown` for a binary built without VCS information).

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
  a reconnect does not reset it. `n4_save_station` neither needs nor spends a write, so a job
  that spent the budget can still save; every write reply (dry run and execution) carries
  `write_budget` `{executed, max, remaining, save_counts: false}`. Plan a bulk job against
  `remaining`; restart the server for a new budget. A batch is one write, except
  `n4_replicate_points` (one per add) and `n4_rollback` (one per re-created component and link).
- `--station-home NAME=PATH` (repeatable): directory holding the `config.bog` of configured
  station NAME; lets `n4_save_station` prove persistence (see Destructive tools).
- `--niagara-home NAME=PATH` (repeatable): Niagara installation directory of configured station
  NAME (the one holding `security/licenses`), when it is reachable from the machine that runs the
  server; lets `n4_license_info` add when each licence was generated and its version. Only the
  attributes of the root element of each `*.license` file are read.
  It also lets `n4_station_runtime` compare the jars of `PATH/modules` with the modules the
  station loaded; only `META-INF/module.xml` of a jar is read.
- `--allow-tier-b NAME` (repeatable): let writes run on that configured station although
  its version is tier B (4.15, 4.3). Only after a PoC matched that build.
  A 4.15 laboratory station is tier B too: its launcher needs `--allow-tier-b NAME`, or
  every execution is refused with "version tier refuses writes" (dry runs still work).
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
`n4_list_px_views`, `n4_px_summary`, `n4_spy_page`, `n4_station_health`,
`n4_read_history`, `n4_export_history`, `n4_read_audit`, `n4_histories_audit`, `n4_license_info`, `n4_read_alarms`, `n4_list_alarm_classes`, `n4_event_report`,
`n4_alarm_health`,
`n4_alarm_text_check`,
`n4_commissioning_report`, `n4_list_schedule_kinds`, `n4_read_schedule`,
`n4_station_runtime`, `n4_read_tags`,
`n4_neql_query`, `n4_list_tag_dictionaries`.
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
A `where` literal is decoded by BQL (`$20` is a space) while slot names are stored escaped: to
match `Numeric$20Writable40005` write `name = 'Numeric$2420Writable40005'` (`$24` is a literal `$`).
An empty result whose quoted literal holds a `$xx` escape carries a `hint` naming that form.
Besides a station subtree, `base` may be `alarm:` (the alarm database) or
`history:/<station>/<history name>` (one history; the id as `n4_read_history` checks it), each
also accepted after `station:|`. The query there is `select <columns> [where ...] [order by ...]`:
no type is needed and a `from` clause is refused (live, `history:` silently ignores it and
answers the whole history, `alarm:` answers HTTP 404), e.g.
`select timestamp, source, sourceState, ackState where source like '*slot:/Lab/P/*' order by timestamp desc`
or `select timestamp, value, status where timestamp.millis >= 1794808800000 order by timestamp desc`.
It sends `GET /ord/<url-encoded alarm:|bql:<query>|view:file:ITableToCsv>` (or
`history:<id>|bql:...`), the same CSV view and reply shape as a station base, and stays
read-only: a non-select, `|`, a control character or an over-long query is refused before any
request, and so is any other scheme (`file:`, `ip:`, a history id that is not two slot names);
a base under `station:` that is not a slot path keeps its slot-path refusal. Niagara decodes
`$xx` / `$uxxxx` escapes inside a BQL body (`BqlQuery.make`), after the ORD is split on a literal
`|`: the `from` rule of a table base, the no-`from` rule of `n4_neql_query`'s `bql` select and the
length bound also read the decoded text (`bql.DecodeForBql`, one pass of `EscUtil.unescape`; a
malformed escape is refused; live, `select uuid$20from$20control:NumericPoint` on `alarm:` passed
the no-`from` rule before), while a decoded `|` or control character is data (`like '*$7c*'`).
`n4_inventory` still reads station subtrees only.
History records: `n4_read_history(history_id | ext_ord, limit=20, period | start/end, summary=false,
summary_max_rows=100000)` returns
the last `limit` records (at most 1000) of one history, newest first (`order: newest_first`), so
a rollout proves records arrive, not only that the extension runs. Give exactly one of
`history_id` (`/<station>/<history name>`, two slot-name segments as `BHistoryId` parses them;
anything else, `|`, `?`, `;` and spaces included, is refused before any request; the id is matched
escaped, so a `$xx` escape in it is a name character and is never decoded) or `ext_ord`
(a `station:|slot:/...` history extension ORD, resolved with one
`select slotPath, historyConfig.id from history:HistoryExt` at the component holding it). It
sends `GET /ord/<url-encoded history:<id>[?<range>]|bql:select timestamp, value, status order by
timestamp desc|view:file:ITableToCsv>`, then the same ORD with `select count(*)` for
`total_records` (`null` with `total_records_note` when that count fails). The range is
`period=<id>` (`today`, `last24Hours`, `yesterday`, `weekToDate`, `lastWeek`, `last7Days`,
`monthToDate`, `lastMonth`, `yearToDate`, `lastYear`: the ids `BDynamicTimeRange` accepts) or
`start=<t>;end=<t>` (BAbsTime text such as `2026-11-15T00:00:00.000-06:00`, or a span back from now,
`30m`, `24h`, `7d`, sent as UTC BAbsTime and echoed in `range`; `start` later than `end` is refused), not both. A `history_id` with no `/` is
a history name of the connected station (`NumericWritable` is `/<station>/NumericWritable`; a refusal
names the text given). Rows are
`{timestamp, value, status}` read by position, as the station renders them (localized display
text); `truncated: true` means the range holds more records than were returned. A history the
station cannot read is an error naming the id and how to list ids
(`n4_bql_query` `select historyConfig.id from history:HistoryExt`); an AuthError stays one.
`summary: true` (retro 2026-11-16 M2, "a cuanto llego" in one call) adds one read of every record of
the range, `select timestamp.encodeToString, value.encodeToString, status order by timestamp desc`
(ISO-8601 times with the station's offset, `21.0` instead of the localized `21,0`), and returns
`summary {count, complete, records_scanned, numeric, min {value, timestamp}, max {value, timestamp},
first and last {value, timestamp, status}, status_counts}` over the whole range, whatever `limit`
cut the rows; a tie keeps the earliest record (when the value was first reached). A non-numeric
history has `numeric: false`, `min`/`max` null and a `note`. A record whose status has the `null`
flag (no value: a new COV history writes `0.0 {null}` records at start) and a `nan` or `inf` record
(a faulted source) are counted (`count`, `status_counts`) but left out of `min`, `max`, `first` and
`last` (retro 2026-11-16 M16); then `excluded_from_extremes {null, nan_inf}` counts them, a `note`
says so, and `first_record` / `last_record` are the raw ends when they differ. Other flags (fault,
down, stale, alarm) keep their value and stay in. A range larger than `summary_max_rows`
is summarized from its newest records with `complete: false` and a `note`, and `range_min` /
`range_max` come from `select min(value), max(value) where status.null = false` over the whole
range (localized text, no time: 4.15 has no `group by` and no aggregate that returns the record).
The `where` leaves out null-status records as the summary does (retro M16b; checked live on
4.15.6.20); a `nan` or `inf` record still counts in the station's aggregate. A failed summary read is
`summary: null` with `summary_note`; the rows are still returned.
There is no `output_file`: an export of whole histories is `n4_export_history`.
History export: `n4_export_history(history_ids, period | start/end, max_rows=100000)` (station-audits S2) writes 1 to 20 histories (ids checked like `history_id`, before any request) to ONE CSV file for client reports: one `select timestamp.encodeToString, value.encodeToString, status order by timestamp desc` per history, the newest `max_rows` records of each written oldest first as `history_id,timestamp,value,status` (ISO-8601 times with the station's offset, `21.0` not `21,0`; CSV quoting, CRLF). The server names the file `<state-dir>/exports/<station>-<UTC time>.csv` (directory 0700, file 0600, never overwritten; the caller never passes a path) and replies `{path, rows_total, per_history [{history_id, rows, truncated}], bytes}`; a history that cannot be read stops the export and removes the file. It reads the station only (`readOnlyHint`, like `n4_bql_query`'s `output_file` and `n4_semantic_export`).
Alarms: `n4_read_alarms(limit=50, source, source_glob, alarm_class, source_state, ack_state, since,
until, open_only=true, state_counts=false)` returns alarm records newest first (`order: newest_first`), by default the open
ones: it sends `GET /ord/<url-encoded alarm:|bql:select timestamp, uuid, source, sourceState,
ackState, alarmClass, priority, alarmData.msgText where <clause> order by timestamp
desc|view:file:ITableToCsv>`, then `select count(*) where <clause>` for `total` (`null` with
`total_note` when that count fails). The default clause is
`(ackState != 'acked' or sourceState != 'normal')`; a `source_state` (`normal`, `offnormal`,
`fault`, `alert`) or `ack_state` (`acked`, `unacked`, `ackPending`) filter replaces it, and
`open_only: false` drops it. `source` (`station:|slot:/Drivers/Net`, each segment a slot name)
becomes `(source like 'slot:/Drivers/Net' or source like 'slot:/Drivers/Net/*' or source like
'local:*slot:/Drivers/Net' or source like 'local:*slot:/Drivers/Net/*')`: that component and
everything below it, in both forms a station stores a source (`slot:/...` and
`local:|station:|slot:/...`), never a sibling such as `/Drivers/NetX`; the clause holds no `|`,
which would end the bql part of the ORD (HTTP 400), `alarm_class` (a slot name) `alarmClass = '<name>'`, and `since` / `until` (BAbsTime text)
`timestamp.millis >= <epoch ms>` / `<=` (a span back from now, `30m`, `24h`, `7d`, is resolved first).
`source_glob` (ORD characters and `*` only: no quote, space, `$` or `|`; `*` also matches an escaped
character) becomes `source like '<glob>'`, e.g. `*slot:/Drivers/Net/D*`. Every value is checked
before any request against its grammar and then as BQL reads it: Niagara decodes `$xx` / `$uxxxx`
inside a BQL body (`BqlQuery.make`; live, `alarm_class=x$27$20or$20alarmClass$20like$20$27$2a`
matched 27 of 27 records), so `source` and `alarm_class` are decoded once like `EscUtil.unescape`
(`bql.DecodeForBql`; `$2427` is the literal text `$27`, as the station reads it) and refused when
they decode to a quote, space, `*`, `|` or control character (to match an escaped slot name
write `$24` for its literal `$`: `My$2420Device` matches the stored `My$20Device`); no quote, wildcard or BQL of the
caller reaches the station; `where` echoes the clause sent. Rows are
`{timestamp, uuid, source, source_state, ack_state, alarm_class, priority, msg_text}` read by
position, as the station renders them (`msg_text` may be a raw `%lexicon(...)%` pattern; once the
source is back to normal the record holds the toNormal text), plus `timestamp_iso`, `normal_time` and
`ack_time` (`encodeToString`: ISO-8601 with the station's offset, `null` for the unset time).
`state_counts: true` (default false: one more read) adds `state_counts`: it reads `select sourceState.encodeToString, ackState.encodeToString where <clause>`
and counts every matching record per state: `{source_state: {normal, offnormal, fault, alert},
ack_state: {acked, unacked, ackPending}, records_counted, complete}` (`null` with
`state_counts_note` when that read fails).
Alarm events: `n4_event_report(source | source_glob, history_id, since, until, margin_minutes=15, max_events=10)` answers "how high did it get during that alarm" in one call: it reads `select timestamp.encodeToString, uuid, source, ... normalTime.encodeToString, ackTime.encodeToString, user` (filters checked as `n4_read_alarms` checks them; every record, open or not), one event per record (the alarm database keeps one record per uuid, updated in place on return to normal and on ack), then for each event the `n4_read_history` summary (count, min, max with its time, first, last) over `[alarm_time - margin, normal_time + margin]` (up to now while still active). The history is `history_id`, or the history extension on the alarm extension's parent or on the source itself (`select slotPath, historyConfig.id from history:HistoryExt`, once per source); none found or a failed read is a null `summary` with `summary_note` on that event, never a failed report.
Audit trail: `n4_read_audit(target | target_glob, slot, user, operation, period | start/end, limit=100)` answers "who changed overAmpsDelay from 5 s to 3 s" in one call. It reads `select timestamp.encodeToString, operation, target, slotName, oldValue, value, userName ... order by timestamp desc` over `history:/<station>/AuditHistory` (the history `AuditHistoryService` writes), and returns rows `{timestamp, operation, target, slot, old_value, value, user}`, newest first, with `truncated` and `total_records` (the same filters under `select count(*)`). `target` is stored as the component's slot path without the `slot:` scheme (`/Folder/Point`): the `target` argument takes a component ORD and matches it and everything below it, while `target_glob` is a like pattern over that stored path. `operation` is one of `Changed`, `Added`, `Removed`, `Renamed`, `Reordered`, `Flags Changed`, `Facets Changed` or `Invoked`; logins go to the security audit history, not here. Every filter is checked on its decoded form (`DecodeForBql`) before any request. By default the audit history keeps 500 records and rolls.
Alarm health: `n4_alarm_health(window="24h", chatter_threshold=10, unacked_days=3, top=10)` answers
"how healthy are my alarms" with two fixed reads of `alarm:` and no write:
`select source, normalTime.encodeToString where timestamp.millis >= <now - window>` and
`select source, sourceState.encodeToString, ackState.encodeToString, alarmClass, timestamp.encodeToString`
over the open records (`(ackState != 'acked' or sourceState != 'normal')`), each capped at 100000
records. The server groups them: `chattering` `[{source, transitions, records}]` (a source stored as `slot:/a` or `local:|station:|slot:/a` is
reported once as `station:|slot:/a`; a record in the
window is one transition out of normal, plus one back when its normal time is set; sources at or
over `chatter_threshold`), `offnormal` `[{source, source_state, alarm_class, since, records}]` (sources
with an open record not back to normal, once per source, `since` the earliest, oldest first), `unacked_older` `[{source, alarm_class, ack_state, alarm_time,
age_days}]` (not acknowledged, alarm time more than `unacked_days` days old) and `top_sources`
`[{source, count}]`. Each list keeps `top` entries (1-50); `counts` holds the full numbers and
`truncated_note` names each cut list or capped read. `window` is a span (`30m`, `24h`, `7d`).
Histories audit: `n4_histories_audit(base, exclude_services=true, stale_factor=3, stale_min=10m, cov_max_age=24h, capacity_warn=0.9, max_histories=500)` answers "what is wrong with the histories" in one read-only call. It reads `select slotPath, type, enabled, historyConfig.id, historyConfig.capacity.encodeToString, historyConfig.fullPolicy.encodeToString, historyConfig.interval.encodeToString, lastRecord.timestamp.encodeToString from history:HistoryExt` and `select slotPath from control:ControlPoint` below `base`, then one `select count(*)` per distinct record-capped history id that has recorded (`count_reads`, at most `max_histories`). It returns `no_history` (control points with no history extension right below them; points under `slot:/Services` are left out unless `exclude_services` is false, `services_excluded` counts them), `disabled` (`enabled` false), `stale` (last record older than the larger of `stale_factor` x the collection interval and `stale_min`, or than `cov_max_age` for a COV history, or never recorded; `stale_min` exists because the history service writes in batches, so a 5 s history's newest record shows up about 20 s late), `near_capacity` (records at least `capacity_warn` of a `1:<records>` capacity; `severity` warning for full policy `stop`, where recording stops when full, info for `roll`, where the oldest records are overwritten) and `colliding_ids` (one history id recorded by several extensions), each capped at 100 items with `counts` holding the totals. A read cut at its cap sets `truncated` and leaves `no_history` unchecked (`counts.no_history` null with `no_history_note`). Capacity `0:0` (unlimited) and `2:<bytes>` histories are not measured.
`n4_list_alarm_classes()` runs `select slotPath, name, priority, ackRequired, totalAlarmCount,
openAlarmCount, inAlarmCount, unackedAlarmCount from alarm:AlarmClass` and
`select slotPath, type from alarm:AlarmRecipient` at the station root; `priority` is decoded to
`{toOffnormal, toFault, toNormal, toAlert}` (0-255) and `ack_required` to the same keys with
booleans (a value in another form is returned as its text). A class routes alarms to a recipient
(console, email, station and other `alarm:AlarmRecipient` types) through a link stored on the
recipient, from the class's `alarm` topic to the recipient's `routeAlarm` action: each recipient
is loaded two levels deep (at most 50, `recipients_truncated` beyond), a link source given by
handle is resolved against one load of the folder holding the classes, and each class lists
`recipients: [{path, type, link_name}]`; `recipients` lists each recipient with its `classes`,
`unresolved_links` the `routeAlarm` links whose source is not a listed class.
`n4_list_alarm_classes(audit=true)` answers "where do my alarms go" from the same reads (no extra
request): `unrouted_classes` `[{path, name}]` are the classes no recipient is linked to
(`unrouted_note` when an unresolved link or an unread recipient may route one of them),
`audit_counts` counts classes, unrouted classes, recipients read and recipients with problems,
and each recipient gains `settings`, `at_default`, `masked_slots` and `problems`. `settings` fails
closed: a value is returned only for an allowlisted slot of a known type (every recipient type:
`transitions`, `daysOfWeek`, `timeRange`, `routeAcks`; `email:EmailRecipient` also `to`, `cc`, `bcc`,
`emailAccount`, `language`; `alarm:StationRecipient` also `remoteStation`, `lastFailureCause`,
`status`, `persistent`, `retryInterval`; `alarm:ConsoleRecipient` also `status`, `defaultTimeRange`).
Every other slot, and every slot of any other recipient type (a custom webhook or Telegram
recipient included), reads `"<set>"` or `""` and is named in `masked_slots`. An email recipient's
`to`, `cc`, `bcc`, `emailAccount` and a station recipient's `remoteStation` read as `""` when the
station omits them at their empty default, named in `at_default`. `problems` reads the raw values
and never quotes one off the allowlist: an email recipient with no address in `to`, `cc` or `bcc`,
or one that is not an email address; a station recipient with no `remoteStation`; `transitions` or
`daysOfWeek` empty; an empty `url`, `chatId` or `webhook` slot; a last delivery failure
(`lastFailureCause`, its text quoted only where allowlisted).
Creating an alarm class has no dedicated tool; the recipe is three ordinary writes, each with its
dry run and token: `n4_create_component(parent_ord: "station:|slot:/Services/AlarmService",
name: "Critical", type: "alarm:AlarmClass")`; `n4_set_slot(ord: "station:|slot:/Services/AlarmService/Critical",
slot: "priority", value: "toOffnormal=10;toFault=20;toNormal=200;toAlert=30;", value_type: "baja:Simple")`
(the `BAlarmPriorities` encoding, every transition followed by `;`) and
`n4_set_slot(..., slot: "ackRequired", value: "7", value_type: "baja:Simple")` (the
`BAlarmTransitionBits` encoding is hexadecimal bits: 1 toOffnormal, 2 toFault, 4 toNormal,
8 toAlert; `7` is all but toAlert, `f` all four); then route it with
`n4_create_link(source_ord: "station:|slot:/Services/AlarmService/Critical", source_slot: "alarm",
target_ord: "station:|slot:/Services/AlarmService/ConsoleRecipient", target_slot: "routeAlarm")`,
once per recipient. Check the result with `n4_list_alarm_classes`.
`n4_alarm_text_check(base="station:|slot:/", max_rows=5000)` is kit rule LC9 made mechanical, the live twin
of `bog-nav <bog> alarm-text` (same rule names and rows): one `select slotPath, toOffnormalText, toFaultText,
toNormalText from alarm:AlarmSourceExt` below `base` (subtypes included); each finding is `{level: WARN, rule, path,
slot, token, text}`: `parent-path` for a `%...%` token rooted at no `BAlarmRecord` field (`%parent...%`, `%out.value%`;
the texts resolve against the alarm record), `empty-tonormal` for a `toOffnormalText` with an empty `toNormalText`
(the return to normal blanks `msgText`), `bad-format` for a text BFormat cannot parse (a `%` that never closes:
it shows as `ERROR <text>`; write `%%` for a literal percent); `%alarmData.<key>%`, record fields, `%lexicon(...)%`
and `%%` pass.
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
answer as a number counts under `unknown` and as not ok, never as ok. `points` also carries
`proxy`, the points whose proxy extension is not `control:NullProxyExt` (the same query selects
`proxyExt.type`; `proxy_unknown` appears when the station left that cell empty): a licence counts
proxy points, not every control point. `totals` are the `n4_inventory` totals of the same rows,
with `rows_truncated` when a query feeding them was cut. `drivers=true` (no extra request: it
groups the rows of those three sections) adds `drivers`, driver health per network: each entry is
`network` (its slot path), `name`, `type`, `status` (its flags), `enabled`, `devices` (its field
devices, the local device left out: `total`, `down`, `fault`, `disabled` (the flag or `enabled`
false), `unknown`), `points` (its proxy points only, a control point with `control:NullProxyExt`
counting nowhere: `total`, `stale`, `fault`, `down`, `disabled`, `unknown`), and `devices_bad`
(`{device, status}`, plus `enabled: false` when so) and `points_bad` (`{point, status}`), the ones
in any category, up to `max_bad` each (default 20; `*_bad_truncated` says a list was cut). A status
with several bits counts once in each of its categories. `drivers.totals` sums the networks and
adds `unassigned_points` (proxy points under no network), `proxy_unknown` and `rows_truncated`;
`drivers` is `null` with a `drivers_note` when one of the three sections failed.
`platform=true` adds `platform`, platform capacity and what is about to expire, from five more
spy page reads (a station user allowed to read spy pages) and one BOX load: `heap` (`used_kb` and `committed_kb`
from `spy:/sysInfo`, `max_kb` the sum of the defined maxima of the heap pools on
`spy:/util/JMX Info`, `fraction` = used over max), `cpu` and `memory` (the system section again;
the CPU usage slots hold -1 until the SystemService is polled on subscription, so the system
section loads it once more when they are missing, and `cpu` is `null` with `cpu_note` when they
stay unmeasured), `history_db` (`histories` and `records` summed from `spy:/localHistory`; `bytes`
is always `null`: no read-only page gives the size of the history files), `licence` (the license
section's nearest expiry; `never_expires: true` when nothing is dated), `certificates` (the
station's own key store certificates first, `store: "station"`, read from the
`CertificateExpiryPoint` of each `Services/SecurityService/certificates` entry: its value is
`days_left`, `expires` is the station's date plus it, `status` is the point's; then the user and
system trust store certificates of `spy:/securityInfo/Cryptography Info`, `{alias, store,
issued_by, expires, days_left}`; each store nearest expiry first, up to 50), `modules` (`count` of
the loaded parts; the list too with `include_modules: true`) and `warnings` (heap at 90% or more;
the licence and each station or user certificate expired or within `expiry_warn_days`, default 30;
the system trust store, the JDK's roots, as one informational summary line). An item that cannot be read is
`null` with a `*_note` saying why. Without `platform` the reply is unchanged.
`capacity` is the Global Capacity table
of the spy metrics page (`spy:/metrics`): `resources` (`type`, `used`, `limit`, `unlimited: true`
where the station shows no limit, and `live`) and `recount` (when the capacity recount last ran
or failed). The two counts of a resource are different things. `used` is the licence's own tally:
quote it against `limit` for licensing; it only grows (removing a component does not decrement
it) until a capacity recount, about every ten minutes on a licensed station, or a station
restart. `live` is what the station really holds now: networks, devices and proxy points of the
whole station (`null` for links, histories and schedules, which this tool does not count, when
`base` is a subtree, or when the count is not whole). `may_be_inflated: true` with an
`inflation_note` says the tally can be above reality: a `used` greater than its `live`, or a
recount that has never run. `capacity` is `null` with a `capacity_note` when the page is
unreachable (the user may not read spy pages) or holds no such table: it is never a guessed
count. It is never silently partial either: the page is read with a row cap above what its byte
cap can hold, `capacity_truncated: true` and the note say when the page was cut before it was
read whole (resources may be missing; a page cut before the table is "unknown", not "no table"),
and rows of the table that do not match its header are counted in the note. `license` is the
licence manager page in short (`n4_license_info` has the whole of it): `licenses` (`name`,
`expires`, `valid`, with `state` when not valid, and `days_left` counted from the station's own
date, the day of `system.systemTime`; up to 100, `licenses_count` when the page has more),
`nearest_expiry` (the earliest dated expiry among licences, certificates and features),
`expired` (what is already past) and a `warning` when the nearest expiry is 60 days away or
fewer or already past, when a licence is not reported valid, or when an expiry could not be read
as a date; `note` says what the reading could not do (an unknown station date, which leaves
`days_left` and `expired` `null`; a cut page, a missing section, unread lines). A licence page
that was not read is `{"error": "..."}` saying which case it is (the user may not read spy pages,
the station has no such page, the answer is not HTML or not a licence page), never an empty
licence; a refused login on it fails the call like on any section. A section that fails is `{"error": "..."}` and the others still answer; only an
authentication failure fails the call.
Spy pages (the station's diagnostics tree, the pages of the Workbench Spy view):
`n4_spy_page(path="", max_rows=500, max_bytes=2097152)` sends one `GET /ord?spy:/<path>` with
the session's credentials and returns the page as data: `title`, `tables` (each with its `title`,
`header` and `rows` of cells), `links` (the child pages, each `path` ready to pass back), `actions`
(the POST forms the page offers, by clean spy path without the query or fragment of the form's
action: named, never invoked; the content of a form field, `textarea`, `select`, `button` or an
`input` value, is never returned, wherever it sits) and `text` (what sits outside the tables). An empty `path` is the index. It only reads:
a path is plain names separated by `/`, and a scheme or `:` (except one `module:Type` segment directly
under `sysManagers/registryManager/types/`, the registry's type pages, e.g.
`sysManagers/registryManager/types/control:NumericWritable`; the `sysManagers/registryManager/types`
page links to each of them), `|`, `..`, a query string (`?`, `&`,
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
The licence in one call: `n4_license_info(feature=None, max_features=50)` reads the station's
licence manager (the spy page `sysManagers/licenseManager`, one GET, read-only) and answers
`host_id`; `certificates` (`name`, `expires`, `valid`); `licenses` (`name`, `expires`, `valid`,
`days_left`) with `license_kind`, the qualifier the page gives them (for example `perpetual`);
`brand`, `owner` and `project` (the `tridium:brand` and `tridium:about` features);
`nearest_expiry`, the earliest dated expiry among licences, certificates and features (`kind`,
`name`, `expires`, `days_left`, and `same_date`: how many share that date); `expired`, what is
already past (up to 50, `expired_count` when there are more); and `features` (`count`, `matched`,
and `list` with each feature's `name`, `expires` and `attributes` by key, capped by `max_features`
with a `truncated` marker). `feature` lists one feature by its exact `vendor:name`
(`tridium:bacnet`), or every feature whose name holds the text. `valid` is true only when the page
writes the state `valid`; any other state is `valid: false` with `state`, the text as written.
`expires` is `never`, a date (`YYYY-MM-DD`) or, when the page writes anything else, that text as
written with `days_left: null`: such entries are counted in `note` and are in neither
`nearest_expiry` nor `expired`. Every `days_left` counts calendar days from the station's own date
(`station_date`, the day of the SystemService `systemTime`, which must be a whole timestamp), never
from the machine that runs the server: 0 on the expiry day, negative once past, `null` for `never`
or when the station date could not be read (`station_date_note` says why, and `expired` is `null`:
unknown, not none). `maintenance` is what the licence states of the software maintenance agreement
(SMA): `sma_attributes`, one entry per feature that has `sma.*` attributes (`feature`, and its
`attributes` by key), up to 50 with `sma_feature_count`; `sma_exempt_features`, the names of the
features whose `sma.exempt` is true, each once, up to 50 with `sma_exempt_count` (the exemption is
per feature: that one feature needs no active agreement, and it says nothing of the licence as a
whole); `sma_truncated` when a list was cut; `feature` (a feature named after maintenance, when
the licence has one); and `expires`, the end date of the agreement, which is `null` unless a local
licence file states it: the page carries no such date (the vendor documentation shows it as the
`maintenanceExpiration` attribute of a licence file). The page does not carry when a licence was
generated nor its version either. Those three facts are only available where the installation's
files are reachable: with `--niagara-home NAME=PATH` for the connected station, each licence adds
`generated` and `version`, and `maintenance` adds `expires`, `days_left` and `license` (the file
that states the earliest `maintenanceExpiration`), read from the root element of the file of that
licence's name under `security/licenses` and nothing else of it (never its features or its
signature). Only a regular file of exactly the licence's name is opened, and a local file that
matches no licence of the page is neither read nor named. Without the option the fields are
omitted and `generated_note` says why; with it, `generated_note` says per licence why its facts
are missing: no file of that name, a link or another non-regular file, a name that differs in
letter case, a file over 1 MiB or not readable as a licence, one issued for another host than the
page's `host_id`, or past the 100 files read in one call.
`status` is `ok`, `not_recognized` (the page answered but is no licence manager page) or a spy
status (`forbidden`, `not_found`, `not_html`, `parse_timeout`): only `ok` carries the licence
fields, so an unreachable page is never an empty licence. A line of the page is read whole or not
at all: `note` counts the lines that matched no known pattern (two entries on one line, a name
with a bracket or a brace, a section heading out of its place) and names a page that was cut, a
missing section, an expiry that is not a date and a list that was cut (the layout was measured on
Niagara 4.15 and is not a contract). The reply is bounded whatever the page holds: every string
taken from the page is stripped of control characters and cut with the marker `...[cut]` (200
characters for a name, an expiry, a state or the host id; 64 for an attribute key and 100 for its
value), a feature keeps 30 attributes (`attributes_truncated`), 100 certificates and 100 licences
are read (`certificates_count`, `licenses_count` when the page has more), every list also stops at
a byte budget, and the whole reply stays under 2 MiB for any page up to the 2 MiB read cap.
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
flags set (`fault`, `down`, `alarm`, `unackedAlarm`, ...), and `status_ok` false when any is set.
A BOX load leaves out a declared (frozen) property still at its default, so a fresh component
would read no slots: `n4_read_slots` lists those after the loaded slots, and `n4_navigate` under
each component or struct slot whose children it lists (with `handle` null), from the type contract
the station's registry gives (`loadTypes`, read once per session for every listed type and its
supertypes; a type the station answers as unknown, or leaves out of a reply that answered other
types, is not asked for again; one left out of a reply that answered nothing, or after a failed
call, is). The sub-entries of a declared struct default are marked `default` too and may show
`type: null` (the contract gives their value, not their type).
Each is marked `default: true` with the contract's default value; when the contract gives no
default value (a component default such as `proxyExt` is not sent) `value` is null and a `note`
says so, never an invented value. `defaults_note` says when a contract could not be read (the
registry failed, or the station does not know or did not describe a type) and names a declared
default the reader could not decode, which is left out; the read itself never fails on one. The
`n4_rollback` read-back counts an omitted frozen property as its declared default; when the
contract cannot be read the read-back reports `readback_error` instead of a `mismatch`. One station session
is active per process; `n4_connect` replaces it (the old session is always closed
first, so a failed reconnect leaves no session). `n4_describe_session` lists the
configured station names and, with or without a session, a `server` block: `build` (the git
revision the binary was built from, `+modified` for a dirty tree, `unknown` without a VCS
stamp), `built_at` (the commit time, else the binary's mtime at start), `started_at` and
`binary` (the executable's path). When the binary on disk is not the one serving, the reply
of `n4_describe_session` and of a successful `n4_connect` carries the same one-line top-level
`warning` naming both builds: new or changed tools reach a client only after its MCP server is
restarted. The git revision stamped in the binary on disk decides when both builds carry one
(a different revision warns, the same revision never does); otherwise the binary's mtime
later than the server start does. A missing or unreadable binary never warns or fails. `n4_list_links` and
`n4_find_dangling_outputs` scan components up to `depth` levels below `ord`
(the ord itself is level 0). Links are seen when their target is at most one level
below `depth`; an out slot used only by a deeper target, or from outside the
subtree, is reported as dangling. A link source given by handle (`h:`) or by absolute
slot path (`station:|slot:/Folder/A`) is resolved; while any link reading `out` has a
source in another form (listed in `unresolved_links`), every out slot not proven used
is reported under `unknown`, never as dangling. An alarm or history extension reads its
point's `out` without a link, so a component with an `alarm:...AlarmExt` or
`history:...HistoryExt` child is listed in `consumed_by_extension` (with the extension type)
instead of `dangling`; the commissioning report's `dangling_outputs` check skips it too
(backlog OD007).

### Registering the server per harness

The server entry is the same `mcpServers` object in every harness; only the file and the tool names
differ. Install the skill into the harness first (`scripts/install-skill.sh --skill mcp-n4 --harness
<claude|pi|gentle-shell|all>`, which also writes the prompt file that gives pi and gentle-shell a bare
`/mcp-n4`), then check it with `scripts/check-skill-drift.sh --skill mcp-n4 --all`.

- **Claude Code:** the project's `.mcp.json` (the example in "Getting the kit and registering it per project"). The tools are named
  `mcp__<server>__<tool>`, the server name as written (`mcp__mcp-n4-<station>__n4_navigate`).
- **pi and pi-based harnesses (gentle-shell):** `<agent-dir>/mcp.json` (`~/.pi/agent`, or
  `$PI_CODING_AGENT_DIR`; `~/.gentle-shell/agent` for gentle-shell), the same `mcpServers` format. A
  hyphen in the server name becomes an underscore in the tool name: a server `mcp-n4-<station>`
  exposes `mcp__mcp_n4_<station>__n4_navigate`. In a harness whose code mode is disabled, give each
  entry `"exposure": "direct"` so its tools are offered to the model as tools:

```json
{
  "mcpServers": {
    "mcp-n4-mystation": {
      "command": "/path/to/niagara-tools/go/bin/mcp-n4",
      "args": ["--station", "MyStation=https://10.0.0.5", "--insecure-tls", "MyStation"],
      "exposure": "direct"
    }
  }
}
```

### What the station runs: uptime, loaded modules, jars on disk

`n4_station_runtime(max_rows=500, timeout_s=60)` answers "what is this station running, and does it
need a restart?" in one read. `uptime` comes from the spy page `sysInfo`: `boot_time`,
`current_time` and `up_time` as the station writes them, and `uptime_s` from its `bootTicks` and
`currentTicks`. `loaded_modules` comes from the spy page `sysManagers/moduleManager`, which lists
only the module parts the station has loaded (`name`, `vendor`, `version`; `count`, `truncated`).
The session lists no module jars, so `jars_on_disk` is `unavailable` unless the server was started
with `--niagara-home NAME=PATH` for the station: then the tool lists the `.jar` files of
`PATH/modules` (a link is never opened: `skipped`) and reads `META-INF/module.xml` only from the
jars that may hold a loaded part and from those changed after the boot. `mismatches` names each
difference with its `reason`: `version_differs`, `modified_after_boot`, `loaded_not_on_disk` or
`not_loaded_modified_after_boot`. The boot is this machine's clock minus `uptime_s`, compared with
the jar times on this machine's filesystem. `restart_needed` is `true` with any mismatch, `false`
only when the uptime, the whole loaded list and every jar are known and none differs, and `null`
otherwise; `restart_note` says which. Use it after a deploy: a station holds its module jars open
while it runs, so a jar it loaded changes only after a restart. Measured on Niagara 4.15: 311
loaded parts of 740 jars, under 1 s with the installation on a local disk and 13 to 20 s through
a mounted Windows drive.

### Runbooks: MCP prompts and resources

Besides its tools, the server offers three recurring procedures as MCP prompts (`prompts/list`,
`prompts/get`) and as Markdown resources (`resources/list`, `resources/read`, URI
`runbook://<name>`), in read-only and write mode alike; `initialize` advertises the `prompts` and
`resources` capabilities. A client shows them as slash commands or attachable documents; the
prompt fills the `{{argument}}` placeholders, the resource is the text as written.

| Name | Arguments | What it walks through |
|---|---|---|
| `post-deploy-check` | `station` (required), `scope`, `module` | after a module deploy and restart: `n4_station_runtime` (`restart_needed`, module version), health deltas, new alarms, the commissioning summary; read-only |
| `alarm-triage` | `station` (required), `source`, `since` | open alarms grouped by source and class, classes without a recipient, each event sized with `n4_event_report`, acknowledgements proposed, never performed without approval |
| `commissioning` | `station`, `scope` (both required) | the commissioning report recipe below, every fix through dry run and token, save with persistence evidence |

The runbooks are files under `go/internal/mcpn4/runbooks/files/` (front matter `title`,
`description`, `arg name[!]: description [= default]`, then the body). A test refuses a runbook that
names a tool the registry does not hold, a placeholder that is not a declared argument, or an
argument never used. An unknown prompt or a missing required argument is a JSON-RPC invalid-params
error; an unknown resource URI answers `-32002`.

### Behaviour diff around a deploy

`n4_behavior_snapshot(name, scope="station:|slot:/", max_points=5000, timeout_s=60)` reads every
control point below `scope` once (`select slotPath, out.value, status.bits from
control:ControlPoint`) and writes it to `<state-dir>/snapshots/<name>.json` (0600, never
overwritten: a taken name is refused). Take it before a module deploy.

After the deploy and restart, `n4_behavior_diff(before, after=None, tolerance=0, limit=50)`
compares that snapshot with a second one (`after`, no station read) or, without `after`, with a
live read of the same scope (needs `n4_connect`). It lists `status_changed` (the status flags by
name, before and after), `value_changed` (the out value text differs), `missing` and `new` points,
each as `{count, items, more}` sorted by path, and `summary` with `unchanged`. Values are compared
as the station writes them; with `tolerance`, two values that both read as numbers (a locale
decimal comma such as `21,5` and a trailing unit such as `21.5 °C` allowed; `1,234.5` is text)
count as equal within it. When either side was truncated by `max_points`, `missing` and `new` are
`{not_checked: why}`, never a zero. `notes` say when the two sides come from different stations or
scopes. Recipe:

1. `n4_behavior_snapshot(name="before_<module>_<date>", scope=<the part the module drives>)`.
2. Deploy, restart, `n4_station_runtime` (`restart_needed: false`).
3. Wait for the drivers to poll (a point read right after the restart may still be `stale`), then
   `n4_behavior_diff(before="before_<module>_<date>", tolerance=<the noise of the process>)`.
4. Explain or fix every item; an intended change is still listed, so say which ones were expected.

### Refrigeration fault rules over history

`n4_refrigeration_faults(rules, period | start/end, max_rows=5000, limit=50)` runs the rules named
in `rules` over history data and reports findings only (read-only). Each history is read once per
call with the locale-free query (`timestamp.encodeToString, value.encodeToString`); every threshold
is a required argument, since the sources disagree (a short cycle is a run under about 2 minutes
for some and under 10 for others).

| Rule | Histories | Thresholds | Finding |
|---|---|---|---|
| `short_cycling` | `run_history` (boolean) | `min_run_s`, `max_starts_per_hour` | a finished run shorter than `min_run_s`; a clock hour with more starts |
| `defrost_termination` | `defrost_history` (boolean), `coil_history` | `terminate_temp`, `max_minutes`, `slack_minutes` | a cycle whose coil never reached `terminate_temp` and lasted at least `max_minutes - slack_minutes` (ended by time); `cycles` counts temperature / time / early / open |
| `setpoint_not_reached` | `temp_history`, optional `exclude_history` (boolean, e.g. defrost) | `setpoint`, `band`, `min_minutes` | a period above `setpoint + band` for at least `min_minutes` (a sample holds until the next) |
| `over_amps` | `amps_history` | `limit` | a clock hour with samples above `limit`, their count and the peak |
| `door_drift` | `door_history` (boolean), `temp_history` | `max_rise`, `after_minutes` | an opening after which the room rose more than `max_rise` until close + `after_minutes` |

Each rule is `{count, items, more}`, `summary` has one line per rule (`ok`, `N finding(s)`,
`not checked: why`), and `histories` lists each read with `records`, `null_records` (records with a null status carry no
value and are left out) and `truncated` (more records existed; the oldest were cut). A history that could not be read makes its rule `not_checked`,
never clean. An event still open at `end` is cut there.

### Finishing a job: the commissioning report

`n4_commissioning_report(scope="station:|slot:/", sections=None, limit=20, expect_histories=None,
expect_alarms=None, spyder_spec=None, link_depth=2, max_components=5000)` answers "what is still wrong or unfinished
below this part of the station?" in one call. It is read-only and composed of reads the server
already makes: one BQL query per table, one link scan, slot loads and the licence page. The reply
starts with `summary`, one line per section (`ok`, `N finding(s)`, `not checked: <why>`,
`skipped: <why>`), then `requests` (the station reads made), then each section with its `state`
(`ok`, `findings`, `partial`, `not_checked`, `skipped`), its `findings` count and its `checks`.

A check that ran is `{count, items, more}`: `items` are the first `limit` offenders as
`{path, name, fact}` and `more` is how many the limit left out. A check that did not run is
`{not_checked: "<why>"}` (the read failed or the station cannot answer it) or `{skipped: "<why>"}`
(the caller did not ask for it). Every fact is a value the station answered; a failed read is never
a zero: a section none of whose checks ran has no `findings` count, and `partial` means the checks
that ran are clean while others did not run.

| Section | Checks | Read |
|---|---|---|
| `points` | `devices_down_or_disabled` (listed first; their points are counted in their flags, `under_listed_devices`, not listed), `networks_not_ok`, `in_fault`, `down`, `stale`, `disabled`, `overridden`, `in_alarm`, `null_output`, `status_unreadable`; `by_network` counts devices and points per driver network | the network and device selects of `n4_station_health`, and `select slotPath, name, displayName, type, status.bits from control:ControlPoint` |
| `histories` | `points_without_history` (with `expect_histories`: the points it keeps, listed, findings; without it: a count with the note `pass expect_histories to list them`, no list, not a finding), `disabled`, `in_fault`, `never_recorded` | `select slotPath, name, type, enabled, status.bits, historyConfig.id, lastRecord.timestamp from history:HistoryExt` |
| `alarms` | `points_without_alarm` (only with `expect_alarms`), `unknown_alarm_class`, `classes_without_recipient`, `open_alarms`, `unacked_alarms` (`alarms` is the sum over the classes listed) | `select slotPath, name, type, alarmClass from alarm:AlarmSourceExt`; `select slotPath, name, openAlarmCount, unackedAlarmCount from alarm:AlarmClass` and one link scan, both over `Services/AlarmService` |
| `links` | `unresolved_source` (a source ORD that is neither a handle nor an absolute slot path, or a slot path that does not exist; `source_outside_scan` counts the sources not judged), `inputs_with_several_links` (a priority input `in1`..`in16` that more than one link targets), `dangling_outputs` (the check of `n4_find_dangling_outputs`) | one link scan of the scope, `link_depth` levels deep |
| `naming` | `duplicate_display_names` among sibling points, `escaped_names` (`$20` and the like in the slot name), `palette_default_names` (a point of the control palette with or without the number the station appends, `NumericWritable`, `NumericWritable1`, `BooleanPoint1`; or a point called by the name of its own type plus that number, a `kitControl:Tstat` called `Tstat1`: a palette drop names the first instance after its type, so a lone `Tstat` is not flagged, and neither is a bare `Ramp` beside a `Ramp1` sibling, though it is likely the first drop of the two; the fact says which) | the points table |
| `station` | `services_in_fault`, `licence_expiry` (the warning of the health tool's `license` section), `unsaved_changes` and `last_backup` | one slot load of `Services`, the SystemService slots and the licence page; only when the scope is the whole station |
| `spyder` (only when named in `sections` or with `spyder_spec`) | the download state of each Honeywell BACnet Spyder: `disabled` (`Enabled=false`: a Spyder left disabled after a download is neither polled nor pinged), `device_status` (`deviceStatus` other than 1), `compile_status` (`compileStatusFlag` other than 0), `error_status` (`sErrorStatus` not empty); `spyders` counts the Spyders read; `io_mismatch` (only with `spyder_spec`) lists each IOComp field that differs from the spec, the spec points with no IOComp and the Spyders not found | `select slotPath, name, enabled, deviceStatus, compileStatusFlag, sErrorStatus from honeywellBacnetSpyder:BacnetSpyder`; with `spyder_spec`, `select slotPath, SensorType, InputLow, InputHigh, OutputLow, OutputHigh, Terminal1 from honeywellSpyderTool:IOComponent` |

`unsaved_changes` and `last_backup` are always `not_checked`, with the reason: the station exposes
neither to a slot read or a query, and the report does not guess. `expect_histories` and
`expect_alarms` say which points should carry the extension: `{"kinds": ["numeric", "writable"],
"name_pattern": "*Temp*"}` (kinds are matched against the point's type name, the glob against its
name or display name; `{}` is every point). Without `expect_histories` the points with no history
extension are only counted (on most stations that is nearly every point); without `expect_alarms` the absence of an alarm
extension is not a finding and the check is skipped.

`spyder_spec` is the IO each Spyder should have, by the names Workbench shows:
`{"2005": {"RA_T": {"sensor_type": 0, "terminal": 1}, "COMP1_A": {"sensor_type": 8, "input_low": 0,
"input_high": 10, "output_low": 0, "output_high": 30, "terminal": 3}}}` (fields `sensor_type`,
`input_low`, `input_high`, `output_low`, `output_high`, `terminal`, compared with the IOComp's
`SensorType`, `InputLow`, `InputHigh`, `OutputLow`, `OutputHigh` and `Terminal1`; numbers compare as
numbers, `0` is `0.0`, texts ignoring case). Given alone, it adds `spyder` to the default sections;
without it only the download state is checked and the IOComp table is not read. A good Spyder after a
Full Download reads `deviceStatus=1`, `compileStatusFlag=0`, `sErrorStatus` empty and `enabled` true
(`ModifiedFlag` and `nvModifiedFlag` stay set after a good download and are not checked). On a
station without the Honeywell Spyder modules the query answers an error and the checks are
`not_checked`. The BacnetSpyder query was read live on a client B field job; the IOComp type is
`honeywellSpyderTool:IOComponent` (BIOComponent, from the module's declared types) and has not yet
been read by BQL on a live station: if a station refuses it, `io_mismatch` is `not_checked` with the
station's error, never clean. LON Spyders are not read.

Cost is bounded and independent of the station's size: 11 station reads for the whole station (7
GETs and 4 slot loads), 8 for a subtree (the `station` section is skipped), fewer with `sections`.
Each query reads at most `max_components` rows. A truncated table sets `truncated` and
`truncated_note` ("truncated at N components: counts are lower bounds and absences are not
asserted; narrow the scope"), and a check that would assert an absence from it is `not_checked`. A
link scan holding more than `max_components` components is not judged. The link scan loads
`link_depth + 3` levels of slots: on a large station point `scope` at the folder that holds the
logic rather than raising `link_depth`.

Recipe for closing a job:

1. `n4_commissioning_report(scope=<the job's folder or network>, expect_histories={...},
   expect_alarms={...})`; read `summary` first.
2. For each section with findings, fix or justify every item (`n4_read_slots` on its `path` shows
   the detail); raise `limit` or narrow `scope` while `more` is above zero.
3. A `not checked` line is work left for a person, not a pass: read its reason, then check it
   another way (Workbench, `n4_bql_query`, `n4_spy_page`).
4. Run it once more over the whole station for the `station` section, then `n4_save_station`.

`commissioning-verify` (`go/cmd/commissioning-verify`) is a different tool with a similar name: the
offline punch-list of a module's sources and a `config.bog` file, run before deployment with an
exit code for a gate. This report reads the running station after deployment. Neither replaces the
other.

### Tags

A direct tag is a dynamic slot with the metadata flag (`m`) whose name is the tag id with its colon
escaped: `hs:point` is the slot `hs$3apoint`. A tag group is not a tag slot: it is a relation slot
(`baja:Relation`, named `r`, `r1`, ...) with `relationId` `n:tagGroup` and `sourceOrd` the group's
definition in a dictionary; the station then implies the group's tags. Implied tags (the
dictionaries' rules, tag groups) are never slots: only the station's session handler computes them.

`n4_read_tags(ord | ords, implied=true, relations="direct")` reads up to 50 components, one load
each: `direct` tags (`id`, `namespace`, `name`, `type`, `value`, `slot`, `flags`), `tag_groups`
(`group_ord`, `slot`), the direct `relations` (`id`, `endpoint_ord`, `inbound`, `slot`), and
`implied` (`id`, `type`, `value`; one `impliedTags` session call per component, the call
BajaScript makes). `relations="all"` adds the implied relations (`n:parent`, `n:child`, ...; one
more call each), `"none"` leaves relations out, and `implied=false` makes no session call.
`implied: null` with `implied_unavailable` means the implied tags were not read, never that there
are none.

`n4_list_tag_dictionaries(dictionary?, max_items=200)` lists the dictionaries under
`/Services/TagDictionaryService` (one BQL query: namespace, enabled, frozen, version, status) and
what each defines: tag definitions (`id`, `info`, value `type`, `default`, `validity`; `type` null
for a smart tag), tag groups (`id`, `group_ord`, the `tags` they imply) and relation ids.

Recipe, find components by tag: `n4_neql_query(neql, base_ord="station:|slot:/", bql?, max_rows=200)`
is one GET of `<base_ord>|neql:<neql>|view:file:ITableToCsv`. It matches the components below
`base_ord` (never `base_ord` itself), implied tags included, and returns their station ORDs:

```text
n4_neql_query(neql='tst:label = "hello"', base_ord="station:|slot:/ImplTAG")
  -> ords ["station:|slot:/ImplTAG/Pt1"]
n4_neql_query(neql="n:point", base_ord="station:|slot:/ImplTAG", bql="select slotPath, out.value")
  -> columns ["Slot Path", "Value"], rows [{"Slot Path": "slot:/ImplTAG/Pt1", "Value": "0,00"}]
```

`bql` is a select of columns only (no `from`): the station runs it on the matches. `|` is refused;
a query the station cannot parse comes back as HTTP 400, reported with the query.

`n4_set_tags(ords, add=[{id, value?, type?}], remove=[id], allow_ad_hoc=false)` writes direct tags
on 1 to 50 components (at most 200 ops) under the usual dry run, token, write scope and rollback:

- A tag an installed dictionary defines takes the definition's value type (another `type` is
  refused) and, without `value`, its default; a Marker takes no value.
- An undefined tag needs `allow_ad_hoc: true` and a namespace no dictionary installs: `n:foo` is
  refused even then (Tagging guide, Adding ad hoc tags).
- A new tag is an `a` op with flag `m` (inverse `v`); a tag holding another value of the same type
  is set (inverse: the old value); a type change is refused (remove, then add); an equal value is
  skipped with a note. A removed tag's inverse re-adds it with its type, value and flags.
- The read-back reloads each target; a value the station normalized is listed under `coerced`.
- Refused at plan, before any write: a tag, group or relation named twice in one list, or both
  added and removed; a group or relation already on the target; a value type other than
  `baja:Marker`, `Boolean`, `Double`, `Float`, `Integer`, `Long`, `String`, `DynamicEnum`
  (`ordinal` or `ordinal@range`), `Format` and `RelTime`, or a value that type does not take.
  The dictionaries on the lab (Niagara, Haystack 4) define tags only of Marker, String,
  DynamicEnum, Double and Integer, so no defined tag is refused by this list.
- A removed relation's inverse re-adds the fields it stored (its endpoint ORD as written).
- In the destructive class: a client such as `n4call` executes it only with `--execute`,
  `--allow-destructive` and the line's own `"execute": true`.
- `add_groups` / `remove_groups: [id]` apply or remove a tag group a dictionary defines: an `r?`
  relation (`n:tagGroup` to `slot:/Services/TagDictionaryService/<dict>/tagGroupDefinitions/<g>`,
  flag `r`); the read-back lists the implied tags and any group tag missing from them.
- `add_relations` / `remove_relations: [{id, endpoint_ord}]` add or remove an outbound relation
  (`sourceOrd` the endpoint's `h:` ORD). The inverse of an added relation removes the slot the
  station named (`r`, `r1`, ...).

Live smoke (lab station, a disposable folder and test dictionaries): `n4_list_tag_dictionaries`;
`n4_set_tags` add a defined tag, a group and a relation; `n4_read_tags` shows them and the implied
group tags; `n4_neql_query` finds the component by a group tag; `n4_rollback` the batch and read
again (nothing left); an undefined tag in an installed namespace is refused.

## Writing safely

Write tools exist only with `--allow-writes` (otherwise `tools/list` omits them
and calling one returns JSON-RPC -32601): `n4_create_component`, `n4_add_history`, `n4_add_alarm`,
`n4_copy_component`, `n4_replicate_points`, `n4_deploy_template`, `n4_set_slot`, `n4_set_flags`, `n4_create_link`, plus the destructive class
(`destructiveHint: true`) `n4_remove_component`, `n4_set_tags` (it removes tags and relations), `n4_rollback`,
`n4_save_station`, `n4_ack_alarm`, `n4_set_schedule` (it replaces every event of the schedule; `n4_rollback`
restores it) and `n4_invoke_action` (`active`, `inactive` and `auto` have no inverse). Example:

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
   not reset it). The save is outside the budget; the refusal says so.
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
   The token is bound to the whole plan, its notes and warnings included: a create
   whose name is taken carries a first note that the station will assign another name,
   so a name taken after the dry run, or a re-used create token, is refused at execute
   as a plan mismatch rather than renamed silently.
   Before a token is issued the dry run refuses what it can check by reading:
   `n4_create_component` an unknown `type` or `proxy_ext` (one the station's type
   registry does not describe, named in the error instead of the station's generic
   "Unable to process request"); `n4_create_link` a source or target slot that
   neither the component nor its type contract has (checked only when the station
   describes the type), and a link identical to one the target already has (same
   source handle, source slot and target slot). Whether the parent accepts a child
   type cannot be checked offline; the station's refusal of it is journaled as
   `refused` (layer 7).
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
   One exception: when the station answers the first op with its own error reply
   (`Illegal parent`, `Slot ... already exists`, `Could not find Source Slot`,
   `Duplicate link already exists`) and a read-back of the targets shows nothing
   applied (no new child under the parent for `n4_create_component`, no new link on
   the target for `n4_create_link`), the batch is `refused`: a `result` record
   `{verdict: "refused", refused: true, station_error, inverse: []}` is journaled,
   the reply says the station refused it and nothing changed, and the audit outcome
   is `refused`. A transport failure, a timeout, a read-back that fails or shows a
   change, and every other tool stay `in-doubt`. The station's text is decoded from
   HTML entities (`&quot;`, `&lt;`, `&amp;`) in every error a tool reports.
   Journal states, as `n4_list_batches` shows them: `completed`, `in-doubt`,
   `refused`.
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
  so its previous value is the real one. A frozen property whose value equals its
  declared default (whatever its type declares, e.g. a 10 s `RelTime`, not only 0, `false`
  or an empty string) is omitted by
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
- A slot that links write (a linked input, a fallback fed by a link) gets the value again at
  the link's next update. `n4_set_slot`'s dry run lists those links as `incoming_links`
  (`source_ord`, `source_slot`, `target_slot`, `enabled`) with a note; the write is still planned,
  and a link added after the dry run makes the token refuse. To keep the value, remove the link
  (`n4_list_links`) or write its source (backlog OD001).
- `n4_set_slot(writes: [{ord, slot, value, value_type, status?, simple_type?}, ...])` is the
  batch form (overamps retro row M6, backlog OD040): 1 to 50 entries, each planned exactly as the
  single form, under one dry run and one `confirmation_token`. `writes` cannot be combined with
  `ord`, `slot`, `value`, `value_type`, `status` or `simple_type`; the same (component, slot)
  twice is refused; every entry's `ord` must be under the write scope (one outside refuses the
  call); a refused entry refuses the batch and its message names it (`writes[2]: ...`). The batch
  spends one write of `--max-writes`, as `n4_set_tags` and `n4_add_history` do. The plan's notes name their entry, and `incoming_links` lists every
  entry's incoming links with the entry's `target_ord`. The read-back lists `{ord, slot,
  requested, observed, verdict}` per entry; the batch is `verified` only when every entry is,
  else `mismatch`. `n4_rollback` of the batch restores every entry. The single form's reply is
  unchanged, except that a missing `ord`, `slot`, `value` or `value_type` is now refused as a
  tool error with the same text (`missing required argument: ord`) instead of a -32602 error,
  because the schema no longer lists them as required.
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
  that never writes; the read-back checks the extension type. A Modbus proxy created this
  way starts at its type's default address (`addressFormat` `hex`, `address` `0`);
  set it on the proxy's ORD with a slot path, because the station leaves out a
  `dataAddress` still at its default: `n4_set_slot(ord=<point>/proxyExt,
  slot="dataAddress/addressFormat", value="modbus", value_type="baja:Simple",
  simple_type="modbusCore:AddressFormatEnum")`, then `slot="dataAddress/address"`
  (`baja:String`).
- `n4_add_history(points, history_name, interval_ms)` adds one interval history
  extension per point (1 to 50), created already configured (`historyName`,
  `interval`, `enabled`, default true) in one batch under one confirmation token.
  The extension type follows the point type, read from the station's type registry
  (a numeric, boolean, enum or string control point gets `NumericInterval`,
  `BooleanInterval`, `EnumInterval` or `StringInterval`), and so does the default
  `ext_name`; an `ext_type` that does not fit the point is refused at the dry run,
  naming the one that does. It refuses a point that already has a child named
  `ext_name` and, when the pattern only uses `%name%` / `%parent...name%` tokens, a
  history id (`/<station>/<history name>`) that would repeat across the batch or
  that an extension on the station already holds (one BQL read of every
  `history:HistoryExt`): such an extension would sit in fault "Duplicate history
  id". The inverse removes every extension; the read-back compares each one,
  including its history id and an empty fault cause.
  `mode=cov` adds a change-of-value extension instead (`NumericCov`,
  `BooleanCov`, ...): it records each change of the point value, has no interval
  and refuses `interval_ms`. `capacity` sets the history capacity in records (the
  oldest roll off when full; 20160 is 7 days at 30 s) or `"unlimited"`, by a set
  op on `<ext_name>/historyConfig/capacity` (`history:Capacity`, encoded `1:<records>`
  or `0:0`) right after each add in the same batch: `historyConfig` is a component,
  which a station refuses inside an add body. Without it the extension default
  stays (500 records, Roll). The plan notes how long the capacity lasts at the
  requested interval (a change-of-value history lasts as long as its changes take),
  and the read-back checks the capacity when one was asked. `full_policy` (`roll`,
  the default, drops the oldest record when the capacity is reached; `stop` stops
  recording) is a second set op on `<ext_name>/historyConfig/fullPolicy`
  (`history:FullPolicy`). `change_tolerance` (a number of at least 0, in the point's
  units) sets `changeTolerance` of a `NumericCov` extension in its add body: a change
  smaller than it records nothing. It is refused for mode interval, for an `ext_type`
  other than `history:NumericCovHistoryExt` and for a point that is not numeric,
  naming the point. One call covers every point that shares these settings (three
  COV histories with a 0.5 A tolerance and 50000 records: one dry run, one token).
  A station failure at op k stops the batch: the ops before it stand and the batch is
  journaled in-doubt; the error names the inverse of the sent ops, which you apply by
  hand (`n4_rollback` refuses in-doubt batches).
- `n4_add_alarm(points)` adds one alarm extension per point (1 to 50), configured,
  in one batch under one confirmation token. See "Alarm extension recipe" below.
- `n4_copy_component(parent_ord, name, new_names)` adds the configuration of one
  child again under each new name (1 to 30) in one batch under one confirmation
  token: type, plain slots and child components up to 6 levels deep (a Modbus
  device down to its point addresses). Runtime slots are left out by their flags, read
  from the station's type definitions through the registry channel: a frozen slot
  declared readonly, transient or default-on-clone (`r`, `t`, `d`) or made transient,
  and a dynamic slot flagged transient or remove-on-clone (`t`, `R`). Flags an
  integrator set otherwise travel with the copy. A Status `out` travels only when the
  parent's type declares it without `r` or `t`: a constant block's configured value
  (`kitControl:NumericConst` and its Boolean, Enum and String siblings) is copied, a
  computed output (a writable's `out`, declared `rtso`) is not. The plan notes list each slot left
  out with its flags and call out a readonly slot holding a non-default value.
  Links are not copied either; the plan counts the links left behind, and a deeper
  source is refused, as is a station whose type definitions cannot be read. A Modbus
  device allows one writable point per register address: a copy of a writable Modbus
  point keeps its source's address, and the station resets it to `Hex:-1` (log
  "Duplicate Writable point for register address", `BModbusClientProxyExt`), even when
  the address is set again afterwards. `copies` gives each such copy its own address
  inside the same add: `copies=[{"name": "M10", "data_address": "40010"}]`, with an
  optional `address_format` (`hex`, `decimal`, `modbus`; default the source's). The plan
  notes every copy that would share a register with its source or an earlier copy, and
  the read-back of such a copy names that cause next to its differences. A read-only
  point (`control:NumericPoint`) is not subject to the rule. Change what else differs
  per copy afterwards (e.g. `deviceAddress` with `n4_set_slot`). The
  inverse removes every copy under its assigned name; the read-back loads each copy
  and compares its whole configured subtree with the plan, listing the paths that differ.
  The plan also notes every writable point (the source or one inside it) whose `fallback`
  is not null: the fallback is the output when no priority input is set, so each copy
  commands its device with it on first communication. A note, never a refusal.
- `n4_replicate_points(source_parent_ord, points, targets)` copies M points of one parent
  (a device's points folder) onto N target parents (other devices' points folders): N x M
  adds in one batch under one confirmation token (at most 200), each the body
  `n4_copy_component` would add (address and `dataType` kept). Per target,
  `copies=[{"point": "Setpoint", "new_name": "SP", "data_address": "40010",
  "address_format": "modbus"}]` overrides one point's name and/or address in the same add.
  Each add spends one `--max-writes` write: the plan notes the count and a replication that
  would leave no write for its own `n4_rollback` (adds + 1 > `write_budget.remaining`) is refused
  before any token. Two targets that differ only by a trailing `/` are one target (refused). A
  failure at add k leaves the batch in doubt: `n4_rollback` refuses it and its error lists the
  inverse ops to remove the k copies by hand. A child that already exists
  under a target with a copy's name is refused. The plan notes writable Modbus copies that
  would share a register on one device (the station resets the later one to `Hex:-1`) and
  writable points with a non-null fallback, as `n4_copy_component` does; points already on a
  target are not read for that check. The read-back compares every copy with the plan; the
  inverse removes every copy. A client B field job replicated 112 points with 630 calls
  in 39 minutes and ran out of budget; this is one dry run and one execution.
- `n4_deploy_template(template_file, parent_ord, name?, settings?)` deploys a component
  template (`.ntpl`, a file on the machine the server runs on) under `parent_ord` in one
  add op, client-side as Workbench does (the station's TemplateService has no deploy
  action). The template component (the first child of `template.bog`'s root folder) is
  added with the template's settings filled: `settings` maps each exposed setting of the
  manifest to a value (`num`, `bool` or `str`; a `num` is checked against the manifest's
  `min`/`max`). A setting writes the slot its manifest `slotPath` names, or the target of
  the template configuration's ConfigBinding for it; a setting not given keeps the
  template's value; a required setting the template leaves empty must be given; an
  unknown setting, the configuration's own metadata (`uID`, `version`...), a station or
  application template and a type the station's registry does not describe (its module
  is missing) are refused before a token. The dry run notes the template (title,
  version, vendor, id), the manifest dependencies, each setting with its value, source
  and slot (warning when a Status slot it writes stays null), and everything not
  carried: the template configuration (its bindings name template handles, which would
  point at unrelated station components, so the deployed component is not linked to its
  template), links inside the template (create them with `n4_create_link`), ORD values
  naming a template handle, passwords, slot facets (`x`) and categories (`c`). A
  template of more than 5,000 properties is refused. The read-back loads the deployed
  component and checks every planned node (type, value, flags; a slot the station omits
  must equal its declared default); the inverse removes it under its assigned name.

Destructive tools use the same pipeline:

- `n4_remove_component(parent_ord, name)` removes a child and its subtree. The plan
  snapshots the subtree to depth 3 (type, plain slots, `wsAnnotation`, and the links
  whose target is inside it) as data; it is the recorded inverse. A Status `out` is
  kept by the same rule as a copy (a constant block's configured value, read from the
  station's type definitions; left out when they cannot be read), and the rollback's
  read-back compares each `out` the snapshot carries. Re-creation is
  limited to that snapshot and is not a full restore: deeper levels, runtime state
  and links coming from outside the subtree are not recreated (the plan says so).
  The plan loads one level deeper to see what the cut drops: anything there is listed in
  `snapshot_truncated: {depth, paths}` (first 20 paths, with the exact count in the note),
  also in the execution reply.
  The dry run also scans for links from the subtree into components outside it (a
  link is stored on its target, so the subtree load cannot see them): by default it
  scans the `--write-scope` prefix that holds `parent_ord` to depth 6, and
  `link_scan_ord` (equal to or under that prefix; any other root refuses) narrows it; a
  root holding more than 100000 nodes to depth 6 refuses as too large to scan and asks for a
  narrower `link_scan_ord` (the scan is never skipped). A link whose target is inside the
  removed component is internal at any depth. Links stored outside the
  scanned root are not detected, and the note names the root (a link outside a narrowed
  root is broken by the remove and not restored by its rollback; a link whose source sits
  more than 3 levels below the removed component, past its snapshot, is not matched either).
  When the scan finds any,
  the dry run refuses and lists each one (`target (target_slot <- source_slot)`);
  repeat it with `acknowledge_links: true` to remove anyway, and the plan lists them in
  `outgoing_links_broken`. The inverse records each one as an `outgoing_link` (target
  ORD and handle, target slot, source path inside the component, source slot), and
  `n4_rollback` re-creates it after the component (both ends in today's write scope,
  else the rollback refuses; a target that is gone or was replaced is not linked). Its
  read-back looks for each such link on its target, from the restored source with the
  recorded slots, and reports `outgoing_links: {restored, missing}`; any missing one
  makes the verdict `mismatch`.
- `n4_rollback(batch_id)` plans the inverse recorded in the journal as a new batch
  (`rollback_of` points to the original). It refuses an unknown batch, one already
  rolled back, a rollback itself, an `in-doubt` batch (it shows the journaled
  intent so a human decides), a `refused` batch (the station applied nothing, so
  there is nothing to roll back) and a batch from another station. A refused
  rollback does not count as an earlier rollback. The current
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
  `relinks: {restored, skipped}`; the outgoing links a remove recorded are re-created
  too, see `n4_remove_component`). A frozen child (e.g. a writable's `proxyExt`, an
  alarm extension's `offnormalAlgorithm`) is never re-added. When the station's type
  contracts (registry channel) declare it, the dry run puts `frozen_config` on its
  component spec (so the token covers it): after its parent's add, a set op swaps the
  captured type in (as `n4_add_alarm` does) and set ops write the captured configuration
  slots the fresh child does not already hold, e.g. `highLimit`/`lowLimit`; readonly,
  transient and `status` slots are never written; transient and `status` slots are not
  compared (`ignored`), while a readonly slot that differs stays listed. The
  read-back compares it with the snapshot and lists any difference in
  `frozen_config_not_restored`, with the station's reason in `refused` for a value it
  refused; without contracts nothing is written. What the remove's snapshot did not
  capture (below depth 3, `snapshot_truncated`: e.g. an out-of-range algorithm's
  `limitEnable/highLimitEnable`) cannot come back: such a child stays listed and the
  operator re-enters it. A link-driven input whose link is not
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
  `evidence`. A `config.bog` that is unreadable right after the save (the station
  moves it to its backup before renaming `config.bog.working` over it) is read once
  more after 1 s; `evidence.retry` records it. Without a station home the station's
  own save job is the proof (the file servlet never serves `.bog` files): the tool
  lists the `baja:StationSaveJob` children of `Services/JobService` before the save
  and polls them after it; a new job that ends `success` gives `persisted: true`,
  `failed` or `canceled` gives `false`, and no new job, a job still running at the
  timeout or an unreadable JobService gives `unknown` with a `hint` that says why
  (`evidence.route` is `station save job`, `evidence.job` the job). A station that is
  already saving skips the new save, and a restart clears the jobs. It needs at least
  one `--write-scope` like every write.
  `backup: true` is the backup before a destructive batch, in the same call: after the
  save the station directory (`--station-home`) is zipped to
  `<state-dir>/backups/<station>-<UTC time>.zip` (0600, never over an existing file)
  without what the BackupService leaves out of its backups by default (files
  `*.hdb`, `*.adb`, `*.lock`, `*backup*`, `console.*`, `config.bog.b*`,
  `config_backup*`; the `history`, `alarm` and `webFileCache` directories of the
  station home; a site may have changed its station's lists; a station home given as a
  symbolic link is resolved, links inside it are listed, not followed; a directory
  without `config.bog` is refused). The reply's `backup` gives
  `path`, `files`, `bytes`, `sha256`, `excluded_files`, `excluded_dirs` and the rules,
  or `{error}` when the zip failed: the save stands and the verdict is `mismatch`
  (the backup asked for does not exist). Without a station home the dry
  run refuses it. The zip is a copy of the files on disk, not the `.dist` Workbench's
  Backup Station streams through Fox or the platform; it serves stations whose
  directory this machine can read (a local or mounted station), not a remote JACE.
- `n4_ack_alarm(uuids, notes)` acknowledges 1 to 100 alarm records by uuid (as
  `n4_read_alarms` returns them) with one call on the station's alarm channel:
  `ackAlarms` with body `{ids: [<uuid>...], notes?}` (box-rt `BAlarmChannel.ackAlarms`;
  the station sets the acking user, ackTime and ackState `ackPending`, updates the alarm
  database and routes the ack). The plan's op is `{channel: "alarm", key: "ackAlarms",
  body}`, journaled like any op; the engine sends only the channel ops of its allow-list
  (`alarm` `ackAlarms`) and refuses any other before sending. The dry run reads the records with
  `alarm:|bql:select uuid, ackState, user, source where (uuid = '<u>' or ...)` and the same with
  `ackState = 'unacked' and` (the state compared by its tag, never by localized text), and
  refuses, naming them, a uuid with no record, a record whose source is outside the write
  scope (the source, stored as `local:|station:|slot:/...` or `slot:/...`, is checked as
  `station:|slot:/...` by the same `--write-scope` check every write uses; a source in any
  other form, holding several ORDs (`; ` or newline separated), whitespace, a `..` segment or a
  chained view, is refused as not a slot path, never assumed in scope) and a record already acknowledged or ack
  pending; a malformed or repeated uuid is refused before any read. The read-back repeats
  both reads: `verified` when no record is still unacked and each has a user; `mismatch`
  when one stayed unacked or the station's reply, the list of alarm classes the user may not
  acknowledge (operatorWrite on the class), is not empty
  (`observed.permission_failed_classes`). An acknowledgement cannot be undone: the plan has
  no inverse and `n4_rollback` cannot reverse it. It needs at least one `--write-scope`.

### Semantic export: Haystack JSON

`n4_semantic_export(neql="hs:site or hs:space or hs:equip or hs:point", base_ord, implied=true,
max_entities=100, save_as)` exports the Project Haystack (`hs:`) tags of the components a NEQL
query matches as a Haystack grid in the Haystack 4 JSON encoding (Hayson), read-only. One NEQL GET
finds the components; each is then read as `n4_read_tags` reads it (one load, plus one
`impliedTags` call unless `implied` is false).

- `id` is a ref made from the slot path with its `$` escapes decoded (`/Plant/Supply$20Temp` →
  `Plant.Supply_Temp`; characters a Haystack ref does not allow become `_`; a clash gets `~2`,
  `~3`). `dis` is `hs:dis` when set, else the decoded slot name.
- Every `hs:` tag becomes a column with the namespace stripped: markers, numbers, booleans and
  strings. Outbound `hs:` relations (`hs:equipRef`, `hs:siteRef`) become refs, pointing at the
  exported row's `id` when the other end is exported too (a station names it by slot path or
  as `h:<handle>`).
- Tags of other namespaces are left out. A value that cannot be converted (for example a
  `baja:TimeZone`) is listed in `skipped` with why, never dropped silently. `truncated` says the
  match list was cut at `max_entities`.
- With `save_as` the grid is written to `<state-dir>/exports/<save_as>.json` (mode 600, never
  over an existing file) and the reply carries `path` and `bytes`.
- Brick output is not produced.

### Alarm extension recipe

Every palette alarm extension is one type, `alarm:AlarmSourceExt`; what makes it an
"OutOfRangeAlarmExt" is its two preset algorithm children (alarm-rt `module.palette`,
folder `Extensions`). `n4_add_alarm` picks the kind from the point type unless `kind`
is given, and a kind that does not fit is refused at the dry run, naming the kinds
that fit:

| kind | `offnormalAlgorithm` | `faultAlgorithm` | point | default name |
|------|----------------------|------------------|-------|--------------|
| `out_of_range` (numeric default) | `alarm:OutOfRangeAlgorithm` | `alarm:OutOfRangeFaultAlgorithm` | numeric | `OutOfRangeAlarmExt` |
| `numeric_change_of_state` | `alarm:NumericChangeOfStateAlgorithm` | (default) | numeric | `NumericChangeOfStateAlarmExt` |
| `boolean_change_of_state` (boolean default) | `alarm:BooleanChangeOfStateAlgorithm` | (default) | boolean | `BooleanChangeOfStateAlarmExt` |
| `boolean_command_failure` | `alarm:BooleanCommandFailureAlgorithm` | (default) | boolean | `BooleanCommandFailureAlarmExt` |
| `enum_change_of_state` (enum default) | `alarm:EnumChangeOfStateAlgorithm` | `alarm:EnumFaultAlgorithm` | enum | `EnumChangeOfStateAlarmExt` |
| `enum_command_failure` | `alarm:EnumCommandFailureAlgorithm` | `alarm:EnumFaultAlgorithm` | enum | `EnumCommandFailureAlarmExt` |
| `string_change_of_value` (string default) | `alarm:StringChangeOfStateAlgorithm` | `alarm:StringChangeOfStateFaultAlgorithm` | string | `StringChangeOfValueAlarmExt` |
| `status` | `alarm:StatusAlgorithm` | `alarm:StatusFaultAlgorithm` | any control point | `StatusAlarmExt` |

The point check is each algorithm's `isGrandparentLegal` (`BOutOfRangeAlgorithm` needs
a `BNumericPoint`, `BBooleanChangeOfStateAlgorithm` a `BBooleanPoint`, ...;
`BAlarmSourceExt.isParentLegal` needs a control point). A station refuses an add body
that nests a component, so the batch is: the add with the plain slots, then set ops
that swap `offnormalAlgorithm` and `faultAlgorithm` by type (empty value, the route
bench retro 2026-11-15 row 10 proved live with `n4_set_slot` `baja:Simple`), then set
ops for the rest by path, e.g. `OutOfRangeAlarmExt/offnormalAlgorithm/highLimit`.

Slots and their encodings (`BAlarmSourceExt`, `BOutOfRangeAlgorithm`):

| argument | slot | type, encoding |
|----------|------|----------------|
| `alarm_class` (default `defaultAlarmClass`) | `alarmClass` | `baja:String`, the slot name of an `alarm:AlarmClass` under the AlarmService; checked by one BQL read (`select slotPath from alarm:AlarmClass`), a missing one is refused listing the classes |
| `source_name` | `sourceName` | `baja:Format`; station default `%parent.displayName%` |
| `to_offnormal_text`, `to_fault_text`, `to_normal_text` | `toOffnormalText`, `toFaultText`, `toNormalText` | `baja:Format`, default empty |
| `time_delay_ms`, `time_delay_to_normal_ms` | `timeDelay`, `timeDelayToNormal` | `baja:RelTime`, milliseconds |
| `hyperlink_ord` | `hyperlinkOrd` | `baja:Ord` |
| `alarm_enable` (list of `offnormal`, `fault`, `normal`, `alert`) | `alarmEnable` | `alarm:AlarmTransitionBits`, the hex bit mask (`encodeToString` is `Integer.toHexString`): offnormal 1, fault 2, normal 4, alert 8; `["offnormal","normal"]` is `5`, all is `f` (the station default); Workbench shows it as `{toOffnormal toNormal}` |
| `inhibit` | `alarmInhibit` | `baja:StatusBoolean`, status ok |
| `high_limit`, `low_limit`, `deadband` | `offnormalAlgorithm/highLimit`, `lowLimit`, `deadband` | `baja:Double`; deadband 0 or more |
| (any limit given) | `offnormalAlgorithm/limitEnable/highLimitEnable`, `lowLimitEnable` | `baja:Boolean` inside the `alarm:LimitEnable` struct; both are false by default, so a limit given is also enabled; without a limit the extension raises nothing |
| `fault_high_limit`, `fault_low_limit`, `fault_deadband` (out_of_range) | `faultAlgorithm/highLimit`, `lowLimit`, `deadband`, and `faultAlgorithm/limitEnable/highLimitEnable`, `lowLimitEnable` | `baja:Double` and `baja:Boolean` on `alarm:OutOfRangeFaultAlgorithm` (`BOutOfRangeFaultAlgorithm`: the same limit slots as the offnormal algorithm); a fault limit given is also enabled; beyond it the point goes to fault and the alarm carries `toFaultText` |
| `high_limit_text`, `low_limit_text` | `offnormalAlgorithm/highLimitText`, `lowLimitText` | `baja:Format`; they override `toOffnormalText` for that limit |
| `alarm_value` (boolean change of state) | `offnormalAlgorithm/alarmValue` | `baja:Boolean`, default true |
| `alarm_values` (numeric or enum change of state), e.g. `{"alarm": 1}` | `offnormalAlgorithm/alarmValues` | `baja:EnumRange`, `{tag=ordinal,...}` (`BEnumRange.encodeToString`) |

Text syntax: the state texts and the limit texts are `baja:Format` patterns stored raw
in the alarm record (`alarmData.msgText`) and resolved against the record, not the
extension, so `%parent.displayName%` does not resolve there (it does in
`source_name`). Use `%alarmData.<key>%`: `sourceName`, `presentValue`, `highLimit`,
`lowLimit`, `deadband`, `alarmValue`, `fromState`, `toState`; `%lexicon(module:key)%`
for a translated text; `%%` for a literal percent. Examples:

```
"high_limit_text": "%alarmData.sourceName% high: %alarmData.presentValue% > %alarmData.highLimit%"
"low_limit_text":  "%alarmData.sourceName% low: %alarmData.presentValue% < %alarmData.lowLimit%"
"to_normal_text":  "%alarmData.sourceName% back to normal"
"source_name":     "AHU1 %parent.displayName%"
```

To force a fault for a test (proven live on a lab station): an `out_of_range`
extension with `fault_high_limit` (for example 100) and `to_fault_text`, then write
the point above it (150): the alarm record shows source state Fault with
`toFaultText` as its message and the point reads `{fault,alarm,unackedAlarm}`; back
below the limit (20) it returns to Normal.

A point that already has an extension of the same kind (a child whose
`offnormalAlgorithm` is of that type) is refused unless `allow_second=true`. The
inverse removes each extension under the name the station assigned. The read-back
reads every slot the batch set and both algorithm types; a slot the station stored
differently is listed under `coerced` as `{slot: {sent, read}}` and the verdict is
`mismatch`. Not covered here: recipients and routing (the AlarmService's
`alarm:StationRecipient` and friends), creating alarm classes, reading alarm records
and acknowledging them (other tools); the status kind's `alarmValues` / `faultValues`
status masks and the string kind's `expression` stay at their defaults (set them with
`n4_set_slot`).

## Schedules

### Kinds of schedules

`n4_list_schedule_kinds` (no station, no session) answers this section as data: every kind with
its station type, value type, default output, parts, purpose, behaviour and usual link, and every
special-event kind with its fields. A 4.15 station has six schedule kinds an integrator creates:

| Kind | Type | Value | What it is for |
|---|---|---|---|
| boolean | `schedule:BooleanSchedule` | true / false | on/off by time: occupancy, lighting, a fan enable |
| numeric | `schedule:NumericSchedule` | a number | a setpoint that changes by time |
| enum | `schedule:EnumSchedule` | an ordinal of its `range` facet | named modes (occupied / standby); set the range before adding events |
| string | `schedule:StringSchedule` | a string | a text by time |
| calendar | `schedule:CalendarSchedule` | (dates only) | a list of days (holidays) that weekly schedules reference |
| trigger | `schedule:TriggerSchedule` | (times only) | fires its `trigger` topic at times of day on chosen days; link it to an action |

The first four are weekly schedules. Their `out` follows, highest first: a non-null `in`; the
default output when today is outside the effective period; the active special event highest in
the list; the active weekly time range; the default output (false for boolean, a null value for
the others). A range runs from its start (inclusive) to its end (exclusive), and an end of 00:00
is the end of the day. A special event controls only the hours it defines: to override a whole
day, give it an all-day range. The station recalculates `out`, `nextTime` and `nextValue` after
every change; no action is needed.

Special events (and calendar dates) come in five kinds: `date` (one day, or every day matching
the fields given: year, month, day, weekday), `date_range` (start to end inclusive),
`week_and_day` (a weekday in a week of a month, e.g. the 4th Thursday of November), `custom`
(lists of years, months, days, weeks and weekdays) and `reference` (the days of a calendar: edit
the calendar and every referencing event follows). A calendar holds the first four. With
`cleanupExpiredEvents` (default true) the station deletes `date` and `date_range` events once
their last day has passed; the other kinds never expire. Schedules can also be exported from one
station and imported read-only into another (`niagaraDriver:NiagaraScheduleExport` /
`NiagaraScheduleImportExt`, and the `bacnet:` equivalents); mcp-n4 has no tool for that.

How a station stores them (schedule-rt 4.15, confirmed live on a 4.15.6.20 lab station on
2026-11-15 over BOX): a weekly schedule's ranges are `schedule:TimeSchedule` children of
`schedule/week/<weekday>/day`, each with `start`, `finish` and a dynamic `effectiveValue` of the
schedule's Status type (flags `o1`); special events are `schedule:DailySchedule` children of
`schedule/specialEvents` whose slot order is their priority, with the day rule in `days` and the
ranges in `day`. Live checks: a range added without `effectiveValue` is ignored by the station
(out did not change); the week-of-month set takes 16 values (`Any Week`=0, `Week 1`..`Week 5`=1-5,
`Last 7 Days`=6, `Calendar Week 1`..`6`=7-12, `7 Days Before Last 7/14/21`=13-15, from the
station's type registry); a date special event for the current day switched `out` to its value and
`outSource` to `Special Event: <name>` at once, and removing it switched it back.

### Reading a schedule

`n4_read_schedule(ord)` decodes one schedule of any kind in one load. Weekly kinds return `out`,
`next_time`, `next_value`, `default_output`, `in` (each value `{value, status}`: a boolean, a
number, an enum ordinal or a string, `null` when the status is null), `out_source`, `effective`
(`always` or the date range rule), `weekly` (monday to sunday, each a list of `{start, end,
value}` sorted by start, `HH:MM`, end `24:00` for a range that runs to midnight) and
`special_events` in priority order (`name`, `kind`, `rule`, `priority` 1 = highest, `ranges`).
A calendar returns its `dates`; a trigger its `times`, `dates`, `enabled`, `last_trigger` and
`next_trigger`. Rules use the field names of `n4_list_schedule_kinds` (`"*"` matches any). A range
the station holds without a value is listed with `value: null, ignored: true`. The station leaves
out every slot that holds its type's default (a 00:00 start, a false `alwaysEffective`); the tool
fills those from the station's type registry and refuses, naming the slot, rather than guess one
the registry does not describe.

### Writing a weekly schedule and its default output

`n4_set_schedule(ord, weekly, default_output)` (a write tool: `--allow-writes`, a write scope, dry run then
`confirmation_token`) replaces the days given and only those: `{"monday": [{"start": "08:00",
"end": "18:00", "value": true}]}` leaves tuesday to sunday as they are, and `{"sunday": []}`
clears sunday. Times are `HH:MM`, `24:00` only as an end (midnight); a range covers its start up
to, not including, its end. Values follow the kind: `true`/`false`, a number, an enum ordinal of
the schedule's `range` facet, or a string. Every fault is named in one refusal before anything is
sent (an unknown day, a malformed time, an end not after its start, overlapping ranges, a missing
or mistyped value, a calendar or trigger, which have no weekly schedule).

`default_output` is the value `out` takes when no range or special event is active: a value of
the kind (a calendar's is a boolean) or `null` for a null status. A trigger has none.

Each day is one BOX set that replaces its DaySchedule whole, the default output one set of its
Status, and each inverse is a set of what was loaded (a default output the station left out is
restored from its declared default), so `n4_rollback` restores the previous schedule exactly.
The read-back reloads the schedule and compares each part written with what `n4_read_schedule` shows; the station
recalculates `out`, `next_time` and `next_value` by itself.

### Special events

`add_special_event` and `remove_special_event` change the special events of a boolean, numeric,
enum or string schedule in the same batch:

```json
{"ord": "station:|slot:/Schedules/Lobby",
 "remove_special_event": ["Christmas Break"],
 "add_special_event": [{"name": "Plant Shutdown", "kind": "date_range",
   "rule": {"start": {"year": 2027, "month": "dec", "day": 24}, "end": {"year": 2027, "month": "dec", "day": 31}},
   "ranges": [{"start": "00:00", "end": "24:00", "value": false}]}]}
```

`kind` and the `rule` fields are those `n4_list_schedule_kinds` lists and `n4_read_schedule`
returns (`"*"` or a missing field matches any; `custom` fields are lists). The events keep their
order, which is their priority (first wins), and new events go last; to change one, remove and add
it in one call. A `date` or `date_range` that ends before today (the date of the machine running
the server) can never occur again: it is refused, naming `allow_expired: true`, which adds it
anyway. A `reference` must name an existing calendar (`schedule:CalendarSchedule`; it is only
read, so it may sit outside the write scope), and a name given twice to remove is refused. The
whole list is one set, so `n4_rollback` restores the events exactly as they were; the read-back
compares each event's name, rule and ranges, and the rollback compares the list it wrote (names,
order, rules and ranges) before overwriting it and after restoring it.

### Calendar dates and trigger times

`add_date` ([{name, kind, rule}], kind `date`, `date_range`, `week_and_day` or `custom`) and
`remove_date` ([names]) change the dates of a calendar, or of a trigger's own `dates` calendar,
under the same expiry rule as special events. A date is a component of the calendar itself, so
each removal is one remove and each addition one add of the bare date type followed by one set
per part (`<name>/yearSchedule`, `/monthSchedule`, ...); the inverse re-adds a removed date the
same way from its loaded parts (live on 4.15.6.20: removing two dates and rolling back read the
calendar back identical). A date cannot be removed and added under the same name in one call:
replace it in two calls, so each rolls back on its own. The rollback compares each date part by
its rule, so a date the station restores with another month reads as a mismatch. `times` (["06:00", "18:30"]) replaces a
trigger's firing times with one set.

### Recipe: a schedule that drives a point

1. `n4_create_component` the schedule (`schedule:BooleanSchedule`, or the numeric, enum or string
   kind that matches the point) next to the point.
2. `n4_set_schedule` its `weekly` ranges, `default_output` and any `add_special_event`.
3. `n4_create_link` the schedule's `out` to the writable point's `in16`. By convention a weekly
   schedule drives priority level 16, though any level is allowed (Niagara help,
   AboutLinkingWeeklySchedules: "By convention, when linking to a target writable point (with
   16-level priority array), you select In16 among its different priority array inputs. However,
   you are free to select any available input level."). A calendar is not linked: special events
   reference it.

Live on 4.15.6.20: a BooleanSchedule with `default_output` false and a special event for today
(00:00-24:00 true), linked to a BooleanWritable's `in16`, put the point's `out` at true @ 16
("Special Event: Today"); removing the event put both back at false (Default Output).

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
with `dry_run=true` and, only when `--execute` is given, executed with the `confirmation_token`
that dry run returned. Without `--execute` every write is a dry run only (`--dry-run-only` says
so explicitly and excludes `--execute`), so a flag lost on the way to n4call never executes a
write. A destructive write (`n4_remove_component`, `n4_rollback`, `n4_save_station`,
`n4_ack_alarm`, `n4_set_tags`, `n4_set_schedule`, `n4_invoke_action`: the server's `destructiveHint`) executes only when the run has
`--execute` and `--allow-destructive` AND its own line carries `"execute": true`
(`{"tool": "n4_remove_component", "write": true, "execute": true, "args": {...}}`); a run with
`--execute` that misses either is refused with exit 2 before the server starts, naming the line.
`"execute"` on a line without `"write": true` is a usage error. A run with writes says on stderr
whether they execute. Output is one JSON line per request (`tool`, `phase`, `error`, `result`);
credential values are masked; the run stops with exit 1 at the first tool error. A dry-run-only
run exits 0 like an executed one: a caller that meant to execute checks for `"phase":"execute"`
lines in the output (or for the stderr line `n4call: dry run only`), never the exit code alone. The server
command line after `--` is the operator's: scopes, tier opt-ins and station homes are whatever it
names, n4call adds nothing.

The server side needs no client discipline: a confirmation token is issued only by a dry run,
keyed by a random per-process key, bound to the tool, the arguments, the plan hash (which names
the connect session) and an expiry, and spent once; a token from another process, another connect
session or no dry run at all is refused (`blackbox.TestATokenFromAnEarlierServerProcessIsRefusedAfterARestart`,
`blackbox.TestAWellFormedTokenNoDryRunIssuedIsRefused`). What a client must not do is chain the dry
run into the execution by default: that is what the n4call rules above prevent.

A laboratory station has no field device to prove device, proxy-point, polling and write work
against: `go/bin/lab-sim` (`go/cmd/lab-sim`) simulates one on loopback (Modbus TCP, a scenario
file, a request log and a control port); see `go/internal/labsim/README.md`.

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
