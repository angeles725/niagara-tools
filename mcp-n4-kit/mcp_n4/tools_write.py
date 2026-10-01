"""Guarded write tools for the MCP server (registered only with `--allow-writes`).

Every call runs the same pipeline: identity check, session write budget, write
scope, plan (reads only), then either a dry run that issues a single-use token or
an execution that spends it, journals the inverse ops, reads the result back and
audits the call. Importing this module has no side effects.
"""
import datetime
import hashlib
import math
import os
import re
import time
import uuid
from collections import namedtuple

from . import box, safety
from .tools_read import LINK_TYPES, Tool, ToolError, _schema, _str

DEFAULT_STATE_DIR = "~/.local/state/mcp-n4"
WRITE = {"readOnlyHint": False, "destructiveHint": False, "openWorldHint": False}
DESTRUCTIVE = {"readOnlyHint": False, "destructiveHint": True, "openWorldHint": False}
SNAPSHOT_DEPTH = 3
#: Actions `n4_invoke_action` may call. Destructive ones (emergency*, save, restart)
#: arrive later under a separate class.
ALLOWED_ACTIONS = ("set", "active", "inactive", "auto")
_BATCH_ID = re.compile(r"^[0-9a-f]{32}$")
VALUE_TYPES = ("baja:Double", "baja:Boolean", "baja:String",
               "baja:StatusNumeric", "baja:StatusBoolean")
_NAME = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
_TYPE_SPEC = re.compile(r"^[A-Za-z][A-Za-z0-9_]*:[A-Za-z][A-Za-z0-9_]*$")
_STATUS = re.compile(r"^[0-9a-fA-F]+(;.*)?$")

#: What a handler's helpers pass around: the exact ops and inverse (both hashed into
#: the token), human notes, and private data the execute step needs.
Planned = namedtuple("Planned", "ops inverse notes data")


class WriteState:
    """Per-process write machinery: scope, tokens, journal, audit and budget."""

    def __init__(self, scopes=(), state_dir=None, token_ttl=300, max_writes=200,
                 clock=time.time, station_homes=None):
        state_dir = os.path.expanduser(state_dir or DEFAULT_STATE_DIR)
        safety.check_state_dir(state_dir)  # refuse a dir we do not own; never chmod it
        self.scope = safety.WriteScope(scopes)
        self.tokens = safety.ConfirmationTokens(token_ttl, clock)
        self.journal, self.audit = safety.Journal(state_dir), safety.AuditLog(state_dir)
        self.max_writes = max_writes
        #: station NAME -> directory holding config.bog (operator policy, for n4_save_station)
        self.station_homes = dict(station_homes or {})
        self.save_timeout, self.save_interval = 30.0, 0.5
        self.sleep, self.monotonic = time.sleep, time.monotonic


# ---- value helpers -------------------------------------------------------

def _coerce(value_type, value, status="0"):
    """Validate `value` for `value_type` and return its normalized form."""
    kind = {"baja:StatusNumeric": "baja:Double", "baja:StatusBoolean": "baja:Boolean"}.get(
        value_type, value_type)
    if kind == "baja:Double":
        if isinstance(value, bool) or not isinstance(value, (int, float)) \
                or not math.isfinite(value):
            raise ToolError("value must be a finite number for %s" % value_type)
        norm = float(value)
    elif kind == "baja:Boolean":
        if not isinstance(value, bool):
            raise ToolError("value must be true or false for %s" % value_type)
        norm = value
    elif kind == "baja:String":
        if not isinstance(value, str):
            raise ToolError("value must be a string for baja:String")
        norm = value
    else:
        raise ToolError("value_type must be one of %s" % ", ".join(VALUE_TYPES))
    if not value_type.startswith("baja:Status"):
        return norm
    if not isinstance(status, str) or not _STATUS.match(status):
        raise ToolError("status must be a hex status string such as '0'")
    return {"value": norm, "status": status}


def _to_bson(value_type, norm):
    if value_type == "baja:StatusNumeric":
        return box.bson_status_numeric(norm["value"], norm["status"])
    if value_type == "baja:StatusBoolean":
        return box.bson_status_boolean(norm["value"], norm["status"])
    if value_type == "baja:Double":
        return box.bson_double(norm)
    if value_type == "baja:Boolean":
        return box.bson_bool(norm)
    return {"nm": "p", "t": "baja:String", "v": norm}


def _observe(nodes, slot, value_type):
    """Current value of `slot` in the same normalized form `_coerce` returns."""
    if value_type.startswith("baja:Status"):
        sv = box.status_value(nodes, slot)
        value = float(sv["value"]) if value_type.endswith("Numeric") else sv["value"] == "true"
        return {"value": value, "status": sv["status"]}
    raw = nodes.get(slot, {}).get("v")  # a plain slot at its type default is omitted
    if value_type == "baja:Double":
        return float(raw or "0.0")
    if value_type == "baja:Boolean":
        return raw == "true"
    return raw or ""


def _handle(client, ord_str, depth):
    """Load `ord_str`; return `(handle, nodes)`."""
    nodes = client.load_tree(ord_str, depth=depth)
    handle = nodes.get("", {}).get("h")
    if not handle:
        raise ToolError("station returned no handle for %s" % ord_str)
    return handle, nodes


def _name(label, value):
    if not isinstance(value, str) or not _NAME.match(value):
        raise ToolError("%s must be a plain slot name (letters, digits, underscore), got %r"
                        % (label, value))
    return value


def _data(planned):
    """A plan's private data as a dict, whatever a handler returned."""
    return planned.data if isinstance(planned.data, dict) else {}


def _send(client, op):
    if "ssc" in op:
        return client.ssc(op["ssc"], op["arg"])
    reply = client.sync(op)
    if op.get("nm") == "v":  # cached handles of removed components are stale now
        client.invalidate_handles()
    return reply


# ---- n4_create_component -------------------------------------------------

def _wire_sheet(ws):
    if ws is None:
        return None
    x, y, w = (ws.get(k, 8 if k == "w" else None) for k in ("x", "y", "w"))
    if any(isinstance(v, bool) or not isinstance(v, int) for v in (x, y, w)) or w < 1:
        raise ToolError("wire_sheet needs integer x, y and an optional integer w >= 1")
    return box.ws_annotation(x, y, w)


