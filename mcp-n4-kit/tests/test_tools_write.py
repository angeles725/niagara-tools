import contextlib
import io
import json
import os
import stat
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_server  # noqa: E402  (module import: its test classes are not re-collected here)
from mcp_n4 import box, server, tools_write  # noqa: E402

WRITE_TOOLS = ["n4_create_component", "n4_set_slot", "n4_invoke_action", "n4_create_link"]
DESTRUCTIVE_TOOLS = ["n4_remove_component", "n4_rollback", "n4_save_station"]
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
        self.srv.ctx.close()
        self.srv = server.Server(**opts)
        self.addCleanup(self.srv.ctx.close)

    def connect_verified(self):
        self.connect(expected_station="FakeStation")
        self.box = self.srv.ctx.session.client

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

    def lines(self, name):
        path = os.path.join(self.state_dir, name)
        if not os.path.exists(path):
            return []
        with open(path) as fh:
            return [json.loads(line) for line in fh]


class TestMode(WriteTestCase):
    def test_read_only_server_has_no_write_tools_and_calling_one_is_32601(self):
        self.start_server(allow_writes=False)
        names = [t["name"] for t in self.srv.dispatch(
            test_server.rpc("tools/list"))["result"]["tools"]]
        for name in WRITE_TOOLS + DESTRUCTIVE_TOOLS:
            self.assertNotIn(name, names)
            res = self.srv.dispatch(test_server.rpc(
                "tools/call", {"name": name, "arguments": {}}))
            self.assertEqual(res["error"]["code"], -32601, name)

    def test_writes_allowed_server_lists_write_tools_with_annotations(self):
        tools = {t["name"]: t for t in self.srv.dispatch(
            test_server.rpc("tools/list"))["result"]["tools"]}
        for name in WRITE_TOOLS:
            ann = tools[name]["annotations"]
            self.assertFalse(ann["readOnlyHint"], name)
            self.assertFalse(ann["destructiveHint"], name)
            self.assertFalse(ann["openWorldHint"], name)
            self.assertEqual(ann.get("idempotentHint", False), name == "n4_set_slot", name)
            props = tools[name]["inputSchema"]["properties"]
            self.assertIn("dry_run", props)
            self.assertIn("confirmation_token", props)

    def test_destructive_tools_are_listed_with_destructive_annotations(self):
        tools = {t["name"]: t for t in self.srv.dispatch(
            test_server.rpc("tools/list"))["result"]["tools"]}
        for name in DESTRUCTIVE_TOOLS:
            ann = tools[name]["annotations"]
            self.assertFalse(ann["readOnlyHint"], name)
            self.assertTrue(ann["destructiveHint"], name)
            self.assertFalse(ann["openWorldHint"], name)
            props = tools[name]["inputSchema"]["properties"]
            self.assertIn("dry_run", props)
            self.assertIn("confirmation_token", props)

    def test_flags_have_documented_defaults_and_scope_is_repeatable(self):
        args = server.parse_args([])
        self.assertEqual((args.write_scope, args.state_dir, args.token_ttl, args.max_writes),
                         ([], None, 300, 200))
        args = server.parse_args(["--write-scope", "a", "--write-scope", "b", "--state-dir", "/x",
                                  "--token-ttl", "9", "--max-writes", "3"])
        self.assertEqual((args.write_scope, args.state_dir, args.token_ttl, args.max_writes),
                         (["a", "b"], "/x", 9, 3))

    def test_default_state_dir_is_under_the_home_state_dir(self):
        srv = server.Server(allow_writes=True)
        self.assertEqual(srv.ctx.write.journal.state_dir,
                         os.path.expanduser("~/.local/state/mcp-n4"))


class TestStateDirStartup(WriteTestCase):
    def loose_dir(self):
        os.makedirs(self.state_dir)
        os.chmod(self.state_dir, 0o777)
        return self.state_dir

    def test_write_mode_refuses_a_loose_existing_state_dir_and_leaves_it_alone(self):
        path = self.loose_dir()
        with self.assertRaises(tools_write.safety.SafetyError):
            server.Server(allow_writes=True, state_dir=path)
        self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o777)

    def test_read_only_mode_does_not_care_about_the_state_dir(self):
        server.Server(allow_writes=False, state_dir=self.loose_dir())

    def test_main_exits_2_with_a_clear_message(self):
        path = self.loose_dir()
        err = io.StringIO()
        with contextlib.redirect_stderr(err):
            code = server.main(["--allow-writes", "--state-dir", path])
        self.assertEqual(code, 2)
        self.assertIn("--state-dir", err.getvalue())

    def test_main_exits_2_for_a_chained_or_non_slot_write_scope(self):
        # H1 review R3-startup-exit-unproved: the refusal reaches the operator as exit 2.
        for bad in ("station:|slot:/A|h:1f", "station:|h:1f", "station:|slot:/A/../B"):
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                code = server.main(["--allow-writes", "--state-dir", self.state_dir,
                                    "--write-scope", bad])
            self.assertEqual(code, 2, bad)
            self.assertIn("--write-scope", err.getvalue(), bad)

    def test_main_rejects_malformed_station_values(self):
        for bad in ("nourl", "=https://h", "name=", ""):
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                code = server.main(["--station", bad])
            self.assertEqual(code, 2, bad)
            self.assertIn("--station expects NAME=URL", err.getvalue(), bad)


