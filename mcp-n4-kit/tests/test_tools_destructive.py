import contextlib
import hashlib
import io
import json
import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import fake_station  # noqa: E402
import test_tools_write as ttw  # noqa: E402  (module import: its tests are not re-collected)
from mcp_n4 import box, safety, server, tools_write  # noqa: E402

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
        self.assertEqual([c["n"] for c in slots["Src"].get("s", [])], [])  # runtime out: dropped
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
        before = self.children()
        text = self.err("n4_remove_component", parent_ord=FOLDER, name="../x")
        self.assertIn("name must be a plain slot name", text)
        self.assertIn("'../x'", text)
        self.assertEqual(self.children(), before)  # nothing was removed
        self.assertEqual(self.lines("journal.jsonl"), [])  # and nothing was journaled
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


class TestOutgoingLinks(DestructiveCase):
    NOTE = "cannot be detected"

    def outside(self, nn, name="Other"):
        other, other_h = self.add(name, out=1.0)
        src_h = self.box.load_tree(FOLDER + "/" + nn, depth=2, **NO_SLEEP)["Src"]["h"]
        self.box.check_links(src_h, "out", other_h, "in10")
        return other

    def test_a_subtree_with_outputs_discloses_that_outgoing_links_cannot_be_seen(self):
        nn, _ = self.group()
        notes = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)["plan"]["notes"]
        note = [n for n in notes if self.NOTE in n]
        self.assertEqual(len(note), 1)
        self.assertIn("outside the removed subtree", note[0])
        self.assertIn("link_scan_ord", note[0])

    def test_a_subtree_without_outputs_has_no_such_note(self):
        nn, _ = self.add("Plain", "baja:Folder")
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)["plan"]
        self.assertFalse([n for n in plan["notes"] if self.NOTE in n])
        self.assertNotIn("outgoing_links_broken", plan)

    def test_a_scan_root_lists_the_outgoing_links_that_the_remove_would_break(self):
        nn, _ = self.group()
        other = self.outside(nn)
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn,
                        link_scan_ord=FOLDER)["plan"]
        self.assertEqual(plan["outgoing_links_broken"], [{
            "target": FOLDER + "/" + other, "target_slot": "in10",
            "source_path": "Src", "source_slot": "out"}])
        self.assertFalse([n for n in plan["notes"] if self.NOTE in n])
        self.assertTrue(any("scanned" in n and FOLDER in n for n in plan["notes"]))

    def test_links_inside_the_subtree_are_not_reported_as_outgoing(self):
        nn, _ = self.group()
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn,
                        link_scan_ord=FOLDER)["plan"]
        self.assertNotIn("outgoing_links_broken", plan)
        note = [n for n in plan["notes"] if "no outgoing links" in n][0]
        self.assertIn("depth %d" % tools_write.LINK_SCAN_DEPTH, note)
        self.assertIn("deeper", note)  # an empty scan must say what it could not see

    def test_the_scan_root_must_be_inside_the_write_scope(self):
        nn, _ = self.group()
        text = self.err("n4_remove_component", parent_ord=FOLDER, name=nn,
                        link_scan_ord="station:|slot:/")
        self.assertIn("outside the write scope", text)

    def test_the_scan_result_is_covered_by_the_plan_hash(self):
        nn, _ = self.group()
        plain = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.outside(nn)
        scanned = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn,
                           link_scan_ord=FOLDER)
        self.assertNotEqual(plain["plan_hash"], scanned["plan_hash"])


class TestPlannedDataGuards(DestructiveCase):
    def test_a_plan_whose_data_is_not_a_dict_still_executes(self):
        from mcp_n4 import tools_write
        impl = tools_write._IMPLS["n4_create_component"]
        orig = impl.plan

        def plan(client, args):
            p = orig(client, args)
            return p._replace(data=None)
        patched = impl._replace(
            plan=plan, readback=lambda c, a, p, r, i: ({}, r[0], {}, "verified"),
            inverse=lambda p, r: [])
        with mock_impl("n4_create_component", patched):
            out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                                 type="kitControl:NumericConst")
        self.assertEqual(out["verdict"], "verified")


