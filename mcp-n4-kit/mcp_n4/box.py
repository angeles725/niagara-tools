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


class BoxError(Exception):
    def __init__(self, message, channel=None, key=None):
        super().__init__(message)
        self.channel, self.key, self.message = channel, key, message


class AuthError(BoxError):
    """HTTP 401/403. Never retried: 5 failures in 30 s lock the account (B1179)."""


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    """Refuse every redirect: urllib would re-send Authorization to the new target."""

    def redirect_request(self, req, fp, code, msg, headers, newurl):
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
        except (urllib.error.URLError, TimeoutError, OSError, ValueError) as exc:
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

        If any step after `make` fails the server session is closed (best effort)
        before the error propagates, so a failed `with BoxClient(...)` never leaks.
        """
        self.sid = self.call(CHANNEL, "make", {})
        try:
            self.call(CHANNEL, "makessc", {"id": self.sid, "scid": SESSION_COMPONENT_ID,
                                           "scts": "box:ComponentSpaceSessionHandler",
                                           "scarg": "station:"})
            root = self.ssc("loadRoot", None)
        except BaseException:
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
        """Read-only tuple of events set aside by `load_tree` (never dropped)."""
        return tuple(self._pending_events)

    def _stash(self, event, ops):
        """Keep `event` minus the consumed ops (all of it when it has no ops)."""
        if not isinstance(event, dict):
            self._pending_events.append(event)
            return
        evs = event.get("evs")
        if ops is None or not isinstance(evs, dict):
            self._pending_events.append(event)
        elif ops:
            self._pending_events.append(dict(event, evs=dict(evs, ops=ops)))

    def _split(self, events, handle):
        """Pick the first `l` op matching `handle` (any `l` op if None); stash the rest."""
        found = None
        for event in events:
            evs = event.get("evs") if isinstance(event, dict) else None
            ops = evs.get("ops") if isinstance(evs, dict) else None
            if not isinstance(ops, list):
                self._stash(event, None)
                continue
            rest = []
            for op in ops:
                if (found is None and isinstance(op, dict) and op.get("nm") == "l"
                        and isinstance(op.get("b"), dict)
                        and (handle is None or op.get("h") == handle)):
                    found = op
                else:
                    rest.append(op)
            self._stash(event, rest)
        return found

    def load_tree(self, ord_str, depth=2, attempts=6, delay=0.5, sleep=time.sleep,
                  handle=None):
        """Load `ord_str` and return a flat {path: node} dict (root key is "").

        Only the load op for the requested target is returned, chosen by this rule:
        events already queued before the request are drained into `pending_events`
        (they cannot belong to it); then, when the node handle is known (the `handle`
        argument, else one learned from `open()` or an earlier load of the same ord),
        only an `l` op carrying that handle is accepted; when unknown, the first `l`
        op arriving after the request is. Everything else is kept in `pending_events`
        and two load ops are never merged into one tree.
        """
        handle = handle if handle is not None else self._handles.get(ord_str)
        self._pending_events.extend(self.poll())
        stale, self._pending_events = self._pending_events, []
        self._split(stale, object())  # nothing matches: stash everything as pending
        self.ssc("loadSlots", {"o": ord_str, "d": depth})
        for attempt in range(attempts):
            op = self._split(self.poll(), handle)
            if op is not None:
                nodes = {}
                _flatten(op["b"], "", nodes)
                if nodes.get("", {}).get("h"):
                    self._handles[ord_str] = nodes[""]["h"]
                return nodes
            if attempt < attempts - 1:
                sleep(delay)
        raise BoxError("no load event for %s after %d polls" % (ord_str, attempts),
                       CHANNEL, "loadSlots")

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
        return self.sync({"nm": "v", "h": parent_h, "n": name})

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
            raise BoxError("malformed checkLinks reply", CHANNEL, "callssc")
        return res[0]

    def invoke_action(self, h, action, bson=None):
        arg = {"h": h, "a": action}
        if bson is not None:
            arg["b"] = bson
        return self.ssc("invokeAction", arg)

    def save_station(self, root_h=DEFAULT_ROOT_HANDLE):
        return self.invoke_action(root_h, "save")
