import json
import os
import subprocess
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box, server, tools_read  # noqa: E402

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
NO_SLEEP = dict(sleep=lambda s: None)


def rpc(method, params=None, id_=1):
    msg = {"jsonrpc": "2.0", "method": method}
    if id_ is not None:
        msg["id"] = id_
    if params is not None:
        msg["params"] = params
    return msg


class TestProtocol(unittest.TestCase):
    def setUp(self):
        self.srv = server.Server()

    def test_initialize_reports_server_info_and_capabilities(self):
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "2025-03-26"}))
        self.assertEqual(res["id"], 1)
        result = res["result"]
        self.assertEqual(result["serverInfo"], {"name": "mcp-n4", "version": server.SERVER_VERSION})
        self.assertEqual(result["capabilities"], {"tools": {"listChanged": False}})
        self.assertEqual(result["protocolVersion"], "2025-03-26")

    def test_initialize_falls_back_to_newest_supported_version(self):
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "1999-01-01"}))
        self.assertEqual(res["result"]["protocolVersion"], server.SUPPORTED_PROTOCOL_VERSIONS[0])
        res = self.srv.dispatch(rpc("initialize"))
        self.assertEqual(res["result"]["protocolVersion"], server.SUPPORTED_PROTOCOL_VERSIONS[0])

    def test_initialized_notification_gets_no_reply(self):
        self.assertIsNone(self.srv.dispatch(rpc("notifications/initialized", id_=None)))

    def test_ping_returns_empty_result(self):
        self.assertEqual(self.srv.dispatch(rpc("ping"))["result"], {})

    def test_unknown_method_is_32601_and_unknown_notification_is_silent(self):
        res = self.srv.dispatch(rpc("nope"))
        self.assertEqual(res["error"]["code"], -32601)
        self.assertIsNone(self.srv.dispatch(rpc("nope", id_=None)))

    def test_parse_error_is_32700_with_null_id(self):
        out = json.loads(self.srv.handle_line("{not json"))
        self.assertEqual(out["error"]["code"], -32700)
        self.assertIsNone(out["id"])

    def test_non_object_message_is_invalid_request(self):
        out = json.loads(self.srv.handle_line("[1, 2]"))
        self.assertEqual(out["error"]["code"], -32600)

    def test_blank_line_is_ignored(self):
        self.assertIsNone(self.srv.handle_line("   "))

    def test_tools_list_entries_have_schema_and_read_only_annotations(self):
        tools = self.srv.dispatch(rpc("tools/list"))["result"]["tools"]
        self.assertEqual([t["name"] for t in tools], [t.name for t in tools_read.TOOLS])
        for tool in tools:
            self.assertEqual(tool["inputSchema"]["type"], "object")
            self.assertEqual(tool["annotations"], {"readOnlyHint": True, "openWorldHint": False})
            self.assertTrue(tool["description"])

    def test_tools_call_rejects_bad_params_with_32602(self):
        for params in (None, {}, {"name": 5}, {"name": "ghost"},
                       {"name": "n4_navigate", "arguments": []}):
            res = self.srv.dispatch(rpc("tools/call", params))
            self.assertEqual(res["error"]["code"], -32602, params)

    def test_tools_call_validates_required_type_and_range(self):
        for name, args in (("n4_connect", {}), ("n4_connect", {"base_url": 5}),
                           ("n4_navigate", {"depth": 4}), ("n4_navigate", {"depth": "1"}),
                           ("n4_connect", {"base_url": "x", "insecure_tls": "yes"})):
            res = self.srv.dispatch(rpc("tools/call", {"name": name, "arguments": args}))
            self.assertEqual(res["error"]["code"], -32602, (name, args))

    def test_tool_needing_a_session_reports_not_connected(self):
        res = self.srv.dispatch(rpc("tools/call", {"name": "n4_navigate", "arguments": {}}))
        self.assertTrue(res["result"]["isError"])
        self.assertIn("not connected", res["result"]["content"][0]["text"])

    def test_tool_result_carries_text_and_structured_content(self):
        tool = tools_read.Tool("echo", "d", {"type": "object", "properties": {}},
                               lambda ctx, args: {"a": 1}, needs_session=False)
        res = server.Server(tools=[tool]).dispatch(
            rpc("tools/call", {"name": "echo", "arguments": {}}))["result"]
        self.assertEqual(res["structuredContent"], {"a": 1})
        self.assertEqual(res["content"], [{"type": "text", "text": json.dumps({"a": 1})}])
        self.assertFalse(res.get("isError", False))

    def test_unexpected_tool_exception_is_short_error_without_message_or_trace(self):
        def boom(ctx, args):
            raise RuntimeError("topsecret-value in /home/x/file.py")
        tool = tools_read.Tool("boom", "d", {"type": "object", "properties": {}}, boom,
                               needs_session=False)
        res = server.Server(tools=[tool]).dispatch(
            rpc("tools/call", {"name": "boom", "arguments": {}}))["result"]
        text = res["content"][0]["text"]
        self.assertTrue(res["isError"])
        self.assertIn("RuntimeError", text)
        self.assertNotIn("topsecret", text)
        self.assertNotIn("Traceback", text)

    def test_tool_error_and_box_error_messages_are_shown(self):
        for exc in (tools_read.ToolError("clear reason"), box.BoxError("clear reason")):
            def boom(ctx, args, exc=exc):
                raise exc
            tool = tools_read.Tool("boom", "d", {"type": "object", "properties": {}}, boom,
                                   needs_session=False)
            res = server.Server(tools=[tool]).dispatch(
                rpc("tools/call", {"name": "boom", "arguments": {}}))["result"]
            self.assertTrue(res["isError"])
            self.assertIn("clear reason", res["content"][0]["text"])

    def test_flags_record_mode(self):
        self.assertEqual(server.Server().ctx.mode, "read-only")
        self.assertEqual(server.Server(allow_writes=True).ctx.mode, "writes-allowed")
        self.assertFalse(server.Server().ctx.allow_http)

    def test_main_parses_flags_without_serving_at_import(self):
        args = server.parse_args(["--allow-writes", "--allow-http-for-tests"])
        self.assertTrue(args.allow_writes)
        self.assertTrue(args.allow_http_for_tests)
        args = server.parse_args([])
        self.assertFalse(args.allow_writes)
        self.assertFalse(args.allow_http_for_tests)


