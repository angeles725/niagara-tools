"""Pure helpers for station file reads and Px view summaries.

A station serves its files on `GET /file/<path>` (web-rt `FileServlet`, mapped to
`/file/*`): the file ORD `file:^a/b` is the URL path `/file/a/b`. The servlet needs read
permission and refuses directories and `.bog` / `.bog.gz` files; `station_path` mirrors
those refusals so a refused path never leaves the client. Nothing here touches the
network; importing it has no side effects.
"""
import re
import urllib.parse
import xml.etree.ElementTree as ET
from collections import Counter

#: Default and hard upper bound on the bytes one file read returns.
DEFAULT_MAX_BYTES = 2 * 1024 * 1024
MAX_BYTES_LIMIT = 16 * 1024 * 1024
MIN_BYTES = 1024
MAX_PATH_LENGTH = 1024
#: Bound on every list a Px summary returns (labels, ORDs, images, ...).
DEFAULT_MAX_ITEMS = 200

FILE_ORD_PREFIXES = ("station:|file:^", "file:^")
#: Refused like `FileServlet.canRead` (a station database is never served).
REFUSED_SUFFIXES = (".bog", ".bog.gz")
TEXT_EXTENSIONS = frozenset((
    "px", "xml", "nav", "chart", "json", "csv", "html", "htm", "txt", "css", "js",
    "svg", "properties", "md", "log", "bajascript"))

_CONTROL = re.compile(r"[\x00-\x1f\x7f]")
_DRIVE = re.compile(r"^[A-Za-z]:")
#: Slot names that hold a value, not a component: stripped to group bindings by component.
VALUE_SLOTS = frozenset({"out", "in", "fallback", "value", "status", "set", "facets",
                         "proxyExt", "override", "emergencyOverride"}
                        | {"in%d" % i for i in range(1, 17)})
_SLOT_HOP = re.compile(r"\|slot:([A-Za-z0-9_$]+)$")


def station_path(file_ord):
    """The station-relative path (`px/General.px`) of a file ORD or relative path.

    Accepts `file:^px/X.px`, `station:|file:^px/X.px` or `px/X.px`. Raises ValueError for
    anything else: an absolute or home path, a drive letter, a backslash, `.`/`..` or empty
    segments, a trailing `/` (a directory), `|`, control characters, `:`, a segment ending
    in `.` or space (Windows aliases), and `.bog`/`.bog.gz`.
    """
    if not isinstance(file_ord, str) or not file_ord:
        raise ValueError("file_ord must be a file ORD such as file:^px/General.px or a "
                         "relative path such as px/General.px")
    path = file_ord
    for prefix in FILE_ORD_PREFIXES:
        if path.startswith(prefix):
            path = path[len(prefix):]
            break
    else:
        if path.startswith(("file:", "station:")):
            raise ValueError("only station-home file ORDs (file:^...) are read, got %r"
                             % (file_ord,))
    if not path:
        raise ValueError("file_ord names no file: %r" % (file_ord,))
    if len(path) > MAX_PATH_LENGTH:
        raise ValueError("file path is longer than %d characters" % MAX_PATH_LENGTH)
    if _CONTROL.search(path):
        raise ValueError("file path must not contain control characters")
    if "|" in path:
        raise ValueError("file path must not contain '|' (no ORD hop after the file)")
    if "\\" in path or _DRIVE.match(path) or path.startswith(("/", "~")):
        raise ValueError("file path must be relative to the station home, got %r" % path)
    if path.endswith("/"):
        raise ValueError("file path names a directory (%r): the station serves files only"
                         % path)
    segments = path.split("/")
    if any(seg in ("", ".", "..") for seg in segments):
        raise ValueError("file path must not contain empty, '.' or '..' segments: %r" % path)
    # Windows aliases: a station on Windows strips trailing dots and spaces from a name
    # ("config.bog." is config.bog, ".. " is "..") and reads "a:b" as an NTFS stream.
    if ":" in path:
        raise ValueError("file path must not contain ':' (drive or NTFS stream): %r" % path)
    if any(seg[-1] in ". " for seg in segments):
        raise ValueError("file path segments must not end in '.' or ' ' (a Windows station "
                         "strips them, aliasing another name): %r" % path)
    if path.lower().endswith(REFUSED_SUFFIXES):
        raise ValueError("the station never serves .bog / .bog.gz files (FileServlet); "
                         "use the offline config.bog route instead")
    return path


def url_path(path):
    """`/file/<path>` with each segment URL-encoded (spaces, `#`, `?`, `%` included)."""
    return "/file/" + "/".join(urllib.parse.quote(seg, safe="") for seg in path.split("/"))


def is_text(path, content_type):
    """True when the file is returned as text: a known text extension or a text type."""
    ext = path.rsplit(".", 1)[-1].lower() if "." in path.rsplit("/", 1)[-1] else ""
    if ext in TEXT_EXTENSIONS:
        return True
    kind = (content_type or "").split(";", 1)[0].strip().lower()
    return kind.startswith("text/") or kind.endswith(("+xml", "/xml", "/json"))


# ---- Px summary -----------------------------------------------------------

