"""Station file reads and Px views: n4_read_file, n4_list_px_views, n4_px_summary.

The Px fixture (`fixtures/synthetic_view.px`) is written for these tests; it is not a
station file.
"""
import hashlib
import os
import stat
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402
from mcp_n4 import files, server, tools_read, tools_write  # noqa: E402

FIXTURES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures")
PASSWORD = "pw-123456"


def synthetic_px():
    with open(os.path.join(FIXTURES, "synthetic_view.px"), "rb") as fh:
        return fh.read()


@go_server.go_twin("files", "unit")
class TestFilePaths(unittest.TestCase):
    def test_a_file_ord_and_a_relative_path_name_the_same_station_file(self):
        for given in ("file:^px/General.px", "px/General.px", "station:|file:^px/General.px"):
            self.assertEqual(files.station_path(given), "px/General.px", given)

    def test_the_url_path_encodes_each_segment(self):
        self.assertEqual(files.url_path("px/floor plan#1?.px"), "/file/px/floor%20plan%231%3F.px")
        self.assertEqual(files.url_path("nav/NavFile.nav"), "/file/nav/NavFile.nav")

    def test_unsafe_paths_are_refused_before_any_request(self):
        for bad in ("", "..", "px/../config.bog", "/etc/passwd", "file:/etc/passwd",
                    "file:px/a.px", "file:^", "px|slot:/Drivers", "station:|slot:/x",
                    "px/", "px//a.px", "./px/a.px", "C:\\px\\a.px", "C:/px/a.px", "~/a.px",
                    "px/a\x00.px", "px/a\n.px", "x" * 1100, None, 5):
            with self.assertRaises(ValueError, msg=repr(bad)):
                files.station_path(bad)

    def test_bog_files_are_refused_like_the_file_servlet(self):
        for bad in ("config.bog", "CONFIG.BOG", "backups/config.bog.gz", "px/a.Bog"):
            with self.assertRaises(ValueError, msg=bad) as cm:
                files.station_path(bad)
            self.assertIn(".bog", str(cm.exception))

    def test_windows_path_aliases_of_refused_files_are_refused(self):
        # A Windows station resolves "config.bog." / "config.bog " / "config.bog::$DATA"
        # and ".. " to the refused name, so the suffix and ".." checks must see through them.
        for bad in ("config.bog.", "config.bog ", "config.bog. .", "config.bog::$DATA",
                    "px/.. /config.bog", "px/.../a.px", "px/... /a.px", "px/a.px:stream",
                    "backups/config.bog.gz.", "px/ /a.px"):
            with self.assertRaises(ValueError, msg=repr(bad)):
                files.station_path(bad)
        self.assertEqual(files.station_path("px/a b.px"), "px/a b.px")
        self.assertEqual(files.station_path("px/v1.2/a.px"), "px/v1.2/a.px")

    def test_text_types_are_detected_by_extension_or_content_type(self):
        for path in ("px/a.px", "a.xml", "nav/NavFile.nav", "c.chart", "d.json", "e.csv",
                     "f.html", "g.txt"):
            self.assertTrue(files.is_text(path, None), path)
        self.assertFalse(files.is_text("images/a.png", "image/png"))
        self.assertTrue(files.is_text("notes", "text/plain; charset=UTF-8"))
        self.assertFalse(files.is_text("blob.bin", "application/octet-stream"))


