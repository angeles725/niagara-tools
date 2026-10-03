#!/usr/bin/env bash
# lint-write-path.sh — write-path matrix coverage lint for Niagara N4 modules (Campaign 8 PR19, B816).
# [ev: retro campaign8-write-path]
#
# Checks that every @NiagaraProperty with Flags.OPERATOR (or "o") has a ROW in
# docs/write-path-matrix.md (D16: row-presence only; CHECK12 is bog-audit's concern).
# With --bog, adds link-traced dashboard/RoomPanel target slots to the required set.
#
# Usage:  lint-write-path.sh <module-root> [--bog <config.bog>] [--matrix <path>] [--strict]
#
# Module-root convention:
#   - If <root>/src exists: scan it directly (single-profile mode).
#   - If not: scan every *-rt / *-ux / *-wb / *-se profile subdir found immediately
#     under <root>, reporting each under its profile name.
#   - If no Java source is found anywhere: ERROR + exit 3.
#
# Matrix resolution (when --matrix is not given):
#   1. Walk up from <module-root> looking for a dir whose docs/write-path-matrix.md
#      contains at least one table row (line starting with |).
#   2. Walk stops at the first .git directory (vcs root) or filesystem root.
#   3. If no valid matrix is found: ERROR + exit 3.
#
# Audit-call check (WARN): every servlet write handler — a doPost / doPut / doDelete / doPatch
# method declared in a scanned src file — must record the write in the audit store before it
# answers success. Heuristic (file scope, comments stripped outside string literals, literal
# contents blanked): the handler file must contain a call
# to an identifier containing "audit" (e.g. appendAudit(, auditLog.record(, buildAuditEntry();
# getters (get*/is*) and comment mentions do not count; a handler that delegates the write to
# another class must audit there AND call it from the handler file. Advisory for legacy modules;
# --strict promotes it to exit 1. Contract: types/security.md § Servlet write audit.
# [ev: retro servlet-write-audit Δ2]
#
#   Row format:  FAIL  lint-write-path  <module>  slot <name>: no matrix row
#                WARN  lint-write-path  <file>:<line>  write handler <method>: no audit call
#   Error format: lint-write-path  ERROR  <module-root>  <reason>
#   Exits: 0 all covered (STALE/DRIFT/audit WARN advisory) · 1 any uncovered, or any STALE/DRIFT/audit WARN under --strict · 3 usage/env/missing-matrix or a sub-directory find cannot enter (K20)
#
# A comment mention of a slot name does NOT satisfy the matrix row requirement (R19.3).
# Dot-directories pruned (D9b). VCS-free by design.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: corpus B816]
# Mutation: WP-stale-concept-decoy -- moves concept marker strip after the test, making an HTML-comment [concept] produce a false STALE
# Mutation: WP-stale-perrow -- removes per-row check, allowing a [concept]-marked row to exempt a co-resident plain stale row
# Mutation: WP-drift-decoy -- removes HTML-comment strip, causing a commented [concept] to produce a false DRIFT
# Mutation: WP-audit -- drops the audit-call WARN, so a servlet write handler that never records the write passes
# Mutation: WP-audit-ok -- drops the audit-call search, so an audited handler gets a false WARN
# Mutation: WP-audit-str-ok -- strips comments without tracking string literals, so "*/*" hides the audit call
# Mutation: WP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports the slot covered
# Mutation: WP-finderr-matrix -- ignoring the matrix-root find status drops names from the covered set
set -u

FAILED=0
STRICT=0
AUDIT=0
STALE=0
DRIFT=0
BOG_FILE=""
MATRIX_OVERRIDE=""

# ---------------------------------------------------------------------------
# Argument parsing.  --strict may appear before or after <module-root>.
# A missing module-root is a usage error: exit 3 (K20 disjoint exit codes).
# ---------------------------------------------------------------------------
MODULE_ROOT=""

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
        --matrix)
            shift
            if [ $# -lt 1 ]; then
                printf 'lint-write-path: --matrix requires a file argument\n' >&2
                exit 3
            fi
            MATRIX_OVERRIDE="$1"
            shift
            ;;
        --strict)
            STRICT=1; shift ;;
        -*)
            printf 'lint-write-path: unknown option: %s\n' "$1" >&2
            exit 3
            ;;
        *)
            if [ -z "$MODULE_ROOT" ]; then
                MODULE_ROOT="$1"
                shift
            else
                printf 'lint-write-path: unexpected argument: %s\n' "$1" >&2
                exit 3
            fi
            ;;
    esac
