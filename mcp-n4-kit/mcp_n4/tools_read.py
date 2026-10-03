"""Read-only station tools for the MCP server, built on `mcp_n4.box`.

Importing this module has no side effects.
"""
import csv
import hashlib
import hmac
import json
import math
import os
import re
import secrets
import time
import uuid
from collections import namedtuple

from . import __version__, box, bql, retro, safety, tiers

READ_ONLY = {"readOnlyHint": True, "openWorldHint": False}


class ToolError(Exception):
    """A tool failure whose message is safe to show to the model."""


class Tool(namedtuple("Tool", "name description input_schema handler needs_session annotations")):
    def __new__(cls, name, description, input_schema, handler, needs_session=True,
                annotations=None):
        return super().__new__(cls, name, description, input_schema, handler, needs_session,
                               READ_ONLY if annotations is None else annotations)


class Session:
    def __init__(self, client, station_name, base_url, root_handle, identity_verified=False,
                 version=None, version_error=None):
        self.client, self.station_name = client, station_name
        self.base_url, self.root_handle = base_url, root_handle
        #: Detected product version (None when undetected) and its tier (METHODOLOGY 5).
        self.version, self.tier = version, tiers.tier_of(version)
        #: Why the version is unknown (shown to the operator), None when it was detected.
        self.version_error = version_error
        #: True only when the session was opened with an `expected_station` that matched.
        self.identity_verified = identity_verified
        self.writes_executed = 0
        #: Fresh per connect: a write plan (so its confirmation token) is bound to it, so a
        #: token from a dry run never executes after a reconnect (audit 2026-10-03 F1).
        self.session_id = uuid.uuid4().hex


#: The station locks an account after 5 failed logins in 30 s (B1179): after one failure,
#: no station call goes out for at least this long (audit 2026-10-03 F4).
AUTH_COOLDOWN_MIN = 30
#: Default per-request HTTP timeout (seconds) of a station client.
HTTP_TIMEOUT_DEFAULT = 20


