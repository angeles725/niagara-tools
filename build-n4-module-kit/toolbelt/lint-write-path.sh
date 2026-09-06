#!/usr/bin/env bash
# lint-write-path.sh — write-path matrix coverage lint for Niagara N4 modules (Campaign 8 PR19, B816).
# [ev: retro campaign8-write-path]
#
# Checks that every @NiagaraProperty with Flags.OPERATOR (or "o") has a ROW in
# docs/write-path-matrix.md (D16: row-presence only; CHECK12 is bog-audit's concern).
# With --bog, adds link-traced dashboard/RoomPanel target slots to the required set.
#
# Usage:  lint-write-path.sh <module-root> [--bog <config.bog>]
#
#   Row format:  FAIL  lint-write-path  <module>  slot <name>: no matrix row
#   Exits:       0  all covered · 1  any uncovered · 3  usage/env (K20)
#
# A comment mention of a slot name does NOT satisfy the matrix row requirement (R19.3).
# Dot-directories pruned (D9b). VCS-free by design.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: corpus B816]
set -u

FAILED=0
BOG_FILE=""

# ---------------------------------------------------------------------------
# Usage guard (exit 3 when no module-root given — K20 disjoint exit codes)
# ---------------------------------------------------------------------------
if [ $# -lt 1 ]; then
    printf 'usage: lint-write-path.sh <module-root> [--bog <config.bog>]\n' >&2
    exit 3
fi

MODULE_ROOT="$1"
shift

while [ $# -gt 0 ]; do
    case "$1" in
        --bog)
            shift
            if [ $# -lt 1 ]; then
                printf 'lint-write-path: --bog requires a file argument\n' >&2
                exit 3
            fi
            BOG_FILE="$1"
            shift
            ;;
        *)
            printf 'lint-write-path: unknown option: %s\n' "$1" >&2
            exit 3
            ;;
    esac
done

if [ ! -d "$MODULE_ROOT" ]; then
    printf 'lint-write-path: not a directory: %s\n' "$MODULE_ROOT" >&2
    exit 3
fi

MATRIX="$MODULE_ROOT/docs/write-path-matrix.md"
SRC_DIR="$MODULE_ROOT/src"

# ---------------------------------------------------------------------------
# 1. Collect OPERATOR-flagged slot names from @NiagaraProperty annotations.
#    Uses paren-balance multi-line state machine — same technique as slot-coverage.sh.
#    Dot-directories pruned (D9b).
# ---------------------------------------------------------------------------
# shellcheck disable=SC2016  # awk programs intentionally in single quotes (no shell expansion needed)
if [ -d "$SRC_DIR" ]; then
    _java_files=$(find "$SRC_DIR" \
        -type d -name '.*' -prune \
        -o -name '*.java' -print \
        2>/dev/null | sort)
else
    _java_files=""
fi

if [ -n "$_java_files" ]; then
    # shellcheck disable=SC2016
    _op_slots=$(printf '%s\n' "$_java_files" | xargs awk '
        BEGIN { in_prop=0; prop_buf=""; prop_name=""; prop_op=0 }
        {
            ln = $0
            if (!in_prop) {
                if (index(ln, "@NiagaraProperty") > 0) {
                    in_prop=1; prop_buf=ln; prop_name=""; prop_op=0
                }
            } else {
                prop_buf = prop_buf " " ln
            }
            if (in_prop) {
                if (prop_name == "" && match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
                    seg = substr(prop_buf, RSTART)
                    sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
                    sub(/".*/, "", seg)
                    prop_name = seg
                }
                if (!prop_op && (index(prop_buf, "OPERATOR") > 0 || index(prop_buf, "\"o\"") > 0)) {
                    prop_op = 1
                }
                depth=0; n=length(prop_buf)
                for (ci=1; ci<=n; ci++) {
                    c = substr(prop_buf, ci, 1)
                    if      (c == "(") depth++
                    else if (c == ")") depth--
                }
                if (depth <= 0 && index(prop_buf, "@NiagaraProperty") > 0) {
                    if (prop_name != "" && prop_op) print prop_name
                    in_prop=0; prop_buf=""; prop_name=""; prop_op=0
                }
            }
        }
        ' 2>/dev/null | sort -u)
else
    _op_slots=""
fi

# ---------------------------------------------------------------------------
# 2. Collect covered slot names from write-path-matrix.md.
#    A row is a |..| line whose first cell begins with a lowercase letter (camelCase).
#    Comment lines and header/separator rows are not data rows (R19.3).
# ---------------------------------------------------------------------------
_matrix_slots=""
if [ -f "$MATRIX" ]; then
    # Extract first pipe-delimited cell from |..| lines; keep only camelCase slot names.
    # - Separator rows: first cell is only [-:] chars => filtered by /^[a-z]/ gate.
    # - Header rows: first cell starts with uppercase (e.g. "Slot", "Writable Slot") => filtered.
    # - Comment lines: do not start with | => not matched.
    # shellcheck disable=SC2016
    _matrix_slots=$(awk -F'|' '
        /^\|/ {
            cell = $2
            gsub(/^[[:space:]`"]+|[[:space:]`"]+$/, "", cell)
            if (cell ~ /^[a-z]/) print cell
        }
    ' "$MATRIX" 2>/dev/null | sort -u)
fi

# ---------------------------------------------------------------------------
# 3. If --bog: add link-traced dashboard/RoomPanel target slots to required set.
#    Reuses the bog grammar from bog-audit.sh as a python3 inline snippet (D16).
# ---------------------------------------------------------------------------
if [ -n "$BOG_FILE" ]; then
    command -v python3 >/dev/null 2>&1 || {
        printf 'lint-write-path: python3 not found (required for --bog parsing)\n' >&2
        exit 3
    }
    _TMP=$(mktemp -d)
    trap 'rm -rf "$_TMP"' EXIT

    # Extract targetSlotName of links: SOURCE is a Dashboard/RoomPanel component,
    # TARGET is an own-module component. Those target slots need matrix rows.
    _bog_slots=$(BOG_FILE="$BOG_FILE" MODULE_ROOT="$MODULE_ROOT" python3 <<'PYEOF' 2>/dev/null
"""
lint-write-path --bog helper: extract link target slots from dashboard into own-module components.
Reuses the same bog grammar (attribute regex) as bog-audit.sh (D16 — no second parser).
"""
import sys, os, re, zipfile
from xml.etree import ElementTree as ET

bog_file  = os.environ.get('BOG_FILE', '')
mod_root  = os.environ.get('MODULE_ROOT', '').rstrip('/')

# ---- load bog XML ----
xml_content = None
try:
    if zipfile.is_zipfile(bog_file):
        with zipfile.ZipFile(bog_file) as z:
            for name in z.namelist():
                if name == 'file.xml' or name.endswith('/file.xml'):
                    xml_content = z.read(name).decode('utf-8', errors='replace')
                    break
    else:
        xml_content = open(bog_file, encoding='utf-8', errors='replace').read()
except Exception:
    sys.exit(0)

if not xml_content:
    sys.exit(0)

def ga(text, name):
    """Get XML attribute — same helper as bog-audit.py."""
    m = re.search(rf"\b{re.escape(name)}='([^']*)'", text)
    if m: return m.group(1)
    m = re.search(rf'\b{re.escape(name)}="([^"]*)"', text)
    return m.group(1) if m else None

# ---- find own-module name from module-include.xml ----
own_mod = None
import glob
for xf in sorted(glob.glob(os.path.join(mod_root, '**', 'module-include.xml'), recursive=True)):
    if any(part.startswith('.') for part in xf.split(os.sep)):
        continue
    try:
        tree = ET.parse(xf)
        name_attr = (tree.getroot().get('name') or
                     tree.getroot().get('moduleName'))
        if name_attr:
            own_mod = name_attr
            break
    except Exception:
        continue
if not own_mod:
    # fallback: strip profile suffix from directory name
    own_mod = re.sub(r'-(rt|wb|ux)$', '', os.path.basename(mod_root))

# ---- build handle->module map ----
comp_module = {}  # handle -> module name
for m in re.finditer(r'<[^>]+>', xml_content):
    tag = m.group(0)
    h   = ga(tag, 'handle')
    mod = ga(tag, 'module')
    if h and mod:
        comp_module[h] = mod

# ---- collect targetSlotNames of dashboard->own-module links ----
dashboard_re = re.compile(r'Dashboard|RoomPanel|DashPanel', re.I)
target_slots = set()
for m in re.finditer(r'<link\b[^>]*/?\s*>', xml_content, re.DOTALL):
    tag      = m.group(0)
    src_h    = ga(tag, 'sourceOrd')
    tgt_h    = ga(tag, 'targetOrd')
    tgt_slot = ga(tag, 'targetSlotName')
    if not (src_h and tgt_h and tgt_slot):
        continue
    src_mod = comp_module.get(src_h, '')
    tgt_mod = comp_module.get(tgt_h, '')
    if dashboard_re.search(src_mod) and tgt_mod == own_mod:
        target_slots.add(tgt_slot)

for s in sorted(target_slots):
    print(s)
PYEOF
)

    if [ -n "$_bog_slots" ]; then
        _op_slots=$(printf '%s\n%s\n' "$_op_slots" "$_bog_slots" | sort -u | grep -v '^$')
    fi
fi

# ---------------------------------------------------------------------------
# 4. Report uncovered slots (D16: row presence only; FAIL names the slot).
# ---------------------------------------------------------------------------
[ -z "$_op_slots" ] && exit 0

_MODULE_NAME=$(basename "$MODULE_ROOT")

while IFS= read -r slot; do
    [ -z "$slot" ] && continue
    if ! printf '%s\n' "$_matrix_slots" | grep -qx "$slot"; then
        printf 'FAIL  lint-write-path  %s  slot %s: no matrix row\n' \
            "$_MODULE_NAME" "$slot"
        FAILED=1
    fi
done <<EOF
$_op_slots
EOF

exit "$FAILED"
