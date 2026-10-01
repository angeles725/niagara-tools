"""Stdlib-only Niagara BOX JSON client (route R-A, no station module needed).

Wire protocol certified live against N4.14 in niagara-research B1199.
Importing this module has no side effects.
"""
import base64
import json
import ssl
import time
import urllib.error
import urllib.request

PROTOCOL_VERSION = "2.3"
CHANNEL = "ssession"


class BoxError(Exception):
    def __init__(self, message, channel=None, key=None):
        super().__init__(message)
        self.channel, self.key, self.message = channel, key, message


class AuthError(BoxError):
    """HTTP 401/403. Never retried: 5 failures in 30 s lock the account (B1179)."""


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
            opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ctx))
        self._opener = opener
        self._seq = 0
        self.sid = None

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
        except urllib.error.HTTPError as exc:
            exc.close()
            if exc.code in (401, 403):
                raise AuthError("HTTP %d from station" % exc.code, channel, key) from None
            raise BoxError("HTTP %d from station" % exc.code, channel, key) from None
        except (urllib.error.URLError, OSError, ValueError) as exc:
            raise BoxError("transport failure: %s" % exc, channel, key) from None
        msg = reply["m"][0]
        if msg.get("t") == "e":
            raise BoxError(msg["b"].get("m", "station error"), channel, key)
        return msg.get("b")

    def ssc(self, key, arg):
        return self.call(CHANNEL, "callssc",
                         {"id": self.sid, "scid": "cs1", "sck": key, "scarg": arg})

    def poll(self):
        return self.call(CHANNEL, "pollchgs", {"id": self.sid}) or []

    # -- session
    def open(self):
        self.sid = self.call(CHANNEL, "make", {})
        self.call(CHANNEL, "makessc", {"id": self.sid, "scid": "cs1",
                                       "scts": "box:ComponentSpaceSessionHandler",
                                       "scarg": "station:"})
        return self.ssc("loadRoot", None)

    def close(self):
        if not self.sid:
            return
        try:
            self.call(CHANNEL, "del", {"id": self.sid})
        except BoxError:
            pass
        self.sid = None

    # -- reading
    def load_tree(self, ord_str, depth=2, attempts=6, delay=0.5, sleep=time.sleep):
        self.ssc("loadSlots", {"o": ord_str, "d": depth})
        for attempt in range(attempts):
            nodes = {}
            for event in self.poll():
                for op in event.get("evs", {}).get("ops", []):
                    if op.get("nm") == "l" and op.get("b"):
                        _flatten(op["b"], "", nodes)
            if nodes:
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
        body = {"nm": "p", "t": type_spec, "s": [ws] if ws else []}
        res = self.sync({"nm": "a", "h": parent_h, "n": name, "b": body})
        return res[0] if res else res

    def remove_component(self, parent_h, name):
        return self.sync({"nm": "v", "h": parent_h, "n": name})

    def set_slot(self, h, path, bson, *, allow_partial_status=False):
        if path.split("/")[-1] in ("value", "status") and "/" in path \
                and not allow_partial_status:
            raise ValueError("write the whole Status slot, not %r (B1199: a value-only "
                             "write leaves status null)" % path)
        return self.sync({"nm": "s", "h": h, "n": path, "b": bson})

    def check_links(self, src_h, src_slot, tgt_h, tgt_slot, add=True):
        res = self.ssc("checkLinks", {"s": src_h, "ss": src_slot,
                                      "t": tgt_h, "ts": tgt_slot, "c": add})
        return res[0] if res else res

    def invoke_action(self, h, action, bson=None):
        arg = {"h": h, "a": action}
        if bson is not None:
            arg["b"] = bson
        return self.ssc("invokeAction", arg)

    def save_station(self, root_h="2"):
        return self.invoke_action(root_h, "save")
