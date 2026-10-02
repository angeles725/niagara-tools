import io
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

    def test_a_newer_protocol_version_is_not_echoed_until_its_semantics_exist(self):
        """Pins the documented supported set: newer revisions are answered with 2025-06-18."""
        self.assertEqual(server.SUPPORTED_PROTOCOL_VERSIONS,
                         ("2025-06-18", "2025-03-26", "2024-11-05"))
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "2025-11-25"}))
        self.assertEqual(res["result"]["protocolVersion"], "2025-06-18")

    def test_doc_contract_states_the_supported_set_and_the_naming_heuristic(self):
        """The one place that pins documentation wording (the operator-facing contract)."""
        self.assertIn("Only add a newer", server.__doc__)
        self.assertIn("Heuristic", box.is_component_type.__doc__)

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
        for name, args in (("n4_connect", {}), ("n4_connect", {"station": 5}),
                           ("n4_navigate", {"depth": 4}), ("n4_navigate", {"depth": "1"}),
                           ("n4_connect", {"station": "x", "expected_station": 7})):
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
        args = server.parse_args(["--station", "a=https://h1", "--station", "b=https://h2:8443",
                                  "--insecure-tls", "b", "--credential-env", "LAB"])
        self.assertEqual(args.station, ["a=https://h1", "b=https://h2:8443"])
        self.assertEqual((args.insecure_tls, args.credential_env), (["b"], "LAB"))
        self.assertEqual(server.parse_args([]).credential_env, "MCP_N4")
        args = server.parse_args([])
        self.assertFalse(args.allow_writes)
        self.assertFalse(args.allow_http_for_tests)


class TestRobustness(unittest.TestCase):
    def test_id_null_is_a_request_and_gets_a_reply(self):
        srv = server.Server()
        out = json.loads(srv.handle_line('{"jsonrpc":"2.0","id":null,"method":"ping"}'))
        self.assertEqual(out, {"jsonrpc": "2.0", "id": None, "result": {}})

    def test_message_without_id_is_still_a_notification(self):
        srv = server.Server()
        self.assertIsNone(srv.handle_line('{"jsonrpc":"2.0","method":"ping"}'))

    def test_validate_supports_number_and_array(self):
        schema = {"properties": {"n": {"type": "number", "minimum": 0},
                                 "a": {"type": "array"}}}
        server._validate(schema, {"n": 1.5, "a": [1]})
        server._validate(schema, {"n": 2})
        for args in ({"n": "x"}, {"n": True}, {"a": {}}, {"n": -1}):
            with self.assertRaises(server.InvalidParams, msg=args):
                server._validate(schema, args)

    def test_validate_fails_clearly_on_an_unknown_schema_type(self):
        with self.assertRaises(ValueError) as cm:
            server._validate({"properties": {"x": {"type": "float"}}}, {"x": 1})
        self.assertIn("float", str(cm.exception))

    def test_every_registered_tool_schema_uses_a_known_type(self):
        for tool in tools_read.TOOLS:
            for spec in tool.input_schema["properties"].values():
                self.assertIn(spec.get("type"), server._JSON_TYPES, tool.name)

    def test_serve_closes_the_session_when_the_loop_raises(self):
        srv = server.Server()
        closed = []
        srv.ctx.close = lambda: closed.append(1)

        def lines():
            yield '{"jsonrpc":"2.0","id":1,"method":"ping"}\n'
            raise RuntimeError("stream broke")
        with self.assertRaises(RuntimeError):
            srv.serve(lines(), io.StringIO())
        self.assertEqual(closed, [1])

    def test_serve_exits_cleanly_on_keyboard_interrupt_and_closes(self):
        srv = server.Server()
        closed = []
        srv.ctx.close = lambda: closed.append(1)

        def lines():
            raise KeyboardInterrupt
            yield
        srv.serve(lines(), io.StringIO())  # no exception
        self.assertEqual(closed, [1])

    def test_close_clears_the_session_even_when_the_client_close_raises(self):
        class Boom:
            def close(self):
                raise OSError("unreachable")
        ctx = tools_read.Context()
        ctx.session = tools_read.Session(Boom(), "S", "https://h", "2")
        with self.assertRaises(OSError):
            ctx.close()
        self.assertIsNone(ctx.session)

    def test_set_secret_scrubs_outputs(self):
        ctx = tools_read.Context()
        ctx.set_secret("hunter2-value")
        self.assertEqual(ctx.scrub("pw=hunter2-value!"), "pw=***!")

    def test_connect_survives_a_failing_close_of_the_old_session(self):
        class Boom:
            def close(self):
                raise OSError("unreachable")
        srv = server.Server(env={}, stations={"S": "https://h"})
        srv.ctx.session = tools_read.Session(Boom(), "S", "https://h", "2")
        res = srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                              "arguments": {"station": "S"}}))["result"]
        self.assertTrue(res["isError"])
        self.assertIsNone(srv.ctx.session)


