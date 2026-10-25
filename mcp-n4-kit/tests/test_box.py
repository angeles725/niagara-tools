import importlib
import json
import os
import socket
import subprocess
import sys
import threading
import unittest
import urllib.error
from unittest import mock

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import go_server  # noqa: E402
from fake_station import FakeStation  # noqa: E402
from mcp_n4 import box  # noqa: E402

NO_SLEEP = dict(sleep=lambda s: None)


@go_server.go_twin("ord", "unit")
class TestChildOrd(unittest.TestCase):
    def test_every_parent_form_joins_with_exactly_one_slash(self):
        for parent, expected in (
                ("station:", "station:|slot:/X"),
                ("station:|slot:/", "station:|slot:/X"),
                ("station:|slot:/A", "station:|slot:/A/X"),
                ("station:|slot:/A/", "station:|slot:/A/X"),
                ("station:|slot:/A/B", "station:|slot:/A/B/X")):
            self.assertEqual(box.child_ord(parent, "X"), expected, parent)
            self.assertNotIn("//", expected)

    def test_bad_names_and_bad_parents_are_refused(self):
        for bad in ("", "a/b", "a|b", "..", None, 5):
            with self.assertRaises(ValueError, msg=repr(bad)):
                box.child_ord("station:|slot:/", bad)
        with self.assertRaises(ValueError):
            box.child_ord("station:|slot://A", "X")  # an already-broken parent

    def test_join_ord_walks_a_relative_path_and_allows_the_empty_path(self):
        self.assertEqual(box.join_ord("station:|slot:/", "A/B"), "station:|slot:/A/B")
        self.assertEqual(box.join_ord("station:|slot:/A/", "B"), "station:|slot:/A/B")
        self.assertEqual(box.join_ord("station:|slot:/A/", ""), "station:|slot:/A")
        self.assertEqual(box.join_ord("station:|slot:/", ""), "station:|slot:/")
        with self.assertRaises(ValueError):
            box.join_ord("station:|slot:/", "A//B")


class BoxTestCase(unittest.TestCase):
    def setUp(self):
        self.fake = FakeStation().start()
        self.addCleanup(self.fake.stop)

    def client(self, **kw):
        kw.setdefault("allow_http", True)
        return box.BoxClient(self.fake.url, "admin", kw.pop("password", "secret"), **kw)

    def opened(self):
        c = self.client()
        c.open()
        self.addCleanup(c.close)
        return c

    def add(self, c, name, type_="kitControl:NumericConst", **kw):
        nn = c.add_component("3", name, type_, **kw)["nn"]  # server-assigned name
        return c.load_tree("station:|slot:/Folder", depth=2, **NO_SLEEP)[nn]["h"]


def hook_for(key, code=500, body=b"", headers=None):
    """FakeStation hook that fails every frame whose first message has key `key`."""
    def hook(frame):
        if frame["m"][0]["k"] == key:
            return code, body, headers or {}
    return hook


def reply_hook(builder):
    """FakeStation hook answering every frame with HTTP 200 and builder(frame) as JSON."""
    def hook(frame):
        return 200, json.dumps(builder(frame)).encode(), {"Content-Type": "application/json"}
    return hook


def error_frame(frame, **kw):
    return dict({"v": "2.3", "p": "box", "n": frame["n"]}, **kw)


class RaisingOpener:
    def __init__(self, exc):
        self.exc = exc

    def open(self, req, timeout=None):
        raise self.exc


class TestSecurity(BoxTestCase):
    @go_server.go_twin("box", "injection")
    def test_the_load_window_follows_the_client_load_wait(self):
        # F11: a longer --load-wait means more polls and a longer total sleep.
        slept = []
        client = self.client(load_wait=10.0)
        self.addCleanup(client.close)
        client.open()
        orig = self.fake._ssc

        def mute_loads(key, arg):
            if key == "loadSlots":
                return None  # never answered
            return orig(key, arg)
        patcher = mock.patch.object(self.fake, "_ssc", mute_loads)
        patcher.start()
        self.addCleanup(patcher.stop)
        with self.assertRaises(box.BoxError):
            client.load_tree("station:|slot:/Folder", sleep=slept.append)
        self.assertAlmostEqual(sum(slept), 10.0, places=3)

    @go_server.go_twin("box", "unit")
    def test_auth_failure_raises_once_without_retry(self):
        c = self.client(password="wrong")
        with self.assertRaises(box.AuthError):
            c.open()
        self.assertEqual(self.fake.requests, 1)

    @go_server.go_twin("box", "unit", twin="box.TestHTTPRefusedWithoutAllowHTTP")
    def test_http_refused_without_allow_http(self):
        with self.assertRaises(ValueError):
            box.BoxClient("http://127.0.0.1:1", "u", "p")

    @go_server.go_twin("box", "unit", twin="box.TestHTTPSAcceptedWithoutFlag")
    def test_https_accepted_without_flag(self):
        box.BoxClient("https://station.example", "u", "p")

    @go_server.go_twin("box", "unit")
    def test_repr_hides_password(self):
        c = self.client(password="s3cr3t-pw")
        self.assertNotIn("s3cr3t-pw", repr(c))
        self.assertNotIn("s3cr3t-pw", str(c))

    @go_server.go_twin("box", "injection", python_only="a Python import side effect; a Go "
                       "package has no import-time code path that could reach the network")
    def test_import_makes_no_network_call(self):
        with mock.patch.object(socket.socket, "connect", side_effect=AssertionError("network")):
            importlib.reload(box)

    @go_server.go_twin("box", "unit", python_only="a Python import side effect; a Go binary "
                       "has no import (its start: cli.TestEofEndsTheServerWithExitZero)")
    def test_import_has_no_side_effects_in_fresh_process(self):
        kit = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        r = subprocess.run([sys.executable, "-c", "import mcp_n4.box"], cwd=kit,
                           capture_output=True, text=True)
        self.assertEqual((r.returncode, r.stdout, r.stderr), (0, "", ""))


@go_server.go_twin("box", "unit")
class TestRedirects(BoxTestCase):
    def test_redirect_is_refused_and_credentials_never_reach_target(self):
        target = FakeStation().start()
        self.addCleanup(target.stop)
        for code in (301, 302, 303, 307, 308):
            self.fake.hook = lambda frame, code=code: (
                code, b"", {"Location": target.url + "/box/"})
            c = self.client()
            with self.assertRaises(box.BoxError) as cm:
                c.open()
            self.assertIn(str(code), str(cm.exception))
            self.assertIn("127.0.0.1", str(cm.exception))
            self.assertNotIn("secret", str(cm.exception))
            self.assertNotIn("YWRtaW46", str(cm.exception))
        self.assertEqual(target.requests, 0)


