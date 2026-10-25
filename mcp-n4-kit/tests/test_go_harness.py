"""The black-box harness itself: the stdio adapter and the FakeStation frame recorder.

These tests pin the harness, not the server; the Go binary is the process they drive
(docs/go-port-plan.md, slices S0 and S17).
"""
import json
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402

PASSWORD = "harness-pw-4242"


def rpc(method, params=None, id_=1):
    msg = {"jsonrpc": "2.0", "method": method}
    if id_ is not None:
        msg["id"] = id_
    if params is not None:
        msg["params"] = params
    return msg


class TestBinaryResolution(unittest.TestCase):
    def test_a_missing_binary_fails_loudly_instead_of_passing(self):
        with self.assertRaises(go_server.GoBinaryMissing) as cm:
            go_server.make_server(binary="/no/such/mcp-n4")
        self.assertIn("/no/such/mcp-n4", str(cm.exception))
        self.assertIn("go build -o bin/ ./cmd/mcp-n4", str(cm.exception))

    def test_the_default_binary_is_go_bin_at_the_repository_root(self):
        self.assertEqual(go_server.DEFAULT_BINARY,
                         os.path.join(go_server.REPO_DIR, "go", "bin", "mcp-n4"))

    def test_options_with_no_command_line_form_are_refused(self):
        with self.assertRaises(TypeError) as cm:
            go_server.GoServer(client_factory=object)
        self.assertIn("client_factory", str(cm.exception))


class TestGoServerOverStdio(unittest.TestCase):
    """The adapter speaks the wire protocol to the binary."""

    def start(self, **kw):
        srv = go_server.GoServer(**kw)
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
            go_server.GoServer(stations={"S": "http://127.0.0.1:9"})
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
        srv = go_server.make_server(allow_http=True, stations={"FakeStation": fake.url},
                                    env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD})
        self.addCleanup(go_server.close_server, srv)
        return srv.dispatch(rpc("tools/call", {"name": "n4_connect",
                                               "arguments": {"station": "FakeStation"}}))

    def test_recording_is_off_by_default(self):
        fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(fake.stop)
        res = self.connect(fake)
        self.assertFalse(res["result"].get("isError", False), res)
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
        self.connect(fake)
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