done

if [ -z "$MODULE_ROOT" ]; then
    printf 'usage: lint-write-path.sh <module-root> [--bog <config.bog>] [--matrix <path>] [--strict]\n' >&2
    exit 3
fi

if [ ! -d "$MODULE_ROOT" ]; then
    printf 'lint-write-path: not a directory: %s\n' "$MODULE_ROOT" >&2
    exit 3
fi

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
# list_java <out> <root> [--prune <name>]... — fail-closed walk: a sub-directory find cannot
# enter is an ERROR row + exit 3, never a shorter file list (a skipped file hides its slots).
list_java() {
    local out="$1" root="$2"
    shift 2
    if ! scan_files "$out" "$_TMP/find.err" "$root" "$@" -name '*.java'; then
        printf 'lint-write-path  ERROR  %s  cannot list every file under %s: %s\n' \
            "$MODULE_ROOT" "$root" "$(head -n 1 "$_TMP/find.err")"
        exit 3
    fi
}

# ---------------------------------------------------------------------------
# Matrix resolution: --matrix override, or walk up to find docs/write-path-matrix.md.
# "Found" means the file exists AND has at least one table row (line starting with |).
# An empty or row-free file is treated as absent (could be a stale artifact).
# Walk stops at the first .git directory (vcs root) or filesystem root.
# Exits 3 with ERROR if no valid matrix is found.
# ---------------------------------------------------------------------------
if [ -n "$MATRIX_OVERRIDE" ]; then
    MATRIX="$MATRIX_OVERRIDE"
    if [ ! -f "$MATRIX" ]; then
        printf 'lint-write-path  ERROR  %s  --matrix file not found: %s\n' "$MODULE_ROOT" "$MATRIX"
        exit 3
    fi
else
    MATRIX=""
    _looked=""
    _dir="$MODULE_ROOT"
    while true; do
        _candidate="$_dir/docs/write-path-matrix.md"
        if [ -z "$_looked" ]; then
            _looked="$_candidate"
        else
            _looked="$_looked, $_candidate"
        fi
        # "Found" requires the file to exist AND contain at least one table row (|).
        # An absent or empty file is not a valid matrix — keep walking up.
        if [ -f "$_candidate" ] && grep -q '^\|' "$_candidate" 2>/dev/null; then
            MATRIX="$_candidate"
            break
        fi
        # Stop at vcs root (.git directory present at this level)
        if [ -d "$_dir/.git" ]; then
            break
        fi
        _parent=$(dirname "$_dir")
        # Stop at filesystem root (dirname of / is /)
        if [ "$_parent" = "$_dir" ]; then
            break
        fi
        _dir="$_parent"
    done
    if [ -z "$MATRIX" ]; then
        printf 'lint-write-path  ERROR  %s  no write-path-matrix.md found (looked in %s)\n' \
            "$MODULE_ROOT" "$_looked"
        exit 3
    fi
fi

# ---------------------------------------------------------------------------
# Matrix root: the directory whose docs/write-path-matrix.md was found.
# = dirname(dirname(MATRIX)) since MATRIX = <root>/docs/write-path-matrix.md.
# Used for matrix-root-wide covered-set harvest (STALE direction).
# ---------------------------------------------------------------------------
MATRIX_ROOT="$(dirname "$(dirname "$MATRIX")")"

# ---------------------------------------------------------------------------
# Matrix-root-wide covered set (for STALE detection only).
# Includes ALL @NiagaraProperty and @NiagaraAction names (any flag) from
# EVERY Java source under the matrix root, build/ and dot-dirs pruned (D9b).
# Multi-line annotations put name = "X" on its own line — match the field,
# not @Niagara… on the same line.  NOT the per-module OPERATOR-only scanner.
# ---------------------------------------------------------------------------
list_java "$_TMP/matrix-java" "$MATRIX_ROOT" --prune build
_all_matrix_java=$(cat "$_TMP/matrix-java")
_covered_names=""
if [ -n "$_all_matrix_java" ]; then
    _covered_names=$(printf '%s\n' "$_all_matrix_java" | \
        xargs grep -hoE 'name[[:space:]]*=[[:space:]]*"[A-Za-z][A-Za-z0-9_]*"' 2>/dev/null | \
        sed -E 's/.*"([^"]+)".*/\1/' | sort -u)
fi

