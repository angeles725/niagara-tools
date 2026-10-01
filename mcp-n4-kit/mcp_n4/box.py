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
                raise AuthError("HTTP %d from station" % exc.code, channel, key) from None
            raise BoxError("HTTP %d from station" % exc.code, channel, key) from None
        except (urllib.error.URLError, OSError, ValueError) as exc:
            raise BoxError("transport failure: %s" % exc, channel, key) from None
        return self._unwrap(reply, channel, key)

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

        If a step after `make` fails with a non-auth error the server session is
        closed (best effort) before the error propagates, so a failed
        `with BoxClient(...)` never leaks. An `AuthError` skips that cleanup `del`
        (it would be one more rejected login toward the 5-in-30-s lock-out) and a
        KeyboardInterrupt is re-raised at once; the local `sid` is dropped either way.
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
        except Exception:
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

    def _split(self, events, handle):
        """Return the first `l` op matching `handle` (any `l` op if None).

        Every other event, and every other op of an event, is kept in pending.
        """
        found = None
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
                    rest.append(op)
            self._keep_rest(event, rest)
        return found

    def _load_once(self, ord_str, depth, attempts, delay, sleep, handle):
        """Request `ord_str` and poll up to `attempts` times for its load op."""
        for event in self.poll():  # queued before the request: cannot be its reply
            self._keep(event)
        self.ssc("loadSlots", {"o": ord_str, "d": depth})
        for attempt in range(attempts):
            op = self._split(self.poll(), handle)
            if op is not None:
                return op
            if attempt < attempts - 1:
                sleep(delay)
        return None

    def load_tree(self, ord_str, depth=2, attempts=6, delay=0.5, sleep=time.sleep,
                  handle=None):
        """Load `ord_str` and return a flat {path: node} dict (root key is "").

        Only the load op for the requested target is returned. Events already
        queued before the request are kept in `pending_events` (they cannot belong
        to it). When the node handle is known (the `handle` argument, else one
        cached from `open()` or an earlier load of the same ord) only an `l` op
        carrying that handle is accepted; when unknown, the first `l` op arriving
        after the request is accepted, whatever its handle. Every other event or
        op is kept in `pending_events` and two load ops are never merged.

        A handle taken from the cache (not passed in) may be stale; if no matching
        op arrives, the request is repeated once with the handle treated as unknown.
        """
        explicit = handle is not None
        handle = handle if explicit else self._handles.get(ord_str)
        op = self._load_once(ord_str, depth, attempts, delay, sleep, handle)
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
        """Forget cached handles for ords starting with `ord_prefix` (all if None)."""
        if ord_prefix is None:
            self._handles.clear()
            return
        for key in [k for k in self._handles if k.startswith(ord_prefix)]:
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
            child = parent + ("/" if "|slot:" in parent else "|slot:/") + name
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
