"""n4_bql_query / n4_inventory (retro 2026-10-02 D1, D7, D8) against a fake HTTP layer.

The CSV fixtures under tests/fixtures/ are recorded station output, anonymized.
"""
import csv
import io
import json
import os
import sys
import tempfile
import unittest
import urllib.error
import urllib.parse

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box, bql, server, tools_read, tools_write  # noqa: E402

FIXTURES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures")


def fixture(name):
    with open(os.path.join(FIXTURES, name), "rb") as fh:
        return fh.read()


class _Resp(io.BytesIO):
    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.close()


class FakeOpener:
    """Records every GET and answers from `routes` (substring of the BQL -> bytes or code)."""

    def __init__(self, routes, inner):
        self.routes, self.requests, self.inner = routes, [], inner

    def open(self, req, timeout=None):
        if "/ord/" not in req.full_url:  # BOX POSTs still reach the fake station
            return self.inner.open(req, timeout=timeout)
        self.requests.append((req, timeout))
        ord_text = urllib.parse.unquote(req.full_url.split("/ord/", 1)[1])
        for needle, answer in self.routes:
            if needle in ord_text:
                if isinstance(answer, int):
                    raise urllib.error.HTTPError(req.full_url, answer, "x", {}, io.BytesIO())
                return _Resp(answer)
        raise AssertionError("unexpected ORD %s" % ord_text)


class TestPureHelpers(unittest.TestCase):
    def test_unescape_slot_decodes_dollar_escapes(self):
        self.assertEqual(bql.unescape_slot("Meter$20Side$20A"), "Meter Side A")
        self.assertEqual(bql.unescape_slot("Ahu$2dUnit$2001_1"), "Ahu-Unit 01_1")
        self.assertEqual(bql.unescape_slot("Caf$u00e9"), "Café")
        self.assertEqual(bql.unescape_slot("plain"), "plain")

    def test_validate_query_accepts_select_only(self):
        self.assertEqual(bql.validate_query("  SELECT name from driver:Device "),
                         "SELECT name from driver:Device")
        where = "select name, out from control:ControlPoint where name like 'T*'"
        self.assertEqual(bql.validate_query(where), where)
        for bad in ("delete from control:ControlPoint", "name from x", "", "select",
                    "selectname from x", 5, None):
            with self.assertRaises(ValueError, msg=repr(bad)):
                bql.validate_query(bad)

    def test_validate_query_refuses_pipe_and_control_characters(self):
        for bad in ("select name from driver:Device|view:file:ITableToCsv",
                    "select name from control:ControlPoint where name = 'a|b'",
                    "select name\nfrom driver:Device", "select name from x\x00"):
            with self.assertRaises(ValueError, msg=repr(bad)):
                bql.validate_query(bad)

    def test_compose_ord_builds_the_one_projected_ord(self):
        q = "select name from driver:Device"
        expected = "station:|slot:/Drivers|bql:%s|view:file:ITableToCsv" % q
        for base in ("station:|slot:/Drivers", "/Drivers", "slot:/Drivers", "station:|slot:/Drivers/"):
            self.assertEqual(bql.compose_ord(base, q), expected, base)
        self.assertEqual(bql.compose_ord("station:|slot:/", q),
                         "station:|slot:/|bql:%s|view:file:ITableToCsv" % q)

    def test_compose_ord_refuses_a_base_that_injects_another_scheme(self):
        for base in ("station:|slot:/Drivers|bql:select x from y", "station:|h:12",
                     "file:^x", "station:|slot://Drivers", "station:|slot:/A\nB", ""):
            with self.assertRaises(ValueError, msg=repr(base)):
                bql.compose_ord(base, "select name from driver:Device")

    def test_parse_csv_strips_bom_and_control_chars_and_caps_rows(self):
        text = fixture("bql_points.csv").decode("utf-8")
        cols, rows, truncated = bql.parse_csv(text, max_rows=100)
        self.assertEqual(cols, ["Slot Path", "Name", "Type", "Out", "Data Address"])
        self.assertEqual(len(rows), 5)
        self.assertFalse(truncated)
        self.assertEqual(rows[1]["Out"], "1,024.0 {ok}")  # a quoted comma survives
        self.assertEqual(rows[0]["Data Address"], "Decimal:302")
        cols, rows, truncated = bql.parse_csv(text, max_rows=2)
        self.assertEqual((len(rows), truncated), (2, True))
        _, rows, _ = bql.parse_csv("A,B\r\nx\x07y,\x1bz\r\n", max_rows=5)
        self.assertEqual(rows, [{"A": "xy", "B": "z"}])

    def test_parse_csv_decodes_the_name_column_and_keeps_the_ord_ready_path(self):
        _, rows, _ = bql.parse_csv(fixture("bql_devices.csv").decode("utf-8"), max_rows=10)
        self.assertEqual(rows[0]["Name"], "Meter A")
        self.assertEqual(rows[0]["Slot Path"], "slot:/Drivers/ModbusTcpNetwork/Meter$20A")

    def test_parse_csv_of_an_empty_body_has_no_columns(self):
        self.assertEqual(bql.parse_csv("", max_rows=5), ([], [], False))

    def test_local_devices_are_flagged_and_counted_apart(self):
        dev = lambda name: {"Slot Path": "slot:/Drivers/SnmpNetwork/" + name, "Name": name,
                            "Type": "snmp:SnmpDevice"}
        self.assertTrue(bql.is_local_device(dev("localDevice")))
        self.assertFalse(bql.is_local_device(dev("Ups1")))
        _, devices, _ = bql.parse_csv(fixture("bql_devices.csv").decode("utf-8"), 50)
        _, points, _ = bql.parse_csv(fixture("bql_points.csv").decode("utf-8"), 50)
        _, nets, _ = bql.parse_csv(fixture("bql_networks.csv").decode("utf-8"), 50)
        inv = bql.summarize_inventory("/Drivers", nets, devices, points)
        self.assertEqual(inv["totals"], {"networks": 4, "field_devices": 4, "local_devices": 1,
                                         "points": 5, "unassigned_points": 0})
        snmp = [n for n in inv["networks"] if n["network"] == "SnmpNetwork"][0]
        self.assertEqual((snmp["field_devices"], snmp["local_devices"], snmp["points"]),
                         (1, 1, 1))
        local = [d for d in inv["devices"] if d["local"]]
        self.assertEqual([d["name"] for d in local], ["localDevice"])
        meter = [d for d in inv["devices"] if d["name"] == "Meter A"][0]
        self.assertEqual((meter["network"], meter["points"]), ("ModbusTcpNetwork", 2))
        empty = [n for n in inv["networks"] if n["network"] == "NiagaraNetwork"][0]
        self.assertEqual((empty["field_devices"], empty["points"]), (0, 0))