class ToolTestCase(unittest.TestCase):
    PASSWORD = "s3cret-pw-value"

    def setUp(self):
        self.fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(self.fake.stop)
        self.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": self.PASSWORD}
        self.srv = server.Server(allow_http=True, env=self.env)
        self.addCleanup(self.srv.ctx.close)

    def call(self, name, **args):
        res = self.srv.dispatch(rpc("tools/call", {"name": name, "arguments": args}))
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

    def connect(self, **kw):
        return self.ok("n4_connect", base_url=self.fake.url, **kw)


class TestConnect(ToolTestCase):
    def test_connect_returns_station_identity_and_mode(self):
        out = self.connect()
        self.assertEqual(out, {"station_name": "FakeStation", "base_url": self.fake.url,
                               "root_handle": "2", "mode": "read-only"})

    def test_connect_reports_writes_allowed_mode(self):
        self.srv = server.Server(allow_writes=True, allow_http=True, env=self.env)
        self.addCleanup(self.srv.ctx.close)
        self.assertEqual(self.connect()["mode"], "writes-allowed")

    def test_connect_reads_credentials_from_custom_env_prefix(self):
        self.srv.ctx.env = {"LAB_USER": "admin", "LAB_PASSWORD": self.PASSWORD}
        self.assertEqual(self.connect(credential_env="LAB")["station_name"], "FakeStation")

    def test_missing_env_vars_name_the_variables_not_values(self):
        self.srv.ctx.env = {}
        text = self.err("n4_connect", base_url=self.fake.url)
        self.assertIn("MCP_N4_USER", text)
        self.assertIn("MCP_N4_PASSWORD", text)
        self.assertEqual(self.fake.requests, 0)

    def test_wrong_password_is_an_error_without_leaking_the_secret(self):
        self.srv.ctx.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "wrong-pw-1234"}
        text = self.err("n4_connect", base_url=self.fake.url)
        self.assertNotIn("wrong-pw-1234", text)
        self.assertEqual(self.fake.requests, 1)  # no login retry (B1179)
        self.assertIn("not connected", self.err("n4_navigate"))

    def test_http_is_refused_without_the_test_flag(self):
        self.srv = server.Server(env=self.env)
        text = self.err("n4_connect", base_url=self.fake.url)
        self.assertIn("http", text)
        self.assertNotIn(self.PASSWORD, text)
        self.assertEqual(self.fake.requests, 0)

    def test_expected_station_match_connects(self):
        self.assertEqual(self.connect(expected_station="FakeStation")["station_name"],
                         "FakeStation")

    def test_expected_station_mismatch_closes_session_and_names_both(self):
        text = self.err("n4_connect", base_url=self.fake.url, expected_station="OtherStation")
        self.assertIn("OtherStation", text)
        self.assertIn("FakeStation", text)
        self.assertIsNone(self.srv.ctx.session)
        self.assertEqual(self.fake.sessions, set())
        self.assertIn("not connected", self.err("n4_navigate"))

    def test_reconnect_closes_the_previous_session(self):
        self.connect()
        first = self.srv.ctx.session.client
        self.connect()
        self.assertIsNone(first.sid)
        self.assertIsNot(self.srv.ctx.session.client, first)

    def test_failed_reconnect_leaves_no_session(self):
        self.connect()
        self.srv.ctx.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "wrong-pw-1234"}
        self.err("n4_connect", base_url=self.fake.url)
        self.assertIsNone(self.srv.ctx.session)