class Context:
    """Per-process state: at most one active station session."""

    def __init__(self, allow_writes=False, allow_http=False, env=None, client_factory=None,
                 stations=None, credential_env="MCP_N4", insecure_tls=(), allow_tier_b=(),
                 allow_tier_c=(), auth_cooldown=AUTH_COOLDOWN_MIN,
                 http_timeout=HTTP_TIMEOUT_DEFAULT, load_wait=box.MAX_POLL_WAIT):
        self.allow_writes, self.allow_http = allow_writes, allow_http
        if isinstance(auth_cooldown, bool) or not isinstance(auth_cooldown, (int, float)) \
                or auth_cooldown < AUTH_COOLDOWN_MIN:
            raise ValueError("--auth-cooldown must be >= %d seconds, got %r"
                             % (AUTH_COOLDOWN_MIN, auth_cooldown))
        #: Operator timing (F11): HTTP timeout per request and total wait of one load.
        self.http_timeout, self.load_wait = http_timeout, load_wait
        #: Seconds every station call stays refused after an authentication failure.
        self.auth_cooldown = auth_cooldown
        #: `{"reason", "until", "credentials"}` after an `AuthError`, else None (F4).
        self.auth_latch = None
        self._cred_key = secrets.token_bytes(32)
        self._active_credentials = None
        #: Operator policy, fixed at server start: the model can only pick a NAME.
        self.stations = dict(stations or {})
        self.credential_env = credential_env
        self.insecure_tls = frozenset(insecure_tls)
        for name, url in self.stations.items():
            if not url.startswith("https://") and not (allow_http and url.startswith("http://")):
                raise ValueError("station %s: URL must start with https://" % name)
        #: Operator opt-in per tier (station names): the only way past the tier gate.
        self.tier_opt_in = {"B": frozenset(allow_tier_b), "C": frozenset(allow_tier_c)}
        for flag, names in (("--insecure-tls", self.insecure_tls),
                            (tiers.OPT_IN_FLAGS["B"], self.tier_opt_in["B"]),
                            (tiers.OPT_IN_FLAGS["C"], self.tier_opt_in["C"])):
            unknown = names - set(self.stations)
            if unknown:
                raise ValueError("%s names unconfigured station(s): %s"
                                 % (flag, ", ".join(sorted(unknown))))
        self.env = os.environ if env is None else env
        self.client_factory = client_factory or box.BoxClient
        self.session = None
        #: Where the journal/audit live (the server sets it; None = the default dir).
        self.state_dir = None
        #: In-memory read observations for the session retro (no audit line exists for reads).
        self.observations = []
        #: When this server started: the default window of the session retro draft.
        self.started_at = retro.now()
        self.write = None  # tools_write.WriteState, set by the server in writes-allowed mode
        #: Monotonic clock for `elapsed_ms` (injectable in tests).
        self.clock = time.monotonic
        #: Operator-chosen file for progress lines of long reads (`--progress-file`), or None.
        self.progress_path = None
        self._secret = None

    @property
    def mode(self):
        return "writes-allowed" if self.allow_writes else "read-only"

    def close(self):
        sess, self.session = self.session, None  # cleared even if the close below raises
        if sess is not None:
            sess.client.close()

    def tier_block(self, sess):
        """Why the session's version tier refuses writes, or None (see `tiers.write_block`)."""
        return tiers.write_block(sess.tier, sess.version, sess.station_name, self.tier_opt_in)

    def tier_report(self, sess):
        """`version`, `version_source`, `tier`, `tier_writes` of a session (None: no session)."""
        if sess is None:
            return {"version": None, "version_source": None, "tier": None, "tier_writes": None}
        if self.tier_block(sess):
            writes = "refused"
        else:
            writes = "allowed" if sess.tier == "A" else "allowed-by-opt-in"
        out = {"version": sess.version,
               "version_source": tiers.ABOUT_PATH if sess.version else None,
               "tier": sess.tier, "tier_writes": writes}
        if sess.version_error:
            out["version_error"] = sess.version_error
        return out

    # ---- authentication-failure latch (audit 2026-10-03 F4) ---------------
    def credentials_fingerprint(self, user, secret):
        """Keyed digest of the credentials (never stored in clear, never output)."""
        return hmac.new(self._cred_key, ("%s\0%s" % (user, secret)).encode(),
                        hashlib.sha256).hexdigest()

    def use_credentials(self, fingerprint):
        """The credentials the next station call goes out with (what a failure latches)."""
        self._active_credentials = fingerprint

    def note_auth_failure(self, exc):
        """Latch: refuse station calls for `auth_cooldown` s after any `AuthError`."""
        self.auth_latch = {"reason": self.scrub(str(exc)),
                           "until": self.clock() + self.auth_cooldown,
                           "credentials": self._active_credentials}

    def auth_retry_in(self):
        """Whole seconds left in the cooldown, or 0 (the latch is cleared once it passed)."""
        if self.auth_latch is None:
            return 0
        left = self.auth_latch["until"] - self.clock()
        if left <= 0:
            self.auth_latch = None
            return 0
        return int(math.ceil(left))

    def auth_block(self, credentials=None):
        """Why station calls are paused, or None.

        `credentials` (a fingerprint) is given by `n4_connect`: other credentials than the
        ones that failed may try again during the cooldown.
        """
        left = self.auth_retry_in()
        if not left:
            return None
        if credentials is not None and credentials != self.auth_latch["credentials"]:
            return None
        return ("station calls are paused after an authentication failure (%s): retry in "
                "%d s, or fix %s_USER/%s_PASSWORD and call n4_connect again; every rejected "
                "call counts toward the station lock-out (5 failures in 30 s)"
                % (self.auth_latch["reason"], left, self.credential_env, self.credential_env))

    def auth_report(self):
        """`{"auth_paused": {...}}` while the latch holds, else {}."""
        left = self.auth_retry_in()
        if not left:
            return {}
        return {"auth_paused": {"reason": self.auth_latch["reason"], "retry_in_s": left}}

    def set_secret(self, secret):
        self._secret = secret

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
    try:
        ctx.close()  # first, so any failure below never leaves the old session active
    except Exception:  # the old station may be unreachable; the session is cleared anyway
        pass
    name = args["station"]
    if not ctx.stations:
        raise ToolError("no station is configured: start the server with --station NAME=URL")
    if name not in ctx.stations:
        raise ToolError("unknown station %r: configured stations are %s (set with --station "
                        "NAME=URL)" % (name, ", ".join(sorted(ctx.stations))))
    prefix = ctx.credential_env
    user, secret = ctx.env.get(prefix + "_USER"), ctx.env.get(prefix + "_PASSWORD")
    if not user or not secret:
        raise ToolError("credentials missing: set env vars %s_USER and %s_PASSWORD"
                        % (prefix, prefix))
    ctx.set_secret(secret)
    credentials = ctx.credentials_fingerprint(user, secret)
    block = ctx.auth_block(credentials)
    if block:  # the same credentials just failed: one more try would only near the lock-out
        raise ToolError(block)
    ctx.use_credentials(credentials)
    timing = {}  # only non-defaults: a custom client_factory need not know these kwargs
    if ctx.http_timeout != HTTP_TIMEOUT_DEFAULT:
        timing["timeout"] = ctx.http_timeout
    if ctx.load_wait != box.MAX_POLL_WAIT:
        timing["load_wait"] = ctx.load_wait
    client = ctx.client_factory(ctx.stations[name], user, secret,
                                insecure_tls=name in ctx.insecure_tls,
                                allow_http=ctx.allow_http, **timing)
    try:
        client.on_auth_error = ctx.note_auth_failure
    except AttributeError:  # a client without the hook: the server still latches on raise
        pass
    try:
        root = client.open()
        root_h = root.get("h") if isinstance(root, dict) else None
        if not root_h:
            raise box.BoxError("station returned no root handle")
        nodes = client.load_tree(ROOT_ORD, depth=1, handle=root_h)
    except BaseException:
        client.close()
        raise
    ctx.auth_latch = None  # these credentials logged in: an earlier failure is moot
    station_name = nodes.get("stationName", {}).get("v")
    expected = args.get("expected_station", name)  # defaults to the configured NAME
    if expected != station_name:
        client.close()
        raise ToolError("station identity mismatch: expected %r but connected to %r"
                        % (expected, station_name))
    version, version_error = _detect_version(client)
    ctx.session = Session(client, station_name, client.base_url, root_h,
                          identity_verified=True, version=version,
                          version_error=version_error and ctx.scrub(version_error))
    return dict({"station_name": station_name, "base_url": client.base_url,
                 "root_handle": root_h, "mode": ctx.mode}, **ctx.tier_report(ctx.session),
                **ctx.auth_report())


