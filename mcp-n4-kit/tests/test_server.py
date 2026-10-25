import contextlib
import json
import os
import signal
import subprocess
import sys
import threading
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
import oracle  # noqa: E402
from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box, server  # noqa: E402

KIT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
NO_SLEEP = dict(sleep=lambda s: None)
#: What the server under test reports and enforces. Each literal is checked against the binary's
#: own output below, so it cannot drift silently: the version and the protocol set against
#: `initialize`, the nesting limit by the two tests that probe it, the tool lists against
#: `tools/list` (read-only here, writes-allowed in test_tiers).
SERVER_VERSION = "0.9.5"
SUPPORTED_PROTOCOL_VERSIONS = ("2025-06-18", "2025-03-26", "2024-11-05")
MAX_NESTING = 1000
READ_TOOLS = ["n4_connect", "n4_describe_session", "n4_navigate", "n4_read_slots",
              "n4_list_links", "n4_find_dangling_outputs", "n4_bql_query", "n4_inventory",
              "n4_read_file", "n4_list_px_views", "n4_px_summary", "n4_list_batches",
              "n4_session_retro_draft"]
WRITE_TOOLS = ["n4_create_component", "n4_create_link", "n4_invoke_action",
               "n4_remove_component", "n4_rollback", "n4_save_station", "n4_set_slot"]


def rpc(method, params=None, id_=1):
    msg = {"jsonrpc": "2.0", "method": method}
    if id_ is not None:
        msg["id"] = id_
    if params is not None:
        msg["params"] = params
    return msg


class TestProtocol(unittest.TestCase):
    def setUp(self):
        self.srv = go_server.make_server()
        self.addCleanup(go_server.close_server, self.srv)

    def test_initialize_reports_server_info_and_capabilities(self):
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "2025-03-26"}))
        self.assertEqual(res["id"], 1)
        result = res["result"]
        self.assertEqual(result["serverInfo"], {"name": "mcp-n4", "version": SERVER_VERSION})
        self.assertEqual(result["capabilities"], {"tools": {"listChanged": False}})
        self.assertEqual(result["protocolVersion"], "2025-03-26")

    def test_initialize_falls_back_to_newest_supported_version(self):
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "1999-01-01"}))
        self.assertEqual(res["result"]["protocolVersion"], SUPPORTED_PROTOCOL_VERSIONS[0])
        res = self.srv.dispatch(rpc("initialize"))
        self.assertEqual(res["result"]["protocolVersion"], SUPPORTED_PROTOCOL_VERSIONS[0])

    def test_a_newer_protocol_version_is_not_echoed_until_its_semantics_exist(self):
        """Pins the documented supported set: newer revisions are answered with 2025-06-18."""
        for version in SUPPORTED_PROTOCOL_VERSIONS:  # every supported revision is echoed
            res = self.srv.dispatch(rpc("initialize", {"protocolVersion": version}))
            self.assertEqual(res["result"]["protocolVersion"], version)
        res = self.srv.dispatch(rpc("initialize", {"protocolVersion": "2025-11-25"}))
        self.assertEqual(res["result"]["protocolVersion"], "2025-06-18")

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
        self.assertEqual([t["name"] for t in tools], READ_TOOLS)
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


