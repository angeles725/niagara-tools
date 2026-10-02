"""Version-tier gate (METHODOLOGY section 5): detection at connect and the write refusal."""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_server  # noqa: E402
import test_tools_write  # noqa: E402
from mcp_n4 import retro, safety, server, tiers, tools_write  # noqa: E402

FOLDER = test_tools_write.FOLDER
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


class TestTierClassification(unittest.TestCase):
    def test_tier_table(self):
        table = {
            "4.13.2.18": "A", "4.14.0.162": "A", "4.14": "A", "4.13.0.0.1": "A",
            "4.15.3.28": "B", "4.3.58.18": "B",
            "4.12.1.16": "C", "4.4.73.24": "C", "4.30.1": "C", "4.1.27": "C",
            "5.0.0": "C", "3.8.401": "C",
            None: "C", "": "C", "garbage": "C", "4": "C", "v4.14.0": "C",
        }
        for version, want in table.items():
            self.assertEqual(tiers.tier_of(version), want, version)

    def test_product_version_is_read_from_the_about_document(self):
        xml = ('<obj is="obix:About"><str name="serverName" val="h"/>'
               '<str name="productVersion" val="4.15.3.28"/></obj>')
        self.assertEqual(tiers.product_version(xml), "4.15.3.28")
        swapped = '<obj><str val="4.13.2.18" name="productVersion" /></obj>'
        self.assertEqual(tiers.product_version(swapped), "4.13.2.18")
        for text in ("", "<obj/>", '<str name="productName" val="Niagara"/>', None):
            self.assertIsNone(tiers.product_version(text))

    def test_write_block_policy(self):
        self.assertIsNone(tiers.write_block("A", "4.14.0.162", "S", {}))
        self.assertIsNone(tiers.write_block("B", "4.15.3.28", "S", {"B": {"S"}}))
        self.assertIsNone(tiers.write_block("C", "4.10.0", "S", {"C": {"S"}}))
        self.assertIsNotNone(tiers.write_block("B", "4.15.3.28", "S", {"B": {"other"}}))
        self.assertIsNotNone(tiers.write_block("B", "4.15.3.28", "S", {"C": {"S"}}))
        self.assertIsNotNone(tiers.write_block("C", None, "S", {"B": {"S"}}))

    def test_refusals_name_the_routes(self):
        b = tiers.write_block("B", "4.15.3.28", "S", {})
        c = tiers.write_block("C", None, "S", {})
        for text in (b, c):
            self.assertIn(safety.REASON_TIER, text)
            self.assertIn("Workbench", text)
            self.assertIn("dry run", text)
        self.assertIn("--allow-tier-b S", b)
        self.assertIn("4.15.3.28", b)
        self.assertIn("--allow-tier-c S", c)
        self.assertIn("unknown", c)
        for step in ("reg.loadContract", "loadRoot", "scratch"):
            self.assertIn(step, c)

    def test_the_retro_classifies_tier_refusals(self):
        needles = [n for r in retro._REFUSALS for n in r[1]]
        self.assertIn(safety.REASON_TIER, needles)


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

    def test_tier_b_station_reports_refused_without_opt_in(self):
        self.fake.product_version = "4.15.3.28"
        out = self.connect()
        self.assertEqual((out["tier"], out["tier_writes"]), ("B", "refused"))

    def test_describe_without_session_has_no_tier(self):
        desc = self.ok("n4_describe_session")
        self.assertIsNone(desc["tier"])
        self.assertIsNone(desc["version"])

    def test_opt_in_for_an_unconfigured_station_is_a_start_error(self):
        for kw in ({"allow_tier_b": ("ghost",)}, {"allow_tier_c": ("ghost",)}):
            with self.assertRaises(ValueError):
                server.Server(allow_http=True, stations=self.stations, **kw)

    def test_cli_parses_the_opt_in_flags(self):
        args = server.parse_args(["--allow-tier-b", "S1", "--allow-tier-b", "S2",
                                  "--allow-tier-c", "S3"])
        self.assertEqual((args.allow_tier_b, args.allow_tier_c), (["S1", "S2"], ["S3"]))


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
        self.assertIn(safety.REASON_TIER, text)
        self.assertIn("--allow-tier-b FakeStation", text)
        self.assertIn("Workbench", text)
        self.assertNotIn("Pump", self.children())
        self.assertEqual(self.audit_outcomes(), ["planned", "refused"])
        self.assertEqual(self.lines("journal.jsonl"), [])

    def test_every_mutating_tool_is_gated(self):
        self.connect_verified()
        self.assertEqual(sorted(t for t, _ in MUTATING), sorted(tools_write.NAMES))
        for tool, args in MUTATING:
            text = self.err(tool, dry_run=False, confirmation_token="1.a.b", **args)
            self.assertIn(safety.REASON_TIER, text, tool)

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
        for needle in (safety.REASON_TIER, "--allow-tier-c FakeStation", "reg.loadContract",
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

    def test_operator_opt_in_allows_writes(self):
        self.start_server(allow_tier_c=("FakeStation",))
        self.connect_verified()
        out = self.run_write("n4_create_component", **self.create())
        self.assertEqual(out["verdict"], "verified")


if __name__ == "__main__":
    unittest.main()