def _detect_version(client):
    """`(version, error)`: the oBIX productVersion, or None and why. Never fails a connect.

    One GET, never retried. An undetected version classifies as tier C (writes refused
    until the operator opts in), so a failure here can only make the server stricter. The
    reason is reported, not swallowed: a 401/403 here may count as a failed login toward
    the station's lock-out, so the operator must see it.
    """
    about = getattr(client, "about", None)
    if about is None:
        return None, "this client cannot read %s" % tiers.ABOUT_PATH
    try:
        version = tiers.product_version(about())
    except box.AuthError as exc:
        return None, ("%s refused: %s; the station may count it as a failed login, so check "
                      "that this user may read oBIX before reconnecting" % (tiers.ABOUT_PATH, exc))
    except box.BoxError as exc:
        return None, "%s unreadable: %s" % (tiers.ABOUT_PATH, exc)
    except Exception as exc:  # never fail the connect over the version probe
        return None, "%s unreadable (%s)" % (tiers.ABOUT_PATH, type(exc).__name__)
    if version is None:
        return None, "%s has no productVersion" % tiers.ABOUT_PATH
    return version, None


# ---- n4_describe_session -------------------------------------------------

def n4_describe_session(ctx, args):
    sess = ctx.session
    return {"connected": sess is not None,
            "station_name": sess.station_name if sess else None,
            "base_url": sess.base_url if sess else None,
            "mode": ctx.mode, "server_version": __version__,
            "configured_stations": sorted(ctx.stations), **ctx.tier_report(sess),
            **ctx.auth_report()}