class ToolTestCase(unittest.TestCase):
    PASSWORD = "s3cret-pw-value"

    def setUp(self):
        self.fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(self.fake.stop)
        self.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": self.PASSWORD}
        self.stations = {"FakeStation": self.fake.url}
        self.srv = server.Server(allow_http=True, env=self.env, stations=self.stations)
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
        return self.ok("n4_connect", station="FakeStation", **kw)


class TestConnect(ToolTestCase):
    def test_connect_returns_station_identity_and_mode(self):
        out = self.connect()
        self.assertEqual(out, {"station_name": "FakeStation", "base_url": self.fake.url,
                               "root_handle": "2", "mode": "read-only"})

    def test_connect_reports_writes_allowed_mode(self):
        self.srv = server.Server(allow_writes=True, allow_http=True, env=self.env,
                                 stations=self.stations)
        self.addCleanup(self.srv.ctx.close)
        self.assertEqual(self.connect()["mode"], "writes-allowed")

    def test_credentials_come_from_the_operator_configured_env_prefix(self):
        self.srv = server.Server(allow_http=True, stations=self.stations, credential_env="LAB",
                                 env={"LAB_USER": "admin", "LAB_PASSWORD": self.PASSWORD})
        self.addCleanup(self.srv.ctx.close)
        self.assertEqual(self.connect()["station_name"], "FakeStation")

    def test_credential_env_is_not_a_tool_argument(self):
        props = [t for t in tools_read.TOOLS if t.name == "n4_connect"][0].input_schema
        self.assertEqual(set(props["properties"]), {"station", "expected_station"})
        self.assertEqual(props["required"], ["station"])
        res = self.srv.dispatch(rpc("tools/call", {"name": "n4_connect", "arguments": {
            "station": "FakeStation", "base_url": "https://evil.example"}}))
        self.assertNotIn("error", res)  # unknown args are ignored, never used
        self.assertEqual(res["result"]["structuredContent"]["base_url"], self.fake.url)

    def test_missing_env_vars_name_the_variables_not_values(self):
        self.srv.ctx.env = {}
        text = self.err("n4_connect", station="FakeStation")
        self.assertIn("MCP_N4_USER", text)
        self.assertIn("MCP_N4_PASSWORD", text)
        self.assertEqual(self.fake.requests, 0)

    def test_wrong_password_is_an_error_without_leaking_the_secret(self):
        self.srv.ctx.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "wrong-pw-1234"}
        text = self.err("n4_connect", station="FakeStation")
        self.assertNotIn("wrong-pw-1234", text)
        self.assertEqual(self.fake.requests, 1)  # no login retry (B1179)
        self.assertIn("not connected", self.err("n4_navigate"))

    def test_http_station_is_refused_at_start_without_the_test_flag(self):
        with self.assertRaises(ValueError) as cm:
            server.Server(env=self.env, stations=self.stations)
        self.assertIn("https", str(cm.exception))
        self.assertEqual(self.fake.requests, 0)

    def test_unknown_station_name_is_refused_and_names_the_flag(self):
        text = self.err("n4_connect", station="other")
        self.assertIn("--station", text)
        self.assertIn("FakeStation", text)
        self.assertEqual(self.fake.requests, 0)

    def test_no_configured_station_is_refused_and_names_the_flag(self):
        self.srv = server.Server(allow_http=True, env=self.env)
        text = self.err("n4_connect", station="FakeStation")
        self.assertIn("--station NAME=URL", text)
        self.assertEqual(self.fake.requests, 0)

    def test_expected_station_defaults_to_the_station_name(self):
        out = self.connect()
        self.assertEqual(out["station_name"], "FakeStation")
        self.assertTrue(self.srv.ctx.session.identity_verified)

    def test_default_expected_name_mismatch_is_refused(self):
        self.srv = server.Server(allow_http=True, env=self.env,
                                 stations={"Other": self.fake.url})
        self.addCleanup(self.srv.ctx.close)
        text = self.err("n4_connect", station="Other")
        self.assertIn("Other", text)
        self.assertIn("FakeStation", text)
        self.assertIsNone(self.srv.ctx.session)

    def test_tls_is_verified_unless_the_operator_marks_the_station_insecure(self):
        seen = []

        def factory(url, user, secret, insecure_tls=False, allow_http=False):
            seen.append(insecure_tls)
            return box.BoxClient(url, user, secret, insecure_tls=insecure_tls,
                                 allow_http=allow_http)
        for insecure, want in (((), False), (("FakeStation",), True)):
            srv = server.Server(allow_http=True, env=self.env, stations=self.stations,
                                insecure_tls=insecure, client_factory=factory)
            self.addCleanup(srv.ctx.close)
            res = srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                                  "arguments": {"station": "FakeStation"}}))
            self.assertNotIn("error", res)
            self.assertEqual(seen[-1], want)

    def test_insecure_tls_for_an_unconfigured_station_is_a_start_error(self):
        with self.assertRaises(ValueError):
            server.Server(allow_http=True, stations=self.stations, insecure_tls=("ghost",))

    def test_failed_connect_on_unknown_station_closes_the_old_session(self):
        self.connect()
        self.err("n4_connect", station="ghost")
        self.assertIsNone(self.srv.ctx.session)
        self.assertEqual(self.fake.sessions, set())

    def test_failed_connect_on_missing_credentials_closes_the_old_session(self):
        self.connect()
        self.srv.ctx.env = {}
        self.err("n4_connect", station="FakeStation")
        self.assertIsNone(self.srv.ctx.session)
        self.assertEqual(self.fake.sessions, set())
        self.assertIn("not connected", self.err("n4_navigate"))

    def test_expected_station_match_connects(self):
        self.assertEqual(self.connect(expected_station="FakeStation")["station_name"],
                         "FakeStation")

    def test_expected_station_mismatch_closes_session_and_names_both(self):
        text = self.err("n4_connect", station="FakeStation", expected_station="OtherStation")
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
        self.err("n4_connect", station="FakeStation")
        self.assertIsNone(self.srv.ctx.session)


