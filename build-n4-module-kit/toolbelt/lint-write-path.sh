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
    # Extract slot name from the first pipe-delimited cell of every |..| table row.
    # The slot name is the FIRST backtick-quoted identifier in the cell, or the bare word
    # if no backticks are present.  Only pure Java identifiers (^[a-z][A-Za-z0-9]+?$)
    # are emitted; this filters out header rows ("Slot", "Writable Slot"), separator rows
    # (---), comment lines (not |..| lines), and descriptive prose cells like
    # "coil sensor" or "both suction sensors = NaN invalid".
    # Accepts ≥ 4 columns; matches on slot-name column only (D16).
    # shellcheck disable=SC2016
    _matrix_slots=$(awk -F'|' '
        /^\|/ {
            cell = $2
            gsub(/^[[:space:]]+|[[:space:]]+$/, "", cell)
            # If first char is a backtick: extract the content of the first `...` pair.
            if (substr(cell,1,1) == "`") {
                sub(/^`/, "", cell)
                sub(/`.*/, "", cell)
            } else {
                # No backtick: take only the leading word (stop at space, backtick, or punctuation).
                sub(/[[:space:]`\/(].*$/, "", cell)
            }
            # Keep only pure Java identifiers: starts with lowercase letter, alphanumeric only.
            if (cell ~ /^[a-z][A-Za-z0-9]*$/) print cell
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
    # Reuses the same bog grammar (line-based attribute regex) as bog-audit.sh (D16).
    _bog_slots=$(BOG_FILE="$BOG_FILE" MODULE_ROOT="$MODULE_ROOT" python3 <<'PYEOF' 2>/dev/null
"""
lint-write-path --bog helper: extract link target slots from dashboard into own-module components.
Line-based bog grammar — same approach as bog-audit.sh (D16 — no second parser).
"""
import sys, os, re, zipfile

bog_file = os.environ.get('BOG_FILE', '')
mod_root = os.environ.get('MODULE_ROOT', '').rstrip('/')

# fallback own-module name from directory (strip -rt/-wb/-ux profile suffix)
own_mod = re.sub(r'-(rt|wb|ux)$', '', os.path.basename(mod_root))

# ---- load bog XML as lines ----
try:
    if zipfile.is_zipfile(bog_file):
        with zipfile.ZipFile(bog_file) as z:
            for entry in z.namelist():
                if entry == 'file.xml' or entry.endswith('/file.xml'):
                    lines = z.read(entry).decode('utf-8', errors='replace').splitlines()
                    break
            else:
                sys.exit(0)
    else:
        with open(bog_file, encoding='utf-8', errors='replace') as fh:
            lines = fh.read().splitlines()
except Exception:
    sys.exit(0)

TAG_RE = re.compile(r'<(/?)([A-Za-z]\w*)\b([^>]*?)(/?)>')

def ga(full, name):
    """Get XML attribute — same helper as bog-audit.sh embedded engine."""
    m = re.search(rf"\b{re.escape(name)}='([^']*)'", full)
    if m: return m.group(1)
    m = re.search(rf'\b{re.escape(name)}="([^"]*)"', full)
    return m.group(1) if m else None

# ---- line-based parse (same structure as bog-audit.sh) ----
prefix_map   = {}   # pfx -> module_name
handle_mod   = {}   # handle -> module_name
# Stack: every non-self-closing <p> is pushed (with module='' when unknown)
# so that </p> pops always balance.
stack        = []   # list of dicts: {'mod': str, 'h': str|None}
in_link      = False
link_buf     = {}
dashboard_re = re.compile(r'Dashboard|RoomPanel|DashPanel', re.I)
target_slots = set()

for raw in lines:
    line = raw.strip()
    if not line:
        continue
    for m in TAG_RE.finditer(line):
        is_closing = bool(m.group(1))
        tag_name   = m.group(2)
        is_self    = bool(m.group(4)) or m.group(0).endswith('/>')
        full       = m.group(0)

        # ---- module prefix registration ----
        m_attr = ga(full, 'm') or ''
        if m_attr:
            for part in m_attr.split():
                if '=' in part:
                    pk, mv = part.split('=', 1)
                    prefix_map[pk] = mv

        # ---- closing tag: pop stack ----
        if is_closing:
            if tag_name in ('p', 'a') and stack:
                popped = stack.pop()
                if popped.get('link'):
                    # finalize link
                    src_h    = link_buf.get('src_h')
                    tgt_slot = link_buf.get('tgt_slot')
                    if src_h and tgt_slot:
                        src_mod     = handle_mod.get(src_h, '')
                        # nearest enclosing component = target module
                        tgt_mod = next(
                            (fr['mod'] for fr in reversed(stack) if fr.get('mod')), '')
                        if dashboard_re.search(src_mod) and tgt_mod == own_mod:
                            target_slots.add(tgt_slot)
                    in_link  = False
                    link_buf = {}
            continue

        if tag_name not in ('p', 'a'):
            continue

        # ---- action <a> ----
        if tag_name == 'a':
            continue

        n      = ga(full, 'n') or ''
        h      = ga(full, 'h')
        t      = ga(full, 't') or ''
        v      = ga(full, 'v')

        # ---- link child elements ----
        if in_link and is_self:
            if n == 'sourceOrd' and v and v.startswith('h:'):
                link_buf['src_h'] = v[2:]
            elif n == 'sourceSlotName' and v:
                link_buf['src_slot'] = v
            elif n == 'targetSlotName' and v:
                link_buf['tgt_slot'] = v
            continue

        # ---- link wrapper <p t='b:Link'> ----
        if t == 'b:Link' and not is_self:
            in_link  = True
            link_buf = {'src_h': None, 'src_slot': None, 'tgt_slot': None}
            stack.append({'mod': '', 'h': None, 'link': True})
            continue

        # ---- component or property node ----
        if not is_self:
            mod = ''
            if h is not None:
                pfx = t.split(':')[0] if ':' in t else ''
                mod = prefix_map.get(pfx, '')
                if mod:
                    handle_mod[h] = mod
            stack.append({'mod': mod, 'h': h, 'link': False})

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
