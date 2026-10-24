"""Tests for mcp_n4.retro: evidence-backed session retro drafts from audit + journal."""
import json
import os
import shutil
import sys
import tempfile
import types
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
import test_server  # noqa: E402
from mcp_n4 import retro, safety, tools_read  # noqa: E402

CANARY = "pw-Zk93-unique-canary"
B1, B2, B3, B4, B5 = ("%032x" % n for n in range(1, 6))
FIXTURE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures", "journal",
                       "python-written")


def audit_row(ts, tool, outcome, reason, batch_id=None, args=None):
    return {"ts": ts, "tool": tool, "batch_id": batch_id, "dry_run": outcome == "planned",
            "outcome": outcome, "reason": reason,
            "args_redacted": safety.redact_args(args or {})}


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
        self.audit, self.journal = safety.AuditLog(self.dir), safety.Journal(self.dir)

    def add_audit(self, *rows):
        for r in rows:
            self.audit.append(r)

    def add_journal(self, rows):
        for r in rows:
            self.journal.append(r)

    def draft(self, **kw):
        return retro.draft(self.dir, station="LLM", date="2026-10-01", **kw)


class TestDerive(Fixture):
    def keys(self, out):
        return {c["key"]: c for c in out["candidates"]}

    @go_server.go_twin("retro", "unit")
    def test_each_refusal_reason_becomes_a_candidate_citing_audit_timestamps(self):
        for i, (reason, _) in enumerate(REFUSALS):
            self.add_audit(audit_row("2026-10-01T10:00:%02d+00:00" % i, "n4_set_slot",
                                     "refused", reason))
        cands = self.keys(self.draft())
        for _, key in REFUSALS:
            self.assertIn("refusal:" + key, cands)
        scope = cands["refusal:scope"]
        self.assertTrue(any("2026-10-01T10:00:05+00:00" in e for e in scope["evidence"]))
        self.assertEqual(scope["count"], 1)

    @go_server.go_twin("retro", "unit")
    def test_same_reason_groups_into_one_candidate_with_a_count(self):
        for i in range(3):
            self.add_audit(audit_row("2026-10-01T10:00:0%d+00:00" % i, "n4_create_link",
                                     "refused", "confirmation_token expired: run again"))
        cand = self.keys(self.draft())["refusal:token_expired"]
        self.assertEqual(cand["count"], 3)
        self.assertEqual(len(cand["evidence"]), 3)

    @go_server.go_twin("retro", "unit")
    def test_readback_verdicts_mismatch_failed_unverified_cite_batch_ids(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"))
        self.add_journal(journal_batch(B2, "2026-10-01T11:01:00+00:00", "failed"))
        self.add_journal(journal_batch(B3, "2026-10-01T11:02:00+00:00", "unverified"))
        self.add_journal(journal_batch(B4, "2026-10-01T11:03:00+00:00", "verified"))
        cands = self.keys(self.draft())
        self.assertIn(B1, " ".join(cands["verdict:mismatch"]["evidence"]))
        self.assertIn(B2, " ".join(cands["verdict:failed"]["evidence"]))
        self.assertIn(B3, " ".join(cands["verdict:unverified"]["evidence"]))
        self.assertNotIn(B4, json.dumps(cands))

    @go_server.go_twin("retro", "unit")
    def test_in_doubt_batches_are_flagged_whether_result_missing_or_flagged(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", None, with_result=False))
        self.add_journal(journal_batch(B2, "2026-10-01T11:01:00+00:00", "unverified",
                                       in_doubt=True))
        cand = self.keys(self.draft())["in_doubt"]
        self.assertEqual(cand["count"], 2)
        self.assertIn(B1, " ".join(cand["evidence"]))
        self.assertIn(B2, " ".join(cand["evidence"]))
        self.assertEqual(cand["priority"], "HIGH")

    @go_server.go_twin("retro", "unit")
    def test_box_errors_are_grouped_by_op_name_from_the_journal(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", None, with_result=False,
                                       ops=({"nm": "a"}, {"nm": "s"})))
        self.add_audit(audit_row("2026-10-01T11:00:01+00:00", "n4_create_component", "error",
                                 "station call failed after the intent was journaled: batch "
                                 "%s is in-doubt" % B1, batch_id=B1),
                       audit_row("2026-10-01T11:05:00+00:00", "n4_set_slot", "error",
                                 "BoxError: HTTP 500"))
        cands = self.keys(self.draft())
        self.assertIn("box_error:a", cands)
        self.assertIn(B1, " ".join(cands["box_error:a"]["evidence"]))
        self.assertIn("box_error:n4_set_slot", cands)

    @go_server.go_twin("retro", "unit")
    def test_dangling_observations_become_a_candidate(self):
        out = self.draft(observations=[{"ts": "2026-10-01T12:00:00+00:00",
                                        "ord": "station:|slot:/F", "count": 2},
                                       {"ts": "2026-10-01T12:05:00+00:00",
                                        "ord": "station:|slot:/F", "count": 3}])
        cand = self.keys(out)["dangling"]
        self.assertEqual(cand["count"], 2)  # count is the number of evidence items, as elsewhere
        self.assertEqual(len(cand["evidence"]), 2)
        self.assertIn("2026-10-01T12:00:00+00:00", " ".join(cand["evidence"]))
        self.assertIn("found 3", cand["evidence"][1])  # the magnitude lives in the evidence

    @go_server.go_twin("retro", "unit")
    def test_count_means_the_number_of_evidence_items_on_every_candidate(self):
        for i in range(2):
            self.add_audit(audit_row("2026-10-01T10:00:0%d+00:00" % i, "n4_set_slot", "refused",
                                     "ord 'x' is outside the write scope"))
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"))
        out = self.draft(observations=[{"ts": "2026-10-01T12:00:00+00:00", "ord": "o",
                                        "count": 9}])
        for cand in out["candidates"]:
            self.assertEqual(cand["count"], len(cand["evidence"]), cand["key"])

    @go_server.go_twin("retro", "unit")
    def test_the_three_scope_reasons_are_one_candidate(self):
        for i, reason in enumerate(("no --write-scope configured: every write is refused",
                                    "ord 'x' is not a plain station ORD",
                                    "ord 'x' is outside the write scope")):
            self.add_audit(audit_row("2026-10-01T10:00:0%d+00:00" % i, "n4_set_slot", "refused",
                                     reason))
        out = self.draft()
        self.assertEqual([c["key"] for c in out["candidates"]], ["refusal:scope"])
        self.assertEqual(out["candidates"][0]["count"], 3)
        self.assertEqual(len([r for r in retro._REFUSALS if r[0] == "scope"]), 1)

    @go_server.go_twin("retro", "unit")
    def test_every_shared_refusal_reason_is_classified_by_the_retro(self):
        needles = [n for r in retro._REFUSALS for n in r[1]]
        for name, text in sorted(vars(safety).items()):
            if name.startswith("REASON_"):
                self.assertTrue(any(n in text for n in needles), name)

    @go_server.go_twin("retro", "unit")
    def test_the_raise_sites_use_the_shared_reason_constants(self):
        scope = safety.WriteScope(["station:|slot:/A"])
        for ord_str, reason in (("station:|slot:/B", safety.REASON_SCOPE_OUTSIDE),
                                ("/etc", safety.REASON_SCOPE_PLAIN)):
            with self.assertRaises(safety.SafetyError) as cm:
                scope.check(ord_str)
            self.assertIn(reason, str(cm.exception))
        with self.assertRaises(safety.SafetyError) as cm:
            safety.WriteScope([]).check("station:|slot:/A")
        self.assertIn(safety.REASON_SCOPE_NONE, str(cm.exception))
        tokens = safety.ConfirmationTokens(60, lambda: 1000)
        with self.assertRaises(safety.SafetyError) as cm:
            tokens.consume("t", {}, "h", None)
        self.assertIn(safety.REASON_TOKEN_MISSING, str(cm.exception))

    @go_server.go_twin("retro", "injection")
    def test_draft_filters_by_since_once(self):
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_set_slot", "refused",
                                 "ord 'x' is outside the write scope"))
        with mock.patch.object(retro, "_since_filter", wraps=retro._since_filter) as spy:
            self.draft(since="2026-10-01T08:00:00Z")
        self.assertEqual(spy.call_count, 1)

    @go_server.go_twin("retro", "unit")
    def test_the_state_dir_default_is_owned_by_safety_alone(self):
        from mcp_n4 import tools_write
        self.assertEqual(safety.DEFAULT_STATE_DIR, "~/.local/state/mcp-n4")
        self.assertFalse(hasattr(retro, "DEFAULT_STATE_DIR"))
        self.assertFalse(hasattr(tools_write, "DEFAULT_STATE_DIR"))

    @go_server.go_twin("retro", "unit")
    def test_the_template_ships_inside_the_package(self):
        pkg = os.path.dirname(os.path.abspath(retro.__file__))
        self.assertEqual(retro.TEMPLATE, os.path.join(pkg, "templates", "retro.template.md"))
        self.assertTrue(os.path.isfile(retro.TEMPLATE))

    @go_server.go_twin("retro", "unit")
    def test_since_filters_older_entries(self):
        self.add_audit(audit_row("2026-10-01T09:00:00+00:00", "n4_set_slot", "refused",
                                 "ord 'x' is outside the write scope"))
        self.add_journal(journal_batch(B1, "2026-10-01T09:00:00+00:00", "mismatch"))
        self.assertEqual(self.draft(since="2026-10-01T10:00:00Z")["candidates"], [])
        self.assertEqual(len(self.draft(since="2026-10-01T08:00:00+00:00")["candidates"]), 2)

    @go_server.go_twin("retro", "unit")
    def test_bad_since_is_a_value_error(self):
        with self.assertRaises(ValueError):
            self.draft(since="yesterday")

    @go_server.go_twin("retro", "unit")
    def test_no_secret_leaks_from_reasons_or_args(self):
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_connect", "error",
                                 "BoxError: login failed for %s" % CANARY,
                                 args={"password": CANARY, "token": CANARY}),
                       audit_row("2026-10-01T10:00:01+00:00", "n4_set_slot", "refused",
                                 "ord 'station:|slot:/%s' is outside the write scope" % CANARY,
                                 args={"credential": CANARY}))
        out = self.draft()
        blob = json.dumps(out) + out["markdown"]
        self.assertTrue(out["candidates"])
        self.assertNotIn(CANARY, blob)

    @go_server.go_twin("retro", "unit")
    def test_missing_state_dir_yields_no_candidates(self):
        out = retro.draft(os.path.join(self.dir, "nope"), station="LLM", date="2026-10-01")
        self.assertEqual(out["candidates"], [])


