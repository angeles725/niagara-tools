import glob
import os
import re
import unittest

KIT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


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


if __name__ == "__main__":
    unittest.main()
