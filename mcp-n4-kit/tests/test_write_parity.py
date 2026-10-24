"""Write parity of slice S10 (docs/go-port-plan.md section 14), black-box.

Each scenario runs twice: on the in-process Python server and on the server under test
(`go_server.make_server`: the Go binary under MCP_N4_IMPL=go), each with its own recording
FakeStation and state dir. Every reply line, every BOX request body and the bytes of
`journal.jsonl` and `audit.jsonl` must be equal once the per-run values are masked: the
station URL, state dir and station home, confirmation tokens, plan hashes, batch and session ids,
expiry, timestamps, a read's elapsed time and a config.bog mtime.
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
import fake_station  # noqa: E402
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
         (re.compile(r'((?:expires_at|elapsed_ms)\\?": )[0-9]+'), r"\1<n>"),
         (re.compile(r'(mtime\\?": )[0-9.e+-]+'), r"\1<n>")]


class Side:
    """One server, its FakeStation and state dir, and what the scenario saw."""

    def __init__(self, test, impl, home=None, **opts):
        """`home` (bytes) starts the server with a station home holding that config.bog,
        which the FakeStation's save rewrites."""
        self.fake = FakeStation(password=PASSWORD).start()
        test.addCleanup(self.fake.stop)
        self.fake.record()
        tmp = tempfile.TemporaryDirectory()
        test.addCleanup(tmp.cleanup)
        self.state_dir = os.path.join(tmp.name, "state")
        self.opts = dict(allow_writes=True, allow_http=True, state_dir=self.state_dir,
                         env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD},
                         stations={"FakeStation": self.fake.url}, write_scopes=[FOLDER])
        self.home = None
        if home is not None:
            self.home = os.path.join(tmp.name, "home")
            os.mkdir(self.home)
            self.fake.config_path = os.path.join(self.home, "config.bog")
            self.write_bog(home)
            self.opts["station_homes"] = {"FakeStation": self.home}
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

    def write_bog(self, content):
        with open(self.fake.config_path, "wb") as fh:
            fh.write(content)

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
        if self.home:
            text = text.replace(self.home, "<home>")
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
    """Make the fake's `name` handler (such as _sync) do its work and answer `reply`."""
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


