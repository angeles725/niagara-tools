# mcp-n4-kit: Python-to-Go port plan

This plan ports the mcp-n4 server from Python to Go without changing its observable
behaviour. It was written against `origin/main` at v0.9.0: the server is about 4.6k lines
across 9 modules, and the suite has 573 tests across 13 files plus `tests/fake_station.py`.

Most of those tests are **not** black-box. They call `srv.dispatch(...)` in-process and reach
into internals (`box.*`, `safety.*`, `tools_write._frozen_check`, `srv.ctx.session = ...`).
A black-box parity run therefore needs a stdio adapter and a per-test skip inventory first:
that is slice S0.

## 1. Package layout (`go/internal/mcpn4/…`, thin `go/cmd/mcp-n4`)

| Go package | Python source (origin/main) |
|---|---|
| `pyjson` (new) | `json.dumps`/`loads` behaviour used in `server.py:89-111`, `safety.py:51` (`canonical`) and `safety.py:217-233` (JSONL append) |
| `jsonrpc` | `server.py:22-190` (dispatch, `_validate` 38-62, protocol versions) |
| `cli` | `server.py:191-312` (`parse_args`, bounded ranges) |
| `box` | `box.py:15-689` (client, auth/HTTP errors, poll delays, `load_tree`, event split, sync ops, BSON encoders 90-113) |
| `ord` | `box.py:119-147` (`child_ord`/`join_ord`), `safety.py:115-155` (`_plain_slot_ord`, `WriteScope`) |
| `tiers` | `tiers.py:1-76` |
| `session` | `tools_read.py:33-195` (Context, auth latch/cooldown, scrub) |
| `readtools` | `tools_read.py:198-500, 789-969` (connect … dangling outputs, `list_batches`, TOOLS) |
| `bql` | `bql.py:1-221`, `tools_read.py:500-654` (output files, private dirs) |
| `files` / `px` / `pyxml` | `files.py:1-233` (Windows alias refusals at `station_path` 38-86), `tools_read.py:689-822`; `pyxml` is the expat parser `ET.fromstring` uses |
| `token` | `safety.py:47-113` (canonical, `bound_args`, redact, `ConfirmationTokens`) |
| `journal` | `safety.py:156-311` (`_JsonlFile`, `Journal` views, `AuditLog`; the state-dir checks are `safety`'s since S6) |
| `retro` | `retro.py:1-289` and `templates/retro.template.md` |
| `plan` | `tools_write.py:76-216, 1260-1500` (coerce, to-BSON, `_process`, `_execute`, `_send_all`, `_record`) |
| `writeops` | `tools_write.py:217-1240` (create, set, invoke, link, remove/snapshot/relink, rollback/frozen, save) |

`tools/*.py` (`mutation_check.py`, `live_smoke.py`, `new_retro.py`) stay Python: they are dev
tooling, not the server.

## 2. Parity strategy

**Black-box adapter.** `tests/go_server.py` holds a `GoServer` whose `dispatch(msg)` writes one
line to a long-lived `go/bin/mcp-n4` subprocess and reads one reply line. `ToolTestCase.setUp`
(`tests/test_server.py`) picks it when `MCP_N4_IMPL=go`.
- FakeStation stays Python on loopback. Test setup keeps using a Python `BoxClient` to arrange
  station state, and the Go server talks to the same FakeStation.
- The `hook(frame)` and `on_sync` fault seams keep working unchanged.
- Tests that touch internals or inject clocks/factories get a skip under Go plus a 1:1 Go table
  test. S0 publishes that inventory, with a count per file (section 5).

**Recorded-frame differential.** FakeStation gets a recorder that logs every raw BOX request
body and reply, plus the MCP stdin/stdout. A replay station then serves the recorded frames to
both servers. The harness diffs, byte for byte:
- the stdout reply lines,
- the BOX request bodies the station receives,
- `audit.jsonl` and `journal.jsonl`.

The harness masks only nonce, token, batch_id, session_id, plan_hash and timestamps. Production
code gets no new seams. The harness follows the `javasrc/differential_test.go` pattern and
skips when python3 is missing.

**Pinned JSON risks.** All of these live in one `pyjson` package, proven against python3 over a
fuzz corpus:
- **Separators:** `json.dumps` writes `", "` and `": "`, but `canonical` uses compact
  separators with `sort_keys`. The encoder takes both as options.
- **Escaping:** Python escapes all non-ASCII as `\uXXXX` (`ensure_ascii`, surrogate pairs) and
  does **not** HTML-escape. Go escapes `<>&` and passes UTF-8 through raw. Neither Go default
  matches.
- **Key order:** replies follow dict insertion order. Use ordered structs or an ordered map,
  never `map[string]any`.
- **Number decoding:** `UseNumber` keeps the int/float distinction. This matters because
  `_validate` rejects bools as integers, and `_coerce` (`tools_write.py:76`) gives `100` and
  `100.0` different treatment upstream.
- **`str(float(x))`:** reproduce Python's repr exactly. The table must pin at least: `100.0`,
  `1e+16` (exponent from 1e16), `1000000000000000.0`, `0.0001`, `1e-05` (two-digit exponent),
  `-0.0`, `nan`, `inf`.
- **Overflow:** an integer literal beyond float range, such as `1e400` written as an integer,
  raises `OverflowError` in Python, not `ToolError`. Decide that error text explicitly.
- **Existing state files:** Go must read journals and audit logs that Python already wrote,
  including in-doubt intents. Use fixture files for this.

## 3. Ordered slices (each ≤ ~400 lines)

Until S16 the Go binary is opt-in only: the MCP config keeps launching Python. Before S10, Go
refused `--allow-writes` with the existing "not in this version" refusal; since S10 it starts
with it and serves the write tools ported so far (section 14).

| # | Slice | Proof |
|---|---|---|
| S0 | `GoServer` adapter, impl switch, frame recorder, skip inventory (Python only) | Python suite still 573 green |
| S1 | `pyjson` + float repr | Differential vs python3 on the fuzz corpus and the boundary table |
| S2 | `jsonrpc` + `cli`; all tool schemas registered | `TestProtocol`/`TestRobustness` black-box; `tools/list` bytes equal |
| S3 | `box` read side (`box.py:229-612`) | Ported `test_box` cases; replay diff of `load_tree` frames |
| S4 | `tiers`, `session`, connect/describe | `test_auth_latch`, `test_tiers`, `TestConnect` black-box |
| S5 | navigate, read_slots, links, dangling | Matching `test_server` classes black-box |
| S6 | `bql` + inventory + output files | `test_bql` black-box; CSV output bytes equal |
| S7 | `files` + `px` | `test_files`; table of `:`, trailing dot/space, `..`, drive, control-char refusals |
| S8 | `token`, `ord`/WriteScope, `audit`/`journal`, `list_batches` | `test_safety` ported; Python-written journal fixtures read identically |
| S9 | `retro` + `session_retro_draft` | `test_retro` output diff |
| S10 | `plan` engine + set_slot + invoke | `test_tools_write` black-box; token bound to plan hash incl. session_id; reuse/expiry/mismatch refusals |
| S11 | create_component + link | `TestTokenLayers`/`Scope` black-box |
| S12 | remove + snapshot + outgoing scan + relinks | Destructive tests 77-494 |
| S13 | rollback + frozen check + unchanged_since | Rollback tests; prior-value comparisons exercise float repr |
| S14 | save_station + `.bog` state | Save tests (mtime/sha256 branches 1216-1224) |
| S15 | #304 in-doubt matrix | See below |
| S16 | Switch the default launcher to Go | Python kept one release as fallback |

**Status (server v0.9.5).** S0-S11 merged: #337, #345, #353, #365, #383, #399, #413, #435, #451,
#462, #475, #489; issues #448 (unreadable journal lines, section 12) and #466 (unreadable audit lines,
section 13) then changed both servers together.
`readtools.Register` (`go/internal/mcpn4/readtools/connect.go`) serves all 13 read tools:
`n4_connect`, `n4_describe_session`, `n4_navigate`, `n4_read_slots`, `n4_list_links`,
`n4_find_dangling_outputs`, `n4_bql_query`, `n4_inventory`, `n4_read_file`, `n4_list_px_views`,
`n4_px_summary`, `n4_list_batches`, `n4_session_retro_draft`. S10 (section 14) adds the plan
engine with `n4_set_slot` and `n4_invoke_action`: `--allow-writes` starts the Go server. S11
(section 15) adds `n4_create_component` and `n4_create_link`, S12 (section 16)
`n4_remove_component`; `n4_rollback` and `n4_save_station` answer the "not implemented" text
(section 6) until S13-S14. Under the Go binary the suite runs 662 tests with 365 skipped
(section 16).