@go_server.go_twin("px", "unit")
class TestPxParser(unittest.TestCase):
    def setUp(self):
        self.summary = files.parse_px(synthetic_px().decode("utf-8"))

    def test_px_summary_reports_media_imports_canvas_and_layers(self):
        s = self.summary
        self.assertEqual(s["media"], "workbench:WbPxMedia")
        self.assertEqual(s["imports"], ["baja", "bajaui", "converters", "gx", "kitPx"])
        self.assertEqual(s["canvases"], [{"type": "CanvasPane", "name": "content",
                                          "view_size": "1024.0,768.0"}])
        self.assertEqual(s["view_size"], "1024.0,768.0")
        self.assertEqual(s["layers"], ["background", "labels", "values"])

    def test_px_summary_counts_widgets_and_bindings_apart(self):
        s = self.summary
        self.assertEqual(s["widget_counts"]["BoundLabel"], 3)
        self.assertEqual(s["widget_counts"]["Image"], 2)
        self.assertEqual(s["widget_counts"]["Bargraph"], 1)
        self.assertNotIn("ValueBinding", s["widget_counts"])
        self.assertEqual(s["binding_counts"], {"ValueBinding": 5, "ActionBinding": 1})

    def test_bound_ords_are_grouped_by_component_with_the_trailing_slot_stripped(self):
        bound = self.summary["bound_components"]
        self.assertEqual(bound["station:|slot:/Plant/AHU1/SupplyTemp"],
                         ["station:|slot:/Plant/AHU1/SupplyTemp",
                          "station:|slot:/Plant/AHU1/SupplyTemp/out"])
        self.assertEqual(bound["station:|slot:/Plant/AHU1/ValvePos"],
                         ["station:|slot:/Plant/AHU1/ValvePos/out"])
        self.assertIn("station:|slot:/Plant/AHU1/FanCmd", bound)
        self.assertIn("station:|slot:/Plant/Schedule", bound)
        self.assertIn("station:|slot:/Plant/AHU1", bound)  # the hyperlink target
        self.assertNotIn("history:/demo/SupplyTemp", bound)

    def test_images_included_px_and_charts_are_listed(self):
        s = self.summary
        self.assertEqual(s["images"], ["module://demoModule/rc/logo.png",
                                       "file:^images/floor%20plan.png"])
        self.assertEqual(s["included_px"], ["file:^px/includes/Header.px"])
        self.assertEqual(s["charts"], [{"type": "WebChart", "name": "trend",
                                        "ords": ["history:/demo/SupplyTemp"]}])
        self.assertIn("history:/demo/SupplyTemp", s["other_ords"])

    def test_label_texts_are_listed_and_bounded(self):
        self.assertEqual(self.summary["labels"][:3], ["Plant Overview", "Supply Temp",
                                                      "%out.value%"])
        many = "".join('<Label text="t%d"/>' % i for i in range(50))
        out = files.parse_px('<px media="m"><content>%s</content></px>' % many, max_items=10)
        self.assertEqual(len(out["labels"]), 10)
        self.assertTrue(out["labels_truncated"])

    def test_a_doctype_or_entity_declaration_is_refused(self):
        for text in ('<!DOCTYPE px [<!ENTITY a "x">]><px>&a;</px>',
                     '<?xml version="1.0"?>\n<!DOCTYPE px SYSTEM "http://x/y.dtd"><px/>'):
            with self.assertRaises(ValueError):
                files.parse_px(text)

    def test_a_non_px_document_or_broken_xml_is_refused(self):
        for text in ("<html><body/></html>", "<px><content>", "not xml"):
            with self.assertRaises(ValueError, msg=text):
                files.parse_px(text)

    def test_bound_component_strips_known_value_slots_and_slot_hops(self):
        self.assertEqual(files.bound_component("station:|slot:/A/B/out"), "station:|slot:/A/B")
        self.assertEqual(files.bound_component("station:|slot:/A/B|slot:out"),
                         "station:|slot:/A/B")
        self.assertEqual(files.bound_component("station:|slot:/A/B"), "station:|slot:/A/B")
        self.assertEqual(files.bound_component("slot:/A/B/in10"), "slot:/A/B")


class FileToolCase(unittest.TestCase):
    """A connected read-only server against a FakeStation that serves /file/ and /ord/."""

    def setUp(self, state_dir=None):
        self.fake = FakeStation(password=PASSWORD).start()
        self.addCleanup(self.fake.stop)
        self.srv = go_server.make_server(
            allow_http=True, env={"MCP_N4_USER": "admin", "MCP_N4_PASSWORD": PASSWORD},
            stations={"FakeStation": self.fake.url}, state_dir=state_dir)
        self.addCleanup(go_server.close_server, self.srv)
        self.ok("n4_connect", station="FakeStation")
        self.fake.files["px/General.px"] = (synthetic_px(), "text/xml")

    def call(self, name, **args):
        res = self.srv.dispatch({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                                 "params": {"name": name, "arguments": args}})
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


