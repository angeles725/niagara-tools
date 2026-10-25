"""The session retro draft (n4_session_retro_draft): evidence-backed candidates from the audit and
the journal, black-box on the Go server."""
import json
import os
import shutil
import sys
import tempfile
import types
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
import oracle  # noqa: E402
import state_files  # noqa: E402
import test_server  # noqa: E402

CANARY = "pw-Zk93-unique-canary"
B1, B2, B3, B4, B5 = ("%032x" % n for n in range(1, 6))
FIXTURE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures", "journal",
                       "python-written")


def audit_row(ts, tool, outcome, reason, batch_id=None, args=None):
    return {"ts": ts, "tool": tool, "batch_id": batch_id, "dry_run": outcome == "planned",
            "outcome": outcome, "reason": reason,
            "args_redacted": state_files.redact_args(args or {})}


def journal_batch(bid, ts, verdict, tool="n4_set_slot", in_doubt=False, with_result=True,
                  ops=({"nm": "s"},)):
    rows = [{"batch_id": bid, "ts": ts, "tool": tool, "ops": list(ops), "phase": "intent",
             "station_name": "LLM", "inverse_plan": [], "targets": {}}]
    if with_result:
        res = {"batch_id": bid, "ts": ts, "phase": "result", "accepted": None,
               "inverse": [], "verdict": verdict}
        if in_doubt:
            res["in_doubt"] = True
        rows.append(res)
    return rows


REFUSALS = [
    ("confirmation_token is missing or malformed: run a dry run first", "token_missing"),
    ("confirmation_token does not match this tool, these arguments", "token_mismatch"),
    ("confirmation_token expired: run the dry run again", "token_expired"),
    ("confirmation_token already used: run the dry run again", "token_reused"),
    ("station identity not verified: reconnect with n4_connect", "identity"),
    ("ord 'station:|slot:/X' is outside the write scope", "scope"),
    ("write budget exhausted: 2 writes already executed in this session", "budget"),
    ("refusing to write 'out' on its own: Status slots are written whole", "partial_status"),
    ("action 'foo' is not allowed in this version (allowed: set)", "unsupported"),
    ("type must look like module:Type, got 'x'", "unsupported"),
]


