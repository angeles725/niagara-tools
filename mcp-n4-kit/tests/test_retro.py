"""Tests for mcp_n4.retro: evidence-backed session retro drafts from audit + journal."""
import json
import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from mcp_n4 import retro, safety, server  # noqa: E402

CANARY = "pw-Zk93-unique-canary"
B1, B2, B3, B4, B5 = ("%032x" % n for n in range(1, 6))


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

    def test_same_reason_groups_into_one_candidate_with_a_count(self):
        for i in range(3):
            self.add_audit(audit_row("2026-10-01T10:00:0%d+00:00" % i, "n4_create_link",
                                     "refused", "confirmation_token expired: run again"))
        cand = self.keys(self.draft())["refusal:token_expired"]
        self.assertEqual(cand["count"], 3)
        self.assertEqual(len(cand["evidence"]), 3)

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

    def test_in_doubt_batches_are_flagged_whether_result_missing_or_flagged(self):
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", None, with_result=False))
        self.add_journal(journal_batch(B2, "2026-10-01T11:01:00+00:00", "unverified",
                                       in_doubt=True))
        cand = self.keys(self.draft())["in_doubt"]
        self.assertEqual(cand["count"], 2)
        self.assertIn(B1, " ".join(cand["evidence"]))
        self.assertIn(B2, " ".join(cand["evidence"]))
        self.assertEqual(cand["priority"], "HIGH")

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

    def test_count_means_the_number_of_evidence_items_on_every_candidate(self):
        for i in range(2):
            self.add_audit(audit_row("2026-10-01T10:00:0%d+00:00" % i, "n4_set_slot", "refused",
                                     "ord 'x' is outside the write scope"))
        self.add_journal(journal_batch(B1, "2026-10-01T11:00:00+00:00", "mismatch"))
        out = self.draft(observations=[{"ts": "2026-10-01T12:00:00+00:00", "ord": "o",
                                        "count": 9}])
        for cand in out["candidates"]:
            self.assertEqual(cand["count"], len(cand["evidence"]), cand["key"])

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

    def test_every_shared_refusal_reason_is_classified_by_the_retro(self):
        needles = [n for r in retro._REFUSALS for n in r[1]]
        for name, text in sorted(vars(safety).items()):
            if name.startswith("REASON_"):
                self.assertTrue(any(n in text for n in needles), name)

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

    def test_draft_filters_by_since_once(self):
        self.add_audit(audit_row("2026-10-01T10:00:00+00:00", "n4_set_slot", "refused",
                                 "ord 'x' is outside the write scope"))
        with mock.patch.object(retro, "_since_filter", wraps=retro._since_filter) as spy:
            self.draft(since="2026-10-01T08:00:00Z")
        self.assertEqual(spy.call_count, 1)

    def test_the_state_dir_default_is_owned_by_safety_alone(self):
        from mcp_n4 import tools_write
        self.assertEqual(safety.DEFAULT_STATE_DIR, "~/.local/state/mcp-n4")
        self.assertFalse(hasattr(retro, "DEFAULT_STATE_DIR"))
        self.assertFalse(hasattr(tools_write, "DEFAULT_STATE_DIR"))

    def test_the_template_ships_inside_the_package(self):
        pkg = os.path.dirname(os.path.abspath(retro.__file__))
        self.assertEqual(retro.TEMPLATE, os.path.join(pkg, "templates", "retro.template.md"))
        self.assertTrue(os.path.isfile(retro.TEMPLATE))

    def test_since_filters_older_entries(self):
        self.add_audit(audit_row("2026-10-01T09:00:00+00:00", "n4_set_slot", "refused",
                                 "ord 'x' is outside the write scope"))
        self.add_journal(journal_batch(B1, "2026-10-01T09:00:00+00:00", "mismatch"))
        self.assertEqual(self.draft(since="2026-10-01T10:00:00Z")["candidates"], [])
        self.assertEqual(len(self.draft(since="2026-10-01T08:00:00+00:00")["candidates"]), 2)

    def test_bad_since_is_a_value_error(self):
        with self.assertRaises(ValueError):
            self.draft(since="yesterday")

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

    def test_missing_state_dir_yields_no_candidates(self):
        out = retro.draft(os.path.join(self.dir, "nope"), station="LLM", date="2026-10-01")
        self.assertEqual(out["candidates"], [])


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
    def rpc(self, srv, method, params=None):
        return srv.dispatch({"jsonrpc": "2.0", "id": 1, "method": method, "params": params})

    def test_tool_is_listed_read_only_in_both_modes(self):
        for allow in (False, True):
            with tempfile.TemporaryDirectory() as tmp:
                srv = server.Server(allow_writes=allow, state_dir=os.path.join(tmp, "s"))
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
            srv = server.Server(state_dir=state)
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
            srv = server.Server(state_dir=state)
            call = lambda args: self.rpc(srv, "tools/call", {
                "name": "n4_session_retro_draft", "arguments": args})["result"]["structuredContent"]
            self.assertEqual(call({})["candidates"], [])
            self.assertEqual(len(call({"since": "2000-01-01T00:00:00Z"})["candidates"]), 1)

    def test_bad_since_is_a_tool_error(self):
        with tempfile.TemporaryDirectory() as tmp:
            srv = server.Server(state_dir=os.path.join(tmp, "s"))
            res = self.rpc(srv, "tools/call", {"name": "n4_session_retro_draft",
                                               "arguments": {"since": "nope"}})["result"]
            self.assertTrue(res["isError"])


if __name__ == "__main__":
    unittest.main()
