"""Version-tier gate (METHODOLOGY section 5): detection at connect and the write refusal."""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_retro  # noqa: E402  (module import: its tests are not re-collected here)
import test_server  # noqa: E402
import test_tools_write  # noqa: E402

FOLDER = test_tools_write.FOLDER
#: The reason every tier refusal starts with (the Go server's tier gate words it; each test
#: that uses it finds it in a real refusal, so it cannot drift silently).
REASON_TIER = "version tier refuses writes"
#: Every mutating tool with arguments that pass schema validation.
MUTATING = [
    ("n4_create_component", {"parent_ord": FOLDER, "name": "X", "type": "kitControl:NumericConst"}),
    ("n4_set_slot", {"ord": FOLDER, "slot": "s", "value": 1, "value_type": "baja:Double"}),
    ("n4_invoke_action", {"ord": FOLDER, "action": "auto"}),
    ("n4_create_link", {"source_ord": FOLDER, "source_slot": "out",
                        "target_ord": FOLDER, "target_slot": "in10"}),
    ("n4_remove_component", {"parent_ord": FOLDER, "name": "X"}),
    ("n4_rollback", {"batch_id": "0" * 32}),
    ("n4_save_station", {}),
]


class TestConnectReportsTier(test_server.ToolTestCase):
    def test_tier_a_station(self):
        out = self.connect()
        self.assertEqual((out["version"], out["tier"], out["tier_writes"]),
                         ("4.14.0.162", "A", "allowed"))
        desc = self.ok("n4_describe_session")
        self.assertEqual((desc["version"], desc["tier"]), ("4.14.0.162", "A"))

    def test_unknown_version_is_tier_c(self):
        self.fake.product_version = None
        out = self.connect()
        self.assertEqual((out["version"], out["version_source"], out["tier"],
                          out["tier_writes"]), (None, None, "C", "refused"))

    def test_a_detected_version_carries_no_version_error(self):
        self.assertNotIn("version_error", self.connect())

    def test_a_missing_about_endpoint_is_reported(self):
        self.fake.product_version = None
        self.assertIn("HTTP 404", self.connect()["version_error"])

    def test_tier_b_station_reports_refused_without_opt_in(self):
        self.fake.product_version = "4.15.3.28"
        out = self.connect()
        self.assertEqual((out["tier"], out["tier_writes"]), ("B", "refused"))

    def test_describe_without_session_has_no_tier(self):
        desc = self.ok("n4_describe_session")
        self.assertIsNone(desc["tier"])
        self.assertIsNone(desc["version"])


class TierWriteCase(test_tools_write.WriteTestCase):
    VERSION = "4.14.0.162"

    def setUp(self):
        super().setUp()
        self.fake.product_version = self.VERSION

    def create(self, **kw):
        return dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst", **kw)

    def audit_outcomes(self):
        return [entry["outcome"] for entry in self.lines("audit.jsonl")]


class TestTierAUnchanged(TierWriteCase):
    def test_tier_a_writes_without_annotation(self):
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertNotIn("tier_gate", plan)
        out = self.run_write("n4_create_component", **self.create())
        self.assertEqual(out["verdict"], "verified")
        self.assertIn("Pump", self.children())


class TestTierB(TierWriteCase):
    VERSION = "4.15.3.28"

    def test_dry_run_is_allowed_and_says_the_tier_would_block(self):
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertIn("confirmation_token", plan)
        gate = plan["tier_gate"]
        self.assertEqual((gate["tier"], gate["version"], gate["would_block"]),
                         ("B", "4.15.3.28", True))
        self.assertIn("--allow-tier-b FakeStation", gate["reason"])
        self.assertNotIn("tier_gate", plan["plan"])  # not part of what the token vouches for

    def test_execution_is_refused_without_opt_in(self):
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        text = self.err("n4_create_component", dry_run=False,
                        confirmation_token=plan["confirmation_token"], **self.create())
        self.assertIn(REASON_TIER, text)
        self.assertIn("--allow-tier-b FakeStation", text)
        self.assertIn("Workbench", text)
        self.assertNotIn("Pump", self.children())
        self.assertEqual(self.audit_outcomes(), ["planned", "refused"])
        self.assertEqual(self.lines("journal.jsonl"), [])

    def test_every_mutating_tool_is_gated(self):
        self.connect_verified()
        self.assertEqual(sorted(t for t, _ in MUTATING), test_server.WRITE_TOOLS)
        listed = [t["name"] for t in self.srv.dispatch(test_server.rpc("tools/list"))[
            "result"]["tools"]]  # the binary's own write tools: WRITE_TOOLS cannot drift
        self.assertEqual(sorted(set(listed) - set(test_server.READ_TOOLS)),
                         test_server.WRITE_TOOLS)
        for tool, args in MUTATING:
            text = self.err(tool, dry_run=False, confirmation_token="1.a.b", **args)
            self.assertIn(REASON_TIER, text, tool)

    def test_operator_opt_in_allows_writes(self):
        self.start_server(allow_tier_b=("FakeStation",))
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertNotIn("tier_gate", plan)
        out = self.run_write("n4_create_component", **self.create())
        self.assertEqual(out["verdict"], "verified")

    def test_reads_are_unaffected(self):
        self.connect_verified()
        self.ok("n4_navigate", ord=FOLDER)
        self.ok("n4_read_slots", ord=FOLDER)
        self.ok("n4_list_links", ord=FOLDER)


class TestTierC(TierWriteCase):
    VERSION = "4.10.1.36"

    def test_execution_is_refused_with_the_probe_steps(self):
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertEqual(plan["tier_gate"]["tier"], "C")
        text = self.err("n4_create_component", dry_run=False,
                        confirmation_token=plan["confirmation_token"], **self.create())
        for needle in (REASON_TIER, "--allow-tier-c FakeStation", "reg.loadContract",
                       "scratch", "Workbench"):
            self.assertIn(needle, text)
        self.assertNotIn("Pump", self.children())

    def test_tier_b_opt_in_does_not_open_tier_c(self):
        self.start_server(allow_tier_b=("FakeStation",))
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertTrue(plan["tier_gate"]["would_block"])

    def test_unknown_version_is_refused_as_tier_c(self):
        self.fake.product_version = None
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.create())
        self.assertEqual((plan["tier_gate"]["tier"], plan["tier_gate"]["version"]), ("C", None))
        text = self.err("n4_create_component", dry_run=False,
                        confirmation_token=plan["confirmation_token"], **self.create())
        self.assertEqual(text, test_retro.tier_refusal("FakeStation"))

    def test_operator_opt_in_allows_writes(self):
        self.start_server(allow_tier_c=("FakeStation",))
        self.connect_verified()
        out = self.run_write("n4_create_component", **self.create())
        self.assertEqual(out["verdict"], "verified")


if __name__ == "__main__":
    unittest.main()