class TestDescribeSession(ToolTestCase):
    def test_describe_before_connect_reports_disconnected(self):
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": False, "station_name": None, "base_url": None,
                               "mode": "read-only", "server_version": server.SERVER_VERSION})

    def test_describe_after_connect(self):
        self.connect()
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": True, "station_name": "FakeStation",
                               "base_url": self.fake.url, "mode": "read-only",
                               "server_version": server.SERVER_VERSION})


class StationTestCase(ToolTestCase):
    """Connected session plus helpers to build components under /Folder."""

    def setUp(self):
        super().setUp()
        self.connect()
        self.box = self.srv.ctx.session.client

    def add(self, name, type_="kitControl:NumericConst"):
        nn = self.box.add_component("3", name, type_)["nn"]
        return nn, self.box.load_tree("station:|slot:/Folder", depth=2, **NO_SLEEP)[nn]["h"]

    def add_with_out(self, name, value=1.0):
        nn, h = self.add(name)
        self.box.set_slot(h, "out", box.bson_status_numeric(value))
        return nn, h


class TestNavigate(StationTestCase):
    def test_default_lists_root_children(self):
        out = self.ok("n4_navigate")
        self.assertIn({"name": "Folder", "type": "baja:Folder", "handle": "3",
                       "has_children": False}, out["children"])

    def test_has_children_true_when_child_holds_components(self):
        self.add("Pump")
        out = self.ok("n4_navigate")
        folder = [c for c in out["children"] if c["name"] == "Folder"][0]
        self.assertTrue(folder["has_children"])

    def test_status_slot_internals_are_not_component_children(self):
        nn, _ = self.add_with_out("Calc")
        out = self.ok("n4_navigate", ord="station:|slot:/Folder")
        calc = [c for c in out["children"] if c["name"] == nn][0]
        self.assertFalse(calc["has_children"])

    def test_navigate_into_a_folder(self):
        nn, h = self.add("Pump")
        out = self.ok("n4_navigate", ord="station:|slot:/Folder")
        self.assertEqual(out["ord"], "station:|slot:/Folder")
        self.assertEqual([(c["name"], c["type"], c["handle"]) for c in out["children"]],
                         [(nn, "kitControl:NumericConst", h)])

    def test_depth_two_nests_grandchildren(self):
        nn, _ = self.add("Pump")
        out = self.ok("n4_navigate", depth=2)
        folder = [c for c in out["children"] if c["name"] == "Folder"][0]
        self.assertEqual([c["name"] for c in folder["children"]], [nn])

    def test_depth_one_has_no_nested_children_key(self):
        self.add("Pump")
        for child in self.ok("n4_navigate")["children"]:
            self.assertNotIn("children", child)

    def test_bad_ord_is_an_error(self):
        self.assertIn("ord", self.err("n4_navigate", ord="/Folder"))

    def test_unknown_node_is_an_error(self):
        self.err("n4_navigate", ord="station:|slot:/Missing")