def _create_component_plan(client, args):
    name = _name("name", args["name"])
    type_ = args["type"]
    if not _TYPE_SPEC.match(type_):
        raise ToolError("type must look like module:Type, got %r" % type_)
    ws = _wire_sheet(args.get("wire_sheet"))
    parent_h, _ = _handle(client, args["parent_ord"], 1)
    body = {"nm": "p", "t": type_, "s": [ws] if ws else []}
    return Planned(
        [{"nm": "a", "h": parent_h, "n": name, "b": body}],
        [{"nm": "v", "h": parent_h, "n": name}],
        ["The station may rename on collision; the real inverse uses the assigned name."],
        {"parent_h": parent_h, "ws": ws["v"] if ws else None,
         "targets": {args["parent_ord"]: parent_h}})


def _create_component_inverse(planned, replies):
    reply = replies[0][0] if isinstance(replies[0], list) and replies[0] else {}
    nn = reply.get("nn", planned.ops[0]["n"]) if isinstance(reply, dict) else planned.ops[0]["n"]
    return [{"nm": "v", "h": planned.data["parent_h"], "n": nn}]


def _create_component_readback(client, args, planned, replies, inverse):
    nn = inverse[0]["n"]
    requested = {"name": nn, "type": args["type"], "wsAnnotation": planned.data["ws"]}
    nodes = client.load_tree(args["parent_ord"], depth=2)
    observed = None
    if nn in nodes:
        observed = {"name": nn, "type": nodes[nn].get("t"),
                    "wsAnnotation": nodes.get(nn + "/wsAnnotation", {}).get("v")}
    return requested, replies[0][0] if replies[0] else None, observed, \
        "verified" if observed == requested else "mismatch"


# ---- n4_set_slot ---------------------------------------------------------

def _set_slot_plan(client, args):
    slot, value_type = args["slot"], args["value_type"]
    if not isinstance(slot, str) or not slot:
        raise ToolError("slot must be a non-empty slot name")
    if "/" in slot and slot.split("/")[-1] in ("value", "status"):
        raise ToolError("refusing to write %r on its own: Status slots are written whole "
                        "(value and status together) because a value-only write leaves the "
                        "status null (B1199 section 1199.4); pass the Status slot itself with "
                        "value_type baja:StatusNumeric or baja:StatusBoolean" % slot)
    for part in slot.split("/"):
        _name("slot", part)
    if value_type not in VALUE_TYPES:
        raise ToolError("value_type must be one of %s" % ", ".join(VALUE_TYPES))
    if "status" in args and not value_type.startswith("baja:Status"):
        raise ToolError("status only applies to baja:StatusNumeric / baja:StatusBoolean")
    requested = _coerce(value_type, args["value"], args.get("status", "0"))
    h, nodes = _handle(client, args["ord"], 2)
    if slot not in nodes:
        raise ToolError("slot %r not found on %s" % (slot, args["ord"]))
    if nodes[slot].get("t") != value_type:
        raise ToolError("slot %r is %s, not %s" % (slot, nodes[slot].get("t"), value_type))
    previous = _observe(nodes, slot, value_type)
    return Planned([{"nm": "s", "h": h, "n": slot, "b": _to_bson(value_type, requested)}],
                   [{"nm": "s", "h": h, "n": slot, "b": _to_bson(value_type, previous)}],
                   [], {"requested": requested, "targets": {args["ord"]: h}})


def _set_slot_readback(client, args, planned, replies, inverse):
    _, nodes = _handle(client, args["ord"], 2)
    observed = _observe(nodes, args["slot"], args["value_type"])
    requested = planned.data["requested"]
    return requested, replies[0], observed, "verified" if observed == requested else "mismatch"


# ---- n4_invoke_action ----------------------------------------------------

_FALLBACK = {"NumericWritable": "baja:StatusNumeric", "BooleanWritable": "baja:StatusBoolean"}
_ARG_TYPES = {"baja:StatusNumeric": "baja:Double", "baja:StatusBoolean": "baja:Boolean"}


def _fallback_type(nodes):
    if "fallback" in nodes:
        return nodes["fallback"].get("t")
    kind = nodes.get("", {}).get("t", "").rpartition(":")[2]
    return _FALLBACK.get(kind)


def _invoke_plan(client, args):
    action = args["action"]
    if action not in ALLOWED_ACTIONS:
        hint = "; use n4_save_station" if action == "save" else ""
        raise ToolError("action %r is not allowed in this version (allowed: %s)%s"
                        % (action, ", ".join(ALLOWED_ACTIONS), hint))
    h, nodes = _handle(client, args["ord"], 2)
    arg = {"h": h, "a": action}
    if action != "set":
        if "arg" in args or "arg_type" in args:
            raise ToolError("action %r takes no arg" % action)
        return Planned([{"ssc": "invokeAction", "arg": arg}], [],
                       ["no inverse: %s changes the control state and is not restored "
                        "automatically" % action], {"h": h, "targets": {args["ord"]: h}})
    fb_type = _fallback_type(nodes)
    if fb_type not in _ARG_TYPES:
        raise ToolError("cannot tell the previous fallback of %s; refusing set" % args["ord"])
    if args.get("arg_type") != _ARG_TYPES[fb_type] or "arg" not in args:
        raise ToolError("set on this component needs arg and arg_type %s" % _ARG_TYPES[fb_type])
    value = _coerce(_ARG_TYPES[fb_type], args["arg"])
    arg["b"] = _to_bson(_ARG_TYPES[fb_type], value)
    previous = _observe(nodes, "fallback", fb_type) if "fallback" in nodes else \
        {"value": 0.0 if fb_type.endswith("Numeric") else False, "status": "0"}
    return Planned([{"ssc": "invokeAction", "arg": arg}],
                   [{"nm": "s", "h": h, "n": "fallback", "b": _to_bson(fb_type, previous)}],
                   [], {"h": h, "value": value, "fb_type": fb_type,
                        "targets": {args["ord"]: h}})


def _invoke_readback(client, args, planned, replies, inverse):
    if args["action"] != "set":
        return {"action": args["action"]}, replies[0], None, "unverified"
    _, nodes = _handle(client, args["ord"], 2)
    observed = {"fallback": _observe(nodes, "fallback", planned.data["fb_type"])["value"]}
    requested = {"fallback": planned.data["value"]}
    return requested, replies[0], observed, "verified" if observed == requested else "mismatch"