class TestDescribeSession(ToolTestCase):
    def test_describe_before_connect_reports_disconnected(self):
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": False, "station_name": None, "base_url": None,
                               "mode": "read-only", "server_version": server.SERVER_VERSION,
                               "configured_stations": ["FakeStation"]})

    def test_describe_after_connect(self):
        self.connect()
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": True, "station_name": "FakeStation",
                               "base_url": self.fake.url, "mode": "read-only",
                               "server_version": server.SERVER_VERSION,
                               "configured_stations": ["FakeStation"]})

    def test_describe_lists_only_names_never_urls_or_credentials(self):
        self.srv = server.Server(allow_http=True, env=self.env,
                                 stations={"b": "http://user:pw@h/", "a": "http://h2/"})
        self.addCleanup(self.srv.ctx.close)
        out = self.ok("n4_describe_session")
        self.assertEqual(out["configured_stations"], ["a", "b"])
        self.assertNotIn("pw", json.dumps(out))


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

    def test_complex_slot_without_a_value_falls_back_to_its_display_string(self):
        nodes = {"": {"t": "modbusCore:ModbusClientNumericProxyExt"},
                 "dataAddress": {"n": "dataAddress", "t": "modbusCore:FlexAddress",
                                 "d": "Decimal:302"},
                 "dataAddress/address": {"n": "address", "t": "baja:String", "v": "302",
                                         "d": "302"},
                 "address": {"n": "address", "t": "bacnet:BacnetAddress",
                             "d": "1:<device-ip>:47808"}}
        flex = tools_read._slot_entry(nodes, "dataAddress")
        self.assertEqual((flex["value"], flex["value_display"]), ("Decimal:302", "Decimal:302"))
        self.assertEqual(tools_read._slot_entry(nodes, "address")["value"], "1:<device-ip>:47808")

    def test_a_decoded_value_is_kept_and_its_display_rides_along(self):
        nodes = {"conversion": {"t": "driver:LinearConversion", "v": "0.1;0.0",
                                "d": "Linear *0.10"}}
        entry = tools_read._slot_entry(nodes, "conversion")
        self.assertEqual((entry["value"], entry["value_display"]), ("0.1;0.0", "Linear *0.10"))

    def test_read_slots_and_navigate_report_elapsed_ms(self):
        nn, _ = self.add_with_out("Calc")
        ticks = iter([10.0, 10.25, 20.0, 20.5])
        self.srv.ctx.clock = lambda: next(ticks)
        out = self.ok("n4_read_slots", ord="station:|slot:/Folder/" + nn)
        self.assertEqual(out["elapsed_ms"], 250)
        self.assertEqual(self.ok("n4_navigate")["elapsed_ms"], 500)

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


