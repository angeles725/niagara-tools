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
from .tools_read import Tool, ToolError, _schema, _str

DEFAULT_STATE_DIR = "~/.local/state/mcp-n4"
WRITE = {"readOnlyHint": False, "destructiveHint": False, "openWorldHint": False}
#: Actions `n4_invoke_action` may call. Destructive ones (emergency*, save, restart)
#: arrive later under a separate class.
ALLOWED_ACTIONS = ("set", "active", "inactive", "auto")
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
                 clock=time.time):
        state_dir = os.path.expanduser(state_dir or DEFAULT_STATE_DIR)
        safety.check_state_dir(state_dir)  # refuse a dir we do not own; never chmod it
        self.scope = safety.WriteScope(scopes)
        self.tokens = safety.ConfirmationTokens(token_ttl, clock)
        self.journal, self.audit = safety.Journal(state_dir), safety.AuditLog(state_dir)
        self.max_writes = max_writes


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


def _send(client, op):
    if "ssc" in op:
        return client.ssc(op["ssc"], op["arg"])
    return client.sync(op)


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
        {"parent_h": parent_h, "ws": ws["v"] if ws else None})


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
                   [], {"requested": requested})


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
        raise ToolError("action %r is not allowed in this version (allowed: %s)"
                        % (action, ", ".join(ALLOWED_ACTIONS)))
    h, nodes = _handle(client, args["ord"], 2)
    arg = {"h": h, "a": action}
    if action != "set":
        if "arg" in args or "arg_type" in args:
            raise ToolError("action %r takes no arg" % action)
        return Planned([{"ssc": "invokeAction", "arg": arg}], [],
                       ["no inverse: %s changes the control state and is not restored "
                        "automatically" % action], {"h": h})
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
                   [], {"h": h, "value": value, "fb_type": fb_type})


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
                   [], {"src_h": src_h, "tgt_h": tgt_h, "existing_links": existing})


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


# ---- pipeline ------------------------------------------------------------

_Impl = namedtuple("_Impl", "scope_ords plan inverse readback recover")
_Impl.__new__.__defaults__ = (None,)


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
    for ord_str in impl.scope_ords(args):
        write.scope.check(ord_str)
    planned = impl.plan(sess.client, args)
    plan = {"tool": name, "ops": planned.ops, "inverse": planned.inverse, "notes": planned.notes}
    plan_hash = hashlib.sha256(safety.canonical(plan).encode()).hexdigest()
    if dry:
        token, expires_at = write.tokens.issue(name, args, plan_hash)
        return {"dry_run": True, "plan": plan, "plan_hash": plan_hash,
                "confirmation_token": token, "expires_at": expires_at}, None
    write.tokens.consume(name, args, plan_hash, args.get("confirmation_token"))
    batch_id = uuid.uuid4().hex
    intent = {"batch_id": batch_id, "ts": _now(), "tool": name, "ops": planned.ops,
              "inverse_plan": planned.inverse, "station_name": sess.station_name,
              "phase": "intent"}
    if isinstance(planned.data, dict) and planned.data.get("rollback_of"):
        intent["rollback_of"] = planned.data["rollback_of"]
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
        why = str(exc) if isinstance(exc, box.BoxError) else type(exc).__name__
        err = ToolError("station call failed after the intent was journaled: batch %s is "
                        "in-doubt (the station may or may not have applied it; inspect it "
                        "before retrying): %s" % (batch_id, why))
        err.batch_id = batch_id
        raise err from None
    out = {"dry_run": False, "batch_id": batch_id}
    warnings = []
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
                       observed=None, verdict="failed", readback_error=ctx.scrub(why))
    out["inverse"] = inverse
    result = {"batch_id": batch_id, "ts": _now(), "phase": "result",
              "accepted": out["accepted"], "inverse": inverse, "verdict": out["verdict"]}
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
NAMES = frozenset(t.name for t in TOOLS)
