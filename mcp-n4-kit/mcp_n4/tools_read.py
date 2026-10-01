"""Read-only station tools for the MCP server, built on `mcp_n4.box`.

Importing this module has no side effects.
"""
import os
from collections import namedtuple

from . import __version__, box

READ_ONLY = {"readOnlyHint": True, "openWorldHint": False}


class ToolError(Exception):
    """A tool failure whose message is safe to show to the model."""


class Tool(namedtuple("Tool", "name description input_schema handler needs_session annotations")):
    def __new__(cls, name, description, input_schema, handler, needs_session=True,
                annotations=None):
        return super().__new__(cls, name, description, input_schema, handler, needs_session,
                               READ_ONLY if annotations is None else annotations)


class Session:
    def __init__(self, client, station_name, base_url, root_handle, identity_verified=False):
        self.client, self.station_name = client, station_name
        self.base_url, self.root_handle = base_url, root_handle
        #: True only when the session was opened with an `expected_station` that matched.
        self.identity_verified = identity_verified
        self.writes_executed = 0


class Context:
    """Per-process state: at most one active station session."""

    def __init__(self, allow_writes=False, allow_http=False, env=None, client_factory=None):
        self.allow_writes, self.allow_http = allow_writes, allow_http
        self.env = os.environ if env is None else env
        self.client_factory = client_factory or box.BoxClient
        self.session = None
        self.write = None  # tools_write.WriteState, set by the server in writes-allowed mode
        self._secret = None

    @property
    def mode(self):
        return "writes-allowed" if self.allow_writes else "read-only"

    def close(self):
        if self.session is not None:
            self.session.client.close()
            self.session = None

    def scrub(self, text):
        """Defence in depth: never let the secret appear in an output string."""
        return text.replace(self._secret, "***") if self._secret else text


# ---- helpers -------------------------------------------------------------

ROOT_ORD = "station:|slot:/"


def _str(desc, **extra):
    return dict({"type": "string", "description": desc}, **extra)


def _int(desc, minimum, maximum, default):
    return {"type": "integer", "description": desc, "minimum": minimum, "maximum": maximum,
            "default": default}


def _schema(properties, required=()):
    return {"type": "object", "properties": properties, "required": list(required)}


LINK_TYPES = ("baja:Link", "baja:ConversionLink")


def _ord_arg(args):
    ord_str = args.get("ord", ROOT_ORD)
    if not ord_str.startswith("station:"):
        raise ToolError("ord must be a station ORD such as %r, got %r" % (ROOT_ORD, ord_str))
    return ord_str


def _is_component(node):
    """Heuristic: a child with no simple value that is not a link or a Status complex."""
    kind = node.get("t", "")
    return "v" not in node and kind not in LINK_TYPES and not kind.startswith("baja:Status")


def _has_component_children(nodes, path):
    return any(_is_component(n) for n in box.children(nodes, path).values())


# ---- n4_connect ----------------------------------------------------------

def n4_connect(ctx, args):
    prefix = args.get("credential_env", "MCP_N4")
    user, secret = ctx.env.get(prefix + "_USER"), ctx.env.get(prefix + "_PASSWORD")
    if not user or not secret:
        raise ToolError("credentials missing: set env vars %s_USER and %s_PASSWORD"
                        % (prefix, prefix))
    ctx.close()  # a failed connect must never leave the old session active
    ctx._secret = secret
    client = ctx.client_factory(args["base_url"], user, secret,
                                insecure_tls=args.get("insecure_tls", False),
                                allow_http=ctx.allow_http)
    try:
        root = client.open()
        root_h = root.get("h") if isinstance(root, dict) else None
        if not root_h:
            raise box.BoxError("station returned no root handle")
        nodes = client.load_tree(ROOT_ORD, depth=1, handle=root_h)
    except BaseException:
        client.close()
        raise
    name = nodes.get("stationName", {}).get("v")
    expected = args.get("expected_station")
    if expected is not None and expected != name:
        client.close()
        raise ToolError("station identity mismatch: expected %r but connected to %r"
                        % (expected, name))
    ctx.session = Session(client, name, client.base_url, root_h,
                          identity_verified=expected is not None)
    return {"station_name": name, "base_url": client.base_url, "root_handle": root_h,
            "mode": ctx.mode}


# ---- n4_describe_session -------------------------------------------------

def n4_describe_session(ctx, args):
    sess = ctx.session
    return {"connected": sess is not None,
            "station_name": sess.station_name if sess else None,
            "base_url": sess.base_url if sess else None,
            "mode": ctx.mode, "server_version": __version__}


# ---- n4_navigate ---------------------------------------------------------

def _tree_entries(nodes, path, levels):
    entries = []
    for name, node in box.children(nodes, path).items():
        child_path = path + "/" + name if path else name
        entry = {"name": name, "type": node.get("t"), "handle": node.get("h"),
                 "has_children": _has_component_children(nodes, child_path)}
        if levels > 1:
            entry["children"] = _tree_entries(nodes, child_path, levels - 1)
        entries.append(entry)
    return entries


def n4_navigate(ctx, args):
    ord_str, depth = _ord_arg(args), args.get("depth", 1)
    # one extra level so has_children is known for the deepest children returned
    nodes = ctx.session.client.load_tree(ord_str, depth=depth + 1)
    return {"ord": ord_str, "children": _tree_entries(nodes, "", depth)}


# ---- n4_read_slots -------------------------------------------------------

