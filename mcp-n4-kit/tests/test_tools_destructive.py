import contextlib
import hashlib
import io
import os
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_tools_write as ttw  # noqa: E402  (module import: its tests are not re-collected)
from mcp_n4 import box, safety, server  # noqa: E402

FOLDER = ttw.FOLDER
NO_SLEEP = ttw.NO_SLEEP


class DestructiveCase(ttw.WriteTestCase):
    def setUp(self):
        super().setUp()
        self.connect_verified()

    def group(self, name="Grp"):
        """A baja:Folder under FOLDER with a wsAnnotation, two NumericConsts and a link."""
        nn = self.box.add_component("3", name, "baja:Folder",
                                    ws=box.ws_annotation(1, 2, 3))["nn"]
        gh = self.box.load_tree(FOLDER + "/" + nn, depth=1, **NO_SLEEP)[""]["h"]
        self.box.add_component(gh, "Src", "kitControl:NumericConst")
        self.box.add_component(gh, "Tgt", "kitControl:NumericConst")
        nodes = self.box.load_tree(FOLDER + "/" + nn, depth=2, **NO_SLEEP)
        src_h, tgt_h = nodes["Src"]["h"], nodes["Tgt"]["h"]
        self.box.set_slot(src_h, "out", box.bson_status_numeric(4.5))
        self.box.check_links(src_h, "out", tgt_h, "in10")
        return nn, gh

    def journal(self):
        return self.srv.ctx.write.journal


class TestRemoveComponent(DestructiveCase):
    def test_plan_snapshots_the_subtree_as_data_and_sends_nothing(self):
        nn, gh = self.group()
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.assertEqual(plan["plan"]["ops"], [{"nm": "v", "h": "3", "n": nn}])
        self.assertIn(nn, self.children())
        add, relink = plan["plan"]["inverse"]
        self.assertEqual((add["nm"], add["h"], add["n"], add["b"]["t"]), ("a", "3", nn, "baja:Folder"))
        slots = {c["n"]: c for c in add["b"]["s"]}
        self.assertEqual(slots["wsAnnotation"]["v"], "1,2,3")
        self.assertEqual({c["n"] for c in slots["Src"]["s"]}, {"out"})
        self.assertNotIn("Link", {c["n"] for c in slots["Tgt"].get("s", [])})  # links are separate
        self.assertNotIn("h", slots["Src"])  # handles are not data
        self.assertEqual(relink["relink"], {"source_path": "Src", "source_slot": "out",
                                            "target_path": "Tgt", "target_slot": "in10"})
        self.assertTrue(any("depth 3" in n and "not a full restore" in n
                            for n in plan["plan"]["notes"]))

    def test_links_from_outside_the_subtree_are_reported_as_not_restorable(self):
        nn, gh = self.group()
        other, other_h = self.add("Other", out=1.0)
        tgt_h = self.box.load_tree(FOLDER + "/" + nn, depth=2, **NO_SLEEP)["Tgt"]["h"]
        self.box.check_links(other_h, "out", tgt_h, "in11")
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.assertEqual(len([e for e in plan["plan"]["inverse"] if "relink" in e]), 1)
        self.assertTrue(any("outside" in n and "not restored" in n for n in plan["plan"]["notes"]))

    def test_execute_removes_verifies_and_journals_the_snapshot_as_the_inverse(self):
        nn, gh = self.group()
        out = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.assertEqual(out["verdict"], "verified")
        self.assertNotIn(nn, self.children())
        view = self.journal().read(out["batch_id"])
        self.assertEqual(view["inverse"][0]["nm"], "a")
        self.assertEqual(view["state"], "completed")

    def test_missing_child_bad_name_and_scope_are_refused(self):
        self.assertIn("not found", self.err("n4_remove_component", parent_ord=FOLDER, name="Nope"))
        self.err("n4_remove_component", parent_ord=FOLDER, name="../x")
        self.assertIn("outside the write scope", self.err(
            "n4_remove_component", parent_ord="station:|slot:/", name="Folder"))

    def test_read_back_reports_a_child_that_is_still_there(self):
        self.add("Keep")
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name="Keep")
        orig = self.fake._sync
        self.fake._sync = lambda op: [] if op["nm"] == "v" else orig(op)
        out = self.ok("n4_remove_component", parent_ord=FOLDER, name="Keep", dry_run=False,
                      confirmation_token=plan["confirmation_token"])
        self.assertEqual(out["verdict"], "mismatch")