# ---- n4_navigate ---------------------------------------------------------

def _tree_entries(nodes, path, levels, types=None):
    """Child entries of `path`, `levels` deep.

    With `types` (a set of type specs) an entry is kept when its type is in the set
    (`matched: true`) or when a nested entry is kept (`matched: false`, the path to it).
    `has_children` is not filtered (issue #179 R3-003): it says whether the entry has any
    component child, so a caller can still descend below the requested depth.
    """
    entries = []
    for name, node in box.children(nodes, path).items():
        child_path = path + "/" + name if path else name
        entry = {"name": name, "type": node.get("t"), "handle": node.get("h"),
                 "has_children": _has_component_children(nodes, child_path)}
        if levels > 1:
            entry["children"] = _tree_entries(nodes, child_path, levels - 1, types)
        if types is not None:
            entry["matched"] = entry["type"] in types
            if not (entry["matched"] or entry.get("children")):
                continue
        entries.append(entry)
    return entries


def _types_arg(args):
    """The `types` filter as the given list, or None when absent or empty (no filter).

    The caller turns it into a set for membership; the list is echoed in the reply as given.
    """
    types = args.get("types") or None
    if types is None:
        return None
    if not all(isinstance(t, str) and t for t in types):
        raise ToolError("types must be a list of non-empty type specs such as "
                        "'bacnet:BacnetDevice', got %r" % (types,))
    return types


def _timed_load(ctx, ord_str, depth):
    """`(nodes, elapsed_ms)` of one `load_tree` (retro 2026-10-02 D2: latency is visible)."""
    start = ctx.clock()
    nodes = ctx.session.client.load_tree(ord_str, depth=depth)
    return nodes, int(round((ctx.clock() - start) * 1000))


def n4_navigate(ctx, args):
    ord_str, depth, types = _ord_arg(args), args.get("depth", 1), _types_arg(args)
    # one extra level so has_children is known for the deepest children returned
    nodes, elapsed = _timed_load(ctx, ord_str, depth + 1)
    out = {"ord": ord_str,
           "children": _tree_entries(nodes, "", depth, None if types is None else set(types)),
           "elapsed_ms": elapsed}
    if types is not None:
        out["types"] = types
    return out


# ---- n4_read_slots -------------------------------------------------------

def _slot_entry(nodes, name):
    node = nodes[name]
    kind = node.get("t")
    display = node.get("d")
    entry = {"name": name, "type": kind, "value": node.get("v"), "status": None}
    if display is not None:
        entry["value_display"] = display
    if not (kind or "").startswith("baja:Status"):
        if entry["value"] is None and display is not None:
            # a complex the reader does not decode (FlexAddress, BacnetAddress, ...):
            # its display string beats a bare None (retro 2026-10-02 D4)
            entry["value"] = display
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
    nodes, elapsed = _timed_load(ctx, ord_str, 2)
    return {"ord": ord_str,
            "slots": [_slot_entry(nodes, name) for name in box.children(nodes, "")],
            "elapsed_ms": elapsed}


# ---- links and dangling outputs -------------------------------------------

def _level(path):
    return path.count("/") + 1 if path else 0


def _abs_path(ord_str, path):
    base = ord_str.split("slot:", 1)[1] if "slot:" in ord_str else "/"
    base = base.rstrip("/")
    return base + "/" + path if path else base or "/"


#: Source ORDs name a component by handle: `h:<handle>` (BOX handle ORD), possibly
#: prefixed by the station as `...|h:<handle>`. Only the part after the last `|` is used.
HANDLE_PREFIX = "h:"


