#!/usr/bin/env bash
# lint-silent-protection.sh — Silent-protection-trip lint (Campaign 9 S18, B824).
#
# Flags a PROTECTION TRIP that forces an output OFF / sheds a stage with NO
# operator surface: no SUMMARY/OPERATOR status-or-reason slot written on its
# path, no BAlarmSourceExt, no readable reason field surfaced.
#
# Usage:  lint-silent-protection.sh [--strict] <java-src-dir>
#
# Row:  WARN  lint-silent-protection  <file>:<line>  <method> forces <output>/sheds stage on condition
#       -- no status/reason/alarm surface in scope; add a *Alarm/*Reason SUMMARY slot or a BAlarmSourceExt
#
# Surface-name allowlist (SUMMARY/OPERATOR slots whose names signal a surface):
#   *Alarm *Fault *Skip* *Reason *Status *Mismatch *Stuck *Available *Fallback
# Effect-slot exemption: writing the trip own forced output slot is NOT a surface.
# Private-field exemption: a private boolean field is NEVER a surface.
# Exactly ONE WARN per trip site (dedupe on <file>:<line>).
#
# Exits: 0 no WARN (or WARN without --strict) | 1 any WARN under --strict | 3 usage/env
# Dot-dirs excluded (D9b). VCS-free by design.
# [ev: corpus B824]  [ev: retro campaign9-silent-protection]
set -u
LC_ALL=C
export LC_ALL

FAILED=0
STRICT=0
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'usage: lint-silent-protection.sh [--strict] <java-src-dir>\n' >&2; exit 3 ;;
    *) break ;;
  esac
done

[ $# -ge 1 ] || { printf 'usage: lint-silent-protection.sh [--strict] <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-silent-protection: not a directory: %s\n' "$SRC" >&2; exit 3; }

# SP9 guard: exit 3 when the source tree contains no Java files (K20 / C8 silent-0 lesson).
# A directory with no .java files cannot have any trips — a silent exit 0 hides a misconfigured
# invocation (wrong path, empty scaffold). Emit an ERROR row and exit 3.
_java_count=$(find "$SRC" -not \( -name '.*' -prune \) -name '*.java' 2>/dev/null | wc -l)
if [ "$_java_count" -eq 0 ]; then
  printf 'ERROR  lint-silent-protection  %s  no Java sources found (K20: wrong path or empty scaffold)\n' "$SRC"
  exit 3
fi

# ---------------------------------------------------------------------------
# Write awk programs to temp files to avoid single-quote issues inside
# awk programs embedded in shell strings.
# ---------------------------------------------------------------------------

# Pass-0 awk: collect @NiagaraProperty declarations from all files.
# Outputs: <name>|<flags_string>  one per property.
cat > "$_TMP/pass0.awk" << 'AWKEOF'
BEGIN { in_prop = 0; prop_buf = ""; prop_name = "" }
FNR == 1 { in_prop = 0; prop_buf = ""; prop_name = "" }
!in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = $0; prop_name = ""
}
in_prop && FNR > 1 { prop_buf = prop_buf " " $0 }
in_prop {
    if (prop_name == "" && match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
        seg = substr(prop_buf, RSTART)
        sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg)
        prop_name = seg
    }
    depth = 0; tmp = prop_buf
    for (ci = 1; ci <= length(tmp); ci++) {
        c = substr(tmp, ci, 1)
        if (c == "(") depth++
        else if (c == ")") depth--
    }
    if (depth <= 0 && index(prop_buf, "(") > 0) {
        prop_flags = ""
        if (match(prop_buf, /flags[[:space:]]*=[[:space:]]*/)) {
            seg2 = substr(prop_buf, RSTART + RLENGTH)
            match(seg2, /^[^,)]*/); prop_flags = substr(seg2, 1, RLENGTH)
        }
        if (prop_name != "") print prop_name "|" prop_flags
        in_prop = 0; prop_buf = ""; prop_name = ""
    }
}
AWKEOF

# Run pass 0 over all Java files (dot-dirs excluded)
find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort | \
    xargs -r awk -f "$_TMP/pass0.awk" 2>/dev/null > "$_TMP/all_props.txt"

# Classify properties into SURF_SLOTS (allowlisted + SUMMARY/OPERATOR) and EFFECT_SLOTS.
: > "$_TMP/surf_slots.txt"
: > "$_TMP/effect_slots.txt"
while IFS='|' read -r pname pflags; do
    [ -z "$pname" ] && continue
    # Check allowlist (case-insensitive glob on the name)
    n=$(printf '%s' "$pname" | tr '[:upper:]' '[:lower:]')
    allowlisted=0
    case "$n" in
        *alarm|*fault|*skip*|*reason|*status|*mismatch|*stuck|*available|*fallback) allowlisted=1 ;;
    esac
    has_surf=0
    case "$pflags" in *SUMMARY*|*OPERATOR*) has_surf=1 ;; esac
    if [ "$allowlisted" -eq 1 ] && [ "$has_surf" -eq 1 ]; then
        printf '%s\n' "$pname" >> "$_TMP/surf_slots.txt"
    else
        printf '%s\n' "$pname" >> "$_TMP/effect_slots.txt"
    fi