class TestRobustness(unittest.TestCase):
    def make_server(self):
        srv = go_server.make_server()
        self.addCleanup(go_server.close_server, srv)
        return srv

    def test_id_null_is_a_request_and_gets_a_reply(self):
        srv = self.make_server()
        out = json.loads(srv.handle_line('{"jsonrpc":"2.0","id":null,"method":"ping"}'))
        self.assertEqual(out, {"jsonrpc": "2.0", "id": None, "result": {}})

    def test_message_without_id_is_still_a_notification(self):
        srv = self.make_server()
        self.assertIsNone(srv.handle_line('{"jsonrpc":"2.0","method":"ping"}'))

    def test_replies_are_the_bytes_the_python_server_writes(self):
        """tools/list (every schema), errors and odd ids, compared byte for byte with what the
        in-process Python server wrote (plan slice S2), frozen in fixtures/oracles (S17)."""
        srv, frozen = self.make_server(), oracle.Frozen(self, "protocol")
        reference = None
        if oracle.RECORD:
            reference = server.Server()
        for line in ('{"jsonrpc":"2.0","id":2,"method":"tools/list"}',
                     '{"id":3,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}',
                     '{"id":[1.0,"\\u00e9",{"k":null}],"method":"nope"}',
                     '{"id":4,"method":"tools/call","params":{"name":"n4_set_slot"}}',
                     '{"id":5,"method":"tools/call","params":{"name":"n4_navigate",'
                     '"arguments":{"depth":0}}}',
                     '{"id":6,"method":"tools/call","params":{"name":"n4_navigate"}}',
                     '\u00a0{"id":7,"method":"ping"}\x0b', '{bad', '7'):
            want = frozen(line, lambda: reference.handle_line(line))
            self.assertEqual(srv.handle_line(line), want, line)

    @staticmethod
    def nested_id(depth):
        """A ping whose request nests `depth` arrays/objects deep (the outer object counts)."""
        inner = depth - 1
        return '{"id":%s1%s,"method":"ping"}' % ("[" * inner, "]" * inner)

    def test_nesting_up_to_the_limit_is_answered(self):
        srv = self.make_server()
        out = json.loads(srv.handle_line(self.nested_id(MAX_NESTING)))
        self.assertEqual(out["result"], {})
        self.assertEqual(json.dumps(out["id"]).count("["), MAX_NESTING - 1)

    def test_deeper_nesting_is_a_parse_error_and_the_server_keeps_serving(self):
        """R8-D15: CPython's json raised RecursionError (not ValueError) and killed serve."""
        srv = self.make_server()
        for depth in (MAX_NESTING + 1, 200000):
            for line in (self.nested_id(depth), '{"a":' * depth + "1" + "}" * depth):
                out = json.loads(srv.handle_line(line))
                self.assertEqual(out, {"jsonrpc": "2.0", "id": None,
                                       "error": {"code": -32700, "message": "parse error"}})
        self.assertEqual(srv.dispatch(rpc("ping"))["result"], {})

    def test_whitespace_around_a_line_is_python_str_strip(self):
        srv = self.make_server()
        self.assertIsNone(srv.handle_line("\u3000\x1c\x0c"))
        out = json.loads(srv.handle_line('\u2028{"id":1,"method":"ping"}\x85'))
        self.assertEqual(out["result"], {})