def _scan(ctx, args):
    """Load `ord` and return (ord, nodes, links, depth).

    `links` holds every link whose target component is at most `depth + 1`
    levels below `ord`; each entry carries `target_level`. A link sits one level
    below its target and its slots one more, so the load goes `depth + 3` deep.
    Callers report components up to `depth` only; the extra level exists so that
    an out slot feeding a target one level below `depth` is seen as used.
    """
    ord_str, depth = _ord_arg(args), args.get("depth", 2)
    nodes = ctx.session.client.load_tree(ord_str, depth=depth + 3)
    by_handle = {n["h"]: _abs_path(ord_str, p) for p, n in nodes.items() if n.get("h")}
    links = []
    for path, node in nodes.items():
        if node.get("t") not in LINK_TYPES or _level(path) - 1 > depth + 1:
            continue
        slots = {k: v.get("v") for k, v in box.children(nodes, path).items()}
        source_ord = slots.get("sourceOrd")
        handle = (source_ord or "").rsplit("|", 1)[-1]
        target = path.rpartition("/")[0]
        is_handle = handle.startswith(HANDLE_PREFIX)
        links.append({"target_path": _abs_path(ord_str, target),
                      "link_name": path.rpartition("/")[2],
                      "source_ord": source_ord,
                      "source_path": by_handle.get(handle[len(HANDLE_PREFIX):])
                      if is_handle else None,
                      "source_slot": slots.get("sourceSlotName"),
                      "target_slot": slots.get("targetSlotName"),
                      "target_level": _level(target)})
    return ord_str, nodes, links, depth


def n4_list_links(ctx, args):
    ord_str, _, links, depth = _scan(ctx, args)
    shown = [{k: v for k, v in l.items() if k != "target_level"}
             for l in links if l["target_level"] <= depth]
    return {"ord": ord_str, "links": shown}


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
    if dangling:  # reads leave no audit line: remember it for the session retro
        ctx.observations.append({"ts": retro.now(), "ord": ord_str, "count": len(dangling)})
    return {"ord": ord_str, "dangling": dangling,
            "scope_note": "Components are reported up to depth levels below ord. "
                          "Links are seen when their target is at most one level below "
                          "depth; an out slot used only by a deeper target, or by a "
                          "target outside the subtree, is reported as dangling."}


# ---- bulk reads: n4_bql_query / n4_inventory (retro 2026-10-02 D1, D7, D8) ----

def _record_read(ctx, tool, args, rows):
    """A read leaves a session observation, plus an `audit.jsonl` line (outcome `read`)
    when the write machinery (and so the audit log) is active. Never raises."""
    ctx.observations.append({"ts": retro.now(), "tool": tool, "rows": rows,
                             "ord": ctx.scrub(str(args.get("base", "")))})
    if ctx.write is None:
        return
    try:
        ctx.write.audit.append({"ts": retro.now(), "tool": tool, "batch_id": None,
                                "dry_run": False, "outcome": "read",
                                "reason": "rows=%d" % rows,
                                "args_redacted": safety.redact_args(args)})
    except OSError:
        pass


def _progress(ctx, tool, step, rows, elapsed_ms, log):
    """Append one progress entry to `log` and, best effort, to the operator's file (D7)."""
    entry = {"ts": retro.now(), "tool": tool, "step": step, "rows": rows,
             "elapsed_ms": elapsed_ms}
    log.append(entry)
    if ctx.progress_path:
        try:
            with open(ctx.progress_path, "a", encoding="utf-8") as fh:
                fh.write(json.dumps(entry) + "\n")
        except OSError:
            pass