done < "$_TMP/all_props.txt"

SURF_SLOTS=$(sort -u "$_TMP/surf_slots.txt" | tr '\n' ' ')
EFFECT_SLOTS=$(sort -u "$_TMP/effect_slots.txt" | tr '\n' ' ')

# Pass-1 awk: collect field names referenced as arguments to SURF_WRITE calls
# across all files (for the cross-file field->slot follow).
# Outputs one field-name per line.
cat > "$_TMP/pass1.awk" << 'AWKEOF'
BEGIN {
    n_ss = split(SURF_SLOTS, ss_arr, " ")
}
{
    for (k = 1; k <= n_ss; k++) {
        sname = ss_arr[k]; if (sname == "") continue
        getter = "get" toupper(substr(sname,1,1)) substr(sname,2)
        # getX().setValue(arg)
        pat = getter "().setValue("
        pos = index($0, pat)
        if (pos > 0) {
            arg = substr($0, pos + length(pat))
            sub(/\).*/, "", arg); gsub(/[[:space:]]/, "", arg)
            if (match(arg, /\.[A-Za-z_][A-Za-z0-9_]*$/))
                print substr(arg, RSTART+1)
            else if (match(arg, /^[A-Za-z_][A-Za-z0-9_]*$/))
                print arg
        }
        setter = "set" toupper(substr(sname,1,1)) substr(sname,2)
        pat2 = setter "("
        pos2 = index($0, pat2)
        if (pos2 > 0 && pos == 0) {
            arg2 = substr($0, pos2 + length(pat2))
            sub(/[,)].*/, "", arg2); gsub(/[[:space:]]/, "", arg2)
            if (match(arg2, /^[A-Za-z_][A-Za-z0-9_]*$/))
                print arg2
        }
    }
}
AWKEOF

find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort | \
    xargs -r awk -v SURF_SLOTS="$SURF_SLOTS" -f "$_TMP/pass1.awk" 2>/dev/null | \
    sort -u > "$_TMP/surf_write_fields.txt"
SURF_WRITE_FIELDS=$(tr '\n' ' ' < "$_TMP/surf_write_fields.txt")

# ---------------------------------------------------------------------------
# Main awk: per-file trip detection and surface resolution.
# ---------------------------------------------------------------------------
cat > "$_TMP/main.awk" << 'AWKEOF'
# Called with: -v FILE=path -v SURF_SLOTS="..." -v EFFECT_SLOTS="..."
#              -v SURF_WRITE_FIELDS="..."
{ lines[NR] = $0 }

function in_list(word, lst,    padded) {
    padded = " " lst " "
    return index(padded, " " word " ") > 0
}
function cap1(s) { return toupper(substr(s,1,1)) substr(s,2) }
function getter(s) { return "get" cap1(s) }
function setter(s) { return "set" cap1(s) }