class TestHeaderRobustness(unittest.TestCase):
    """Audit 2026-10-03 F7 (duplicate headers) and F12 (localized headers)."""

    def test_duplicate_headers_keep_every_column(self):
        columns, rows, _ = bql.parse_csv("Type,Type,Type#2,Type\nA,B,C,D\n")
        self.assertEqual(columns, ["Type", "Type#2", "Type#2#2", "Type#3"])
        self.assertEqual(rows, [{"Type": "A", "Type#2": "B", "Type#2#2": "C", "Type#3": "D"}])

    def test_selected_slots_name_the_queried_columns_in_order(self):
        self.assertEqual(bql.selected_slots("select slotPath, name ,type from driver:Device"),
                         ["slotPath", "name", "type"])
        self.assertEqual(bql.selected_slots("SELECT proxyExt.dataAddress FROM x where a = 1"),
                         ["proxyExt.dataAddress"])
        self.assertIsNone(bql.selected_slots("select * from driver:Device"))

    def test_the_name_column_is_decoded_by_position_whatever_its_header(self):
        columns, rows, _ = bql.parse_csv("Ruta,Nombre\nslot:/A$20B,A$20B\n",
                                         name_positions=[1])
        self.assertEqual(rows, [{"Ruta": "slot:/A$20B", "Nombre": "A B"}])

    def test_by_position_maps_rows_to_the_queried_slot_names(self):
        columns = ["Ruta", "Nombre", "Tipo"]
        rows = [{"Ruta": "slot:/D/N", "Nombre": "N", "Tipo": "x:Net"}]
        self.assertEqual(bql.by_position(columns, rows, ["Slot Path", "Name", "Type"]),
                         [{"Slot Path": "slot:/D/N", "Name": "N", "Type": "x:Net"}])
        with self.assertRaisesRegex(ValueError, "2 column.*at least 3"):
            bql.by_position(columns[:2], rows, ["Slot Path", "Name", "Type"])


def localized(name):
    """A fixture with Spanish display headers (the station localizes them)."""
    head, _, rest = fixture(name).partition(b"\n")
    head = head.replace(b"Slot Path", b"Ruta de slot").replace(b"Name", b"Nombre") \
        .replace(b"Type", b"Tipo").replace(b"Out", b"Salida")
    return head + b"\n" + rest