def _bql(ctx, base, query, max_rows, timeout):
    """`(columns, rows, truncated, elapsed_ms)` of one projected BQL GET."""
    try:
        ord_text = bql.compose_ord(base, query)
    except ValueError as exc:
        raise ToolError(str(exc)) from None
    start = ctx.clock()
    text, cut = ctx.session.client.get_ord(ord_text, timeout=timeout,
                                           max_bytes=bql.MAX_RESPONSE_BYTES)
    slots = bql.selected_slots(query)  # localized headers: decode `name` by position (F12)
    positions = None if slots is None else [i for i, n in enumerate(slots)
                                             if n.lower() == "name"]
    columns, rows, capped = bql.parse_csv(text, max_rows, name_positions=positions)
    return columns, rows, cut or capped, int(round((ctx.clock() - start) * 1000))


#: `output_file` of n4_bql_query: a plain file name with a .csv or .json extension.
_OUTPUT_FILE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.(csv|json)")


def _output_path(ctx, name):
    """`<state-dir>/bql/<name>` for a plain new file name, else ToolError (nothing read)."""
    if not isinstance(name, str) or not _OUTPUT_FILE.fullmatch(name) or ".." in name:
        raise ToolError("output_file must be a plain file name ending in .csv or .json "
                        "(letters, digits, '.', '_', '-'), written under <state-dir>/bql/; "
                        "got %r" % (name,))
    state_dir = ctx.state_dir or os.path.expanduser(safety.DEFAULT_STATE_DIR)
    try:
        safety.check_state_dir(state_dir)
    except safety.SafetyError as exc:
        raise ToolError(str(exc)) from None
    path = os.path.join(state_dir, "bql", name)
    if os.path.lexists(path):
        raise ToolError("output_file %s already exists: choose another name (the server "
                        "never overwrites a result file)" % path)
    return path


def _write_output(path, base, query, columns, rows):
    """Write the result privately (dirs 0700, file 0600, never over an existing file)."""
    for directory in (os.path.dirname(os.path.dirname(path)), os.path.dirname(path)):
        if not os.path.isdir(directory):
            os.makedirs(directory, mode=0o700, exist_ok=True)
            os.chmod(directory, 0o700)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="") as fh:
            if path.endswith(".csv"):
                writer = csv.writer(fh)
                writer.writerow(columns)
                writer.writerows([row.get(c, "") for c in columns] for row in rows)
            else:
                json.dump({"base": base, "query": query, "columns": columns, "rows": rows}, fh)
    except BaseException:  # never leave a truncated file a reader could trust (H5 review R3)
        try:
            os.unlink(path)
        except OSError:
            pass
        raise


def n4_bql_query(ctx, args):
    base = args.get("base", "station:|slot:/")
    path = _output_path(ctx, args["output_file"]) if "output_file" in args else None
    columns, rows, truncated, elapsed = _bql(
        ctx, base, args["query"], args.get("max_rows", bql.DEFAULT_MAX_ROWS),
        args.get("timeout_s", 60))
    _record_read(ctx, "n4_bql_query", args, len(rows))
    out = {"base": base, "query": args["query"].strip(), "columns": columns,
           "row_count": len(rows), "truncated": truncated, "elapsed_ms": elapsed}
    if path is None:
        out["rows"] = rows
        return out
    try:
        _write_output(path, base, out["query"], columns, rows)
    except OSError as exc:
        raise ToolError("output_file %s could not be written (%s)"
                        % (path, exc.strerror or type(exc).__name__)) from None
    out["output_file"] = path
    return out


INVENTORY_QUERIES = (
    ("networks", "select slotPath, name, type from driver:DeviceNetwork"),
    ("devices", "select slotPath, name, type from driver:Device"),
    ("points", "select slotPath, name, type, out from control:ControlPoint"),
)
#: The keys `bql.summarize_inventory` reads, in the order every inventory query selects them.
INVENTORY_COLUMNS = ("Slot Path", "Name", "Type")


