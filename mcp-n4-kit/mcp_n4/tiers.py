"""Version tiers (METHODOLOGY section 5, B1197 section 1197.6): pure classification and policy.

- A: 4.13 and 4.14 (BOX 2.3 with `checkLinks`): full support.
- B: 4.15 and 4.3: read-only until a PoC matches this build; writes need operator opt-in.
- C: any other or undetected build: writes only after `reg.loadContract`, `loadRoot` and a
  harmless scratch write succeeded. The server has no probe tool, so the operator records
  that outcome with an opt-in.

Importing this module has no side effects.
"""
import re

from . import safety

TIER_A = frozenset({(4, 13), (4, 14)})
TIER_B = frozenset({(4, 15), (4, 3)})
#: Where the version comes from: oBIX `productVersion` (live-certified, B457/B1199).
ABOUT_PATH = "/obix/about/"
OPT_IN_FLAGS = {"B": "--allow-tier-b", "C": "--allow-tier-c"}

_VERSION = re.compile(r"([0-9]+)\.([0-9]+)(?:\.[0-9]+)*")
_STR_TAG = re.compile(r"<str\b[^>]*>")
_ATTR = re.compile(r'\b(name|val)\s*=\s*"([^"]*)"')


def parse_version(text):
    """`(major, minor)` of a dotted version such as `4.14.0.162`, else None."""
    if not isinstance(text, str):
        return None
    match = _VERSION.fullmatch(text.strip())
    return (int(match.group(1)), int(match.group(2))) if match else None


def tier_of(version):
    """`A`, `B` or `C`; an unknown or unparseable version is `C`."""
    key = parse_version(version)
    if key in TIER_A:
        return "A"
    if key in TIER_B:
        return "B"
    return "C"


def product_version(about_xml):
    """`productVersion` from an oBIX About document, else None (attribute order free)."""
    if not isinstance(about_xml, str):
        return None
    for tag in _STR_TAG.findall(about_xml):
        attrs = dict(_ATTR.findall(tag))
        if attrs.get("name") == "productVersion" and attrs.get("val"):
            return attrs["val"]
    return None


def write_block(tier, version, station, opt_in):
    """Why this tier refuses writes on `station`, or None when they are allowed.

    `opt_in` maps a tier (`B`/`C`) to the station names the operator opted in at start.
    The message is possibility-first: it names every route to the write.
    """
    if tier == "A" or station in (opt_in or {}).get(tier, ()):
        return None
    shown = version if version else "unknown (no oBIX productVersion at %s)" % ABOUT_PATH
    routes = ("Routes: (1) dry runs stay allowed here, so the plan can be reviewed now; "
              "(2) make the change in Workbench; ")
    if tier == "B":
        return ("%s: station %s is tier B (version %s): it stays read-only until a PoC on a "
                "scratch station matches this build (METHODOLOGY section 5). %s(3) once the "
                "PoC matches, restart the server with `%s %s`."
                % (safety.REASON_TIER, station, shown, routes, OPT_IN_FLAGS["B"], station))
    return ("%s: station %s is tier C (version %s): writes need the probe sequence first "
            "(METHODOLOGY section 5): `reg.loadContract` and `loadRoot` succeed and one "
            "harmless scratch write (create, read back, remove in a scratch folder) is "
            "verified on this build. The server has no probe tool. %s(3) run that probe "
            "(e.g. in Workbench or a PoC script), then restart the server with `%s %s`."
            % (safety.REASON_TIER, station, shown, routes, OPT_IN_FLAGS["C"], station))