class TestTransportErrors(BoxTestCase):
    @go_server.go_twin("box", "unit")
    def test_http_401_names_the_auth_scheme_the_password_and_no_retry(self):
        c = self.client(password="wrong")
        with self.assertRaises(box.AuthError) as cm:
            c.open()
        text = str(cm.exception)
        for needle in ("HTTP 401", "HTTPBasicScheme", "password", "do not retry"):
            self.assertIn(needle, text)
        self.assertNotIn("wrong", text)  # never echoes the password
        self.assertEqual(self.fake.requests, 1)

    @go_server.go_twin("box", "unit", twin="box.TestAuthMessageNamesWhatToCheck")
    def test_http_403_says_permission_and_no_retry(self):
        self.fake.hook = hook_for("make", 403)
        with self.assertRaises(box.AuthError) as cm:
            self.client().open()
        self.assertIn("HTTP 403", str(cm.exception))
        self.assertIn("do not retry", str(cm.exception))

    @go_server.go_twin("box", "unit")
    def test_http_403_is_auth_error(self):
        self.fake.hook = hook_for("make", 403)
        with self.assertRaises(box.AuthError):
            self.client().open()
        self.assertEqual(self.fake.requests, 1)

    @go_server.go_twin("box", "unit")
    def test_http_500_is_boxerror_not_autherror(self):
        self.fake.hook = hook_for("make", 500)
        with self.assertRaises(box.BoxError) as cm:
            self.client().open()
        self.assertNotIsInstance(cm.exception, box.AuthError)

    @go_server.go_twin("box", "injection", twin="box.TestTransportFailuresAreBoxErrors")
    def test_timeout_and_refused_and_oserror_are_boxerror(self):
        for exc in (TimeoutError("timed out"),
                    urllib.error.URLError(ConnectionRefusedError("refused")),
                    OSError("network down")):
            c = self.client(opener=RaisingOpener(exc))
            with self.assertRaises(box.BoxError):
                c.open()

    @go_server.go_twin("box", "unit", twin="box.TestTransportFailuresAreBoxErrors")
    def test_connection_refused_for_real(self):
        c = box.BoxClient("http://127.0.0.1:1", "u", "p", allow_http=True, timeout=2)
        with self.assertRaises(box.BoxError):
            c.open()

    @go_server.go_twin("box", "unit")
    def test_invalid_json_body_is_boxerror(self):
        self.fake.hook = lambda frame: (200, b"<html>not json", {})
        with self.assertRaises(box.BoxError):
            self.client().open()

    @go_server.go_twin("box", "unit")
    def test_an_oversized_box_reply_is_refused_with_the_cap_named(self):
        # call() reads at most max_reply_bytes (32 MB by default, like get_ord): a reply
        # past the cap is refused, never parsed from a truncated body.
        self.assertEqual(box.BoxClient.max_reply_bytes, 32 * 1024 * 1024)
        c = self.client()
        c.max_reply_bytes = 1024
        self.fake.hook = lambda frame: (200, b'{"pad": "' + b"x" * 4096 + b'"}', {})
        with self.assertRaises(box.BoxError) as cm:
            c.open()
        self.assertNotIsInstance(cm.exception, box.AuthError)
        self.assertIn("1024", str(cm.exception))
        self.assertIn("larger than", str(cm.exception))

    @go_server.go_twin("box", "unit")
    def test_malformed_reply_shapes_are_boxerror(self):
        shapes = {
            "missing m": lambda f: error_frame(f),
            "empty m": lambda f: error_frame(f, m=[]),
            "m not a list": lambda f: error_frame(f, m="x"),
            "non-dict message": lambda f: error_frame(f, m=["x"]),
            "reply not a dict": lambda f: [1, 2],
            "error with non-dict b": lambda f: error_frame(f, m=[{"t": "e", "b": "boom"}]),
            "error with null b": lambda f: error_frame(f, m=[{"t": "e", "b": None}]),
        }
        for label, builder in shapes.items():
            self.fake.hook = reply_hook(builder)
            c = self.client()
            with self.subTest(label), self.assertRaises(box.BoxError):
                c.open()

    @go_server.go_twin("box", "unit")
    def test_error_frame_message_is_propagated(self):
        self.fake.hook = reply_hook(
            lambda f: error_frame(f, m=[{"t": "e", "b": {"m": "boom"}}]))
        with self.assertRaises(box.BoxError) as cm:
            self.client().open()
        self.assertEqual(cm.exception.message, "boom")

    @go_server.go_twin("box", "unit")
    def test_reply_seq_mismatch_is_boxerror(self):
        self.fake.hook = reply_hook(
            lambda f: error_frame(f, n=f["n"] + 41, m=[{"t": "rp", "b": "sess-1"}]))
        with self.assertRaises(box.BoxError) as cm:
            self.client().open()
        self.assertIn("seq", str(cm.exception))

    @go_server.go_twin("box", "unit")
    def test_reply_without_n_is_accepted(self):
        self.fake.hook = reply_hook(
            lambda f: {"v": "2.3", "p": "box", "m": [{"t": "rp", "b": "sess-1"}]})
        c = self.client()
        c.sid = None
        self.assertEqual(c.call("ssession", "make", {}), "sess-1")

    @go_server.go_twin("box", "injection")
    def test_close_swallows_any_exception(self):
        c = self.opened()
        with mock.patch.object(c, "call", side_effect=RuntimeError("password=secret")):
            c.close()
        self.assertIsNone(c.sid)


