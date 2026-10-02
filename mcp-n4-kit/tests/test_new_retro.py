"""Tests for tools/new_retro.py: writes the session retro file and the INDEX row."""
import io
import os
import shutil
import sys
import tempfile
import unittest
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
KIT = os.path.dirname(HERE)
sys.path.insert(0, KIT)
sys.path.insert(0, os.path.join(KIT, "tools"))

from mcp_n4 import safety  # noqa: E402
import new_retro  # noqa: E402

B1 = "%032x" % 1


class TestNewRetro(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.state = os.path.join(tmp.name, "state")
        self.out = os.path.join(tmp.name, "retros")
        os.makedirs(self.out)
        shutil.copy(os.path.join(KIT, "retros", "INDEX.md"), self.out)
        self.index = os.path.join(self.out, "INDEX.md")

    def friction(self):
        for row in ({"batch_id": B1, "ts": "2026-10-01T11:00:00+00:00", "tool": "n4_set_slot",
                     "ops": [{"nm": "s"}], "phase": "intent", "station_name": "LLM",
                     "inverse_plan": [], "targets": {}},
                    {"batch_id": B1, "ts": "2026-10-01T11:00:01+00:00", "phase": "result",
                     "accepted": None, "inverse": [], "verdict": "mismatch"}):
            safety.Journal(self.state).append(row)

    def run_cli(self, *extra):
        buf = io.StringIO()
        code = new_retro.main(["--station", "LLM", "--state-dir", self.state,
                               "--out-dir", self.out, "--date", "2026-10-01", *extra], out=buf)
        return code, buf.getvalue()

    def read(self, path):
        with open(path) as fh:
            return fh.read()

    def test_writes_file_and_appends_pending_index_row_with_delta_count(self):
        self.friction()
        code, _ = self.run_cli()
        self.assertEqual(code, 0)
        name = "2026-10-01-LLM-session.md"
        text = self.read(os.path.join(self.out, name))
        self.assertIn("<!-- review-status: pending -->", text)
        self.assertIn(B1, text)
        self.assertIn("| %s | LLM | 2026-10-01 | pending | 1 |" % name, self.read(self.index))

    def test_keeps_the_index_mode_and_gives_a_new_retro_0644_under_the_umask(self):
        os.chmod(self.index, 0o664)
        old = os.umask(0o022)
        self.addCleanup(os.umask, old)
        code, _ = self.run_cli()
        self.assertEqual(code, 0)
        self.assertEqual(os.stat(self.index).st_mode & 0o777, 0o664)  # not mkstemp's 0600
        retro_file = os.path.join(self.out, "2026-10-01-LLM-session.md")
        self.assertEqual(os.stat(retro_file).st_mode & 0o777, 0o644)

    def test_the_umask_is_read_without_changing_it(self):
        old = os.umask(0o027)
        self.addCleanup(os.umask, old)
        with mock.patch.object(new_retro.os, "umask", wraps=os.umask) as spy:
            self.assertEqual(new_retro._current_umask(), 0o027)
        if os.path.exists("/proc/self/status"):  # Linux: no set-and-restore at all
            spy.assert_not_called()
        self.assertEqual(os.umask(0o027), 0o027)  # unchanged

    def test_no_friction_writes_the_honesty_line_and_zero_deltas(self):
        code, _ = self.run_cli()
        self.assertEqual(code, 0)
        text = self.read(os.path.join(self.out, "2026-10-01-LLM-session.md"))
        self.assertIn("no new deltas;", text)
        self.assertIn("| 2026-10-01-LLM-session.md | LLM | 2026-10-01 | pending | 0 |",
                      self.read(self.index))

    def test_refuses_to_overwrite_without_force_and_leaves_everything_untouched(self):
        self.run_cli()
        before = (self.read(self.index), self.read(os.path.join(
            self.out, "2026-10-01-LLM-session.md")))
        self.friction()
        code, _ = self.run_cli()
        self.assertEqual(code, 3)
        self.assertEqual(before, (self.read(self.index), self.read(os.path.join(
            self.out, "2026-10-01-LLM-session.md"))))

    def test_force_rewrites_the_file_and_updates_the_row_without_duplicating_it(self):
        self.run_cli()
        self.friction()
        code, _ = self.run_cli("--force")
        self.assertEqual(code, 0)
        index = self.read(self.index)
        self.assertEqual(index.count("2026-10-01-LLM-session.md"), 1)
        self.assertIn("| pending | 1 |", index)

    def test_dry_run_prints_and_writes_nothing(self):
        self.friction()
        before = self.read(self.index)
        code, printed = self.run_cli("--dry-run")
        self.assertEqual(code, 0)
        self.assertIn("## Proposed kit deltas", printed)
        self.assertEqual(os.listdir(self.out), ["INDEX.md"])
        self.assertEqual(before, self.read(self.index))

    def test_since_filters_and_bad_input_is_exit_3(self):
        self.friction()
        code, _ = self.run_cli("--since", "2026-10-02T00:00:00Z", "--dry-run")
        self.assertEqual(code, 0)
        code, _ = self.run_cli("--since", "nope")
        self.assertEqual(code, 3)
        code = new_retro.main(["--station", "../evil", "--state-dir", self.state,
                               "--out-dir", self.out], out=io.StringIO())
        self.assertEqual(code, 3)

    def test_since_takes_effect_on_the_written_draft(self):
        self.friction()
        _, early = self.run_cli("--since", "2026-10-01T00:00:00Z", "--dry-run")
        _, late = self.run_cli("--since", "2026-10-02T00:00:00Z", "--dry-run")
        self.assertIn(B1, early)
        self.assertNotIn(B1, late)
        self.assertIn("no new deltas;", late)

    def test_an_impossible_calendar_date_is_refused(self):
        for bad in ("2026-13-45", "2026-02-30", "0000-00-00"):
            code = new_retro.main(["--station", "LLM", "--state-dir", self.state,
                                   "--out-dir", self.out, "--date", bad], out=io.StringIO())
            self.assertEqual(code, 3, bad)
        self.assertEqual(os.listdir(self.out), ["INDEX.md"])

    def test_both_files_are_written_atomically_via_replace(self):
        self.friction()
        real = os.replace
        calls = []
        with mock.patch.object(os, "replace", side_effect=lambda a, b: (
                calls.append(os.path.basename(b)), real(a, b))[1]):
            self.assertEqual(self.run_cli()[0], 0)
        self.assertEqual(sorted(calls), ["2026-10-01-LLM-session.md", "INDEX.md"])
        self.assertEqual(sorted(os.listdir(self.out)), ["2026-10-01-LLM-session.md", "INDEX.md"])

    def test_a_failed_index_replace_leaves_no_temp_file_and_the_old_index(self):
        self.friction()
        before = self.read(self.index)
        real = os.replace

        def flaky(src, dst):
            if os.path.basename(dst) == "INDEX.md":
                raise OSError("disk full")
            real(src, dst)
        with mock.patch.object(os, "replace", side_effect=flaky):
            with self.assertRaises(OSError):
                self.run_cli()
        self.assertEqual(self.read(self.index), before)
        self.assertEqual([n for n in os.listdir(self.out) if n.endswith(".tmp") or ".tmp" in n],
                         [])

    def test_missing_index_is_exit_3(self):
        os.remove(self.index)
        code, _ = self.run_cli()
        self.assertEqual(code, 3)
        self.assertEqual(os.listdir(self.out), [])

    def test_importing_the_module_only_prepends_the_kit_to_sys_path_and_default_out_dir_is_the_kit(self):
        self.assertEqual(new_retro.DEFAULT_OUT_DIR, os.path.join(KIT, "retros"))
        self.assertIn(KIT, sys.path)


class TestRetroStepIsDocumented(unittest.TestCase):
    """The closing retro step is enforced here: the docs must name the tool and the CLI."""

    def doc(self, *parts):
        with open(os.path.join(KIT, *parts), encoding="utf-8") as fh:
            return fh.read()

    def test_methodology_and_skill_name_the_closing_step(self):
        for text in (self.doc("METHODOLOGY.md"), self.doc("skill", "SKILL.md")):
            self.assertIn("n4_session_retro_draft", text)
            self.assertIn("new_retro.py", text)
            self.assertIn("never apply", text)

    def test_methodology_no_longer_marks_the_retro_step_manual_until_tooling(self):
        self.assertNotIn("tooling arrives in T7", self.doc("METHODOLOGY.md"))

    def test_the_template_and_index_exist_with_the_agreed_grammar(self):
        self.assertIn("| file | Station | Date | pending\\|folded | deltas |",
                      self.doc("retros", "INDEX.md"))
        self.assertIn("## Proposed kit deltas", self.doc("mcp_n4", "templates", "retro.template.md"))


if __name__ == "__main__":
    unittest.main()
