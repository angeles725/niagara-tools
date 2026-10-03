"""Tests for tools/mutation_check.py: clears __pycache__ and runs without writing bytecode."""
import io
import os
import sys
import tempfile
import unittest
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
KIT = os.path.dirname(HERE)
sys.path.insert(0, os.path.join(KIT, "tools"))

import mutation_check  # noqa: E402


class TestMutationCheck(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = os.path.join(tmp.name, "kit")
        for sub in ("__pycache__", os.path.join("pkg", "__pycache__"), "pkg"):
            os.makedirs(os.path.join(self.root, sub), exist_ok=True)
        for pyc in ("__pycache__/a.cpython-314.pyc", "pkg/__pycache__/b.cpython-314.pyc"):
            with open(os.path.join(self.root, pyc), "wb") as fh:
                fh.write(b"stale")
        with open(os.path.join(self.root, "pkg", "keep.py"), "w") as fh:
            fh.write("x = 1\n")
        self.out = os.path.join(tmp.name, "env.txt")
        stderr = mock.patch("sys.stderr", new_callable=io.StringIO)
        stderr.start()
        self.addCleanup(stderr.stop)

    def _caches(self):
        return [d for d, _, _ in os.walk(self.root) if os.path.basename(d) == "__pycache__"]

    def test_every_pycache_under_the_root_is_removed_and_sources_are_kept(self):
        self.assertEqual(mutation_check.clear_pycache(self.root), 2)
        self.assertEqual(self._caches(), [])
        self.assertTrue(os.path.isfile(os.path.join(self.root, "pkg", "keep.py")))

    def test_the_command_runs_with_bytecode_writing_disabled_after_the_clear(self):
        script = ("import os, sys; open(sys.argv[1], 'w').write("
                  "os.environ.get('PYTHONDONTWRITEBYTECODE', '') + ' ' + "
                  "str(os.path.isdir(sys.argv[2])))")
        cache = os.path.join(self.root, "__pycache__")
        with mock.patch.dict(os.environ, {"PYTHONDONTWRITEBYTECODE": ""}):
            code = mutation_check.main(["--root", self.root, "--",
                                        sys.executable, "-c", script, self.out, cache])
        self.assertEqual(code, 0)
        with open(self.out) as fh:
            self.assertEqual(fh.read(), "1 False")

    def test_the_command_exit_code_is_passed_through(self):
        code = mutation_check.main(["--root", self.root, "--",
                                    sys.executable, "-c", "raise SystemExit(3)"])
        self.assertEqual(code, 3)

    def test_a_missing_command_is_a_usage_error_and_clears_nothing(self):
        with self.assertRaises(SystemExit) as ctx:
            mutation_check.main(["--root", self.root, "--"])
        self.assertEqual(ctx.exception.code, 2)
        self.assertEqual(len(self._caches()), 2)


if __name__ == "__main__":
    unittest.main()