# ---- n4_create_link ------------------------------------------------------

def _link_plan(client, args):
    ss, ts = _name("source_slot", args["source_slot"]), _name("target_slot", args["target_slot"])
    src_h, _ = _handle(client, args["source_ord"], 1)
    tgt_h, tgt_nodes = _handle(client, args["target_ord"], 1)
    existing = sorted(k for k, v in box.children(tgt_nodes, "").items()
                      if v.get("t") == "baja:Link")
    return Planned([{"ssc": "checkLinks",
                     "arg": {"s": src_h, "ss": ss, "t": tgt_h, "ts": ts, "c": True}}],
                   [{"nm": "v", "h": tgt_h, "n": "<link name assigned by the station>"}],
                   [], {"src_h": src_h, "tgt_h": tgt_h, "existing_links": existing,
                    "targets": {args["target_ord"]: tgt_h}})


def _link_result(replies):
    res = replies[0]
    if not (isinstance(res, list) and res and isinstance(res[0], list) and res[0]
            and isinstance(res[0][0], dict)):
        raise box.BoxError("malformed checkLinks reply", box.CHANNEL, "checkLinks")
    return res[0][0]


def _link_inverse(planned, replies):
    """`[]` for an explicit refusal; raises BoxError when the reply cannot be trusted."""
    result = _link_result(replies)
    if "v" not in result:
        raise box.BoxError("ambiguous checkLinks reply: no verdict", box.CHANNEL, "checkLinks")
    if not result["v"]:
        return []  # the station refused the link: nothing changed
    if not result.get("s"):
        raise box.BoxError("ambiguous checkLinks reply: link accepted but not named",
                           box.CHANNEL, "checkLinks")
    return [{"nm": "v", "h": planned.data["tgt_h"], "n": result["s"]}]


def _link_slots(nodes, name):
    slots = {k: v.get("v") for k, v in box.children(nodes, name).items()}
    return {"link": name, "sourceOrd": slots.get("sourceOrd"),
            "sourceSlotName": slots.get("sourceSlotName"),
            "targetSlotName": slots.get("targetSlotName")}


def _link_recover(client, args, planned, replies):
    """After an ambiguous reply: find the new link by its source and target slots.

    Returns `(observed, inverse)`; `(None, [])` unless exactly one new link matches.
    """
    _, nodes = _handle(client, args["target_ord"], 2)
    found = [_link_slots(nodes, name) for name, node in box.children(nodes, "").items()
             if node.get("t") == "baja:Link" and name not in planned.data["existing_links"]]
    found = [f for f in found
             if (f["sourceOrd"] or "").rsplit("|", 1)[-1] == "h:" + planned.data["src_h"]
             and f["sourceSlotName"] == args["source_slot"]
             and f["targetSlotName"] == args["target_slot"]]
    if len(found) != 1:
        return None, []
    return found[0], [{"nm": "v", "h": planned.data["tgt_h"], "n": found[0]["link"]}]


def _link_readback(client, args, planned, replies, inverse):
    result = _link_result(replies)
    if not inverse:  # refused by the station (see _link_inverse)
        return None, result, None, "failed"
    name = inverse[0]["n"]
    requested = {"link": name, "sourceOrd": "h:" + planned.data["src_h"],
                 "sourceSlotName": args["source_slot"], "targetSlotName": args["target_slot"]}
    _, nodes = _handle(client, args["target_ord"], 2)
    observed = _link_slots(nodes, name) if name in nodes else None

    def tail(d):  # the station may prefix the handle ORD; compare only `h:<handle>`
        return d and dict(d, sourceOrd=(d["sourceOrd"] or "").rsplit("|", 1)[-1])
    return requested, result, observed, "verified" if tail(observed) == tail(requested) \
        else "mismatch"


# ---- n4_remove_component -------------------------------------------------

def _snapshot(node, path, links, handles):
    """Data-only copy of a loaded subtree: type, slot values, nested slots.

    Handles are not kept; links are collected in `links` and their owner component
    path is recorded, because a link cannot be re-created without both ends.
    """
    if node.get("h"):
        handles[node["h"]] = path
    out = {"nm": "p", "t": node.get("t")}
    if path and "n" in node:
        out["n"] = node["n"]
    if node.get("v") is not None:
        out["v"] = node["v"]
    kids = []
    for child in node.get("s", []):
        child_path = child["n"] if not path else path + "/" + child["n"]
        if child.get("t") in LINK_TYPES:
            slots = {k.get("n"): k.get("v") for k in child.get("s", [])}
            links.append({"target_path": path, "source_ord": slots.get("sourceOrd"),
                          "source_slot": slots.get("sourceSlotName"),
                          "target_slot": slots.get("targetSlotName")})
            continue
        kids.append(_snapshot(child, child_path, links, handles))
    if kids:
        out["s"] = kids
    return out


#: How deep the optional outgoing-link scan loads below `link_scan_ord`.
LINK_SCAN_DEPTH = 6


def _has_outputs(node):
    """True when any descendant slot is a Status value, i.e. something links can read from."""
    return any(str(c.get("t", "")).startswith("baja:Status") or _has_outputs(c)
               for c in node.get("s", []))


def _outgoing_links(node, path, handles, found):
    """Links stored on a component outside the removed subtree whose source is inside it."""
    here = node.get("h")
    for child in node.get("s", []):
        if child.get("t") in LINK_TYPES:
            slots = {k.get("n"): k.get("v") for k in child.get("s", [])}
            tail = (slots.get("sourceOrd") or "").rsplit("|", 1)[-1]
            if tail.startswith("h:") and tail[2:] in handles and here not in handles:
                found.append({"path": path, "target_slot": slots.get("targetSlotName"),
                              "source_path": handles[tail[2:]],
                              "source_slot": slots.get("sourceSlotName")})
        else:
            _outgoing_links(child, path + "/" + child["n"] if path else child["n"],
                            handles, found)
    return found


