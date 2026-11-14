"""Shared black-box helpers for the remaining Python tool tests (test_tools_write,
test_tools_destructive, test_bql, test_list_batches, test_retro).

The protocol, connect, read-tool, tier and shutdown tests that lived here are ported to Go on
the black-box harness in go/internal/mcpn4/blackbox (one Go test per former test, or a named Go
twin); only the base classes and constants the remaining Python tests import stay here.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402,F401  (re-exported: test_tools_write uses it)
import box_client as box  # noqa: E402

NO_SLEEP = dict(sleep=lambda s: None)
#: The tool lists the server serves; go/internal/mcpn4/blackbox pins them against tools/list.
READ_TOOLS = ["n4_connect", "n4_describe_session", "n4_navigate", "n4_read_slots",
              "n4_list_links", "n4_find_dangling_outputs", "n4_bql_query", "n4_inventory",
              "n4_read_file", "n4_list_px_views", "n4_px_summary", "n4_list_batches",
              "n4_session_retro_draft"]
WRITE_TOOLS = ["n4_add_history", "n4_copy_component", "n4_create_component", "n4_create_link", "n4_invoke_action", "n4_remove_component", "n4_rollback", "n4_save_station", "n4_set_slot"]


def rpc(method, params=None, id_=1):
    msg = {"jsonrpc": "2.0", "method": method}
    if id_ is not None:
        msg["id"] = id_
    if params is not None:
        msg["params"] = params
    return msg


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