class TestCreateLinkParity(ParityTestCase):
    def link_args(self, side):
        side.add("Src", out=box.bson_status_numeric(1.0))
        side.add("Tgt")
        return dict(source_ord=FOLDER + "/Src", source_slot="out", target_ord=FOLDER + "/Tgt",
                    target_slot="in10")

    def test_links_and_every_refusal(self):
        def scenario(side):
            args = self.link_args(side)
            side.call("n4_create_link", **args)  # not connected
            side.connect()
            for kw in ({"source_slot": "a b"}, {"target_slot": ""}, {"target_slot": "in10\n"},
                       {"source_slot": "1x", "target_slot": "a b"}, {"source_ord": FOLDER + "/Missing"},
                       {"target_ord": FOLDER + "/Missing"}, {"source_ord": "station:|slot:/Other"},
                       {"target_ord": "station:|slot:/Other"}, {"target_ord": FOLDER + "/Tgt|h:x"}):
                side.call("n4_create_link", **dict(args, **kw))
            side.write("n4_create_link", **args)
            side.write("n4_create_link", **args)  # a second link: Link1
            side.write("n4_create_link", **dict(args, target_slot="in11"))
            orig = side.fake._check_link

            def lying(arg):
                res = orig(arg)
                side.fake.by_handle[arg["t"]].child(res[0]["s"]).child("sourceSlotName").value = "x"
                return res
            side.fake._check_link = lying
            side.write("n4_create_link", **args)

            def prefixed(arg):  # the station may prefix the handle ORD: still verified
                res = orig(arg)
                link = side.fake.by_handle[arg["t"]].child(res[0]["s"])
                link.child("sourceOrd").value = "station:|" + link.child("sourceOrd").value
                return res
            side.fake._check_link = prefixed
            side.write("n4_create_link", **args)
            side.fake._check_link = orig
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_refused_ambiguous_and_malformed_replies(self):
        def scenario(side):
            args = self.link_args(side)
            side.connect()
            orig = side.fake._check_link
            for reply in ([{"v": False, "r": "type mismatch", "s": None}], [{"v": 0}],
                          [{"v": True, "r": None, "s": ""}], [{"v": True}], [{"r": 1}], "junk", [],
                          [None], [[]], [[{"v": True}]], [{"v": 1, "s": 5}], [{"v": "yes", "s": ["l"]}],
                          [{"v": True, "s": "Ghost"}], [{"v": True, "s": {"a": 1}}]):
                side.fake._check_link = lambda arg, r=reply: r
                side.write("n4_create_link", **args)
            # accepted but not named: the recover hook finds the new link by its slots
            side.fake._check_link = lambda arg: [dict(orig(arg)[0], s=None)]
            side.write("n4_create_link", **args)

            def prefixed(arg):  # ... also behind a station-prefixed source ORD
                res = orig(arg)
                link = side.fake.by_handle[arg["t"]].child(res[0]["s"])
                link.child("sourceOrd").value = "station:|" + link.child("sourceOrd").value
                return [dict(res[0], s=None)]
            side.fake._check_link = prefixed
            side.write("n4_create_link", **args)
            def twice(arg):
                orig(arg)
                return [dict(orig(arg)[0], s=None)]
            side.fake._check_link = twice
            side.write("n4_create_link", **args)  # two new matching links: none is chosen

            def other_slot(arg):
                res = orig(arg)
                side.fake.by_handle[arg["t"]].child(res[0]["s"]).child("targetSlotName").value = "x"
                return [dict(res[0], s=None)]
            side.fake._check_link = other_slot
            side.write("n4_create_link", **args)
            side.fake._check_link = lambda arg: [dict(orig(arg)[0], s=None)]
            fail_after(side, "checkLinks", "loadSlots", (500, b"boom"))
            side.write("n4_create_link", **args)  # the recover load fails
            side.fake.hook = None
            side.fake._check_link = orig
            fail_after(side, "checkLinks", "loadSlots", (500, b"boom"))
            side.write("n4_create_link", **args)  # the read-back load fails
            fail_after(side, None, "checkLinks", (500, b"boom"))
            side.write("n4_create_link", **args)
            side.fake.hook = None
            side.call("n4_list_batches")
        self.assertSameRun(scenario)


def remove_group(side, name="Grp"):
    """A Folder with a wsAnnotation, Src -> Tgt.in10 inside, Other.out -> Tgt.in11 from
    outside, Src.out -> Out.in10 to outside, and a link-driven Status input in12 on Tgt."""
    nn = side.box.add_component("3", name, "baja:Folder", ws=box.ws_annotation(1, 2, 3))["nn"]
    grp = FOLDER + "/" + nn
    gh = side.box.load_tree(grp, depth=1, **NO_SLEEP)[""]["h"]
    side.box.add_component(gh, "Src", "kitControl:NumericConst")
    side.box.add_component(gh, "Tgt", "kitControl:NumericConst")
    nodes = side.box.load_tree(grp, depth=2, **NO_SLEEP)
    src_h, tgt_h = nodes["Src"]["h"], nodes["Tgt"]["h"]
    side.box.set_slot(src_h, "out", box.bson_status_numeric(4.5, "0;activeLevel=e_in16"))
    side.box.set_slot(src_h, "fallback", box.bson_status_numeric(7.0, "40;activeLevel=e_def"))
    side.box.check_links(src_h, "out", tgt_h, "in10")
    other_h = side.add("Other" + nn, out=box.bson_status_numeric(1.0))
    side.box.check_links(other_h, "out", tgt_h, "in11")
    out_h = side.add("Out" + nn)
    side.box.check_links(src_h, "out", out_h, "in10")
    tgt = side.fake.by_handle[tgt_h]
    for slot in ("in12", "in10"):  # as the live station loads it: frozen children, no type
        node = fake_station._Node(slot, "baja:StatusNumeric")
        node.children = [fake_station._Node("value", None, "2.5"),
                         fake_station._Node("status", None, "0;activeLevel=e:17")]
        tgt.children.append(node)
    side.box.check_links(src_h, "out", tgt_h, "in12")
    return nn