class RawStation:
    """Loopback TCP server answering every request with the same raw bytes, then closing.

    It stands in for a station (or a proxy) whose answer is not well-formed HTTP; the
    FakeStation always writes a correct status line and Content-Length.
    """

    def __init__(self, answer):
        self.answer = answer
        self._sock = socket.create_server(("127.0.0.1", 0))
        self.url = "http://127.0.0.1:%d" % self._sock.getsockname()[1]
        threading.Thread(target=self._serve, daemon=True).start()

    def _serve(self):
        while True:
            try:
                conn, _ = self._sock.accept()
            except OSError:  # closed by stop()
                return
            with conn:
                conn.settimeout(5)
                try:
                    self._answer(conn)
                except OSError:  # the client gave up: nothing left to prove on this socket
                    pass

    def _answer(self, conn):
        """Read the whole request, answer, then end with a FIN, never a RST.

        Closing a socket that still holds unread request bytes makes Linux send RST, and the
        client then sees "Connection reset by peer" instead of the short body. So the head
        and its Content-Length body are read first, the write side is shut down (FIN after
        the answer), and the socket is drained until the client closes.
        """
        data = b""
        while b"\r\n\r\n" not in data:
            chunk = conn.recv(65536)
            if not chunk:
                return
            data += chunk
        head, _, body = data.partition(b"\r\n\r\n")
        length = 0
        for line in head.split(b"\r\n")[1:]:
            name, _, value = line.partition(b":")
            if name.strip().lower() == b"content-length":
                length = int(value.strip())
        while len(body) < length:
            chunk = conn.recv(65536)
            if not chunk:
                return
            body += chunk
        conn.sendall(self.answer)
        conn.shutdown(socket.SHUT_WR)
        while conn.recv(65536):
            pass

    def stop(self):
        self._sock.close()


#: Every client read: (label, callable(client)). Each one must turn a bad answer into BoxError.
CLIENT_READS = (
    ("call", lambda c: c.call("ssession", "make", {})),
    ("get_ord", lambda c: c.get_ord("station:|slot:/")),
    ("get_file", lambda c: c.get_file("/file/a.txt")),
    ("about", lambda c: c.about()),
)


@go_server.go_twin("box", "unit")
class TestTransportTruth(unittest.TestCase):
    """An answer that is not complete HTTP is a transport failure, never data (0.9.2)."""

    def reads_against(self, answer):
        """(label, raised exception or the returned value) of every client read."""
        station = RawStation(answer)
        self.addCleanup(station.stop)
        client = box.BoxClient(station.url, "u", "p", allow_http=True, timeout=5)
        outcomes = []
        for label, read in CLIENT_READS:
            try:
                outcomes.append((label, read(client)))
            except Exception as exc:  # noqa: BLE001 - the type is what is asserted
                outcomes.append((label, exc))
        return outcomes

    def test_a_non_http_answer_is_a_transport_failure(self):
        for label, exc in self.reads_against(b"NOT HTTP AT ALL\r\n\r\n"):
            self.assertIs(type(exc), box.BoxError, label)
            self.assertEqual(str(exc), "transport failure: invalid HTTP answer (BadStatusLine)",
                             label)

    def test_a_body_shorter_than_its_content_length_is_never_returned(self):
        answer = (b"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 100\r\n"
                  b"\r\n" + b'{"m": [{"b": 1}]}')
        for label, exc in self.reads_against(answer):
            self.assertIs(type(exc), box.BoxError, label)
            self.assertEqual(str(exc), "transport failure: incomplete read (got 17 of 100 "
                             "bytes)", label)

    def test_a_cut_chunked_body_is_a_transport_failure(self):
        answer = b"HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nab"
        for label, exc in self.reads_against(answer):
            self.assertIs(type(exc), box.BoxError, label)
            self.assertEqual(str(exc), "transport failure: invalid HTTP answer (IncompleteRead)",
                             label)

    def test_a_body_longer_than_the_cap_is_still_only_truncated(self):
        # The Content-Length check compares with what was asked for: a capped read of a long,
        # complete body is the documented truncation, not an incomplete read.
        body = b"a,b\n" * 64
        answer = (b"HTTP/1.1 200 OK\r\nContent-Type: text/csv\r\nContent-Length: %d\r\n\r\n"
                  % len(body)) + body
        station = RawStation(answer)
        self.addCleanup(station.stop)
        client = box.BoxClient(station.url, "u", "p", allow_http=True, timeout=5)
        self.assertEqual(client.get_ord("station:|slot:/", max_bytes=10),
                         (("a,b\n" * 3)[:10], True))
        self.assertEqual(client.about(max_bytes=8), "a,b\na,b\n")


def poll_hook(events_json):
    """FakeStation hook answering every pollchgs with the raw JSON `events_json` as its body."""
    def hook(frame):
        if frame["m"][0]["k"] == "pollchgs":
            body = '{"v": "2.3", "p": "box", "n": %d, "m": [{"t": "rp", "b": %s}]}' % (
                frame["n"], events_json)
            return 200, body.encode(), {"Content-Type": "application/json"}
    return hook


def load_event(tree):
    """A pollchgs body holding one load op whose slot tree is `tree` (a JSON-ready value)."""
    return json.dumps([{"evs": {"ops": [{"nm": "l", "h": "3", "b": tree}]}}])