class Fixture(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = os.path.join(tmp.name, "state")
        self.audit, self.journal = state_files.AuditLog(self.dir), state_files.Journal(self.dir)

    def add_audit(self, *rows):
        for r in rows:
            self.audit.append(r)

    def add_journal(self, rows):
        for r in rows:
            self.journal.append(r)


class TestServerTool(unittest.TestCase):
    """Black-box: the tool as the server lists and answers it."""

    def server(self, **kw):
        srv = go_server.make_server(**kw)
        self.addCleanup(go_server.close_server, srv)
        return srv

    def rpc(self, srv, method, params=None):
        return srv.dispatch({"jsonrpc": "2.0", "id": 1, "method": method, "params": params})

    def test_tool_is_listed_read_only_in_both_modes(self):
        for allow in (False, True):
            with tempfile.TemporaryDirectory() as tmp:
                srv = self.server(allow_writes=allow, state_dir=os.path.join(tmp, "s"))
                tools = {t["name"]: t for t in
                         self.rpc(srv, "tools/list")["result"]["tools"]}
                self.assertIn("n4_session_retro_draft", tools)
                self.assertEqual(tools["n4_session_retro_draft"]["annotations"],
                                 {"readOnlyHint": True, "openWorldHint": False})

    def test_tool_returns_markdown_and_candidates_for_the_server_state_dir(self):
        with tempfile.TemporaryDirectory() as tmp:
            state = os.path.join(tmp, "s")
            for r in journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"):
                state_files.Journal(state).append(r)
            srv = self.server(state_dir=state)
            res = self.rpc(srv, "tools/call", {"name": "n4_session_retro_draft",
                                               "arguments": {"since": "2000-01-01T00:00:00Z"}}
                           )["result"]
            out = res["structuredContent"]
            self.assertFalse(res.get("isError", False))
            self.assertEqual([c["key"] for c in out["candidates"]], ["verdict:mismatch"])
            self.assertIn("## Proposed kit deltas", out["markdown"])
            res = self.rpc(srv, "tools/call", {"name": "n4_session_retro_draft",
                                               "arguments": {"since": "2026-10-02T00:00:00Z"}})
            self.assertEqual(res["result"]["structuredContent"]["candidates"], [])

    def test_since_defaults_to_the_server_start_so_a_draft_covers_the_session(self):
        with tempfile.TemporaryDirectory() as tmp:
            state = os.path.join(tmp, "s")
            for r in journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"):
                state_files.Journal(state).append(r)  # written before this server started
            srv = self.server(state_dir=state)
            call = lambda args: self.rpc(srv, "tools/call", {
                "name": "n4_session_retro_draft", "arguments": args})["result"]["structuredContent"]
            self.assertEqual(call({})["candidates"], [])
            self.assertEqual(call({"since": ""})["candidates"], [])  # empty is the default too
            self.assertEqual(len(call({"since": "2000-01-01T00:00:00Z"})["candidates"]), 1)

    def test_bad_since_is_a_tool_error(self):
        with tempfile.TemporaryDirectory() as tmp:
            srv = self.server(state_dir=os.path.join(tmp, "s"))
            res = self.rpc(srv, "tools/call", {"name": "n4_session_retro_draft",
                                               "arguments": {"since": "nope"}})["result"]
            self.assertTrue(res["isError"])
            self.assertEqual(res["content"][0]["text"], "not an ISO-8601 timestamp: 'nope'")


def tier_refusal(station="LLM"):
    """The tier-C refusal of a station with no version, as the Go server words it
    (test_tiers.TestTierC compares it with a real refusal, so it cannot drift silently)."""
    return ("version tier refuses writes: station %s is tier C (version unknown (no oBIX "
            "productVersion at /obix/about/)): writes need the probe sequence first (METHODOLOGY "
            "section 5): `reg.loadContract` and `loadRoot` succeed and one harmless scratch write "
            "(create, read back, remove in a scratch folder) is verified on this build. The server "
            "has no probe tool. Routes: (1) dry runs stay allowed here, so the plan can be "
            "reviewed now; (2) make the change in Workbench; (3) run that probe (e.g. in "
            "Workbench or a PoC script), then restart the server with `--allow-tier-c %s`." % (station, station))


#: Refusal reasons of every class, worded as the raise sites word them.
EVERY_REFUSAL = [r for r, _ in REFUSALS] + [
    "write budget too small: the rollback needs 3 writes, 1 left",
    "not connected: call n4_connect first",
    "value_type must be one of: bool, numeric, string",
    "no --write-scope configured: every write is refused",
    "ord '/etc' is not a plain station ORD",
    tier_refusal(),
    "something no class knows",
]


class TestTheDraftIsPythonsDraft(Fixture):
    """The tool's reply, byte for byte, is the one the Python server's
    tools_read.n4_session_retro_draft computed in-process on the same state dir, frozen in
    fixtures/oracles (S17)."""

    WHOLE = "1970-01-01T00:00:00Z"

    def setUp(self):
        super().setUp()
        self.srv = go_server.make_server(state_dir=self.dir)
        self.addCleanup(go_server.close_server, self.srv)
        self.frozen = oracle.Frozen(self, "retro_draft", aliases=[
            (self.dir, "<state>"), (oracle.today_utc(), oracle.TODAY)])

    def reply(self, **args):
        msg = {"jsonrpc": "2.0", "id": 1, "method": "tools/call",
               "params": {"name": "n4_session_retro_draft", "arguments": args}}
        return self.srv.dispatch(msg)["result"]

    def python(self, args):
        """What the Python server answered (recording only): the draft, or the error text."""
        from mcp_n4 import tools_read
        ctx = types.SimpleNamespace(session=None, state_dir=self.dir, observations=[],
                                    started_at=None)  # every call here passes since
        try:
            return {"draft": tools_read.n4_session_retro_draft(ctx, args)}
        except ValueError as exc:
            return {"error": str(exc)}
        except Exception as exc:  # noqa: BLE001 - the server names the type only
            return {"error": "internal error (%s)" % type(exc).__name__}

    def expected(self, **args):
        return self.frozen(args, lambda: self.python(args))

    def assertSameDraft(self, **args):
        result = self.reply(**args)
        want = self.expected(**args).get("draft")
        self.assertIsNotNone(want, "Python refused %r" % (args,))
        self.assertFalse(result.get("isError", False), result)
        self.assertEqual(result["content"][0]["text"], json.dumps(want))
        self.assertEqual(result["structuredContent"], want)
        return want

    def assertSameError(self, **args):
        result = self.reply(**args)
        text = self.expected(**args).get("error")
        if text is None:
            self.fail("Python drafted %r" % (args,))
        self.assertTrue(result["isError"], result)
        self.assertEqual(result["content"][0]["text"], text)

    def test_every_refusal_class_and_the_tier_refusal(self):
        for i, reason in enumerate(EVERY_REFUSAL * 2):
            self.add_audit(audit_row("2026-10-01T10:%02d:00+00:00" % i, "n4_set_slot",
                                     "refused", reason))
        out = self.assertSameDraft(since=self.WHOLE)
        self.assertIn("refusal:tier", [c["key"] for c in out["candidates"]])
        self.assertIn("; +1 more", out["markdown"])  # more than five evidence items

    def test_verdicts_in_doubt_and_box_errors(self):
        for bid, verdict in ((B1, "mismatch"), (B2, "failed"), (B3, "partial"),
                             (B4, "unverified"), (B5, "verified")):
            self.add_journal(journal_batch(bid, "2026-10-01T11:00:00+00:00", verdict))
        self.add_journal(journal_batch("d1", "2026-10-01T11:01:00Z", None, with_result=False,
                                       ops=({"nm": "z"}, {"nm": "a"}, {"nm": "a"}, "odd",
                                            {"nm": "bad name"})))
        self.add_journal(journal_batch("d2", "2026-10-01T11:02:00", "unverified",
                                       in_doubt=True))
        self.add_journal(journal_batch("d3", "2026-10-01T11:03:00+00:00", None,
                                       with_result=False, ops=()))
        failed = "station call failed after the intent was journaled"
        self.add_audit(audit_row("2026-10-01T11:01:01+00:00", "n4_create_component", "error",
                                 failed, batch_id="d1"),
                       audit_row("2026-10-01T11:03:01+00:00", "n4_remove", "error", failed,
                                 batch_id="d3"),
                       audit_row("2026-10-01T11:04:00+00:00", "n4_link", "error", failed,
                                 batch_id="never-journaled"),
                       audit_row("2026-10-01T11:05:00+00:00", "n4_set_slot", "error",
                                 "BoxError: HTTP 500"),
                       audit_row("2026-10-01T11:06:00+00:00", "n4_set_slot", "error",
                                 "ValueError: no"),
                       audit_row("2026-10-01T11:07:00+00:00", "n4 bad|tool", "error",
                                 "BoxError: x", batch_id=""))
        self.assertSameDraft(since=self.WHOLE)

    def test_the_python_written_state_dir(self):
        shutil.rmtree(self.dir, ignore_errors=True)
        shutil.copytree(FIXTURE, self.dir)
        out = self.assertSameDraft(since=self.WHOLE)
        self.assertIn("in_doubt", [c["key"] for c in out["candidates"]])

    def test_no_friction_and_no_state_dir_give_the_honesty_line(self):
        self.assertSameDraft(since=self.WHOLE)  # the dir does not exist yet
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_set_slot", "executed",
                                 "verdict=verified", batch_id=B1))
        self.add_journal(journal_batch(B1, "2026-10-01T10:00:00+00:00", "verified"))
        out = self.assertSameDraft(since=self.WHOLE)
        self.assertEqual(out["candidates"], [])
        default = self.reply()["structuredContent"]  # the window starts at the server start
        self.assertEqual(default["candidates"], [])
        self.assertIn("- Window: since 20", default["markdown"])

    def test_the_window_parses_timestamps_as_fromisoformat_does(self):
        stamps = ["2026-10-01T10:00:00+00:00", "2026-10-01T10:00:00Z", "2026-10-01 10:00",
                  "2026-10-01T09:30:00-01:00", "20261001T113000", "2026-W40-4T10:00",
                  "2026W404T1000", "2026-10-01T10:00:00.5+00:00", "2026-10-01T24:00",
                  " 2026-10-01T10:00:00 ", "2026-10-01T10:00:00+01", "2026-10-01T10:00Zjunk",
                  "not a time", "", None, 7, "2026-13-01T00:00", "2026-10-01\u00e9"
                  "10:00", "2026-10-01T10:00:00,123456789"]
        for i, ts in enumerate(stamps):
            self.add_audit(dict(audit_row(None, "n4_set_slot", "refused",
                                          "write budget exhausted: %d" % i), ts=ts))
        for since in ("2026-10-01T10:00:00Z", "2026-10-01T10:00", "2026-10-01T11:00:00+01:00",
                      "2026-10-01T10:30Z", " 2026-W40-4 ", "2026-10-02", "20261001"):
            self.assertSameDraft(since=since)

    def test_bad_since_values_are_the_same_value_error(self):
        for since in ("yesterday", "2026-10-01T25:00", "2026-02-30", "2026-10-01T10:00+24:00",
                      "\u0000" * 9, "2026-10-01T10:00:00Z\u00e9", "2026-10-01T10:61"):
            self.assertSameError(since=since)

    def test_odd_audit_values_are_summarized_as_python_summarizes_them(self):
        for outcome in ("refused", None, 1, True, 1.0, "a|b", "planned", "x" * 65, 2.5,
                        False, 0, "None"):
            self.add_audit(dict(audit_row("2026-10-01T10:00:00+00:00", 5, "refused",
                                          "write budget exhausted"), outcome=outcome))
        self.add_journal([{"batch_id": "v", "ts": "2026-10-01T10:00:00+00:00", "tool": "t",
                           "phase": "intent", "ops": []},
                          {"batch_id": "v", "phase": "result", "verdict": 3}])
        self.assertSameDraft(since=self.WHOLE)

    def test_values_python_cannot_hash_or_iterate_are_the_same_internal_error(self):
        self.add_audit(dict(audit_row("2026-10-01T10:00:00+00:00", "n4_set_slot", "planned",
                                      ""), outcome=["x"]))
        self.assertSameError(since=self.WHOLE)
        os.remove(os.path.join(self.dir, "audit.jsonl"))
        self.add_journal([{"batch_id": "n", "ts": "2026-10-01T10:00:00+00:00", "phase": "intent",
                           "ops": 5}])
        self.add_audit(audit_row("2026-10-01T10:00:01+00:00", "n4_set_slot", "error",
                                 "station call failed", batch_id="n"))
        self.assertSameError(since=self.WHOLE)
        os.remove(os.path.join(self.dir, "audit.jsonl"))
        self.add_audit(dict(audit_row("2026-10-01T10:00:02+00:00", "n4_set_slot", "error",
                                      "x"), batch_id={"a": 1}))
        self.assertSameError(since=self.WHOLE)

    def test_an_unreadable_audit_line_costs_that_line_only(self):
        """Issue #466: a line that is not UTF-8 is left out and named; the rest is drafted.

        Bad lines end in CRLF and in CR, and one is cut at the 8192-byte read chunk boundary
        (a lead byte at 8191, an ASCII byte at 8192), where `readlines()` failed the file."""
        row = lambda i: json.dumps(audit_row("2026-10-01T10:%02d:00+00:00" % i, "n4_set_slot",
                                             "refused", "write budget exhausted: %d" % i))
        head = (row(0) + "\r\n").encode() + b"\xff bad\r\n" + (row(1) + "\r").encode() + b"\xfe\r"
        cut = b'{"x": "' + b"a" * (8191 - len(head) - 7) + b"\xc3(" + b'"}\n'
        self.assertEqual(len(head) + cut.index(b"\xc3"), 8191)
        self.add_audit(audit_row(self.WHOLE, "t", "refused", "x"))  # creates it private
        with open(os.path.join(self.dir, "audit.jsonl"), "wb") as fh:
            fh.write(head + cut + (row(2) + "\n").encode())
        out = self.assertSameDraft(since=self.WHOLE)
        self.assertIn("- Audit entries: 3 (refused 3)", out["markdown"])
        self.assertEqual([c["count"] for c in out["candidates"]], [3])
        for n in (2, 4, 5):
            self.assertIn("\n- unreadable audit line %d: UnicodeDecodeError" % n, out["markdown"])
        self.assertIn("- Incomplete: 3 audit line(s) are unreadable and left out of this draft",
                      out["markdown"])
        self.assertNotIn("unreadable audit line 1:", out["markdown"])

    def test_no_secret_reaches_the_reply(self):
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_connect", "error",
                                 "BoxError: login failed for %s" % CANARY,
                                 args={"password": CANARY}),
                       audit_row("2026-10-01T10:00:01+00:00", "n4_set_slot", "refused",
                                 "ord 'station:|slot:/%s' is outside the write scope" % CANARY,
                                 args={"credential": CANARY}))
        out = self.assertSameDraft(since=self.WHOLE)
        self.assertNotIn(CANARY, json.dumps(out))


class TestDanglingReadsReachTheDraft(test_server.StationTestCase):
    """A dangling report is an in-memory observation: only the server process knows it."""

    def test_a_dangling_report_becomes_a_candidate_of_this_session(self):
        self.add_with_out("A")
        self.assertEqual(len(self.ok("n4_find_dangling_outputs",
                                     ord="station:|slot:/Folder")["dangling"]), 1)
        self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        out = self.ok("n4_session_retro_draft")
        self.assertEqual(out["station"], "FakeStation")
        self.assertEqual([c["key"] for c in out["candidates"]], ["dangling"])
        self.assertEqual(out["candidates"][0]["count"], 2)
        for item in out["candidates"][0]["evidence"]:
            self.assertRegex(item, r"^audit-less read \d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\+00:00 "
                                   r"n4_find_dangling_outputs found 1$")
        self.assertIn("# Retro — FakeStation · %s · " % out["date"], out["markdown"])


if __name__ == "__main__":
    unittest.main()
