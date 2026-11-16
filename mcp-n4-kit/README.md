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
- `--niagara-home NAME=PATH` (repeatable): Niagara installation directory of configured station
  NAME (the one holding `security/licenses`), when it is reachable from the machine that runs the
  server; lets `n4_license_info` add when each licence was generated and its version. Only the
  attributes of the root element of each `*.license` file are read.
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
`n4_list_px_views`, `n4_px_summary`, `n4_spy_page`, `n4_station_health`,
`n4_read_history`, `n4_license_info`, `n4_read_alarms`, `n4_list_alarm_classes`,
`n4_commissioning_report`.
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
History records: `n4_read_history(history_id | ext_ord, limit=20, period | start/end)` returns
the last `limit` records (at most 1000) of one history, newest first (`order: newest_first`), so
a rollout proves records arrive, not only that the extension runs. Give exactly one of
`history_id` (`/<station>/<history name>`, two slot-name segments as `BHistoryId` parses them;
anything else, `|`, `?`, `;` and spaces included, is refused before any request) or `ext_ord`
(a `station:|slot:/...` history extension ORD, resolved with one
`select slotPath, historyConfig.id from history:HistoryExt` at the component holding it). It
sends `GET /ord/<url-encoded history:<id>[?<range>]|bql:select timestamp, value, status order by
timestamp desc|view:file:ITableToCsv>`, then the same ORD with `select count(*)` for
`total_records` (`null` with `total_records_note` when that count fails). The range is
`period=<id>` (`today`, `last24Hours`, `yesterday`, `weekToDate`, `lastWeek`, `last7Days`,
`monthToDate`, `lastMonth`, `yearToDate`, `lastYear`: the ids `BDynamicTimeRange` accepts) or
`start=<t>;end=<t>` (BAbsTime text such as `2026-11-15T00:00:00.000-06:00`), not both. Rows are
`{timestamp, value, status}` read by position, as the station renders them (localized display
text); `truncated: true` means the range holds more records than were returned. A history the
station cannot read is an error naming the id and how to list ids
(`n4_bql_query` `select historyConfig.id from history:HistoryExt`); an AuthError stays one.
There is no `output_file`: it is not an export of whole histories.
Alarms: `n4_read_alarms(limit=50, source, alarm_class, source_state, ack_state, since, until,
open_only=true)` returns alarm records newest first (`order: newest_first`), by default the open
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
`timestamp.millis >= <epoch ms>` / `<=`. Every value is checked before any request (no quote,
wildcard or BQL of the caller reaches the station); `where` echoes the clause sent. Rows are
`{timestamp, uuid, source, source_state, ack_state, alarm_class, priority, msg_text}` read by
position, as the station renders them (`msg_text` may be a raw `%lexicon(...)%` pattern).
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
with `rows_truncated` when a query feeding them was cut. `capacity` is the Global Capacity table
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

### Finishing a job: the commissioning report

`n4_commissioning_report(scope="station:|slot:/", sections=None, limit=20, expect_histories=None,
expect_alarms=None, link_depth=2, max_components=5000)` answers "what is still wrong or unfinished
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
| `naming` | `duplicate_display_names` among sibling points, `escaped_names` (`$20` and the like in the slot name), `palette_default_names` (a point of the control palette, `NumericWritable`, `BooleanPoint1`; or a point called by the name of its own type, a `kitControl:Tstat` called `Tstat`: the fact says which) | the points table |
| `station` | `services_in_fault`, `licence_expiry` (the warning of the health tool's `license` section), `unsaved_changes` and `last_backup` | one slot load of `Services`, the SystemService slots and the licence page; only when the scope is the whole station |

`unsaved_changes` and `last_backup` are always `not_checked`, with the reason: the station exposes
neither to a slot read or a query, and the report does not guess. `expect_histories` and
`expect_alarms` say which points should carry the extension: `{"kinds": ["numeric", "writable"],
"name_pattern": "*Temp*"}` (kinds are matched against the point's type name, the glob against its
name or display name; `{}` is every point). Without `expect_histories` the points with no history
extension are only counted (on most stations that is nearly every point); without `expect_alarms` the absence of an alarm
extension is not a finding and the check is skipped.

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

## Writing safely

Write tools exist only with `--allow-writes` (otherwise `tools/list` omits them
and calling one returns JSON-RPC -32601): `n4_create_component`, `n4_add_history`, `n4_add_alarm`,
`n4_copy_component`, `n4_set_slot`, `n4_set_flags`, `n4_invoke_action`, `n4_create_link`, plus the destructive class
(`destructiveHint: true`) `n4_remove_component`, `n4_rollback`,
`n4_save_station` and `n4_ack_alarm`. Example:

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
  and the read-back checks the capacity when one was asked.
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
  source is refused, as is a station whose type definitions cannot be read. Change
  what differs per copy afterwards (e.g. `deviceAddress` with `n4_set_slot`). The
  inverse removes every copy under its assigned name; the read-back loads each copy
  and compares its whole configured subtree with the plan, listing the paths that differ.

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
  too, see `n4_remove_component`). A frozen child (e.g. a writable's `proxyExt`) is
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
(`--dry-run-only` stops after the dry run). A write of a destructive tool (`n4_remove_component`,
`n4_rollback`, `n4_save_station`, `n4_ack_alarm`: the server's `destructiveHint`) is refused with exit 2 before
the server starts unless `--allow-destructive` or `--dry-run-only` is given, and a run with
writes says on stderr whether they execute. Output is one JSON line per request (`tool`,
`phase`, `error`, `result`); credential values are masked; the run stops with exit 1 at the
first tool error. The server command line after `--` is the operator's: scopes, tier opt-ins
and station homes are whatever it names, n4call adds nothing.

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