class BqlToolCase(unittest.TestCase):
    """A connected read-only server whose HTTP GETs go to a FakeOpener."""

    def setUp(self, allow_writes=False, state_dir=None):
        self.fake = FakeStation(password="pw-123456").start()
        self.addCleanup(self.fake.stop)
        self.srv = server.Server(allow_http=True, allow_writes=allow_writes,
                                 env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "pw-123456"},
                                 stations={"FakeStation": self.fake.url}, state_dir=state_dir,
                                 write_scopes=["station:|slot:/Folder"] if allow_writes else ())
        self.addCleanup(self.srv.ctx.close)
        self.call("n4_connect", station="FakeStation")
        client = self.srv.ctx.session.client
        self.opener = FakeOpener([("driver:DeviceNetwork", fixture("bql_networks.csv")),
                                  ("driver:Device", fixture("bql_devices.csv")),
                                  ("control:ControlPoint", fixture("bql_points.csv"))],
                                 client._opener)
        client._opener = self.opener

    def call(self, name, **args):
        res = self.srv.dispatch({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                                 "params": {"name": name, "arguments": args}})
        self.assertNotIn("error", res, res)
        return res["result"]

    def ok(self, name, **args):
        result = self.call(name, **args)
        self.assertFalse(result.get("isError", False), result)
        return result["structuredContent"]

    def err(self, name, **args):
        result = self.call(name, **args)
        self.assertTrue(result["isError"], result)
        return result["content"][0]["text"]


class TestBqlQueryTool(BqlToolCase):
    def test_query_returns_rows_and_columns_from_the_path_form_ord(self):
        out = self.ok("n4_bql_query", base="station:|slot:/Drivers",
                      query="select slotPath, name, type from driver:Device")
        self.assertEqual(out["columns"], ["Slot Path", "Name", "Type"])
        self.assertEqual(out["row_count"], 5)
        self.assertFalse(out["truncated"])
        self.assertIn("elapsed_ms", out)
        req, timeout = self.opener.requests[0]
        self.assertEqual(req.get_method(), "GET")
        self.assertIn("/ord/station%3A%7Cslot%3A%2FDrivers%7Cbql%3A", req.full_url)
        self.assertNotIn("/ord?", req.full_url)
        self.assertTrue(req.get_header("Authorization").startswith("Basic "))
        self.assertEqual(timeout, 60)

    def test_row_cap_and_timeout_are_passed_through(self):
        out = self.ok("n4_bql_query", base="/Drivers", max_rows=2, timeout_s=5,
                      query="select slotPath from control:ControlPoint")
        self.assertEqual((out["row_count"], out["truncated"]), (2, True))
        self.assertEqual(self.opener.requests[0][1], 5)

    def test_non_select_and_pipe_injection_send_nothing(self):
        for query in ("delete from driver:Device",
                      "select name from driver:Device|view:web:PropertySheet"):
            self.assertIn("select", self.err("n4_bql_query", base="/Drivers", query=query))
        self.err("n4_bql_query", base="/Drivers|bql:select x from y", query="select a from b")
        self.assertEqual(self.opener.requests, [])

    def test_a_401_on_the_bql_path_is_actionable_and_not_retried(self):
        self.opener.routes = [("driver:Device", 401)]
        text = self.err("n4_bql_query", base="/Drivers", query="select name from driver:Device")
        self.assertIn("HTTPBasicScheme", text)
        self.assertIn("do not retry", text)
        self.assertNotIn("pw-123456", text)
        self.assertEqual(len(self.opener.requests), 1)

    def test_a_400_names_the_query_as_the_likely_cause(self):
        self.opener.routes = [("driver:Device", 400)]
        self.assertIn("rejected", self.err("n4_bql_query", base="/Drivers",
                                           query="select name from driver:Device"))

    def test_the_read_is_recorded_as_a_session_observation(self):
        self.ok("n4_bql_query", base="/Drivers", query="select name from driver:Device")
        obs = self.srv.ctx.observations[-1]
        self.assertEqual((obs["tool"], obs["rows"]), ("n4_bql_query", 5))
        self.assertNotIn("count", obs)  # `count` means dangling outputs to the retro

    def test_it_is_listed_as_a_read_only_tool(self):
        names = [t.name for t in tools_read.TOOLS]
        self.assertIn("n4_bql_query", names)
        self.assertIn("n4_inventory", names)
        self.assertNotIn("n4_bql_query", tools_write.NAMES)