class TestReplyShapes(BoxTestCase):
    """A reply of the wrong shape is a 'malformed BOX reply' BoxError, never data (0.9.2)."""

    def assert_malformed(self, action, detail):
        with self.assertRaises(box.BoxError) as cm:
            action()
        self.assertIs(type(cm.exception), box.BoxError)
        self.assertEqual(str(cm.exception), "malformed BOX reply: " + detail)

    @go_server.go_twin("box", "unit")
    def test_pollchgs_must_answer_a_list_of_events(self):
        c = self.opened()
        for body, kind in (('{"a": 1}', "an object"), ('"ab"', "a string"), ("5", "a number"),
                           ("true", "a boolean"), ("{}", "an object"), ('""', "a string")):
            self.fake.hook = poll_hook(body)
            with self.subTest(body):
                self.assert_malformed(c.poll, "pollchgs answered %s, not a list" % kind)
        for body in ("null", "[]"):
            self.fake.hook = poll_hook(body)
            self.assertEqual(c.poll(), [], body)

    @go_server.go_twin("box", "unit", twin="box.TestFlattenRefusesMalformedTrees")
    def test_a_load_tree_of_the_wrong_shape_is_refused(self):
        c = self.opened()
        cases = (
            ({"s": None}, "slot (root) has children that are null, not a list"),
            ({"s": "ab"}, "slot (root) has children that are a string, not a list"),
            ({"s": {"n": "a"}}, "slot (root) has children that are an object, not a list"),
            ({"s": ["x"]}, "slot (root) has a child that is a string, not an object"),
            ({"s": [{"t": "x"}]}, "slot (root) has a child without a slot name"),
            ({"s": [{"n": 5}]}, "slot (root) has a child without a slot name"),
            ({"s": [{"n": ""}]}, "slot (root) has a child without a slot name"),
            ({"s": [{"n": "a", "s": [{"n": 5}]}]}, "slot a has a child without a slot name"),
            ({"s": [{"n": "a", "s": [{"n": "b", "s": 1}]}]},
             "slot a/b has children that are a number, not a list"),
        )
        for tree, detail in cases:
            self.fake.hook = poll_hook(load_event(tree))
            with self.subTest(tree):
                self.assert_malformed(
                    lambda: c.load_tree("station:|slot:/Folder", attempts=2, **NO_SLEEP), detail)

    @go_server.go_twin("box", "unit")
    def test_an_early_event_whose_ops_are_no_list_is_kept_not_walked(self):
        # The poll before a loadSlots settles the load ops it sees; ops that are no list are
        # kept with their event (as _split does), never iterated as keys or characters.
        c = self.opened()
        for ops in (5, "ab", {"nm": "l"}):
            self.fake.hook = poll_hook(json.dumps([{"evs": {"ops": ops}}]))
            with self.subTest(ops):
                with self.assertRaises(box.BoxError) as cm:
                    c.load_tree("station:|slot:/Folder", attempts=2, **NO_SLEEP)
                self.assertIn("no load event", str(cm.exception))
                self.assertIn({"evs": {"ops": ops}}, c.drain_pending_events())

    @go_server.go_twin("box", "unit", twin="box.TestFlattenRefusesMalformedTrees")
    def test_a_well_formed_tree_still_loads(self):
        c = self.opened()
        self.fake.hook = poll_hook(load_event({"h": "3", "s": [{"n": "a", "s": [{"n": "b"}]}]}))
        nodes = c.load_tree("station:|slot:/Folder", attempts=2, **NO_SLEEP)
        self.assertEqual(sorted(nodes), ["", "a", "a/b"])

    @go_server.go_twin("box", "unit")
    def test_a_reply_nested_past_the_limit_is_refused(self):
        self.assertEqual(box.MAX_REPLY_NESTING, 1000)
        c = self.opened()
        # The frame wraps the body in 3 levels ({"m": [{"b": ...}]}): 997 more fit the limit.
        self.fake.hook = poll_hook("[" * 997 + "]" * 997)
        self.assertEqual(len(c.poll()), 1)
        for depth in (998, 200000):  # 200000 is past CPython's C stack: RecursionError
            self.fake.hook = poll_hook("[" * depth + "]" * depth)
            with self.subTest(depth):
                self.assert_malformed(c.poll, "nested deeper than 1000 levels")


class TestSession(BoxTestCase):
    @go_server.go_twin("box", "unit")
    def test_open_returns_root(self):
        c = self.client()
        root = c.open()
        self.assertEqual(root["h"], "2")
        self.assertEqual(root["t"], "baja:Station")

    @go_server.go_twin("box", "unit")
    def test_context_manager_deletes_session(self):
        with self.client() as c:
            self.assertEqual(self.fake.sessions, {"sess-1"})
        self.assertEqual(self.fake.sessions, set())

    @go_server.go_twin("box", "unit")
    def test_open_closes_session_when_makessc_fails(self):
        self.fake.hook = hook_for("makessc", 500)
        c = self.client()
        with self.assertRaises(box.BoxError):
            c.open()
        self.assertEqual(self.fake.sessions, set())
        self.assertIsNone(c.sid)

    @go_server.go_twin("box", "unit")
    def test_open_closes_session_when_loadroot_fails(self):
        def hook(frame):
            m = frame["m"][0]
            if m["k"] == "callssc" and m["b"]["sck"] == "loadRoot":
                return 200, json.dumps(error_frame(
                    frame, m=[{"t": "e", "b": {"m": "no root"}}])).encode(), {}
        self.fake.hook = hook
        with self.assertRaises(box.BoxError):
            self.client().open()
        self.assertEqual(self.fake.sessions, set())

    @go_server.go_twin("box", "unit", twin="box.TestNonAuthFailureAfterMakeClosesTheSession")
    def test_with_block_does_not_leak_on_open_failure(self):
        self.fake.hook = hook_for("makessc", 500)
        with self.assertRaises(box.BoxError):
            with self.client():
                pass
        self.assertEqual(self.fake.sessions, set())

    @go_server.go_twin("box", "unit")
    def test_constants(self):
        self.assertEqual(box.SESSION_COMPONENT_ID, "cs1")
        self.assertEqual(box.DEFAULT_ROOT_HANDLE, "2")

    @go_server.go_twin("box", "unit")
    def test_station_error_frame_raises_boxerror(self):
        c = self.opened()
        with self.assertRaises(box.BoxError):
            c.ssc("noSuchKey", None)