def _scan_outgoing(client, scan_ord, handles):
    nodes = client.load_tree(scan_ord, depth=LINK_SCAN_DEPTH)
    found = _outgoing_links(nodes[""], "", handles, [])
    return sorted(({"target": scan_ord.rstrip("/") + ("/" + f["path"] if f["path"] else ""),
                    "target_slot": f["target_slot"], "source_path": f["source_path"],
                    "source_slot": f["source_slot"]} for f in found),
                  key=lambda f: (f["target"], f["target_slot"], f["source_path"]))


def _remove_plan(client, args):
    name, parent_ord = _name("name", args["name"]), args["parent_ord"]
    parent_h, parent_nodes = _handle(client, parent_ord, 1)
    if name not in parent_nodes:
        raise ToolError("component %r not found under %s" % (name, parent_ord))
    _, nodes = _handle(client, parent_ord.rstrip("/") + "/" + name, SNAPSHOT_DEPTH)
    links, handles = [], {}
    body = _snapshot(nodes[""], "", links, handles)
    inverse, notes = [{"nm": "a", "h": parent_h, "n": name, "b": body}], [
        "Automatic re-creation (n4_rollback) is limited to this snapshot: type, plain slots "
        "and wsAnnotation to depth %d; it is not a full restore." % SNAPSHOT_DEPTH]
    for link in links:
        tail = (link["source_ord"] or "").rsplit("|", 1)[-1]
        source = handles.get(tail[2:]) if tail.startswith("h:") else None
        if source is None:
            notes.append("link into %r (%s <- %s) comes from outside the removed subtree: "
                         "not restored" % (link["target_path"] or name, link["target_slot"],
                                           link["source_slot"]))
            continue
        inverse.append({"relink": {"source_path": source, "source_slot": link["source_slot"],
                                   "target_path": link["target_path"],
                                   "target_slot": link["target_slot"]}})
    data = {"targets": {parent_ord: parent_h}}
    scan_ord = args.get("link_scan_ord")
    if scan_ord is not None:
        found = _scan_outgoing(client, scan_ord, handles)
        if found:
            data["outgoing_links_broken"] = found
            notes.append("scanned %s to depth %d: the remove will break %d link(s) from this "
                         "subtree into components outside it (see outgoing_links_broken); "
                         "they are not restored" % (scan_ord, LINK_SCAN_DEPTH, len(found)))
        else:
            notes.append("scanned %s to depth %d: no outgoing links from this subtree found"
                         % (scan_ord, LINK_SCAN_DEPTH))
    elif _has_outputs(nodes[""]):
        notes.append("outgoing links to components outside the removed subtree are stored on "
                     "those targets: they cannot be detected from the subtree load and will "
                     "be broken by the remove; pass "
                     "link_scan_ord (inside the write scope) to scan a wider root")
    return Planned([{"nm": "v", "h": parent_h, "n": name}], inverse, notes, data)


def _remove_readback(client, args, planned, replies, inverse):
    _, nodes = _handle(client, args["parent_ord"], 1)
    gone = args["name"] not in nodes
    return {"removed": args["name"]}, replies[0], {"gone": gone}, \
        "verified" if gone else "mismatch"


# ---- n4_rollback ---------------------------------------------------------

def _rollback_plan(client, args, ctx):
    write, sess, bid = ctx.write, ctx.session, args["batch_id"]
    if not isinstance(bid, str) or not _BATCH_ID.match(bid):
        raise ToolError("batch_id must be a 32-character hex batch id")
    view = write.journal.read(bid)
    if view is None:
        raise ToolError("unknown batch %s: it is not in the journal" % bid)
    if view.get("rollback_of"):
        raise ToolError("batch %s is itself a rollback of %s: roll forward by repeating the "
                        "original write instead" % (bid, view["rollback_of"]))
    if view["state"] == "in-doubt":
        raise ToolError("batch %s is in-doubt (the station may or may not have applied it); "
                        "rolling it back blindly could do harm, decide by hand. Journaled "
                        "intent: %s" % (bid, safety.canonical(
                            {k: view.get(k) for k in ("tool", "ops", "inverse_plan",
                                                      "station_name", "targets")})))
    if view.get("station_name") != sess.station_name:
        raise ToolError("batch %s was recorded on station %r, this session is on %r"
                        % (bid, view.get("station_name"), sess.station_name))
    for earlier in write.journal.rollbacks_of(bid):
        if earlier["state"] == "in-doubt":
            raise ToolError("a rollback of batch %s (%s) is in-doubt: inspect the station"
                            % (bid, earlier["batch_id"]))
        if earlier.get("verdict") != "failed":
            raise ToolError("batch %s was already rolled back by %s" % (bid, earlier["batch_id"]))
        if earlier.get("readback_failed"):  # the station accepted its ops: a retry would repeat them
            raise ToolError("a rollback of batch %s (%s) is in-doubt: the station accepted its "
                            "ops but the read-back failed, so it may already be applied; "
                            "inspect the station, do not retry blindly. Accepted: %s"
                            % (bid, earlier["batch_id"], safety.canonical(earlier.get("accepted"))))
    inverse = view.get("inverse") or []
    ops = [e for e in inverse if isinstance(e, dict) and "nm" in e]
    relinks = [_relink_spec(e["relink"]) for e in inverse
               if isinstance(e, dict) and "relink" in e]
    if not ops:
        raise ToolError("batch %s has no inverse recorded (verdict %s): nothing to roll back"
                        % (bid, view.get("verdict")))
    targets = view.get("targets") or {}
    if not targets or any(op.get("h") not in targets.values() for op in ops):
        raise ToolError("batch %s does not record which components its inverse touches: "
                        "refusing" % bid)
    for ord_str, recorded in targets.items():
        write.scope.check(ord_str)  # today's scope, not the one the batch ran under
        current, _ = _handle(client, ord_str, 1)
        if current != recorded:
            raise ToolError("%s now has handle %s but the batch recorded %s: the component "
                            "was replaced, refusing" % (ord_str, current, recorded))
    components = []
    for i, op in enumerate(ops):  # one add per component: nested bodies are rejected live
        if op["nm"] == "a":
            ops[i] = dict(op)
            ops[i]["b"], specs = _flatten(i, op["b"])
            components += specs
    own = [{"nm": "v", "h": op["h"], "n": op["n"]} for op in ops if op["nm"] == "a"]
    planned_links = len(relinks) * len(own)
    need = 1 + len(components) + planned_links
    if sess.writes_executed + need > write.max_writes:
        raise ToolError("write budget too small: this rollback needs %d write(s) (1 batch + %d "
                        "component(s) + %d relink(s)) but only %d remain (--max-writes %d)"
                        % (need, len(components), planned_links,
                           write.max_writes - sess.writes_executed, write.max_writes))
    for spec in components:  # every re-created component, under today's scope
        top = ops[spec["top"]]
        write.scope.check(_end_ord(_ord_of(targets, top["h"]) + "/" + top["n"],
                                   _spec_path(spec)))
    for op in own:  # both ends of every relink, under today's scope (name as requested)
        for spec in relinks:
            _check_ends(write.scope, "%s/%s" % (_ord_of(targets, op["h"]), op["n"]), spec)
    notes = ["rolls back batch %s (%s)" % (bid, view.get("tool"))]
    if relinks:
        notes.append("then re-creates %d link(s) between restored components, where both "
                     "ends exist" % len(relinks))
    if components:
        notes.append("re-creates %d nested component(s) one add per component, parents "
                     "first: the station rejects an add that nests components" % len(components))
    if not own or len(own) != len(ops):
        notes.append("the rollback itself has no automatic inverse for slot restores or "
                     "removals")
    return Planned(ops, own if len(own) == len(ops) else [], notes,
                   {"rollback_of": bid, "targets": targets, "relinks": relinks,
                    "components": components,
                    "ord_of": {h: o for o, h in targets.items()}})