def _slot_entry(nodes, name):
    node = nodes[name]
    kind = node.get("t")
    entry = {"name": name, "type": kind, "value": node.get("v"), "status": None}
    if not (kind or "").startswith("baja:Status"):
        return entry
    try:
        entry.update(box.status_value(nodes, name))
    except ValueError:  # a Status type without a known default (e.g. StatusString)
        kids = box.children(nodes, name)
        entry.update(value=kids.get("value", {}).get("v"),
                     status=kids.get("status", {}).get("v", "0"))
    flags = box.parse_status(entry["status"])
    entry.update(status_ok=flags["ok"], status_null=flags["null"])
    return entry


def n4_read_slots(ctx, args):
    ord_str = _ord_arg(args)
    # depth 2: slots, plus the value/status children of Status complexes
    nodes = ctx.session.client.load_tree(ord_str, depth=2)
    return {"ord": ord_str,
            "slots": [_slot_entry(nodes, name) for name in box.children(nodes, "")]}


# ---- links and dangling outputs -------------------------------------------

def _level(path):
    return path.count("/") + 1 if path else 0


def _abs_path(ord_str, path):
    base = ord_str.split("slot:", 1)[1] if "slot:" in ord_str else "/"
    base = base.rstrip("/")
    return base + "/" + path if path else base or "/"


def _scan(ctx, args):
    """Load `ord` and return (ord, nodes, links) for components up to `depth` levels down.

    A link sits one level below its target component and its slots one more, so
    the load goes `depth + 2` deep and links of deeper components are dropped
    (their slots would not be loaded).
    """
    ord_str, depth = _ord_arg(args), args.get("depth", 2)
    nodes = ctx.session.client.load_tree(ord_str, depth=depth + 2)
    by_handle = {n["h"]: _abs_path(ord_str, p) for p, n in nodes.items() if n.get("h")}
    links = []
    for path, node in nodes.items():
        if node.get("t") not in LINK_TYPES or _level(path) - 1 > depth:
            continue
        slots = {k: v.get("v") for k, v in box.children(nodes, path).items()}
        source_ord = slots.get("sourceOrd")
        handle = (source_ord or "").rsplit("|", 1)[-1]
        target = path.rpartition("/")[0]
        links.append({"target_path": _abs_path(ord_str, target),
                      "link_name": path.rpartition("/")[2],
                      "source_ord": source_ord,
                      "source_path": by_handle.get(handle[2:]) if handle.startswith("h:") else None,
                      "source_slot": slots.get("sourceSlotName"),
                      "target_slot": slots.get("targetSlotName")})
    return ord_str, nodes, links, depth


def n4_list_links(ctx, args):
    ord_str, _, links, _ = _scan(ctx, args)
    return {"ord": ord_str, "links": links}


def n4_find_dangling_outputs(ctx, args):
    ord_str, nodes, links, depth = _scan(ctx, args)
    used = {(l["source_path"], l["source_slot"]) for l in links}
    dangling = []
    for path, node in nodes.items():
        if _level(path) > depth or node.get("t") in LINK_TYPES:
            continue
        if "out" in box.children(nodes, path):
            abs_path = _abs_path(ord_str, path)
            if (abs_path, "out") not in used:
                dangling.append({"path": abs_path, "type": node.get("t")})
    return {"ord": ord_str, "dangling": dangling,
            "scope_note": "Only links inside the scanned subtree are seen; an out slot "
                          "linked from outside the subtree is reported as dangling."}


TOOLS = [
    Tool("n4_connect",
         "Open a session to a Niagara N4 station (replaces any previous session). "
         "Credentials come from env vars <credential_env>_USER and <credential_env>_PASSWORD, "
         "never from arguments. Pass expected_station to refuse the wrong station.",
         _schema({"base_url": _str("Station URL, https://host[:port]"),
                  "credential_env": _str("Env var prefix for credentials", default="MCP_N4"),
                  "insecure_tls": {"type": "boolean", "default": False,
                                   "description": "Skip TLS certificate verification "
                                                  "(self-signed station certificates)"},
                  "expected_station": _str("Required stationName; mismatch closes the session")},
                 ["base_url"]),
         n4_connect, needs_session=False),
    Tool("n4_describe_session",
         "Report whether a station session is active, which station, and the server mode. "
         "Works without a session.",
         _schema({}), n4_describe_session, needs_session=False),
    Tool("n4_navigate",
         "List the children of a component (name, type, handle, has_children). "
         "depth > 1 nests grandchildren under a 'children' key.",
         _schema({"ord": _str("Station ORD, default station:|slot:/", default=ROOT_ORD),
                  "depth": _int("Levels to list (1-3)", 1, 3, 1)}),
         n4_navigate),
    Tool("n4_read_slots",
         "Read all slots of one component as {name, type, value, status}. Status complexes "
         "add status_ok/status_null; omitted values take the type default.",
         _schema({"ord": _str("Component ORD, e.g. station:|slot:/Folder/Pump")}, ["ord"]),
         n4_read_slots),
    Tool("n4_list_links",
         "List baja:Link / baja:ConversionLink links of components up to depth levels below "
         "ord. source_path is null when the source handle is outside the loaded subtree.",
         _schema({"ord": _str("Subtree root ORD"),
                  "depth": _int("Component levels below ord to scan (1-4)", 1, 4, 2)},
                 ["ord"]),
         n4_list_links),
    Tool("n4_find_dangling_outputs",
         "List components (up to depth levels below ord) with an 'out' slot that no link in "
         "the subtree uses as a source. Links from outside the subtree are not seen.",
         _schema({"ord": _str("Subtree root ORD"),
                  "depth": _int("Component levels below ord to scan (1-4)", 1, 4, 2)},
                 ["ord"]),
         n4_find_dangling_outputs),
]
