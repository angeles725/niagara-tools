"""The Go-port harness itself: impl switch, stdio adapter and FakeStation frame recorder.

These tests run under both implementations: they pin the harness, not the server. The stdio
adapter is exercised against the Python server launched as a subprocess, so no Go binary is
needed (docs/go-port-plan.md, slice S0).
"""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402
from mcp_n4 import server  # noqa: E402

PASSWORD = "harness-pw-4242"
PYTHON_STDIO = [sys.executable, "-m", "mcp_n4.server"]


def rpc(method, params=None, id_=1):
    msg = {"jsonrpc": "2.0", "method": method}
    if id_ is not None:
        msg["id"] = id_
    if params is not None:
        msg["params"] = params
    return msg


class TestImplSwitch(unittest.TestCase):
    def test_python_is_the_default_and_builds_the_in_process_server(self):
        srv = go_server.make_server(impl="python")
        self.addCleanup(go_server.close_server, srv)
        self.assertIsInstance(srv, server.Server)

    def test_an_unknown_impl_is_refused_with_the_variable_name(self):
        with self.assertRaises(ValueError) as cm:
            go_server.make_server(impl="rust")
        self.assertIn("MCP_N4_IMPL", str(cm.exception))

    def test_go_without_a_binary_fails_loudly_instead_of_passing(self):
        missing = os.path.join(tempfile.gettempdir(), "no-such-dir-4242", "mcp-n4")
        with self.assertRaises(go_server.GoBinaryMissing) as cm:
            go_server.make_server(impl="go", binary=missing)
        text = str(cm.exception)
        self.assertIn(missing, text)
        self.assertIn("MCP_N4_IMPL=go", text)
        self.assertIn("MCP_N4_GO_BIN", text)
        # Not a ValueError: a test expecting a startup refusal must not swallow it.
        self.assertNotIsInstance(cm.exception, ValueError)

    def test_options_with_no_command_line_form_are_refused(self):
        with self.assertRaises(TypeError) as cm:
            go_server.GoServer(command=PYTHON_STDIO, client_factory=object)
        self.assertIn("client_factory", str(cm.exception))


class TestGoServerOverStdio(unittest.TestCase):
    """The adapter speaks the wire protocol; the Python server stands in for the binary."""

    def start(self, **kw):
        srv = go_server.GoServer(command=PYTHON_STDIO, cwd=go_server.KIT_DIR, **kw)
        self.addCleanup(srv.close)
        return srv

    def test_dispatch_returns_the_reply_object(self):
        srv = self.start()
        res = srv.dispatch(rpc("initialize", {"protocolVersion": "2025-03-26"}))
        self.assertEqual(res["result"]["serverInfo"]["name"], "mcp-n4")
        self.assertEqual(srv.dispatch(rpc("ping", id_=7)), {"jsonrpc": "2.0", "id": 7,
                                                            "result": {}})

    def test_a_notification_returns_none_without_blocking(self):
        srv = self.start()
        self.assertIsNone(srv.dispatch(rpc("notifications/initialized", id_=None)))
        self.assertEqual(srv.dispatch(rpc("ping"))["result"], {})

    def test_handle_line_passes_raw_lines_through(self):
        srv = self.start()
        self.assertEqual(json.loads(srv.handle_line("{not json"))["error"]["code"], -32700)
        self.assertIsNone(srv.handle_line("   "))

    def test_a_startup_refusal_is_a_value_error_carrying_stderr(self):
        with self.assertRaises(ValueError) as cm:
            go_server.GoServer(command=PYTHON_STDIO, cwd=go_server.KIT_DIR,
                               stations={"S": "http://127.0.0.1:9"})
        self.assertIn("https", str(cm.exception))

    def test_close_is_idempotent_and_a_closed_server_refuses_calls(self):
        srv = self.start()
        srv.close()
        srv.close()
        with self.assertRaises(go_server.GoServerError):
            srv.dispatch(rpc("ping"))

    def test_a_session_against_the_fake_station(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        srv = self.start(allow_http=True, stations={"FakeStation": fake.url},
                         env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD})
        res = srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                              "arguments": {"station": "FakeStation"}}))
        self.assertEqual(res["result"]["structuredContent"]["station_name"], "FakeStation")


class TestFrameRecorder(unittest.TestCase):
    def connect(self, fake):
        srv = server.Server(allow_http=True, stations={"FakeStation": fake.url},
                            env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD})
        self.addCleanup(srv.ctx.close)
        res = srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                              "arguments": {"station": "FakeStation"}}))
        self.assertFalse(res["result"].get("isError", False), res)

    def test_recording_is_off_by_default(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        self.connect(fake)
        self.assertGreater(fake.requests, 0)
        self.assertIsNone(fake.frames)

    def test_recording_keeps_every_raw_box_request_and_reply_body(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        fake.record()
        self.connect(fake)
        self.assertEqual(len(fake.frames), fake.requests)
        first = fake.frames[0]
        self.assertEqual(first["status"], 200)
        self.assertEqual(json.loads(first["request"])["m"][0]["k"], "make")
        json.loads(first["reply"])  # the reply body exactly as sent

    def test_injected_and_unauthorized_replies_are_recorded_too(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        fake.record()
        fake.hook = lambda frame: (503, b"busy", {})
        srv = server.Server(allow_http=True, stations={"FakeStation": fake.url},
                            env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD})
        self.addCleanup(srv.ctx.close)
        srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                        "arguments": {"station": "FakeStation"}}))
        self.assertEqual((fake.frames[0]["status"], fake.frames[0]["reply"]), (503, "busy"))

    def test_dump_frames_writes_one_json_line_per_frame(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        fake.record()
        self.connect(fake)
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "frames.jsonl")
            fake.dump_frames(path)
            with open(path, encoding="utf-8") as fh:
                rows = [json.loads(line) for line in fh]
        self.assertEqual(rows, fake.frames)


if __name__ == "__main__":
    unittest.main()
