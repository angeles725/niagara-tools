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


def fail_after(side, first, key, reply):
    """A hook: `key` (a callssc sck) answers `reply` (code, body) once a `first` callssc was
    seen, or from the start when `first` is None."""
    seen = [] if first else [True]

    def hook(frame):
        sck = [m.get("b", {}).get("sck") for m in frame["m"] if isinstance(m.get("b"), dict)]
        if first is not None and first in sck:
            seen.append(True)
        elif seen and key in sck:
            return reply[0], reply[1], {}
        return None
    side.fake.hook = hook


ERROR_FRAME = b'{"m": [{"t": "e", "b": {"m": "boom"}}]}'


class TestReadBackAndInDoubtParity(ParityTestCase):
    def test_read_back_outcomes(self):
        def scenario(side):
            side.connect()
            h = status_numeric(side)
            node = side.fake.by_handle[h]
            for tamper in (lambda: setattr(node.child("out").child("value"), "value", "99.0"),
                           lambda: setattr(node.child("out").child("value"), "value", "x y"),
                           lambda: node.children.remove(node.child("out"))):
                side.fake.on_sync = lambda op, t=tamper: t()
                side.write("n4_set_slot", **SET)
                side.fake.on_sync = None
                side.box.set_slot(h, "out", box.bson_status_numeric(1.0))
            for reply in ((500, b"boom"), (200, ERROR_FRAME)):
                fail_after(side, "syncTo", "loadSlots", reply)
                side.write("n4_set_slot", **SET)
                side.fake.hook = None
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_a_failed_send_leaves_an_in_doubt_batch(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.add("Sp", "control:NumericWritable")
            for key, reply in (("syncTo", (500, b"boom")), ("syncTo", (200, ERROR_FRAME)),
                               ("invokeAction", (500, b"boom"))):
                fail_after(side, None, key, reply)
                if key == "syncTo":
                    side.write("n4_set_slot", **SET)
                else:
                    side.write("n4_invoke_action", ord=FOLDER + "/Sp", action="auto")
                side.fake.hook = None
            side.write("n4_invoke_action", ord=CALC, action="active")  # sent: no read-back
            side.call("n4_list_batches")
        self.assertSameRun(scenario)


class TestInvokeActionParity(ParityTestCase):
    def test_every_action_and_refusal(self):
        def scenario(side):
            side.connect()
            side.add("Sp", "control:NumericWritable")
            side.add("Flag", "control:BooleanWritable")
            sp = FOLDER + "/Sp"
            for action in ("save", "restart", "emergencyOverride", "Set", "x"):
                side.call("n4_invoke_action", ord=sp, action=action)
            side.write("n4_invoke_action", ord=sp, action="set", arg=7.0, arg_type="baja:Double")
            side.write("n4_invoke_action", ord=sp, action="set", arg=7, arg_type="baja:Double")
            for kw in ({}, {"arg": 1.0}, {"arg_type": "baja:Double"},
                       {"arg": 1.0, "arg_type": "baja:Int"}, {"arg": True, "arg_type": "baja:Double"},
                       {"arg": 10 ** 400, "arg_type": "baja:Double"}):
                side.call("n4_invoke_action", ord=sp, action="set", **kw)
            for action in ("active", "inactive", "auto"):
                side.write("n4_invoke_action", ord=sp, action=action)
            side.call("n4_invoke_action", ord=sp, action="auto", arg=1.0)
            side.call("n4_invoke_action", ord=sp, action="auto", arg_type="baja:Double")
            side.call("n4_invoke_action", ord=FOLDER, action="set", arg=1.0, arg_type="baja:Double")
            side.write("n4_invoke_action", ord=FOLDER + "/Flag", action="set", arg=True,
                       arg_type="baja:Boolean")  # the fake station sets numeric writables only
            orig = side.fake._invoke

            def lying(arg):
                orig(arg)
                side.fake.by_handle[arg["h"]].child("fallback").child("value").value = "1.0"
            side.fake._invoke = lying
            side.write("n4_invoke_action", ord=sp, action="set", arg=7.0, arg_type="baja:Double")
            side.fake._invoke = orig
        self.assertSameRun(scenario)


class TestPolicyParity(ParityTestCase):
    def test_the_process_budget(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", **SET)
            side.connect()
            side.write("n4_set_slot", **dict(SET, value=6.0))
        self.assertSameRun(scenario, max_writes=1)

    def test_no_write_scope(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", **SET)
        self.assertSameRun(scenario, write_scopes=[])

    def test_the_tier_gate_annotates_a_dry_run_and_refuses_the_write(self):
        def scenario(side):
            side.fake.product_version = "4.15.0.1"
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", **SET)
            side.fake.product_version = None  # undetected: tier C
            side.connect()
            side.write("n4_invoke_action", ord=CALC, action="auto")
        self.assertSameRun(scenario)

    def test_an_opt_in_lets_the_write_through(self):
        def scenario(side):
            side.fake.product_version = "4.15.0.1"
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", **SET)
        self.assertSameRun(scenario, allow_tier_b=["FakeStation"])

    def test_secrets_are_redacted_and_reads_are_audited(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", password="hunter2", api_Token=["x"], **SET)
            side.fake.ord_routes = [("baja:Component", b"Slot Path,Name\n/Folder/Calc,Calc\n")]
            side.call("n4_bql_query", base=FOLDER, query="select slotPath, name from baja:Component",
                      password="p")
        self.assertSameRun(scenario)


class TestStateFileParity(ParityTestCase):
    def path(self, side, name):
        return os.path.join(side.state_dir, name)

    def test_a_loose_state_file_fails_before_the_token_is_spent(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            side.write("n4_set_slot", **SET)  # creates journal and audit, both 0600
            for name in ("journal.jsonl", "audit.jsonl"):
                token = side.call("n4_set_slot", **SET)["confirmation_token"]
                os.chmod(self.path(side, name), 0o644)
                side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)
                os.chmod(self.path(side, name), 0o600)
                side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)
        self.assertSameRun(scenario)

    def test_an_unwritable_intent_sends_nothing_and_spends_the_token(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            os.makedirs(self.path(side, "journal.jsonl"), mode=0o700)
            token = side.call("n4_set_slot", **SET)["confirmation_token"]
            side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)
            os.rmdir(self.path(side, "journal.jsonl"))
            side.call("n4_set_slot", dry_run=False, confirmation_token=token, **SET)
        self.assertSameRun(scenario)

    def test_an_unwritable_result_or_audit_line_is_a_warning(self):
        def scenario(side):
            side.connect()
            status_numeric(side)
            journal = self.path(side, "journal.jsonl")

            def lose_the_journal(op):
                os.rename(journal, journal + ".kept")
                os.makedirs(journal, mode=0o700)
            side.fake.on_sync = lose_the_journal
            side.write("n4_invoke_action", ord=CALC, action="auto")  # an ssc: no on_sync
            side.write("n4_set_slot", **SET)
            side.fake.on_sync = None
            os.rmdir(journal)
            os.rename(journal + ".kept", journal)
            os.rename(self.path(side, "audit.jsonl"), self.path(side, "audit.kept"))
            os.makedirs(self.path(side, "audit.jsonl"), mode=0o700)
            side.write("n4_set_slot", **dict(SET, value=6.0))
            side.call("n4_set_slot", **dict(SET, slot="out/value"))
        sides = self.assertSameRun(scenario)
        for side in sides:
            self.assertTrue(os.path.isdir(self.path(side, "audit.jsonl")))
        python, under_test = (side.masked(side.state("audit.kept")) for side in sides)
        self.assertEqual(under_test, python)


CREATE = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")


def answering(side, name, reply):
    """Make the fake's `name` handler (_sync, _check_link) do its work and answer `reply`."""
    orig = getattr(side.fake, name)
    setattr(side.fake, name, lambda arg: (orig(arg), reply)[1])
    return orig


class TestCreateComponentParity(ParityTestCase):
    def test_creates_and_every_refusal(self):
        def scenario(side):
            side.call("n4_create_component", **CREATE)  # not connected
            side.connect()
            for kw in ({"name": "../x"}, {"name": "a b"}, {"name": ""}, {"name": "1x"},
                       {"name": "Pump\n"}, {"name": "Pump\n\n"}, {"type": "NoModule"},
                       {"type": "a:b:c"}, {"type": ":T"}, {"type": "k:T\n"}, {"type": "k:T\n\n"},
                       {"type": "k:T-1"}, {"wire_sheet": {"x": 1}}, {"wire_sheet": {"x": "a", "y": 1}},
                       {"wire_sheet": {"x": 1, "y": 2, "w": True}}, {"wire_sheet": {"x": True, "y": 2}},
                       {"wire_sheet": {"x": 1.0, "y": 2}}, {"wire_sheet": {"x": 1, "y": 2, "w": 0}},
                       {"wire_sheet": {"x": 1, "y": 2, "w": -10 ** 30}},
                       {"wire_sheet": {"x": 1, "y": 2, "w": 1.5}}, {"wire_sheet": {"x": 1, "y": 2, "w": None}},
                       {"wire_sheet": {"x": -3, "y": 10 ** 30, "w": 10 ** 25}}, {"wire_sheet": {}},
                       {"wire_sheet": None}, {"wire_sheet": [1]}, {"name": "a b", "type": "x"},
                       {"type": "x", "wire_sheet": {}}, {"wire_sheet": {}, "parent_ord": FOLDER + "/Missing"},
                       {"parent_ord": FOLDER + "/Missing"}, {"parent_ord": "station:|slot:/Elsewhere"},
                       {"parent_ord": FOLDER + "|h:3"}):
                side.call("n4_create_component", **dict(CREATE, **kw))
            side.write("n4_create_component", **CREATE)
            side.write("n4_create_component", **CREATE)  # the station renames it Pump1
            side.write("n4_create_component", wire_sheet={"x": 4, "y": 6, "w": 10},
                       **dict(CREATE, name="Ws"))
            side.write("n4_create_component", wire_sheet={"x": -1, "y": 0}, **dict(CREATE, name="Ws"))
            token = side.call("n4_create_component", **dict(CREATE, name="Once"))["confirmation_token"]
            for _ in range(2):  # reused
                side.call("n4_create_component", dry_run=False, confirmation_token=token,
                          **dict(CREATE, name="Once"))
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_read_back_and_odd_replies(self):
        def scenario(side):
            side.connect()

            def drop_annotation(op):
                node = side.fake.folder.child("Pump")
                node.children = [c for c in node.children if c.name != "wsAnnotation"]
            side.fake.on_sync = drop_annotation
            side.write("n4_create_component", wire_sheet={"x": 1, "y": 2}, **CREATE)
            side.fake.on_sync = lambda op: setattr(side.fake.folder.child("T"), "type", "k:Other")
            side.write("n4_create_component", **dict(CREATE, name="T"))
            side.fake.on_sync = None
            for reply in ([], [{"nn": None}], [{"nn": "Ghost"}], [{"id": "a"}], ["x"], [[1]],
                          "junk", "", {"a": 1}, 7, 0, True, None, [{"nn": ["l"]}], [{"nn": {"a": 1}}],
                          [{"nn": 5}], [{"nn": "T"}], [{"nn": ""}]):
                orig = answering(side, "_sync", reply)
                side.write("n4_create_component", **dict(CREATE, name="R"))
                side.fake._sync = orig
            for key, reply in (("syncTo", (500, b"boom")), ("syncTo", (200, ERROR_FRAME))):
                fail_after(side, None, key, reply)
                side.write("n4_create_component", **dict(CREATE, name="D"))
                side.fake.hook = None
            fail_after(side, "syncTo", "loadSlots", (500, b"boom"))
            side.write("n4_create_component", **dict(CREATE, name="L"))
            side.fake.hook = None
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