def deep_group(side, name="Deep", leaves=1):
    """Folder/L1/L2/C with `leaves` components under C: level 4, below the snapshot cut."""
    nn = side.box.add_component("3", name, "baja:Folder")["nn"]
    path, h = FOLDER + "/" + nn, None
    for child, type_ in (("L1", "baja:Folder"), ("L2", "baja:Folder"), ("C", "baja:Folder")):
        h = side.box.load_tree(path, depth=1, **NO_SLEEP)[""]["h"]
        side.box.add_component(h, child, type_)
        path += "/" + child
    h = side.box.load_tree(path, depth=1, **NO_SLEEP)[""]["h"]
    for i in range(leaves):
        side.box.add_component(h, "K%02d" % i, "kitControl:NumericConst")
    return nn


class TestRemoveComponentParity(ParityTestCase):
    def test_removes_and_every_refusal(self):
        def scenario(side):
            side.call("n4_remove_component", parent_ord=FOLDER, name="Grp")  # not connected
            side.connect()
            nn = remove_group(side)
            for kw in ({"name": "../x"}, {"name": ""}, {"name": "Grp\n"}, {"name": "a b"},
                       {"name": 7}, {"name": "Nope"}, {"parent_ord": FOLDER + "/Missing"},
                       {"parent_ord": "station:|slot:/"}, {"parent_ord": FOLDER + "|h:3"},
                       {"link_scan_ord": "station:|slot:/"},
                       {"link_scan_ord": FOLDER + "|h:3"}, {"link_scan_ord": FOLDER + "/Missing"}):
                side.call("n4_remove_component", **dict({"parent_ord": FOLDER, "name": nn}, **kw))
            side.call("n4_remove_component", parent_ord=FOLDER, name=nn)
            side.call("n4_remove_component", parent_ord=FOLDER, name=nn, link_scan_ord=FOLDER)
            side.call("n4_remove_component", parent_ord=FOLDER, name=nn,
                      link_scan_ord=FOLDER + "/" + nn)  # every link starts inside: none found
            side.write("n4_remove_component", parent_ord=FOLDER, name=nn, link_scan_ord=FOLDER)
            side.call("n4_remove_component", parent_ord=FOLDER, name=nn)  # gone now
            side.write("n4_remove_component", parent_ord=FOLDER, name="Other" + nn)  # has outputs
            side.write("n4_remove_component", parent_ord=FOLDER, name="Out" + nn)
            plain = side.box.add_component("3", "Plain", "baja:Folder")["nn"]
            token = side.call("n4_remove_component", parent_ord=FOLDER,
                              name=plain)["confirmation_token"]
            for _ in range(2):  # reused
                side.call("n4_remove_component", parent_ord=FOLDER, name=plain, dry_run=False,
                          confirmation_token=token)
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_truncation_read_back_and_in_doubt(self):
        def scenario(side):
            side.connect()
            for name, leaves in (("D1", 1), ("D20", 20), ("D21", 21)):
                side.write("n4_remove_component", parent_ord=FOLDER, name=deep_group(side, name, leaves))
            nn = remove_group(side, "Moving")
            token = side.call("n4_remove_component", parent_ord=FOLDER,
                              name=nn)["confirmation_token"]
            tgt = side.fake.folder.child(nn).child("Tgt")
            tgt.child("in12").child("value").value = "9.75"  # the link keeps propagating
            side.call("n4_remove_component", parent_ord=FOLDER, name=nn, dry_run=False,
                      confirmation_token=token)
            side.add("Keep")
            orig = side.fake._sync
            side.fake._sync = lambda op: [] if op["nm"] == "v" else orig(op)
            side.write("n4_remove_component", parent_ord=FOLDER, name="Keep")  # still there
            side.fake._sync = orig
            for first, key, reply in ((None, "syncTo", (500, b"boom")),
                                      (None, "syncTo", (200, ERROR_FRAME)),
                                      ("syncTo", "loadSlots", (500, b"boom"))):
                name = side.box.add_component("3", "Gone", "kitControl:NumericConst")["nn"]
                token = side.call("n4_remove_component", parent_ord=FOLDER,
                                  name=name)["confirmation_token"]
                fail_after(side, first, key, reply)
                side.call("n4_remove_component", parent_ord=FOLDER, name=name, dry_run=False,
                          confirmation_token=token)
                side.fake.hook = None
            side.call("n4_list_batches")
        self.assertSameRun(scenario)