class TestIdentity(WriteTestCase):
    def test_write_on_an_unverified_session_is_refused_and_audited(self):
        self.connect()
        self.srv.ctx.session.identity_verified = False
        text = self.err("n4_create_component", parent_ord=FOLDER, name="Pump",
                        type="kitControl:NumericConst")
        self.assertIn("expected_station", text)
        self.assertEqual(self.children(), [])
        audit = self.lines("audit.jsonl")
        self.assertEqual((audit[-1]["outcome"], audit[-1]["tool"]),
                         ("refused", "n4_create_component"))

    def test_default_connect_verifies_identity_so_writes_are_allowed(self):
        self.connect()
        self.assertTrue(self.srv.ctx.session.identity_verified)

    def test_write_without_a_session_is_refused(self):
        self.assertIn("not connected", self.err(
            "n4_create_component", parent_ord=FOLDER, name="Pump",
            type="kitControl:NumericConst"))


class TestScope(WriteTestCase):
    def test_out_of_scope_parent_is_refused(self):
        self.connect_verified()
        text = self.err("n4_create_component", parent_ord="station:|slot:/",
                        name="Pump", type="kitControl:NumericConst")
        self.assertIn("outside the write scope", text)
        self.assertEqual(self.children(), [])

    def test_no_scope_configured_refuses_every_write(self):
        self.start_server(write_scopes=[])
        self.connect_verified()
        self.assertIn("no --write-scope", self.err(
            "n4_create_component", parent_ord=FOLDER, name="Pump",
            type="kitControl:NumericConst"))

    def test_prefix_boundary_is_respected(self):
        self.start_server(write_scopes=["station:|slot:/Fold"])
        self.connect_verified()
        self.assertIn("outside the write scope", self.err(
            "n4_create_component", parent_ord=FOLDER, name="Pump",
            type="kitControl:NumericConst"))

    def test_every_ord_of_a_link_must_be_in_scope(self):
        self.connect_verified()
        _, h = self.add("A", out=1)
        for src, tgt in (("station:|slot:/Other", FOLDER + "/A"),
                         (FOLDER + "/A", "station:|slot:/Other")):
            text = self.err("n4_create_link", source_ord=src, source_slot="out",
                            target_ord=tgt, target_slot="in10")
            self.assertIn("outside the write scope", text)


class TestTokenLayers(WriteTestCase):
    ARGS = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")

    def setUp(self):
        super().setUp()
        self.connect_verified()

    def test_dry_run_is_the_default_and_sends_no_mutation(self):
        plan = self.ok("n4_create_component", **self.ARGS)
        self.assertTrue(plan["dry_run"])
        self.assertEqual(self.children(), [])
        self.assertEqual(plan["plan"]["tool"], "n4_create_component")
        self.assertEqual(plan["plan"]["ops"], [{"nm": "a", "h": "3", "n": "Pump",
                                                "b": {"nm": "p", "t": "kitControl:NumericConst",
                                                      "s": []}}])
        self.assertEqual(plan["plan"]["inverse"][0]["nm"], "v")
        self.assertRegex(plan["plan_hash"], r"^[0-9a-f]{64}$")
        self.assertTrue(plan["confirmation_token"])
        self.assertGreater(plan["expires_at"], 0)
        self.assertEqual(self.lines("journal.jsonl"), [])

    def test_execute_without_a_token_is_refused(self):
        self.assertIn("confirmation_token", self.err("n4_create_component", dry_run=False,
                                                     **self.ARGS))
        self.assertEqual(self.children(), [])

    def test_a_token_is_single_use(self):
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.ok("n4_create_component", dry_run=False, confirmation_token=token, **self.ARGS)
        self.assertIn("already used", self.err("n4_create_component", dry_run=False,
                                               confirmation_token=token, **self.ARGS))
        self.assertEqual(self.children(), ["Pump"])

    def test_an_expired_token_is_refused(self):
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.srv.ctx.write.tokens.clock = lambda: 4e9
        self.assertIn("expired", self.err("n4_create_component", dry_run=False,
                                          confirmation_token=token, **self.ARGS))
        self.assertEqual(self.children(), [])

    def test_a_tampered_token_is_refused(self):
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        bad = token[:-1] + ("0" if token[-1] != "0" else "1")
        self.err("n4_create_component", dry_run=False, confirmation_token=bad, **self.ARGS)
        self.assertEqual(self.children(), [])

    def test_args_changed_after_the_token_was_issued_are_refused(self):
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        changed = dict(self.ARGS, name="Other")
        self.assertIn("does not match", self.err(
            "n4_create_component", dry_run=False, confirmation_token=token, **changed))
        self.assertEqual(self.children(), [])

    def test_a_token_for_another_tool_is_refused(self):
        _, h = self.add("A", out=1)
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.err("n4_set_slot", ord=FOLDER + "/A", slot="out", value=2.0,
                 value_type="baja:StatusNumeric", dry_run=False, confirmation_token=token)

    def test_station_state_changed_since_the_dry_run_is_refused(self):
        _, h = self.add("A", out=1)
        args = dict(ord=FOLDER + "/A", slot="out", value=5.0, value_type="baja:StatusNumeric")
        token = self.dry("n4_set_slot", **args)["confirmation_token"]
        self.box.set_slot(h, "out", box.bson_status_numeric(3))  # someone else writes
        self.assertIn("this plan", self.err("n4_set_slot", dry_run=False,
                                               confirmation_token=token, **args))