class mock_impl:
    def __init__(self, name, impl):
        from mcp_n4 import tools_write
        self.table, self.name, self.impl = tools_write._IMPLS, name, impl

    def __enter__(self):
        self.old = self.table[self.name]
        self.table[self.name] = self.impl

    def __exit__(self, *exc):
        self.table[self.name] = self.old


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
        self.assertIsNone(grp.child("Src").child("out"))  # runtime output: not restored
        link = grp.child("Tgt").child("Link")
        self.assertEqual(link.child("targetSlotName").value, "in10")
        self.assertEqual(back["relinks"], {"restored": 1, "skipped": 0})
        view = self.journal().read(back["batch_id"])
        tgt_h = grp.child("Tgt").handle
        self.assertEqual(view["inverse"], [{"nm": "v", "h": tgt_h, "n": "Link"},
                                           {"nm": "v", "h": "3", "n": nn}])

    def test_the_snapshot_keeps_configuration_and_drops_runtime_output_slots(self):
        nn, gh = self.group()
        src_h = self.box.load_tree(FOLDER + "/%s/Src" % nn, depth=1, **NO_SLEEP)[""]["h"]
        self.box.set_slot(src_h, "fallback", box.bson_status_numeric(7.0, "0;activeLevel=e_def"))
        self.box.set_slot(src_h, "out", box.bson_status_numeric(4.5, "0;activeLevel=e_in16"))
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        self.rollback(removed["batch_id"])
        src = [op for op in sent if op["nm"] == "a" and op["n"] == "Src"][0]
        names = [c["n"] for c in src["b"]["s"]]
        self.assertNotIn("out", names)
        self.assertIn("fallback", names)
        fallback = self.fake.folder.child(nn).child("Src").child("fallback")
        self.assertEqual(fallback.child("value").value, "7.0")
        self.assertEqual(fallback.child("status").value, "0")  # no runtime activeLevel facet
        self.assertEqual(tools_write.RUNTIME_OUTPUT_SLOTS, ("out",))

    def test_the_snapshot_keeps_configured_status_bits_and_drops_only_facets(self):
        """Review R4-status-coercion: a null (0x40) fallback must not come back as ok."""
        nn, gh = self.group()
        src_h = self.box.load_tree(FOLDER + "/%s/Src" % nn, depth=1, **NO_SLEEP)[""]["h"]
        self.box.set_slot(src_h, "fallback", box.bson_status_numeric(7.0, "40;activeLevel=e_def"))
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.rollback(removed["batch_id"])
        fallback = self.fake.folder.child(nn).child("Src").child("fallback")
        self.assertEqual(fallback.child("status").value, "40")  # null bit kept, facet dropped

    # ---- live finding 3: null types and link-driven inputs -------------------
    def live_input(self, comp, name, value, status):
        """A Status input as the live station loads it: frozen children carry no type."""
        slot = fake_station._Node(name, "baja:StatusNumeric")
        slot.children = [fake_station._Node("value", None, value),
                         fake_station._Node("status", None, status)]
        comp.children.append(slot)

    def linked_group(self):
        nn, gh = self.group()
        tgt = self.fake.folder.child(nn).child("Tgt")
        self.live_input(tgt, "in10", "4.5", "0;activeLevel=e:17@control:PriorityLevel")
        self.live_input(tgt, "inB", "25.0", "0;activeLevel=e:17@control:PriorityLevel")
        return nn

    def test_the_station_model_rejects_null_types_and_status_facets(self):
        for body in ({"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "t": None, "n": "v"}]},
                     {"nm": "p", "t": "baja:Folder", "s": [
                         {"nm": "p", "t": "baja:StatusNumeric", "n": "x", "s": [
                             {"nm": "p", "n": "status", "v": "0;activeLevel=e:17"}]}]}):
            with self.assertRaises(box.BoxError) as caught:
                self.box.sync({"nm": "a", "h": "3", "n": "Bad", "b": body})
            self.assertIn("Unable to process request", str(caught.exception))

    def test_a_linked_group_rolls_back_without_null_types_or_link_driven_inputs(self):
        nn = self.linked_group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        self.assertNotIn('"t": null', safety.canonical([op["b"] for op in sent]).replace(
            '"t":null', '"t": null'))
        tgt = [op for op in sent if op["nm"] == "a" and op["n"] == "Tgt"][0]
        names = [c["n"] for c in tgt["b"].get("s", [])]
        self.assertNotIn("in10", names)  # link target: the relink re-establishes it
        self.assertIn("inB", names)      # plain configured input is kept
        inb = [c for c in tgt["b"]["s"] if c["n"] == "inB"][0]
        status = [c for c in inb["s"] if c["n"] == "status"][0]
        self.assertEqual(status["v"], "0")  # bits kept, facet dropped
        self.assertTrue(all("t" in c or c["n"] in ("value", "status") for c in inb["s"]))
        value = [c for c in inb["s"] if c["n"] == "value"][0]
        self.assertNotIn("t", value)
        self.assertEqual(back["relinks"], {"restored": 1, "skipped": 0})
        self.assertEqual(self.fake.folder.child(nn).child("Tgt").child("Link")
                         .child("targetSlotName").value, "in10")

    def test_snapshot_omits_t_for_typeless_nodes_and_recognizes_status_by_name(self):
        node = {"t": "baja:StatusNumeric", "h": "9", "s": [
            {"n": "value", "v": "1.0"}, {"n": "status", "v": "40;activeLevel=e:17"}]}
        snap = tools_write._snapshot(node, "x", [], {})
        self.assertEqual(snap["s"], [{"nm": "p", "n": "value", "v": "1.0"},
                                     {"nm": "p", "n": "status", "v": "40"}])

    def test_a_folder_removed_at_the_station_root_rolls_back(self):
        """Live finding 2 (2026-10-01): `station:|slot:/` + `/` made `station:|slot://X`."""
        self.start_server(write_scopes=["station:|slot:/"])
        self.connect_verified()
        top = self.box.add_component("2", "RootGrp", "baja:Folder")["nn"]
        top_h = self.box.load_tree("station:|slot:/" + top, depth=1,
                                   **NO_SLEEP)[""]["h"]
        self.box.add_component(top_h, "Src", "kitControl:NumericConst")
        removed = self.run_write("n4_remove_component", parent_ord="station:|slot:/", name=top)
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        self.assertIsNotNone(self.fake.root.child(top).child("Src"))

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

    # ---- relinks are first-class planned ops (R3-relink-unjournaled-writes) ----

    def removed_group(self):
        nn, gh = self.group()
        return nn, self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)

    def test_relinks_are_journaled_with_intent_before_the_send_and_a_result(self):
        nn, removed = self.removed_group()
        back = self.rollback(removed["batch_id"])
        phases = [(e["phase"], e["batch_id"]) for e in self.lines("journal.jsonl")
                  if e["batch_id"] == back["batch_id"]]
        self.assertEqual([p for p, _ in phases], ["intent", "component-intent", "component-intent",
                                                  "relink-intent", "result"])
        relink_intent = [e for e in self.lines("journal.jsonl")
                         if e["phase"] == "relink-intent"][0]
        self.assertEqual(len(relink_intent["ops"]), 1)
        self.assertEqual(relink_intent["ops"][0]["ssc"], "checkLinks")
        self.assertEqual(relink_intent["ops"][0]["arg"]["ss"], "out")
        view = self.journal().read(back["batch_id"])
        self.assertEqual(view["relink_ops"], relink_intent["ops"])
        self.assertEqual(view["relinks"], {"restored": 1, "skipped": 0})

    def test_the_relink_intent_is_on_disk_before_the_relink_is_sent(self):
        nn, removed = self.removed_group()
        seen = []
        orig = self.fake._check_link

        def spy(arg):
            seen.append([e["phase"] for e in self.lines("journal.jsonl")])
            return orig(arg)
        self.fake._check_link = spy
        self.rollback(removed["batch_id"])
        self.assertIn("relink-intent", seen[-1])

    def test_relinks_consume_the_write_budget(self):
        nn, removed = self.removed_group()
        before = self.srv.ctx.session.writes_executed
        self.rollback(removed["batch_id"])
        # 1 batch + 2 child components (Src, Tgt) + 1 relink
        self.assertEqual(self.srv.ctx.session.writes_executed, before + 1 + 2 + 1)

    def test_a_rollback_whose_relinks_exceed_the_budget_is_refused_before_sending(self):
        self.start_server(max_writes=2)
        self.connect_verified()
        nn, removed = self.removed_group()
        text = self.err("n4_rollback", batch_id=removed["batch_id"])
        self.assertIn("--max-writes", text)
        self.assertIn("relink", text)
        self.assertNotIn(nn, self.children())  # nothing was re-created

    def test_a_relink_end_outside_the_write_scope_is_refused_and_nothing_is_sent(self):
        nn, removed = self.removed_group()
        real = self.srv.ctx.write.scope

        class Spy:
            def check(self, ord_str):
                if ord_str.endswith("/Tgt"):
                    raise safety.SafetyError("ord %r is outside the write scope" % ord_str)
                real.check(ord_str)

            def require_any(self):
                real.require_any()
        self.srv.ctx.write.scope = Spy()
        text = self.err("n4_rollback", batch_id=removed["batch_id"])
        self.assertIn("outside the write scope", text)
        self.assertNotIn(nn, self.children())

    def test_relink_specs_are_hashed_by_path_not_by_handle(self):
        nn, removed = self.removed_group()
        plan = self.dry("n4_rollback", batch_id=removed["batch_id"])
        self.assertEqual(plan["plan"]["relinks"], [{
            "source_path": "Src", "source_slot": "out", "target_path": "Tgt",
            "target_slot": "in10"}])

    def test_the_rollback_journals_an_inverse_that_removes_the_links_it_created(self):
        nn, removed = self.removed_group()
        back = self.rollback(removed["batch_id"])
        for op in self.journal().read(back["batch_id"])["inverse"]:
            self.box.sync(op)  # what undoing the rollback means, link first
        self.assertNotIn(nn, self.children())

    def test_an_ambiguous_relink_reply_marks_the_batch_in_doubt(self):
        nn, removed = self.removed_group()
        self.fake._check_link = lambda arg: [{"r": "huh"}]  # no verdict, no link name
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["relinks"], {"restored": 0, "skipped": 1, "ambiguous": 1})
        self.assertEqual(self.journal().read(back["batch_id"])["state"], "in-doubt")

    # ---- one add per component (the live station rejects nested adds) ------------

    def sent_adds(self):
        sent = []
        orig = self.fake._sync
        self.fake._sync = lambda op: (sent.append(op), orig(op))[1]
        return sent

    def test_the_station_model_rejects_an_add_that_nests_components(self):
        """Regression for the live failure of 2026-10-01 (N4.14 station LLM)."""
        nested = {"nm": "p", "t": "baja:Folder", "s": [
            {"nm": "p", "n": "Src", "t": "kitControl:NumericConst"}]}
        with self.assertRaises(box.BoxError) as caught:
            self.box.sync({"nm": "a", "h": "3", "n": "Grp", "b": nested})
        self.assertIn("Unable to process request", str(caught.exception))
        plain = {"nm": "p", "t": "baja:Folder", "s": [box.ws_annotation(1, 2, 3)]}
        self.box.sync({"nm": "a", "h": "3", "n": "Ok", "b": plain})  # plain slots are fine

    def test_the_station_model_rejects_double_slashes_like_the_live_station(self):
        with self.assertRaises(box.BoxError) as caught:
            self.box.load_tree("station:|slot://Folder", depth=1, **NO_SLEEP)
        self.assertIn("Illegal double slashes", str(caught.exception))

    def test_rollback_adds_one_component_per_op_parents_first_with_the_new_handle(self):
        nn, gh = self.group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        self.rollback(removed["batch_id"])
        adds = [op for op in sent if op["nm"] == "a"]
        self.assertEqual([op["n"] for op in adds], [nn, "Src", "Tgt"])
        for op in adds:
            self.assertEqual([c for c in op["b"].get("s", []) if c["t"].partition(":")[0]
                              != "baja"], [])  # only plain slots and wsAnnotation
        grp = self.fake.folder.child(nn)
        self.assertEqual(adds[0]["h"], "3")
        self.assertEqual([op["h"] for op in adds[1:]], [grp.handle, grp.handle])
        self.assertEqual(adds[0]["b"]["s"][0]["n"], "wsAnnotation")

    def writable_group(self):
        """A group holding a NumericWritable and a BooleanWritable (frozen proxyExt each)."""
        nn, gh = self.group()
        self.box.add_component(gh, "Sp", "control:NumericWritable")
        self.box.add_component(gh, "En", "control:BooleanWritable")
        return nn, gh

    def test_the_station_model_gives_writables_a_frozen_proxy_ext_and_rejects_another(self):
        nn, gh = self.writable_group()
        sp = self.fake.folder.child(nn).child("Sp")
        self.assertEqual(sp.child("proxyExt").type, "control:NullProxyExt")
        with self.assertRaises(box.BoxError) as caught:
            self.box.add_component(sp.handle, "proxyExt", "control:NullProxyExt")
        self.assertIn('Illegal child "control:NullProxyExt" for parent '
                      '"control:NumericWritable".', str(caught.exception))

    def test_a_group_with_writables_rolls_back_without_re_adding_frozen_children(self):
        nn, gh = self.writable_group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        adds = [op for op in sent if op["nm"] == "a"]
        self.assertEqual([op["n"] for op in adds], [nn, "Src", "Tgt", "Sp", "En"])
        group = self.fake.folder.child(nn)
        for name in ("Sp", "En"):
            self.assertEqual([c.type for c in group.child(name).children
                              if c.name == "proxyExt"], ["control:NullProxyExt"])
        frozen = back["frozen_children_not_restored"]
        self.assertEqual(sorted((f["n"], f["type"]) for f in frozen),
                         [("proxyExt", "control:NullProxyExt")] * 2)
        self.assertEqual(sorted(f["path"].rsplit("/", 2)[-2] for f in frozen), ["En", "Sp"])

    def test_descendants_of_a_skipped_frozen_child_are_skipped_too(self):
        nn, gh = self.writable_group()
        sp = self.fake.folder.child(nn).child("Sp")
        self.box.add_component(sp.child("proxyExt").handle, "Inner", "kitControl:NumericConst")
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        back = self.rollback(removed["batch_id"])
        # Inner is gone after the rollback: never "verified" (issue #179 R4-002)
        self.assertEqual(back["verdict"], "partial", back)
        self.assertNotIn("Inner", [op["n"] for op in sent if op["nm"] == "a"])
        self.assertIn(("Inner", "kitControl:NumericConst"),
                      [(f["n"], f["type"]) for f in back["frozen_children_not_restored"]])
        lost = {f["path"].rsplit("/", 1)[-1]: f for f in back["frozen_config_not_restored"]}
        self.assertEqual(sorted(lost), ["Inner"])
        self.assertTrue(lost["Inner"]["missing"])

    def test_a_rollback_without_frozen_children_reports_none(self):
        nn, removed = self.removed_group()
        self.assertNotIn("frozen_children_not_restored", self.rollback(removed["batch_id"]))

    # ---- frozen-child configuration (issue #179 R4-002 / B1200-G1) -------------------

    def configure_proxy_ext(self, nn, value="Fast"):
        """Give Sp's frozen proxyExt a configured (non-default) slot, as a real point has."""
        ext = self.fake.folder.child(nn).child("Sp").child("proxyExt")
        self.box.sync({"nm": "s", "h": ext.handle, "n": "tuningPolicyName",
                       "b": {"nm": "p", "t": "baja:String", "v": value}})
        return ext

    def test_frozen_child_config_not_restored_downgrades_the_verdict_to_partial(self):
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "partial", back)
        lost = back["frozen_config_not_restored"]
        self.assertEqual([f["path"].rsplit("/", 2)[-2:] for f in lost], [["Sp", "proxyExt"]])
        self.assertEqual(lost[0]["slots"], ["tuningPolicyName"])
        self.assertEqual(lost[0]["captured"]["s"],
                         [{"nm": "p", "t": "baja:String", "n": "tuningPolicyName", "v": "Fast"}])
        self.assertFalse(lost[0]["missing"])
        self.assertEqual(self.journal().read(back["batch_id"])["verdict"], "partial")

    def test_a_frozen_child_of_another_type_is_reported_as_not_restored(self):
        nn, gh = self.writable_group()
        self.fake.folder.child(nn).child("Sp").child("proxyExt").type = \
            "bacnet:BacnetNumericProxyExt"
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "partial", back)
        (lost,) = back["frozen_config_not_restored"]
        self.assertEqual((lost["type"], lost["live_type"]),
                         ("bacnet:BacnetNumericProxyExt", "control:NullProxyExt"))

    def test_a_frozen_child_whose_config_matches_keeps_the_verdict_verified(self):
        """The fresh frozen child already holds the captured config: nothing was lost."""
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        orig = self.fake._sync

        def default_fast(op):
            res = orig(op)
            if op["nm"] == "a" and op["n"] == "Sp":  # model only: no BOX call in a hook
                ext = self.fake.folder.child(nn).child("Sp").child("proxyExt")
                ext.children.append(fake_station._Node("tuningPolicyName", "baja:String",
                                                       "Fast"))
            return res
        self.fake._sync = default_fast
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        self.assertNotIn("frozen_config_not_restored", back)

    def test_the_frozen_check_treats_an_omitted_default_as_equal(self):
        """The station omits slots at their default; a captured `0` status is not a loss."""
        captured = {"nm": "p", "t": "control:NullProxyExt", "s": [
            {"nm": "p", "n": "status", "t": "baja:Status", "v": "0"},
            {"nm": "p", "n": "readValue", "t": "baja:StatusNumeric", "s": [
                {"nm": "p", "n": "value", "v": "0.0"}, {"nm": "p", "n": "status", "v": "0"}]},
            {"nm": "p", "n": "tuningPolicyName", "t": "baja:String", "v": "Fast"}]}

        class Client:
            def __init__(self, nodes):
                self.nodes = nodes

            def load_tree(self, ord_str, depth):
                return self.nodes
        check = {"path": "station:|slot:/F/Sp/proxyExt", "b": captured}
        live = {"": {"t": "control:NumericWritable"},
                "proxyExt": {"t": "control:NullProxyExt"},
                "proxyExt/readValue": {"t": "baja:StatusNumeric"},
                "proxyExt/tuningPolicyName": {"t": "baja:String", "v": "Fast"}}
        self.assertIsNone(tools_write._frozen_check(Client(live), check))
        live["proxyExt/tuningPolicyName"]["v"] = "Slow"
        gap = tools_write._frozen_check(Client(live), check)
        self.assertEqual((gap["slots"], gap["missing"]), (["tuningPolicyName"], False))

    def test_the_frozen_check_treats_an_omitted_nested_default_slot_as_equal(self):
        """A Status slot the station omits whole (value and status at default) is not a
        loss (v0.28.0 advisory review R4-002); one with a non-default nested value still is."""
        captured = {"nm": "p", "t": "control:NullProxyExt", "s": [
            {"nm": "p", "n": "readValue", "t": "baja:StatusNumeric", "s": [
                {"nm": "p", "n": "value", "v": "0.0"}, {"nm": "p", "n": "status", "v": "0"}]}]}

        class Client:
            def load_tree(self, ord_str, depth):
                return {"": {"t": "control:NumericWritable"},
                        "proxyExt": {"t": "control:NullProxyExt"}}
        check = {"path": "station:|slot:/F/Sp/proxyExt", "b": captured}
        self.assertIsNone(tools_write._frozen_check(Client(), check))
        captured["s"][0]["s"][0]["v"] = "21.5"
        gap = tools_write._frozen_check(Client(), check)
        self.assertEqual((gap["slots"], gap["missing"]), (["readValue"], False))

    def test_a_frozen_check_read_error_is_reported_distinctly_not_as_missing(self):
        """A station read error is not evidence the frozen child is gone (v0.28.0 advisory review
        R4-001/R2-001/R3-001)."""
        class Client:
            def load_tree(self, ord_str, depth):
                raise box.BoxError("HTTP 503 from station", box.CHANNEL, "loadSlots")
        check = {"path": "station:|slot:/F/Sp/proxyExt",
                 "b": {"nm": "p", "t": "control:NullProxyExt"}}
        gap = tools_write._frozen_check(Client(), check)
        self.assertNotIn("missing", gap)
        self.assertEqual(gap["readback_error"], "HTTP 503 from station")
        self.assertEqual(gap["path"], check["path"])

    def test_a_frozen_check_read_error_makes_the_rollback_unverified(self):
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        client = self.srv.ctx.session.client
        orig = client.load_tree

        def failing(ord_str, depth=2, **kw):  # only the frozen-child read-back fails
            if depth == 7:
                raise box.BoxError("HTTP 503 from station", box.CHANNEL, "loadSlots")
            return orig(ord_str, depth=depth, **kw)
        client.load_tree = failing
        with mock.patch.object(tools_write, "FROZEN_CHECK_DEPTH", 7):
            back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "unverified", back)
        gaps = back["frozen_config_not_restored"]
        self.assertTrue(gaps and all("missing" not in g for g in gaps), gaps)
        self.assertTrue(all(g["readback_error"] == "HTTP 503 from station" for g in gaps))
        self.assertEqual(self.journal().read(back["batch_id"])["verdict"], "unverified")

    def fail_frozen_reads(self, message, only=None):
        """Make the frozen-child read-back (depth 7) raise; `only` limits it to ORDs ending so."""
        client = self.srv.ctx.session.client
        orig = client.load_tree

        def failing(ord_str, depth=2, **kw):
            if depth == 7 and (only is None or ord_str.endswith(only)):
                raise box.BoxError(message, box.CHANNEL, "loadSlots")
            return orig(ord_str, depth=depth, **kw)
        client.load_tree = failing
        return mock.patch.object(tools_write, "FROZEN_CHECK_DEPTH", 7)

    def test_a_frozen_check_read_error_is_scrubbed_of_the_session_secret(self):
        """The gap's readback_error goes through ctx.scrub (issue #190 R3-001)."""
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        message = "HTTP 401 from station: credentials %s rejected" % self.PASSWORD
        with self.fail_frozen_reads(message):
            back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "unverified", back)
        gaps = back["frozen_config_not_restored"]
        self.assertTrue(gaps, back)
        for gap in gaps:
            self.assertEqual(gap["readback_error"],
                             "HTTP 401 from station: credentials *** rejected")
        self.assertNotIn(self.PASSWORD, json.dumps(back))

    def test_a_read_error_beside_a_real_frozen_gap_makes_the_rollback_unverified(self):
        """Precedence: an unread frozen child outranks a real config gap (issue #190 R3-002)."""
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)  # Sp's proxyExt: a real, readable gap
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        with self.fail_frozen_reads("HTTP 503 from station", only="/En"):
            back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "unverified", back)
        gaps = {g["path"].rsplit("/", 2)[-2]: g for g in back["frozen_config_not_restored"]}
        self.assertEqual(sorted(gaps), ["En", "Sp"])
        self.assertEqual(gaps["En"]["readback_error"], "HTTP 503 from station")
        self.assertNotIn("missing", gaps["En"])
        self.assertEqual((gaps["Sp"]["slots"], gaps["Sp"]["missing"]),
                         (["tuningPolicyName"], False))
        self.assertEqual(self.journal().read(back["batch_id"])["verdict"], "unverified")

    def test_a_skipped_relink_beside_a_read_error_makes_the_rollback_a_mismatch(self):
        """Precedence: a skipped relink (mismatch) outranks an unread frozen child (#190 R3-002)."""
        nn, gh = self.writable_group()
        self.configure_proxy_ext(nn)
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        orig = self.fake._sync

        def drop_src(op):
            res = orig(op)
            if op["nm"] == "a" and op["n"] == "Src":
                node = self.fake.folder.child(nn)
                node.children = [c for c in node.children if c.name != "Src"]
            return res
        self.fake._sync = drop_src
        with self.fail_frozen_reads("HTTP 503 from station"):
            back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "mismatch", back)
        self.assertEqual(back["relinks"], {"restored": 0, "skipped": 1})
        gaps = back["frozen_config_not_restored"]
        self.assertTrue(gaps and all(g["readback_error"] == "HTTP 503 from station"
                                     for g in gaps), gaps)
        self.assertEqual(self.journal().read(back["batch_id"])["verdict"], "mismatch")

    # ---- link-targeted inputs (issue #179 R3-002 / B1200-G2) -------------------------

    def test_the_remove_plan_records_the_value_of_a_link_driven_input(self):
        nn = self.linked_group()
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)
        (relink,) = [e["relink"] for e in plan["plan"]["inverse"] if "relink" in e]
        self.assertEqual(relink["target_slot"], "in10")
        # volatile: never part of the hashed plan (PR #187 blocking review R3-001)
        self.assertNotIn("prior", relink)
        self.assertTrue(any("captured when the confirmed remove runs" in n
                            for n in plan["plan"]["notes"]), plan["plan"]["notes"])
        (seen,) = plan["link_input_values"]  # the operator's preview, outside the hash
        self.assertEqual((seen["target_path"], seen["target_slot"]), ("Tgt", "in10"))
        self.assertEqual(seen["value"]["t"], "baja:StatusNumeric")
        self.assertEqual({c["n"]: c["v"] for c in seen["value"]["s"]},
                         {"value": "4.5", "status": "0"})  # facets dropped, bits kept

    def test_a_link_driven_value_that_changes_after_the_dry_run_keeps_the_token_valid(self):
        """The link source keeps propagating between dry run and confirm.

        PR #187 blocking review R3-001."""
        nn = self.linked_group()
        plan = self.dry("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.fake.folder.child(nn).child("Tgt").child("in10").child("value").value = "7.25"
        removed = self.ok("n4_remove_component", parent_ord=FOLDER, name=nn, dry_run=False,
                          confirmation_token=plan["confirmation_token"])
        self.assertNotIn(nn, self.children())
        (relink,) = [e["relink"] for e in self.journal().read(removed["batch_id"])["inverse"]
                     if "relink" in e]
        self.assertEqual({c["n"]: c["v"] for c in relink["prior"]["s"]},
                         {"value": "7.25", "status": "0"})  # the value seen at execution

    def test_a_skipped_relink_reports_the_captured_input_value(self):
        nn = self.linked_group()
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        orig = self.fake._sync

        def drop_src(op):
            res = orig(op)
            if op["nm"] == "a" and op["n"] == "Src":
                node = self.fake.folder.child(nn)
                node.children = [c for c in node.children if c.name != "Src"]
            return res
        self.fake._sync = drop_src
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "mismatch", back)
        (lost,) = back["link_inputs_not_restored"]
        self.assertTrue(lost["target"].endswith("/%s/Tgt" % nn), lost)
        self.assertEqual((lost["target_slot"], lost["source_slot"], lost["reason"]),
                         ("in10", "out", "relink skipped"))
        self.assertEqual({c["n"]: c["v"] for c in lost["value"]["s"]},
                         {"value": "4.5", "status": "0"})

    def test_an_input_linked_from_outside_the_subtree_is_reported_not_lost(self):
        nn = self.linked_group()
        other, other_h = self.add("Other", out=1.0)
        tgt = self.fake.folder.child(nn).child("Tgt")
        self.live_input(tgt, "in11", "1.0", "0;activeLevel=e:17@control:PriorityLevel")
        self.box.check_links(other_h, "out", tgt.handle, "in11")
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        (unlinked,) = [e["unlinked_input"] for e in
                       self.journal().read(removed["batch_id"])["inverse"]
                       if "unlinked_input" in e]
        self.assertEqual((unlinked["target_path"], unlinked["target_slot"]), ("Tgt", "in11"))
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "partial", back)
        (lost,) = back["link_inputs_not_restored"]
        self.assertEqual((lost["target_slot"], lost["reason"]),
                         ("in11", "link source outside the removed subtree"))
        self.assertEqual({c["n"]: c["v"] for c in lost["value"]["s"]},
                         {"value": "1.0", "status": "0"})

    def test_grandchildren_are_added_after_their_parent_breadth_first(self):
        nn, gh = self.group()
        sub = self.box.add_component(gh, "Sub", "baja:Folder")["nn"]
        sub_h = self.box.load_tree(FOLDER + "/%s/%s" % (nn, sub), depth=1, **NO_SLEEP)[""]["h"]
        self.box.add_component(sub_h, "Deep", "kitControl:NumericConst")
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        sent = self.sent_adds()
        back = self.rollback(removed["batch_id"])
        self.assertEqual(back["verdict"], "verified", back)
        self.assertEqual([op["n"] for op in sent if op["nm"] == "a"],
                         [nn, "Src", "Tgt", "Sub", "Deep"])
        self.assertIsNotNone(self.fake.folder.child(nn).child("Sub").child("Deep"))

    def test_the_plan_lists_components_by_path_and_its_hash_is_stable(self):
        nn, removed = self.removed_group()
        first = self.dry("n4_rollback", batch_id=removed["batch_id"])
        second = self.dry("n4_rollback", batch_id=removed["batch_id"])
        self.assertEqual(first["plan_hash"], second["plan_hash"])
        comps = first["plan"]["components"]
        self.assertEqual([(c["parent_path"], c["n"]) for c in comps], [("", "Src"), ("", "Tgt")])
        self.assertNotIn("h", comps[0])
        self.assertEqual(first["plan"]["ops"][0]["h"], "3")

    def test_the_component_budget_is_checked_before_anything_is_sent(self):
        self.start_server(max_writes=4)  # remove 1 + rollback needs 1 + 2 components + 1 relink
        self.connect_verified()
        nn, removed = self.removed_group()
        text = self.err("n4_rollback", batch_id=removed["batch_id"])
        self.assertIn("2 component(s)", text)
        self.assertNotIn(nn, self.children())

    def test_each_component_is_journaled_before_it_is_sent_and_counts_against_the_budget(self):
        nn, removed = self.removed_group()
        seen = []
        orig = self.fake._sync

        def spy(op):
            seen.append((op["n"], [e["phase"] for e in self.lines("journal.jsonl")]))
            return orig(op)
        self.fake._sync = spy
        before = self.srv.ctx.session.writes_executed
        back = self.rollback(removed["batch_id"])
        src_seen = dict(seen)["Src"]
        self.assertEqual(src_seen.count("component-intent"), 1)
        self.assertEqual(dict(seen)["Tgt"].count("component-intent"), 2)
        self.assertEqual(self.srv.ctx.session.writes_executed, before + 4)
        view = self.journal().read(back["batch_id"])
        self.assertEqual([op["n"] for op in view["component_ops"]], ["Src", "Tgt"])

    def test_a_failure_mid_way_is_in_doubt_and_lists_what_was_created(self):
        nn, removed = self.removed_group()
        orig = self.fake._sync

        def reject_tgt(op):
            if op["nm"] == "a" and op["n"] == "Tgt":
                raise ValueError("Unable to process request.")
            return orig(op)
        self.fake._sync = reject_tgt
        plan = self.dry("n4_rollback", batch_id=removed["batch_id"])
        text = self.err("n4_rollback", batch_id=removed["batch_id"], dry_run=False,
                        confirmation_token=plan["confirmation_token"])
        self.assertIn("in-doubt", text)
        self.assertIn("%s/%s" % (FOLDER, nn), text)
        self.assertIn("%s/%s/Src" % (FOLDER, nn), text)
        self.assertNotIn("/Tgt", text)  # the failed add is not reported as created
        batch = [e["batch_id"] for e in self.lines("journal.jsonl")
                 if e.get("rollback_of") == removed["batch_id"]][0]
        self.assertEqual(self.journal().read(batch)["state"], "in-doubt")
        self.assertIsNone(self.fake.folder.child(nn).child("Tgt"))

    def test_a_failing_relink_load_is_in_doubt_not_a_crash(self):
        nn, removed = self.removed_group()
        orig = self.box.load_tree

        def flaky(ord_str, depth=2, **kw):
            if depth == tools_write.SNAPSHOT_DEPTH:  # only the relink phase loads this deep
                raise box.BoxError("boom", box.CHANNEL, "loadSlots")
            return orig(ord_str, depth=depth, **kw)
        self.box.load_tree = flaky
        self.addCleanup(setattr, self.box, "load_tree", orig)
        plan = self.dry("n4_rollback", batch_id=removed["batch_id"])
        text = self.err("n4_rollback", batch_id=removed["batch_id"], dry_run=False,
                        confirmation_token=plan["confirmation_token"])
        self.assertIn("in-doubt", text)
        batch = [e["batch_id"] for e in self.lines("journal.jsonl")
                 if e.get("rollback_of") == removed["batch_id"]][0]
        self.assertEqual(self.journal().read(batch)["state"], "in-doubt")

    def test_the_nested_read_back_checks_type_and_annotation_not_just_presence(self):
        nn, gh = self.group()
        src_h = self.box.load_tree(FOLDER + "/" + nn, depth=2, **NO_SLEEP)["Src"]["h"]
        self.box.set_slot(src_h, "wsAnnotation", box.ws_annotation(5, 6, 7))
        removed = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn)
        self.assertEqual(self.rollback(removed["batch_id"])["verdict"], "verified")

        def tamper(kind):
            nn2, _ = self.group("Grp2")
            h = self.box.load_tree(FOLDER + "/" + nn2, depth=2, **NO_SLEEP)["Src"]["h"]
            self.box.set_slot(h, "wsAnnotation", box.ws_annotation(5, 6, 7))
            gone = self.run_write("n4_remove_component", parent_ord=FOLDER, name=nn2)
            orig = self.fake._sync

            def wrapped(op):
                res = orig(op)
                if op["nm"] == "a" and op["n"] == "Src":
                    node = self.fake.folder.child(nn2).child("Src")
                    if kind == "type":
                        node.type = "kitControl:BooleanConst"
                    else:
                        node.children = [c for c in node.children if c.name != "wsAnnotation"]
                return res
            self.fake._sync = wrapped
            try:
                return self.rollback(gone["batch_id"])["verdict"]
            finally:
                self.fake._sync = orig
        self.assertEqual(tamper("type"), "mismatch")
        self.assertEqual(tamper("annotation"), "mismatch")

    # ---- retry after a failed rollback (R3-rollback-retry-duplicates) ----------

    def break_readback_after_sync(self):
        sent = []
        orig_sync = self.fake._sync
        self.fake._sync = lambda op: (sent.append(op), orig_sync(op))[1]
        orig_load = self.box.load_tree

        def flaky(*a, **k):
            if sent:
                raise box.BoxError("boom", box.CHANNEL, "loadSlots")
            return orig_load(*a, **k)
        self.box.load_tree = flaky
        return lambda: setattr(self.box, "load_tree", orig_load)

    def test_a_rollback_accepted_but_unverified_cannot_be_retried(self):
        self.add("Pump")
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        heal = self.break_readback_after_sync()
        failed = self.rollback(out["batch_id"])
        heal()
        self.assertEqual(failed["verdict"], "failed")
        self.assertEqual(self.children(), ["Pump"])  # the remove was applied
        text = self.err("n4_rollback", batch_id=out["batch_id"])
        self.assertIn("in-doubt", text)
        self.assertIn(failed["batch_id"], text)
        self.assertIn("accepted", text)
        self.assertEqual(len(self.lines("journal.jsonl")), 4)  # no retry was journaled

    def test_a_failed_rollback_whose_ops_were_not_accepted_may_still_be_retried(self):
        out = self.run_write("n4_create_component", parent_ord=FOLDER, name="Pump",
                             type="kitControl:NumericConst")
        self.journal().append({"batch_id": "cd" * 16, "phase": "intent",
                               "rollback_of": out["batch_id"]})
        self.journal().append({"batch_id": "cd" * 16, "phase": "result", "verdict": "failed",
                               "inverse": [], "accepted": None})
        self.assertEqual(self.rollback(out["batch_id"])["verdict"], "verified")

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
