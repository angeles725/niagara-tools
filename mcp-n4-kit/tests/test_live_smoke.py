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


class StubClient:
    """Scripted in-memory stand-in for McpClient (Scenario-level tests, no subprocess)."""
    WRITES = {"n4_create_component", "n4_create_link", "n4_invoke_action", "n4_save_station",
              "n4_remove_component", "n4_rollback"}

    def __init__(self, hook=None):
        self.present, self.calls, self.vals, self.hook = False, [], {}, hook
        self.counts = {}

    def call(self, tool, **args):
        executing = "confirmation_token" in args
        key = (tool, executing)
        self.counts[key] = self.counts.get(key, 0) + 1
        self.calls.append((tool, executing, args))
        if self.hook:
            override = self.hook(self, tool, executing, args)
            if override is not None:
                return override
        if tool in self.WRITES and not executing:
            return False, {"confirmation_token": "t", "plan_hash": "h"}
        ok = {"batch_id": "b-" + tool, "verdict": "verified", "observed": {}}
        if tool == "n4_connect":
            return False, {"station_name": args["station"]}
        if tool == "n4_navigate":
            return False, {"children": [{"name": "McpSmoke" if self.present else "Services"}]}
        if tool == "n4_create_component":
            self.present = self.present or args["name"] == live_smoke.SCRATCH
        elif tool == "n4_remove_component":
            self.present = False
        elif tool == "n4_rollback":
            self.present = True
        elif tool == "n4_invoke_action":
            self.vals[args["ord"].rsplit("/", 1)[-1]] = args["arg"]
        elif tool == "n4_find_dangling_outputs":
            return False, {"dangling": [{"path": live_smoke.FOLDER_ORD + "/Cooling"}]}
        elif tool == "n4_read_slots":
            hot = self.vals.get("Temp", 0) > self.vals.get("Setpoint", 0)
            return False, {"slots": [{"name": "out", "value": hot}]}
        elif tool == "n4_save_station":
            return False, dict(ok, persisted=True, evidence="stub")
        return False, ok


def stub_scenario(hook=None):
    client = StubClient(hook)
    return client, live_smoke.Scenario(client, "SmokeStation", 0.2, lambda text: None)


def names(scenario):
    return [s["name"] for s in scenario.steps]


def verdicts(scenario):
    return {s["name"]: s["verdict"] for s in scenario.steps}


class TestSecretScrubBeforeEncoding(unittest.TestCase):
    NASTY = 'p"w\\x\n\x01caf\u00e9-unique'

    def test_render_report_removes_raw_and_escaped_forms(self):
        report = {"error": "bad " + self.NASTY, "steps": [{"detail": {"k": [self.NASTY]}}],
                  "server_stderr": ["login " + self.NASTY], self.NASTY: 1}
        text = live_smoke.render_report(report, [self.NASTY, ""])
        for form in (self.NASTY, json.dumps(self.NASTY)[1:-1],
                     json.dumps(self.NASTY, ensure_ascii=False)[1:-1]):
            self.assertNotIn(form, text)
        self.assertIn("***", text)
        self.assertEqual(json.loads(text)["error"], "bad ***")

    def test_scrub_value_walks_nested_structures(self):
        out = live_smoke.scrub_value({"a": ["x S y", ("S",)], "n": 3, "b": None}, ["S"])
        self.assertEqual(out, {"a": ["x *** y", ["***"]], "n": 3, "b": None})


class TestWriteBounds(unittest.TestCase):
    def test_every_write_in_a_full_run_targets_the_scratch_folder_or_its_root_create_remove(self):
        client, sc = stub_scenario()
        sc.run()
        writes = [(t, a) for t, ex, a in client.calls if t in StubClient.WRITES and ex]
        self.assertGreaterEqual(len(writes), 12)
        for tool, args in writes:
            ords = [args[k] for k in ("ord", "source_ord", "target_ord") if k in args]
            ords += [args["parent_ord"].rstrip("/") + "/" + args["name"]] if "parent_ord" in args else []
            for o in ords:
                self.assertTrue(o == live_smoke.FOLDER_ORD
                                or o.startswith(live_smoke.FOLDER_ORD + "/"), (tool, args))
            if args.get("parent_ord") == live_smoke.ROOT_ORD:
                self.assertEqual(args["name"], live_smoke.SCRATCH)
                self.assertIn(tool, ("n4_create_component", "n4_remove_component"))

    def test_the_runner_refuses_a_write_outside_the_scratch_folder(self):
        client, sc = stub_scenario()
        for tool, args in (("n4_create_component", dict(parent_ord=live_smoke.ROOT_ORD,
                                                        name="Other", type="baja:Folder")),
                           ("n4_remove_component", dict(parent_ord=live_smoke.ROOT_ORD, name="X")),
                           ("n4_invoke_action", dict(ord="station:|slot:/Other", action="set")),
                           ("n4_create_link", dict(source_ord=live_smoke.FOLDER_ORD + "/A",
                                                   target_ord="station:|slot:/B"))):
            with self.assertRaises(live_smoke.SmokeError):
                sc.write({}, tool, **args)
        self.assertEqual(client.calls, [])


