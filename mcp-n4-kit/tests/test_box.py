import importlib
import os
import socket
import subprocess
import sys
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box  # noqa: E402

NO_SLEEP = dict(sleep=lambda s: None)


class BoxTestCase(unittest.TestCase):
    def setUp(self):
        self.fake = FakeStation().start()
        self.addCleanup(self.fake.stop)

    def client(self, **kw):
        kw.setdefault("allow_http", True)
        return box.BoxClient(self.fake.url, "admin", kw.pop("password", "secret"), **kw)

    def opened(self):
        c = self.client()
        c.open()
        self.addCleanup(c.close)
        return c

    def add(self, c, name, type_="kitControl:NumericConst", **kw):
        c.add_component("3", name, type_, **kw)
        return c.load_tree("station:|slot:/Folder", depth=2, **NO_SLEEP)[name]["h"]


class TestSecurity(BoxTestCase):
    def test_auth_failure_raises_once_without_retry(self):
        c = self.client(password="wrong")
        with self.assertRaises(box.AuthError):
            c.open()
        self.assertEqual(self.fake.requests, 1)

    def test_http_refused_without_allow_http(self):
        with self.assertRaises(ValueError):
            box.BoxClient("http://127.0.0.1:1", "u", "p")

    def test_https_accepted_without_flag(self):
        box.BoxClient("https://station.example", "u", "p")

    def test_repr_hides_password(self):
        c = self.client(password="s3cr3t-pw")
        self.assertNotIn("s3cr3t-pw", repr(c))
        self.assertNotIn("s3cr3t-pw", str(c))

    def test_import_makes_no_network_call(self):
        with mock.patch.object(socket.socket, "connect", side_effect=AssertionError("network")):
            importlib.reload(box)

    def test_import_has_no_side_effects_in_fresh_process(self):
        kit = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        r = subprocess.run([sys.executable, "-c", "import mcp_n4.box"], cwd=kit,
                           capture_output=True, text=True)
        self.assertEqual((r.returncode, r.stdout, r.stderr), (0, "", ""))


class TestSession(BoxTestCase):
    def test_open_returns_root(self):
        c = self.client()
        root = c.open()
        self.assertEqual(root["h"], "2")
        self.assertEqual(root["t"], "baja:Station")

    def test_context_manager_deletes_session(self):
        with self.client() as c:
            self.assertEqual(self.fake.sessions, {"sess-1"})
        self.assertEqual(self.fake.sessions, set())

    def test_station_error_frame_raises_boxerror(self):
        c = self.opened()
        with self.assertRaises(box.BoxError):
            c.ssc("noSuchKey", None)