def n4_inventory(ctx, args):
    base = args.get("base", "station:|slot:/Drivers")
    max_rows, timeout = args.get("max_rows", bql.DEFAULT_MAX_ROWS), args.get("timeout_s", 60)
    results, progress, truncated, total = {}, [], False, 0
    keyed = {}
    for step, query in INVENTORY_QUERIES:
        columns, rows, cut, elapsed = _bql(ctx, base, query, max_rows, timeout)
        results[step], truncated, total = rows, truncated or cut, total + elapsed
        _progress(ctx, "n4_inventory", step, len(rows), elapsed, progress)
        try:  # by the queried slot's position: headers are localized display names (F12)
            keyed[step] = bql.by_position(columns, rows, INVENTORY_COLUMNS)
        except ValueError as exc:
            raise ToolError("%s query: %s" % (step, exc)) from None
    try:
        out = bql.summarize_inventory(base, keyed["networks"], keyed["devices"],
                                      keyed["points"])
    except ValueError as exc:
        raise ToolError(str(exc)) from None
    _record_read(ctx, "n4_inventory", args, sum(len(r) for r in results.values()))
    out.update(base=base, truncated=truncated, progress=progress, elapsed_ms=total)
    if args.get("include_points"):
        out["point_rows"] = results["points"]
    return out


def n4_list_batches(ctx, args):
    """The journaled write batches, newest first (reads the journal only, never the station).

    Audit 2026-10-03 F14: before this, finding a `batch_id` for `n4_rollback` meant reading
    `journal.jsonl` by hand.
    """
    views = safety.Journal(ctx.state_dir or os.path.expanduser(safety.DEFAULT_STATE_DIR)).views()
    rolled = {}
    for view in views:
        if view.get("rollback_of"):
            rolled.setdefault(view["rollback_of"], []).append(view.get("batch_id"))
    limit = args.get("limit", 50)
    batches = [{"batch_id": v.get("batch_id"), "tool": v.get("tool"),
                "station_name": v.get("station_name"), "ts": v.get("ts"),
                "result_ts": v.get("result_ts"), "state": v.get("state"),
                "verdict": v.get("verdict"), "rollback_of": v.get("rollback_of"),
                "rolled_back_by": rolled.get(v.get("batch_id"), [])}
               for v in reversed(views)]
    return {"total": len(batches), "batches": batches[:limit]}


def n4_session_retro_draft(ctx, args):
    """Draft the session retro for the server's state dir (reads only, never writes).

    The window starts at the server start unless `since` says otherwise, so a draft
    covers this session and not the whole history of the state directory.
    """
    station = ctx.session.station_name if ctx.session else "unknown"
    return retro.draft(ctx.state_dir or safety.DEFAULT_STATE_DIR, station=station,
                       since=args.get("since") or ctx.started_at,
                       observations=ctx.observations)