class TestRollback(DestructiveCase):
    def rollback(self, batch_id):
        return self.run_write("n4_rollback", batch_id=batch_id)

    def test_rollback_of_a_create_removes_the_component_as_a_linked_batch(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        plan = self.dry("n4_rollback", batch_id=out["batch_id"])
        self.assertEqual(plan["plan"]["ops"], [{"nm": "v", "h": "3", "n": "Pump"}])
        self.assertEqual(self.children(), ["Pump"])
        back = self.ok("n4_rollback", batch_id=out["batch_id"], dry_run=False,
                       confirmation_token=plan["confirmation_token"])
        self.assertEqual((back["verdict"], self.children()), ("verified", []))
        view = self.journal().read(back["batch_id"])
        self.assertEqual((view["rollback_of"], view["tool"]), (out["batch_id"], "n4_rollback"))

    def test_rollback_follows_the_server_assigned_name(self):
        self.add("Pump")
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        self.rollback(out["batch_id"])
        self.assertEqual(self.children(), ["Pump"])

    def test_rollback_of_a_set_slot_restores_the_previous_value(self):
        nn, h = self.add("Calc", out=1.0)
        out = self.run_write("n4_set_slot", ord=FOLDER + "/" + nn, slot="out", value=9.0,
                             value_type="baja:StatusNumeric")
        back = self.rollback(out["batch_id"])
        self.assertEqual(back["verdict"], "verified")
        self.assertEqual(self.fake.by_handle[h].child("out").child("value").value, "1.0")

    def test_rollback_of_a_link_removes_it(self):
        src, _ = self.add("Src", out=1.0)
        tgt, tgt_h = self.add("Tgt")
        out = self.run_write("n4_create_link", source_ord=FOLDER + "/" + src, source_slot="out",
                             target_ord=FOLDER + "/" + tgt, target_slot="in10")
        back = self.rollback(out["batch_id"])
        self.assertEqual(back["verdict"], "verified")
        self.assertIsNone(self.fake.by_handle[tgt_h].child("Link"))

    def test_rollback_of_invoke_set_restores_the_fallback(self):
        nn, h = self.add("Sp", "control:NumericWritable")
        out = self.run_write("n4_invoke_action", ord=FOLDER + "/" + nn, action="set", arg=7.0,
                             arg_type="baja:Double")
        back = self.rollback(out["batch_id"])
        self.assertEqual(back["verdict"], "verified")
        self.assertEqual(self.fake.by_handle[h].child("fallback").child("value").value, "0.0")

    def test_rollback_of_a_remove_recreates_type_slots_annotation_and_links(self):
        nn, gh = self.group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        grp = self.fake.folder.child(nn)
        self.assertEqual(grp.type, "baja:Folder")
        self.assertEqual(grp.child("wsAnnotation").value, "1,2,3")
        self.assertEqual(grp.child("Src").child("out").child("value").value, "4.5")
        link = grp.child("Tgt").child("Link")
        self.assertEqual(link.child("targetSlotName").value, "in10")
        self.assertEqual(back["relinks"], {"restored": 1, "skipped": 0})
        view = self.journal().read(back["batch_id"])
        self.assertEqual(view["inverse"], [{"nm": "v", "h": "3", "n": nn}])

    def test_relinks_are_skipped_when_an_end_is_missing(self):
        nn, gh = self.group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        orig = self.fake._sync

        def drop_tgt(op):
            res = orig(op)
            if op["nm"] == "a":
                node = self.fake.folder.child(nn)
                node.children = [c for c in node.children if c.name != "Tgt"]
            return res
        self.fake._sync = drop_tgt
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["relinks"], {"restored": 0, "skipped": 1})

    def test_unknown_batch_is_refused(self):
        self.assertIn("unknown batch", self.err("n4_rollback", batch_id="f" * 32))

    def test_a_batch_cannot_be_rolled_back_twice_nor_a_rollback_rolled_back(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        back = self.rollback(out["batch_id"])
        self.assertIn("already rolled back", self.err("n4_rollback", batch_id=out["batch_id"]))
        self.assertIn("is itself a rollback", self.err("n4_rollback", batch_id=back["batch_id"]))

    def test_in_doubt_batch_is_refused_and_shows_the_intent(self):
        plan = self.dry("n4_create_component", parent_ord=FOLDER, name="Pump",
                        type="kitControl:NumericConst")
        self.fake.hook = lambda frame: (500, b"boom", {}) if any(
            m.get("b", {}).get("sck") == "syncTo" for m in frame["m"]) else None
        self.err("n4_create_component", parent_ord=FOLDER, name="Pump",
                 type="kitControl:NumericConst", dry_run=False,
                 confirmation_token=plan["confirmation_token"])
        self.fake.hook = None
        batch = self.lines("journal.jsonl")[0]["batch_id"]
        text = self.err("n4_rollback", batch_id=batch)
        self.assertIn("in-doubt", text)
        self.assertIn('"n4_create_component"', text)  # the intent is shown to the human

    def test_nothing_to_roll_back_is_refused(self):
        nn, _ = self.add("Sp", "control:NumericWritable")
        out = self.run_write("n4_invoke_action", ord=FOLDER + "/" + nn, action="active")
        self.assertIn("no inverse", self.err("n4_rollback", batch_id=out["batch_id"]))

    def test_a_batch_from_another_station_is_refused(self):
        self.journal().append({"batch_id": "ab" * 16, "phase": "intent", "tool": "n4_set_slot",
                               "station_name": "OtherStation", "ops": [], "inverse_plan": [],
                               "targets": {FOLDER: "3"}})
        self.journal().append({"batch_id": "ab" * 16, "phase": "result", "verdict": "verified",
                               "inverse": [{"nm": "v", "h": "3", "n": "x"}], "accepted": None})
        self.assertIn("OtherStation", self.err("n4_rollback", batch_id="ab" * 16))

    def test_current_scope_is_enforced_on_every_recorded_target(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        self.srv.ctx.write.scope = safety.WriteScope(["station:|slot:/Elsewhere"])
        self.assertIn("outside the write scope", self.err("n4_rollback", batch_id=out["batch_id"]))
        self.assertEqual(self.children(), ["Pump"])

    def test_a_target_whose_handle_changed_is_refused(self):
        nn, h = self.add("Calc", out=1.0)
        out = self.run_write("n4_set_slot", ord=FOLDER + "/" + nn, slot="out", value=9.0,
                             value_type="baja:StatusNumeric")
        self.box.remove_component("3", nn)
        self.add(nn, out=2.0)  # same ORD, new component
        text = self.err("n4_rollback", batch_id=out["batch_id"])
        self.assertIn("handle", text)

    def test_a_failed_rollback_may_be_retried(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        self.journal().append({"batch_id": "cd" * 16, "phase": "intent",
                               "rollback_of": out["batch_id"]})
        self.journal().append({"batch_id": "cd" * 16, "phase": "result", "verdict": "failed",
                               "inverse": []})
        self.assertEqual(self.rollback(out["batch_id"])["verdict"], "verified")


class TestSaveStation(DestructiveCase):
    def home(self, content=b"bog"):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        path = os.path.join(tmp.name, "config.bog")
        with open(path, "wb") as fh:
            fh.write(content)
        self.fake.config_path = path
        self.srv.ctx.write.station_homes = {"FakeStation": tmp.name}
        self.srv.ctx.write.save_timeout, self.srv.ctx.write.save_interval = 1.0, 0.01
        return path

    def test_plan_is_a_save_on_the_root_handle_with_no_inverse(self):
        plan = self.dry("n4_save_station")
        self.assertEqual(plan["plan"]["ops"], [{"ssc": "invokeAction",
                                                "arg": {"h": "2", "a": "save"}}])
        self.assertEqual(plan["plan"]["inverse"], [])
        self.assertEqual(self.fake.saves, 0)

    def test_without_a_station_home_persistence_is_unknown_and_says_how_to_configure(self):
        out = self.run_write("n4_save_station")
        self.assertEqual((out["persisted"], out["verdict"]), ("unknown", "unverified"))
        self.assertIn("--station-home FakeStation=", out["evidence"]["hint"])
        self.assertEqual(self.fake.saves, 1)

    def test_a_changed_config_bog_proves_persistence_with_evidence(self):
        path = self.home(b"bog-before")
        before = hashlib.sha256(b"bog-before").hexdigest()
        out = self.run_write("n4_save_station")
        self.assertEqual((out["persisted"], out["verdict"]), (True, "verified"))
        ev = out["evidence"]
        self.assertEqual(ev["before"]["sha256"], before)
        self.assertNotEqual(ev["after"]["sha256"], before)
        self.assertGreater(ev["after"]["mtime"], ev["before"]["mtime"])
        self.assertEqual(ev["path"], path)

    def test_an_unchanged_config_bog_is_persisted_false_after_the_poll(self):
        self.home()
        self.fake.save_rewrites = False
        out = self.run_write("n4_save_station")
        self.assertEqual((out["persisted"], out["verdict"]), (False, "mismatch"))
        self.assertEqual(out["evidence"]["before"], out["evidence"]["after"])

    def test_a_missing_config_bog_is_unknown(self):
        path = self.home()
        os.remove(path)
        out = self.run_write("n4_save_station")
        self.assertEqual(out["persisted"], "unknown")
        self.assertIn("config.bog", out["evidence"]["hint"])

    def test_the_reply_to_save_is_never_taken_as_proof(self):
        self.home()
        self.fake.save_rewrites = False
        out = self.run_write("n4_save_station")
        self.assertIsNone(out["accepted"])
        self.assertIsNot(out["persisted"], True)

    def test_a_write_scope_is_still_required(self):
        self.start_server(write_scopes=[])
        self.connect_verified()
        self.assertIn("no --write-scope", self.err("n4_save_station"))
        self.assertEqual(self.fake.saves, 0)


class TestStationHomeFlag(unittest.TestCase):
    def test_flag_is_repeatable_and_defaults_empty(self):
        self.assertEqual(server.parse_args([]).station_home, [])
        args = server.parse_args(["--station-home", "A=/x", "--station-home", "B=/y"])
        self.assertEqual(args.station_home, ["A=/x", "B=/y"])

    def test_malformed_values_exit_2_with_a_clear_message(self):
        for bad in ("nopath", "=/x", "A="):
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                code = server.main(["--station-home", bad])
            self.assertEqual(code, 2, bad)
            self.assertIn("--station-home expects NAME=PATH", err.getvalue(), bad)

    def test_server_passes_the_homes_to_the_write_state(self):
        with tempfile.TemporaryDirectory() as tmp:
            srv = server.Server(allow_writes=True, state_dir=os.path.join(tmp, "s"),
                                station_homes={"A": "/x"})
            self.assertEqual(srv.ctx.write.station_homes, {"A": "/x"})
            srv.ctx.close()


if __name__ == "__main__":
    unittest.main()