class TestReadWrite(BoxTestCase):
    def test_add_component_visible_with_handle_and_annotation(self):
        c = self.opened()
        res = c.add_component("3", "Temp", "kitControl:NumericConst", ws=box.ws_annotation(10, 20))
        self.assertEqual(res["nn"], "Temp")
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertTrue(nodes["Temp"]["h"])
        self.assertEqual(nodes["Temp/wsAnnotation"]["v"], "10,20,8")

    def test_load_tree_root_key_is_empty_path(self):
        c = self.opened()
        nodes = c.load_tree("station:", **NO_SLEEP)
        self.assertEqual(nodes[""]["h"], "2")
        self.assertIn("Folder", nodes)

    def test_load_tree_gives_up_when_no_event(self):
        c = self.opened()
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:", attempts=3, **NO_SLEEP)

    def test_sync_rejects_multiple_ops(self):
        c = self.opened()
        with self.assertRaises(ValueError):
            c.sync([{"nm": "v", "h": "3", "n": "a"}, {"nm": "v", "h": "3", "n": "b"}])

    def test_set_slot_rejects_partial_status_by_default(self):
        c = self.opened()
        h = self.add(c, "Temp")
        for path in ("fallback/value", "out/status"):
            with self.assertRaises(ValueError):
                c.set_slot(h, path, box.bson_double(30.0))
        c.set_slot(h, "out", box.bson_status_numeric(1.0))
        c.set_slot(h, "out/value", box.bson_double(30.0), allow_partial_status=True)

    def test_set_slot_whole_status_numeric_reads_back_ok(self):
        c = self.opened()
        h = self.add(c, "Temp")
        c.set_slot(h, "out", box.bson_status_numeric(30.0))
        nodes = c.load_tree("station:|slot:/Folder", depth=3, **NO_SLEEP)
        got = box.status_value(nodes, "Temp/out")
        self.assertEqual(got["value"], "30.0")
        self.assertTrue(box.parse_status(got["status"])["ok"])

    def test_check_links_returns_link_name_and_creates_link(self):
        c = self.opened()
        a, b = self.add(c, "A"), self.add(c, "B")
        res = c.check_links(a, "out", b, "in10")
        self.assertEqual(res[0]["s"], "Link")
        self.assertTrue(res[0]["v"])
        nodes = c.load_tree("station:|slot:/Folder/B", depth=2, **NO_SLEEP)
        self.assertEqual(nodes["Link"]["t"], "baja:Link")
        self.assertEqual(nodes["Link/sourceSlotName"]["v"], "out")
        self.assertEqual(nodes["Link/targetSlotName"]["v"], "in10")

    def test_invoke_action_set_writes_fallback_whole(self):
        c = self.opened()
        h = self.add(c, "W", "control:NumericWritable")
        c.invoke_action(h, "set", box.bson_double(7.5))
        nodes = c.load_tree("station:|slot:/Folder", depth=3, **NO_SLEEP)
        got = box.status_value(nodes, "W/fallback")
        self.assertEqual(got["value"], "7.5")
        self.assertEqual(got["status"], "0")

    def test_save_station_increments_counter(self):
        c = self.opened()
        c.save_station()
        self.assertEqual(self.fake.saves, 1)

    def test_remove_component(self):
        c = self.opened()
        self.add(c, "Gone")
        c.remove_component("3", "Gone")
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertNotIn("Gone", nodes)


class TestPureHelpers(unittest.TestCase):
    def test_status_value_applies_type_defaults(self):
        nodes = {
            "sb": {"t": "baja:StatusBoolean"},
            "sn": {"t": "baja:StatusNumeric"},
        }
        self.assertEqual(box.status_value(nodes, "sb"), {"value": "false", "status": "0"})
        self.assertEqual(box.status_value(nodes, "sn"), {"value": "0.0", "status": "0"})

    def test_status_value_reads_present_children(self):
        nodes = {
            "sb": {"t": "baja:StatusBoolean"},
            "sb/value": {"t": "baja:Boolean", "v": "true"},
            "sb/status": {"t": "baja:Status", "v": "40"},
        }
        self.assertEqual(box.status_value(nodes, "sb"), {"value": "true", "status": "40"})

    def test_children_lists_direct_children_only(self):
        nodes = {"": {}, "a": {}, "a/b": {}, "c": {}}
        self.assertEqual(sorted(box.children(nodes, "")), ["a", "c"])
        self.assertEqual(sorted(box.children(nodes, "a")), ["b"])

    def test_parse_status(self):
        s = box.parse_status("40;activeLevel=e:17@control:PriorityLevel")
        self.assertEqual(s["bits"], 0x40)
        self.assertTrue(s["null"])
        self.assertFalse(s["ok"])
        self.assertEqual(s["facets"], "activeLevel=e:17@control:PriorityLevel")
        ok = box.parse_status("0")
        self.assertTrue(ok["ok"])
        self.assertFalse(ok["null"])
        self.assertEqual(ok["facets"], "")

    def test_encoders(self):
        self.assertEqual(box.bson_double(30), {"nm": "p", "t": "baja:Double", "v": "30.0"})
        self.assertEqual(box.bson_bool(True)["v"], "true")
        sb = box.bson_status_boolean(False)
        self.assertEqual(sb["s"][0]["v"], "false")
        self.assertEqual(sb["s"][1], {"nm": "p", "n": "status", "t": "baja:Status", "v": "0"})
        self.assertEqual(box.ws_annotation(1, 2, 3)["v"], "1,2,3")


if __name__ == "__main__":
    unittest.main()