TOOLS = [
    Tool("n4_connect",
         "Open a session to one of the stations the operator configured at server start "
         "(replaces any previous session). The URL, credentials and TLS policy are fixed by "
         "the operator and cannot be chosen here. The station's real stationName must equal "
         "expected_station (default: the configured station NAME), or the session is closed. "
         "Reports the detected version (oBIX productVersion) and its tier: A (4.13/4.14) "
         "writes; B (4.15/4.3) and C (other or unknown) refuse writes unless the operator "
         "opted the station in.",
         _schema({"station": _str("Configured station NAME (see n4_describe_session)"),
                  "expected_station": _str("Required stationName; default: the station NAME")},
                 ["station"]),
         n4_connect, needs_session=False),
    Tool("n4_describe_session",
         "Report whether a station session is active, which station, its version tier, "
         "and the server mode. "
         "Works without a session.",
         _schema({}), n4_describe_session, needs_session=False),
    Tool("n4_navigate",
         "List the children of a component (name, type, handle, has_children). "
         "depth > 1 nests grandchildren under a 'children' key. elapsed_ms is the load "
         "time. For a station inventory use n4_bql_query or n4_inventory instead: one "
         "navigate is one round trip per component. types (optional) keeps only children "
         "whose type is one of the given specs (e.g. bacnet:BacnetDevice) plus the path to "
         "them; each kept entry then carries matched true/false. has_children ignores the "
         "types filter: it is true when the entry has any component child, matching or not.",
         _schema({"ord": _str("Station ORD, default station:|slot:/", default=ROOT_ORD),
                  "depth": _int("Levels to list (1-3)", 1, 3, 1),
                  "types": {"type": "array", "items": {"type": "string"},
                            "description": "Optional type specs to keep, e.g. "
                                           "['bacnet:BacnetDevice']; default: no filter"}}),
         n4_navigate),
    Tool("n4_read_slots",
         "Read all slots of one component as {name, type, value, status}. Status complexes "
         "add status_ok/status_null; omitted values take the type default. value_display "
         "carries the station's display string; a complex the reader cannot decode "
         "(e.g. a Modbus FlexAddress) returns that display string as value. elapsed_ms "
         "is the load time.",
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
    Tool("n4_bql_query",
         "Bulk read: one projected BQL select over a subtree, returned as rows (one HTTP GET, "
         "usually under a second, where a navigate crawl costs one round trip per "
         "component). Use it FIRST for any inventory. Only 'select <cols> from <type> "
         "[where ...]' is accepted; '|' is refused. Columns are the station's display "
         "headers (e.g. 'Slot Path', 'Name'; a localized station translates them, a repeated "
         "header becomes 'Type#2'); the queried name column is decoded ($20 -> space), "
         "'Slot Path' stays ORD-ready. Proxy-extension fields project as proxyExt.<slot>, e.g. "
         "proxyExt.dataAddress. Rows are capped (truncated=true when cut).",
         _schema({"base": _str("Subtree ORD or slot path, e.g. station:|slot:/Drivers "
                               "(default: the station root)"),
                  "query": _str("select <columns> from <type> [where ...], e.g. select "
                                "slotPath, name, out from control:ControlPoint"),
                  "max_rows": _int("Row cap", 1, bql.MAX_ROWS_LIMIT, bql.DEFAULT_MAX_ROWS),
                  "timeout_s": _int("HTTP timeout in seconds", 1, 600, 60),
                  "output_file": _str("Optional plain file name (.csv or .json): the rows go "
                                      "to <state-dir>/bql/<name> (0600, never overwritten) "
                                      "instead of the reply")},
                 ["query"]),
         n4_bql_query),
    Tool("n4_inventory",
         "Station inventory through three BQL queries: networks, devices and points below "
         "base (default /Drivers), counted per network and per device. A network's built-in "
         "local device slot (localDevice, e.g. SnmpNetwork's own agent) is flagged "
         "local=true and counted apart from field devices. progress lists each step's row "
         "count; the operator's --progress-file gets the same lines.",
         _schema({"base": _str("Subtree ORD, default station:|slot:/Drivers"),
                  "include_points": {"type": "boolean", "default": False,
                                     "description": "Also return the point rows"},
                  "max_rows": _int("Row cap per query", 1, bql.MAX_ROWS_LIMIT,
                                   bql.DEFAULT_MAX_ROWS),
                  "timeout_s": _int("HTTP timeout per query in seconds", 1, 600, 60)}),
         n4_inventory),
    Tool("n4_list_batches",
         "List the write batches in this server's journal, newest first: batch_id, tool, "
         "station_name, ts, state (completed / in-doubt), verdict, rollback_of and "
         "rolled_back_by. Use it to find the batch_id for n4_rollback. Reads the journal "
         "only; works without a session.",
         _schema({"limit": _int("Most recent batches to return", 1, 1000, 50)}),
         n4_list_batches, needs_session=False),
    Tool("n4_session_retro_draft",
         "Draft the session retro: reads this server's audit and journal and returns markdown "
         "plus evidence-backed CANDIDATE kit deltas (refusals, bad read-back verdicts, in-doubt "
         "batches, BOX errors, dangling outputs). Proposes only; never applies or stages "
         "anything. The window starts at the server start; pass since (ISO-8601, e.g. "
         "1970-01-01T00:00:00Z for the whole history) to change it.",
         _schema({"since": _str("ISO-8601 timestamp; only newer entries are considered "
                                "(default: when this server started)")}),
         n4_session_retro_draft, needs_session=False),
]