# ---------------------------------------------------------------------------
# Determine scan targets.
# Single-profile mode: <root>/src exists → one scan named after basename(root).
# Module-root mode:   <root>/src absent → iterate *-rt/*-ux/*-wb/*-se profile subdirs.
# If no Java source is found anywhere under the root: ERROR + exit 3.
# ---------------------------------------------------------------------------
_SCAN_NAMES=()
_SCAN_SRCS=()

if [ -d "$MODULE_ROOT/src" ]; then
    _SCAN_NAMES+=("$(basename "$MODULE_ROOT")")
    _SCAN_SRCS+=("$MODULE_ROOT/src")
else
    while IFS= read -r _pd; do
        [ -d "$_pd/src" ] || continue
        _SCAN_NAMES+=("$(basename "$_pd")")
        _SCAN_SRCS+=("$_pd/src")
    done < <(find "$MODULE_ROOT" -maxdepth 1 -mindepth 1 -type d \
        \( -name '*-rt' -o -name '*-ux' -o -name '*-wb' -o -name '*-se' \) \
        2>/dev/null | sort)

    if [ "${#_SCAN_SRCS[@]}" -eq 0 ]; then
        printf 'lint-write-path  ERROR  %s  no src found\n' "$MODULE_ROOT"
        exit 3
    fi
fi