def _spec_path(spec):
    return spec["n"] if not spec["parent_path"] else spec["parent_path"] + "/" + spec["n"]


def _run_components(sess, write, batch_id, planned, replies):
    """Create the nested components of a restored subtree, one add op each, parents first.

    Each op is a member of the rollback's own batch: scope-checked, journaled (write-ahead,
    one `component-intent` record) before it is sent, and counted against the write budget.
    The parent's NEW handle is loaded through the client just before its children are
    added. Returns the created paths per top op; a failure mid-way raises the batch
    in-doubt error listing what was created so far.
    """
    data = planned.data
    base = {i: "%s/%s" % (data["ord_of"][op["h"]], _assigned_name(op, reply))
            for i, (op, reply) in enumerate(zip(planned.ops, replies)) if op["nm"] == "a"}
    actual = {(i, ""): base[i] for i in base}  # (top, spec path) -> assigned ORD
    created = []
    for spec in data["components"]:
        parent = actual[(spec["top"], spec["parent_path"])]
        try:
            write.scope.check(parent + "/" + spec["n"])
            parent_h, _ = _handle(sess.client, parent, 1)
        except Exception as exc:
            raise _partial(batch_id, base, created, exc) from None
        op = {"nm": "a", "h": parent_h, "n": spec["n"], "b": spec["b"]}
        try:
            write.journal.append({"batch_id": batch_id, "ts": _now(),
                                  "phase": "component-intent", "ops": [op]})
        except OSError:
            raise _partial(batch_id, base, created, "the component intent could not be "
                           "written, nothing more was sent") from None
        sess.writes_executed += 1
        try:
            reply = _send(sess.client, op)
        except Exception as exc:
            raise _partial(batch_id, base, created, exc) from None
        name = _assigned_name(op, reply)
        actual[(spec["top"], _spec_path(spec))] = parent + "/" + name
        created.append(parent + "/" + name)
    return created


def _partial(batch_id, base, created, exc):
    """The in-doubt error of a rollback that stopped mid-way, listing what exists now."""
    err = _in_doubt(batch_id, exc if isinstance(exc, Exception) else ToolError(exc))
    done = sorted(base.values()) + created
    err.args = ("%s. Components already created by this batch (remove the top-level one "
                "to clean up): %s" % (err.args[0], ", ".join(done)),)
    return err


def _is_component(type_):
    """True for a type the station must create with its own add op (not a plain slot value)."""
    return isinstance(type_, str) and (
        type_.partition(":")[0] != "baja" or type_ == "baja:Folder")


def _own_slots(body):
    """`body` without its component children: plain slot values and wsAnnotation only."""
    out = {k: v for k, v in body.items() if k not in ("s", "n")}
    kept = [c for c in body.get("s", []) if not _is_component(c.get("t"))]
    if kept:
        out["s"] = kept
    return out


def _flatten(top, body):
    """Split a nested snapshot body into `(top_body, specs)`, one spec per descendant.

    The real station rejects an add whose body nests components, so each component
    becomes its own add, parents before children (breadth-first). Specs carry paths
    relative to the top component, never handles, so the plan hash stays stable.
    """
    specs, queue = [], [("", body)]
    while queue:
        path, node = queue.pop(0)
        for kid in node.get("s", []):
            if _is_component(kid.get("t")):
                name = _name("component name", kid.get("n"))
                specs.append({"top": top, "parent_path": path, "n": name,
                              "b": _own_slots(kid)})
                queue.append((name if not path else path + "/" + name, kid))
    return _own_slots(body), specs


def _relink_spec(spec):
    """Validate a journaled relink spec (paths and slot names only, never handles)."""
    keys = ("source_path", "source_slot", "target_path", "target_slot")
    if not isinstance(spec, dict) or any(not isinstance(spec.get(k), str) for k in keys):
        raise ToolError("batch records a malformed relink: refusing")
    for key in keys:
        for part in (spec[key].split("/") if spec[key] else []):
            _name(key, part)
    return {k: spec[k] for k in keys}


def _ord_of(targets, handle):
    return next(o for o, h in targets.items() if h == handle)


def _end_ord(comp_ord, path):
    return comp_ord.rstrip("/") + ("/" + path if path else "")


def _check_ends(scope, comp_ord, spec):
    for key in ("source_path", "target_path"):
        scope.check(_end_ord(comp_ord, spec[key]))


def _assigned_name(op, reply):
    first = reply[0] if isinstance(reply, list) and reply else {}
    return first.get("nn", op["n"]) if isinstance(first, dict) else op["n"]