def fail_nth(side, key, nth, reply):
    """A hook: the nth callssc `key` from now on answers `reply`: (code, body), or a
    transport fault (fake_station.DROP or fake_station.Raw)."""
    seen = []

    def hook(frame):
        sck = [m.get("b", {}).get("sck") for m in frame["m"] if isinstance(m.get("b"), dict)]
        if key in sck:
            seen.append(True)
            if len(seen) == nth:
                if reply is fake_station.DROP or isinstance(reply, fake_station.Raw):
                    return reply
                return reply[0], reply[1], {}
        return None
    side.fake.hook = hook


def craft(side, bid, inverse, targets, station="FakeStation", verdict="verified", **intent):
    """Journal a batch the server never ran (an intent and its result), through the file."""
    journal = safety.Journal(side.state_dir)
    journal.append(dict({"batch_id": bid, "phase": "intent", "tool": "n4_remove_component",
                         "station_name": station, "ops": [], "inverse_plan": inverse,
                         "targets": targets}, **intent))
    journal.append({"batch_id": bid, "phase": "result", "verdict": verdict, "inverse": inverse,
                    "accepted": None})


def writable_group(side, name="Wg"):
    """remove_group plus a NumericWritable whose frozen proxyExt is configured and a
    BooleanWritable: frozen children the rollback must not re-add."""
    nn = remove_group(side, name)
    gh = side.box.load_tree(FOLDER + "/" + nn, depth=1, **NO_SLEEP)[""]["h"]
    side.box.add_component(gh, "Sp", "control:NumericWritable")
    side.box.add_component(gh, "En", "control:BooleanWritable")
    ext = side.fake.folder.child(nn).child("Sp").child("proxyExt")
    side.box.sync({"nm": "s", "h": ext.handle, "n": "tuningPolicyName",
                   "b": {"nm": "p", "t": "baja:String", "v": "Fast"}})
    sub = side.box.add_component(gh, "Sub", "baja:Folder")["nn"]
    sub_h = side.box.load_tree(FOLDER + "/%s/%s" % (nn, sub), depth=1, **NO_SLEEP)[""]["h"]
    side.box.add_component(sub_h, "Deep", "kitControl:NumericConst")
    return nn


def removed(side, nn, **kw):
    return side.write("n4_remove_component", parent_ord=FOLDER, name=nn, **kw)["batch_id"]


