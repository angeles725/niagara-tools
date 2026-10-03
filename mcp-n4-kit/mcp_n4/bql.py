"""Pure helpers for projected BQL reads (retro 2026-10-02 D1, D8).

A station answers `GET /ord/<url-encoded ORD>` with HTTP Basic auth, where the ORD is
`station:|slot:<base>|bql:select <cols> from <type> [where ...]|view:file:ITableToCsv`.
Only the path form works: `/ord?<ORD>` answers HTTP 400. One query returns a whole
subtree as CSV in well under a second, where a navigate/read crawl costs one BOX round
trip per component. Nothing here touches the network; importing it has no side effects.
"""
import csv
import io
import re

#: Default and hard upper bound on the rows one query returns.
DEFAULT_MAX_ROWS = 5000
MAX_ROWS_LIMIT = 50000
#: Bytes read from one response at most; beyond it the result is truncated.
MAX_RESPONSE_BYTES = 32 * 1024 * 1024
MAX_QUERY_LENGTH = 2000
VIEW = "view:file:ITableToCsv"
STATION_SLOT = "station:|slot:"

_SELECT = re.compile(r"select\s+\S.*?\s+from\s+\S+(?:\s+.*)?", re.I | re.S)
_CONTROL = re.compile(r"[\x00-\x1f\x7f﻿]")
_ESCAPE = re.compile(r"\$(?:u([0-9a-fA-F]{4})|([0-9a-fA-F]{2}))")

#: Columns that hold an escaped slot NAME; they are decoded for display. `Slot Path`
#: stays escaped on purpose: it is ORD-ready (`station:|` + it, for n4_read_slots).
NAME_COLUMNS = ("Name",)
#: Frozen slots where a network keeps the station's own device (SnmpNetwork's
#: `localDevice` is a BSnmpAgent: the station acting as an agent, not a field device).
LOCAL_DEVICE_SLOTS = ("localDevice",)


def unescape_slot(text):
    """Decode Niagara slot-name escaping: `$20` -> ' ', `$2d` -> '-', `$u00e9` -> 'é'."""
    return _ESCAPE.sub(lambda m: chr(int(m.group(1) or m.group(2), 16)), text)


def strip_control(text):
    """Drop control characters (and a BOM) from one cell or header."""
    return _CONTROL.sub("", text)


def validate_query(query):
    """Return the stripped query, or raise ValueError unless it is one plain `select`.

    No `|` (it would chain another ORD scheme after the composed one) and no control
    characters, anywhere: a `|` inside a quoted literal is refused too.
    """
    if not isinstance(query, str):
        raise ValueError("query must be a string starting with select")
    query = query.strip()
    if "|" in query:
        raise ValueError("query must not contain '|': only one select is composed into the ORD")
    if _CONTROL.search(query):
        raise ValueError("query must not contain control characters")
    if len(query) > MAX_QUERY_LENGTH:
        raise ValueError("query is longer than %d characters" % MAX_QUERY_LENGTH)
    if not _SELECT.fullmatch(query):
        raise ValueError("only 'select <columns> from <type> [where ...]' is allowed")
    return query


def base_slot(base):
    """The slot path (`/Drivers`, `/` for the root) of a base given as an ORD or a path."""
    if not isinstance(base, str) or not base:
        raise ValueError("base must be a station slot ORD such as station:|slot:/Drivers")
    if base.startswith(STATION_SLOT):
        path = base[len(STATION_SLOT):]
    elif base.startswith("slot:"):
        path = base[len("slot:"):]
    else:
        path = base
    if not path.startswith("/") or "|" in path or "//" in path or _CONTROL.search(path):
        raise ValueError("base must be a plain slot path such as station:|slot:/Drivers, "
                         "got %r" % base)
    return path.rstrip("/") or "/"


def compose_ord(base, query):
    """The single ORD for `query` over `base`: the only place a BQL ORD is built."""
    return "%s%s|bql:%s|%s" % (STATION_SLOT, base_slot(base), validate_query(query), VIEW)


_SELECT_LIST = re.compile(r"select\s+(.*?)\s+from\s", re.I | re.S)


def selected_slots(query):
    """The queried slot names in order (`select slotPath, name from ...`), None for `*`.

    The CSV headers are the station's DISPLAY names, which a localized station translates
    (audit 2026-10-03 F12); the select list is ours, so it names columns by position.
    """
    match = _SELECT_LIST.match(query.strip())
    if not match:
        return None
    names = [part.strip() for part in match.group(1).split(",")]
    return None if any(n == "*" or not n for n in names) else names


