"""In-memory fake Niagara BOX station for unit tests (plain HTTP on 127.0.0.1)."""
import json
import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import base64

from mcp_n4 import box

# Values equal to the type default are omitted from events, like the real station.
_DEFAULTS = {"baja:Status": "0", "baja:Double": "0.0", "baja:Boolean": "false"}


class _Node:
    def __init__(self, name, type_, value=None, handle=None):
        self.name, self.type, self.value, self.handle = name, type_, value, handle
        self.children = []

    def child(self, name):
        return next((c for c in self.children if c.name == name), None)


#: Writables own a FROZEN `proxyExt` child (live N4.14, 2026-10-01): it exists as soon as
#: the writable does, and adding another child of a ProxyExt type is illegal.
_WRITABLES = ("control:NumericWritable", "control:BooleanWritable")

#: reg/loadContract contract chains (B1200-G3, live N5 5.0.0.28 evidence 2026-10-04):
#: type spec -> the frozen-slot contract chain as a dict keyed by the FULL ancestor
#: chain (the type itself and, for components, `baja:Component` among the keys). Each
#: key maps to that ancestor's frozen-slot entries; they are empty here because the
#: tests inspect keys and chain length only. Shapes:
#:   control:NumericWritable  component: the certified 22-key chain, `baja:Component` in it
#:   control:PriorityLevel    slot value: 2-key chain, no `baja:Component`
#:   baja:Folder              length 1: AMBIGUOUS (a component whose chain holds only itself)
#:   baja:WsAnnotation        length 1: a slot value with the same shape
#:   baja:TestComponent       test-only stand-in for a baja component missing from the
#:                            static COMPONENT_TYPES table (what the live lookup fixes)
#: Any other type raises, like the live station's error frame for an unknown type.
_CONTRACT_CHAINS = {
    "control:NumericWritable": {key: [] for key in (
        "control:NumericWritable", "control:NumericPoint", "control:ControlPoint",
        "control:PointExtension", "control:AbstractProxyExt", "control:IWritablePoint",
        "control:NullProxyExt", "control:Override", "control:NumericOverride",
        "baja:INumeric", "baja:IStatusValue", "baja:StatusValue", "baja:Status",
        "baja:StatusNumeric", "baja:INiagaraSyncCapableComplex", "baja:IActionAuditProvider",
        "baja:Interface", "baja:Struct", "baja:Facets", "baja:AbsTime", "baja:RelTime",
        "baja:Component")},
    "control:PriorityLevel": {"control:PriorityLevel": [], "baja:FrozenEnum": []},
    "baja:Folder": {"baja:Folder": []},
    "baja:WsAnnotation": {"baja:WsAnnotation": []},
    "baja:TestComponent": {"baja:TestComponent": [], "baja:Component": []},
}


