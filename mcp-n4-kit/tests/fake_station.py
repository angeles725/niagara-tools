"""In-memory fake Niagara BOX station for unit tests (plain HTTP on 127.0.0.1)."""
import json
import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

import base64

# Values equal to the type default are omitted from events, like the real station.
_DEFAULTS = {"baja:Status": "0", "baja:Double": "0.0", "baja:Boolean": "false"}


class _Node:
    def __init__(self, name, type_, value=None, handle=None):
        self.name, self.type, self.value, self.handle = name, type_, value, handle
        self.children = []

    def child(self, name):
        return next((c for c in self.children if c.name == name), None)


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

    @staticmethod
    def _is_component(type_):
        return type_.partition(":")[0] != "baja" or type_ == "baja:Folder"

    def _fill(self, node, bson):
        node.type = bson.get("t", node.type)
        node.value = bson.get("v")
        node.children = []
        for sub in bson.get("s", []):
            module = sub["t"].partition(":")[0]
            if module != "baja" or sub["t"] == "baja:Folder":  # a nested component
                child = self._component(node, sub["n"], sub["t"])
            else:
                child = _Node(sub["n"], sub["t"])
                node.children.append(child)
            self._fill(child, sub)

    def _resolve(self, ord_str):
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

    def _sync(self, op):
        nm = op["nm"]
        if nm == "a":
            parent = self.by_handle[op["h"]]
            # Real N4.14 station (2026-10-01): an add whose body nests COMPONENT children
            # fails with this generic error; plain slots and wsAnnotation are fine.
            if any(self._is_component(sub["t"]) for sub in op["b"].get("s", [])):
                raise ValueError("Unable to process request. Please contact your system "
                                 "administrator.")
            name = self._unique(parent, op["n"])
            node = self._component(parent, name, op["b"]["t"])
            self._fill(node, op["b"])
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
