"""Write parity of slice S10 (docs/go-port-plan.md section 14), black-box.

Each scenario runs twice: on the in-process Python server and on the server under test
(`go_server.make_server`: the Go binary under MCP_N4_IMPL=go), each with its own recording
FakeStation and state dir. Every reply line, every BOX request body and the bytes of
`journal.jsonl` and `audit.jsonl` must be equal once the per-run values are masked: the
station URL and state dir, confirmation tokens, plan hashes, batch and session ids, expiry, timestamps and a
read's elapsed time.
Every dry run of the server under test must also hash its plan as Python does:
`plan_hash == sha256(safety.canonical(plan))`.

Under the default implementation both sides are Python, which checks the harness.
"""
import hashlib
import json
import os
import re
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box, safety  # noqa: E402

PASSWORD = "s3cret-pw-value"
FOLDER = "station:|slot:/Folder"
CALC = FOLDER + "/Calc"
NO_SLEEP = dict(sleep=lambda s: None)
MASKS = [(re.compile(r"[0-9]+\.[0-9a-f]{16}\.[0-9a-f]{64}"), "<token>"),
         (re.compile(r"[0-9a-f]{64}"), "<hash>"),
         (re.compile(r"[0-9a-f]{32}"), "<id>"),
         (re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\+00:00"), "<ts>"),
         (re.compile(r'((?:expires_at|elapsed_ms)\\?": )[0-9]+'), r"\1<n>")]


class Side:
    """One server, its FakeStation and state dir, and what the scenario saw."""

    def __init__(self, test, impl, **opts):
        self.fake = FakeStation(password=PASSWORD).start()
        test.addCleanup(self.fake.stop)
        self.fake.record()
        tmp = tempfile.TemporaryDirectory()
        test.addCleanup(tmp.cleanup)
        self.state_dir = os.path.join(tmp.name, "state")
        self.opts = dict(allow_writes=True, allow_http=True, state_dir=self.state_dir,
                         env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD},
                         stations={"FakeStation": self.fake.url}, write_scopes=[FOLDER])
        self.opts.update(opts)
        self.srv = go_server.make_server(impl, **self.opts)
        test.addCleanup(lambda: go_server.close_server(self.srv))
        self.replies, self.dry_runs = [], []
        self.client = None

    @property
    def box(self):
        """A Python BoxClient that arranges station state (its frames are recorded too)."""
        if self.client is None:
            self.client = box.BoxClient(self.fake.url, "admin", PASSWORD, allow_http=True)
            self.client.open()
        return self.client

    def call(self, tool, **args):
        msg = {"jsonrpc": "2.0", "id": 1, "method": "tools/call",
               "params": {"name": tool, "arguments": args}}
        line = self.srv.handle_line(json.dumps(msg))
        self.replies.append(line)
        result = json.loads(line).get("result", {})
        out = result.get("structuredContent")
        if isinstance(out, dict) and out.get("dry_run") is True:
            self.dry_runs.append(out)
        return out if isinstance(out, dict) else {}

    def write(self, tool, **args):
        """A dry run, then its execution with the token it returned."""
        token = self.call(tool, **args).get("confirmation_token")
        return self.call(tool, dry_run=False, confirmation_token=token, **args)

    def connect(self):
        return self.call("n4_connect", station="FakeStation")

    def add(self, name="Calc", type_="kitControl:NumericConst", **slots):
        nn = self.box.add_component("3", name, type_)["nn"]
        h = self.box.load_tree(FOLDER, depth=2, **NO_SLEEP)[nn]["h"]
        for slot, bson in slots.items():
            self.box.set_slot(h, slot, bson)
        return h

    def state(self, name):
        path = os.path.join(self.state_dir, name)
        if not os.path.isfile(path):
            return None
        with open(path, "rb") as fh:
            return fh.read().decode("utf-8", "surrogateescape")

    def masked(self, text):
        if text is None:
            return None
        text = text.replace(self.fake.url, "<url>").replace(self.state_dir, "<state>")
        for pattern, repl in MASKS:
            text = pattern.sub(repl, text)
        return text


class ParityTestCase(unittest.TestCase):
    def assertSameRun(self, scenario, **opts):
        """Run scenario(side) on both servers and require the same bytes after masking."""
        sides = [Side(self, "python", **opts), Side(self, None, **opts)]
        for side in sides:
            scenario(side)
            if side.client is not None:
                side.client.close()
        python, under_test = sides
        for i, (want, got) in enumerate(zip(python.replies, under_test.replies)):
            self.assertEqual(under_test.masked(got), python.masked(want), "reply %d" % i)
        self.assertEqual(len(under_test.replies), len(python.replies))
        self.assertEqual([f["request"] for f in under_test.fake.frames],
                         [f["request"] for f in python.fake.frames], "BOX request bodies")
        for name in ("journal.jsonl", "audit.jsonl"):
            self.assertEqual(under_test.masked(under_test.state(name)),
                             python.masked(python.state(name)), name)
        for out in under_test.dry_runs:
            digest = hashlib.sha256(safety.canonical(out["plan"]).encode()).hexdigest()
            self.assertEqual(out["plan_hash"], digest, "plan hash bytes")
        return sides