class TestTokenStationBinding(WriteTestCase):
    """F1 (audit 2026-10-03): a token vouches for the station session it was issued on."""
    ARGS = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")

    def test_the_plan_names_the_station_and_session_it_was_made_on(self):
        self.connect_verified()
        plan = self.dry("n4_create_component", **self.ARGS)["plan"]
        sess = self.srv.ctx.session
        self.assertEqual(plan["station"], {"name": "FakeStation", "base_url": self.fake.url,
                                           "session_id": sess.session_id})
        self.assertRegex(sess.session_id, r"^[0-9a-f]{32}$")

    def test_a_token_from_an_earlier_session_is_refused_after_a_reconnect(self):
        self.connect_verified()
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.connect_verified()  # same station, new session
        self.assertIn("does not match", self.err(
            "n4_create_component", dry_run=False, confirmation_token=token, **self.ARGS))
        self.assertEqual(self.children(), [])

    def test_a_token_from_station_a_cannot_execute_on_station_b(self):
        other = test_server.FakeStation(password=self.PASSWORD, station_name="Other").start()
        self.addCleanup(other.stop)
        self.stations = dict(self.stations, Other=other.url)
        self.start_server()
        self.connect_verified()
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.ok("n4_connect", station="Other")
        self.assertIn("does not match", self.err(
            "n4_create_component", dry_run=False, confirmation_token=token, **self.ARGS))
        self.assertEqual([c.name for c in other.folder.children], [])


class TestRollbackLiveContract(WriteTestCase):
    """B1200-G3: the rollback plan classifies snapshot children with the live
    reg/loadContract verdict when decisive, and falls back to the static table when
    the lookup is ambiguous (chain length 1) or unavailable (fail-open)."""

    def setUp(self):
        super().setUp()
        self.connect_verified()

    def nested_batch(self, child_type):
        """A remove batch whose inverse re-creates a folder holding one `child_type`."""
        self.box.add_component("3", "Live", "baja:Folder")
        gh = self.box.load_tree(FOLDER + "/Live", depth=1, **NO_SLEEP)[""]["h"]
        self.box.add_component(gh, "Kid", child_type)
        out = self.run_write("n4_remove_component", parent_ord=FOLDER, name="Live")
        return out["batch_id"]

    @staticmethod
    def reg_error_hook():
        """Answer every reg call with a station error frame, pass everything else."""
        def hook(frame):
            if frame["m"][0]["c"] == "reg":
                reply = {"p": "box", "n": frame["n"],
                         "m": [{"c": "reg", "k": "loadContract", "r": 0, "t": "e",
                                "b": {"isErr": True, "m": "boom"}}]}
                return 200, json.dumps(reply).encode(), {}
            return None
        return hook

    def test_the_live_verdict_splits_a_component_the_static_table_misses(self):
        # baja:TestComponent (a fake-station-only stand-in for a baja component missing
        # from COMPONENT_TYPES): the static heuristic says value, the live chain says
        # component, so the rollback re-creates the child with its own add op.
        batch = self.nested_batch("baja:TestComponent")
        plan = self.dry("n4_rollback", batch_id=batch)
        self.assertEqual([c["n"] for c in plan["plan"]["components"]], ["Kid"])
        self.assertNotIn("s", plan["plan"]["ops"][0]["b"])
        back = self.ok("n4_rollback", batch_id=batch, dry_run=False,
                       confirmation_token=plan["confirmation_token"])
        self.assertEqual(back["verdict"], "verified")

    def test_an_ambiguous_chain_falls_back_to_the_static_table(self):
        # baja:Folder's chain holds only itself (the live evidence anomaly): the live
        # verdict is None and the static table says component, so the child splits out.
        batch = self.nested_batch("baja:Folder")
        plan = self.dry("n4_rollback", batch_id=batch)
        self.assertEqual([c["n"] for c in plan["plan"]["components"]], ["Kid"])

    def test_a_failing_lookup_falls_open_to_the_static_table(self):
        self.fake.hook = self.reg_error_hook()
        batch = self.nested_batch("baja:TestComponent")
        plan = self.dry("n4_rollback", batch_id=batch)
        self.assertNotIn("components", plan["plan"])  # static verdict: a plain value
        self.assertIn("s", plan["plan"]["ops"][0]["b"])


class TestLimitsStartup(unittest.TestCase):
    """F16 (audit 2026-10-03): a zero or negative TTL or budget is refused at startup."""

    def test_zero_or_negative_ttl_and_budget_are_refused_by_the_parser(self):
        for flag in ("--token-ttl", "--max-writes"):
            for bad in ("0", "-5"):
                err = io.StringIO()
                with contextlib.redirect_stderr(err), self.assertRaises(SystemExit) as cm:
                    server.parse_args([flag, bad])
                self.assertEqual(cm.exception.code, 2, (flag, bad))
                self.assertIn(flag, err.getvalue())

    def test_the_write_state_refuses_them_too(self):
        with tempfile.TemporaryDirectory() as tmp:
            for kw in ({"token_ttl": 0}, {"max_writes": 0}, {"token_ttl": -1}):
                with self.assertRaises(ValueError, msg=kw):
                    tools_write.WriteState(["station:|slot:/A"], tmp, **kw)


class TestBudget(WriteTestCase):
    MAX_WRITES = 1

    def test_writes_beyond_the_session_budget_are_refused(self):
        self.connect_verified()
        self.run_write("n4_create_component", parent_ord=FOLDER, name="A",
                       type="kitControl:NumericConst")
        text = self.err("n4_create_component", parent_ord=FOLDER, name="B",
                        type="kitControl:NumericConst")
        self.assertIn("budget", text)
        self.assertEqual(self.children(), ["A"])

    def test_reconnecting_keeps_the_process_budget(self):
        # F5 (audit 2026-10-03): the budget is per server process. A per-session budget
        # let a reconnect reset --max-writes, so it bounded nothing across sessions.
        self.connect_verified()
        self.run_write("n4_create_component", parent_ord=FOLDER, name="A",
                       type="kitControl:NumericConst")
        self.connect_verified()
        text = self.err("n4_create_component", parent_ord=FOLDER, name="B",
                        type="kitControl:NumericConst")
        self.assertIn("write budget exhausted", text)
        self.assertIn("server process", text)
        self.assertEqual(self.children(), ["A"])


