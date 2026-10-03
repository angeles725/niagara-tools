"""Audit 2026-10-03 F4: an authentication failure pauses every station call.

The station locks the account after 5 failed logins in 30 s (B1179). One rejected call
must not be followed by more rejected calls: any `AuthError` (connect, oBIX about, the
`/ord` GET, a BOX call) latches the server until a cooldown passes or the operator
reconnects with other credentials.
"""
import contextlib
import io
import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import test_server  # noqa: E402
from mcp_n4 import box, server  # noqa: E402

FOLDER = "station:|slot:/Folder"


class Clock:
    def __init__(self):
        self.now = 1000.0

    def __call__(self):
        return self.now


class LatchCase(test_server.ToolTestCase):
    def setUp(self):
        super().setUp()
        self.clock = Clock()
        self.srv.ctx.clock = self.clock

    def wrong_password(self):
        self.srv.ctx.env = dict(self.env, MCP_N4_PASSWORD="wrong-pw-1234")

    def right_password(self):
        self.srv.ctx.env = dict(self.env)

    def requests(self):
        return self.fake.requests + self.fake.about_requests


class TestConnectLatch(LatchCase):
    def test_a_rejected_login_pauses_the_next_connect_with_the_same_credentials(self):
        self.wrong_password()
        self.err("n4_connect", station="FakeStation")
        before = self.requests()
        text = self.err("n4_connect", station="FakeStation")
        self.assertEqual(self.requests(), before)  # nothing reached the station
        self.assertIn("authentication failure", text)
        self.assertIn("HTTP 401", text)
        self.assertIn("30 s", text)
        self.assertNotIn("wrong-pw-1234", text)

    def test_after_the_cooldown_one_more_attempt_is_allowed(self):
        self.wrong_password()
        self.err("n4_connect", station="FakeStation")
        self.clock.now += 29
        self.assertIn("retry in 1 s", self.err("n4_connect", station="FakeStation"))
        self.clock.now += 1
        self.right_password()
        self.assertEqual(self.connect()["station_name"], "FakeStation")
        self.assertNotIn("auth_paused", self.ok("n4_describe_session"))

    def test_other_credentials_may_reconnect_during_the_cooldown(self):
        self.wrong_password()
        self.err("n4_connect", station="FakeStation")
        self.right_password()
        self.assertEqual(self.connect()["station_name"], "FakeStation")
        self.ok("n4_navigate", ord="station:|slot:/")  # a good login clears the latch

    def test_the_cooldown_is_configurable_but_never_below_30_seconds(self):
        self.srv = server.Server(allow_http=True, env=self.env, stations=self.stations,
                                 auth_cooldown=120)
        self.addCleanup(self.srv.ctx.close)
        self.srv.ctx.clock = self.clock
        self.wrong_password()
        self.err("n4_connect", station="FakeStation")
        self.clock.now += 60
        self.assertIn("retry in 60 s", self.err("n4_connect", station="FakeStation"))
        for bad in ("29", "0", "-1"):
            err = io.StringIO()
            with contextlib.redirect_stderr(err), self.assertRaises(SystemExit):
                server.parse_args(["--auth-cooldown", bad])
            self.assertIn("--auth-cooldown", err.getvalue())
        self.assertEqual(server.parse_args([]).auth_cooldown, 30)
        with self.assertRaises(ValueError):
            server.Server(auth_cooldown=10)


class HooklessClient(box.BoxClient):
    """A client that cannot take the `on_auth_error` hook (the server must still latch)."""

    @property
    def on_auth_error(self):
        return None

    @on_auth_error.setter
    def on_auth_error(self, value):
        raise AttributeError("this client has no auth hook")