class TestReadWrite(BoxTestCase):
    @go_server.go_twin("box", "unit")
    def test_add_component_visible_with_handle_and_annotation(self):
        c = self.opened()
        res = c.add_component("3", "Temp", "kitControl:NumericConst", ws=box.ws_annotation(10, 20))
        self.assertEqual(set(res), {"id", "nn"})
        self.assertEqual(res["nn"], "Temp")
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertTrue(nodes["Temp"]["h"])
        self.assertEqual(nodes["Temp/wsAnnotation"]["v"], "10,20,8")

    @go_server.go_twin("box", "unit",
                       twin="writeops.TestTheResultRecordsTheServerAssignedNameNotThePlannedOne")
    def test_add_component_name_collision_yields_distinct_nn(self):
        c = self.opened()
        first = c.add_component("3", "Dup", "kitControl:NumericConst")
        second = c.add_component("3", "Dup", "kitControl:NumericConst")
        self.assertEqual(first["nn"], "Dup")
        self.assertNotEqual(second["nn"], first["nn"])
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertIn(first["nn"], nodes)
        self.assertIn(second["nn"], nodes)

    @go_server.go_twin("box", "injection", python_only="add_component's reply check; no tool "
                       "calls it, and create reads nn leniently (_create_component_inverse)")
    def test_add_component_malformed_reply_is_boxerror(self):
        c = self.opened()
        for bad in (None, [], ["x"], "x"):
            with mock.patch.object(c, "sync", return_value=bad), \
                    self.assertRaises(box.BoxError):
                c.add_component("3", "X", "kitControl:NumericConst")

    @go_server.go_twin("box", "unit")
    def test_load_tree_ignores_stale_load_for_other_ord(self):
        c = self.opened()
        c.ssc("loadSlots", {"o": "station:|slot:/Folder", "d": 1})  # stale, handle 3
        nodes = c.load_tree("station:", **NO_SLEEP)
        self.assertEqual(nodes[""]["h"], "2")
        self.assertEqual(nodes[""]["t"], "baja:Station")
        stale = [op for ev in c.pending_events for op in ev["evs"]["ops"]]
        self.assertEqual([op["h"] for op in stale], ["3"])

    @go_server.go_twin("box", "unit")
    def test_load_tree_unknown_handle_ignores_stale_event(self):
        c = self.client()
        c.open()
        self.addCleanup(c.close)
        c.ssc("loadSlots", {"o": "station:", "d": 1})  # stale root load
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertEqual(nodes[""]["h"], "3")

    @go_server.go_twin("box", "injection")
    def test_two_load_ops_are_never_merged(self):
        c = self.opened()
        root_op = {"nm": "l", "h": "2", "b": {"nm": "p", "t": "baja:Station", "h": "2",
                                                "s": [{"n": "RootKid", "t": "baja:Folder"}]}}
        folder_op = {"nm": "l", "h": "3", "b": {"nm": "p", "t": "baja:Folder", "h": "3",
                                                  "s": [{"n": "FolderKid", "t": "baja:Folder"}]}}
        event = {"evs": {"nm": "sync", "ops": [root_op, folder_op]}}
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[[], [event]]):
            nodes = c.load_tree("station:|slot:/Folder", handle="3", **NO_SLEEP)
        self.assertIn("FolderKid", nodes)
        self.assertNotIn("RootKid", nodes)
        kept = [op for ev in c.pending_events for op in ev["evs"]["ops"]]
        self.assertEqual(kept, [root_op])

    @go_server.go_twin("box", "unit")
    def test_pending_events_is_read_only_view(self):
        c = self.opened()
        self.assertEqual(len(c.pending_events), 0)
        with self.assertRaises((TypeError, AttributeError)):
            c.pending_events.append({})
        with self.assertRaises(AttributeError):
            c.pending_events = []

    @go_server.go_twin("box", "injection")
    def test_non_load_events_are_kept(self):
        c = self.opened()
        other = {"evs": {"nm": "sync", "ops": [{"nm": "s", "h": "9", "n": "x"}]}}
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[[other], [{"evs": {"ops": [
                    {"nm": "l", "h": "2", "b": {"nm": "p", "t": "baja:Station"}}]}}]]):
            nodes = c.load_tree("station:", **NO_SLEEP)
        self.assertEqual(nodes[""]["t"], "baja:Station")
        self.assertEqual(list(c.pending_events), [other])

    @go_server.go_twin("box", "injection")
    def test_load_tree_retries_n_empty_polls_then_succeeds(self):
        c = self.opened()
        good = {"evs": {"ops": [{"nm": "l", "h": "2",
                                 "b": {"nm": "p", "t": "baja:Station", "h": "2"}}]}}
        sleeps = []
        # first poll is the pre-request drain, then 3 empty polls, then the event
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[[], [], [], [], [good]]):
            nodes = c.load_tree("station:", attempts=6, delay=0.25, sleep=sleeps.append)
        self.assertEqual(nodes[""]["h"], "2")
        self.assertEqual(sleeps, [0.25, 0.5, 0.8])  # backoff from `delay`, capped at 0.8 s

    @go_server.go_twin("box", "injection")
    def test_load_tree_polls_with_a_short_first_delay_and_backoff(self):
        c = self.opened()
        good = {"evs": {"ops": [{"nm": "l", "h": "2",
                                 "b": {"nm": "p", "t": "baja:Station", "h": "2"}}]}}
        sleeps = []
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[[], [], [], [], [], [good]]):
            c.load_tree("station:", sleep=sleeps.append)
        self.assertEqual(sleeps, [0.1, 0.2, 0.4, 0.8])

    @go_server.go_twin("box", "injection")
    def test_load_tree_total_wait_is_bounded_and_each_delay_capped(self):
        c = self.opened()
        sleeps = []
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:", attempts=50, sleep=sleeps.append, handle="2")
        self.assertLessEqual(max(sleeps), box.MAX_POLL_DELAY)
        self.assertAlmostEqual(sum(sleeps), box.MAX_POLL_WAIT)
        self.assertEqual(sleeps[:4], [0.1, 0.2, 0.4, 0.8])

    @go_server.go_twin("box", "unit")
    def test_poll_delays_are_a_pure_bounded_schedule(self):
        self.assertEqual(list(box.poll_delays(0.1, 0.8, 3.0, 20)),
                         [0.1, 0.2, 0.4, 0.8, 0.8, 0.7])
        self.assertEqual(list(box.poll_delays(0.1, 0.8, 3.0, 3)), [0.1, 0.2])
        self.assertEqual(list(box.poll_delays(0.1, 0.8, 3.0, 1)), [])

    @go_server.go_twin("box", "unit")
    def test_load_tree_root_key_is_empty_path(self):
        c = self.opened()
        nodes = c.load_tree("station:", **NO_SLEEP)
        self.assertEqual(nodes[""]["h"], "2")
        self.assertIn("Folder", nodes)

    @go_server.go_twin("box", "injection")
    def test_load_tree_gives_up_when_no_event(self):
        c = self.opened()
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:", attempts=3, **NO_SLEEP)

    @go_server.go_twin("box", "unit")
    def test_sync_rejects_multiple_ops(self):
        c = self.opened()
        with self.assertRaises(ValueError):
            c.sync([{"nm": "v", "h": "3", "n": "a"}, {"nm": "v", "h": "3", "n": "b"}])

    @go_server.go_twin("box", "unit", twin="writeops.TestPlanAndReadbackDifferentialPython")
    def test_set_slot_rejects_partial_status_by_default(self):
        c = self.opened()
        h = self.add(c, "Temp")
        for path in ("fallback/value", "out/status"):
            with self.assertRaises(ValueError):
                c.set_slot(h, path, box.bson_double(30.0))
        c.set_slot(h, "out", box.bson_status_numeric(1.0))
        c.set_slot(h, "out/value", box.bson_double(30.0), allow_partial_status=True)

    @go_server.go_twin("box", "unit")
    def test_set_slot_whole_status_numeric_reads_back_ok(self):
        c = self.opened()
        h = self.add(c, "Temp")
        c.set_slot(h, "out", box.bson_status_numeric(30.0))
        nodes = c.load_tree("station:|slot:/Folder", depth=3, **NO_SLEEP)
        got = box.status_value(nodes, "Temp/out")
        self.assertEqual(got["value"], "30.0")
        self.assertTrue(box.parse_status(got["status"])["ok"])

    @go_server.go_twin("box", "unit")
    def test_check_links_returns_link_name_and_creates_link(self):
        c = self.opened()
        a, b = self.add(c, "A"), self.add(c, "B")
        res = c.check_links(a, "out", b, "in10")
        self.assertIsInstance(res, list)
        self.assertEqual(len(res), 1)
        self.assertEqual(set(res[0]), {"v", "r", "s"})
        self.assertEqual(res[0]["s"], "Link")
        self.assertTrue(res[0]["v"])
        nodes = c.load_tree("station:|slot:/Folder/B", depth=2, **NO_SLEEP)
        self.assertEqual(nodes["Link"]["t"], "baja:Link")
        self.assertEqual(nodes["Link/sourceSlotName"]["v"], "out")
        self.assertEqual(nodes["Link/targetSlotName"]["v"], "in10")

    @go_server.go_twin("box", "unit")
    def test_invoke_action_set_writes_fallback_whole(self):
        c = self.opened()
        h = self.add(c, "W", "control:NumericWritable")
        c.invoke_action(h, "set", box.bson_double(7.5))
        nodes = c.load_tree("station:|slot:/Folder", depth=3, **NO_SLEEP)
        got = box.status_value(nodes, "W/fallback")
        self.assertEqual(got["value"], "7.5")
        self.assertEqual(got["status"], "0")

    @go_server.go_twin("box", "unit")
    def test_save_station_increments_counter(self):
        c = self.opened()
        c.save_station()
        self.assertEqual(self.fake.saves, 1)

    @go_server.go_twin("box", "unit")
    def test_remove_component(self):
        c = self.opened()
        self.add(c, "Gone")
        c.remove_component("3", "Gone")
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertNotIn("Gone", nodes)