class FakeStation:
    def __init__(self, user="admin", password="secret", station_name="FakeStation"):
        self.user, self.password = user, password
        self.requests = 0
        self.saves = 0
        # When set, `save` rewrites this file (like a station persisting config.bog);
        # `save_rewrites=False` makes save a no-op on disk.
        self.config_path = None
        self.save_rewrites = True
        self.sessions = set()
        # Fault injection: hook(frame) -> None (serve normally) or (code, body_bytes, headers).
        self.hook = None
        # Called with each applied syncTo op, so a test can tamper with the model
        # afterwards (e.g. to simulate a read-back mismatch).
        self.on_sync = None
        self.invoked = []  # (action, handle) of every accepted invokeAction except save
        #: What `GET /obix/about/` reports as productVersion; None answers 404 (no oBIX).
        self.product_version = "4.14.0.162"
        self.about_requests = 0  # GETs are not counted in `requests` (BOX POSTs only)
        self._next_handle = 0x10
        self._events = []
        self.root = _Node(None, "baja:Station", handle="2")
        self.by_handle = {"2": self.root}
        self.root.children.append(_Node("stationName", "baja:String", station_name))
        self.folder = self._component(self.root, "Folder", "baja:Folder", "3")
        self._server = None

    # ---- lifecycle -------------------------------------------------------
    def start(self):
        station = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_GET(self):
                station.about_requests += 1
                if not station._authorized(self.headers.get("Authorization", "")):
                    code, out = 401, b""
                elif self.path.rstrip("/") != "/obix/about" or station.product_version is None:
                    code, out = 404, b""
                else:  # shape of a live 4.14 answer (B457): one <str> per property
                    code, out = 200, (
                        '<?xml version="1.0" encoding="UTF-8"?>\n'
                        '<obj is="obix:About" href="/obix/about/">\n'
                        '  <str name="obixVersion" val="1.1"/>\n'
                        '  <str name="serverName" val="fake-host"/>\n'
                        '  <str name="vendorName" val="Tridium"/>\n'
                        '  <str val="%s" name="productVersion"/>\n'
                        '</obj>\n' % station.product_version).encode()
                self.send_response(code)
                self.send_header("Content-Type", "text/xml")
                self.send_header("Content-Length", str(len(out)))
                self.end_headers()
                self.wfile.write(out)

            def do_POST(self):
                station.requests += 1
                length = int(self.headers.get("Content-Length", 0))
                raw = self.rfile.read(length)
                if not station._authorized(self.headers.get("Authorization", "")):
                    self.send_response(401)
                    self.send_header("WWW-Authenticate", 'Basic realm="station"')
                    self.send_header("Content-Length", "0")
                    self.end_headers()
                    return
                if station.hook is not None:
                    injected = station.hook(json.loads(raw))
                    if injected is not None:
                        code, payload, headers = injected
                        self.send_response(code)
                        for name, value in headers.items():
                            self.send_header(name, value)
                        self.send_header("Content-Length", str(len(payload)))
                        self.end_headers()
                        self.wfile.write(payload)
                        return
                out = json.dumps(station.handle_frame(json.loads(raw))).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(out)))
                self.end_headers()
                self.wfile.write(out)

        self._server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=self._server.serve_forever, kwargs={"poll_interval": 0.01}, daemon=True).start()
        self.url = "http://127.0.0.1:%d" % self._server.server_address[1]
        return self

    def stop(self):
        self._server.shutdown()
        self._server.server_close()

    def _authorized(self, header):
        expected = base64.b64encode(("%s:%s" % (self.user, self.password)).encode()).decode()
        return header == "Basic " + expected

    # ---- model helpers ---------------------------------------------------
    def _component(self, parent, name, type_, handle=None):
        if handle is None:
            handle = "%x" % self._next_handle
            self._next_handle += 1
        node = _Node(name, type_, handle=handle)
        parent.children.append(node)
        self.by_handle[handle] = node
        return node

    def _unique(self, parent, name):
        candidate, i = name, 0
        while parent.child(candidate):
            i += 1
            candidate = "%s%d" % (name, i)
        return candidate

    _is_component = staticmethod(box.is_component_type)  # same rule as the write tools

    def _fill(self, node, bson):
        node.type = bson.get("t", node.type)
        node.value = bson.get("v")
        node.children = []
        for sub in bson.get("s", []):
            if self._is_component(sub.get("t")):  # a nested component
                child = self._component(node, sub["n"], sub["t"])
            else:
                child = _Node(sub["n"], sub.get("t"))
                node.children.append(child)
            self._fill(child, sub)

    @staticmethod
    def _no_double_slashes(path):
        """Models the live N4.14 behavior (2026-10-01): `station:|slot://X` is refused."""
        if isinstance(path, str) and "//" in path:
            raise ValueError("Illegal double slashes")

    def _resolve(self, ord_str):
        self._no_double_slashes(ord_str)
        path = ord_str.split("slot:", 1)[1] if "slot:" in ord_str else ""
        node = self.root
        for part in [p for p in path.split("/") if p]:
            node = node.child(part)
            if node is None:
                raise KeyError(ord_str)
        return node

    def _serialize(self, node, depth, is_root=False):
        out = {"nm": "p", "t": node.type}
        if not is_root:
            out["n"] = node.name
        if node.handle:
            out["h"] = node.handle
        if node.value is not None:
            out["v"] = node.value
        if depth > 0:
            kids = [self._serialize(c, depth - 1) for c in node.children
                    if _DEFAULTS.get(c.type) is None or c.value != _DEFAULTS[c.type]]
            if kids:
                out["s"] = kids
        return out

    # ---- protocol --------------------------------------------------------
    def load_contract(self, type_spec):
        """A copy of the contract chain of `type_spec`; unknown types raise (error frame)."""
        chain = _CONTRACT_CHAINS.get(type_spec)
        if chain is None:
            raise ValueError("unknown type %s" % (type_spec,))
        return {key: list(entries) for key, entries in chain.items()}

    def handle_frame(self, frame):
        replies = []
        for m in frame["m"]:
            try:
                body = self._dispatch(m["c"], m["k"], m["b"])
                replies.append({"c": m["c"], "k": m["k"], "r": 0, "b": body, "t": "rp"})
            except Exception as exc:  # unknown op or bad request -> error frame
                replies.append({"c": m["c"], "k": m["k"], "r": 0, "t": "e",
                                "b": {"isErr": True, "m": "%s: %s" % (type(exc).__name__, exc)}})
        return {"v": "2.3", "p": "box", "n": frame.get("n"), "m": replies}

    def _dispatch(self, channel, key, body):
        if channel == "reg":
            if key == "loadContract":
                return self.load_contract(body)
            raise ValueError("unknown reg key %s" % key)
        if channel != "ssession":
            raise ValueError("unknown channel %s" % channel)
        if key == "make":
            self.sessions.add("sess-1")
            return "sess-1"
        if key == "makessc":
            return {"isReadonly": False}
        if key == "pollchgs":
            events, self._events = self._events, []
            return events
        if key == "del":
            self.sessions.discard(body["id"])
            return None
        if key == "callssc":
            return self._ssc(body["sck"], body.get("scarg"))
        raise ValueError("unknown key %s" % key)

    def _ssc(self, key, arg):
        if key == "loadRoot":
            return {"h": "2", "t": "baja:Station"}
        if key == "loadSlots":
            node = self._resolve(arg["o"])
            tree = self._serialize(node, arg["d"], is_root=True)
            self._events.append({"evs": {"nm": "sync", "ver": "1.0",
                                         "ops": [{"nm": "l", "h": node.handle, "b": tree}]}})
            return None
        if key == "syncTo":
            ops = arg["ops"]
            if len(ops) != 1:
                raise ValueError("exactly one op per syncTo")
            reply = self._sync(ops[0])
            if self.on_sync is not None:
                self.on_sync(ops[0])
            return reply
        if key == "checkLinks":
            return [self._check_link(arg)]
        if key == "invokeAction":
            return self._invoke(arg)
        raise ValueError("unknown ssc key %s" % key)

    @classmethod
    def _unencodable(cls, body):
        """Models live finding 3 (2026-10-01): an add whose body has a `"t": null` node or
        a Status child carrying runtime facets (`;` in its value) is rejected."""
        if "t" in body and body["t"] is None:
            return True
        if (body.get("n") == "status" or body.get("t") == "baja:Status") and \
                ";" in str(body.get("v", "")):
            return True
        return any(cls._unencodable(sub) for sub in body.get("s", []))

    def _sync(self, op):
        nm = op["nm"]
        self._no_double_slashes(op.get("n"))
        if nm == "a":
            parent = self.by_handle[op["h"]]
            # Real N4.14 station (2026-10-01): an add whose body nests COMPONENT children
            # fails with this generic error; plain slots and wsAnnotation are fine.
            if any(self._is_component(sub.get("t")) for sub in op["b"].get("s", [])):
                raise ValueError("Unable to process request. Please contact your system "
                                 "administrator.")
            if self._unencodable(op["b"]):
                raise ValueError("Unable to process request. Please contact your system "
                                 "administrator.")
            if parent.type in _WRITABLES and (
                    op["n"] == "proxyExt" or "ProxyExt" in str(op["b"].get("t"))):
                raise ValueError('Illegal child "%s" for parent "%s".'
                                 % (op["b"].get("t"), parent.type))
            name = self._unique(parent, op["n"])
            node = self._component(parent, name, op["b"]["t"])
            self._fill(node, op["b"])
            if node.type in _WRITABLES:  # the frozen slot, created with its parent
                self._component(node, "proxyExt", "control:NullProxyExt")
            return [{"id": "a", "nn": name}]
        if nm == "s":
            node = self.by_handle[op["h"]]
            parts = op["n"].split("/")
            for part in parts[:-1]:
                node = node.child(part)
            leaf = node.child(parts[-1])
            if leaf is None:
                leaf = _Node(parts[-1], op["b"]["t"])
                node.children.append(leaf)
            self._fill(leaf, op["b"])
            return []
        if nm == "v":
            parent = self.by_handle[op["h"]]
            victim = parent.child(op["n"])
            if victim is None:
                raise KeyError(op["n"])
            parent.children.remove(victim)
            return []
        raise ValueError("unknown sync op %s" % nm)

    def _check_link(self, arg):
        target = self.by_handle[arg["t"]]
        name = self._unique(target, "Link")
        if arg.get("c"):
            link = _Node(name, "baja:Link")
            src = self.by_handle[arg["s"]]
            for slot, val in (("sourceOrd", "h:%s" % src.handle),
                              ("sourceSlotName", arg["ss"]),
                              ("targetSlotName", arg["ts"])):
                link.children.append(_Node(slot, "baja:String", val))
            target.children.append(link)
        return [{"v": True, "r": None, "s": name}]

    def _invoke(self, arg):
        node = self.by_handle[arg["h"]]
        if arg["a"] == "save" and node is self.root:
            self.saves += 1
            if self.config_path and self.save_rewrites:
                with open(self.config_path, "ab") as fh:
                    fh.write(b"saved%d" % self.saves)
                st = os.stat(self.config_path)
                os.utime(self.config_path, (st.st_atime, st.st_mtime + 5))
            return None
        if arg["a"] == "set" and node.type.endswith("NumericWritable"):
            whole = {"t": "baja:StatusNumeric", "s": [
                {"n": "value", "t": "baja:Double", "v": arg["b"]["v"]},
                {"n": "status", "t": "baja:Status", "v": "0"}]}
            leaf = node.child("fallback") or _Node("fallback", whole["t"])
            if leaf not in node.children:
                node.children.append(leaf)
            self._fill(leaf, whole)
            self.invoked.append(("set", arg["h"]))
            return None
        if arg["a"] in ("active", "inactive", "auto"):
            self.invoked.append((arg["a"], arg["h"]))
            return None
        raise ValueError("unsupported action %s" % arg["a"])