def unique_columns(header):
    """Header names made unique: a repeated `Type` becomes `Type#2`, `Type#3`, ... (F7)."""
    columns, seen = [], set()
    for name in header:
        unique, n = name, 2
        while unique in seen:
            unique, n = "%s#%d" % (name, n), n + 1
        seen.add(unique)
        columns.append(unique)
    return columns


def by_position(columns, rows, names):
    """Rows re-keyed by `names`, one per leading column (header text is not trusted)."""
    if len(columns) < len(names):
        raise ValueError("the BQL answer has %d column(s), expected at least %d (%s)"
                         % (len(columns), len(names), ", ".join(names)))
    return [{name: row.get(col, "") for name, col in zip(names, columns)} for row in rows]


def parse_csv(text, max_rows=DEFAULT_MAX_ROWS, name_positions=None):
    """`(columns, rows, truncated)` from an ITableToCsv body; rows are dicts by column.

    Repeated headers are made unique (`unique_columns`). The slot-name cells to decode are
    the columns at `name_positions` (from the query's select list) or, without it, the
    columns whose header is in `NAME_COLUMNS`.
    """
    reader = csv.reader(io.StringIO(text.lstrip("\ufeff"), newline=""))
    header = next(reader, None)
    if not header:
        return [], [], False
    columns = unique_columns([strip_control(h).strip() for h in header])
    decode = set(name_positions) if name_positions is not None else \
        {i for i, col in enumerate(columns) if col in NAME_COLUMNS}
    rows, truncated = [], False
    for record in reader:
        if not record:
            continue
        if len(rows) >= max_rows:
            truncated = True
            break
        row = {}
        for i, col in enumerate(columns):
            cell = strip_control(record[i]) if i < len(record) else ""
            row[col] = unescape_slot(cell) if i in decode else cell
        rows.append(row)
    return columns, rows, truncated


# ---- inventory (D8) -------------------------------------------------------

def _rel_parts(row, base):
    """Path segments of a row's `Slot Path` below `base` (escaped, as in the ORD)."""
    path = row.get("Slot Path", "")
    path = path[len("slot:"):] if path.startswith("slot:") else path
    prefix = "" if base == "/" else base
    if not path.startswith(prefix + "/"):
        return []
    return [p for p in path[len(prefix) + 1:].split("/") if p]


def is_local_device(row):
    """True for a network's built-in local device slot (counted apart from field devices)."""
    path = row.get("Slot Path", "")
    return path.rsplit("/", 1)[-1] in LOCAL_DEVICE_SLOTS


def summarize_inventory(base, network_rows, device_rows, point_rows):
    """Devices and points per network below `base`, local devices flagged and apart.

    The network of a row is its first path segment below `base`; a point belongs to the
    deepest device whose path prefixes its own. Points under no device are counted as
    `unassigned_points`.
    """
    base = base_slot(base)
    networks = {}

    def net(name):
        return networks.setdefault(name, {"network": unescape_slot(name), "type": None,
                                          "field_devices": 0, "local_devices": 0,
                                          "points": 0})

    for row in network_rows:
        parts = _rel_parts(row, base)
        if parts:
            net(parts[0])["type"] = row.get("Type")
    devices = []
    for row in device_rows:
        parts = _rel_parts(row, base)
        if not parts:
            continue
        local = is_local_device(row)
        net(parts[0])["local_devices" if local else "field_devices"] += 1
        devices.append({"path": row.get("Slot Path"), "name": row.get("Name"),
                        "type": row.get("Type"), "network": unescape_slot(parts[0]),
                        "local": local, "points": 0})
    by_path = sorted(devices, key=lambda d: len(d["path"] or ""), reverse=True)
    unassigned = 0
    for row in point_rows:
        parts = _rel_parts(row, base)
        if parts:
            net(parts[0])["points"] += 1
        path = row.get("Slot Path") or ""
        owner = next((d for d in by_path if d["path"] and path.startswith(d["path"] + "/")),
                     None)
        if owner is None:
            unassigned += 1
        else:
            owner["points"] += 1
    nets = list(networks.values())
    return {"networks": nets, "devices": devices,
            "totals": {"networks": len(nets),
                       "field_devices": sum(n["field_devices"] for n in nets),
                       "local_devices": sum(n["local_devices"] for n in nets),
                       "points": len(point_rows), "unassigned_points": unassigned}}