class TestRollbackParity(ParityTestCase):
    def test_rollbacks_of_every_tool_and_every_refusal(self):
        def scenario(side):
            side.call("n4_rollback", batch_id="ab" * 16)  # not connected
            side.connect()
            for bid in ("f" * 32 + "\n", "F" * 32, "f" * 31, 7, "f" * 32):
                side.call("n4_rollback", batch_id=bid)
            created = side.write("n4_create_component", **CREATE)["batch_id"]
            side.write("n4_rollback", batch_id=created)
            side.call("n4_rollback", batch_id=created)  # already rolled back
            last = [json.loads(line) for line in side.state("journal.jsonl").splitlines()][-1]
            side.call("n4_rollback", batch_id=last["batch_id"])  # a rollback of a rollback
            h = status_numeric(side)
            set_id = side.write("n4_set_slot", **SET)["batch_id"]
            token = side.call("n4_rollback", batch_id=set_id)["confirmation_token"]
            side.box.set_slot(h, "out", box.bson_status_numeric(4.0))  # changed after the dry run
            side.call("n4_rollback", batch_id=set_id, dry_run=False, confirmation_token=token)
            side.box.set_slot(h, "out", box.bson_status_numeric(5.5))
            side.write("n4_rollback", batch_id=set_id)
            sp = side.add("Sp", "control:NumericWritable")
            invoked = side.write("n4_invoke_action", ord=FOLDER + "/Sp", action="set", arg=7.0,
                                 arg_type="baja:Double")["batch_id"]
            side.box.set_slot(sp, "fallback", box.bson_status_numeric(3.0))
            side.call("n4_rollback", batch_id=invoked)  # the fallback changed
            side.box.set_slot(sp, "fallback", box.bson_status_numeric(7.0))
            side.write("n4_rollback", batch_id=invoked)
            active = side.write("n4_invoke_action", ord=FOLDER + "/Sp", action="active")["batch_id"]
            side.call("n4_rollback", batch_id=active)  # no inverse
            side.add("Tgt")
            linked = side.write("n4_create_link", source_ord=CALC, source_slot="out",
                                target_ord=FOLDER + "/Tgt", target_slot="in10")["batch_id"]
            side.write("n4_rollback", batch_id=linked)
            calc = side.write("n4_set_slot", **dict(SET, value=6.5))["batch_id"]
            side.box.remove_component("3", "Calc")
            status_numeric(side)  # same ORD, a new component
            side.call("n4_rollback", batch_id=calc)
            side.write("n4_rollback", batch_id=removed(side, remove_group(side)))
            nn = writable_group(side)
            bid = removed(side, nn)
            for _ in range(2):
                token = side.call("n4_rollback", batch_id=bid)["confirmation_token"]
            for _ in range(2):  # reused
                side.call("n4_rollback", batch_id=bid, dry_run=False, confirmation_token=token)
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_crafted_journal_records(self):
        def scenario(side):
            side.connect()
            side.add("Calc")
            ok_add = {"nm": "a", "h": "3", "n": "X", "b": {"nm": "p", "t": "baja:Folder"}}
            cases = [
                ([{"nm": "v", "h": "3", "n": "x"}], {FOLDER: "3"}, {"station": "OtherStation"}),
                ([ok_add, {"relink": {"source_path": 1, "source_slot": "out"}}], {FOLDER: "3"}, {}),
                ([ok_add, {"relink": {"source_path": "a b", "source_slot": "out",
                                      "target_path": "", "target_slot": "x"}}], {FOLDER: "3"}, {}),
                ([ok_add, {"unlinked_input": 5}], {FOLDER: "3"}, {}),
                ([ok_add, {"unlinked_input": {"source_slot": "o", "target_path": "Y",
                                              "target_slot": "in", "prior": {"v": 1}}}],
                 {FOLDER: "3"}, {}),
                ([{"nm": "v", "h": "3", "n": "x"}], {}, {}),
                ([{"nm": "v", "h": "9", "n": "x"}], {FOLDER: "3"}, {}),
                ([{"nm": "v", "h": "3", "n": "x"}], [FOLDER], {}),
                ([{"nm": "v", "h": "3", "n": "x"}], {FOLDER: "4"}, {}),
                ([{"nm": "v", "h": "3", "n": "x"}], {"station:|slot:/Elsewhere": "3"}, {}),
                ([], {FOLDER: "3"}, {}),
                ("ab", {FOLDER: "3"}, {}),
                (7, {FOLDER: "3"}, {}),
                ([{"nm": "a", "h": "3", "n": "Y", "b": {"nm": "p", "t": "baja:Folder", "s": [
                    {"nm": "p", "t": "kitControl:NumericConst", "n": "bad name"}]}}],
                 {FOLDER: "3"}, {}),
                ([{"nm": "a", "h": "3", "n": "Z", "b": [1]}], {FOLDER: "3"}, {}),
                ([{"nm": "a", "h": "3", "n": "W"}], {FOLDER: "3"}, {}),
                ([{"nm": "s", "h": "3", "n": "q", "b": {"nm": "p", "t": "baja:Double", "v": "1"}}],
                 {FOLDER: "3"}, {}),
                ([{"nm": "v", "h": "3", "n": "x"}], {FOLDER: "3"},
                 {"rollback_of": "cd" * 16}),
            ]
            for i, (inverse, targets, kw) in enumerate(cases):
                bid = "%02x" % i * 16
                station = kw.pop("station", "FakeStation")
                craft(side, bid, inverse, targets, station=station, **kw)
                side.call("n4_rollback", batch_id=bid)
            for verdict, extra in (("failed", {}), ("failed", {"readback_failed": True})):
                done = side.write("n4_create_component", **dict(CREATE, name="F"))["batch_id"]
                journal = safety.Journal(side.state_dir)
                journal.append({"batch_id": "ee" * 16, "phase": "intent", "rollback_of": done})
                journal.append(dict({"batch_id": "ee" * 16, "phase": "result", "verdict": verdict,
                                     "inverse": [], "accepted": [1]}, **extra))
                side.write("n4_rollback", batch_id=done)
            with open(os.path.join(side.state_dir, "journal.jsonl"), "ab") as fh:
                fh.write(b'{"batch_id": [1], "phase": "intent"}\n{"batch_id": "\xff"}\n')
            side.call("n4_rollback", batch_id="00" * 16)
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_budget_scope_and_station_root(self):
        def scenario(side):
            side.connect()
            side.call("n4_rollback", batch_id=removed(side, remove_group(side)))
            side.call("n4_list_batches")
        self.assertSameRun(scenario, max_writes=3)

        def root(side):
            side.connect()
            top = side.box.add_component("2", "RootGrp", "baja:Folder")["nn"]
            top_h = side.box.load_tree("station:|slot:/" + top, depth=1, **NO_SLEEP)[""]["h"]
            side.box.add_component(top_h, "Src", "kitControl:NumericConst")
            side.write("n4_rollback", batch_id=side.write(
                "n4_remove_component", parent_ord="station:|slot:/", name=top)["batch_id"])
        self.assertSameRun(root, write_scopes=["station:|slot:/"])

    def test_read_back_outcomes_and_in_doubt_runs(self):
        def scenario(side):
            side.connect()
            nn = writable_group(side, "A")
            bid = removed(side, nn)
            orig = side.fake._sync

            def drop(name, kind="drop"):
                def sync(op):
                    res = orig(op)
                    if op["nm"] == "a" and op["n"] == name:
                        node = side.fake.folder.child(nn) if name != nn else side.fake.folder
                        kid = node.child(name)
                        if kind == "type":
                            kid.type = "kitControl:BooleanConst"
                        elif kind == "ws":
                            kid.children = [c for c in kid.children if c.name != "wsAnnotation"]
                        else:
                            node.children = [c for c in node.children if c.name != name]
                    return res
                return sync
            side.fake._sync = drop("Src")  # a relink end is missing: skipped, mismatch
            side.write("n4_rollback", batch_id=bid)
            for name, kind in (("Tgt", "type"), (nn, "ws")):
                nn2 = remove_group(side, "B" + kind)
                bid2 = removed(side, nn2)
                nn = nn2
                side.fake._sync = drop(name if name != "Tgt" else "Tgt", kind)
                side.write("n4_rollback", batch_id=bid2)
            side.fake._sync = orig
            orig_link = side.fake._check_link
            for reply in ([{"r": "huh"}], [{"v": False}], [{"v": True}], "junk"):
                nn = remove_group(side, "L")
                bid = removed(side, nn)
                side.fake._check_link = lambda arg, reply=reply: reply
                side.write("n4_rollback", batch_id=bid)
            side.fake._check_link = orig_link
            for key, nth, reply in (("syncTo", 1, (500, b"boom")), ("syncTo", 2, (200, ERROR_FRAME)),
                                    ("syncTo", 3, (500, b"boom")), ("checkLinks", 1, (500, b"boom")),
                                    ("checkLinks", 1, (200, ERROR_FRAME))) + tuple(
                                        ("loadSlots", n, (500, b"boom")) for n in range(1, 12)):
                nn = remove_group(side, "D")
                bid = removed(side, nn)
                token = side.call("n4_rollback", batch_id=bid)["confirmation_token"]
                fail_nth(side, key, nth, reply)
                side.call("n4_rollback", batch_id=bid, dry_run=False, confirmation_token=token)
                side.fake.hook = None
            three = [{"nm": "a", "h": "3", "n": n, "b": {"nm": "p", "t": "baja:Folder"}}
                     for n in ("A2", "B2", "C2")]
            for bid, nth in (("ab" * 16, 2), ("ac" * 16, 1), ("ad" * 16, 3)):  # op nth fails
                craft(side, bid, three, {FOLDER: "3"})
                token = side.call("n4_rollback", batch_id=bid)["confirmation_token"]
                fail_nth(side, "syncTo", nth, (500, b"boom"))
                side.call("n4_rollback", batch_id=bid, dry_run=False, confirmation_token=token)
                side.fake.hook = None
                side.call("n4_rollback", batch_id=bid)  # its rollback is in-doubt now
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_transport_faults_and_unexpected_failures_keep_the_batch_id(self):
        """A dropped connection or a non-HTTP answer (BoxError) on a send, op 2 of 3, the
        relink and the read-back; then a nested add answered with a name that is no slot
        name (ValueError, the unexpected-failure wrap after the sends)."""
        faults = (fake_station.DROP, fake_station.Raw(b"garbage\r\n\r\n"))

        def scenario(side):
            side.connect()
            status_numeric(side)

            def faulted(tool, key, fault, nth=1, **args):
                token = side.call(tool, **args)["confirmation_token"]
                fail_nth(side, key, nth, fault)
                side.call(tool, dry_run=False, confirmation_token=token, **args)
                side.fake.hook = None
            for i, fault in enumerate(faults):
                faulted("n4_set_slot", "syncTo", fault, **SET)
                faulted("n4_set_slot", "loadSlots", fault, nth=2, **SET)  # the read-back
                bid = "a%d" % i * 16
                craft(side, bid, [{"nm": "a", "h": "3", "n": "T%d%s" % (i, n),
                                   "b": {"nm": "p", "t": "baja:Folder"}} for n in "ABC"],
                      {FOLDER: "3"})
                faulted("n4_rollback", "syncTo", fault, nth=2, batch_id=bid)
                bid = removed(side, remove_group(side, "Tl%d" % i))
                faulted("n4_rollback", "checkLinks", fault, batch_id=bid)
            orig = side.fake._sync
            for i, name in enumerate((["l"], "a/b", "")):
                bid = removed(side, remove_group(side, "Tn%d" % i))
                token = side.call("n4_rollback", batch_id=bid)["confirmation_token"]
                side.fake._sync = lambda op, name=name: (orig(op), [{"id": "a", "nn": name}])[1] \
                    if op["nm"] == "a" and op["n"] == "Src" else orig(op)
                side.call("n4_rollback", batch_id=bid, dry_run=False, confirmation_token=token)
                side.fake._sync = orig
            side.call("n4_list_batches")
        for side in self.assertSameRun(scenario):  # every error names its batch, in the audit too
            errors = [json.loads(line) for line in side.state("audit.jsonl").splitlines()]
            errors = [a for a in errors if a["outcome"] == "error" and a["tool"] != "n4_connect"]
            texts = [json.loads(r)["result"]["content"][0]["text"] for r in side.replies]
            self.assertTrue(errors)
            for audit in errors:
                self.assertTrue(any(audit["batch_id"] in t and "in-doubt" in t for t in texts),
                                audit)

    def test_journal_faults_in_rollback_runs(self):
        """The journal turns unwritable mid-run: before a component intent, before the relink
        intent, and before the in-doubt result of a batch whose op 2 of 3 failed."""
        def scenario(side):
            side.connect()
            journal = os.path.join(side.state_dir, "journal.jsonl")

            def lose_after_add(name):  # once the add of `name` is applied, until restored
                def on_sync(op):
                    if op["nm"] == "a" and op["n"] == name and not os.path.isdir(journal):
                        os.rename(journal, journal + ".kept")
                        os.makedirs(journal, mode=0o700)
                side.fake.on_sync = on_sync

            def run(bid, after, fail=None):
                token = side.call("n4_rollback", batch_id=bid)["confirmation_token"]
                lose_after_add(after)
                if fail:
                    fail_nth(side, "syncTo", *fail)
                side.call("n4_rollback", batch_id=bid, dry_run=False, confirmation_token=token)
                side.fake.on_sync = side.fake.hook = None
                os.rmdir(journal)
                os.rename(journal + ".kept", journal)
                return bid
            nn = remove_group(side, "Jc")
            bids = [run(removed(side, nn), nn)]  # the top add is sent, Src's intent is not
            bids.append(run(removed(side, remove_group(side, "Jr")), "Tgt"))  # no relink sent
            three = [{"nm": "a", "h": "3", "n": n, "b": {"nm": "p", "t": "baja:Folder"}}
                     for n in ("A3", "B3", "C3")]
            craft(side, "af" * 16, three, {FOLDER: "3"})
            bids.append(run("af" * 16, "A3", fail=(2, (500, b"boom"))))
            side.call("n4_list_batches")
            for bid in bids:  # each stays in-doubt
                side.call("n4_rollback", batch_id=bid)
        self.assertSameRun(scenario)