class TestReadFileTool(FileToolCase):
    def test_a_px_file_is_returned_as_text_over_the_file_servlet_path(self):
        out = self.ok("n4_read_file", file_ord="file:^px/General.px")
        self.assertEqual((out["status"], out["http_status"]), ("ok", 200))
        self.assertEqual(out["path"], "px/General.px")
        self.assertEqual(out["text"], synthetic_px().decode("utf-8"))
        self.assertEqual(out["size_bytes"], len(synthetic_px()))
        self.assertEqual(out["sha256"], hashlib.sha256(synthetic_px()).hexdigest())
        self.assertFalse(out["truncated"])
        self.assertEqual(self.fake.file_requests, ["px/General.px"])

    def test_a_binary_file_returns_size_and_digest_without_content(self):
        png = b"\x89PNG\r\n\x1a\n" + bytes(range(256)) * 4
        self.fake.files["images/logo.png"] = (png, "image/png")
        out = self.ok("n4_read_file", file_ord="images/logo.png")
        self.assertNotIn("text", out)
        self.assertTrue(out["binary"])
        self.assertEqual((out["size_bytes"], out["content_type"]), (len(png), "image/png"))
        self.assertEqual(out["sha256"], hashlib.sha256(png).hexdigest())

    def test_the_size_cap_truncates_and_says_so(self):
        self.fake.files["px/Big.px"] = (b"<px>" + b"a" * 5000 + b"</px>", "text/xml")
        out = self.ok("n4_read_file", file_ord="px/Big.px", max_bytes=1024)
        self.assertTrue(out["truncated"])
        self.assertEqual(out["size_bytes"], 1024)
        self.assertEqual(len(out["text"]), 1024)
        self.assertIsNone(out["sha256"])  # a digest of a prefix would mislead
        self.assertEqual(out["content_length"], 5009)

    @go_server.go_twin("files", "internals")
    def test_missing_and_forbidden_files_are_typed_results_without_a_latch(self):
        out = self.ok("n4_read_file", file_ord="file:^nav/NavFile.nav")
        self.assertEqual((out["status"], out["http_status"]), ("not_found", 404))
        self.fake.files["px/Secret.px"] = 403
        out = self.ok("n4_read_file", file_ord="px/Secret.px")
        self.assertEqual((out["status"], out["http_status"]), ("forbidden", 403))
        self.assertIsNone(self.srv.ctx.auth_latch)
        self.assertEqual(self.ok("n4_read_file", file_ord="px/General.px")["status"], "ok")

    def test_unsafe_paths_and_bog_files_send_nothing(self):
        for bad in ("../config.bog", "/etc/passwd", "px/", "px|slot:/", "config.bog",
                    "file:/etc/passwd"):
            self.err("n4_read_file", file_ord=bad)
        self.assertEqual(self.fake.file_requests, [])

    def test_a_401_on_the_file_path_latches_and_is_not_retried(self):
        self.fake.password = "changed"  # the station now rejects the session's credentials
        before = self.fake.about_requests
        text = self.err("n4_read_file", file_ord="px/General.px")
        self.assertIn("HTTPBasicScheme", text)
        self.assertNotIn(PASSWORD, text)
        self.assertEqual(self.fake.about_requests - before, 1)
        self.assertIn("paused", self.err("n4_read_file", file_ord="px/General.px"))
        self.assertEqual(self.fake.about_requests - before, 1)  # the latch sent nothing

    @go_server.go_twin("files", "internals")
    def test_the_read_is_recorded_as_a_session_observation(self):
        self.ok("n4_read_file", file_ord="file:^px/General.px")
        obs = self.srv.ctx.observations[-1]
        self.assertEqual((obs["tool"], obs["ord"]), ("n4_read_file", "file:^px/General.px"))

    def test_the_tools_are_listed_as_read_only(self):
        tools = {t["name"]: t for t in self.srv.dispatch(
            {"jsonrpc": "2.0", "id": 1, "method": "tools/list"})["result"]["tools"]}
        for name in ("n4_read_file", "n4_list_px_views", "n4_px_summary"):
            self.assertEqual(tools[name]["annotations"]["readOnlyHint"], True, name)