class TestUnknownCreateOutcome(unittest.TestCase):
    def test_timeout_after_the_folder_create_applied_still_cleans_up(self):
        def hook(client, tool, executing, args):
            if tool == "n4_create_component" and executing and args["name"] == "McpSmoke":
                client.present = True  # the station applied it, the reply never arrived
                raise live_smoke.SmokeError("timeout waiting for tools/call")
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertFalse(client.present)
        self.assertIn("cleanup", names(sc))
        self.assertEqual(verdicts(sc)["cleanup"], "verified")
        self.assertEqual(verdicts(sc)["create Folder McpSmoke"], "failed")

    def test_a_refused_dry_run_marks_nothing_and_does_not_clean_up(self):
        def hook(client, tool, executing, args):
            if tool == "n4_create_component" and not executing:
                return True, "refused"
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertFalse(any(n.startswith("cleanup") for n in names(sc)))


class TestStepContainment(unittest.TestCase):
    def test_a_malformed_reply_is_a_failed_step_not_an_exception(self):
        def hook(client, tool, executing, args):
            if tool == "n4_navigate":
                return False, {"children": [{"nom": "x"}]}
        client, sc = stub_scenario(hook)
        sc.run()  # must not raise KeyError
        self.assertEqual(verdicts(sc)["folder absent before"], "failed")

    def test_a_failed_read_after_the_rollback_still_cleans_up(self):
        def hook(client, tool, executing, args):
            if tool == "n4_navigate" and client.counts.get(("n4_rollback", True)) \
                    and not client.counts.get("failed_once"):
                client.counts["failed_once"] = 1
                raise live_smoke.SmokeError("timeout waiting for tools/call")
        client, sc = stub_scenario(hook)
        sc.run()  # must not raise
        self.assertEqual(verdicts(sc)["remove folder (again)"], "failed")
        self.assertFalse(client.present)
        self.assertEqual(verdicts(sc)["cleanup"], "verified")

    def test_cleanup_reads_are_recorded_steps(self):
        def hook(client, tool, executing, args):
            if tool == "n4_invoke_action" and executing:
                raise live_smoke.SmokeError("boom")
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertIn("cleanup: folder present?", names(sc))

    def test_cleanup_assumes_present_when_the_probe_fails(self):
        state = {"after_abort": False}

        def hook(client, tool, executing, args):
            if tool == "n4_invoke_action" and executing:
                state["after_abort"] = True
                raise live_smoke.SmokeError("boom")
            if tool == "n4_navigate" and state["after_abort"]:
                raise live_smoke.SmokeError("navigate down")
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertEqual(verdicts(sc)["cleanup: folder present?"], "failed")
        self.assertEqual(verdicts(sc)["cleanup"], "verified")
        self.assertFalse(client.present)


class TestCleanupSave(unittest.TestCase):
    def test_abort_after_the_first_save_removes_and_saves_again(self):
        def hook(client, tool, executing, args):
            if tool == "n4_remove_component" and executing and \
                    client.counts[("n4_remove_component", True)] == 1:
                raise live_smoke.SmokeError("timeout waiting for tools/call")
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertFalse(client.present)
        self.assertEqual(names(sc)[-2:], ["cleanup", "cleanup save"])
        self.assertEqual(verdicts(sc)["cleanup save"], "verified")
        self.assertEqual(client.counts[("n4_save_station", True)], 2)

    def test_no_cleanup_save_when_the_folder_was_never_persisted(self):
        def hook(client, tool, executing, args):
            if tool == "n4_invoke_action" and executing:
                raise live_smoke.SmokeError("boom")
        client, sc = stub_scenario(hook)
        sc.run()
        self.assertNotIn("cleanup save", names(sc))


class TestSingleScenarioSource(unittest.TestCase):
    def test_plan_rows_equal_the_executed_step_names_and_tools(self):
        client, sc = stub_scenario()
        sc.run()
        executed = [(s["name"], s["tool"]) for s in sc.steps]
        self.assertEqual(live_smoke.plan_rows(), executed)
        self.assertEqual(client.present, False)

    def test_plan_text_lists_every_row_and_the_components(self):
        text = live_smoke.plan_text()
        for name, _tool in live_smoke.plan_rows():
            self.assertIn(name, text)
        self.assertIn("kitControl:GreaterThan", text)


class TestSaveParameter(unittest.TestCase):
    def test_save_takes_an_explicit_folder_present_flag(self):
        client, sc = stub_scenario()
        entry = {}
        sc.save(folder_present=True)(entry)
        self.assertTrue(sc.saved_with_folder)
        sc.save(folder_present=False)(entry)
        self.assertFalse(sc.saved_with_folder)


class TestShape(unittest.TestCase):
    def test_import_has_no_side_effects_and_main_is_guarded(self):
        self.assertTrue(callable(live_smoke.main))

    def test_scope_is_the_scratch_folder_plus_the_root(self):
        self.assertEqual(live_smoke.write_scopes(), ["station:|slot:/McpSmoke", "station:|slot:/"])


if __name__ == "__main__":
    unittest.main()