**S15 in-doubt matrix.** Each branch gets an explicit Go test plus the black-box Python test,
driven by `hook` faults and an unwritable state dir:
- `_in_doubt` (1075),
- `_in_doubt_partial`, including the "could not be journaled" suffix (1346-1375),
- `_in_doubt_with_created` at 797, 812, 818, 1025, 1046, 1056,
- the relink intent unwritable path, so no link is sent (`test_tools_destructive` 1270, 1287),
- the batch id kept after an unexpected failure (1354),
- refusal of an in-doubt batch or in-doubt rollback (626-642).

Then hand-mutate each Go branch: it must go RED, and GREEN after restore.

If a write slice runs over 400 lines, split it per op.

**Switch-over criterion for each slice:** its black-box classes are green under
`MCP_N4_IMPL=go`, the replay diff is empty, and gofmt, vet and test pass in CI.

## 4. Risks and go/no-go for retiring Python

**Risks**
- **HTTP differences** between `urllib` and `net/http`: User-Agent, `_NoRedirect` semantics,
  Basic auth header, insecure TLS, the 32 MB / 2 MB / 64 KB read caps, timeout error text. Pin
  each one in the replay diff.
- **Error text** feeds tests and the audit log. Copy the `REASON_*` strings verbatim.
- **Clock and nonce injection** is only possible in-process, so expiry tests are Go-only.
- **File permissions:** state-dir 0700/0600 and O_EXCL semantics must match, including the rule
  that an existing file is never chmodded.

**Go when all of these hold**
1. All 573 tests are green: black-box under Go, or skipped with a named Go twin; zero
   unexplained skips.
2. The replay diff is empty over the full recorded corpus, including every write tool and every
   in-doubt fault.
3. Go reads Python-written state dirs, and an in-doubt batch from Python is still refused by Go.
4. The mutation check is RED on every write and in-doubt branch.
5. One release runs with the Python fallback and no reported regression.

**No-go if:** any unmasked byte differs in stdout, BOX bodies or the journal; any token or scope
check is weaker; or a Go test needs a live station.

## 5. S0 harness and skip inventory

**Running the suite against Go.** `MCP_N4_IMPL` selects the server the tests drive:

```
# Python, in-process (the default; no Go binary needed)
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s mcp-n4-kit/tests
# Go binary over stdio (MCP_N4_GO_BIN, else go/bin/mcp-n4)
MCP_N4_IMPL=go python3 -m unittest discover -s mcp-n4-kit/tests
```

- `tests/go_server.py`: `make_server(**kw)` builds `mcp_n4.server.Server(**kw)` or `GoServer`,
  which maps the same keywords to command-line flags. Each request is followed by a `ping`
  sentinel, so a notification (no reply) never blocks, and more than one reply line is an error.
- A missing binary raises `GoBinaryMissing` (a `RuntimeError`, so a test expecting a startup
  `ValueError` cannot swallow it), and an unknown `MCP_N4_IMPL` value is a `ValueError`: a Go run
  never passes on Python by accident. A start refusal (exit 2) is a `ValueError` carrying stderr,
  like `Server(...)`.
- Under Go, `ToolTestCase.station_client()` arranges station state through a separate Python
  `BoxClient` on the same FakeStation; in-process it is the session's own client, as before.
- `FakeStation.record()` turns on the frame recorder (off by default): `frames` holds one
  `{"request", "status", "reply"}` entry per BOX POST with the raw bodies, including injected
  `hook` replies and 401s; `dump_frames(path)` writes them as JSON Lines for the replay
  differential.
- `@go_server.go_twin(package, category)` skips a test, or every test of a class, under Go and
  names its twin, `<package>.Test<CamelCaseName>` in `go/internal/mcpn4/<package>`.

**Validating the inventory.** Any stdio server can stand in for the binary. With
`MCP_N4_GO_BIN` pointing at a wrapper that runs `python3 -m mcp_n4.server "$@"` from the kit
directory, the whole suite passes under `MCP_N4_IMPL=go` (590 tests, 339 skipped at S0; 622 tests,
349 skipped at `818d573e`, the inventory total below): every unmarked server test is black-box
over stdio.

**Categories.**

| Category | Meaning |
|---|---|
| `unit` | Calls a Python module function or class directly; the twin is a Go unit test |
| `internals` | Reads or writes in-process server state (`srv.ctx`: session, env, write state, journal) |
| `injection` | Replaces behaviour in-process (mock/patch, clock, `client_factory`, opener, custom tools) |
| `in-process` | Uses an entry point with no wire form (`parse_args`, `main`, `serve`, `_validate`, docstrings, the Python `TOOLS` table, an in-process `Server`) |

`test_live_smoke.py`, `test_mutation_check.py`, `test_new_retro.py` and
`test_methodology_names.py` cover dev tooling that stays Python, and `test_go_harness.py` covers
the harness itself: they run unchanged under both implementations. Several `in-process` tests
(start refusals, `TestRobustness` id handling) can move to `make_server` in the slice that ports
their code instead of getting a Go twin; S9 moved `test_retro.TestServerTool` this way.

The table is generated: `python3 mcp-n4-kit/tests/go_server.py` prints it, and
`test_go_harness` fails when it and the markers disagree.

<!-- go-skip-inventory:begin -->
| Test file | Tests | unit | internals | injection | in-process | Skipped under Go | Runs under Go |
|---|---|---|---|---|---|---|---|
| `test_auth_latch.py` | 10 | 0 | 0 | 10 | 0 | 10 | 0 |
| `test_box.py` | 92 | 69 | 0 | 23 | 0 | 92 | 0 |
| `test_bql.py` | 39 | 15 | 0 | 20 | 0 | 35 | 4 |
| `test_files.py` | 30 | 14 | 2 | 0 | 0 | 16 | 14 |
| `test_go_harness.py` | 25 | 0 | 0 | 0 | 0 | 0 | 25 |
| `test_list_batches.py` | 6 | 0 | 0 | 0 | 0 | 0 | 6 |
| `test_live_smoke.py` | 30 | 0 | 0 | 0 | 0 | 0 | 30 |
| `test_methodology_names.py` | 1 | 0 | 0 | 0 | 0 | 0 | 1 |
| `test_mutation_check.py` | 7 | 0 | 0 | 0 | 0 | 0 | 7 |
| `test_new_retro.py` | 17 | 0 | 0 | 0 | 0 | 0 | 17 |
| `test_retro.py` | 36 | 20 | 0 | 1 | 0 | 21 | 15 |
| `test_safety.py` | 46 | 33 | 0 | 13 | 0 | 46 | 0 |
| `test_server.py` | 101 | 4 | 14 | 5 | 7 | 30 | 71 |
| `test_tiers.py` | 25 | 5 | 1 | 1 | 2 | 9 | 16 |
| `test_tools_destructive.py` | 101 | 4 | 22 | 28 | 2 | 56 | 45 |
| `test_tools_write.py` | 76 | 2 | 10 | 4 | 7 | 23 | 53 |
| `test_write_parity.py` | 20 | 0 | 0 | 0 | 0 | 0 | 20 |
| **Total** | **662** | **166** | **49** | **105** | **18** | **338** | **324** |
<!-- go-skip-inventory:end -->

