"""BOX client differential for the Go port (docs/go-port-plan.md, slice S3).

The Go `box` package (go/internal/mcpn4/box) is proven against the Python `BoxClient` two ways,
both driven by the same small op language (`run_ops`):

- **Recorded frames.** `SCENARIOS` port the read-side `test_box` cases. Each runs a Python
  `BoxClient` against a recording `FakeStation` (hooks script faults and poll replies) and
  stores the raw BOX frames and the op results in `go/internal/mcpn4/box/testdata/replay.jsonl`.
  The Go test replays the frames: every request body Go sends must equal the recorded one, and
  every op result must equal Python's, byte for byte. Regenerate with
  `python3 mcp-n4-kit/tests/go_box.py --write`; `test_go_harness` fails while it is stale.
- **Live.** `--live` reads `{"client": {...}, "ops": [...]}` on stdin, runs the ops against
  the client's URL (a Go test server) and prints one result line per op, so the Go test can
  compare HTTP-level behaviour (headers, redirects, caps, GET reads, transport failures).

Op results are `{"ok": value}` or `{"error": {...}}`: a BoxError family member or ValueError
with its type, message, channel, key and code; any other exception with its type only (the
server shows only the type). A `load_tree` result also carries the requested `sleeps`.
"""
import json
import os
import sys

TESTS_DIR = os.path.dirname(os.path.abspath(__file__))
KIT_DIR = os.path.dirname(TESTS_DIR)
REPLAY_FILE = os.path.join(os.path.dirname(KIT_DIR), "go", "internal", "mcpn4", "box", "testdata",
                           "replay.jsonl")
for path in (KIT_DIR, TESTS_DIR):
    if path not in sys.path:
        sys.path.insert(0, path)

from mcp_n4 import box  # noqa: E402


def describe(exc):
    """The comparable part of an exception (see the module docstring)."""
    if isinstance(exc, box.BoxError):
        return {"type": type(exc).__name__, "message": str(exc), "channel": exc.channel,
                "key": exc.key, "code": getattr(exc, "code", None)}
    if isinstance(exc, ValueError):  # the server shows str() of any ValueError
        return {"type": "ValueError", "message": str(exc)}
    return {"type": type(exc).__name__}


def make_client(cfg, url):
    client = box.BoxClient(url, cfg.get("username", "admin"), cfg.get("password", "secret"),
                           timeout=cfg.get("timeout", 20), insecure_tls=cfg.get("insecure_tls", False),
                           allow_http=cfg.get("allow_http", True),
                           load_wait=cfg.get("load_wait", box.MAX_POLL_WAIT))
    if "max_reply_bytes" in cfg:
        client.max_reply_bytes = cfg["max_reply_bytes"]
    return client


def _kw(op, *names):
    return {n: op[n] for n in names if n in op}


def run_ops(client, ops):
    """Run `ops` on `client`; one JSON-able result per op."""
    auth_errors = []
    client.on_auth_error = lambda exc: auth_errors.append(str(exc))
    out = []
    for op in ops:
        kind, sleeps = op["op"], []
        try:
            if kind == "open":
                res = client.open()
            elif kind == "close":
                res = client.close()
            elif kind == "call":
                res = client.call(op["channel"], op["key"], op.get("body"))
            elif kind == "ssc":
                res = client.ssc(op["key"], op.get("arg"))
            elif kind == "sync":
                res = client.sync(op["sync"])
            elif kind == "poll":
                res = list(client.poll())
            elif kind == "load_tree":
                res = client.load_tree(op["ord"], sleep=sleeps.append,
                                       **_kw(op, "depth", "attempts", "delay", "handle"))
            elif kind == "get_ord":
                res = list(client.get_ord(op["ord"], **_kw(op, "timeout", "max_bytes")))
            elif kind == "get_file":
                res = client.get_file(op["path"], **_kw(op, "timeout", "max_bytes"))
                res = dict(res, body=res["body"].hex())
            elif kind == "about":
                res = client.about(**_kw(op, "timeout", "max_bytes"))
            elif kind == "drain":
                res = client.drain_pending_events()
            elif kind == "invalidate":
                res = client.invalidate_handles(op.get("prefix"))
            elif kind == "set_handle":
                res = client._handles.__setitem__(op["ord"], op["handle"])
            elif kind == "state":
                res = {"sid": client.sid, "handles": client._handles,
                       "pending": list(client.pending_events), "dropped": client.dropped_events,
                       "outstanding": client._loads_outstanding, "auth_errors": auth_errors}
            else:
                raise SystemExit("unknown op %r" % kind)
            entry = {"ok": res}
        except Exception as exc:  # the result under test
            entry = {"error": describe(exc)}
        if kind == "load_tree":
            entry["sleeps"] = sleeps
        out.append(json.loads(json.dumps(entry)))  # a snapshot: later ops mutate nothing here
    return out