class TestHooklessClientLatch(LatchCase):
    """H2 review R4/R2/R3 fallback-latch: the `Server._call` fallback must record every
    failure, also while an earlier latch (for other credentials) is held."""

    def setUp(self):
        super().setUp()
        self.srv.ctx.close()
        self.srv = server.Server(allow_http=True, env=self.env, stations=self.stations,
                                 client_factory=HooklessClient)
        self.addCleanup(self.srv.ctx.close)
        self.srv.ctx.clock = self.clock

    def test_a_second_rejected_credential_is_latched_by_the_fallback(self):
        self.wrong_password()
        self.err("n4_connect", station="FakeStation")
        self.srv.ctx.env = dict(self.env, MCP_N4_PASSWORD="another-wrong-pw")
        self.assertIn("HTTP 401", self.err("n4_connect", station="FakeStation"))
        before = self.requests()
        self.assertIn("authentication failure", self.err("n4_connect", station="FakeStation"))
        self.assertEqual(self.requests(), before)


class TestSessionLatch(LatchCase):
    def setUp(self):
        super().setUp()
        self.connect()

    def test_a_rejected_box_call_pauses_every_later_station_read(self):
        self.fake.password = "rotated"  # the station now rejects this session's login
        first = self.err("n4_navigate", ord="station:|slot:/")
        self.assertIn("HTTP 401", first)
        before = self.requests()
        for tool, args in (("n4_navigate", {"ord": "station:|slot:/"}),
                           ("n4_read_slots", {"ord": FOLDER}),
                           ("n4_bql_query", {"query": "select name from control:NumericWritable"})):
            text = self.err(tool, **args)
            self.assertIn("authentication failure", text, tool)
            self.assertIn("retry in 30 s", text, tool)
        self.assertEqual(self.requests(), before)
        paused = self.ok("n4_describe_session")["auth_paused"]
        self.assertEqual(paused["retry_in_s"], 30)
        self.assertIn("HTTP 401", paused["reason"])

    def test_a_rejected_ord_get_pauses_too(self):
        self.fake.password = "rotated"
        self.assertIn("HTTP 401", self.err(
            "n4_bql_query", query="select name from control:NumericWritable"))
        self.assertIn("authentication failure", self.err("n4_navigate"))

    def test_the_reads_resume_after_the_cooldown(self):
        self.fake.password = "rotated"
        self.err("n4_navigate", ord="station:|slot:/")
        self.fake.password = self.PASSWORD
        self.clock.now += 30
        self.ok("n4_navigate", ord="station:|slot:/")


class TestAboutLatch(LatchCase):
    def test_an_about_rejection_during_connect_pauses_later_station_calls(self):
        def about_401(client, *a, **kw):
            raise client._auth_error(401, "obix", "about")
        patcher = mock.patch.object(box.BoxClient, "about", about_401)  # README isolation rule
        patcher.start()
        self.addCleanup(patcher.stop)
        out = self.connect()  # the version probe never fails the connect
        self.assertIn("HTTP 401", out["version_error"])
        self.assertIn("auth_paused", out)
        self.assertIn("authentication failure", self.err("n4_navigate"))


class TestWriteLatch(LatchCase):
    def setUp(self):
        super().setUp()
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.srv.ctx.close()
        self.srv = server.Server(allow_writes=True, allow_http=True, env=self.env,
                                 stations=self.stations, write_scopes=[FOLDER],
                                 state_dir=os.path.join(tmp.name, "state"))
        self.addCleanup(self.srv.ctx.close)
        self.srv.ctx.clock = self.clock
        self.connect()

    def test_a_write_dry_run_is_refused_while_paused(self):
        self.fake.password = "rotated"
        self.err("n4_navigate", ord="station:|slot:/")
        before = self.requests()
        res = self.srv.dispatch(test_server.rpc("tools/call", {
            "name": "n4_create_component",
            "arguments": {"parent_ord": FOLDER, "name": "A", "type": "kitControl:NumericConst"}}))
        self.assertTrue(res["result"]["isError"], res)
        text = res["result"]["content"][0]["text"]
        self.assertIn("authentication failure", text)
        self.assertEqual(self.requests(), before)


if __name__ == "__main__":
    unittest.main()