class TestDeepTargets(StationTestCase):
    """A link whose target is deeper than `depth` must not cause false positives."""

    def setUp(self):
        super().setUp()
        self.a = self.add_with_out("A")
        self.c = self.add("C")
        nn = self.box.add_component(self.c[1], "B", "kitControl:NumericConst")["nn"]
        tree = self.box.load_tree("station:|slot:/Folder/" + self.c[0], depth=1, **NO_SLEEP)
        self.b = (nn, tree[nn]["h"])
        self.box.check_links(self.a[1], "out", self.b[1], "in10", add=True)

    def test_output_feeding_a_deeper_target_is_not_dangling(self):
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder", depth=1)
        self.assertNotIn("/Folder/" + self.a[0], [d["path"] for d in out["dangling"]])

    def test_source_in_the_reported_range_resolves_for_a_deeper_target(self):
        out = self.ok("n4_list_links", ord="station:|slot:/Folder", depth=2)
        self.assertEqual([(l["source_path"], l["target_path"]) for l in out["links"]],
                         [("/Folder/" + self.a[0], "/Folder/%s/%s" % (self.c[0], self.b[0]))])

    def test_list_links_reports_only_links_of_components_within_depth(self):
        self.assertEqual(self.ok("n4_list_links", ord="station:|slot:/Folder", depth=1)["links"],
                         [])

    def test_scope_note_states_what_is_and_is_not_seen(self):
        note = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")["scope_note"]
        self.assertIn("one level below depth", note)
        self.assertIn("outside the subtree", note)


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
            call(3, "n4_connect", station="FakeStation"),
            call(4, "n4_navigate"),
            call(5, "n4_describe_session"),
        ], ["--station", "FakeStation=" + fake.url, "--allow-http-for-tests"])
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual([r["id"] for r in replies], [1, 2, 3, 4, 5])
        self.assertEqual(replies[0]["result"]["serverInfo"]["name"], "mcp-n4")
        self.assertEqual(len(replies[1]["result"]["tools"]), 9)  # 7 + n4_bql_query + n4_inventory
        self.assertEqual(replies[2]["result"]["structuredContent"]["station_name"], "FakeStation")
        names = [c["name"] for c in replies[3]["result"]["structuredContent"]["children"]]
        self.assertIn("Folder", names)
        self.assertTrue(replies[4]["result"]["structuredContent"]["connected"])
        self.assertEqual(fake.sessions, set())  # closed when stdin ended
        self.assertNotIn(self.PASSWORD, proc.stdout + proc.stderr)

    def test_http_station_is_refused_without_the_flag_over_stdio(self):
        fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(fake.stop)
        proc, replies = self.run_server([], ["--station", "FakeStation=" + fake.url])
        self.assertNotEqual(proc.returncode, 0)
        self.assertEqual(replies, [])
        self.assertIn("https", proc.stderr)
        self.assertEqual(fake.requests, 0)

    def test_connect_without_a_configured_station_is_refused_over_stdio(self):
        proc, replies = self.run_server([rpc("tools/call", {
            "name": "n4_connect", "arguments": {"station": "FakeStation"}}, 1)])
        self.assertTrue(replies[0]["result"]["isError"])
        self.assertIn("--station", replies[0]["result"]["content"][0]["text"])


if __name__ == "__main__":
    unittest.main()