def _run_relinks(sess, write, batch_id, planned, replies):
    """Re-create the links of restored components as ops of the rollback's own batch.

    Runs after the component ops, once the new handles exist: scope is checked for both
    ends, the concrete ops are journaled (write-ahead) before the first is sent, and each
    counts against the write budget. Returns `(report, inverse, doubt)`; `inverse` removes
    the links created. A send failure raises the batch in-doubt error.
    """
    data, ops, skipped, meta = planned.data, [], 0, []
    for op, reply in zip(planned.ops, replies):
        if op["nm"] != "a":
            continue
        comp = "%s/%s" % (data["ord_of"][op["h"]], _assigned_name(op, reply))
        nodes = sess.client.load_tree(comp, depth=SNAPSHOT_DEPTH)
        for spec in data["relinks"]:
            src, tgt = nodes.get(spec["source_path"]), nodes.get(spec["target_path"])
            try:
                _check_ends(write.scope, comp, spec)
            except safety.SafetyError:
                src = None
            if not (src and tgt and src.get("h") and tgt.get("h")):
                skipped += 1  # an end is missing (or out of scope): nothing to link
                continue
            ops.append({"ssc": "checkLinks", "arg": {
                "s": src["h"], "ss": spec["source_slot"], "t": tgt["h"],
                "ts": spec["target_slot"], "c": True}})
            meta.append(tgt["h"])
    inverse, restored, ambiguous = [], 0, 0
    if ops:
        try:
            write.journal.append({"batch_id": batch_id, "ts": _now(), "phase": "relink-intent",
                                  "ops": ops})
        except OSError:
            raise ToolError("nothing more was sent: the relink intent could not be written "
                            "(batch %s is in-doubt: components were re-created, links were "
                            "not)" % batch_id) from None
        sess.writes_executed += len(ops)
        for op, tgt_h in zip(ops, meta):
            try:
                result = _link_result([_send(sess.client, op)])
            except box.BoxError:
                result = {}
            except Exception as exc:
                raise _in_doubt(batch_id, exc) from None
            if result.get("v") and result.get("s"):
                restored += 1
                inverse.append({"nm": "v", "h": tgt_h, "n": result["s"]})
            elif "v" not in result or result.get("v"):
                ambiguous += 1  # no verdict, or accepted without a name: maybe applied
    report = {"restored": restored, "skipped": len(data["relinks"]) * sum(
        1 for o in planned.ops if o["nm"] == "a") - restored}
    if ambiguous:
        report["ambiguous"] = ambiguous
    return report, inverse, bool(ambiguous)


def _in_doubt(batch_id, exc):
    why = str(exc) if isinstance(exc, box.BoxError) else type(exc).__name__
    err = ToolError("station call failed after the intent was journaled: batch %s is "
                    "in-doubt (the station may or may not have applied it; inspect it "
                    "before retrying): %s" % (batch_id, why))
    err.batch_id = batch_id
    return err


def _rollback_inverse(planned, replies):
    out = []
    for op, reply in zip(planned.ops, replies):
        if op["nm"] == "a":
            first = reply[0] if isinstance(reply, list) and reply else {}
            nn = first.get("nn", op["n"]) if isinstance(first, dict) else op["n"]
            out.append({"nm": "v", "h": op["h"], "n": nn})
    return out if len(out) == len(planned.ops) else []


def _bson_value(b):
    t = b["t"]
    if t.startswith("baja:Status"):
        kids = {c.get("n"): c.get("v") for c in b.get("s", [])}
        raw = kids.get("value")
        value = float(raw or "0.0") if t.endswith("Numeric") else raw == "true"
        return {"value": value, "status": kids.get("status", "0")}
    if t == "baja:Double":
        return float(b.get("v") or "0.0")
    return b.get("v") == "true" if t == "baja:Boolean" else (b.get("v") or "")


def _rollback_readback(client, args, planned, replies, inverse):
    ord_of, ok, relinks = planned.data["ord_of"], True, None
    for i, op in enumerate(planned.ops):
        where = ord_of[op["h"]]
        nodes = client.load_tree(where, depth=2)
        if op["nm"] == "v":
            ok &= op["n"] not in nodes
        elif op["nm"] == "a":
            nn = inverse[i]["n"] if len(inverse) == len(planned.ops) else op["n"]
            ok &= nn in nodes and nodes[nn].get("t") == op["b"]["t"]
            if planned.data["relinks"]:
                relinks = planned.data.get("relink_report") or {"restored": 0, "skipped": 1}
                ok &= relinks["skipped"] == 0 and not relinks.get("ambiguous")
        else:
            ok &= op["n"] in nodes and \
                _observe(nodes, op["n"], op["b"]["t"]) == _bson_value(op["b"])
    for path in planned.data.get("created", []):  # every nested component is really there
        head, _, leaf = path.rpartition("/")
        ok &= leaf in client.load_tree(head, depth=1)
    observed = {"restored": bool(ok)}
    if relinks is not None:
        observed["relinks"] = relinks
    return {"rolled_back": planned.data["rollback_of"]}, replies, observed, \
        "verified" if ok else "mismatch"


# ---- n4_save_station -----------------------------------------------------

def _bog_state(home):
    path = os.path.join(home, "config.bog")
    try:
        st = os.stat(path)
        digest = hashlib.sha256()
        with open(path, "rb") as fh:
            for chunk in iter(lambda: fh.read(1 << 20), b""):
                digest.update(chunk)
    except OSError:
        return None
    return {"mtime": st.st_mtime, "size": st.st_size, "sha256": digest.hexdigest()}


def _save_plan(client, args, ctx):
    home = ctx.write.station_homes.get(ctx.session.station_name)
    note = "the BOX reply to save is null and proves nothing; "
    note += ("persistence is checked on %s" % os.path.join(home, "config.bog") if home else
             "no --station-home for this station, so persistence cannot be checked")
    return Planned([{"ssc": "invokeAction",
                     "arg": {"h": ctx.session.root_handle, "a": "save"}}], [],
                   [note, "no inverse: a save cannot be undone"],
                   {"home": home, "before": _bog_state(home) if home else None,
                    "station": ctx.session.station_name})