# ---------------------------------------------------------------------------
# Extract covered slot names from the matrix (shared across all profiles).
# A row is a |..| line whose first cell is a pure Java identifier.
# Comment lines and header/separator rows are not data rows (R19.3).
# ---------------------------------------------------------------------------
# shellcheck disable=SC2016
_matrix_slots=$(awk -F'|' '
    /^\|/ {
        cell = $2
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", cell)
        if (substr(cell,1,1) == "`") {
            sub(/^`/, "", cell)
            sub(/`.*/, "", cell)
        } else {
            sub(/[[:space:]`\/(].*$/, "", cell)
        }
        if (cell ~ /^[a-z][A-Za-z0-9]*$/) print cell
    }
' "$MATRIX" 2>/dev/null | sort -u)

# ---------------------------------------------------------------------------
# --bog: pre-compute bog-linked slots once using the module root.
# The python3 helper derives own_mod from the MODULE_ROOT directory name (stripping
# any -rt/-ux/-wb suffix), so it works correctly for both single-profile and
# module-root invocations.
# ---------------------------------------------------------------------------
_bog_extra=""
if [ -n "$BOG_FILE" ]; then
    command -v python3 >/dev/null 2>&1 || {
        printf 'lint-write-path: python3 not found (required for --bog parsing)\n' >&2
        exit 3
    }
    _bog_extra=$(BOG_FILE="$BOG_FILE" MODULE_ROOT="$MODULE_ROOT" python3 <<'PYEOF' 2>/dev/null
"""
lint-write-path --bog helper: extract link target slots from dashboard into own-module components.
Line-based bog grammar — same approach as bog-audit.sh (D16 — no second parser).
"""
import sys, os, re, zipfile

bog_file = os.environ.get('BOG_FILE', '')
mod_root = os.environ.get('MODULE_ROOT', '').rstrip('/')

# Derive own-module name: strip -rt/-ux/-wb/-se profile suffix if present.
own_mod = re.sub(r'-(rt|wb|ux|se)$', '', os.path.basename(mod_root))

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
    m = re.search(rf"\b{re.escape(name)}='([^']*)'", full)
    if m: return m.group(1)
    m = re.search(rf'\b{re.escape(name)}="([^"]*)"', full)
    return m.group(1) if m else None

prefix_map   = {}
handle_mod   = {}
stack        = []
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

        m_attr = ga(full, 'm') or ''
        if m_attr:
            for part in m_attr.split():
                if '=' in part:
                    pk, mv = part.split('=', 1)
                    prefix_map[pk] = mv

        if is_closing:
            if tag_name in ('p', 'a') and stack:
                popped = stack.pop()
                if popped.get('link'):
                    src_h    = link_buf.get('src_h')
                    tgt_slot = link_buf.get('tgt_slot')
                    if src_h and tgt_slot:
                        src_mod = handle_mod.get(src_h, '')
                        tgt_mod = next(
                            (fr['mod'] for fr in reversed(stack) if fr.get('mod')), '')
                        if dashboard_re.search(src_mod) and tgt_mod == own_mod:
                            target_slots.add(tgt_slot)
                    in_link  = False
                    link_buf = {}
            continue

        if tag_name not in ('p', 'a'):
            continue
        if tag_name == 'a':
            continue

        n = ga(full, 'n') or ''
        h = ga(full, 'h')
        t = ga(full, 't') or ''
        v = ga(full, 'v')

        if in_link and is_self:
            if n == 'sourceOrd' and v and v.startswith('h:'):
                link_buf['src_h'] = v[2:]
            elif n == 'sourceSlotName' and v:
                link_buf['src_slot'] = v
            elif n == 'targetSlotName' and v:
                link_buf['tgt_slot'] = v
            continue

        if t == 'b:Link' and not is_self:
            in_link  = True
            link_buf = {'src_h': None, 'src_slot': None, 'tgt_slot': None}
            stack.append({'mod': '', 'h': None, 'link': True})
            continue

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
fi

# ---------------------------------------------------------------------------
# Per-profile scan: extract OPERATOR slots, merge bog extras, report FAILs.
# Bog extras are added only to profiles that have OPERATOR slots (to avoid
# false FAILs on -ux/-wb profiles with no runtime slot annotations).
# ---------------------------------------------------------------------------
# shellcheck disable=SC2016
_AWK_SCANNER='
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
'

_idx=0
for _scan_name in "${_SCAN_NAMES[@]}"; do
    _scan_src="${_SCAN_SRCS[$_idx]}"
    _idx=$(( _idx + 1 ))

    # Collect OPERATOR slots from this profile's src/.
    list_java "$_TMP/src.$_idx" "$_scan_src"
    _java_files=$(cat "$_TMP/src.$_idx")

    if [ -n "$_java_files" ]; then
        # shellcheck disable=SC2016
        _op_slots=$(printf '%s\n' "$_java_files" | xargs awk "$_AWK_SCANNER" 2>/dev/null | sort -u)
    else
        _op_slots=""
    fi

    # Add bog-derived required slots only when this profile already has OPERATOR slots
    # (bog links target runtime slots; -ux/-wb profiles have none).
    if [ -n "$_op_slots" ] && [ -n "$_bog_extra" ]; then
        _op_slots=$(printf '%s\n%s\n' "$_op_slots" "$_bog_extra" | sort -u | grep -v '^$')
    fi

    [ -z "$_op_slots" ] && continue

    while IFS= read -r slot; do
        [ -z "$slot" ] && continue
        if ! printf '%s\n' "$_matrix_slots" | grep -qx "$slot"; then
            printf 'FAIL  lint-write-path  %s  slot %s: no matrix row\n' \
                "$_scan_name" "$slot"
            FAILED=1
        fi
    done <<EOF
$_op_slots
EOF
done

# ---------------------------------------------------------------------------
# Audit-call pass (WARN, advisory; --strict promotes). File scope: a file that declares a servlet
# write handler must contain an audit-recording call. [ev: retro servlet-write-audit Δ2]
# ---------------------------------------------------------------------------
_idx=0
for _scan_src in "${_SCAN_SRCS[@]}"; do
    _idx=$(( _idx + 1 ))
    while IFS= read -r _jf; do
        [ -n "$_jf" ] || continue
        _arow=$(awk -v FILE="$_jf" '
            BEGIN { in_bc = 0; hl = 0; audited = 0; DQ = "\""; SQ = "\047"; BS = "\\" }
            {
                # Strip // and /* */ comments outside string/char literals and blank literal
                # contents ("x" -> ""), so a "*/*" or "http://" literal opens no comment and a
                # literal that names an audit call is not one. Sibling string-aware strippers:
                # lint-delays.sh strip_comment (awk), lint-arbitrary-ord.sh code_part (bash).
                # [polish-2026-10-02 P2b, #199 WU6b]
                ln = $0; out = ""; j = 1; L = length(ln); q = ""
                while (j <= L) {
                    c = substr(ln, j, 1)
                    if (in_bc) { if (substr(ln, j, 2) == "*/") { in_bc = 0; j += 2 } else j++ }
                    else if (q != "") {
                        if (c == BS) j += 2
                        else { if (c == q) { out = out c; q = "" }; j++ }
                    }
                    else if (c == DQ || c == SQ) { q = c; out = out c; j++ }
                    else if (substr(ln, j, 2) == "//") break
                    else if (substr(ln, j, 2) == "/*") { in_bc = 1; j += 2 }
                    else { out = out c; j++ }
                }
                if (!hl && match(out, /void[[:space:]]+do(Post|Put|Delete|Patch)[[:space:]]*\(/)) {
                    hl = FNR; hm = substr(out, RSTART, RLENGTH); sub(/^void[[:space:]]+/, "", hm); sub(/[[:space:]]*\($/, "", hm)
                }
                t = out
                while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
                    id = substr(t, RSTART, RLENGTH); sub(/[[:space:]]*\($/, "", id)
                    t = substr(t, RSTART + RLENGTH)
                    if (tolower(id) ~ /audit/ && id !~ /^(get|is)[A-Z]/) audited = 1
                }
            }
            END {
                if (hl && !audited)
                    printf "WARN  lint-write-path  %s:%d  write handler %s: no audit call (record user, ord, old, new, source before answering success)\n", FILE, hl, hm
            }
        ' "$_jf")
        if [ -n "$_arow" ]; then
            printf '%s\n' "$_arow"
            AUDIT=1
        fi
    done < "$_TMP/src.$_idx"
done

# ---------------------------------------------------------------------------
# Per-row STALE pass.  For each data row in the matrix, extract the
# backtick-inner slot name (^[a-z][A-Za-z0-9]*$ only; prose/multi-word
# cells are not slots).  Skip rows carrying the literal [concept] token
# (checked on the comment-stripped row; <!-- --> stripped before matching).
# A [concept] marker exempts only its own row — never another row with the
# same name (per-row, not per-name).  The covered set is the matrix-root-wide
# @NiagaraProperty|Action harvest ∪ --bog extras (NOT the OPERATOR-only
# per-module scanner).  STALE rows are advisory (exit 0) unless --strict.
# Row grammar: STATUS-first, same shape as the FAIL row at :374.
# ---------------------------------------------------------------------------
_covered_flat=" $(printf '%s\n%s\n' "$_covered_names" "$_bog_extra" | tr '\n' ' ') "
_ln=0
while IFS= read -r _mline; do
    _ln=$(( _ln + 1 ))
    # Only data rows (start with |)
    case "$_mline" in '|'*) : ;; *) continue ;; esac
    # Strip markdown comments <!-- … --> from the row before [concept] check.
    _row=$(printf '%s' "$_mline" | sed 's/<!--[^>]*-->//g')
    # Capture the [concept] marker (do not skip yet — fall through to name filter).
    _is_concept=0
    case "$_row" in *'[concept]'*) _is_concept=1 ;; esac
    # Extract backtick-inner content of the first cell.
    _cell=$(printf '%s' "$_row" | awk -F'|' 'NR==1{cell=$2; gsub(/^[[:space:]]+|[[:space:]]+$/,"",cell); print cell}')
    case "$_cell" in '`'*) : ;; *) continue ;; esac
    # shellcheck disable=SC2016
    _name=$(printf '%s' "$_cell" | sed -E 's/^`([^`]+)`.*/\1/')
    # Must match slot name pattern: ^[a-z][A-Za-z0-9]*$
    printf '%s' "$_name" | grep -qE '^[a-z][A-Za-z0-9]*$' || continue
    # DRIFT: [concept]-marked row whose slot IS in the covered set -> marker is stale.
    # True concept: [concept]-marked row whose slot is NOT covered -> stay silent.
    case "$_covered_flat" in *" $_name "*)
        if [ "$_is_concept" -eq 1 ]; then
            printf 'DRIFT  lint-write-path  %s:%d  slot %s: concept marker but a source slot exists\n' \
                "$MATRIX" "$_ln" "$_name"
            DRIFT=1
        fi
        continue ;;
    esac
    [ "$_is_concept" -eq 1 ] && continue    # true concept row: name absent from source -> silent
    # Emit STALE advisory (STATUS-first, same column order as FAIL row).
    printf 'STALE  lint-write-path  %s:%d  slot %s: no source slot with that name\n' \
        "$MATRIX" "$_ln" "$_name"
    STALE=1
done < "$MATRIX"

# ---------------------------------------------------------------------------
# Exit: uncovered FAIL always exits 1 (unchanged, with and without --strict).
# --strict promotes STALE, DRIFT or an audit-call WARN to exit 1.  Otherwise exit 0.
# Exit 3 (usage/env/missing-matrix) is handled above — range {0,1}∪{3} (K20).
# ---------------------------------------------------------------------------
[ "$FAILED" -eq 1 ] && exit 1
if [ "$STRICT" -eq 1 ]; then
    if [ "$STALE" -eq 1 ] || [ "$DRIFT" -eq 1 ] || [ "$AUDIT" -eq 1 ]; then
        exit 1
    fi
fi
exit 0