## 6. S2 protocol layer

**Registry.** `go/internal/mcpn4/registry/registry.json` is the Python surface the Go server
serves verbatim: `SERVER_NAME`, `SERVER_VERSION`, `SUPPORTED_PROTOCOL_VERSIONS`, the argparse
usage and help text (`server.build_parser()` at 80 columns, no color), and every tool's name,
description, `inputSchema`, annotations and `needs_session` in table order. It is generated, never
edited: after changing a tool table, a flag or the version, run
`python3 mcp-n4-kit/tests/go_registry.py --write`. `test_go_harness.TestGoRegistry` fails while
the file and the Python tables disagree, under both implementations.

**Packages.** `jsonrpc` holds framing (`str.strip()` whitespace, lines split at `\n` only, as
Python's stdin splits them on POSIX), dispatch, `Validate` and the reply shapes, all encoded with
`pyjson`. `cli` holds the argparse surface (exact option strings, unique long prefixes, `-h`
with attached flags, `--`, the type and range messages, usage then
`mcp_n4.server: error: ...`, exit 2) and the start checks of `main()`/`Server(...)` (NAME=URL and
NAME=PATH pairs, https URLs, names the per-station flags may use). `go/cmd/mcp-n4` only calls
`cli.Main`. Both packages carry a differential test that runs the same lines / argv through
python3 and compares the bytes.

**Not ported yet.** A read tool that needs a session answers `not connected: call n4_connect
first` without one and the auth pause while it holds (section 8), as Python does; past those
checks, and for any other tool, an unported tool answers `<name> is not implemented in this
version of the Go server`; until S10 `--allow-writes` was refused at start with
`mcp_n4.server: --allow-writes is not implemented in this version of the Go server` (exit 2).
Under `MCP_N4_IMPL=go` the harness skips a test that meets exactly that text
(`go_server.skip_if_unported`), so these skips shrink by themselves as slices land and are not
in the inventory above. Known command-line differences: `int()` flag values take ASCII digits
only (Python also takes other Unicode decimal digits; `float()` values take them since S10),
integers past int64 saturate, and Python's `str.isprintable` is approximated by
`unicode.IsPrint` in `%r` text.

**Nesting limit (R8-D15).** CPython's `json` has no fixed depth limit: it raises
`RecursionError`, not `ValueError`, at a depth set by the C stack, and that escaped
`handle_line` and ended `serve`. Both servers now answer the normal parse error for a request
nested deeper than `server.MAX_NESTING` (1000, arrays and objects with the outer object
counted); Go checks the depth before decoding (`jsonrpc.NestsDeeper`).

## 7. S3 `box` read side

**Package.** `go/internal/mcpn4/box` ports `box.py:229-612` plus the helpers it needs
(`poll_delays`, `auth_message`, `_flatten`): `New` (the http/https refusals), `Call`/`SSC`/`Poll`,
`Open`/`Close` (no cleanup `del` after an AuthError), the pending-event split, `LoadTree` with the
handle cache and the stale-handle early abort, `GetOrd` (HTML refusal; the cut-record trimming is
`bql.complete_records`, slice S6), `GetFile` and `About`. Errors are one `*box.Error` (`Kind`:
BoxError, AuthError, HttpStatusError) or `*pyjson.ValueError`. Since 0.9.2 no Python exception
escapes the client: a non-HTTP answer, a cut body and a reply of the wrong shape are BoxErrors in
both (see "Reply truth" below). Every
JSON body goes through `pyjson`; `pyjson.Loads` words a bad reply as `json.loads(raw.decode())`
does. The readers `children`/`status_value`/`parse_status` and the write side (`sync` and the
encoders) come with the slices that use them; no tool is wired yet (each read tool needs the S4
session).

**Proof.** `tests/go_box.py` holds one op language run by both clients:
- *Recorded frames*: `SCENARIOS` port the read-side `test_box` cases (and pin a few malformed
  replies). `python3 mcp-n4-kit/tests/go_box.py --write` runs them against the recording
  FakeStation into `go/internal/mcpn4/box/testdata/replay.jsonl`; `TestReplayRecordedFrames`
  replays the frames to Go and requires equal request bodies, no frame left over and equal op
  results. Nothing is masked: the FakeStation session id and handles are deterministic.
  `test_go_harness.TestGoBoxReplay` fails while the file is stale.
- *Live*: `TestLiveDifferential` runs `go_box.py --live` and the Go client against the same Go
  test server (redirects, statuses, caps, BOM, invalid UTF-8, a short body, a non-HTTP answer, a
  timeout, a refused port, the GET reads) and compares results, request bodies and every request
  header except User-Agent.

**Target Python.** Every Go differential (pyjson, jsonrpc, cli, box) compares against one CPython,
`pytarget.Minor` = 3.15: the `python3` the MCP config launches on the reference host. Some of the
compared text moves between minors (JSON decode errors say "Illegal trailing comma" since 3.13),
so `pytarget.Python` skips a differential, printing the version found and the target, when
`python3` is another minor; with `MCP_N4_PYTHON` set the mismatch fails instead. CI installs 3.15
with `actions/setup-python` before the Go step and runs `go test` with `MCP_N4_PYTHON` set to it,
so a wrong interpreter fails CI rather than skipping; the mcp-n4-kit unittest step runs on it too.
Locally `scripts/ci-local.sh` uses the `python3` on PATH. The kit suite, replay-corpus check
included, also passes on 3.12.

**Known differences.** The User-Agent is Go's, not `Python-urllib/3.x`. A TLS verification
failure and other OS errors without an errno keep Go's wording inside `<urlopen error ...>`.
`http_proxy`/`https_proxy` are honoured by both, but Go never proxies
localhost.

**Reply truth (0.9.2).** The S3 port pinned three Python behaviours that misreported a failure;
both clients now share the fixed behaviour, proven by `test_box.TestTransportTruth` /
`TestReplyShapes` on the Python side and by the replay corpus and the live differential on Go's:
- An answer that is not HTTP (`BadStatusLine`) or a chunked body cut short (`IncompleteRead`) is
  `transport failure: invalid HTTP answer (<type>)`; Go maps its "malformed HTTP" error and a
  chunked `io.ErrUnexpectedEOF` to the same names.
- A body shorter than both its Content-Length and the read cap is `transport failure: incomplete
  read (got N of M bytes)`, never a complete result (`truncated=false` on a cut table). A body
  longer than the cap stays the documented truncation.
- `pollchgs` must answer a list (`null` reads as none); a load tree's `s` must be a list of objects,
  each with a non-empty string `n`; otherwise `malformed BOX reply: ...`. An early event whose
  ops are no list is kept, never walked.

The two S3 divergences are closed the same way: a load child whose `n` is not a string (a
non-string dict key in Python, TypeError in Go) is `malformed BOX reply: slot ... has a child
without a slot name` in both, and a reply nested deeper than `MAX_REPLY_NESTING` (1000, the
server's request limit) is `malformed BOX reply: nested deeper than 1000 levels` in both, so
CPython's RecursionError past its C stack no longer differs from Go decoding any depth. One edge
remains: a reply that is both invalid JSON and nested deeper than CPython's C stack (~52k) is a
nesting error in Python and a JSON error in Go.

## 8. S4 `tiers`, `session`, connect/describe

**Packages.** `tiers` ports `tiers.py` (`TierOf`, `ProductVersion`, `WriteBlock`); the About scan
keeps Python's Unicode `\s` and `\b` (`pyjson.IsSpace`, shared with `jsonrpc.Strip`).
`session` ports `Context` and `Session` (`tools_read.py:33-195`): the operator policy, the active
session, the tier report, the scrub and the auth latch (keyed credential fingerprint, whole-second
`retry in`, `AuthBlockFor` letting other credentials reconnect). `readtools` serves `n4_connect`
and `n4_describe_session`; `cli.NewContext` builds the context from the flags `cli.Check`
accepted. `jsonrpc` now follows `Server._call`: a session tool needs a connection, then no auth
pause, then its handler; an AuthError from a handler re-latches, a BoxError or ValueError shows
its words, every tool error text is scrubbed (not the unported text the harness keys on), and
`Serve` closes the station session however it ends. `pyjson.Truthy` is `bool(v)`, the one
truth test every package uses.

**Proof.** Under `MCP_N4_IMPL=go` the black-box `TestConnect`, `TestDescribeSession` and
`test_tiers.TestConnectReportsTier` tests run against the binary and FakeStation. Their marked
tests and every `test_auth_latch` test have Go twins: `session/connect_test.go`,
`session/authlatch_test.go`, `tiers/connect_test.go` and `cli/context_test.go`, on
`readtools/readtoolstest` (an in-memory station and a server whose environment, clock and client
factory a test sets). `tiers.TestDifferentialPython` compares the three tier functions with
`tiers.py` on the target CPython. Twins still owed: `TestAWriteDryRunIsRefusedWhilePaused` (S10)
and `TestTheRetroClassifiesTierRefusals` (S9).

**Non-UTF-8 credentials (R8-D19).** Python's fingerprint `.encode()` raised UnicodeEncodeError on a
credential that is not UTF-8, and its text named the bad character's position: the credential's
length. Both servers now refuse it before any station call with `credentials are not valid UTF-8:
check env vars <PREFIX>_USER and <PREFIX>_PASSWORD`
(`test_server.TestConnect.test_a_non_utf8_credential_is_refused_without_its_position`, black-box,
and its Go twin in `session/connect_test.go`). No S4 difference is left.

## 9. S5 navigate, read_slots, links, dangling

**Packages.** `box/readers.go` ports the readers of `box.py`: `Children`, `StatusValue`,
`ParseStatus` (`int(_, 16)`'s whitespace, sign, `0x`, underscore and Unicode-digit rules and its
200-character `%.200R` error cut) and `Node`/`ChildValue` for `nodes[path]` and
`.get(name, {}).get("v", default)`. `readtools` serves `n4_navigate` (`navigate.go`: `_ord_arg`,
`_is_component`, `_tree_entries`, `_types_arg`, `_timed_load`), `n4_read_slots` (`readslots.go`:
`_slot_entry`) and `n4_list_links` / `n4_find_dangling_outputs` (`links.go`: `_abs_path`,
`_source_path`, `_scan`). A dangling report appends the session observation the retro reads
(`session.Context.Observations`, `session.Now` = `retro.now()`); S9 consumes it. A node shape
Python cannot handle fails the same way: a truthy non-string type, status or source ORD is
`internal error (AttributeError)`, a list or dict where a set or dict key is needed (a filtered
type, a handle, a link's `sourceSlotName`) is `internal error (TypeError)`, both through
`pyjson.Exception`; status bits `int()` cannot read are its ValueError text.

**Proof.** Under `MCP_N4_IMPL=go` `test_server.TestNavigate`, `TestReadSlots`, `TestListLinks`,
`TestDanglingOutputs` and `TestDeepTargets` run black-box against the binary and FakeStation; no
test skips on these four tools any more. The three marked `TestReadSlots` tests have Go twins in
`readtools/readslots_test.go`, on `readtoolstest.Station.Trees` (the nodes a load of an ORD
answers). `readtools.TestDifferentialPython` runs the four tools on the same nodes in
`tools_read.py` on the target CPython and in Go and requires the same reply text byte for byte,
for a station-like subtree and for the odd shapes above; `box.TestParseStatusDifferentialPython`
does the same for `parse_status`. The replay corpus (section 7) is unchanged: these tools only add
`load_tree` calls it already pins.

**Known differences.** None found. `"out" in children(nodes, path)` is a direct `path/out` lookup
in Go, the same answer without a scan per component.

## 10. S6 `bql`, inventory and output files

**Packages.** `bql` ports `bql.py`: `query.go` (`validate_query`, `base_slot`, `compose_ord`,
`selected_slots`, `unique_columns`, `by_position`, `unescape_slot`, `complete_records`; `\s`, `\S`
and `str.strip()` use `str.isspace()`'s set, lengths count code points), `csv.go` (`parse_csv` on a
port of `_csv.c`'s reader state machine for the excel dialect over `StringIO(newline="")` lines,
with the 131072-character field limit) and `inventory.go` (`summarize_inventory`). `readtools`
serves `n4_bql_query` and `n4_inventory` (`bql.go`: `_bql`, `_record_read`, `_progress`) through
`box.Client.GetOrd` (S3), and writes result files (`output.go`: `_output_path`, `_private_dirs`,
`_write_output`; S7's `n4_read_file` reuses them). `safety` starts with what result files need:
`check_state_dir` and `loose_file_reason` (`statedir.go`) and the `os.path` rules the shown paths
follow (`pypath.go`: `join` and `dirname` without cleaning, `expanduser`); `cli` expands
`--state-dir` and `--progress-file` into `session.Config` as `server.py` does. `pyjson` gains WTF-8
walking (`Chr`, `DecodeRune`, `Len`): an escape such as `$ud800` decodes to a lone surrogate, as
`chr()` gives one.

**Files.** As in Python: the state dir and `bql/` are created 0700 (missing parents 0777 less the
umask, like `os.makedirs`) and chmodded only when this call created them; the result file is
opened `O_WRONLY|O_CREAT|O_EXCL` 0600 after an `lexists` check, so a dangling link is refused and
never written through, and it is removed on any failure. A CSV row is `csv.writer`'s excel line (a
cell holding `,` `"` `\r` or `\n` quoted with its quotes doubled, one empty cell as `""`, `\r\n`
ends); JSON is `json.dump`'s bytes. A lone surrogate in a CSV row fails as its
`UnicodeEncodeError` text (position, or position range, within that row); JSON escapes it.
`--progress-file` lines are appended best effort (0666 less the umask when created).

**Proof.** Under `MCP_N4_IMPL=go` `test_bql.TestNonCsvAnswers` runs black-box: no test skips on
`n4_bql_query` or `n4_inventory` any more (622 tests, 461 skipped, from 465). The 35 marked
`test_bql` tests have Go twins: the 15 unit ones in `bql/query_test.go`, `csv_test.go` and
`inventory_test.go`, the 19 injection ones in `bql/tools_test.go` and `output_test.go` on
`readtoolstest.Station.OrdRoutes` (test_bql's FakeOpener: an answer per needle, an HTTP code, a
byte cap), and `cli.TestServerFlagSetsTheProgressFile`. `bql.TestHelpersDifferentialPython` runs
every helper in `bql.py` on CPython 3.15 and in Go on one corpus (csv quirks, Unicode spaces,
`re.I`'s long s, escapes to control characters and surrogates, the inventory walk);
`readtools.TestBqlDifferentialPython` runs both tools in `tools_read.py` and in Go on the same
answers and state dir and requires the same reply text and the same result file bytes and mode.
Extra Go tests pin the permission rules: an existing result dir keeps its mode, a link is never
followed, a group-writable state dir or a loose `audit.jsonl` is refused and left unchanged.

**Known differences.** None found. `TestBqlAuditInWritesMode` (the `read` line in `audit.jsonl`)
had a named Go twin that skipped until S10, when `--allow-writes` started the Go server; it now
runs (`bql.TestTheReadIsAuditedWithOutcomeRead`).

## 11. S7 `files`, `px` and the station file reads

**Packages.** `files` ports the path helpers of `files.py`: `station_path` (every refusal in
Python's order and words: prefixes, length in code points, control characters, `|`, absolute,
home and drive paths, directories, empty / `.` / `..` segments, `:`, segments ending in `.` or
space, `.bog` / `.bog.gz`), `url_path` (`quote(seg, safe="")`, a lone surrogate failing as its
segment's UnicodeEncodeError) and `is_text`; `Lower` is `str.lower()` as an ASCII comparison sees
it (U+0130 lowers to `i` + U+0307). `px` ports `parse_px` and `bound_component`: the DOCTYPE /
ENTITY refusal is `re.I`'s match (U+0130 and U+0131 match `i`), lists are `_Bounded`, counts keep
first-seen order, `strip()` uses `str.isspace()`'s set. `pyxml` is `ET.fromstring(str)`: expat fed
UTF-8 (a declared encoding is ignored) with namespace processing, ported from niagara-go
`internal/pyxml` (b291c25c) without its DTD and encoding parts (parse_px refuses a DOCTYPE first;
`Parse` answers `ErrDoctype` there). The differential found four expat 2.9.0 behaviours that port
missed, now fixed here: an XML declaration after the root is junk, not misplaced; any non-ASCII
character after `<` outside the root starts an element; a declaration keyword (`<!KEYWORD`) is
ASCII only; and a prolog name's `:` makes it prefixed or an NMTOKEN (`:?`, `a:b:c?` invalid,
`a:` at the end unclosed), with `#` and `%` names stopping at `:`. `readtools` serves
`n4_read_file`, `n4_list_px_views` and `n4_px_summary` (`files.go`) on `box.Client.GetFile` (S3,
`KindHTTPStatus` for 403/404) and S6's `query`, `outputPath` and `writePrivate`; `recordRead`
names its target key (`file_ord`, `root`). `pyjson.EncodeUTF8` (S6's `utf8Strict`, moved) serves
both the CSV writer and `url_path`.

**Files.** `output_file` is S6's rule: `<state-dir>/files/<name>` for a plain name with an
extension, checked before the GET, the dirs 0700, the file `O_EXCL` 0600 and removed on failure,
never over an existing name or a dangling link; a read cut at `max_bytes` is not saved
(`output_skipped`). The read caps are `files.DefaultMaxBytes` (2 MiB) up to `MaxBytesLimit`
(16 MiB); `n4_px_summary` refuses a cut file, naming the cap.

**Proof.** Under `MCP_N4_IMPL=go` `TestReadFileTool`, `TestReadFileOutput`,
`TestListPxViewsTool` and `TestPxSummaryTool` run black-box (with stub handlers they failed 13
times): no test skips on these three tools any more (622 tests, 448 skipped, from 461). The 16
marked tests have Go twins: `files/paths_test.go` (TestFilePaths, plus the refusal table with its
words and order), `px/px_test.go` (TestPxParser) and `files/tools_test.go` (the two `internals`
ones on `readtoolstest.Station.Files`, plus output mode, overwrite, link and refusal-before-GET
checks). Differentials on CPython 3.15: `pyxml.TestDifferentialPython` (probes and 3,000 mutants
of two documents, plus niagara-go's 3,000-document expat suite when `PYXML_EXPAT_SUITE` names
it; 360,000 mutants were run once), `files.TestHelpersDifferentialPython` (the refusal table
behind both ORD prefixes, `url_path`, `is_text` against 16 content types),
`px.TestParseDifferentialPython` (edge documents at two caps, 1,500 mutants, `bound_component`)
and `readtools.TestFilesDifferentialPython` (the three tools: reply text and output file bytes and
mode). `pyxml.TestPinnedExpatAnswers` keeps the four fixes without a Python.

**Known differences.** None found. A DOCTYPE never reaches the parser in either implementation.

## 12. S8 tokens, write scope, journal and audit, `n4_list_batches`

**Packages.** `token` ports `safety.py:47-113`: `BoundArgs`, `Redact` (`str.lower()` is
`pyjson.Lower`, moved from `files`: U+212A lowers to `k`, U+0130 to `i` + U+0307) and `Tokens`
(`ConfirmationTokens`: `<expiry>.<nonce>.<mac>`, HMAC-SHA256 over `canonical([tool,
canonical(bound_args), plan_hash, expiry, nonce])` under a per-process random key; malformed,
mismatch, expired and reused refusals in that order and in Python's words; an expired nonce is
dropped; entries not after the clock are pruned on issue). `ord` ports `child_ord`/`join_ord`
(`box.py:183-207`) and `_plain_slot_ord`/`WriteScope` (`Scope.Check` takes any value, as a
caller's argument may be one). `journal` ports `_JsonlFile`, `Journal` and `AuditLog`
(`safety.py:156-311`); the state-dir checks stay in `safety` (S6), which now also holds the
`REASON_*` texts verbatim and the create-then-chmod rule (`PrivateDir`, `MakeDirs`, `IsDir`, moved
from `readtools`). `journal.Batches` is `n4_list_batches`' list and `RollbackTarget` /
`EarlierRollbacks` are the journal checks `_rollback_plan` makes before and after its station
check (S13 composes them).
`readtools` serves `n4_list_batches` (no session, journal only). The server still refuses
`--allow-writes` with the existing text: nothing in this slice writes.

**Files.** An append creates the state dir 0700 only when it is not a directory yet (missing
parents 0777 less the umask) and chmods only what it created; the file is opened
`O_WRONLY|O_APPEND|O_CREAT|O_EXCL` 0600, and on `EEXIST` reopened without `O_CREAT` and refused
with Python's `PermissionError` text when it is not private (never chmodded); one write of
`json.dumps(entry) + "\n"`. Reading follows Python exactly: a file that cannot be opened is empty,
a line that is not a JSON object is skipped, and `\r` and `\r\n` end lines. Batch ids are keyed
as Python dict keys: `1`, `true` and `1.0` are one batch, every JSON `NaN` is one object, and a
list or dict id cannot key a batch only in a phase that keys it; `component-intent` ops extend as
`list.extend` does (a string by characters, a dict by keys, a number cannot).

**Unreadable lines (issue #448, v0.9.4, both servers).** The journal is read as bytes, split as
`bytes.splitlines` splits (the line ends `readlines()` honours) and decoded line by line, so the
`TextIOWrapper` chunk quirk no longer applies to it. Since #466 (v0.9.5) the audit log uses the
same splitter (`_JsonlFile._lines`, Go `File.lines`; section 13), and the `readlines()` readers
(`entries()`, `File.Entries`, `pyjson.ReadLines`) are gone. A line is *unreadable* when it is not
UTF-8 (`UnicodeDecodeError`: its batch is unknown) or when, in a batch phase, its `batch_id` (or, in an `intent` or `result`, a truthy `rollback_of`)
cannot key a dict, or its component ops cannot extend a list (`TypeError`: its batch is known).
An unreadable line is left out of every view. `n4_list_batches` lists the readable batches as
before and, only when a line is unreadable, adds `incomplete` (a sentence saying the list is
incomplete) and `unreadable` (one `unreadable journal line N: <error class>` row per line; no
byte of the line is echoed). The rollback rule: `n4_rollback` of batch B is refused with
`journal line unreadable: batch B may be recorded on journal.jsonl line(s) N (reason), ...`,
before B is looked up, when an unreadable line has an unknown batch, or its `batch_id` is B, or its
`rollback_of` is B (a lost rollback of B could let B be rolled back twice); an unreadable line of
another known batch does not block B. `_BATCH_ID` is `[0-9a-f]{32}\Z` under `re.match` and the Go
pattern `^[0-9a-f]{32}$`: an id with a trailing newline is malformed in both. A torn line (not
JSON) is still skipped silently, as before: the server writes each line in one `write`.

**Proof.** The 49 `@go_twin` tests of `test_safety` (46) and `test_box.TestChildOrd` (3) have Go twins of
the same names (`token`, `ord`, `journal`; the `audit` markers now name `journal` and `token`,
where the code lives) plus `readtools.TestItIsReadOnlyAndWorksWithoutASession`. The new
`tests/test_list_batches.py` (5 tests) runs black-box under both implementations on a state dir
the Python server wrote (`tests/fixtures/journal/generate.py` → `python-written/`: a create and
its rollback, a remove and its rollback with component and relink records, an intent whose send
failed, a link whose result is `in_doubt`) and requires `tools_read.n4_list_batches`' reply byte
for byte, also after torn lines, a non-UTF-8 line and unhashable ids in the middle of the journal
(6 tests since #448). Under `MCP_N4_IMPL=go`: 627 tests, 448 skipped at S8; #448 (after S9)
adds one list test and three rollback tests of `test_tools_destructive`, which skip until S13
(their journal checks have Go tests of the same names in `journal`): 641 / 448. Differentials on CPython 3.15:
`token.TestTokensDifferentialPython` (key, clock and pending nonces injected into one
`ConfirmationTokens` instance, never by patching the package: mac, bound args, redaction,
consume verdict, pending set, issue expiry), `ord.TestScopeAndJoinDifferentialPython` (16
scopes × 36 ORDs, 13 parents × 15 names), `pyjson.TestReadLinesDifferentialPython` (144 files
around the chunk boundaries; retired with `pyjson.ReadLines` by #466) and `journal.TestJournalDifferentialPython` (the Python-written
dir and 18 edge journals, two with a bad line between two batches: views, `read`, `rollbacks_of`, `n4_list_batches` at three limits and
the journal verdict of the real `_rollback_plan`, every branch reached). Both in-doubt batches
the Python server wrote are refused by `RollbackTarget` with Python's text
(`journal.TestAPythonInDoubtBatchIsRefused`). 44 hand mutants of the refusal, binding, scope,
view, append and read branches all go RED (two through a panic, none through a build failure),
and a reversed batch order fails `test_list_batches` under Go.

**Known differences.** Expiry and nonce tests are Go-only twins (no in-process clock exists
across the process boundary). A journal line nested deeper than CPython's recursion limit makes
Python's `n4_list_batches` answer `internal error (RecursionError)`; Go parses it (the server
writes those lines itself). `File.Append` completes a short write, where `os.write` would leave a
partial line, and an `OSError` from opening the file is Go's error text; no S8 caller shows it
(the write tools of S10 will). `ChildOrd` and `JoinOrd` take strings: their non-string refusals
are Python-only paths. Observed in Python 0.9.3 and fixed in both servers by issue #448 (v0.9.4): one non-UTF-8 byte
or one unhashable batch id anywhere in `journal.jsonl` made `n4_list_batches` and every rollback
fail, and `_BATCH_ID.match` accepted an id with one trailing newline.

## 13. S9 `retro` and `n4_session_retro_draft`

**Packages.** `retro` ports `retro.py`: `refusals.go` (`_REFUSALS` and `_VERDICTS` in their
order, `_safe`; `RefusalClass` is the class lookup the derive loop and the tier twin share),
`derive.go` (`_derive`: refusal groups keyed by class in first-seen order with their first row's
advice, verdict and in-doubt evidence, BOX errors by op from the unfiltered journal, dangling
observations), `draft.go` (`_since_filter`, `_window`, `draft`) and `render.go` (`_summary`,
`_delta_table`, `cell`, `render`). `isodate.go` and `isotime.go` are `datetime.fromisoformat` as
CPython 3.15's `_datetimemodule.c` runs it on the UTF-8 bytes (the NUL past the end and an
embedded NUL stop a parse alike; every date form, the separator rules, a surrogate separator,
fractions, offsets, hour 24 rolling over, `0000-12-31T24:00` becoming year 1), behind
`_parse_ts`'s `strip()` and `Z` replacement; the `Z` branch of the C parser is left out, since no
`Z` reaches it. The template is embedded: `retro/retro.template.md` must equal
`mcp_n4/templates/retro.template.md` byte for byte. `journal.dictKey` moved to `pyjson.DictKey`:
the summary counts outcomes and verdicts as dict keys (`1`, `true` and `1.0` are one key, the
first one kept and sorted by its `str()`), and an audit `batch_id` is looked up the same way, so
a list or dict is the `TypeError` Python raises. `session.Context.StartedAt` is `ctx.started_at`
(`session.Now()` at start), and `readtools` serves `n4_session_retro_draft` (no session, the
server's state dir, `since` or the server start, `ctx.observations`). Nothing in this slice
writes.

**Proof.** `test_retro.TestServerTool` now runs black-box through `make_server` (its
`--allow-writes` case skipped under Go until S10), and the new `TestTheDraftIsPythonsDraft` (9) and
`TestDanglingReadsReachTheDraft` (1) run black-box under both implementations: the reply text
must be the one `tools_read.n4_session_retro_draft` computes in-process on the same state dir
(every refusal class with the tier refusal, verdicts, in-doubt and BOX errors with odd ops, the
S8 Python-written state dir, odd timestamps and windows, bad `since` values, odd and unhashable
summary values, the secret canary). With a stub handler 13 of them failed. Under
`MCP_N4_IMPL=go`: 637 tests, 445 skipped (627 / 448 before; 345 of them carry `@go_twin`). The
21 `retro` markers have Go twins in `retro/refusals_test.go`, `derive_test.go` and
`render_test.go` (`count_deltas` is tooling-only, so its twin counts rows with a test helper),
and `tiers.TestTheRetroClassifiesTierRefusals` is in `tiers/retro_test.go`;
`readtools.TestTheRetroDraftCoversThisSession` drives the tool through the dispatcher.
Differentials on CPython 3.15: `retro.TestParseTimestampDifferentialPython` (112 seeds and 40,000
mutants; 1,000,000 mutants were run once) requires `_parse_ts`'s instant or refusal for each, and
`retro.TestDraftDifferentialPython` requires `retro.draft`'s JSON, byte for byte, or its error,
for 27 session states (the Python-written fixture, a missing dir, every refusal class including
both tier refusals, all verdicts, in-doubt and BOX errors, dangling and BQL observations, windows,
`+N more` evidence, 64- and 65-character names, template placeholders in the station name, a
non-UTF-8 audit lines across CRLF, CR, end of file and the 8192-byte chunk boundary, unhashable
and non-iterable values). 42 hand mutants of the classification,
window, summary, render, parse and tool-wiring branches all go RED, none through a panic or a
build failure.

**Known differences.** None found. `since=""` is `None` for the Go `retro.Draft` API (the tool
maps an empty `since` to the server start, as Python's `or` does). An observation's `count` is
printed with `str()`, where Python calls `int()`: the server only records whole counts.

**Unreadable audit lines (issue #466, v0.9.5, both servers).** `audit.jsonl` is read as the
journal is since #448: bytes, split as `bytes.splitlines` splits, decoded line by line
(`AuditLog.scan`, Go `AuditLog.Scan`). A line that is not UTF-8 is left out of the draft; the
session summary then adds `- Incomplete: N audit line(s) are unreadable and left out of this
draft` and one `- unreadable audit line N: UnicodeDecodeError` row per line (numbered as
`readlines()` numbers; no byte of the line is echoed; never filtered by `since`, having no
timestamp). Before, one such byte anywhere failed the whole draft in both servers. Proof:
`test_retro.TestTheDraftIsPythonsDraft.test_an_unreadable_audit_line_costs_that_line_only`
(black-box), its Go twin `retro.TestAnUnreadableAuditLineCostsThatLineOnly`, and two new
differential states. Under `MCP_N4_IMPL=go`: 642 tests, 448 skipped (the black-box test runs).

## 14. S10 plan engine, `n4_set_slot` and `n4_invoke_action`

**Packages.** `plan` is the write pipeline (`tools_write.py:76-142, 1260-1500`): `values.go`
(`_coerce`, `_to_bson`, `_name`; `_NAME` and `_STATUS` keep Python's `$`, which also matches
before one final `\n`), `observe.go` (`_observe`, `_handle`, `float()` of a station value with
Python's `ValueError`/`TypeError`/`OverflowError` texts, the `KeyError` of a slot gone from a
read-back), `state.go` (`WriteState`: positive limits, `check_state_dir`, `WriteScope`, tokens,
journal, audit, the process budget, `uuid4().hex` batch ids), `engine.go` (`_make_handler`,
`_process`, `_record`; `Hashed` is the plan a token authorizes, `station` with the
`session_id` included, and its `plan_hash`) and `execute.go` (`_send`, `_execute`). `errors.go`
holds `_in_doubt` and the class/message pairs the audit log and read-back errors show.
`writeops` holds the tools as `plan.Impl` values (`setslot.go`, `invoke.go`) and `Register`.
The box client gains `Sync` (`BoxClient.sync`), and `float()` moves to `pyjson.ParseFloat`
(Unicode decimal digits read as Python reads them; `--load-wait` shares it).

**Wiring.** `cli.NewServer` is `Server(...)`: with `--allow-writes` it makes the write state
(`WriteState`'s start checks in Python's order; a refusal is `mcp_n4.server: <text>`, exit 2),
lists the seven write tools after the read tools (`jsonrpc.Server.EnableWrites`), checks the auth
latch before every write tool as Python's `_call` does, and serves `n4_set_slot` and
`n4_invoke_action`; the other five answered the "not implemented" text until S11 (section 15). `readtools.Tools.Audit` is
`ctx.write.audit`: BQL reads now leave their `read` audit line under `--allow-writes`.

**Proof.** With the two handlers stubbed, the 18 runnable tests of `TestSetSlot`,
`TestInvokeAction`, `TestTokenLayers` and `TestJournalAndAudit` failed under Go (RED); with the
port they pass, as do the `TestMode` listing tests. The new `tests/test_write_parity.py` (14
tests) runs every scenario on the in-process Python server and on the server under test, each on
its own recording FakeStation and state dir, and requires every reply line, every BOX request body
and the bytes of `journal.jsonl` and `audit.jsonl` to be equal after masking the station URL and
state dir, tokens, plan hashes, batch and session ids, expiry, timestamps and a read's elapsed
time; every Go dry run must also satisfy `plan_hash == sha256(safety.canonical(plan))`. The
scenarios cover each token refusal (missing, malformed, tampered, other args, another tool,
reused, an earlier session, a station that changed since the dry run), every value type and
status form (`"0\n"` is accepted, as Python's `$` accepts it), every plan refusal (including a
value past the float range, Python's `OverflowError`), read-back mismatch, `ValueError`,
`KeyError` and BOX failures, in-doubt sends (HTTP 500 and a station error frame on `syncTo` and
`invokeAction`), every action and its refusals, the budget across a reconnect, no scope, the tier
gate (B and undetected C) and its opt-in, redaction and the BQL `read` audit line, a loose state
file before the token is spent, an unwritable intent (token spent, nothing sent), and an
unwritable result or audit line (warnings). Go differentials on CPython 3.15:
`writeops.TestPlanAndReadbackDifferentialPython` runs 2,994 cases (each argument set on odd,
missing and malformed loads, read back from four or five loads) through the real `_IMPLS` plan and
read-back and requires the plan a token is bound to, its `plan_hash`, the targets and the
read-back, or Python's error class and message, byte for byte (2,142 refusals, 372 internal
errors, 129 read-back errors, 210 unverified, 115 mismatch, 26 verified);
`cli.TestWriteStartDifferentialPython` (10 `--allow-writes` starts on loose, file, missing and
loose-file state dirs and bad scopes) and `pyjson.TestParseFloatDifferentialPython`. The replay
corpus gains five write scenarios (58 in all). Go twins of the marked tests: `plan` (identity,
expired token, the plan's station, a `None` inverse, a loose state file, the four journal and
audit failures, the read-back BOX error, secrets; plus a write without a session, scrubbed
refusals and read-back errors, and the any-scope rule of a tool that names no ORD), `cli` (the
seven start tests), `session.TestAWriteDryRunIsRefusedWhilePaused` and
`bql.TestTheReadIsAuditedWithOutcomeRead`, which no longer skips. 73 hand mutants of the refusal,
binding, scope, budget, in-doubt, read-back, audit, BOX body, start and transport branches all go
RED, none through a build failure (three that failed to build were replaced by value changes, one
equivalent mutant by one that changes the result). The first run left three survivors, each now
closed: an executed write without a session (a parity line and a twin), a value type checked
before `status` (a corpus and a parity case), and the in-doubt wrap of an unexpected failure
after the sends, dead in this slice and removed. E01 (no session, executed) goes RED through a
nil dereference, both in the Go twin and as a crashed binary in the Python suite, and C01 (a
start error ignored) through a nil state. Under `MCP_N4_IMPL=go`: 656 tests, 420 skipped (642 /
448 before; the 14 new tests run under both implementations).

**Known differences.** No byte difference was found. Left to the slices whose tools reach them
(none is reachable from a one-op tool with a static inverse): `_hashed_inverse` and the
`link_input_values` preview, the `relinks` / `components` / `unlinked_inputs` /
`outgoing_links_broken` / `snapshot_truncated` plan keys and intent keys, the partial in-doubt
record of a multi-op batch (`_in_doubt_partial`), an inverse that cannot be built and the
`recover` hook, `_scrub_gap_errors`, `planned.data["write"]` of the save, and the in-doubt wrap of
an unexpected failure after the sends (S15's "batch id kept"). A Python exception the port does
not model (a `RecursionError`, a `MemoryError`) has no Go counterpart, and a Go panic in a plan
or read-back ended the process where Python's `except Exception` would record it (closed in S11,
section 15); no input reaches either. `box.PyEqual` still treats two infinities as unequal, which no verdict can observe (a
request is always finite). An `OSError` from appending a state file keeps Go's text, which no
reply shows. Expiry tests stay Go-only. Python bugs: none found.

## 15. S11 `n4_create_component`, `n4_create_link` and the engine's later branches

**Packages.** `writeops/create.go` is `_create_component_plan`, `_wire_sheet` (Python ints of any
size; a bool or float is refused; `w` defaults to 8), `_create_component_inverse` (the station's
`nn`, else the requested name) and `_create_component_readback` (Python's errors for an odd reply:
`KeyError: 0`, "object is not subscriptable", an unhashable name, the first code point of a
string). `writeops/link.go` is `_link_plan`, `_link_result`, `_link_inverse`, `_link_recover` and
`_link_readback` (only the part after the last `|` of a source ORD is compared). `plan.Impl` gains
`Inverse` (nil: the planned one) and `Recover`, and `Readback` receives the inverse. `execute`
ports the inverse that cannot be built (verdict `unverified`, `in_doubt` on the reply and the
result record, the station's words or the class name), the recover hook, and Python's `except
Exception` around inverse, recover and read-back: a failure there discards whatever the tool
returned with it. `plan/prior.go` ports `_hashed_inverse`, the `link_input_values` preview of a dry
run and the plan keys: `Planned.Keys` holds what Python keeps in the plan's data, its truthy
`relinks`, `unlinked_inputs`, `components`, `outgoing_links_broken` and `snapshot_truncated` join
the hashed plan and its `rollback_of`, `relinks` and `components` the intent, in Python's order; no
S11 tool sets them, S12-S13 will. `jsonrpc.Recover` turns a handler panic into `internal error
(Exception)` as Python's catch-all answers an unexpected exception, and the engine recovers too: a
panic before the intent is an audited error, a panicking send leaves the batch in-doubt, a panic in
inverse, recover or read-back is a result. `tiers.ReasonTier` is now the only copy of `REASON_TIER`
(the `safety` duplicate is gone).

**Proof.** With both handlers stubbed, 45 runnable tests failed under Go (RED):
`TestCreateComponent`, `TestCreateLink`, `TestCreateLinkRejected`, `TestLinkReplies`, `TestScope`,
`TestTokenLayers`, `TestTokenStationBinding`, `TestBudget`, `TestJournalAndAudit`, the tier write
tests, the rollback tests that first create a component and the new parity tests; with the port
they pass. `TestCreateLinkRejected` and `TestLinkReplies` were `go_twin` internals only to read the
journal through `srv.ctx`: they now read the state file the server wrote (`safety.Journal`) and run
black-box on both servers; `TestPlannedDataGuards` patches the Python table and became
`go_twin("plan", "injection")` with `plan.TestAPlanWhoseDataIsNotADictStillExecutes`.
`test_write_parity` gains four scenarios (18 in all): every create and link refusal, renames, token
reuse, read-back mismatch, 18 odd `syncTo` replies, 14 refused, ambiguous and malformed
`checkLinks` replies, a lying or prefixed station, recover finding one, none or two links or
failing to load, failed read-back loads and failed sends. Go differentials on CPython 3.15:
`writeops.TestInverseRecoverAndReadbackDifferentialPython` runs 1,648 cases through the real
`_IMPLS` plan, inverse, recover and read-back and requires the plan, its `plan_hash`, the targets,
the inverse and the read-back or recovery, or Python's error class and message, byte for byte (49
refusals, 7 internal errors, 87 verified, 377 mismatch, 96 failed, 600 read-back errors, 27
recovered, 153 recovered none, 252 recover errors);
`plan.TestHashedInverseAndLinkInputValuesDifferentialPython` covers `_hashed_inverse` and
`_link_input_values` on odd inverses;
`jsonrpc.TestAPanickingToolAnswersAsPythonsCatchAllAndTheServerLivesOn` compares a panicking
handler with a Python tool that raises. Engine twins cover the plan and intent keys, a `prior` that
moves between dry run and execution (the token still executes, the journal keeps the new value), an
unbuildable preview and every panic branch. 61 hand mutants of the create and link refusals,
inverse, recover and read-back branches, the scope binding, the engine's in-doubt and recover
branches, the preview, hashed inverse and keys, the panic recovers and the tier reason all go RED,
none through a build failure. The first run left four survivors, each closed: a source ORD with two
`|` and a non-link child with matching slots (corpus cases), an inverse or recover error that
returned values (a twin), and an equivalent `list()` that was removed. Under `MCP_N4_IMPL=go`: 660
tests, 380 skipped (656 / 420 before).

**Known differences.** No byte difference was found. The plan's private link list is a Go set,
where Python keeps it sorted; nothing shows it. Left to S12-S15: the multi-op partial in-doubt
record (`_in_doubt_partial`; both S11 tools send one op), `_scrub_gap_errors`,
`planned.data["write"]` of the save and the relink and component runs of a rollback. A Go panic is
reported as `Exception`, whatever Python class the same bug would raise. Python bugs: none found.

## 16. S12 `n4_remove_component`: snapshot, relinks, truncation and the outgoing scan

**Packages.** `writeops/snapshot.go` is `_snapshot` over `_prune` (`kidsOf` cuts the loaded tree
at depth 3 instead of copying it): type, value and children as data, no handles; `"t"` omitted
for typeless frozen children; the runtime output `out` of a Status* slot dropped unless a link
drives it; a `baja:Status` value, or a `status` child of a Status* parent, cut to the bits before
`;` (of Python's `str()` of any value). Links are recorded apart with the holder's path, and an
input a link drives inside the subtree stays out of the body and rides on its link as `prior`.
Handles map to paths through `pyjson.DictKey` (1, 1.0 and True collide; a list or dict is
Python's TypeError, a set element's for a driven slot name). `writeops/remove.go` is
`_remove_plan` and `_remove_readback`: the scope covers `parent_ord` and, when given,
`link_scan_ord`; the child is loaded one level deeper than the snapshot keeps; the inverse is the
re-add, one `relink` per link whose source handle (after the last `|`, only behind `h:`) is in the
subtree and one `unlinked_input` per other link, with their notes; `snapshot_truncated` keeps
the first 20 sorted paths below the cut. `writeops/scan.go` is `_has_outputs`, `_outgoing_links`
and `_scan_outgoing` (depth 6, `outgoing_links_broken`), and `writeops/pysort.go` is `sorted()`
for decoded values: CPython's list sort for fewer than 64 items (`count_run`, then binary
insertion), with tuple, string, number (NaN and infinity equality as json.loads values) and list
ordering and its TypeError naming the operands CPython compares. The engine echoes a truthy
`snapshot_truncated` on the execution, as `_execute` does, and `_send`'s cache clear after a sent
removal (`ClearHandles`) is now reached and tested.

**Proof.** With the handler stubbed, 20 runnable tests failed under Go (RED): `TestRemoveComponent`,
`TestOutgoingLinks`, `TestSnapshotTruncation` and the rollback tests that remove first; with the
port they pass (the rollback ones then skip, unported). Three remove tests that were go_twin only
to read the journal through `srv.ctx` or patch the FakeStation
(`test_execute_removes_verifies_and_journals_the_snapshot_as_the_inverse`,
`test_read_back_reports_a_child_that_is_still_there`,
`test_a_link_driven_value_that_changes_after_the_dry_run_keeps_the_token_valid`) now run black-box
on both servers. `test_write_parity` gains two scenarios (20 in all): every refusal, scans that
find none or some links, relinks, unlinked and link-driven inputs, a prior that moves after the dry
run, truncation at 1, 20 and 21 paths, token reuse, read-back mismatch, in-doubt sends (HTTP 500
and a station error frame) and a failed read-back load. Go differentials on CPython 3.15:
`writeops.TestRemovePlanAndReadbackDifferentialPython` runs 461 cases (32 subtrees with odd
handles, types, values, links, driven inputs, runtime outputs and depths below the cut, each
without a scan and against 13 scan roots, plus refusals and unjoinable parents) through the real
`_IMPLS` plan as `_process` hashes it and the read-back, and requires the hashed plan with its
keys, its `plan_hash`, the `link_input_values` preview, the full journaled inverse, the targets
and the read-back, or Python's error class and message, byte for byte (5 refusals, 165 internal
errors: 86 TypeError, 54 AttributeError, 21 ValueError, 4 BoxError; 100 verified, 98 mismatch, 93
read-back errors; 56 with relinks, 66 with unlinked inputs, 69 with priors, 78 truncated, 55 with
outgoing links); `writeops.TestPySortDifferentialPython` compares 5,324 key lists with `sorted()`
(2,805 TypeErrors). Go twins: `writeops.TestSnapshotOmitsTForTypelessNodesAndRecognizesStatusByName`;
`plan.TestARemovalClearsEveryCachedHandle` (a sent removal clears, a failed one or any other op
does not) and `plan.TestRemoveComponentIsServedAndClearsTheHandles`. 65 hand mutants of the
refusals, scope binding, snapshot, prior, relink and unlinked classification, truncation, scan,
sort, read-back, the execution echo, the cache clear and the registration all go RED, none
through a build failure (61 in the Go tests, 4 only in the black-box suite under the binary).
The first run left six survivors, each closed: a source ORD `hxs1`, a scan link from `h:`, a root
node with its own `n`, a holder-level link beside a nested one to an input of the same name and
parent ORDs that load but cannot be joined (corpus cases), and the registration itself (a plan
test: without it every black-box test only skipped as unported); a source path set after the
relink entry was equivalent and four mutants that failed to build became value changes. Under
`MCP_N4_IMPL=go`: 662 tests, 365 skipped (660 / 380 before; the two new tests run under both).

**Known differences.** No byte difference was found. `pySort` follows CPython's comparisons for
fewer than 64 links; a scan finding 64 or more outgoing links whose keys cannot all be ordered (or
hold NaN) may raise with other operands or order them otherwise, as it inserts where CPython
merges. Left to S13-S15: the rollback's relink and component runs (`_run_relinks`,
`_run_components`, `relinks_skipped`, `frozen_children_not_restored`), `_scrub_gap_errors`
(only a rollback read-back has gaps), the multi-op partial in-doubt record (`_in_doubt_partial`;
a remove sends one op) and `planned.data["write"]` of the save. Python bugs: none found.
