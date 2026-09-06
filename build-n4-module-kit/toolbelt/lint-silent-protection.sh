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
# Mutation: S23-pos -- removes Pattern-B alarm-adapter exemption, causing false-WARN on alarmed trips
# Mutation: S23-and -- removes the BIAlarmSource check, allowing bare newOffnormalAlarm to suppress WARN
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
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
# Parse @NiagaraProperty annotations (single-line and multi-line).
# Emits: <name>|<flags_string>  one per property.
# Fix: use prop_first flag so the opening line is NOT also appended by the
# accumulation rule (which was doubling the first line and breaking multi-line
# depth tracking for annotations whose ( and ) are on separate lines).
BEGIN { in_prop = 0; prop_buf = ""; prop_name = ""; prop_first = 0 }
FNR == 1 { in_prop = 0; prop_buf = ""; prop_name = ""; prop_first = 0 }
!in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = $0; prop_name = ""; prop_first = 1
}
in_prop && !prop_first { prop_buf = prop_buf " " $0 }
in_prop {
    prop_first = 0
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
        in_prop = 0; prop_buf = ""; prop_name = ""; prop_first = 0
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

# Pass 0b: build dir-wide ALARM_CLASSES index (D3b).
# Pattern A (existing): file contains BAlarmSourceExt OR BAlarmRecord.
# Pattern B (NEW): file contains "implements" + BIAlarmSource AND (newOffnormalAlarm OR new AlarmSupport().
# Comments are stripped (// only; /* */ block stripping kept in the per-file awk below) before token match.
# Output: one class name per line → joined into ALARM_CLASSES space-separated string.
while IFS= read -r _p0b_f; do
    awk '
    BEGIN { cn=""; ha=0; bi=0; bn=0; bc=0; in_bc=0 }
    {
        s = $0
        # strip /* */ block comments (state carries across lines)
        out = ""; j = 1; L = length(s)
        while (j <= L) {
            if (in_bc) {
                if (substr(s,j,2) == "*/") { in_bc = 0; j += 2 }
                else j++
            } else {
                t2 = substr(s,j,2)
                if (t2 == "//") { break }
                else if (t2 == "/*") { in_bc = 1; j += 2 }
                else { out = out substr(s,j,1); j++ }
            }
        }
        s = out
        if (cn == "" && match(s, /class[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
            t = substr(s, RSTART); sub(/class[[:space:]]+/, "", t)
            match(t, /^[A-Za-z_][A-Za-z0-9_]*/); cn = substr(t, 1, RLENGTH)
        }
        if (!ha && (index(s,"BAlarmSourceExt")>0 || index(s,"BAlarmRecord")>0)) ha=1
        if (!bi && index(s,"implements")>0 && index(s,"BIAlarmSource")>0) bi=1
        if (!bn && index(s,"newOffnormalAlarm")>0) bn=1
        if (!bc && index(s,"new AlarmSupport(")>0) bc=1
    }
    END { if (cn!="" && (ha || (bi && (bn||bc)))) print cn }
    ' "$_p0b_f" 2>/dev/null
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort) | sort -u > "$_TMP/alarm_classes.txt"
ALARM_CLASSES=$(tr '\n' ' ' < "$_TMP/alarm_classes.txt")

# ---------------------------------------------------------------------------
# Main awk: per-file trip detection and surface resolution.
# Write the shared method-boundary fragment; consumed via a second -f below.
# ---------------------------------------------------------------------------
printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"
cat > "$_TMP/main.awk" << 'AWKEOF'
# Called with: -v FILE=path -v SURF_SLOTS="..." -v EFFECT_SLOTS="..."
#              -v SURF_WRITE_FIELDS="..." -v ALARM_CLASSES="..."
{ lines[NR] = $0 }

function in_list(word, lst,    padded) {
    padded = " " lst " "
    return index(padded, " " word " ") > 0
}
function cap1(s) { return toupper(substr(s,1,1)) substr(s,2) }
function getter(s) { return "get" cap1(s) }
function setter(s) { return "set" cap1(s) }

# Extract the single-identifier condition from an if(...) on this line.
# Returns the identifier, or "" when the condition is compound / absent.
# Used for criterion (ii-B): if the condition field is itself surfaced via
# SURF_WRITE_FIELDS (cross-file follow) the trip is considered CLEAN.
function extract_cond_field(stripped,    rest, paren_depth, ci, ch, cond) {
    if (!match(stripped, /(^|[^A-Za-z0-9_])if[[:space:]]*\(/)) return ""
    # advance to the opening "("
    rest = substr(stripped, RSTART + RLENGTH - 1)
    rest = substr(rest, 2)          # skip "("
    paren_depth = 1; cond = ""
    for (ci = 1; ci <= length(rest); ci++) {
        ch = substr(rest, ci, 1)
        if      (ch == "(") { paren_depth++; cond = cond ch }
        else if (ch == ")") { paren_depth--; if (paren_depth == 0) break; cond = cond ch }
        else                 { cond = cond ch }
    }
    # strip leading ! and whitespace (handles "!fieldName" negations)
    gsub(/^[[:space:]!]*/, "", cond)
    gsub(/[[:space:]]*$/, "", cond)
    # accept only a pure identifier: no operators, spaces, dots, parens
    if (!match(cond, /^[A-Za-z_][A-Za-z0-9_]*$/)) return ""
    return cond
}

END {
    # --- D6: strip // and /* */ comments from lines[] (blank, not delete; line numbers preserved) ---
    in_bc = 0
    for (i = 1; i <= NR; i++) {
        ln = lines[i]; out = ""; j = 1; L = length(ln)
        while (j <= L) {
            if (in_bc) {
                if (substr(ln,j,2) == "*/") { in_bc = 0; j += 2 }
                else j++
            } else {
                t2 = substr(ln,j,2)
                if (t2 == "//") { break }
                else if (t2 == "/*") { in_bc = 1; j += 2 }
                else { out = out substr(ln,j,1); j++ }
            }
        }
        lines[i] = out
    }

    # --- A: collect private boolean fields and their declaration lines ---
    for (i = 1; i <= NR; i++) {
        ln = lines[i]
        if (match(ln, /private[[:space:]]+(final[[:space:]]+)?boolean[[:space:]]+/)) {
            rest = substr(ln, RSTART + RLENGTH)
            match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)
            fname = substr(rest, 1, RLENGTH)
            if (fname != "") {
                private_fields[fname] = 1
                private_field_line[fname] = i   # declaration line (used by Pattern 7)
            }
        }
    }

    # --- B: determine this_class from FILE for ALARM_CLASSES lookup (D3c) ---
    # Java class name is the filename without the directory path and .java extension.
    this_class = FILE; sub(/.*\//, "", this_class); sub(/\.java$/, "", this_class)

    # --- C: collect SURF_WRITE lines in this file ---
    n_ss = split(SURF_SLOTS, ss_arr, " ")
    for (i = 1; i <= NR; i++) {
        ln = lines[i]
        for (k = 1; k <= n_ss; k++) {
            sname = ss_arr[k]; if (sname == "") continue
            g = getter(sname); s = setter(sname)
            if (index(ln, g "().setValue") > 0 || index(ln, s "(") > 0) {
                surf_write_line[i] = 1
                # extract argument for same-file follow
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

    # --- D: parse method boundaries (shared fragment, PEAK depth) ---
    # PEAK depth: open ⟺ max_d > old_d && max_d >= 2; close ⟺ brace_depth < m_dep,
    # tested in the SAME iteration. A one-liner opens AND closes on one line;
    # the class-FIELD guard (>= 2) prevents the class body from ever being entered.
    # [ev: design D1g; lib/method-boundary.sh mb_parse]
    n_meth = mb_parse(lines, NR, meth_start, meth_end, meth_name)

    # --- E: for each method, find TRIPS guarded by if() ---
    # Patterns detected (B824 §824.2):
    #   P1: Math.min(target, ...) inside if()
    #   P3: getEffectSlot().setValue(false) inside if()  [effect-slot exemption: NOT a surface]
    #   P4: this.<privateBoolField> = true/false inside if()
    #   P7: return X || <privateField> in *Inhibited/*Trip method  (no if guard needed)
    # Patterns 2 (target=0) and 6 (return in if block) are intentionally absent:
    #   P2 fired on demand gates and bounds clamps (false positives)
    #   P6 fired on every early-guard return (113 false positives in real trees)
    n_trips = 0

    for (mi = 0; mi < n_meth; mi++) {
        ms = meth_start[mi]; me = meth_end[mi]; mn = meth_name[mi]

        for (i = ms; i <= me; i++) {
            ln = lines[i]
            stripped = ln; sub(/\/\/.*$/, "", stripped)

            # Pattern 7: *Inhibited/*Trip method — return lhs || <private field>
            # Report at the FIELD DECLARATION line (private_field_line[rhs]), not the return line.
            if ((mn ~ /[Ii]nhibited$/ || mn ~ /[Tt]rip$/) &&
                match(stripped, /(^|[[:space:]])return[[:space:]]+[A-Za-z_][A-Za-z0-9_. ]*\|\|[[:space:]]*[A-Za-z_][A-Za-z0-9_]*/)) {
                seg = substr(stripped, RSTART)
                sub(/(^|[[:space:]])return[[:space:]]+/, "", seg)
                sub(/.*\|\|[[:space:]]*/, "", seg)
                match(seg, /^[A-Za-z_][A-Za-z0-9_]*/)
                rhs = substr(seg, 1, RLENGTH)
                if (rhs in private_fields) {
                    n_trips++
                    trip_line[n_trips] = private_field_line[rhs]   # field declaration line
                    trip_meth[n_trips] = mn
                    trip_ms[n_trips] = ms; trip_me[n_trips] = me; trip_out[n_trips] = rhs
                    trip_cond_field[n_trips] = ""   # P7 has no if-condition
                }
                continue
            }

            # --- check if this line has any trip action ---
            has_trip = 0; tout = ""; p4_fname = ""

            # Pattern 1: Math.min(target, ...)
            if (!has_trip && (index(stripped, "Math.min(target,") > 0 || index(stripped, "Math.min(target ,") > 0)) {
                has_trip = 1; tout = "stage"
            }
            # Pattern 3: getEffectSlot().setValue(false) with an inline if(condition) on the same line.
            # Restricting to same-line if avoids false positives where setValue(false) is inside a
            # multi-line block opened by an if() on a different line (e.g. power-on stagger guards,
            # state-reset sequences). The condition must also be a single-identifier boolean field
            # (extract_cond_field returns non-empty).
            if (!has_trip) {
                n_es = split(EFFECT_SLOTS, es_arr, " ")
                for (k = 1; k <= n_es; k++) {
                    ename = es_arr[k]; if (ename == "") continue
                    eg = getter(ename)
                    if (index(stripped, eg "().setValue(false)") > 0 ||
                        index(stripped, eg "().setValue( false )") > 0) {
                        p3_cond = extract_cond_field(stripped)
                        if (p3_cond != "") { has_trip = 1; tout = ename; break }
                    }
                }
            }
            # Pattern 4: this.<privateBoolField> = true/false
            if (!has_trip && match(stripped, /this\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*(true|false)/)) {
                seg = substr(stripped, RSTART)
                sub(/this\./, "", seg)
                match(seg, /^[A-Za-z_][A-Za-z0-9_]*/)
                p4_fname = substr(seg, 1, RLENGTH)
                if (p4_fname in private_fields) {
                    has_trip = 1; tout = p4_fname
                }
            }

            if (!has_trip) continue

            # Verify guarded by an if() — scan backward up to 10 lines within the method
            guarded = 0
            for (j = i; j >= ms && j >= i - 10; j--) {
                ln2 = lines[j]; sub(/\/\/.*$/, "", ln2)
                if (match(ln2, /(^|[^A-Za-z0-9_])if[[:space:]]*\(/)) { guarded = 1; break }
            }
            if (!guarded) continue

            # Extract condition field for criterion (ii-B): if the if(...) condition on
            # this line is a single named boolean field that appears in SURF_WRITE_FIELDS
            # (written to a surface slot somewhere in the module), the trip is CLEAN.
            cond_field = extract_cond_field(stripped)

            n_trips++
            trip_line[n_trips] = i; trip_meth[n_trips] = mn
            trip_ms[n_trips] = ms; trip_me[n_trips] = me; trip_out[n_trips] = tout
            trip_cond_field[n_trips] = cond_field
        }
    }

    # --- F: for each trip, check surfaces ---
    n_swf = split(SURF_WRITE_FIELDS, swf_arr, " ")

    for (t = 1; t <= n_trips; t++) {
        tl = trip_line[t]; tm = trip_meth[t]
        tms = trip_ms[t]; tme = trip_me[t]
        surfaced = 0

        # (1) this class or its B<Pure> adapter is in ALARM_CLASSES (Pattern A or B, D3c)
        # Pattern A: class itself in ALARM_CLASSES (BAlarmSourceExt/BAlarmRecord in this file)
        # Pattern B adapter->pure follow: B+this_class in ALARM_CLASSES (BIAlarmSource + alarm method)
        if (in_list(this_class, ALARM_CLASSES) || in_list("B" this_class, ALARM_CLASSES)) {
            surfaced = 1
        }

        # (ii-B) condition field is surfaced via cross-file field->slot follow
        if (!surfaced && trip_cond_field[t] != "" && in_list(trip_cond_field[t], SURF_WRITE_FIELDS)) {
            surfaced = 1
        }

        # (2) same method contains a SURF_WRITE
        if (!surfaced) {
            for (i = tms; i <= tme; i++) {
                if (i in surf_write_line) { surfaced = 1; break }
            }
        }

        # (3) cross-file field->slot follow:
        #     a non-private field is assigned near the trip line AND appears in SURF_WRITE_FIELDS
        if (!surfaced && n_swf > 0) {
            for (j = tl; j <= tl + 8 && j <= tme; j++) {
                ln3 = lines[j]; sub(/\/\/.*$/, "", ln3)
                for (k = 1; k <= n_swf; k++) {
                    fld = swf_arr[k]; if (fld == "") continue
                    if (fld in private_fields) continue
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
        -v ALARM_CLASSES="$ALARM_CLASSES" \
        -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" "$f" 2>/dev/null >> "$_WARN_FILE"
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