@go_server.go_twin("retro", "unit")
class TestRender(Fixture):
    def test_markdown_follows_the_template_with_delta_table(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"))
        md = self.draft()["markdown"]
        self.assertTrue(md.startswith("<!-- review-status: pending -->"))
        self.assertIn("## Proposed kit deltas", md)
        self.assertIn("| # | Proposed change | Target (file · section) | Evidence "
                      "(batch_id / tool call / audit line) | Type | Priority |", md)
        self.assertIn("## Already covered", md)
        self.assertIn("## Honest verdict", md)
        self.assertIn(B1, md)
        self.assertIn("LLM", md)
        self.assertIn("2026-10-01", md)
        self.assertNotIn("{{", md)

    def test_honesty_line_when_there_is_no_friction(self):
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_set_slot", "executed",
                                 "verdict=verified", batch_id=B1))
        self.add_journal(journal_batch(B1, "2026-10-01T10:00:00+00:00", "verified"))
        out = self.draft()
        self.assertEqual(out["candidates"], [])
        self.assertIn("no new deltas;", out["markdown"])
        self.assertEqual(retro.count_deltas(out["markdown"]), 0)

    def test_count_deltas_counts_table_rows_under_the_heading_only(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"))
        self.add_journal(journal_batch(B2, "2026-10-01T11:01:00+00:00", "failed"))
        out = self.draft()
        self.assertEqual(retro.count_deltas(out["markdown"]), 2)
        self.assertEqual(len(out["candidates"]), 2)

    def test_pipes_in_cells_are_escaped(self):
        self.assertEqual(retro.cell("a|b\nc"), "a\\|b c")


class TestServerTool(unittest.TestCase):
    """Black-box under both implementations (MCP_N4_IMPL=go included)."""

    def server(self, **kw):
        srv = go_server.make_server(**kw)
        self.addCleanup(go_server.close_server, srv)
        return srv

    def rpc(self, srv, method, params=None):
        return srv.dispatch({"jsonrpc": "2.0", "id": 1, "method": method, "params": params})

    def test_tool_is_listed_read_only_in_both_modes(self):
        for allow in (False, True):  # under Go --allow-writes is refused at start: a skip
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
                safety.Journal(state).append(r)
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
                safety.Journal(state).append(r)  # written before this server started
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


def tier_refusal():
    from mcp_n4 import tiers
    return tiers.write_block("C", None, "LLM", {})


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
    """The tool's reply, byte for byte, is the one tools_read.n4_session_retro_draft computes
    in-process on the same state dir (black-box under both implementations)."""

    WHOLE = "1970-01-01T00:00:00Z"

    def setUp(self):
        super().setUp()
        self.srv = go_server.make_server(state_dir=self.dir)
        self.addCleanup(go_server.close_server, self.srv)

    def reply(self, **args):
        msg = {"jsonrpc": "2.0", "id": 1, "method": "tools/call",
               "params": {"name": "n4_session_retro_draft", "arguments": args}}
        return self.srv.dispatch(msg)["result"]

    def expected(self, **args):
        ctx = types.SimpleNamespace(session=None, state_dir=self.dir, observations=[],
                                    started_at=None)  # every call here passes since
        return tools_read.n4_session_retro_draft(ctx, args)

    def assertSameDraft(self, **args):
        result = self.reply(**args)
        want = self.expected(**args)
        self.assertFalse(result.get("isError", False), result)
        self.assertEqual(result["content"][0]["text"], json.dumps(want))
        self.assertEqual(result["structuredContent"], want)
        return want

    def assertSameError(self, **args):
        result = self.reply(**args)
        try:
            self.expected(**args)
        except ValueError as exc:
            text = str(exc)
        except Exception as exc:  # noqa: BLE001 - the server names the type only
            text = "internal error (%s)" % type(exc).__name__
        else:
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
