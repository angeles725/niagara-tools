"""Stdlib-only Niagara BOX JSON client (route R-A, no station module needed).

Wire protocol certified live against N4.14 in niagara-research B1199.
Importing this module has no side effects.
"""
import base64
import json
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request

PROTOCOL_VERSION = "2.3"
CHANNEL = "ssession"
#: Id of the session component created by `makessc` and addressed by `callssc`.
SESSION_COMPONENT_ID = "cs1"
#: Default station root handle. It is the value `loadRoot` returns (`open()` result
#: `["h"]`); callers should pass `root["h"]` rather than rely on this default.
DEFAULT_ROOT_HANDLE = "2"
#: Upper bound on events kept in `pending_events`; the oldest are dropped first.
MAX_PENDING_EVENTS = 500
#: Load polling (retro 2026-10-02 D2): the first delay is short, then it doubles up to
#: MAX_POLL_DELAY; the total sleep of one polling window never exceeds MAX_POLL_WAIT.
FIRST_POLL_DELAY = 0.1
MAX_POLL_DELAY = 0.8
MAX_POLL_WAIT = 3.0
#: Actionable auth hints (retro 2026-10-02 D5). A failed login is never retried.
AUTH_HINTS = {
    401: "check the user's Authentication Scheme Name = HTTPBasicScheme and the password; "
         "do not retry (lockout: 5 failures in 30 s)",
    403: "the user is authenticated but lacks permission on this resource; "
         "do not retry (lockout: 5 failures in 30 s)",
}


def auth_message(code):
    """The `AuthError` text for HTTP `code` (401/403): the status plus what to check."""
    hint = AUTH_HINTS.get(code)
    return "HTTP %d from station" % code + (": " + hint if hint else "")


def poll_delays(first, cap, total, attempts):
    """The sleeps between `attempts` polls: `first`, doubling, each <= `cap`, sum <= `total`.

    Pure, so the schedule is testable without a clock: (0.1, 0.8, 3.0, 20) gives
    0.1, 0.2, 0.4, 0.8, 0.8, 0.7. No sleep follows the last poll.
    """
    slept, delay = 0.0, first
    for _ in range(max(0, attempts - 1)):
        step = round(min(delay, cap, total - slept), 6)
        if step <= 0:
            return
        yield step
        slept += step
        delay *= 2


class BoxError(Exception):
    def __init__(self, message, channel=None, key=None):
        super().__init__(message)
        self.channel, self.key, self.message = channel, key, message