# ---- recorded scenarios --------------------------------------------------------------------

def _answer(frame, body, kind="rp"):
    out = {"v": "2.3", "p": "box", "n": frame["n"], "m": [{"r": 0, "t": kind, "b": body}]}
    return 200, json.dumps(out).encode(), {"Content-Type": "application/json"}


def _key(frame):
    m = frame["m"][0]
    return m["b"]["sck"] if m["k"] == "callssc" else m["k"]


class Script:
    """A FakeStation hook: `fail` maps a key (or callssc sck) to a status or `"e"` (an error
    frame); the first `muted` loadSlots get an empty answer (no load event); `pollchgs` is
    answered from `polls` while it lasts; `raw` answers every frame with these bytes."""

    def __init__(self, fail=None, muted=0, polls=(), raw=None):
        self.fail, self.muted, self.polls, self.raw = fail or {}, muted, list(polls), raw

    def __call__(self, frame):
        key = _key(frame)
        if self.raw is not None:
            return 200, self.raw(frame) if callable(self.raw) else self.raw, {}
        if key in self.fail:
            code = self.fail[key]
            return _answer(frame, {"m": "rejected"}, "e") if code == "e" else (code, b"", {})
        if key == "loadSlots" and self.muted:
            self.muted -= 1
            return _answer(frame, None)
        if key == "pollchgs" and self.polls:
            return _answer(frame, self.polls.pop(0))
        return None


def _load(handle, **body):
    """A poll event carrying one load op for `handle`; `body` is the loaded node."""
    return {"evs": {"nm": "sync", "ops": [{"nm": "l", "h": handle, "b": dict({"nm": "p"}, **body)}]}}


def _frame(**kw):
    return lambda f: json.dumps(dict({"v": "2.3", "p": "box", "n": f["n"]}, **kw)).encode()


OPEN, STATE = {"op": "open"}, {"op": "state"}
FOLDER = "station:|slot:/Folder"
OTHER = {"evs": {"nm": "sync", "ops": [{"nm": "s", "h": "9", "n": "x"}]}}
ROOT_OP = {"nm": "l", "h": "2", "b": {"nm": "p", "t": "baja:Station", "h": "2",
                                      "s": [{"n": "RootKid", "t": "baja:Folder"}]}}
FOLDER_OP = {"nm": "l", "h": "3", "b": {"nm": "p", "t": "baja:Folder", "h": "3",
                                        "s": [{"n": "FolderKid", "t": "baja:Folder"}]}}


def _nested(depth):
    """A poll body of `depth` nested lists: the reply frame adds 3 levels around it."""
    out = []
    for _ in range(depth - 1):
        out = [out]
    return out


def _sync(nm, h="3", **body):
    return {"op": "sync", "sync": dict({"nm": nm, "h": h}, **body)}


def _lt(ord_str=FOLDER, **kw):
    return dict({"op": "load_tree", "ord": ord_str}, **kw)


NUMERIC = {"nm": "p", "t": "kitControl:NumericConst", "s": []}