class TestShutdown(unittest.TestCase):
    """Black-box: the Go server process deletes the station's BOX session however it is stopped
    (docs/go-port-plan.md section 22)."""
    PASSWORD = "s3cret-pw-value"
    TIMEOUT = 15

    def setUp(self):
        self.fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(self.fake.stop)
        cmd = [go_server.resolve_binary()]
        env = dict(os.environ, MCP_N4_USER="admin", MCP_N4_PASSWORD=self.PASSWORD)
        self.proc = subprocess.Popen(
            cmd + go_server.server_args(allow_http=True, stations={"FakeStation": self.fake.url}),
            cwd=KIT_DIR, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE)
        self.addCleanup(self.reap)

    def reap(self):
        if self.proc.poll() is None:  # only the child this test started
            self.proc.kill()
        self.proc.wait()
        for stream in (self.proc.stdin, self.proc.stdout, self.proc.stderr):
            with contextlib.suppress(OSError):
                stream.close()

    def send(self, msg):
        self.proc.stdin.write((json.dumps(msg) + "\n").encode())
        self.proc.stdin.flush()

    def connect(self):
        self.send(rpc("tools/call", {"name": "n4_connect", "arguments": {"station": "FakeStation"}}))

    def stopped_by(self, sig):
        """Send sig with stdin still open (EOF alone closes the session too): exit code, the
        output after it, and the station's sessions."""
        self.proc.send_signal(sig)
        code = self.proc.wait(self.TIMEOUT)
        return code, self.proc.stdout.read(), self.proc.stderr.read(), self.fake.sessions

    def assert_signal_closes_the_session(self, sig):
        self.connect()
        self.assertIn(b'"station_name"', self.proc.stdout.readline())
        self.assertEqual(self.fake.sessions, {"sess-1"})
        self.assertEqual(self.stopped_by(sig), (0, b"", b"", set()))

    def test_sigint_deletes_the_station_session_and_exits_zero(self):
        self.assert_signal_closes_the_session(signal.SIGINT)

    def test_sigterm_deletes_the_station_session_and_exits_zero(self):
        self.assert_signal_closes_the_session(signal.SIGTERM)

    def test_sigint_after_make_still_deletes_the_station_session(self):
        reached, release = threading.Event(), threading.Event()
        self.addCleanup(release.set)

        def hook(frame):
            if frame["m"][0]["k"] == "makessc":
                reached.set()
                release.wait(self.TIMEOUT)
        self.fake.hook = hook
        self.connect()
        self.assertTrue(reached.wait(self.TIMEOUT))
        self.assertEqual(self.fake.sessions, {"sess-1"})
        self.assertEqual(self.stopped_by(signal.SIGINT), (0, b"", b"", set()))

    def test_a_client_gone_mid_reply_still_deletes_the_station_session(self):
        self.connect()
        self.proc.stdout.readline()
        self.proc.stdout.close()
        with contextlib.suppress(BrokenPipeError):
            self.send(rpc("ping"))  # its reply cannot be written: EPIPE, not a SIGPIPE death
        code = self.proc.wait(self.TIMEOUT)
        self.assertGreater(code, 0)  # an exit status (Python 120, Go 1), never a signal
        self.assertEqual(self.fake.sessions, set())


class ToolTestCase(unittest.TestCase):
    PASSWORD = "s3cret-pw-value"

    def setUp(self):
        self.fake = FakeStation(password=self.PASSWORD).start()
        self.addCleanup(self.fake.stop)
        self.env = {"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": self.PASSWORD}
        self.stations = {"FakeStation": self.fake.url}
        self.srv = self.make_server(allow_http=True, env=self.env, stations=self.stations)

    def make_server(self, **kw):
        """The server under test (go_server.make_server), closed when the test ends."""
        srv = go_server.make_server(**kw)
        self.addCleanup(go_server.close_server, srv)
        return srv

    def station_client(self):
        """A BoxClient to arrange station state: a test client logged in to the same
        FakeStation as the server under test."""
        client = box.BoxClient(self.fake.url, self.env.get("MCP_N4_USER", "admin"),
                               self.PASSWORD, allow_http=True)
        client.open()
        self.addCleanup(client.close)
        return client

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
                               "root_handle": "2", "mode": "read-only",
                               "version": "4.14.0.162", "version_source": "/obix/about/",
                               "tier": "A", "tier_writes": "allowed"})

    def test_connect_reports_writes_allowed_mode(self):
        self.srv = self.make_server(allow_writes=True, allow_http=True, env=self.env,
                                    stations=self.stations)
        self.assertEqual(self.connect()["mode"], "writes-allowed")

    def test_credentials_come_from_the_operator_configured_env_prefix(self):
        self.srv = self.make_server(allow_http=True, stations=self.stations,
                                    credential_env="LAB",
                                    env={"LAB_USER": "admin", "LAB_PASSWORD": self.PASSWORD})
        self.assertEqual(self.connect()["station_name"], "FakeStation")

    def test_a_non_utf8_credential_is_refused_without_its_position(self):
        # R8-D19: the UnicodeEncodeError text named the bad character's position, so the
        # reply leaked how long the credential is. A non-UTF-8 env byte reads as a surrogate.
        for env in ({"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": "pw-\udcff-tail"},
                    {"MCP_N4_USER": "ad\udcffmin", "MCP_N4_PASSWORD": self.PASSWORD}):
            self.srv = self.make_server(allow_http=True, env=env, stations=self.stations)
            self.assertEqual(self.err("n4_connect", station="FakeStation"),
                             "credentials are not valid UTF-8: check env vars MCP_N4_USER and "
                             "MCP_N4_PASSWORD")
        self.assertEqual(self.fake.requests, 0)

    def test_credential_env_is_not_a_tool_argument(self):
        tools = self.srv.dispatch(rpc("tools/list"))["result"]["tools"]
        props = [t for t in tools if t["name"] == "n4_connect"][0]["inputSchema"]
        self.assertEqual(set(props["properties"]), {"station", "expected_station"})
        self.assertEqual(props["required"], ["station"])
        res = self.srv.dispatch(rpc("tools/call", {"name": "n4_connect", "arguments": {
            "station": "FakeStation", "base_url": "https://evil.example"}}))
        self.assertNotIn("error", res)  # unknown args are ignored, never used
        self.assertEqual(res["result"]["structuredContent"]["base_url"], self.fake.url)

    def test_http_station_is_refused_at_start_without_the_test_flag(self):
        with self.assertRaises(ValueError) as cm:
            self.make_server(env=self.env, stations=self.stations)
        self.assertIn("https", str(cm.exception))
        self.assertEqual(self.fake.requests, 0)

    def test_unknown_station_name_is_refused_and_names_the_flag(self):
        text = self.err("n4_connect", station="other")
        self.assertIn("--station", text)
        self.assertIn("FakeStation", text)
        self.assertEqual(self.fake.requests, 0)

    def test_no_configured_station_is_refused_and_names_the_flag(self):
        self.srv = self.make_server(allow_http=True, env=self.env)
        text = self.err("n4_connect", station="FakeStation")
        self.assertIn("--station NAME=URL", text)
        self.assertEqual(self.fake.requests, 0)

    def test_expected_station_match_connects(self):
        self.assertEqual(self.connect(expected_station="FakeStation")["station_name"],
                         "FakeStation")