class TestLockoutSafeOpen(BoxTestCase):
    def deletes(self):
        return [f for f in self.seen if f["m"][0]["k"] == "del"]

    def track(self, key, code):
        self.seen = []

        def hook(frame):
            self.seen.append(frame)
            m = frame["m"][0]
            if m["k"] == key or (key == "loadRoot" and m["k"] == "callssc"
                                 and m["b"]["sck"] == "loadRoot"):
                return code, b"", {}
        self.fake.hook = hook

    @go_server.go_twin("box", "unit")
    def test_auth_failure_in_makessc_sends_no_cleanup_del(self):
        self.track("makessc", 401)
        c = self.client()
        with self.assertRaises(box.AuthError):
            c.open()
        self.assertEqual(self.deletes(), [])
        self.assertIsNone(c.sid)

    @go_server.go_twin("box", "unit")
    def test_auth_failure_in_loadroot_sends_no_cleanup_del(self):
        self.track("loadRoot", 403)
        c = self.client()
        with self.assertRaises(box.AuthError):
            c.open()
        self.assertEqual(self.deletes(), [])
        self.assertIsNone(c.sid)

    @go_server.go_twin("box", "unit", twin="box.TestNonAuthFailureAfterMakeClosesTheSession")
    def test_non_auth_failure_still_cleans_up(self):
        self.track("makessc", 500)
        with self.assertRaises(box.BoxError):
            self.client().open()
        self.assertEqual(len(self.deletes()), 1)

    @go_server.go_twin("box", "injection")
    def test_interrupt_after_make_still_closes_the_server_session(self):
        for exc in (KeyboardInterrupt, SystemExit):
            with self.subTest(exc=exc.__name__):
                self.fake.sessions.clear()
                c = self.client()
                real = c.call

                def fake_call(channel, key, body, real=real, exc=exc):
                    if key == "makessc":
                        raise exc
                    return real(channel, key, body)
                with mock.patch.object(c, "call", side_effect=fake_call):
                    with self.assertRaises(exc):
                        c.open()
                self.assertEqual(self.fake.sessions, set())
                self.assertIsNone(c.sid)