def _save_readback(client, args, planned, replies, inverse, ctx=None):
    home, before = planned.data["home"], planned.data["before"]
    name = planned.data["station"]
    if not home:
        hint = ("configure --station-home %s=<directory containing config.bog> to check "
                "persistence" % name)
        return {"action": "save"}, replies[0], {"persisted": "unknown",
                                                "evidence": {"hint": hint}}, "unverified"
    path = os.path.join(home, "config.bog")
    if before is None:
        return {"action": "save"}, replies[0], {"persisted": "unknown", "evidence": {
            "hint": "config.bog was not readable at %s before the save; check that "
                    "--station-home %s points at the station directory" % (path, name)}}, \
            "unverified"
    write = planned.data["write"]
    deadline, after = write.monotonic() + write.save_timeout, None
    while True:
        after = _bog_state(home)
        if after is None or after["sha256"] != before["sha256"] \
                or after["mtime"] > before["mtime"] or write.monotonic() >= deadline:
            break
        write.sleep(write.save_interval)
    evidence = {"path": path, "before": before, "after": after}
    if after is None:
        evidence["hint"] = "config.bog became unreadable after the save"
        persisted, verdict = "unknown", "unverified"
    elif after["sha256"] != before["sha256"] or after["mtime"] > before["mtime"]:
        persisted, verdict = True, "verified"
    else:
        persisted, verdict = False, "mismatch"
    return {"action": "save"}, replies[0], {"persisted": persisted, "evidence": evidence}, verdict


# ---- pipeline ------------------------------------------------------------

_Impl = namedtuple("_Impl", "scope_ords plan inverse readback recover needs_ctx")
_Impl.__new__.__defaults__ = (None, False)


def _static_inverse(planned, replies):
    return planned.inverse


_IMPLS = {
    "n4_create_component": _Impl(lambda a: [a["parent_ord"]], _create_component_plan,
                                 _create_component_inverse, _create_component_readback),
    "n4_set_slot": _Impl(lambda a: [a["ord"]], _set_slot_plan, _static_inverse,
                         _set_slot_readback),
    "n4_invoke_action": _Impl(lambda a: [a["ord"]], _invoke_plan, _static_inverse,
                              _invoke_readback),
    "n4_create_link": _Impl(lambda a: [a["source_ord"], a["target_ord"]], _link_plan,
                            _link_inverse, _link_readback, _link_recover),
    "n4_remove_component": _Impl(
        lambda a: [a["parent_ord"]] + ([a["link_scan_ord"]] if "link_scan_ord" in a else []),
        _remove_plan, _static_inverse, _remove_readback),
    "n4_rollback": _Impl(lambda a: [], _rollback_plan, _rollback_inverse, _rollback_readback,
                         None, True),
    "n4_save_station": _Impl(lambda a: [], _save_plan, _static_inverse, _save_readback,
                             None, True),
}


def _process(ctx, name, args):
    impl, write, sess = _IMPLS[name], ctx.write, ctx.session
    dry = args.get("dry_run", True)
    if sess is None:
        raise ToolError("not connected: call n4_connect first")
    if not sess.identity_verified:
        raise ToolError("station identity not verified: reconnect with n4_connect and pass "
                        "expected_station=<the station's stationName> before any write")
    if sess.writes_executed >= write.max_writes:
        raise ToolError("write budget exhausted: %d writes already executed in this session "
                        "(--max-writes)" % write.max_writes)
    ords = impl.scope_ords(args)
    for ord_str in ords:
        write.scope.check(ord_str)
    if not ords:  # whole-station or journal-driven: at least one scope must be configured
        write.scope.require_any()
    planned = impl.plan(sess.client, args, ctx) if impl.needs_ctx else \
        impl.plan(sess.client, args)
    if name == "n4_save_station":
        planned.data["write"] = write  # readback polls with the operator's timing
    plan = {"tool": name, "ops": planned.ops, "inverse": planned.inverse, "notes": planned.notes}
    data = _data(planned)
    for key in ("relinks", "components", "outgoing_links_broken"):  # part of what the token authorizes
        if data.get(key):
            plan[key] = data[key]
    plan_hash = hashlib.sha256(safety.canonical(plan).encode()).hexdigest()
    if dry:
        token, expires_at = write.tokens.issue(name, args, plan_hash)
        return {"dry_run": True, "plan": plan, "plan_hash": plan_hash,
                "confirmation_token": token, "expires_at": expires_at}, None
    write.tokens.consume(name, args, plan_hash, args.get("confirmation_token"))
    batch_id = uuid.uuid4().hex
    intent = {"batch_id": batch_id, "ts": _now(), "tool": name, "ops": planned.ops,
              "inverse_plan": planned.inverse, "station_name": sess.station_name,
              "phase": "intent", "targets": data.get("targets", {})}
    if data.get("rollback_of"):
        intent["rollback_of"] = data["rollback_of"]
    if data.get("relinks"):
        intent["relinks"] = data["relinks"]
    if data.get("components"):
        intent["components"] = data["components"]
    try:  # write-ahead: no intent on disk, no op on the wire
        write.journal.append(intent)
    except OSError:
        raise ToolError("nothing was sent: the journal intent could not be written (batch %s); "
                        "the confirmation token is spent, run the dry run again"
                        % batch_id) from None
    sess.writes_executed += 1  # counted once sent, whatever the station answers
    try:
        replies = [_send(sess.client, op) for op in planned.ops]
    except Exception as exc:
        raise _in_doubt(batch_id, exc) from None
    out = {"dry_run": False, "batch_id": batch_id}
    warnings, relink_inverse, relink_doubt = [], [], False
    if data.get("components"):  # nested components: one add per component, same batch
        data["created"] = _run_components(sess, write, batch_id, planned, replies)
    if data.get("relinks"):  # the rollback's links: same batch, same guards, journaled
        data["relink_report"], relink_inverse, relink_doubt = _run_relinks(
            sess, write, batch_id, planned, replies)
    try:
        inverse = impl.inverse(planned, replies)
    except Exception as exc:  # the station's reply cannot be trusted: outcome unknown
        why = str(exc) if isinstance(exc, box.BoxError) else type(exc).__name__
        observed, inverse = None, []
        if impl.recover is not None:
            try:
                observed, inverse = impl.recover(sess.client, args, planned, replies)
            except Exception:
                observed, inverse = None, []
        out.update(requested=None, accepted=replies[0] if replies else None,
                   observed=observed, verdict="unverified", in_doubt=True,
                   readback_error=ctx.scrub(why))
    else:
        try:
            requested, accepted, observed, verdict = impl.readback(
                sess.client, args, planned, replies, inverse)
            out.update(requested=requested, accepted=accepted, observed=observed,
                       verdict=verdict)
        except Exception as exc:  # whatever the read-back hits, the write already happened
            why = str(exc) if isinstance(exc, box.BoxError) else \
                "%s: %s" % (type(exc).__name__, exc)
            out.update(requested=None, accepted=replies[0] if replies else None,
                       observed=None, verdict="failed", readback_failed=True,
                       readback_error=ctx.scrub(why))
    inverse = relink_inverse + inverse  # links first: undoing must precede removing their ends
    out["inverse"] = inverse
    if isinstance(out.get("observed"), dict):  # promote the headline evidence of a tool
        out.update({k: out["observed"][k] for k in ("persisted", "evidence", "relinks")
                    if k in out["observed"]})
    result = {"batch_id": batch_id, "ts": _now(), "phase": "result",
              "accepted": out["accepted"], "inverse": inverse, "verdict": out["verdict"]}
    if relink_doubt:
        out["in_doubt"] = True
    if data.get("relink_report"):
        result["relinks"] = data["relink_report"]
    if out.get("readback_failed"):  # accepted by the station, outcome unverified
        result["readback_failed"] = True
    if out.get("in_doubt"):
        result["in_doubt"] = True
    try:
        write.journal.append(result)
    except OSError:
        warnings.append("the journal result could not be written: batch %s is in-doubt; "
                        "inverse ops: %s" % (batch_id, safety.canonical(inverse)))
    if warnings:
        out["warnings"] = warnings
    return out, batch_id