def _is_ord(value):
    return value.startswith(("station:|", "slot:", "h:", "local:|", "history:", "ip:",
                             "file:", "module:", "bql:", "service:"))


def _is_slot_ord(value):
    return value.startswith(("station:|slot:", "slot:", "local:|slot:"))


def _is_image(value):
    return value.lower().rsplit("?", 1)[0].endswith((".png", ".gif", ".jpg", ".jpeg",
                                                      ".svg", ".bmp", ".ico"))


def bound_component(ord_text):
    """The component ORD a binding addresses: a trailing value slot or slot hop removed.

    A heuristic: `.../Point/out` and `.../Point|slot:out` give `.../Point`. A last segment
    that is not a known value slot (`VALUE_SLOTS`) is taken as the component itself.
    """
    hop = _SLOT_HOP.search(ord_text)
    if hop:
        return ord_text[:hop.start()]
    head, sep, last = ord_text.rpartition("/")
    if sep and last in VALUE_SLOTS and not head.endswith(":"):
        return head
    return ord_text


class _Bounded:
    """An ordered, de-duplicated list capped at `limit`; `truncated` once one is dropped."""

    def __init__(self, limit):
        self.limit, self.items, self.truncated = limit, [], False

    def add(self, value):
        if value in self.items:
            return
        if len(self.items) >= self.limit:
            self.truncated = True
            return
        self.items.append(value)


def parse_px(text, max_items=DEFAULT_MAX_ITEMS):
    """Summarize one `.px` document (stdlib XML only); ValueError when it is not Px XML.

    Returns media, imported modules, canvases and their `viewSize`, layer names, widget and
    binding type counts, bound ORDs grouped by component, images, included Px files, charts,
    other ORDs and label texts. Every list is capped at `max_items` (`<list>_truncated`).
    A DOCTYPE or ENTITY declaration is refused before parsing (no entity expansion).
    """
    if not isinstance(text, str):
        raise ValueError("px text must be a string")
    if re.search(r"<!(DOCTYPE|ENTITY)", text, re.I):
        raise ValueError("refusing a Px document with a DOCTYPE or ENTITY declaration")
    try:
        root = ET.fromstring(text)
    except ET.ParseError as exc:
        raise ValueError("not well-formed Px XML: %s" % exc) from None
    if root.tag != "px":
        raise ValueError("not a Px document: the root element is <%s>, expected <px>"
                         % root.tag)
    imports = _Bounded(max_items)
    for module in root.iter("module"):
        if module.get("name"):
            imports.add(module.get("name"))
    content = root.find("content")
    scope = content if content is not None else root
    widgets, bindings = Counter(), Counter()
    canvases, layers = _Bounded(max_items), _Bounded(max_items)
    images, included, labels = _Bounded(max_items), _Bounded(max_items), _Bounded(max_items)
    others, charts = _Bounded(max_items), _Bounded(max_items)
    bound = {}
    bound_cut = False
    for elem in scope.iter():
        if elem is scope:
            continue
        tag = elem.tag
        if tag.endswith("Binding"):
            bindings[tag] += 1
        elif tag in ("layer", "Layer"):
            if elem.get("name"):
                layers.add(elem.get("name"))
        else:
            widgets[tag] += 1
        if elem.get("viewSize"):
            canvases.add({"type": tag, "name": elem.get("name"),
                          "view_size": elem.get("viewSize")})
        if elem.get("layer"):
            layers.add(elem.get("layer"))
        if elem.get("text") is not None and elem.get("text").strip():
            labels.add(elem.get("text"))
        if "Chart" in tag:
            chart_ords = [v for sub in elem.iter() for v in sub.attrib.values() if _is_ord(v)]
            charts.add({"type": tag, "name": elem.get("name"), "ords": chart_ords})
        for name, value in elem.attrib.items():
            value = value.strip()
            if not _is_ord(value) and not value.startswith("module://"):
                continue
            if _is_slot_ord(value):
                key = bound_component(value)
                if key not in bound and len(bound) >= max_items:
                    bound_cut = True
                    continue
                group = bound.setdefault(key, [])
                if value not in group:
                    group.append(value)
            elif tag == "PxInclude" or (value.startswith("file:^") and
                                        value.lower().endswith(".px")):
                included.add(value)
            elif _is_image(value) or name == "image":
                images.add(value)
            else:
                others.add(value)
    canvas_list = canvases.items
    out = {"version": root.get("version"), "media": root.get("media"),
           "imports": imports.items, "canvases": canvas_list,
           "view_size": canvas_list[0]["view_size"] if canvas_list else None,
           "layers": layers.items, "widget_counts": dict(widgets),
           "binding_counts": dict(bindings), "bound_components": bound,
           "images": images.items, "included_px": included.items, "charts": charts.items,
           "other_ords": others.items, "labels": labels.items}
    for key, bounded in (("imports", imports), ("canvases", canvases), ("layers", layers),
                         ("images", images), ("included_px", included), ("charts", charts),
                         ("other_ords", others), ("labels", labels)):
        if bounded.truncated:
            out[key + "_truncated"] = True
    if bound_cut:
        out["bound_components_truncated"] = True
    return out