class TestHandleCache(BoxTestCase):
    @go_server.go_twin("box", "unit", twin="plan.TestARemovalClearsEveryCachedHandle")
    def test_remove_component_invalidates_child_and_descendants(self):
        c = self.opened()
        self.add(c, "Gone")
        c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        c._handles["station:|slot:/Folder/Gone"] = "x1"
        c._handles["station:|slot:/Folder/Gone/Kid"] = "x2"
        c._handles["station:|slot:/Folder/GoneNot"] = "x3"
        c.remove_component("3", "Gone")
        self.assertNotIn("station:|slot:/Folder/Gone", c._handles)
        self.assertNotIn("station:|slot:/Folder/Gone/Kid", c._handles)
        self.assertIn("station:|slot:/Folder/GoneNot", c._handles)
        self.assertIn("station:|slot:/Folder", c._handles)

    @go_server.go_twin("box", "injection", twin="plan.TestARemovalClearsEveryCachedHandle")
    def test_remove_component_with_unknown_parent_clears_cache(self):
        c = self.opened()
        self.add(c, "Gone")
        c._handles["station:|slot:/Other/Gone"] = "x1"
        c.remove_component("3", "Gone")  # parent "3" learned, child ord derived
        c._handles["station:|slot:/Other"] = "x9"
        with mock.patch.object(c, "sync", return_value=[]):
            c.remove_component("unknown-handle", "Gone")
        self.assertEqual(c._handles, {})

    @go_server.go_twin("box", "unit")
    def test_invalidate_handles_prefix_and_all(self):
        c = self.opened()
        c._handles.update({"station:|slot:/A": "1", "station:|slot:/A/B": "2",
                           "station:|slot:/C": "3"})
        c.invalidate_handles("station:|slot:/A")
        self.assertEqual(sorted(c._handles), ["station:", "station:|slot:/C"])
        c.invalidate_handles()
        self.assertEqual(c._handles, {})

    @go_server.go_twin("box", "unit")
    def test_invalidate_handles_is_boundary_aware(self):
        c = self.opened()
        c._handles.update({"station:|slot:/A": "1", "station:|slot:/A/B": "2",
                           "station:|slot:/AB": "3", "station:|slot:/AB/C": "4"})
        c.invalidate_handles("station:|slot:/A")
        self.assertEqual(sorted(c._handles),
                         ["station:", "station:|slot:/AB", "station:|slot:/AB/C"])
        c.invalidate_handles("station:|slot:/AB/")  # a trailing slash is the same boundary
        self.assertEqual(sorted(c._handles), ["station:"])

    @go_server.go_twin("box", "unit")
    def test_the_station_model_rejects_an_add_nesting_a_baja_folder(self):
        c = self.opened()
        nested = {"nm": "p", "t": "baja:Folder", "s": [{"nm": "p", "n": "Sub", "t": "baja:Folder"}]}
        with self.assertRaises(box.BoxError):
            c.sync({"nm": "a", "h": "3", "n": "Grp", "b": nested})

    @go_server.go_twin("box", "injection")
    def test_a_stale_cached_handle_does_not_wait_a_second_polling_window(self):
        c = self.opened()
        c._handles["station:|slot:/Folder"] = "dead"
        sleeps = []
        nodes = c.load_tree("station:|slot:/Folder", sleep=sleeps.append)
        self.assertEqual(nodes[""]["h"], "3")
        self.assertEqual(sleeps, [])  # a load op with another handle proves it is stale

    @go_server.go_twin("box", "unit")
    def test_split_returns_the_other_load_flag_instead_of_keeping_it_on_the_client(self):
        c = self.opened()
        other = {"evs": {"ops": [{"nm": "l", "h": "9", "b": {"nm": "p", "t": "baja:Folder"}}]}}
        found, saw_other = c._split([other], "3")
        self.assertEqual((found, saw_other), (None, True))
        self.assertFalse(hasattr(c, "_saw_other_load"))

    @go_server.go_twin("box", "injection")
    def test_an_unanswered_earlier_load_disables_the_stale_handle_early_abort(self):
        c = self.opened()
        c._handles["station:|slot:/Folder"] = "3"
        late = {"evs": {"ops": [{"nm": "l", "h": "7", "b": {"nm": "p", "t": "baja:Folder"}}]}}
        mine = {"evs": {"ops": [{"nm": "l", "h": "3", "b": {"nm": "p", "t": "baja:Folder",
                                                            "h": "3"}}]}}
        with mock.patch.object(c, "ssc", return_value=None):
            with mock.patch.object(c, "poll", return_value=[]):
                with self.assertRaises(box.BoxError):  # an earlier request times out
                    c.load_tree("station:|slot:/Other", attempts=1, **NO_SLEEP)
            polls = iter([[], [late], [mine]])
            sleeps = []
            with mock.patch.object(c, "poll", side_effect=lambda: next(polls, [])):
                nodes = c.load_tree("station:|slot:/Folder", attempts=3,
                                    sleep=sleeps.append)
        # the late reply of the earlier request is not proof the cache is stale
        self.assertEqual(nodes[""]["h"], "3")
        self.assertEqual(len(sleeps), 1)  # it kept polling instead of giving up at once

    @go_server.go_twin("box", "injection")
    def test_an_answered_load_clears_the_count_left_by_a_timed_out_one(self):
        c = self.opened()
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):  # no reply: the request stays counted
                c.load_tree("station:|slot:/Other", attempts=1, **NO_SLEEP)
        self.assertEqual(c._loads_outstanding, 1)
        c.load_tree("station:|slot:/Folder", **NO_SLEEP)  # its own reply arrives
        # replies come in request order: the earlier request will not be answered now
        self.assertEqual(c._loads_outstanding, 0)

    @go_server.go_twin("box", "injection")
    def test_a_cached_handle_regains_the_early_abort_after_a_timed_out_load(self):
        c = self.opened()
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:|slot:/Other", attempts=1, **NO_SLEEP)
        c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        c._handles["station:|slot:/Folder"] = "dead"
        sleeps = []
        nodes = c.load_tree("station:|slot:/Folder", sleep=sleeps.append)
        self.assertEqual(nodes[""]["h"], "3")
        self.assertEqual(sleeps, [])  # the session is trusted again

    @go_server.go_twin("box", "injection")
    def test_a_rejected_load_request_is_not_left_outstanding(self):
        c = self.opened()
        with mock.patch.object(c, "ssc", side_effect=box.BoxError("rejected")):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertEqual(c._loads_outstanding, 0)

    @go_server.go_twin("box", "unit")
    def test_stale_cached_handle_is_retried_once_as_unknown(self):
        c = self.opened()
        c._handles["station:|slot:/Folder"] = "dead"
        nodes = c.load_tree("station:|slot:/Folder", **NO_SLEEP)
        self.assertEqual(nodes[""]["h"], "3")
        self.assertEqual(c._handles["station:|slot:/Folder"], "3")

    @go_server.go_twin("box", "unit")
    def test_explicit_handle_is_not_retried_as_unknown(self):
        c = self.opened()
        with self.assertRaises(box.BoxError):
            c.load_tree("station:|slot:/Folder", handle="dead", attempts=2, **NO_SLEEP)

    @go_server.go_twin("box", "injection")
    def test_stale_cached_handle_failure_still_gives_up(self):
        c = self.opened()
        c._handles["station:|slot:/Folder"] = "dead"
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", return_value=[]):
            with self.assertRaises(box.BoxError):
                c.load_tree("station:|slot:/Folder", attempts=2, **NO_SLEEP)


class TestPendingEvents(BoxTestCase):
    @go_server.go_twin("box", "injection")
    def test_drain_returns_and_clears(self):
        c = self.opened()
        other = {"evs": {"ops": [{"nm": "s", "h": "9"}]}}
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[[other], [{"evs": {"ops": [
                    {"nm": "l", "h": "2", "b": {"nm": "p", "t": "baja:Station"}}]}}]]):
            c.load_tree("station:", **NO_SLEEP)
        drained = c.drain_pending_events()
        self.assertEqual(drained, [other])
        self.assertEqual(c.pending_events, ())
        self.assertEqual(c.drain_pending_events(), [])

    @go_server.go_twin("box", "injection")
    def test_pending_events_are_capped_oldest_dropped(self):
        c = self.opened()
        total = box.MAX_PENDING_EVENTS + 7
        events = [{"evs": {"ops": [{"nm": "s", "h": str(i)}]}} for i in range(total)]
        with mock.patch.object(c, "ssc", return_value=None), \
                mock.patch.object(c, "poll", side_effect=[events, [{"evs": {"ops": [
                    {"nm": "l", "h": "2", "b": {"nm": "p", "t": "baja:Station"}}]}}]]):
            c.load_tree("station:", **NO_SLEEP)
        self.assertEqual(len(c.pending_events), box.MAX_PENDING_EVENTS)
        self.assertEqual(c.dropped_events, 7)
        self.assertEqual(c.pending_events[0]["evs"]["ops"][0]["h"], "7")

    @go_server.go_twin("box", "unit")
    def test_dropped_events_starts_at_zero(self):
        self.assertEqual(self.client().dropped_events, 0)