def _now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="seconds")


def _record(ctx, name, args, batch_id, outcome, reason):
    entry = {"ts": _now(), "tool": name, "batch_id": batch_id,
             "dry_run": args.get("dry_run", True), "outcome": outcome,
             "reason": ctx.scrub(reason), "args_redacted": safety.redact_args(args)}
    try:
        ctx.write.audit.append(entry)
    except OSError:
        return False
    return True


def _make_handler(name):
    def handler(ctx, args):
        try:
            result, batch_id = _process(ctx, name, args)
        except (ToolError, safety.SafetyError) as exc:
            doubt = getattr(exc, "batch_id", None)  # station call failed after the intent
            _record(ctx, name, args, doubt, "error" if doubt else "refused", str(exc))
            raise ToolError(str(exc)) from None
        except Exception as exc:
            _record(ctx, name, args, None, "error", "%s: %s" % (type(exc).__name__, exc))
            raise
        if result["dry_run"]:
            logged = _record(ctx, name, args, None, "planned", "dry run")
        else:
            logged = _record(ctx, name, args, batch_id, "executed",
                             "verdict=%s" % result["verdict"])
        if not logged:
            result.setdefault("warnings", []).append("the audit log could not be written")
        return result
    return handler


# ---- tool table ----------------------------------------------------------

def _write_schema(properties, required):
    props = dict(properties, dry_run={
        "type": "boolean", "default": True,
        "description": "true (default) returns the plan and a confirmation_token without "
                       "touching the station; false executes and needs that token"},
        confirmation_token=_str("Token returned by the matching dry run (single use)"))
    return _schema(props, required)


def _tool(name, description, properties, required, annotations=WRITE):
    return Tool(name, description + " Dry run by default; to execute, repeat the call with "
                "dry_run=false and the confirmation_token from the dry run.",
                _write_schema(properties, required), _make_handler(name),
                needs_session=False, annotations=annotations)


TOOLS = [
    _tool("n4_create_component",
          "Add a component under parent_ord (optionally with a wire sheet position).",
          {"parent_ord": _str("Parent component ORD"),
           "name": _str("Requested slot name (the station may rename on collision)"),
           "type": _str("Type spec module:Type, e.g. kitControl:NumericConst"),
           "wire_sheet": {"type": "object", "description": "Optional {x, y, w} wire sheet "
                          "annotation (integers; w defaults to 8)"}},
          ["parent_ord", "name", "type"]),
    _tool("n4_set_slot",
          "Set one slot to a value. Status slots are always written whole (value and status).",
          {"ord": _str("Component ORD"), "slot": _str("Slot name; must exist"),
           "value": {"description": "New value (number, boolean or string by value_type)"},
           "value_type": _str("One of " + ", ".join(VALUE_TYPES), enum=list(VALUE_TYPES)),
           "status": _str("Status string for Status types", default="0")},
          ["ord", "slot", "value", "value_type"],
          dict(WRITE, idempotentHint=True)),
    _tool("n4_invoke_action",
          "Invoke a control action. Only %s are allowed in this version." %
          ", ".join(ALLOWED_ACTIONS),
          {"ord": _str("Component ORD"), "action": _str("Action name"),
           "arg": {"description": "Argument, only for set"},
           "arg_type": _str("baja:Double or baja:Boolean, only for set")},
          ["ord", "action"]),
    _tool("n4_create_link",
          "Create a link from a source slot to a target slot.",
          {"source_ord": _str("Source component ORD"), "source_slot": _str("Source slot name"),
           "target_ord": _str("Target component ORD"), "target_slot": _str("Target slot name")},
          ["source_ord", "source_slot", "target_ord", "target_slot"]),
]
TOOLS += [
    _tool("n4_remove_component",
          "Remove the child `name` of parent_ord with its whole subtree. The plan snapshots "
          "the subtree to depth 3 (type, plain slots, wsAnnotation, links inside it) as the "
          "inverse; n4_rollback can re-create only that snapshot, not a full restore.",
          {"parent_ord": _str("Parent component ORD"), "name": _str("Child slot name"),
           "link_scan_ord": _str("Optional wider ORD (inside the write scope) to scan for "
                                 "links from the subtree into components outside it")},
          ["parent_ord", "name"], DESTRUCTIVE),
    _tool("n4_rollback",
          "Undo a journaled batch by running its recorded inverse through the same dry run / "
          "token pipeline, as a new batch linked to the original. Refuses unknown, "
          "already rolled back, in-doubt and other-station batches.",
          {"batch_id": _str("batch_id returned by an earlier write")}, ["batch_id"],
          DESTRUCTIVE),
    _tool("n4_save_station",
          "Save the station (persist the running config to disk). The BOX reply proves "
          "nothing; persisted is true/false/unknown from config.bog when the server has "
          "--station-home for this station.",
          {}, [], DESTRUCTIVE),
]
NAMES = frozenset(t.name for t in TOOLS)
