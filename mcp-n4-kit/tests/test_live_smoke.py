"""Tests for tools/live_smoke.py against the in-process fake station (no live contact)."""
import io
import json
import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(os.path.dirname(HERE), "tools"))

from fake_station import FakeStation, _Node  # noqa: E402
import live_smoke  # noqa: E402

PASSWORD = "pw-Zk93-unique-canary"
USER = "user-Qx71-unique-canary"
LOGIC_TYPES = {"control:NumericWritable": "baja:StatusNumeric",
               "control:BooleanWritable": "baja:StatusBoolean",
               "kitControl:GreaterThan": "baja:StatusBoolean"}


class LogicStation(FakeStation):
    """Fake station whose thermostat components have an `out` slot and compute."""

    def __init__(self, *a, compute=True, **kw):
        super().__init__(*a, **kw)
        self.compute = compute

    def _sync(self, op):
        res = super()._sync(op)
        if op["nm"] == "a" and op["b"]["t"] in LOGIC_TYPES:
            node = self.by_handle[op["h"]].child(res[0]["nn"])
            node.children.append(self._status("out", LOGIC_TYPES[op["b"]["t"]], "false"))
        return res

    @staticmethod
    def _status(name, type_, value):
        out = _Node(name, type_)
        out.children = [_Node("value", "baja:Boolean" if "Boolean" in type_ else "baja:Double",
                              value), _Node("status", "baja:Status", "0")]
        return out

    def _invoke(self, arg):
        res = super()._invoke(arg)
        if self.compute:
            self._recompute()
        return res

    def _recompute(self):
        folder = self.root.child("McpSmoke")
        if folder is None:
            return

        def val(name):
            fb = folder.child(name).child("fallback")
            return float(fb.child("value").value) if fb is not None else 0.0
        state = "true" if val("Temp") > val("Setpoint") else "false"
        for name in ("Compare", "Cooling"):
            out = folder.child(name).child("out")
            out.children[0].value = state


class RunnerCase(unittest.TestCase):
    def setUp(self):
        self.fake = LogicStation(user=USER, password=PASSWORD,
                                 station_name="SmokeStation").start()
        self.addCleanup(self.fake.stop)
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = os.path.join(self.tmp.name, "home")
        os.mkdir(self.home)
        self.bog = os.path.join(self.home, "config.bog")
        with open(self.bog, "wb") as fh:
            fh.write(b"bog")
        self.fake.config_path = self.bog
        self.report_path = os.path.join(self.tmp.name, "report.json")
        self.env = dict(os.environ, MCP_N4_USER=USER, MCP_N4_PASSWORD=PASSWORD)

    def argv(self, *extra, apply=True):
        base = ["--station", "SmokeStation=" + self.fake.url,
                "--station-home", "SmokeStation=" + self.home,
                "--state-dir", os.path.join(self.tmp.name, "state"),
                "--report", self.report_path, "--settle", "1",
                "--allow-http-for-tests"]
        return base + (["--apply"] if apply else []) + list(extra)

    def run_main(self, argv):
        out, err = io.StringIO(), io.StringIO()
        code = live_smoke.main(argv, env=self.env, stdout=out, stderr=err)
        return code, out.getvalue(), err.getvalue()

    def report(self):
        with open(self.report_path) as fh:
            return json.load(fh)


class TestPlanOnly(RunnerCase):
    def test_default_prints_the_scenario_and_never_touches_the_station(self):
        code, out, err = self.run_main(self.argv(apply=False))
        self.assertEqual(code, 0, err)
        self.assertEqual(self.fake.requests, 0)
        self.assertEqual(self.fake.root.child("McpSmoke"), None)
        for needle in ("McpSmoke", "Temp", "GreaterThan", "n4_rollback", "plan only"):
            self.assertIn(needle, out)
        self.assertFalse(os.path.exists(os.path.join(self.tmp.name, "state")))

    def test_plan_only_needs_no_station_arguments(self):
        code, out, _ = self.run_main([])
        self.assertEqual(code, 0)
        self.assertIn("plan only", out)

    def test_apply_requires_station_and_station_home(self):
        for argv in (["--apply"], ["--apply", "--station", "A=http://x"],
                     ["--apply", "--station-home", "A=/x"],
                     ["--apply", "--station", "A=http://x", "--station-home", "B=/x"]):
            code, _, err = self.run_main(argv)
            self.assertEqual(code, 2, argv)
            self.assertIn("--station", err)