def _shape(name, kw):
    return dict(name="malformed_reply_" + name, ports="test_malformed_reply_shapes_are_boxerror",
                hook=Script(raw=_frame(**kw)), ops=[OPEN, STATE])


#: name, the test_box test it ports, client config, hook, ops.
SCENARIOS = [
    dict(name="open_returns_root", ops=[OPEN, STATE]),
    dict(name="context_manager_deletes_session", ops=[OPEN, {"op": "close"}, STATE]),
    dict(name="auth_failure_raises_once_without_retry", client={"password": "wrong"},
         ports="test_http_401_names_the_auth_scheme_the_password_and_no_retry", ops=[OPEN, STATE]),
    dict(name="http_403_is_auth_error", hook=Script(fail={"make": 403}), ops=[OPEN, STATE]),
    dict(name="http_500_is_boxerror_not_autherror", hook=Script(fail={"make": 500}), ops=[OPEN]),
    dict(name="open_closes_session_when_makessc_fails", hook=Script(fail={"makessc": 500}),
         ops=[OPEN, STATE]),
    dict(name="open_closes_session_when_loadroot_fails", hook=Script(fail={"loadRoot": "e"}),
         ops=[OPEN, STATE]),
    dict(name="auth_failure_in_makessc_sends_no_cleanup_del", hook=Script(fail={"makessc": 401}),
         ops=[OPEN, STATE]),
    dict(name="auth_failure_in_loadroot_sends_no_cleanup_del", hook=Script(fail={"loadRoot": 403}),
         ops=[OPEN, STATE]),
    dict(name="invalid_json_body_is_boxerror", hook=Script(raw=b"<html>not json"), ops=[OPEN]),
    dict(name="an_oversized_box_reply_is_refused_with_the_cap_named", client={"max_reply_bytes": 1024},
         hook=Script(raw=b'{"pad": "' + b"x" * 4096 + b'"}'), ops=[OPEN]),
    _shape("missing_m", {}), _shape("empty_m", {"m": []}), _shape("m_not_a_list", {"m": "x"}),
    _shape("non_dict_message", {"m": ["x"]}),
    dict(name="malformed_reply_not_a_dict", ports="test_malformed_reply_shapes_are_boxerror",
         hook=Script(raw=b"[1, 2]"), ops=[OPEN]),
    _shape("error_with_non_dict_b", {"m": [{"t": "e", "b": "boom"}]}),
    _shape("error_with_null_b", {"m": [{"t": "e", "b": None}]}),
    dict(name="error_frame_message_is_propagated", ops=[OPEN],
         hook=Script(raw=_frame(m=[{"t": "e", "b": {"m": "boom"}}]))),
    dict(name="reply_seq_mismatch_is_boxerror", ops=[OPEN],
         hook=Script(raw=lambda f: json.dumps({"n": f["n"] + 41, "m": [{"t": "rp", "b": "s"}]}).encode())),
    dict(name="reply_without_n_is_accepted", hook=Script(raw=b'{"m": [{"t": "rp", "b": "sess-1"}]}'),
         ops=[{"op": "call", "channel": "ssession", "key": "make", "body": {}}]),
    dict(name="station_error_frame_raises_boxerror", ops=[OPEN, {"op": "ssc", "key": "noSuchKey"}]),
    dict(name="load_tree_root_key_is_empty_path", ops=[OPEN, _lt("station:"), STATE]),
    dict(name="load_tree_ignores_stale_load_for_other_ord", ops=[
        OPEN, {"op": "ssc", "key": "loadSlots", "arg": {"o": FOLDER, "d": 1}}, _lt("station:"), STATE]),
    dict(name="load_tree_unknown_handle_ignores_stale_event", ops=[
        OPEN, {"op": "ssc", "key": "loadSlots", "arg": {"o": "station:", "d": 1}}, _lt(), STATE]),
    dict(name="stale_cached_handle_is_retried_once_as_unknown", ops=[
        OPEN, {"op": "set_handle", "ord": FOLDER, "handle": "dead"}, _lt(), STATE]),
    dict(name="a_stale_cached_handle_does_not_wait_a_second_polling_window", ops=[
        OPEN, {"op": "set_handle", "ord": FOLDER, "handle": "dead"}, _lt(), STATE]),
    dict(name="explicit_handle_is_not_retried_as_unknown", ops=[
        OPEN, _lt(handle="dead", attempts=2), STATE]),
    dict(name="the_load_window_follows_the_client_load_wait", client={"load_wait": 10.0},
         hook=Script(muted=99), ops=[OPEN, _lt(), STATE]),
    dict(name="two_load_ops_are_never_merged",
         hook=Script(muted=1, polls=[[], [{"evs": {"nm": "sync", "ops": [ROOT_OP, FOLDER_OP]}}]]),
         ops=[OPEN, _lt(handle="3"), STATE]),
    dict(name="non_load_events_are_kept", hook=Script(muted=1, polls=[[OTHER], [_load("2", t="baja:Station")]]),
         ops=[OPEN, _lt("station:"), STATE]),
    dict(name="load_tree_retries_n_empty_polls_then_succeeds",
         hook=Script(muted=1, polls=[[], [], [], [], [_load("2", t="baja:Station", h="2")]]),
         ops=[OPEN, _lt("station:", attempts=6, delay=0.25)]),
    dict(name="load_tree_polls_with_a_short_first_delay_and_backoff",
         hook=Script(muted=1, polls=[[], [], [], [], [], [_load("2", t="baja:Station", h="2")]]),
         ops=[OPEN, _lt("station:")]),
    dict(name="load_tree_total_wait_is_bounded_and_each_delay_capped", hook=Script(muted=1),
         ops=[OPEN, _lt("station:", attempts=50, handle="2")]),
    dict(name="load_tree_gives_up_when_no_event", hook=Script(muted=1), ops=[OPEN, _lt(attempts=3)]),
    dict(name="an_unanswered_earlier_load_disables_the_stale_handle_early_abort",
         hook=Script(muted=2, polls=[[], [], [], [_load("7", t="baja:Folder")],
                                     [_load("3", t="baja:Folder", h="3")]]),
         ops=[OPEN, {"op": "set_handle", "ord": FOLDER, "handle": "3"},
              _lt("station:|slot:/Other", attempts=1), STATE, _lt(attempts=3), STATE]),
    dict(name="an_answered_load_clears_the_count_left_by_a_timed_out_one",
         hook=Script(muted=1, polls=[[], []]),
         ops=[OPEN, _lt("station:|slot:/Other", attempts=1), STATE, _lt(), STATE]),
    dict(name="a_cached_handle_regains_the_early_abort_after_a_timed_out_load",
         hook=Script(muted=1, polls=[[], []]),
         ops=[OPEN, _lt("station:|slot:/Other", attempts=1), _lt(),
              {"op": "set_handle", "ord": FOLDER, "handle": "dead"}, _lt(), STATE]),
    dict(name="a_rejected_load_request_is_not_left_outstanding", hook=Script(fail={"loadSlots": "e"}),
         ops=[OPEN, _lt(), STATE]),
    dict(name="stale_cached_handle_failure_still_gives_up", hook=Script(muted=2),
         ops=[OPEN, {"op": "set_handle", "ord": FOLDER, "handle": "dead"}, _lt(attempts=2), STATE]),
    dict(name="drain_returns_and_clears", hook=Script(muted=1, polls=[[OTHER], [_load("2", t="baja:Station")]]),
         ops=[OPEN, _lt("station:"), {"op": "drain"}, STATE, {"op": "drain"}]),
    dict(name="pending_events_are_capped_oldest_dropped", hook=Script(muted=1, polls=[
        [{"evs": {"ops": [{"nm": "s", "h": str(i)}]}} for i in range(box.MAX_PENDING_EVENTS + 7)],
        [_load("2", t="baja:Station")]]), ops=[OPEN, _lt("station:"), STATE]),
    dict(name="invalidate_handles_prefix_and_all", ops=[
        OPEN, {"op": "set_handle", "ord": "station:|slot:/A", "handle": "1"},
        {"op": "set_handle", "ord": "station:|slot:/A/B", "handle": "2"},
        {"op": "set_handle", "ord": "station:|slot:/AB", "handle": "3"},
        {"op": "invalidate", "prefix": "station:|slot:/A/"}, STATE, {"op": "invalidate"}, STATE]),
    # A malformed poll or load reply is a "malformed BOX reply" BoxError (0.9.2), and an
    # early event whose ops are no list is kept, never walked.
    dict(name="a_poll_answering_an_object_is_malformed", ports=None,
         hook=Script(muted=1, polls=[{"k1": 1, "k2": 2}, [_load("2", t="baja:Station")]]),
         ops=[OPEN, _lt("station:"), STATE]),
    dict(name="a_poll_answering_a_number_is_malformed", ports=None,
         hook=Script(muted=1, polls=[5]), ops=[OPEN, _lt("station:"), STATE]),
    dict(name="a_poll_answering_a_string_is_malformed",
         ports="test_pollchgs_must_answer_a_list_of_events",
         hook=Script(muted=1, polls=["ab"]), ops=[OPEN, {"op": "poll"}, STATE]),
    dict(name="an_early_event_with_ops_that_are_no_list_is_kept",
         ports="test_an_early_event_whose_ops_are_no_list_is_kept_not_walked",
         hook=Script(muted=1, polls=[[{"evs": {"ops": 5}}, {"evs": {"ops": "ab"}}],
                                     [_load("2", t="baja:Station")]]),
         ops=[OPEN, _lt("station:"), STATE]),
    dict(name="a_load_child_without_a_name_is_malformed", ports=None,
         hook=Script(muted=1, polls=[[], [_load("2", s=[{"t": "x"}])]]), ops=[OPEN, _lt("station:"), STATE]),
    dict(name="a_load_child_with_a_numeric_name_is_malformed", ports=None,
         hook=Script(muted=1, polls=[[], [_load("2", s=[{"n": 5}])]]), ops=[OPEN, _lt("station:")]),
    dict(name="load_children_that_are_no_list_are_malformed", ports=None,
         hook=Script(muted=1, polls=[[], [_load("2", s={"a": 1})]]), ops=[OPEN, _lt("station:")]),
    dict(name="a_load_child_that_is_no_object_is_malformed", ports=None,
         hook=Script(muted=1, polls=[[], [_load("2", s=[{"n": "a", "s": [None]}])]]),
         ops=[OPEN, _lt("station:")]),
    dict(name="a_reply_nested_past_the_limit_is_malformed",
         ports="test_a_reply_nested_past_the_limit_is_refused",
         hook=Script(polls=[_nested(997), _nested(998)]),
         ops=[OPEN, {"op": "poll"}, {"op": "poll"}, STATE]),
    dict(name="a_load_with_an_empty_child_name_is_malformed", ports=None,
         hook=Script(muted=1, polls=[[], [_load("2", h="2", s=[{"n": "", "t": "x", "s": [{"n": "k"}]}])]]),
         ops=[OPEN, _lt("station:"), STATE]),
    # The write side (S10): the plan engine sends a set as one syncTo op and an action as an
    # invokeAction ssc; the load after it reads the written Status slot back.
    dict(name="set_slot_whole_status_numeric_reads_back_ok", ops=[
        OPEN, _sync("a", n="Temp", b={"nm": "p", "t": "kitControl:NumericConst", "s": []}),
        _sync("s", h="10", n="out", b=box.bson_status_numeric(30.0)), _lt(depth=3), STATE]),
    dict(name="invoke_action_set_writes_fallback_whole", ops=[
        OPEN, _sync("a", n="W", b={"nm": "p", "t": "control:NumericWritable", "s": []}),
        {"op": "ssc", "key": "invokeAction", "arg": {"h": "10", "a": "set",
                                                     "b": box.bson_double(7.5)}},
        _lt(depth=3), STATE]),
    dict(name="sync_rejects_multiple_ops", ops=[
        OPEN, {"op": "sync", "sync": [{"nm": "v", "h": "3", "n": "a"}]}, STATE]),
    dict(name="sync_station_error_is_boxerror", ports=None,
         ops=[OPEN, _sync("q", n="x"), _sync("s", h="99", n="x", b=box.bson_bool(True))]),
    dict(name="sync_http_500_is_boxerror", ports=None, hook=Script(fail={"syncTo": 500}),
         ops=[OPEN, _sync("s", h="3", n="x", b=box.bson_bool(True))]),
    # S16b: the BoxClient write helpers no server tool calls, as the frames they send. Go has
    # no such helpers; its plan engine sends these ops through the same Sync and SSC.
    dict(name="add_component_visible_with_handle_and_annotation", ops=[
        OPEN, _sync("a", n="Temp", b=dict(NUMERIC, s=[box.ws_annotation(10, 20)])), _lt(), STATE]),
    dict(name="check_links_returns_link_name_and_creates_link", ops=[
        OPEN, _sync("a", n="A", b=NUMERIC), _sync("a", n="B", b=NUMERIC),
        {"op": "ssc", "key": "checkLinks",
         "arg": {"s": "10", "ss": "out", "t": "11", "ts": "in10", "c": True}},
        _lt(FOLDER + "/B"), STATE]),
    dict(name="remove_component", ops=[
        OPEN, _sync("a", n="Gone", b=NUMERIC), _sync("v", n="Gone"), _lt(), STATE]),
    dict(name="save_station_increments_counter", ops=[
        OPEN, {"op": "ssc", "key": "invokeAction", "arg": {"h": "2", "a": "save"}}, STATE]),
    dict(name="the_station_model_rejects_an_add_nesting_a_baja_folder", ops=[
        OPEN, _sync("a", n="Grp", b={"nm": "p", "t": "baja:Folder",
                                     "s": [{"nm": "p", "n": "Sub", "t": "baja:Folder"}]}), STATE]),
]


