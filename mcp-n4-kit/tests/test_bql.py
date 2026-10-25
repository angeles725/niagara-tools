"""n4_bql_query / n4_inventory (retro 2026-10-02 D1, D7, D8) against a fake HTTP layer.

The CSV fixtures under tests/fixtures/ are recorded station output, anonymized.
"""
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_server  # noqa: E402  (module import: its test classes are not re-collected here)

FIXTURES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures")


def fixture(name):
    with open(os.path.join(FIXTURES, name), "rb") as fh:
        return fh.read()


class TestNonCsvAnswers(test_server.ToolTestCase):
    """An HTTP 200 that is not a CSV table is an error, never a zero-row result."""

    QUERY = "select slotPath, name from driver:Device"

    def setUp(self):
        super().setUp()
        self.connect()

    def answer(self, body):
        self.fake.ord_routes = [("driver:Device", body)]

    def test_an_html_page_is_an_error_whatever_its_content_type(self):
        page = b"<!DOCTYPE html>\n<html><body>Login</body></html>\n"
        for body in ((page, "text/html; charset=UTF-8"), (page, "text/csv"),
                     (b"Slot Path,Name\n", "text/html")):
            self.answer(body)
            self.assertIn("HTML", self.err("n4_bql_query", base="/Drivers", query=self.QUERY))

    def test_an_empty_body_is_an_error_not_zero_rows(self):
        self.answer(b"")
        text = self.err("n4_bql_query", base="/Drivers", query=self.QUERY)
        self.assertIn("no header row", text)

    def test_an_empty_body_fails_the_inventory_instead_of_zero_totals(self):
        self.fake.ord_routes = [("driver:DeviceNetwork", fixture("bql_networks.csv")),
                                ("driver:Device", b""),
                                ("control:ControlPoint", fixture("bql_points.csv"))]
        self.assertIn("no header row", self.err("n4_inventory"))

    def test_a_ragged_row_is_an_error(self):
        self.answer(b"Slot Path,Name\nslot:/A,a\nslot:/B\n")
        text = self.err("n4_bql_query", base="/Drivers", query=self.QUERY)
        self.assertIn("record 2 has 1 cell", text)


if __name__ == "__main__":
    unittest.main()