END {
    # --- A: collect private boolean fields ---
    for (i = 1; i <= NR; i++) {
        ln = lines[i]
        if (match(ln, /private[[:space:]]+(final[[:space:]]+)?boolean[[:space:]]+/)) {
            rest = substr(ln, RSTART + RLENGTH)
            match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)
            fname = substr(rest, 1, RLENGTH)
            if (fname != "") private_fields[fname] = 1
        }
    }

    # --- B: file-level alarm ext check ---
    file_has_alarm = 0
    for (i = 1; i <= NR; i++) {
        if (index(lines[i], "BAlarmSourceExt") > 0 || index(lines[i], "BAlarmRecord") > 0) {
            file_has_alarm = 1; break
        }
    }

    # --- C: collect SURF_WRITE lines in this file ---
    n_ss = split(SURF_SLOTS, ss_arr, " ")
    for (i = 1; i <= NR; i++) {
        ln = lines[i]
        for (k = 1; k <= n_ss; k++) {
            sname = ss_arr[k]; if (sname == "") continue
            g = getter(sname); s = setter(sname)
            if (index(ln, g "().setValue") > 0 || index(ln, s "(") > 0) {
                surf_write_line[i] = 1
                # extract argument for cross-file follow (already covered by pass1,
                # but also capture here for same-file SP2 cases)
                pos = index(ln, g "().setValue(")
                if (pos > 0) {
                    arg = substr(ln, pos + length(g "().setValue("))
                    sub(/\).*/, "", arg); gsub(/[[:space:]]/, "", arg)
                    if (match(arg, /\.[A-Za-z_][A-Za-z0-9_]*$/))
                        surf_write_arg[i] = substr(arg, RSTART+1)
                    else if (match(arg, /^[A-Za-z_][A-Za-z0-9_]*$/))
                        surf_write_arg[i] = arg
                }
            }
        }
    }

    # --- D: parse method boundaries and find trips ---
    n_meth = 0
    brace_depth = 0
    in_m = 0; m_name = ""; m_start = 0; m_depth_at_open = 0

    for (i = 1; i <= NR; i++) {
        ln = lines[i]
        stripped = ln; sub(/\/\/.*$/, "", stripped)

        # count braces
        for (ci = 1; ci <= length(stripped); ci++) {
            c = substr(stripped, ci, 1)
            if (c == "{") brace_depth++
            else if (c == "}") brace_depth--
        }

        if (!in_m && brace_depth > 0) {
            # detect method open: identifier( ... ) [throws ...] { on this line
            if (match(stripped, /[A-Za-z_][A-Za-z0-9_<>[\]]*[[:space:]]*\([^)]*\)[[:space:]]*(throws[^{]*)?\{/)) {
                seg = substr(stripped, RSTART)
                match(seg, /^[A-Za-z_][A-Za-z0-9_<>[\]]*/); mname = substr(seg, 1, RLENGTH)
                if (mname !~ /^(if|for|while|switch|catch|try|else|do|new)$/) {
                    in_m = 1; m_name = mname; m_start = i
                    m_depth_at_open = brace_depth
                }
            }
        }

        if (in_m) {
            if (brace_depth < m_depth_at_open) {
                # method closed
                meth_start[n_meth] = m_start
                meth_end[n_meth]   = i
                meth_name[n_meth]  = m_name
                n_meth++
                in_m = 0; m_name = ""; m_start = 0
            }
        }
    }

    # --- E: for each method, find TRIPS guarded by if() ---
    # Strategy: for each line with a trip action, scan backward within the method
    # to find a guarding if( within 10 lines. Handles both single-line if and
    # multi-line if blocks without brace-depth complexity.
    n_trips = 0

    # Helper: given line i in method [ms..me], return 1 if guarded by an if()
    # by scanning backward up to 10 lines within the method.
    # We store the result in has_if_guard[i].
    for (mi = 0; mi < n_meth; mi++) {
        ms = meth_start[mi]; me = meth_end[mi]; mn = meth_name[mi]

        for (i = ms; i <= me; i++) {
            ln = lines[i]
            stripped = ln; sub(/\/\/.*$/, "", stripped)

            # Pattern 7: *Inhibited/*Trip method with return X || Y (no if guard required)
            if ((mn ~ /[Ii]nhibited$/ || mn ~ /[Tt]rip$/) &&
                match(stripped, /(^|[[:space:]])return[[:space:]]+[A-Za-z_][A-Za-z0-9_. ]*\|\|[[:space:]]*[A-Za-z_][A-Za-z0-9_]*/)) {
                seg = substr(stripped, RSTART)
                sub(/(^|[[:space:]])return[[:space:]]+/, "", seg)
                sub(/.*\|\|[[:space:]]*/, "", seg)
                match(seg, /^[A-Za-z_][A-Za-z0-9_]*/)
                rhs = substr(seg, 1, RLENGTH)
                if (rhs in private_fields) {
                    n_trips++
                    trip_line[n_trips] = i; trip_meth[n_trips] = mn
                    trip_ms[n_trips] = ms; trip_me[n_trips] = me; trip_out[n_trips] = rhs
                }
                continue
            }

            # --- check if this line has any trip action ---
            has_trip = 0; tout = ""

            # Pattern 1: Math.min(target, ...)
            if (!has_trip && (index(stripped, "Math.min(target,") > 0 || index(stripped, "Math.min(target ,") > 0)) {
                has_trip = 1; tout = "stage"
            }
            # Pattern 2: target = 0 / target-- / target = target - 1
            if (!has_trip && (match(stripped, /target[[:space:]]*=[[:space:]]*0[^0-9]/) ||
                match(stripped, /target--/) ||
                match(stripped, /target[[:space:]]*=[[:space:]]*target[[:space:]]*-[[:space:]]*1/))) {
                has_trip = 1; tout = "stage"
            }
            # Pattern 3: getEffectSlot().setValue(false)
            if (!has_trip) {
                n_es = split(EFFECT_SLOTS, es_arr, " ")
                for (k = 1; k <= n_es; k++) {
                    ename = es_arr[k]; if (ename == "") continue
                    eg = getter(ename)
                    if (index(stripped, eg "().setValue(false)") > 0 ||
                        index(stripped, eg "().setValue( false )") > 0) {
                        has_trip = 1; tout = ename; break
                    }
                }
            }
            # Pattern 6: return inside a brace-delimited if-block.
            # Only fire when the return line itself contains a { before the return
            # (i.e. the if-block opens on the same line) OR the if( is on the same line.
            # This avoids false positives on post-if plain returns like `return target;`.
            if (!has_trip && match(stripped, /(^|[[:space:]])return([[:space:];{]|$)/)) {
                ret_pos = RSTART
                has_brace_before = (index(substr(stripped, 1, ret_pos), "{") > 0)
                has_if_same = match(stripped, /(^|[^A-Za-z0-9_])if[[:space:]]*\(/)
                if (has_brace_before || has_if_same) {
                    has_trip = 1; tout = "skip"
                }
            }

            if (!has_trip) continue

            # Verify guarded by an if() — scan backward up to 10 lines
            guarded = 0
            for (j = i; j >= ms && j >= i - 10; j--) {
                ln2 = lines[j]; sub(/\/\/.*$/, "", ln2)
                if (match(ln2, /(^|[^A-Za-z0-9_])if[[:space:]]*\(/)) { guarded = 1; break }
            }
            if (!guarded) continue

            n_trips++
            trip_line[n_trips] = i; trip_meth[n_trips] = mn
            trip_ms[n_trips] = ms; trip_me[n_trips] = me; trip_out[n_trips] = tout
        }
    }

    # --- F: for each trip, check surfaces ---
    n_swf = split(SURF_WRITE_FIELDS, swf_arr, " ")

    for (t = 1; t <= n_trips; t++) {
        tl = trip_line[t]; tm = trip_meth[t]
        tms = trip_ms[t]; tme = trip_me[t]
        surfaced = 0

        # (1) file has alarm ext
        if (file_has_alarm) { surfaced = 1 }

        # (2) same method has a SURF_WRITE
        if (!surfaced) {
            for (i = tms; i <= tme; i++) {
                if (i in surf_write_line) { surfaced = 1; break }
            }
        }

        # (3) cross-file field->slot follow:
        #     a non-private field is assigned in the guarded block (near tl),
        #     AND that field name appears in SURF_WRITE_FIELDS (from any file)
        if (!surfaced && n_swf > 0) {
            for (j = tl; j <= tl + 8 && j <= tme; j++) {
                ln3 = lines[j]; sub(/\/\/.*$/, "", ln3)
                for (k = 1; k <= n_swf; k++) {
                    fld = swf_arr[k]; if (fld == "") continue
                    if (fld in private_fields) continue
                    # check if fld is assigned (= but not ==) anywhere in this window
                    pos = index(ln3, fld)
                    if (pos > 0) {
                        rest = substr(ln3, pos + length(fld))
                        gsub(/^[[:space:]]*/, "", rest)
                        c1 = substr(rest, 1, 1); c2 = substr(rest, 2, 1)
                        if (c1 == "=" && c2 != "=") { surfaced = 1; break }
                    }
                }
                if (surfaced) break
            }
        }

        if (!surfaced) {
            printf "WARN  lint-silent-protection  %s:%d  %s forces %s/sheds stage on condition" \
                " -- no status/reason/alarm surface in scope;" \
                " add a *Alarm/*Reason SUMMARY slot or a BAlarmSourceExt\n",
                FILE, tl, tm, trip_out[t]
        }
    }
}
AWKEOF

# ---------------------------------------------------------------------------
# Run the main awk over each Java file; collect results then dedupe.
# ---------------------------------------------------------------------------
_WARN_FILE="$_TMP/warns.txt"
: > "$_WARN_FILE"

while IFS= read -r f; do
    awk -v FILE="$f" \
        -v SURF_SLOTS="$SURF_SLOTS" \
        -v EFFECT_SLOTS="$EFFECT_SLOTS" \
        -v SURF_WRITE_FIELDS="$SURF_WRITE_FIELDS" \
        -f "$_TMP/main.awk" "$f" 2>/dev/null >> "$_WARN_FILE"
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# Dedupe by <file>:<line> (first WARN per site wins)
if [ -s "$_WARN_FILE" ]; then
    awk '
    {
        n = split($0, a, /[[:space:]]{2,}/)
        key = (n >= 3) ? a[3] : $0
        if (!(key in seen)) { seen[key] = 1; print }
    }
    ' "$_WARN_FILE"
    FAILED=1
fi

[ "$STRICT" -eq 1 ] && [ "$FAILED" -eq 1 ] && exit 1
exit 0