def save_and(side, then):
    """The FakeStation's save, then `then()` (until the next restore)."""
    orig = side.fake._invoke

    def invoke(arg):
        out = orig(arg)
        if arg["a"] == "save":
            then()
        return out
    side.fake._invoke = invoke
    return orig


class TestSaveStationParity(ParityTestCase):
    """S14: n4_save_station. An unchanged config.bog is polled for the full 30 s on both
    servers, so the mismatch branch is left to the Go differential."""

    def test_no_home_and_every_token_refusal(self):
        def scenario(side):
            side.call("n4_save_station")  # not connected
            side.connect()
            token = side.call("n4_save_station").get("confirmation_token")
            side.call("n4_save_station", dry_run=False, confirmation_token=token)
            side.call("n4_save_station", dry_run=False, confirmation_token=token)  # spent
            side.call("n4_save_station", dry_run=False)
            side.call("n4_save_station", dry_run=False, confirmation_token="1.ab.cd")
            side.call("n4_list_batches")
        self.assertSameRun(scenario)

    def test_no_write_scope(self):
        def scenario(side):
            side.connect()
            side.write("n4_save_station")
        self.assertSameRun(scenario, write_scopes=[])

    def test_changed_missing_and_unreadable_config_bog_and_in_doubt(self):
        def scenario(side):
            side.connect()
            side.write("n4_save_station")  # changed: persisted
            side.write("n4_save_station")  # the second save changes it again
            os.remove(side.fake.config_path)
            side.write("n4_save_station")  # missing before the save
            side.write_bog(b"bog")
            orig = save_and(side, lambda: os.remove(side.fake.config_path))
            side.write("n4_save_station")  # unreadable after the save
            side.fake._invoke = orig
            side.write_bog(b"bog")
            token = side.call("n4_save_station")["confirmation_token"]
            side.write_bog(b"edited after the dry run")  # read again when it executes
            side.call("n4_save_station", dry_run=False, confirmation_token=token)
            for reply in ((500, b"boom"), (200, ERROR_FRAME)):
                fail_after(side, None, "invokeAction", reply)
                side.write("n4_save_station")
                side.fake.hook = None
            side.call("n4_list_batches")
        self.assertSameRun(scenario, home=b"bog-before")