class TestReadFileOutput(FileToolCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.state = os.path.join(tmp.name, "state")
        os.makedirs(self.state, mode=0o700)
        super().setUp(state_dir=self.state)

    def test_output_file_saves_the_complete_bytes_privately(self):
        png = b"\x89PNG\r\n\x1a\n" + b"\x00" * 300
        self.fake.files["images/logo.png"] = (png, "image/png")
        out = self.ok("n4_read_file", file_ord="images/logo.png", output_file="logo.png")
        path = os.path.join(self.state, "files", "logo.png")
        self.assertEqual(out["output_file"], path)
        with open(path, "rb") as fh:
            self.assertEqual(fh.read(), png)
        self.assertEqual(stat.S_IMODE(os.stat(path).st_mode), 0o600)

    def test_a_truncated_read_is_not_saved_and_a_path_name_is_refused(self):
        self.fake.files["px/Big.px"] = (b"x" * 4000, "text/xml")
        out = self.ok("n4_read_file", file_ord="px/Big.px", max_bytes=1024,
                      output_file="big.px")
        self.assertNotIn("output_file", out)
        self.assertIn("truncated", out["output_skipped"])
        self.assertFalse(os.path.exists(os.path.join(self.state, "files", "big.px")))
        self.err("n4_read_file", file_ord="px/General.px", output_file="../escape.px")
        self.assertEqual(self.fake.file_requests, ["px/Big.px"])  # refused before the GET


PX_VIEW_ROWS = (b"Slot Path,Display Name,Px File,Media\n"
                b"NULL,General,file:^px/General.px,workbench:WbPxMedia\n"
                b"slot:/Plant/AHU1/graphic,AHU Graphic,file:^px/AHU.px,workbench:WbPxMedia\n")


class TestListPxViewsTool(FileToolCase):
    def test_px_views_are_listed_with_their_file_and_media(self):
        self.fake.ord_routes = [("baja:PxView", PX_VIEW_ROWS)]
        out = self.ok("n4_list_px_views")
        self.assertEqual(out["row_count"], 2)
        first, second = out["views"]
        self.assertEqual(first, {"display_name": "General", "px_file": "file:^px/General.px",
                                 "media": "workbench:WbPxMedia", "slot_path": None,
                                 "owner_path": None})
        self.assertEqual(second["owner_path"], "station:|slot:/Plant/AHU1")
        self.assertIn("NULL", out["owner_note"])
        ord_text = self.fake.ord_requests[0]
        self.assertTrue(ord_text.startswith("station:|slot:/|bql:select slotPath, displayName, "
                                            "pxFile, media from baja:PxView"), ord_text)
        self.assertTrue(ord_text.endswith("|view:file:ITableToCsv"))

    def test_the_root_scopes_the_query_and_a_pipe_is_refused(self):
        self.fake.ord_routes = [("baja:PxView", PX_VIEW_ROWS)]
        self.ok("n4_list_px_views", root="station:|slot:/Plant")
        self.assertTrue(self.fake.ord_requests[0].startswith("station:|slot:/Plant|bql:"))
        self.err("n4_list_px_views", root="/Plant|bql:select x from y")
        self.assertEqual(len(self.fake.ord_requests), 1)


class TestPxSummaryTool(FileToolCase):
    def test_the_summary_parses_the_fetched_px(self):
        out = self.ok("n4_px_summary", file_ord="file:^px/General.px")
        self.assertEqual(out["status"], "ok")
        self.assertEqual(out["summary"]["media"], "workbench:WbPxMedia")
        self.assertEqual(out["summary"]["included_px"], ["file:^px/includes/Header.px"])
        self.assertNotIn("text", out)

    def test_a_missing_px_is_a_typed_result(self):
        out = self.ok("n4_px_summary", file_ord="px/Nope.px")
        self.assertEqual(out["status"], "not_found")
        self.assertNotIn("summary", out)

    def test_a_truncated_px_is_refused_with_the_cap_named(self):
        self.assertIn("max_bytes", self.err("n4_px_summary", file_ord="px/General.px",
                                            max_bytes=1024))

    def test_a_file_that_is_not_px_xml_is_refused(self):
        self.fake.files["px/Bad.px"] = (b"<html/>", "text/xml")
        self.assertIn("px", self.err("n4_px_summary", file_ord="px/Bad.px"))


if __name__ == "__main__":
    unittest.main()
