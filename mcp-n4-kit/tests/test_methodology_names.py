import glob
import json
import os
import re
import unittest

KIT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GO_TESTS = os.path.join(os.path.dirname(KIT), "go", "internal", "mcpn4")


class TestMethodologyCitations(unittest.TestCase):
    def test_every_test_name_cited_in_methodology_exists(self):
        with open(os.path.join(KIT, "METHODOLOGY.md"), encoding="utf-8") as fh:
            cited = set(re.findall(r"\btest_[a-z0-9_]+\b(?!\.py)", fh.read()))
        defined = set()
        for path in glob.glob(os.path.join(KIT, "tests", "test_*.py")):
            with open(path, encoding="utf-8") as fh:
                defined |= set(re.findall(r"^\s*def (test_[a-z0-9_]+)\(", fh.read(), re.M))
        self.assertTrue(cited, "METHODOLOGY.md cites no tests: the gate would be vacuous")
        self.assertEqual(sorted(cited - defined), [])

    def test_every_go_test_cited_in_methodology_exists(self):
        """A cited `pkg.TestName` is a `func TestName(t *testing.T)` of go/internal/mcpn4/pkg; a
        cited `box.TestReplayRecordedFrames/<scenario>` is a scenario of box/testdata/replay.jsonl."""
        with open(os.path.join(KIT, "METHODOLOGY.md"), encoding="utf-8") as fh:
            cited = set(re.findall(r"`([a-z]+)\.(Test[A-Za-z0-9]+)(?:/([a-z0-9_]+))?`", fh.read()))
        with open(os.path.join(GO_TESTS, "box", "testdata", "replay.jsonl"), encoding="utf-8") as fh:
            scenarios = {json.loads(line)["name"] for line in fh}
        missing = []
        for pkg, name, sub in sorted(cited):
            replay = (pkg, name) == ("box", "TestReplayRecordedFrames")
            if sub and (not replay or sub not in scenarios):
                missing.append("%s.%s/%s" % (pkg, name, sub))
            text = ""
            for path in glob.glob(os.path.join(GO_TESTS, pkg, "*_test.go")):
                with open(path, encoding="utf-8") as fh:
                    text += fh.read()
            if not re.search(r"^func %s\(t \*testing\.T\)" % name, text, re.M):
                missing.append("%s.%s" % (pkg, name))
        self.assertGreater(len(cited), 30, "METHODOLOGY.md cites too few Go tests: a vacuous gate")
        self.assertEqual(missing, [])


if __name__ == "__main__":
    unittest.main()