class TestReadSlots(StationTestCase):
    def slots(self, nn):
        out = self.ok("n4_read_slots", ord="station:|slot:/Folder/" + nn)
        return {slot["name"]: slot for slot in out["slots"]}

    def test_status_slot_is_rendered_with_value_status_and_flags(self):
        nn, _ = self.add_with_out("Calc", 5.5)
        out = self.slots(nn)["out"]
        self.assertEqual(out, {"name": "out", "type": "baja:StatusNumeric", "value": "5.5",
                               "status": "0", "status_ok": True, "status_null": False})

    def test_omitted_value_takes_the_type_default(self):
        nn, _ = self.add_with_out("Calc", 0)  # the station omits value 0.0
        self.assertEqual(self.slots(nn)["out"]["value"], "0.0")

    def test_boolean_default_and_null_status(self):
        nn, h = self.add("Sw")
        self.box.set_slot(h, "out", box.bson_status_boolean(False, status="40"))
        out = self.slots(nn)["out"]
        self.assertEqual((out["value"], out["status"]), ("false", "40"))
        self.assertTrue(out["status_null"])
        self.assertFalse(out["status_ok"])

    def test_plain_slot_has_raw_value_and_no_status(self):
        nn, h = self.add("Calc")
        self.box.set_slot(h, "label", {"nm": "p", "t": "baja:String", "v": "hello"})
        slot = self.slots(nn)["label"]
        self.assertEqual(slot, {"name": "label", "type": "baja:String", "value": "hello",
                                "status": None})

    def test_unsupported_status_type_falls_back_without_error(self):
        nn, h = self.add("Calc")
        self.box.set_slot(h, "txt", {"nm": "p", "t": "baja:StatusString", "s": [
            {"nm": "p", "n": "value", "t": "baja:String", "v": "abc"}]})
        slot = self.slots(nn)["txt"]
        self.assertEqual((slot["value"], slot["status"]), ("abc", "0"))

    def test_ord_is_required(self):
        res = self.srv.dispatch(rpc("tools/call", {"name": "n4_read_slots", "arguments": {}}))
        self.assertEqual(res["error"]["code"], -32602)

    def test_response_names_the_ord(self):
        nn, _ = self.add_with_out("Calc")
        ord_str = "station:|slot:/Folder/" + nn
        self.assertEqual(self.ok("n4_read_slots", ord=ord_str)["ord"], ord_str)