def record(scenario):
    """Run one scenario against a recording FakeStation: its replay.jsonl entry."""
    from fake_station import FakeStation
    fake = FakeStation().start()
    try:
        fake.record()
        fake.hook = scenario.get("hook")
        cfg = scenario.get("client", {})
        results = run_ops(make_client(cfg, fake.url), scenario["ops"])
        frames = [{"request": f["request"], "status": f["status"], "reply": f["reply"]}
                  for f in fake.frames]
    finally:
        fake.stop()
    ports = scenario.get("ports", "test_" + scenario["name"])
    return {"name": scenario["name"], "ports": ports, "client": cfg, "ops": scenario["ops"],
            "frames": frames, "results": [json.dumps(r) for r in results]}


def replay_text():
    return "".join(json.dumps(record(s)) + "\n" for s in SCENARIOS)  # op args keep their order


def live():
    spec = json.load(sys.stdin)
    try:
        client = make_client(spec["client"], spec["client"]["url"])
    except Exception as exc:  # the constructor's refusal is the result
        print(json.dumps({"error": describe(exc)}))
        return
    for result in run_ops(client, spec["ops"]):
        print(json.dumps(result))


def main(argv):
    if argv == ["--live"]:
        live()
        return 0
    if argv == ["--write"]:
        os.makedirs(os.path.dirname(REPLAY_FILE), exist_ok=True)
        with open(REPLAY_FILE, "w", encoding="ascii", newline="") as fh:
            fh.write(replay_text())
        return 0
    sys.stdout.write(replay_text())
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
