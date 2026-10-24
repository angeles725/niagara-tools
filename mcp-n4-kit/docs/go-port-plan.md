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
| `files` / `px` | `files.py:1-233` (Windows alias refusals at `station_path` 38-86), `tools_read.py:654-789` |
| `token` | `safety.py:47-113` (canonical, `bound_args`, redact, `ConfirmationTokens`) |
| `audit` / `journal` | `safety.py:156-311` (state-dir checks, `_JsonlFile`, `Journal` views, `AuditLog`) |
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
refuses `--allow-writes` with the existing "not in this version" refusal.

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
directory, the whole suite passes under `MCP_N4_IMPL=go` (590 tests, 339 skipped): every unmarked
server test is black-box over stdio.

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
(start refusals, `TestRobustness` id handling, `TestServerTool`) can move to `make_server` in the
slice that ports their code instead of getting a Go twin.

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
| `test_live_smoke.py` | 30 | 0 | 0 | 0 | 0 | 0 | 30 |
| `test_methodology_names.py` | 1 | 0 | 0 | 0 | 0 | 0 | 1 |
| `test_mutation_check.py` | 7 | 0 | 0 | 0 | 0 | 0 | 7 |
| `test_new_retro.py` | 17 | 0 | 0 | 0 | 0 | 0 | 17 |
| `test_retro.py` | 25 | 20 | 0 | 1 | 4 | 25 | 0 |
| `test_safety.py` | 46 | 33 | 0 | 13 | 0 | 46 | 0 |
| `test_server.py` | 101 | 4 | 14 | 5 | 7 | 30 | 71 |
| `test_tiers.py` | 25 | 5 | 1 | 1 | 2 | 9 | 16 |
| `test_tools_destructive.py` | 98 | 4 | 24 | 28 | 2 | 58 | 40 |
| `test_tools_write.py` | 76 | 2 | 15 | 4 | 7 | 28 | 48 |
| **Total** | **622** | **166** | **56** | **105** | **22** | **349** | **273** |
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
version of the Go server`, and `--allow-writes` is refused at start with
`mcp_n4.server: --allow-writes is not implemented in this version of the Go server` (exit 2)
until S10. Under `MCP_N4_IMPL=go` the harness skips a test that meets exactly that text
(`go_server.skip_if_unported`), so these skips shrink by themselves as slices land and are not
in the inventory above. Known command-line differences: `int()`/`float()` flag values take
ASCII digits only (Python also takes other Unicode decimal digits), integers past int64
saturate, and Python's `str.isprintable` is approximated by `unicode.IsPrint` in `%r` text.

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