class TestCreateComponent(WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()

    def test_happy_path_is_verified_with_annotation(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst",
                             wire_sheet={"x": 4, "y": 6, "w": 10})
        self.assertEqual(out["verdict"], "verified")
        self.assertEqual(out["accepted"]["nn"], "Pump")
        self.assertEqual(out["observed"], {"name": "Pump", "type": "kitControl:NumericConst",
                                           "wsAnnotation": "4,6,10"})
        node = self.fake.folder.child("Pump")
        self.assertEqual(node.child("wsAnnotation").value, "4,6,10")
        self.assertEqual(out["inverse"], [{"nm": "v", "h": "3", "n": "Pump"}])

    def test_collision_rename_is_followed_in_read_back_and_inverse(self):
        self.add("Pump")
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        self.assertEqual((out["accepted"]["nn"], out["verdict"]), ("Pump1", "verified"))
        self.assertEqual(out["inverse"], [{"nm": "v", "h": "3", "n": "Pump1"}])

    def test_bad_name_or_type_is_refused(self):
        for kw in ({"name": "../x"}, {"name": "a b"}, {"name": ""}, {"type": "NoModule"},
                   {"type": "a:b:c"}):
            args = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")
            args.update(kw)
            self.err("n4_create_component", **args)
        self.assertEqual(self.children(), [])

    def test_bad_wire_sheet_is_refused(self):
        for ws in ({"x": 1}, {"x": "a", "y": 1}, {"x": 1, "y": 2, "w": True}):
            self.err("n4_create_component", parent_ord=FOLDER, name="P",
                     type="kitControl:NumericConst", wire_sheet=ws)

    def test_read_back_mismatch_is_reported_not_raised(self):
        def drop_annotation(op):
            node = self.fake.folder.child("Pump")
            node.children = [c for c in node.children if c.name != "wsAnnotation"]
        self.fake.on_sync = drop_annotation
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst", wire_sheet={"x": 1, "y": 2})
        self.assertEqual(out["verdict"], "mismatch")
        self.assertRegex(out["batch_id"], r"^[0-9a-f]{32}$")


class TestSetSlot(WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()
        self.nn, self.h = self.add("Calc", out=1.0)
        self.ord = FOLDER + "/" + self.nn

    def test_status_numeric_happy_path_and_inverse_restores_previous(self):
        plan = self.dry("n4_set_slot", ord=self.ord, slot="out", value=5.5,
                        value_type="baja:StatusNumeric")
        self.assertEqual(plan["plan"]["ops"],
                         [{"nm": "s", "h": self.h, "n": "out",
                           "b": box.bson_status_numeric(5.5, "0")}])
        self.assertEqual(plan["plan"]["inverse"],
                         [{"nm": "s", "h": self.h, "n": "out",
                           "b": box.bson_status_numeric(1.0, "0")}])
        out = self.run_write("n4_set_slot", ord=self.ord, slot="out", value=5.5,
                             value_type="baja:StatusNumeric")
        self.assertEqual(out["verdict"], "verified")
        self.assertEqual(out["requested"], {"value": 5.5, "status": "0"})
        self.assertEqual(out["observed"], {"value": 5.5, "status": "0"})

    def test_status_boolean_is_written_whole(self):
        self.box.set_slot(self.h, "flag", box.bson_status_boolean(False))
        out = self.run_write("n4_set_slot", ord=self.ord, slot="flag", value=True,
                             value_type="baja:StatusBoolean")
        self.assertEqual((out["verdict"], out["observed"]), ("verified",
                                                              {"value": True, "status": "0"}))

    def test_plain_types_happy_path(self):
        for slot, type_, first, new in (("d", "baja:Double", box.bson_double(2), 3.5),
                                        ("b", "baja:Boolean", box.bson_bool(True), False),
                                        ("s", "baja:String", {"nm": "p", "t": "baja:String",
                                                              "v": "old"}, "new")):
            self.box.set_slot(self.h, slot, first)
            out = self.run_write("n4_set_slot", ord=self.ord, slot=slot, value=new,
                                 value_type=type_)
            self.assertEqual((out["verdict"], out["observed"]), ("verified", new), slot)

    def test_value_or_status_child_path_is_refused_with_the_reason(self):
        for slot in ("out/value", "out/status"):
            text = self.err("n4_set_slot", ord=self.ord, slot=slot, value=2.0,
                            value_type="baja:Double")
            self.assertIn("whole", text)
            self.assertIn("status", text)
        self.assertEqual(self.fake.by_handle[self.h].child("out").child("value").value, "1.0")

    def test_a_status_for_a_plain_value_type_is_refused(self):
        # F13 (audit 2026-10-03): untested branch.
        self.box.set_slot(self.h, "d", box.bson_double(2))
        text = self.err("n4_set_slot", ord=self.ord, slot="d", value=3.0,
                        value_type="baja:Double", status="0")
        self.assertIn("status only applies", text)

    def test_bad_values_types_and_status_are_refused(self):
        for kw in ({"value": True, "value_type": "baja:Double"},
                   {"value": float("nan"), "value_type": "baja:Double"},
                   {"value": "x", "value_type": "baja:Boolean"},
                   {"value": 1, "value_type": "baja:String"},
                   {"value": 1.0, "value_type": "baja:Int"},
                   {"value": 1.0, "value_type": "baja:StatusNumeric", "status": "zz"}):
            self.err("n4_set_slot", ord=self.ord, slot="out", **kw)

    def test_slot_at_its_type_default_is_not_listed_by_the_station_so_it_is_refused(self):
        self.box.set_slot(self.h, "zero", box.bson_double(0))  # default values are omitted
        text = self.err("n4_set_slot", ord=self.ord, slot="zero", value=1.0,
                        value_type="baja:Double")
        self.assertIn("not found", text)
        # F10 (audit 2026-10-03): the refusal explains why and names the routes.
        self.assertIn("type default", text)
        self.assertIn("n4_read_slots", text)
        self.assertIn("Workbench", text)

    def test_a_nested_status_slot_plans_its_real_previous_value(self):
        # F3 (audit 2026-10-03): `grp/sp` needs its value/status children loaded, else the
        # previous value (and the journaled inverse) silently became the type default.
        self.box.set_slot(self.h, "grp", {"nm": "p", "t": "baja:Struct", "s": [
            dict(box.bson_status_numeric(7.5), n="sp")]})
        args = dict(ord=self.ord, slot="grp/sp", value=9.0, value_type="baja:StatusNumeric")
        plan = self.dry("n4_set_slot", **args)
        self.assertEqual(plan["plan"]["inverse"],
                         [{"nm": "s", "h": self.h, "n": "grp/sp",
                           "b": box.bson_status_numeric(7.5, "0")}])
        out = self.ok("n4_set_slot", dry_run=False,
                      confirmation_token=plan["confirmation_token"], **args)
        self.assertEqual((out["verdict"], out["observed"]),
                         ("verified", {"value": 9.0, "status": "0"}))

    def test_unknown_slot_and_type_mismatch_with_the_slot_are_refused(self):
        self.assertIn("not found", self.err("n4_set_slot", ord=self.ord, slot="nope",
                                            value=1.0, value_type="baja:Double"))
        self.assertIn("baja:StatusNumeric", self.err(
            "n4_set_slot", ord=self.ord, slot="out", value=True,
            value_type="baja:StatusBoolean"))

    def test_read_back_mismatch_is_reported_not_raised(self):
        def tamper(op):
            self.fake.by_handle[self.h].child("out").child("value").value = "99.0"
        self.fake.on_sync = tamper
        out = self.run_write("n4_set_slot", ord=self.ord, slot="out", value=5.0,
                             value_type="baja:StatusNumeric")
        self.assertEqual(out["verdict"], "mismatch")
        self.assertEqual(out["observed"]["value"], 99.0)
        self.assertEqual([e["phase"] for e in self.lines("journal.jsonl")],
                         ["intent", "result"])  # still journaled


class TestInvokeAction(WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()
        self.nn, self.h = self.add("Sp", "control:NumericWritable")
        self.ord = FOLDER + "/" + self.nn

    def test_actions_outside_the_allowlist_are_refused(self):
        for action in ("save", "restart", "emergencyOverride", "emergencyAuto", "x"):
            text = self.err("n4_invoke_action", ord=self.ord, action=action)
            self.assertIn("not allowed in this version", text)
            self.assertEqual("n4_save_station" in text, action == "save", action)
        self.assertEqual(self.fake.invoked, [])
        self.assertEqual(self.fake.saves, 0)

    def test_set_happy_path_verifies_fallback_and_inverse_restores_it(self):
        plan = self.dry("n4_invoke_action", ord=self.ord, action="set", arg=7.0,
                        arg_type="baja:Double")
        self.assertEqual(plan["plan"]["ops"][0]["arg"],
                         {"h": self.h, "a": "set", "b": box.bson_double(7.0)})
        self.assertEqual(plan["plan"]["inverse"],
                         [{"nm": "s", "h": self.h, "n": "fallback",
                           "b": box.bson_status_numeric(0.0, "0")}])
        out = self.run_write("n4_invoke_action", ord=self.ord, action="set", arg=7.0,
                             arg_type="baja:Double")
        self.assertEqual((out["verdict"], out["observed"]), ("verified", {"fallback": 7.0}))
        self.assertEqual(self.fake.invoked, [("set", self.h)])

    def test_set_needs_an_arg_and_the_others_take_none(self):
        self.err("n4_invoke_action", ord=self.ord, action="set")
        self.err("n4_invoke_action", ord=self.ord, action="set", arg=1.0, arg_type="baja:Int")
        self.err("n4_invoke_action", ord=self.ord, action="auto", arg=1.0, arg_type="baja:Double")

    def test_active_has_no_inverse_and_the_plan_says_so(self):
        plan = self.dry("n4_invoke_action", ord=self.ord, action="active")
        self.assertEqual(plan["plan"]["inverse"], [])
        self.assertTrue(any("no inverse" in n for n in plan["plan"]["notes"]))

    def test_state_changing_actions_are_unverified_not_verified(self):
        for action in ("active", "inactive", "auto"):
            out = self.run_write("n4_invoke_action", ord=self.ord, action=action)
            self.assertEqual(out["verdict"], "unverified", action)
        self.assertEqual([a for a, _ in self.fake.invoked], ["active", "inactive", "auto"])

    def test_set_read_back_mismatch_is_reported(self):
        plan = self.dry("n4_invoke_action", ord=self.ord, action="set", arg=7.0,
                        arg_type="baja:Double")
        orig = self.fake._invoke

        def lying(arg):
            orig(arg)
            self.fake.by_handle[self.h].child("fallback").child("value").value = "1.0"
        self.fake._invoke = lying
        out = self.ok("n4_invoke_action", ord=self.ord, action="set", arg=7.0,
                      arg_type="baja:Double", dry_run=False,
                      confirmation_token=plan["confirmation_token"])
        self.assertEqual(out["verdict"], "mismatch")


class TestCreateLink(WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()
        self.src, self.src_h = self.add("Src", out=1.0)
        self.tgt, self.tgt_h = self.add("Tgt")
        self.args = dict(source_ord=FOLDER + "/" + self.src, source_slot="out",
                         target_ord=FOLDER + "/" + self.tgt, target_slot="in10")

    def test_happy_path_is_verified_and_inverse_removes_the_link(self):
        plan = self.dry("n4_create_link", **self.args)
        self.assertEqual(plan["plan"]["ops"], [{"ssc": "checkLinks", "arg": {
            "s": self.src_h, "ss": "out", "t": self.tgt_h, "ts": "in10", "c": True}}])
        self.assertEqual(self.fake.by_handle[self.tgt_h].child("Link"), None)
        out = self.run_write("n4_create_link", **self.args)
        self.assertEqual(out["verdict"], "verified")
        self.assertEqual(out["observed"], {"link": "Link", "sourceOrd": "h:" + self.src_h,
                                           "sourceSlotName": "out", "targetSlotName": "in10"})
        self.assertEqual(out["inverse"], [{"nm": "v", "h": self.tgt_h, "n": "Link"}])

    def test_bad_slot_names_are_refused(self):
        self.err("n4_create_link", **dict(self.args, target_slot="in 10"))
        self.assertIsNone(self.fake.by_handle[self.tgt_h].child("Link"))

    def test_read_back_mismatch_is_reported(self):
        orig = self.fake._check_link

        def lying(arg):
            res = orig(arg)
            link = self.fake.by_handle[self.tgt_h].child(res[0]["s"])
            link.child("sourceSlotName").value = "other"
            return res
        self.fake._check_link = lying
        out = self.run_write("n4_create_link", **self.args)
        self.assertEqual(out["verdict"], "mismatch")


class TestCreateLinkRejected(WriteTestCase):
    def test_station_rejection_is_failed_with_reason_and_nothing_journaled(self):
        self.connect_verified()
        src, _ = self.add("Src", out=1.0)
        tgt, tgt_h = self.add("Tgt")
        self.fake._check_link = lambda arg: [{"v": False, "r": "type mismatch", "s": None}]
        args = dict(source_ord=FOLDER + "/" + src, source_slot="out",
                    target_ord=FOLDER + "/" + tgt, target_slot="in10")
        out = self.run_write("n4_create_link", **args)
        self.assertEqual(out["verdict"], "failed")
        self.assertEqual(out["accepted"]["r"], "type mismatch")
        self.assertIsNone(self.fake.by_handle[tgt_h].child("Link"))
        self.assertEqual(out["inverse"], [])
        intent, result = self.lines("journal.jsonl")
        self.assertEqual((intent["phase"], result["phase"], result["verdict"], result["inverse"]),
                         ("intent", "result", "failed", []))
        self.assertEqual(self.srv.ctx.write.journal.read(out["batch_id"])["state"], "completed")


class TestLinkReplies(WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()
        self.src, self.src_h = self.add("Src", out=1.0)
        self.tgt, self.tgt_h = self.add("Tgt")
        self.args = dict(source_ord=FOLDER + "/" + self.src, source_slot="out",
                         target_ord=FOLDER + "/" + self.tgt, target_slot="in10")

    def journal_view(self, out):
        view = self.srv.ctx.write.journal.read(out["batch_id"])
        self.assertIsNotNone(view, "every returned batch_id has a journal entry")
        return view

    def test_an_explicit_refusal_is_a_failed_result_with_an_empty_inverse(self):
        self.fake._check_link = lambda arg: [{"v": False, "r": "type mismatch", "s": None}]
        out = self.run_write("n4_create_link", **self.args)
        view = self.journal_view(out)
        self.assertEqual((out["verdict"], view["verdict"], view["inverse"], view["state"]),
                         ("failed", "failed", [], "completed"))

    def test_v_true_without_a_link_name_is_ambiguous_and_finds_the_link_by_its_slots(self):
        orig = self.fake._check_link
        self.fake._check_link = lambda arg: [dict(orig(arg)[0], s=None)]
        out = self.run_write("n4_create_link", **self.args)
        view = self.journal_view(out)
        self.assertEqual((out["verdict"], view["verdict"], view["state"]),
                         ("unverified", "unverified", "in-doubt"))
        self.assertTrue(out["in_doubt"])
        self.assertEqual(out["observed"]["link"], "Link")
        self.assertEqual(out["inverse"], [{"nm": "v", "h": self.tgt_h, "n": "Link"}])
        self.assertEqual(view["inverse"], out["inverse"])

    def test_an_ambiguous_reply_with_no_link_found_records_an_empty_inverse(self):
        self.fake._check_link = lambda arg: [{"v": True, "r": None, "s": ""}]
        out = self.run_write("n4_create_link", **self.args)
        view = self.journal_view(out)
        self.assertEqual((out["verdict"], out["observed"], out["inverse"], view["state"]),
                         ("unverified", None, [], "in-doubt"))

    def test_a_malformed_reply_is_in_doubt_not_a_crash(self):
        for junk in ("junk", [], [None], [[]]):
            self.fake._check_link = lambda arg, j=junk: j
            out = self.run_write("n4_create_link", **self.args)
            view = self.journal_view(out)
            self.assertEqual((out["verdict"], view["state"]), ("unverified", "in-doubt"), junk)
            self.assertIn("readback_error", out)


class TestNoneInverse(WriteTestCase):
    def test_a_none_inverse_is_normalized_to_an_empty_list(self):
        self.connect_verified()
        nn, _ = self.add("Pump")
        impl = tools_write._IMPLS["n4_invoke_action"]._replace(
            inverse=lambda planned, replies: None)
        with mock.patch.dict(tools_write._IMPLS, {"n4_invoke_action": impl}):
            out = self.run_write("n4_invoke_action", ord=FOLDER + "/" + nn, action="auto")
        self.assertEqual(out["inverse"], [])
        result = [e for e in self.lines("journal.jsonl") if e["phase"] == "result"][-1]
        self.assertEqual(result["inverse"], [])


class TestLooseStateFileBeforeSend(WriteTestCase):
    ARGS = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")

    def setUp(self):
        super().setUp()
        self.connect_verified()
        self.run_write("n4_create_component", parent_ord=FOLDER, name="Seed",
                       type="kitControl:NumericConst")  # creates journal + audit, mode 0600

    def test_a_state_file_loosened_while_running_fails_before_any_send(self):
        for filename in ("journal.jsonl", "audit.jsonl"):
            path = os.path.join(self.state_dir, filename)
            plan = self.dry("n4_create_component", **self.ARGS)
            os.chmod(path, 0o644)
            kw = dict(dry_run=False, confirmation_token=plan["confirmation_token"], **self.ARGS)
            before = (self.children(), self.srv.ctx.write.writes_executed)
            self.assertIn("chmod 600", self.err("n4_create_component", **kw), filename)
            self.assertEqual((self.children(), self.srv.ctx.write.writes_executed), before)
            os.chmod(path, 0o600)
            out = self.ok("n4_create_component", **kw)  # the token was not spent
            self.assertEqual(out["verdict"], "verified")
            self.box.remove_component("3", "Pump")


class TestNestedReadbackPairing(WriteTestCase):
    def test_mismatched_components_and_created_lists_are_a_verdict_mismatch(self):
        spec = {"n": "K", "parent_path": "", "top": 0, "b": {"t": "kitControl:NumericConst"}}
        planned = tools_write.Planned([], [], [], {
            "ord_of": {}, "relinks": None, "rollback_of": "b1",
            "components": [spec, dict(spec, n="L")], "created": [FOLDER + "/K"]})
        client = mock.Mock()
        client.load_tree.return_value = {"K": {"t": "kitControl:NumericConst"},
                                         "K/wsAnnotation": {"v": None}}
        *_, verdict = tools_write._rollback_readback(client, {}, planned, [], [])
        self.assertEqual(verdict, "mismatch")


class TestJournalAndAudit(WriteTestCase):
    ARGS = dict(parent_ord=FOLDER, name="Pump", type="kitControl:NumericConst")

    def setUp(self):
        super().setUp()
        self.connect_verified()

    def test_journal_holds_a_write_ahead_intent_and_a_result(self):
        out = self.run_write("n4_create_component", **self.ARGS)
        intent, result = self.lines("journal.jsonl")
        self.assertEqual(set(intent), {"batch_id", "ts", "tool", "ops", "inverse_plan",
                                       "station_name", "phase", "targets"})
        self.assertEqual(intent["targets"], {FOLDER: "3"})
        self.assertEqual((intent["phase"], intent["batch_id"]), ("intent", out["batch_id"]))
        self.assertRegex(intent["batch_id"], r"^[0-9a-f]{32}$")
        self.assertEqual((intent["tool"], intent["station_name"]),
                         ("n4_create_component", "FakeStation"))
        self.assertEqual(intent["inverse_plan"], [{"nm": "v", "h": "3", "n": "Pump"}])
        self.assertEqual(intent["ops"][0]["nm"], "a")
        self.assertEqual(set(result), {"batch_id", "ts", "phase", "accepted", "inverse",
                                       "verdict"})
        self.assertEqual((result["phase"], result["batch_id"], result["verdict"]),
                         ("result", out["batch_id"], "verified"))
        self.assertEqual(result["inverse"], [{"nm": "v", "h": "3", "n": "Pump"}])
        self.assertEqual(result["accepted"], {"id": "a", "nn": "Pump"})

    def test_the_result_records_the_server_assigned_name_not_the_planned_one(self):
        self.add("Pump")
        out = self.run_write("n4_create_component", **self.ARGS)
        intent, result = self.lines("journal.jsonl")
        self.assertEqual(intent["inverse_plan"][0]["n"], "Pump")
        self.assertEqual(result["inverse"][0]["n"], "Pump1")
        view = self.srv.ctx.write.journal.read(out["batch_id"])
        self.assertEqual((view["state"], view["inverse"][0]["n"]), ("completed", "Pump1"))

    def test_the_intent_is_on_disk_before_the_station_receives_the_op(self):
        seen = []
        self.fake.on_sync = lambda op: seen.append(self.lines("journal.jsonl"))
        self.run_write("n4_create_component", **self.ARGS)
        self.assertEqual([[e["phase"] for e in lines] for lines in seen], [["intent"]])

    def test_an_intent_that_cannot_be_written_sends_nothing_and_keeps_the_token_spent(self):
        plan = self.dry("n4_create_component", **self.ARGS)

        def boom(entry):
            raise OSError("disk full")
        self.srv.ctx.write.journal.append = boom
        kw = dict(dry_run=False, confirmation_token=plan["confirmation_token"], **self.ARGS)
        self.assertIn("nothing was sent", self.err("n4_create_component", **kw))
        self.assertEqual(self.children(), [])
        self.assertEqual(self.srv.ctx.write.writes_executed, 0)
        del self.srv.ctx.write.journal.append
        self.assertIn("already used", self.err("n4_create_component", **kw))
        self.assertEqual(self.children(), [])

    def test_a_send_exception_after_the_token_was_spent_leaves_an_in_doubt_batch(self):
        plan = self.dry("n4_create_component", **self.ARGS)
        self.fake.hook = lambda frame: (500, b"boom", {}) if any(
            m.get("b", {}).get("sck") == "syncTo" for m in frame["m"]) else None
        kw = dict(dry_run=False, confirmation_token=plan["confirmation_token"], **self.ARGS)
        text = self.err("n4_create_component", **kw)
        self.assertIn("in-doubt", text)
        intent = self.lines("journal.jsonl")[0]
        self.assertIn(intent["batch_id"], text)
        self.assertEqual([e["phase"] for e in self.lines("journal.jsonl")], ["intent"])
        self.assertEqual(self.srv.ctx.write.journal.read(intent["batch_id"])["state"], "in-doubt")
        self.fake.hook = None
        self.assertIn("already used", self.err("n4_create_component", **kw))
        self.assertEqual(self.lines("audit.jsonl")[-2]["batch_id"], intent["batch_id"])

    def test_a_result_that_cannot_be_written_is_a_warning_with_the_inverse(self):
        real = self.srv.ctx.write.journal.append

        def flaky(entry):
            if entry["phase"] == "result":
                raise OSError("disk full")
            real(entry)
        self.srv.ctx.write.journal.append = flaky
        out = self.run_write("n4_create_component", **self.ARGS)
        self.assertEqual(out["verdict"], "verified")
        self.assertTrue(any("in-doubt" in w for w in out["warnings"]), out)
        self.assertEqual(out["inverse"], [{"nm": "v", "h": "3", "n": "Pump"}])

    def test_a_read_back_box_error_is_failed_with_the_batch_id(self):
        def break_reads(op):
            def boom(*a, **kw):
                raise box.BoxError("no load event for x after 6 polls")
            self.box.load_tree = boom
        self.fake.on_sync = break_reads
        out = self.run_write("n4_create_component", **self.ARGS)
        self.assertEqual(out["verdict"], "failed")
        self.assertIn("no load event", out["readback_error"])
        self.assertRegex(out["batch_id"], r"^[0-9a-f]{32}$")
        self.assertEqual(self.lines("journal.jsonl")[-1]["verdict"], "failed")

    def test_a_read_back_that_raises_anything_else_is_failed_not_a_crash(self):
        _, h = self.add("Calc", out=1.0)
        self.fake.on_sync = lambda op: setattr(
            self.fake.by_handle[h].child("out").child("value"), "value", "not-a-number")
        out = self.run_write("n4_set_slot", ord=FOLDER + "/Calc", slot="out", value=2.0,
                             value_type="baja:StatusNumeric")
        self.assertEqual(out["verdict"], "failed")
        self.assertIn("ValueError", out["readback_error"])
        self.assertRegex(out["batch_id"], r"^[0-9a-f]{32}$")

    def test_an_unwritable_audit_log_is_a_warning_not_an_error(self):
        def boom(entry):
            raise OSError("read-only")
        self.srv.ctx.write.audit.append = boom
        out = self.run_write("n4_create_component", **self.ARGS)
        self.assertEqual(out["verdict"], "verified")
        self.assertEqual(out["warnings"], ["the audit log could not be written"])

    def test_audit_records_planned_executed_and_refused_calls(self):
        token = self.dry("n4_create_component", **self.ARGS)["confirmation_token"]
        self.ok("n4_create_component", dry_run=False, confirmation_token=token, **self.ARGS)
        self.err("n4_create_component", parent_ord="station:|slot:/", name="X",
                 type="kitControl:NumericConst")
        planned, executed, refused = self.lines("audit.jsonl")
        self.assertEqual((planned["outcome"], planned["dry_run"], planned["batch_id"]),
                         ("planned", True, None))
        self.assertEqual((executed["outcome"], executed["dry_run"]), ("executed", False))
        self.assertRegex(executed["batch_id"], r"^[0-9a-f]{32}$")
        self.assertEqual((refused["outcome"], refused["batch_id"]), ("refused", None))
        self.assertIn("outside the write scope", refused["reason"])
        for line in (planned, executed, refused):
            self.assertEqual(set(line), {"ts", "tool", "batch_id", "dry_run", "outcome",
                                         "reason", "args_redacted"})

    def test_audit_and_outputs_hold_no_secret_value(self):
        extra = dict(password="hunter2", api_secret="s3cr3t-x")
        plan = self.dry("n4_create_component", **dict(self.ARGS, **extra))
        token = plan["confirmation_token"]
        out = self.ok("n4_create_component", dry_run=False, confirmation_token=token,
                      **dict(self.ARGS, **extra))
        self.err("n4_create_component", dry_run=False, confirmation_token=token,
                 **dict(self.ARGS, **extra))
        with open(os.path.join(self.state_dir, "audit.jsonl")) as fh:
            audit = fh.read()
        for secret in ("hunter2", "s3cr3t-x", token, self.PASSWORD):
            self.assertNotIn(secret, audit)
        self.assertNotIn(self.PASSWORD, json.dumps([plan, out]))
        self.assertIn(tools_write.safety.REDACTED, audit)

    def test_state_files_are_private(self):
        self.run_write("n4_create_component", **self.ARGS)
        self.assertEqual(stat.S_IMODE(os.stat(self.state_dir).st_mode), 0o700)
        for name in ("journal.jsonl", "audit.jsonl"):
            mode = stat.S_IMODE(os.stat(os.path.join(self.state_dir, name)).st_mode)
            self.assertEqual(mode, 0o600, name)


if __name__ == "__main__":
    unittest.main()