class TestDescribeSession(ToolTestCase):
    def test_describe_before_connect_reports_disconnected(self):
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": False, "station_name": None, "base_url": None,
                               "mode": "read-only", "server_version": SERVER_VERSION,
                               "configured_stations": ["FakeStation"], "version": None,
                               "version_source": None, "tier": None, "tier_writes": None})

    def test_describe_after_connect(self):
        self.connect()
        out = self.ok("n4_describe_session")
        self.assertEqual(out, {"connected": True, "station_name": "FakeStation",
                               "base_url": self.fake.url, "mode": "read-only",
                               "server_version": SERVER_VERSION,
                               "configured_stations": ["FakeStation"],
                               "version": "4.14.0.162", "version_source": "/obix/about/",
                               "tier": "A", "tier_writes": "allowed"})

    def test_describe_lists_only_names_never_urls_or_credentials(self):
        self.srv = self.make_server(allow_http=True, env=self.env,
                                    stations={"b": "http://user:pw@h/", "a": "http://h2/"})
        out = self.ok("n4_describe_session")
        self.assertEqual(out["configured_stations"], ["a", "b"])
        self.assertNotIn("pw", json.dumps(out))


class StationTestCase(ToolTestCase):
    """Connected session plus helpers to build components under /Folder."""

    def setUp(self):
        super().setUp()
        self.connect()
        self.box = self.station_client()

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

    def test_types_filter_keeps_only_matching_children(self):
        nn, h = self.add("Pump")
        self.box.add_component("3", "Sub", "baja:Folder")
        out = self.ok("n4_navigate", ord="station:|slot:/Folder",
                      types=["kitControl:NumericConst"])
        self.assertEqual([(c["name"], c["type"], c["matched"]) for c in out["children"]],
                         [(nn, "kitControl:NumericConst", True)])
        self.assertEqual(out["types"], ["kitControl:NumericConst"])

    def test_types_filter_keeps_the_path_to_a_deeper_match(self):
        nn, _ = self.add("Pump")
        out = self.ok("n4_navigate", depth=2, types=["kitControl:NumericConst"])
        self.assertEqual([c["name"] for c in out["children"]], ["Folder"])
        folder = out["children"][0]
        self.assertFalse(folder["matched"])
        self.assertEqual([(c["name"], c["matched"]) for c in folder["children"]], [(nn, True)])

    def test_types_filter_with_no_match_returns_no_children(self):
        self.add("Pump")
        out = self.ok("n4_navigate", depth=2, types=["bacnet:BacnetDevice"])
        self.assertEqual(out["children"], [])

    def test_no_types_filter_is_the_unfiltered_listing(self):
        self.add("Pump")
        for types in (None, []):
            args = {} if types is None else {"types": types}
            out = self.ok("n4_navigate", depth=2, **args)
            self.assertNotIn("types", out)
            for child in out["children"]:
                self.assertNotIn("matched", child)
            self.assertIn("Folder", [c["name"] for c in out["children"]])

    def test_types_filter_rejects_non_string_entries(self):
        self.assertIn("types", self.err("n4_navigate", types=["baja:Folder", 5]))
        self.assertIn("types", self.err("n4_navigate", types=[""]))

    def test_types_filter_must_be_an_array(self):
        res = self.srv.dispatch(rpc("tools/call", {"name": "n4_navigate",
                                                   "arguments": {"types": "baja:Folder"}}))
        self.assertEqual(res["error"]["code"], -32602)

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

    def link_with_source_ord(self, source_ord):
        """A -> B link whose sourceOrd the station reports as `source_ord`."""
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.box.check_links(a[1], "out", b[1], "in10", add=True)
        self.fake.by_handle[b[1]].child("Link").child("sourceOrd").value = source_ord
        return a, b

    def test_a_slot_path_source_is_resolved_like_a_handle(self):
        a, b = self.link_with_source_ord("station:|slot:/Folder/A")
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        self.assertEqual([d["path"] for d in out["dangling"]], ["/Folder/" + b[0]])
        self.assertEqual(out["unknown"], [])
        links = self.ok("n4_list_links", ord="station:|slot:/Folder")["links"]
        self.assertEqual(links[0]["source_path"], "/Folder/" + a[0])

    def test_an_unresolvable_source_makes_candidates_unknown_not_dangling(self):
        a, b = self.link_with_source_ord("local:|foxs:|station:|bql:select x from y")
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder")
        self.assertEqual(out["dangling"], [])
        self.assertEqual(sorted(d["path"] for d in out["unknown"]),
                         sorted(["/Folder/" + a[0], "/Folder/" + b[0]]))
        self.assertEqual(out["unresolved_links"], [{
            "target_path": "/Folder/" + b[0], "link_name": "Link",
            "source_ord": "local:|foxs:|station:|bql:select x from y", "source_slot": "out"}])
        self.assertIn("unknown", out["scope_note"])

    def test_a_handle_source_outside_the_subtree_keeps_candidates_dangling(self):
        a, b = self.add_with_out("A"), self.add_with_out("B")
        self.box.check_links(a[1], "out", b[1], "in10", add=True)
        out = self.ok("n4_find_dangling_outputs", ord="station:|slot:/Folder/" + b[0], depth=1)
        self.assertEqual((out["dangling"], out["unknown"]),
                         ([{"path": "/Folder/" + b[0], "type": "kitControl:NumericConst"}], []))


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
        command = [go_server.resolve_binary()]
        proc = subprocess.run(
            command + list(args), cwd=KIT_DIR, env=env,
            input="".join(json.dumps(m) + "\n" for m in lines), capture_output=True,
            text=True, timeout=60)
        return proc, [json.loads(line) for line in proc.stdout.splitlines()]

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
        self.assertEqual(len(replies[1]["result"]["tools"]), 13)  # 10 + read_file + px views + px summary
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