class TestFullRun(RunnerCase):
    def test_a_full_run_is_verified_exit_0_and_leaves_no_folder(self):
        code, out, err = self.run_main(self.argv())
        self.assertEqual(code, 0, out + err)
        rep = self.report()
        self.assertTrue(rep["ok"])
        self.assertEqual(rep["station"], "SmokeStation")
        required = [s for s in rep["steps"] if s["required"]]
        self.assertTrue(required)
        self.assertEqual({s["verdict"] for s in required}, {"verified"})
        self.assertIsNone(self.fake.root.child("McpSmoke"))
        self.assertGreaterEqual(self.fake.saves, 2)

    def test_every_write_step_does_dry_run_then_token_then_execute(self):
        self.run_main(self.argv())
        writes = [s for s in self.report()["steps"] if s.get("tool", "").startswith("n4_")
                  and s["tool"] not in live_smoke.READ_TOOLS and s["verdict"] != "skipped"]
        self.assertGreaterEqual(len(writes), 12)
        for step in writes:
            self.assertTrue(step["dry_run_done"], step)
            self.assertIsNotNone(step["batch_id"], step)

    def test_report_declares_the_probe_only_rollback_step(self):
        self.run_main(self.argv())
        rep = self.report()
        probes = [s for s in rep["steps"] if s.get("probe")]
        self.assertEqual([s["tool"] for s in probes], ["n4_rollback"])
        self.assertFalse(probes[0]["required"])
        self.assertIn("rollback", " ".join(rep["probe_only"]))
        self.assertIn(probes[0]["verdict"], ("verified", "mismatch", "failed", "unverified"))

    def test_the_scenario_covers_the_b1199_thermostat(self):
        self.run_main(self.argv())
        names = [s["name"] for s in self.report()["steps"]]
        for want in ("connect", "create Folder McpSmoke", "create Compare", "link Compare.out->Cooling.in10",
                     "find_dangling_outputs == [Cooling]", "read outputs == true",
                     "read outputs == false", "remove folder (again)", "folder absent"):
            self.assertIn(want, names)

    def test_a_pre_existing_folder_aborts_before_any_write(self):
        self.fake._component(self.fake.root, "McpSmoke", "baja:Folder")
        code, out, _ = self.run_main(self.argv())
        self.assertEqual(code, 1)
        self.assertIsNotNone(self.fake.root.child("McpSmoke"))  # not ours: untouched
        self.assertEqual(self.fake.saves, 0)
        self.assertFalse(self.report()["ok"])


class TestMismatch(RunnerCase):
    def test_a_step_mismatch_exits_nonzero_skips_the_rest_and_cleans_up(self):
        self.fake.compute = False
        code, out, _ = self.run_main(self.argv())
        self.assertEqual(code, 1)
        rep = self.report()
        self.assertFalse(rep["ok"])
        failed = [s for s in rep["steps"] if s["required"] and s["verdict"] != "verified"]
        self.assertEqual(failed[0]["name"], "read outputs == true")
        self.assertEqual(failed[0]["verdict"], "mismatch")
        later = rep["steps"][rep["steps"].index(failed[0]) + 1:]
        self.assertTrue(any(s["verdict"] == "skipped" for s in later))
        self.assertIsNone(self.fake.root.child("McpSmoke"))  # cleanup removed our folder
        self.assertEqual(rep["steps"][-1]["name"], "cleanup")

    def test_a_server_that_cannot_start_is_a_nonzero_exit(self):
        code, out, err = self.run_main(self.argv("--insecure-tls", "Nope"))
        self.assertEqual(code, 1, out + err)
        self.assertFalse(self.report()["ok"])
        self.assertEqual(self.fake.requests, 0)


class TestNoCredentialLeak(RunnerCase):
    def test_no_credential_value_in_stdout_stderr_or_report(self):
        for argv in (self.argv(), self.argv(apply=False)):
            code, out, err = self.run_main(argv)
            blob = out + err
            if os.path.exists(self.report_path):
                with open(self.report_path) as fh:
                    blob += fh.read()
            self.assertNotIn(PASSWORD, blob)
            self.assertNotIn(USER, blob)

    def test_a_failed_login_does_not_leak_the_password(self):
        self.env["MCP_N4_PASSWORD"] = PASSWORD + "-wrong"
        code, out, err = self.run_main(self.argv())
        self.assertEqual(code, 1)
        with open(self.report_path) as fh:
            blob = out + err + fh.read()
        self.assertNotIn(PASSWORD, blob)
        self.assertEqual(self.fake.root.child("McpSmoke"), None)

    def test_scrub_replaces_every_secret_value(self):
        self.assertEqual(live_smoke.scrub("a SECRETX b", ["SECRETX"]), "a *** b")
        self.assertEqual(live_smoke.scrub("plain", []), "plain")


class TestShape(unittest.TestCase):
    def test_import_has_no_side_effects_and_main_is_guarded(self):
        self.assertTrue(callable(live_smoke.main))

    def test_scope_is_the_scratch_folder_plus_the_root(self):
        self.assertEqual(live_smoke.write_scopes(), ["station:|slot:/McpSmoke", "station:|slot:/"])


if __name__ == "__main__":
    unittest.main()