class AuthError(BoxError):
    """HTTP 401/403. Never retried: 5 failures in 30 s lock the account (B1179)."""


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    """Refuse every redirect: urllib would re-send Authorization to the new target."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        fp.close()  # release the 3xx response before raising
        host = urllib.parse.urlsplit(newurl).hostname or "unknown host"
        raise BoxError("refusing HTTP %d redirect to host %s (credentials are never "
                       "forwarded)" % (code, host))


# ---- encoders ------------------------------------------------------------

def bson_double(x):
    return {"nm": "p", "t": "baja:Double", "v": str(float(x))}


def bson_bool(b):
    return {"nm": "p", "t": "baja:Boolean", "v": "true" if b else "false"}


def _status_whole(type_, value_bson, status):
    return {"nm": "p", "t": type_, "s": [
        dict(value_bson, n="value"),
        {"nm": "p", "n": "status", "t": "baja:Status", "v": str(status)},
    ]}


def bson_status_numeric(value, status="0"):
    return _status_whole("baja:StatusNumeric", bson_double(value), status)


def bson_status_boolean(value, status="0"):
    return _status_whole("baja:StatusBoolean", bson_bool(value), status)


def ws_annotation(x, y, w=8):
    return {"nm": "p", "n": "wsAnnotation", "t": "baja:WsAnnotation", "v": "%s,%s,%s" % (x, y, w)}


# ---- ORD grammar ---------------------------------------------------------

def child_ord(parent_ord, name):
    """The ORD of child `name` under `parent_ord`: the ONE place an ORD is joined.

    `station:` -> `station:|slot:/X`; `station:|slot:/` -> `station:|slot:/X`;
    `station:|slot:/A` (or `.../A/`) -> `station:|slot:/A/X`. Never produces `//`
    (the real station refuses it: "Illegal double slashes"). `name` must be one plain
    slot name: not empty, no `/`, no `|`, not `..`.
    """
    if not isinstance(name, str) or not name or "/" in name or "|" in name or name == "..":
        raise ValueError("ORD child name must be one plain slot name, got %r" % (name,))
    if not isinstance(parent_ord, str) or "//" in parent_ord:
        raise ValueError("parent ORD is malformed (double slashes?): %r" % (parent_ord,))
    if "|slot:" not in parent_ord:
        parent_ord = parent_ord.rstrip("|") + "|slot:/"
    return parent_ord.rstrip("/") + "/" + name


def join_ord(parent_ord, rel_path):
    """`child_ord` applied to each segment of a relative `a/b/c` path ("" -> the parent)."""
    if not rel_path:
        return parent_ord if parent_ord.endswith(":|slot:/") else parent_ord.rstrip("/")
    out = parent_ord
    for part in rel_path.split("/"):
        out = child_ord(out, part)
    return out


# ---- readers -------------------------------------------------------------

_DEFAULT_VALUE = {"baja:StatusBoolean": "false", "baja:StatusNumeric": "0.0"}


def parse_status(v):
    bits_str, _, facets = (v or "0").partition(";")
    bits = int(bits_str or "0", 16)
    return {"bits": bits, "null": bool(bits & 0x40), "ok": bits == 0, "facets": facets}


def children(nodes, path):
    """Direct children of `path` in a load_tree() dict, as {name: node}."""
    prefix = path + "/" if path else ""
    out = {}
    for key, node in nodes.items():
        if key and key != path and key.startswith(prefix) and "/" not in key[len(prefix):]:
            out[key[len(prefix):]] = node
    return out


def status_value(nodes, path):
    """Value and status of a Status* slot; omitted children take the type default."""
    default = _DEFAULT_VALUE.get(nodes[path].get("t"))
    if default is None:
        raise ValueError("%s is not a StatusBoolean/StatusNumeric" % path)
    kids = children(nodes, path)
    return {"value": kids.get("value", {}).get("v", default),
            "status": kids.get("status", {}).get("v", "0")}


#: Type contract table (B1200-G3): type spec -> True when the type is a BComponent the
#: station creates with its own add op, False when it is a slot value (BSimple/BStruct).
#: An entry here always wins over the naming heuristic below. Each entry is backed by the
#: type's declaration (`extends`) in the decompiled N4 sources or by a live load:
#:   baja:Component          BComponent itself
#:   baja:Folder             BFolder extends BComponent
#:   baja:UnrestrictedFolder BUnrestrictedFolder extends BFolder (heuristic says value)
#:   control:NullProxyExt    BNullProxyExt extends BAbstractProxyExt (live: a writable's frozen child)
#:   control:PriorityLevel   BPriorityLevel extends BFrozenEnum (heuristic says component)
#:   baja:Link, baja:WsAnnotation  slot values the write tools rely on
#: The kit has no live contract lookup yet (BOX `reg.loadContract`, B1173): it would need a
#: new channel certified on a station. Add an entry here when a type is misclassified.
COMPONENT_TYPES = {
    "baja:Component": True,
    "baja:Folder": True,
    "baja:UnrestrictedFolder": True,
    "control:NullProxyExt": True,
    "control:PriorityLevel": False,
    "baja:Link": False,
    "baja:WsAnnotation": False,
}


def is_component_type(type_):
    """True when the station must create `type_` with its own add op.

    Shared by the write tools and the test fake. An entry in `COMPONENT_TYPES` decides
    first. Heuristic fallback for any other type: every type outside the `baja` module
    is a component (`kitControl:NumericConst`, `control:...`); the other `baja:` types
    (`Double`, `StatusNumeric`, ...) are plain slot values. The fallback is a naming
    rule, not a type-registry lookup: a `baja:` component or a non-`baja` value type
    missing from the table is misclassified, so extend the table when one shows up.
    """
    if not isinstance(type_, str) or ":" not in type_:
        return False
    if type_ in COMPONENT_TYPES:
        return COMPONENT_TYPES[type_]
    return type_.partition(":")[0] != "baja"


def _flatten(node, path, out):
    out[path] = node
    for child in node.get("s", []):
        _flatten(child, child["n"] if not path else path + "/" + child["n"], out)


# ---- client --------------------------------------------------------------

class BoxClient:
    def __init__(self, base_url, username, password, *, timeout=20,
                 insecure_tls=False, allow_http=False, opener=None):
        if base_url.startswith("http://") and not allow_http:
            raise ValueError("refusing http:// (password would travel in clear); use https")
        if not base_url.startswith(("http://", "https://")):
            raise ValueError("base_url must start with https://")
        self.base_url = base_url.rstrip("/")
        self.username = username
        self.timeout = timeout
        self._auth = "Basic " + base64.b64encode(
            ("%s:%s" % (username, password)).encode()).decode()
        if opener is None:
            ctx = None
            if insecure_tls:
                ctx = ssl.create_default_context()
                ctx.check_hostname = False
                ctx.verify_mode = ssl.CERT_NONE
            opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ctx),
                                                 _NoRedirect())
        self._opener = opener
        self._seq = 0
        self.sid = None
        self._pending_events = []
        #: loadSlots requests sent minus load ops seen: > 0 means an earlier request may
        #: still be answered later, so an unexpected load op is not necessarily ours
        self._loads_outstanding = 0
        self.dropped_events = 0  # events discarded because the pending list was full
        self._handles = {}  # ord -> node handle, learned from open() and load_tree()

    def __repr__(self):
        return "BoxClient(base_url=%r, username=%r)" % (self.base_url, self.username)

    def __enter__(self):
        self.open()
        return self

    def __exit__(self, *exc):
        self.close()

    # -- transport
    def call(self, channel, key, body):
        self._seq += 1
        frame = {"p": "box", "v": PROTOCOL_VERSION,
                 "m": [{"r": 0, "t": "rt", "c": channel, "k": key, "b": body}], "n": self._seq}
        if self.sid:
            frame["id"] = self.sid
        req = urllib.request.Request(
            self.base_url + "/box/", data=json.dumps(frame).encode(), method="POST",
            headers={"Authorization": self._auth, "Content-Type": "application/json"})
        try:
            with self._opener.open(req, timeout=self.timeout) as resp:
                reply = json.loads(resp.read().decode())
        except BoxError:
            raise
        except urllib.error.HTTPError as exc:
            exc.close()
            if exc.code in (401, 403):
                raise AuthError(auth_message(exc.code), channel, key) from None
            raise BoxError("HTTP %d from station" % exc.code, channel, key) from None
        except (urllib.error.URLError, OSError, ValueError) as exc:
            raise BoxError("transport failure: %s" % exc, channel, key) from None
        return self._unwrap(reply, channel, key)

    def get_ord(self, ord_text, timeout=None, max_bytes=32 * 1024 * 1024):
        """`GET /ord/<url-encoded ORD>` (path form; `/ord?` answers 400): `(text, truncated)`.

        Stateless HTTP Basic, no BOX session needed. The same opener refuses redirects,
        a 401/403 raises `AuthError` once (never retried), and at most `max_bytes` are
        read; `truncated` is True when the body was longer.
        """
        url = self.base_url + "/ord/" + urllib.parse.quote(ord_text, safe="")
        req = urllib.request.Request(url, method="GET", headers={
            "Authorization": self._auth, "Accept": "text/csv"})
        try:
            with self._opener.open(req, timeout=timeout or self.timeout) as resp:
                raw = resp.read(max_bytes + 1)
        except BoxError:
            raise
        except urllib.error.HTTPError as exc:
            exc.close()
            if exc.code in (401, 403):
                raise AuthError(auth_message(exc.code), "ord", "get") from None
            if exc.code == 400:
                raise BoxError("HTTP 400 from station: the ORD or BQL query was rejected "
                               "(check the type spec, column names and where clause)",
                               "ord", "get") from None
            raise BoxError("HTTP %d from station" % exc.code, "ord", "get") from None
        except (urllib.error.URLError, OSError, ValueError) as exc:
            raise BoxError("transport failure: %s" % exc, "ord", "get") from None
        truncated = len(raw) > max_bytes
        text = raw[:max_bytes].decode("utf-8", "replace")
        if truncated:  # never hand back a torn last line
            text = text[:text.rfind("\n") + 1]
        return text, truncated

    def _unwrap(self, reply, channel, key):
        """Validate a reply frame and return its first message body, else BoxError."""
        if not isinstance(reply, dict):
            raise BoxError("malformed reply: not an object", channel, key)
        if "n" in reply and reply["n"] != self._seq:
            raise BoxError("reply seq %r does not match request seq %d"
                           % (reply["n"], self._seq), channel, key)
        msgs = reply.get("m")
        if not isinstance(msgs, list) or not msgs or not isinstance(msgs[0], dict):
            raise BoxError("malformed reply: missing or empty message list", channel, key)
        msg = msgs[0]
        if msg.get("t") == "e":
            body = msg.get("b")
            text = body.get("m") if isinstance(body, dict) else None
            raise BoxError(text if isinstance(text, str) else "station error", channel, key)
        return msg.get("b")

    def ssc(self, key, arg):
        return self.call(CHANNEL, "callssc",
                         {"id": self.sid, "scid": SESSION_COMPONENT_ID, "sck": key, "scarg": arg})

    def poll(self):
        return self.call(CHANNEL, "pollchgs", {"id": self.sid}) or []

    # -- session
    def open(self):
        """Open a session and return the `loadRoot` reply (`{"h": ..., "t": ...}`).

        If a step after `make` fails with anything but an `AuthError` (including
        KeyboardInterrupt and SystemExit) the server session is closed (best effort)
        before the exception propagates, so a failed `with BoxClient(...)` never
        leaks. An `AuthError` skips that cleanup `del` (it would be one more rejected
        login toward the 5-in-30-s lock-out); the local `sid` is dropped either way.
        """
        self.sid = self.call(CHANNEL, "make", {})
        try:
            self.call(CHANNEL, "makessc", {"id": self.sid, "scid": SESSION_COMPONENT_ID,
                                           "scts": "box:ComponentSpaceSessionHandler",
                                           "scarg": "station:"})
            root = self.ssc("loadRoot", None)
        except AuthError:
            self.sid = None
            raise
        except BaseException:  # incl. KeyboardInterrupt/SystemExit: do not leak the session
            self.close()
            raise
        if isinstance(root, dict) and root.get("h"):
            self._handles["station:"] = root["h"]
        return root

    def close(self):
        if not self.sid:
            return
        try:
            self.call(CHANNEL, "del", {"id": self.sid})
        except Exception:  # best effort; never surface (or log) anything sensitive
            pass
        self.sid = None

    # -- reading
    @property
    def pending_events(self):
        """Read-only tuple of events set aside by `load_tree`.

        The list holds at most `MAX_PENDING_EVENTS` entries; when full the oldest
        is dropped and `dropped_events` is incremented. Use `drain_pending_events`
        to consume them.
        """
        return tuple(self._pending_events)

    def drain_pending_events(self):
        """Return the pending events as a list and clear them."""
        drained, self._pending_events = self._pending_events, []
        return drained

    def _keep(self, event):
        """Append `event` to the pending list, dropping the oldest when full."""
        self._pending_events.append(event)
        while len(self._pending_events) > MAX_PENDING_EVENTS:
            del self._pending_events[0]
            self.dropped_events += 1

    def _keep_rest(self, event, ops):
        """Keep `event` minus the consumed ops (all of it when it has no ops)."""
        evs = event.get("evs") if isinstance(event, dict) else None
        if ops is None or not isinstance(evs, dict):
            self._keep(event)
        elif ops:
            self._keep(dict(event, evs=dict(evs, ops=ops)))

    def _settle(self, op):
        """A load op was seen: one outstanding loadSlots request is answered."""
        if isinstance(op, dict) and op.get("nm") == "l":
            self._loads_outstanding = max(0, self._loads_outstanding - 1)

    def _split(self, events, handle):
        """Return `(op, saw_other)`: the first `l` op matching `handle` (any if None).

        `saw_other` is True when a load op carrying another handle was seen. Every other
        event, and every other op of an event, is kept in pending. Each load op seen
        settles one outstanding loadSlots request.
        """
        found, saw_other = None, False
        for event in events:
            evs = event.get("evs") if isinstance(event, dict) else None
            ops = evs.get("ops") if isinstance(evs, dict) else None
            if not isinstance(ops, list):
                self._keep_rest(event, None)
                continue
            rest = []
            for op in ops:
                if (found is None and isinstance(op, dict) and op.get("nm") == "l"
                        and isinstance(op.get("b"), dict)
                        and (handle is None or op.get("h") == handle)):
                    found = op
                else:
                    if (handle is not None and isinstance(op, dict) and op.get("nm") == "l"
                            and isinstance(op.get("b"), dict) and op.get("h") != handle):
                        saw_other = True
                    rest.append(op)
                self._settle(op)
            self._keep_rest(event, rest)
        return found, saw_other

    def _load_once(self, ord_str, depth, attempts, delay, sleep, handle, cached=False):
        """Request `ord_str` and poll up to `attempts` times for its load op.

        With `cached=True` (the handle came from the cache, not the caller) a load op
        carrying another handle is taken as proof the cached one is stale, and the
        wait ends at once instead of exhausting the polling window. That proof holds only
        when no earlier loadSlots request is still unanswered (a late reply to it would
        also carry another handle); otherwise the full window is waited.
        """
        for event in self.poll():  # queued before the request: cannot be its reply
            self._keep(event)
            evs = event.get("evs") if isinstance(event, dict) else None
            for op in (evs.get("ops") or []) if isinstance(evs, dict) else []:
                self._settle(op)
        trusted = self._loads_outstanding == 0
        self._loads_outstanding += 1
        try:
            self.ssc("loadSlots", {"o": ord_str, "d": depth})
        # Assumption (PR #184 review R3-001): `ssc` raises either before the request is sent
        # (transport error) or on an explicit error reply, so no load op will follow. A
        # raise after the station received the request (a lost reply) would under-count
        # by one; the in-order reset below bounds that to the next answered load.
        except BaseException:  # rejected or never sent: no load op will answer it
            self._loads_outstanding = max(0, self._loads_outstanding - 1)
            raise
        delays = poll_delays(delay, MAX_POLL_DELAY, MAX_POLL_WAIT, attempts)
        while True:
            op, saw_other = self._split(self.poll(), handle)
            if op is not None:
                # Replies come in request order: once this request is answered, an
                # earlier one that timed out will not be answered any more, so the
                # session is trusted again (issue #179 R4-001). Assumption: the station
                # answers one session's loads in order, as observed live; it is not a
                # documented BOX guarantee, and an out-of-order reply would be matched by
                # handle in `_split` anyway, only the early-abort trust would be early.
                self._loads_outstanding = 0
                return op
            if cached and trusted and saw_other:
                return None
            pause = next(delays, None)
            if pause is None:
                return None
            sleep(pause)

    def load_tree(self, ord_str, depth=2, attempts=12, delay=FIRST_POLL_DELAY,
                  sleep=time.sleep, handle=None):
        """Load `ord_str` and return a flat {path: node} dict (root key is "").

        Polling: at most `attempts` polls per window, sleeping `poll_delays(delay, ...)`
        between them (short first delay, backoff, each capped at MAX_POLL_DELAY, total
        sleep per window bounded by MAX_POLL_WAIT).

        Only the load op for the requested target is returned. Events already
        queued before the request are kept in `pending_events` (they cannot belong
        to it). When the node handle is known (the `handle` argument, else one
        cached from `open()` or an earlier load of the same ord) only an `l` op
        carrying that handle is accepted; when unknown, the first `l` op arriving
        after the request is accepted, whatever its handle. Every other event or
        op is kept in `pending_events` and two load ops are never merged.

        A handle taken from the cache (not passed in) may be stale; if no matching
        op arrives (or a load op with another handle does, which proves it stale),
        the request is repeated once with the handle treated as unknown.
        """
        explicit = handle is not None
        handle = handle if explicit else self._handles.get(ord_str)
        op = self._load_once(ord_str, depth, attempts, delay, sleep, handle,
                             cached=handle is not None and not explicit)
        if op is None and handle is not None and not explicit:
            self._handles.pop(ord_str, None)
            op = self._load_once(ord_str, depth, attempts, delay, sleep, None)
        if op is None:
            raise BoxError("no load event for %s after %d polls" % (ord_str, attempts),
                           CHANNEL, "loadSlots")
        nodes = {}
        _flatten(op["b"], "", nodes)
        if nodes.get("", {}).get("h"):
            self._handles[ord_str] = nodes[""]["h"]
        return nodes

    def invalidate_handles(self, ord_prefix=None):
        """Forget cached handles of `ord_prefix` and its descendants (all if None).

        The match is boundary-aware: `.../A` clears `.../A` and `.../A/B`, never `.../AB`.
        """
        if ord_prefix is None:
            self._handles.clear()
            return
        base = ord_prefix.rstrip("/")
        for key in [k for k in self._handles if k == base or k.startswith(base + "/")]:
            del self._handles[key]

    # -- writing (one op per syncTo)
    def sync(self, op):
        if not isinstance(op, dict):
            raise ValueError("sync() takes exactly one op dict")
        return self.ssc("syncTo", {"nm": "sync", "ver": 1.0, "ops": [op]})

    def add_component(self, parent_h, name, type_spec, ws=None):
        """Add a component; return `{"id": ..., "nn": <server-assigned name>}`.

        The station may rename on collision, so always use `nn`, not `name`.
        """
        body = {"nm": "p", "t": type_spec, "s": [ws] if ws else []}
        res = self.sync({"nm": "a", "h": parent_h, "n": name, "b": body})
        if not (isinstance(res, list) and res and isinstance(res[0], dict)
                and "nn" in res[0]):
            raise BoxError("malformed addComponent reply", CHANNEL, "syncTo")
        return res[0]

    def remove_component(self, parent_h, name):
        """Remove a child and forget cached handles of it and its descendants.

        If `parent_h` is not a handle this client has cached, the child's ord
        cannot be derived, so the whole handle cache is cleared (safe, just slower).
        """
        res = self.sync({"nm": "v", "h": parent_h, "n": name})
        parents = [o for o, h in self._handles.items() if h == parent_h]
        if not parents:
            self._handles.clear()
        for parent in parents:
            try:
                child = child_ord(parent, name)
            except ValueError:  # an unjoinable cached key: forget everything, stay safe
                self._handles.clear()
                break
            for key in [k for k in self._handles
                        if k == child or k.startswith(child + "/")]:
                del self._handles[key]
        return res

    def set_slot(self, h, path, bson, *, allow_partial_status=False):
        if path.split("/")[-1] in ("value", "status") and "/" in path \
                and not allow_partial_status:
            raise ValueError("write the whole Status slot, not %r (B1199: a value-only "
                             "write leaves status null)" % path)
        return self.sync({"nm": "s", "h": h, "n": path, "b": bson})

    def check_links(self, src_h, src_slot, tgt_h, tgt_slot, add=True):
        """Check (and with add=True create) one link; return a list of
        `{"v": valid, "r": reason, "s": link name}` dicts for that single pair."""
        res = self.ssc("checkLinks", {"s": src_h, "ss": src_slot,
                                      "t": tgt_h, "ts": tgt_slot, "c": add})
        # The station wraps the per-pair result list in one outer list (one entry
        # per requested pair); we always send exactly one pair, so unwrap it.
        if not (isinstance(res, list) and res and isinstance(res[0], list)
                and all(isinstance(r, dict) for r in res[0])):
            raise BoxError("malformed checkLinks reply", CHANNEL, "checkLinks")
        return res[0]

    def invoke_action(self, h, action, bson=None):
        arg = {"h": h, "a": action}
        if bson is not None:
            arg["b"] = bson
        return self.ssc("invokeAction", arg)

    def save_station(self, root_h=DEFAULT_ROOT_HANDLE):
        return self.invoke_action(root_h, "save")
