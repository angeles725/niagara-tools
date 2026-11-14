"""Helpers only: WriteTestCase, FOLDER and NO_SLEEP, which test_tools_destructive imports.

The tests of this file were ported to the Go black-box harness
(go/internal/mcpn4/blackbox/write_*_test.go).
"""
import json
import os
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
import test_server  # noqa: E402  (module import: its test classes are not re-collected here)
import state_files  # noqa: E402
import box_client as box  # noqa: E402

FOLDER = "station:|slot:/Folder"
NO_SLEEP = dict(sleep=lambda s: None)


class WriteTestCase(test_server.ToolTestCase):
    SCOPES = (FOLDER,)
    MAX_WRITES = 200

    def setUp(self):
        super().setUp()
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.state_dir = os.path.join(tmp.name, "state")
        self.start_server()

    def start_server(self, **kw):
        opts = dict(allow_writes=True, allow_http=True, env=self.env, state_dir=self.state_dir,
                    stations=self.stations,
                    write_scopes=list(self.SCOPES), max_writes=self.MAX_WRITES)
        opts.update(kw)
        go_server.close_server(self.srv)
        self.srv = self.make_server(**opts)

    def connect_verified(self):
        self.connect(expected_station="FakeStation")
        self.box = self.station_client()

    def add(self, name, type_="kitControl:NumericConst", out=None):
        nn = self.box.add_component("3", name, type_)["nn"]
        h = self.box.load_tree(FOLDER, depth=2, **NO_SLEEP)[nn]["h"]
        if out is not None:
            self.box.set_slot(h, "out", box.bson_status_numeric(out))
        return nn, h

    def children(self):
        return [c.name for c in self.fake.folder.children]

    # Positional-only tool name: create_component has a `name` argument of its own.
    def call(self, tool, /, **args):
        res = self.srv.dispatch(test_server.rpc("tools/call", {"name": tool, "arguments": args}))
        self.assertNotIn("error", res, res)
        return res["result"]

    def ok(self, tool, /, **args):
        result = self.call(tool, **args)
        self.assertFalse(result.get("isError", False), result)
        return result["structuredContent"]

    def err(self, tool, /, **args):
        result = self.call(tool, **args)
        self.assertTrue(result["isError"], result)
        return result["content"][0]["text"]

    def dry(self, tool, /, **args):
        return self.ok(tool, **args)

    def run_write(self, tool, /, **args):
        plan = self.dry(tool, **args)
        return self.ok(tool, dry_run=False, confirmation_token=plan["confirmation_token"], **args)

    def batch(self, batch_id):
        """The merged journal view of a batch, read from the state file the server wrote."""
        return state_files.Journal(self.state_dir).read(batch_id)

    def lines(self, name):
        path = os.path.join(self.state_dir, name)
        if not os.path.exists(path):
            return []
        with open(path) as fh:
            return [json.loads(line) for line in fh]