class TestBqlAuditInWritesMode(BqlToolCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        os.chmod(self.tmp, 0o700)
        super().setUp(allow_writes=True, state_dir=self.tmp)

    def test_the_read_is_audited_with_outcome_read(self):
        self.ok("n4_bql_query", base="/Drivers", query="select name from driver:Device")
        with open(os.path.join(self.tmp, "audit.jsonl")) as fh:
            lines = [json.loads(line) for line in fh]
        self.assertEqual([(e["tool"], e["outcome"]) for e in lines],
                         [("n4_bql_query", "read")])
        self.assertNotIn("pw-123456", json.dumps(lines))


class TestInventoryTool(BqlToolCase):
    def test_inventory_counts_devices_and_points_per_network_with_local_apart(self):
        out = self.ok("n4_inventory")
        self.assertEqual(out["totals"]["field_devices"], 4)
        self.assertEqual(out["totals"]["local_devices"], 1)
        self.assertEqual(out["totals"]["points"], 5)
        self.assertEqual([p["step"] for p in out["progress"]], ["networks", "devices", "points"])
        self.assertEqual([p["rows"] for p in out["progress"]], [4, 5, 5])
        self.assertNotIn("point_rows", out)

    def test_progress_lines_go_to_the_operator_progress_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "progress.jsonl")
            self.srv.ctx.progress_path = path
            self.ok("n4_inventory", base="/Drivers")
            with open(path) as fh:
                steps = [json.loads(line) for line in fh]
        self.assertEqual([(s["tool"], s["step"], s["rows"]) for s in steps],
                         [("n4_inventory", "networks", 4), ("n4_inventory", "devices", 5),
                          ("n4_inventory", "points", 5)])

    def test_include_points_returns_the_point_rows(self):
        out = self.ok("n4_inventory", include_points=True)
        self.assertEqual(len(out["point_rows"]), 5)

    def test_server_flag_sets_the_progress_file(self):
        args = server.parse_args(["--progress-file", "/x/p.jsonl"])
        self.assertEqual(args.progress_file, "/x/p.jsonl")


class TestLocalizedHeaders(BqlToolCase):
    def setUp(self):
        super().setUp()
        self.opener.routes = [("driver:DeviceNetwork", localized("bql_networks.csv")),
                              ("driver:Device", localized("bql_devices.csv")),
                              ("control:ControlPoint", localized("bql_points.csv"))]

    def test_inventory_does_not_depend_on_english_headers(self):
        out = self.ok("n4_inventory")
        self.assertEqual((out["totals"]["field_devices"], out["totals"]["local_devices"],
                          out["totals"]["points"], out["totals"]["unassigned_points"]),
                         (4, 1, 5, 0))
        self.assertIn("Meter A", [d["name"] for d in out["devices"]])

    def test_bql_query_decodes_the_queried_name_column(self):
        out = self.ok("n4_bql_query", base="/Drivers",
                      query="select slotPath, name, type from driver:Device")
        self.assertEqual(out["columns"], ["Ruta de slot", "Nombre", "Tipo"])
        self.assertIn("Meter A", [r["Nombre"] for r in out["rows"]])
        self.assertIn("slot:/Drivers/ModbusTcpNetwork/Meter$20A",
                      [r["Ruta de slot"] for r in out["rows"]])


class TestBqlOutputFile(BqlToolCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.state = os.path.join(self.tmp.name, "state")
        super().setUp(state_dir=self.state)

    def test_rows_go_to_a_private_file_under_the_state_dir_not_into_the_reply(self):
        out = self.ok("n4_bql_query", base="/Drivers", output_file="devices.json",
                      query="select slotPath, name, type from driver:Device")
        path = os.path.join(self.state, "bql", "devices.json")
        self.assertEqual(out["output_file"], path)
        self.assertNotIn("rows", out)
        self.assertEqual(out["row_count"], 5)
        self.assertEqual(os.stat(path).st_mode & 0o777, 0o600)
        self.assertEqual(os.stat(os.path.dirname(path)).st_mode & 0o777, 0o700)
        with open(path) as fh:
            saved = json.load(fh)
        self.assertEqual((saved["columns"], len(saved["rows"])), (["Slot Path", "Name", "Type"], 5))

    def test_a_csv_output_file_keeps_the_columns_in_order(self):
        self.ok("n4_bql_query", base="/Drivers", output_file="devices.csv",
                query="select slotPath, name, type from driver:Device")
        with open(os.path.join(self.state, "bql", "devices.csv"), newline="") as fh:
            lines = list(csv.reader(fh))
        self.assertEqual(lines[0], ["Slot Path", "Name", "Type"])
        self.assertEqual(len(lines), 6)

    def test_a_path_or_an_existing_file_is_refused_and_nothing_is_read(self):
        for bad in ("../x.json", "a/b.json", "/tmp/x.json", "x.txt", ".hidden.json", ""):
            self.assertIn("output_file", self.err(
                "n4_bql_query", base="/Drivers", output_file=bad,
                query="select name from driver:Device"), bad)
        self.assertEqual(self.opener.requests, [])
        self.ok("n4_bql_query", base="/Drivers", output_file="d.json",
                query="select name from driver:Device")
        self.assertIn("exists", self.err("n4_bql_query", base="/Drivers", output_file="d.json",
                                         query="select name from driver:Device"))


if __name__ == "__main__":
    unittest.main()