def status_numeric(side, value=1.0):
    return side.add(out=box.bson_status_numeric(value))


SET = dict(ord=CALC, slot="out", value=5.5, value_type="baja:StatusNumeric")


class TestSetSlotParity(ParityTestCase):
    def test_a_write_and_every_token_refusal(self):
        def scenario(side):
            side.call("n4_set_slot", **SET)  # not connected: refused, dry run or not
            side.call("n4_set_slot", dry_run=False, confirmation_token="1.ab.cd", **SET)
            side.connect()
            status_numeric(side)
            token = side.call("n4_set_slot", **SET)["confirmation_token"]
            side.call("n4_set_slot", dry_run=False, **SET)  # no token
            for bad in ("x.y.z", "1.2", 7, token[:-1] + ("0" if token[-1] != "0" else "1")):
                side.call("n4_set_slot", dry_run=False, confirmation_token=bad, **SET)
            side.call("n4_set_slot", dry_run=False, confirmation_token=token,
                      **dict(SET, value=6.0))  # other args
            side.call("n4_invoke_action", dry_run=False, confirmation_token=token, ord=CALC,
                      action="auto")  # another tool
            side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)
            side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)  # reused
            stale = side.call("n4_set_slot", **dict(SET, value=7.0))["confirmation_token"]
            side.connect()  # a new session: the plan names the old one
            side.call("n4_set_slot", dry_run=False, confirmation_token=stale,
                      **dict(SET, value=7.0))
            moved = side.call("n4_set_slot", **dict(SET, value=8.0))["confirmation_token"]
            side.box.set_slot("10", "out", box.bson_status_numeric(3.0))  # someone else writes
            side.call("n4_set_slot", dry_run=False, confirmation_token=moved,
                      **dict(SET, value=8.0))
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_every_value_type_and_status(self):
        def scenario(side):
            side.connect()
            side.add(d=box.bson_double(2), b=box.bson_bool(True),
                     s={"nm": "p", "t": "baja:String", "v": "old"},
                     flag=box.bson_status_boolean(False), out=box.bson_status_numeric(1.0),
                     grp={"nm": "p", "t": "baja:Struct",
                          "s": [dict(box.bson_status_numeric(7.5), n="sp")]})
            for slot, type_, value in (("d", "baja:Double", 3), ("d", "baja:Double", 2 ** 53 + 1),
                                       ("d", "baja:Double", 1e-05), ("b", "baja:Boolean", False),
                                       ("s", "baja:String", "new é😀"),
                                       ("flag", "baja:StatusBoolean", True),
                                       ("grp/sp", "baja:StatusNumeric", 9.0)):
                side.write("n4_set_slot", ord=CALC, slot=slot, value=value, value_type=type_)
            for status in ("8", "a0;facets", "0\n", "0;x\n"):
                side.write("n4_set_slot", **dict(SET, status=status))
        self.assertSameRun(scenario)

    def test_every_plan_refusal(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.add("Other", d=box.bson_double(0.5))
            for slot in ("out/value", "out/status", "a b", "out\n", "1x", "", "a//b", "grp/"):
                side.call("n4_set_slot", **dict(SET, slot=slot))
            for kw in ({"value": True, "value_type": "baja:Double"},
                       {"value": float("nan")}, {"value": float("inf")}, {"value": 10 ** 400},
                       {"value": "x", "value_type": "baja:Boolean"},
                       {"value": 1, "value_type": "baja:String"},
                       {"value": 1.0, "value_type": "baja:Int"},
                       {"value": 1.0, "value_type": "baja:Int", "status": "0"}, {"status": "zz"}, {"status": ""}, {"status": "0\n\n"}, {"status": 7},
                       {"value_type": "baja:Double", "status": "0"},
                       {"slot": "nope", "value_type": "baja:Double"},
                       {"value": True, "value_type": "baja:StatusBoolean"},
                       {"ord": FOLDER + "/Missing"}, {"ord": "station:|slot:/Elsewhere"},
                       {"ord": CALC + "|h:10"}, {"ord": FOLDER + "/Other", "slot": "zero",
                                                 "value_type": "baja:Double"}):
                side.call("n4_set_slot", **dict(SET, **kw))
            side.call("n4_set_slot", ord=CALC, slot="out", value_type="baja:StatusNumeric")
        self.assertSameRun(scenario)

