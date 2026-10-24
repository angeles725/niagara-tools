"""n4_list_batches on a state dir the Python server wrote (fixtures/journal/generate.py).

Black-box under both implementations (MCP_N4_IMPL=go included): the reply must be the one
tools_read.n4_list_batches computes in-process on the same files, byte for byte.
"""
import json
import os
import shutil
import sys
import tempfile
import types
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_server  # noqa: E402
from mcp_n4 import tools_read  # noqa: E402

FIXTURE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures", "journal",
                       "python-written")


class TestListBatchesOnAPythonJournal(test_server.ToolTestCase):
    def setUp(self):
        super().setUp()
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.state_dir = os.path.join(tmp.name, "state")
        shutil.copytree(FIXTURE, self.state_dir)
        self.journal = os.path.join(self.state_dir, "journal.jsonl")
        self.srv = self.make_server(allow_http=True, env=self.env, stations=self.stations,
                                    state_dir=self.state_dir)

    def expected(self, **args):
        return tools_read.n4_list_batches(types.SimpleNamespace(state_dir=self.state_dir), args)

    def append(self, data):
        with open(self.journal, "ab") as fh:
            fh.write(data)

    def test_the_python_written_batches_are_listed_as_python_lists_them(self):
        for args in ({}, {"limit": 1}, {"limit": 6}, {"limit": 1000}):
            result = self.call("n4_list_batches", **args)
            self.assertEqual(result["structuredContent"], self.expected(**args), args)
            self.assertEqual(result["content"][0]["text"], json.dumps(self.expected(**args)))
        batches = self.ok("n4_list_batches")["batches"]
        self.assertEqual([b["state"] for b in batches].count("in-doubt"), 2)
        self.assertEqual(sum(1 for b in batches if b["rolled_back_by"]), 2)

    def test_torn_and_odd_lines_are_read_as_python_reads_them(self):
        self.append(b'{torn\n[1]\r{"batch_id": 1, "phase": "intent"}\r\n'
                    b'{"batch_id": true, "phase": "result", "verdict": "v", "ts": NaN}\n')
        result = self.call("n4_list_batches")
        self.assertEqual(result["content"][0]["text"], json.dumps(self.expected()))
        self.assertEqual(result["structuredContent"]["total"], 7)

    def insert(self, after, data):
        """Put data after line `after` of the journal (1-based), between two batches."""
        with open(self.journal, "rb") as fh:
            lines = fh.read().splitlines(keepends=True)
        with open(self.journal, "wb") as fh:
            fh.write(b"".join(lines[:after]) + data + b"".join(lines[after:]))

    def assert_listed(self, unreadable):
        for args in ({}, {"limit": 1}):
            result = self.call("n4_list_batches", **args)
            self.assertEqual(result["content"][0]["text"], json.dumps(self.expected(**args)))
        reply = self.ok("n4_list_batches")
        self.assertEqual((reply["total"], reply["unreadable"]), (6, unreadable))
        self.assertIn("incomplete", reply["incomplete"])
        return reply

    def test_a_line_python_cannot_decode_is_one_unreadable_row_not_a_failed_list(self):
        self.insert(4, b'{"batch_id": "\xff", "phase": "result"}\n')  # issue #448
        self.assert_listed(["unreadable journal line 5: UnicodeDecodeError"])

    def test_an_unhashable_batch_id_is_one_unreadable_row_not_a_failed_list(self):
        self.insert(2, b'{"batch_id": [1], "phase": "intent"}\n')
        self.append(b'{"batch_id": "ee", "phase": "component-intent", "ops": 7}\n'
                    b'{"batch_id": "ef", "phase": "result", "rollback_of": {"a": 1}}\n')
        self.assert_listed(["unreadable journal line 3: TypeError",
                            "unreadable journal line 16: TypeError",
                            "unreadable journal line 17: TypeError"])

    def test_a_readable_journal_says_nothing_about_unreadable_lines(self):
        self.assertEqual(sorted(self.ok("n4_list_batches")), ["batches", "total"])

    def test_it_needs_no_session_and_no_journal_lists_nothing(self):
        os.remove(self.journal)
        self.assertEqual(self.ok("n4_list_batches"), {"total": 0, "batches": []})


if __name__ == "__main__":
    unittest.main()