class TestListLinks(StationTestCase):
    def link(self, src, tgt, src_slot="out", tgt_slot="in10"):
        self.box.check_links(src[1], src_slot, tgt[1], tgt_slot, add=True)

    def test_lists_link_with_resolved_source_path(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.link(a, b)
        out = self.ok("n4_list_links", ord="station:|slot:/Folder")
        self.assertEqual(out["links"], [{
            "target_path": "/Folder/" + b[0], "link_name": "Link",
            "source_ord": "h:" + a[1], "source_path": "/Folder/" + a[0],
            "source_slot": "out", "target_slot": "in10"}])

    def test_source_outside_the_loaded_subtree_has_null_path(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.link(a, b)
        out = self.ok("n4_list_links", ord="station:|slot:/Folder/" + b[0], depth=1)
        self.assertEqual(len(out["links"]), 1)
        self.assertIsNone(out["links"][0]["source_path"])
        self.assertEqual(out["links"][0]["source_ord"], "h:" + a[1])

    def test_no_links_gives_empty_list(self):
        self.add_with_out("A")
        self.assertEqual(self.ok("n4_list_links", ord="station:|slot:/Folder")["links"], [])

    def test_conversion_links_are_included(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.link(a, b)
        ln = [n for n in self.box.load_tree("station:|slot:/Folder/" + b[0], depth=2,
                                            **NO_SLEEP) if n.startswith("Link")][0]
        self.fake.by_handle[b[1]].child(ln).type = "baja:ConversionLink"
        out = self.ok("n4_list_links", ord="station:|slot:/Folder")
        self.assertEqual(len(out["links"]), 1)

    def test_depth_limits_how_deep_components_are_scanned(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.link(a, b)
        # from the root, B sits two levels down: depth 1 does not scan it, depth 2 does
        self.assertEqual(self.ok("n4_list_links", ord="station:|slot:/", depth=1)["links"], [])
        self.assertEqual(len(self.ok("n4_list_links", ord="station:|slot:/", depth=2)["links"]), 1)


class TestDanglingOutputs(StationTestCase):
    def test_reports_components_whose_out_is_not_a_link_source(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.box.check_links(a[1], "out", b[1], "in10", add=True)
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        self.assertEqual(out["dangling"], [{"path": "/Folder/" + b[0],
                                            "type": "kitControl:NumericConst"}])

    def test_scope_note_warns_about_links_from_outside(self):
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        self.assertEqual(out["dangling"], [])
        self.assertIn("outside the subtree", out["scope_note"])

    def test_component_without_out_slot_is_not_reported(self):
        self.add("NoOut")
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        self.assertEqual(out["dangling"], [])

    def test_link_from_another_slot_does_not_cover_out(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.box.set_slot(a[1], "alt", box.bson_status_numeric(1))
        self.box.check_links(a[1], "alt", b[1], "in10", add=True)
        paths = [d["path"] for d in self.ok("n4_find_dangling_outputs",
                                            ord="station:|slot:/Folder")["dangling"]]
        self.assertEqual(sorted(paths), sorted(["/Folder/" + a[0], "/Folder/" + b[0]]))


class TestStdioEndToEnd(unittest.TestCase):
    PASSWORD = "e2e-pw-value-77"

    def run_server(self, lines, args=()):
        env = dict(os.environ, MCP_N4_USER="admin", MCP_N4_PASSWORD=self.PASSWORD)
        proc = subprocess.run(
            [sys.executable, "-m", "mcp_n4.server"] + list(args), cwd=KIT_DIR, env=env,
            input="".join(json.dumps(m) + "\n" for m in lines), capture_output=True,
            text=True, timeout=60)
        return proc, [json.loads(line) for line in proc.stdout.splitlines()]

    def test_import_prints_nothing(self):
        proc = subprocess.run([sys.executable, "-c", "import mcp_n4.server"], cwd=KIT_DIR,
                              capture_output=True, text=True, timeout=60)
        self.assertEqual((proc.returncode, proc.stdout, proc.stderr), (0, "", ""))

    def test_session_over_stdio_against_the_fake_station(self):
        fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(fake.stop)
        call = lambda i, name, **a: rpc("tools/call", {"name": name, "arguments": a}, i)
        proc, replies = self.run_server([
            rpc("initialize", {"protocolVersion": "2025-06-18"}, 1),
            rpc("notifications/initialized", id_=None),
            rpc("tools/list", id_=2),
            call(3, "n4_connect", base_url=fake.url),
            call(4, "n4_navigate"),
            call(5, "n4_describe_session"),
        ], ["--allow-http-for-tests"])
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual([r["id"] for r in replies], [1, 2, 3, 4, 5])
        self.assertEqual(replies[0]["result"]["serverInfo"]["name"], "mcp-n4")
        self.assertEqual(len(replies[1]["result"]["tools"]), 6)
        self.assertEqual(replies[2]["result"]["structuredContent"]["station_name"], "FakeStation")
        names = [c["name"] for c in replies[3]["result"]["structuredContent"]["children"]]
        self.assertIn("Folder", names)
        self.assertTrue(replies[4]["result"]["structuredContent"]["connected"])
        self.assertEqual(fake.sessions, set())  # closed when stdin ended
        self.assertNotIn(self.PASSWORD, proc.stdout + proc.stderr)

    def test_http_is_refused_without_the_flag_over_stdio(self):
        fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(fake.stop)
        proc, replies = self.run_server([rpc("tools/call", {
            "name": "n4_connect", "arguments": {"base_url": fake.url}}, 1)])
        self.assertTrue(replies[0]["result"]["isError"])
        self.assertEqual(fake.requests, 0)


if __name__ == "__main__":
    unittest.main()