class TestErrorLabelsAndRedirectClose(BoxTestCase):
    @go_server.go_twin("box", "injection", python_only="the labels of add_component and "
                       "check_links, BoxClient helpers no server tool calls and Go does not have")
    def test_malformed_reply_keys_use_session_component_key(self):
        c = self.opened()
        with mock.patch.object(c, "sync", return_value=None), \
                self.assertRaises(box.BoxError) as cm:
            c.add_component("3", "X", "kitControl:NumericConst")
        self.assertEqual(cm.exception.key, "syncTo")
        with mock.patch.object(c, "ssc", return_value=None), \
                self.assertRaises(box.BoxError) as cm:
            c.check_links("a", "b", "c", "d")
        self.assertEqual(cm.exception.key, "checkLinks")

    @go_server.go_twin("box", "injection", python_only="urllib's redirect handler; Go's client "
                       "refuses redirects itself and closes every response body by defer")
    def test_redirect_handler_closes_response_before_raising(self):
        fp = mock.Mock()
        with self.assertRaises(box.BoxError):
            box._NoRedirect().redirect_request(
                mock.Mock(), fp, 302, "Found", {}, "https://evil.example/box/")
        fp.close.assert_called_once_with()


class TestPureHelpers(unittest.TestCase):
    @go_server.go_twin("box", "unit", twin="box.TestChildrenAndStatusValue")
    def test_status_value_applies_type_defaults(self):
        nodes = {
            "sb": {"t": "baja:StatusBoolean"},
            "sn": {"t": "baja:StatusNumeric"},
        }
        self.assertEqual(box.status_value(nodes, "sb"), {"value": "false", "status": "0"})
        self.assertEqual(box.status_value(nodes, "sn"), {"value": "0.0", "status": "0"})

    @go_server.go_twin("box", "unit", twin="box.TestChildrenAndStatusValue")
    def test_status_value_reads_present_children(self):
        nodes = {
            "sb": {"t": "baja:StatusBoolean"},
            "sb/value": {"t": "baja:Boolean", "v": "true"},
            "sb/status": {"t": "baja:Status", "v": "40"},
        }
        self.assertEqual(box.status_value(nodes, "sb"), {"value": "true", "status": "40"})

    @go_server.go_twin("box", "unit")
    def test_is_component_type_is_the_shared_station_add_rule(self):
        for type_ in ("kitControl:NumericConst", "control:NumericWritable", "baja:Folder"):
            self.assertTrue(box.is_component_type(type_), type_)
        for type_ in ("baja:Double", "baja:StatusNumeric", "baja:WsAnnotation", "baja:Link",
                      "", None):
            self.assertFalse(box.is_component_type(type_), type_)

    @go_server.go_twin("box", "unit",
                       twin="box.TestIsComponentTypeIsTheSharedStationAddRule")
    def test_driver_address_structs_are_values_not_components(self):
        # F17 (audit 2026-10-03, issue #200): BFlexAddress / BBacnetAddress extend BStruct;
        # the naming heuristic (non-baja -> component) misclassified them.
        for type_ in ("modbusCore:FlexAddress", "bacnet:BacnetAddress"):
            self.assertIs(box.COMPONENT_TYPES[type_], False, type_)
            self.assertFalse(box.is_component_type(type_), type_)

    @go_server.go_twin("box", "unit",
                       twin="box.TestIsComponentTypeIsTheSharedStationAddRule")
    def test_the_type_table_wins_over_the_naming_heuristic(self):
        """B1200-G3: an explicit entry decides; unknown types fall back to the heuristic."""
        self.assertIs(box.COMPONENT_TYPES["control:PriorityLevel"], False)  # BFrozenEnum
        self.assertFalse(box.is_component_type("control:PriorityLevel"))  # heuristic: True
        self.assertIs(box.COMPONENT_TYPES["baja:UnrestrictedFolder"], True)  # extends BFolder
        self.assertTrue(box.is_component_type("baja:UnrestrictedFolder"))  # heuristic: False
        self.assertTrue(box.is_component_type("baja:Component"))
        self.assertTrue(box.is_component_type("acme:UnknownWidget"))  # fallback
        self.assertFalse(box.is_component_type("baja:RelTime"))  # fallback

    @go_server.go_twin("box", "unit", twin="box.TestChildrenAndStatusValue")
    def test_children_lists_direct_children_only(self):
        nodes = {"": {}, "a": {}, "a/b": {}, "c": {}}
        self.assertEqual(sorted(box.children(nodes, "")), ["a", "c"])
        self.assertEqual(sorted(box.children(nodes, "a")), ["b"])

    @go_server.go_twin("box", "unit", twin="box.TestParseStatusDifferentialPython")
    def test_parse_status(self):
        s = box.parse_status("40;activeLevel=e:17@control:PriorityLevel")
        self.assertEqual(s["bits"], 0x40)
        self.assertTrue(s["null"])
        self.assertFalse(s["ok"])
        self.assertEqual(s["facets"], "activeLevel=e:17@control:PriorityLevel")
        ok = box.parse_status("0")
        self.assertTrue(ok["ok"])
        self.assertFalse(ok["null"])
        self.assertEqual(ok["facets"], "")

    @go_server.go_twin("box", "unit", twin="writeops.TestPlanAndReadbackDifferentialPython")
    def test_encoders(self):
        self.assertEqual(box.bson_double(30), {"nm": "p", "t": "baja:Double", "v": "30.0"})
        self.assertEqual(box.bson_bool(True)["v"], "true")
        sb = box.bson_status_boolean(False)
        self.assertEqual(sb["s"][0]["v"], "false")
        self.assertEqual(sb["s"][1], {"nm": "p", "n": "status", "t": "baja:Status", "v": "0"})
        self.assertEqual(box.ws_annotation(1, 2, 3)["v"], "1,2,3")


if __name__ == "__main__":
    unittest.main()
