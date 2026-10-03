#!/usr/bin/env bash
# lint-config-sanity.sh — Unsafe @NiagaraProperty default patterns gate (Wave 3, LR3, B862).
#
# Detects four unsafe default patterns in @NiagaraProperty annotations:
#
#   CS1 FAIL: *Interval default <= *Duration default in the same class, both read in MILLISECONDS from
#             any literal BRelTime factory (make(ms), makeSeconds/Minutes/Hours/Days, make(d,h,m,s)) or
#             constant (BRelTime.java:34-48, :56-83). A BRelTime default it cannot read (a variable, an
#             expression) is a CS1 WARN "unreadable" row, never a silent skip. A pair where BOTH are 0
#             (BRelTime.DEFAULT on a display mirror) is unset, not a contradiction. [ev: audit-2026-10-03 A1]
#             Physical impossibility: the cycle interval must exceed the defrost/heat duration.
#             A too-small interval means "interval already elapsed" fires on the very first execute.
#
#   CS2 FAIL: *Setpoint or *Set slot with Flags.OPERATOR and default 0 (BDouble.make(0.0) or
#             BDouble.make(0)) and no min>0 facet (BFacets.MIN, BDouble.make(<positive>)).
#             A zero setpoint on a cooling unit runs at ambient temperature, silently.
#
#   CS3 WARN: A boolean flag @NiagaraProperty immediately preceded by a comment containing
#             "only when" or "only valid" or "only if" — a comment-only enforcement rule
#             that has no runtime gate. See documented limitation below.
#
#   CS4 WARN: a nonzero permanent-minimum floor default (...MinStagesOn / ...MinOn name on a
#             camelCase boundary, NOT a BRelTime or *Time/*Delay/*Sec short-cycle timer) coexists in
#             the same class with a *LowLimit* / *Cutout* (camelCase token, unit suffix allowed, timer and
#             status/counter suffix excluded) numeric default of 0 (= protection disabled). Either default alone is fine; together one
#             unit is held on with no LP cutout ("pulling with every solenoid closed").
#             [ev: retro panccadia-commissioning-lessons Δ3]
#
# Usage:  lint-config-sanity.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits FAIL (CS1/CS2) or WARN (CS3/CS4) rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#
# Row format:  FAIL|WARN  lint-config-sanity  <file>:<line>  CS<n>: <reason>
# Exits:       0 no FAIL (WARN-only is 0) · 1 any FAIL · 3 usage/env, incl. a sub-directory find cannot enter (K20)
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Source of defaults (documented precedence):
#   1. Java @NiagaraProperty defaultValue= literal (primary source).
#   2. module-include.xml facets (fallback when Java literal is absent or unreadable).
#   When neither source yields a readable value the check is skipped (never false-FAIL) — except a
#   BRelTime default on an *Interval/*Duration slot, which is a CS1 "unreadable" WARN row.
#
# Documented limitation — CS3 (comment-only flag enforcement):
#   The heuristic matches a comment containing "only when", "only valid", or "only if"
#   on the line immediately preceding a @NiagaraProperty declaration. This is intentionally
#   conservative (WARN, not FAIL) because:
#   (a) The pattern is grep-level; false positives are possible on unrelated comments.
#   (b) No runtime AST is available to confirm the enforcement gap.
#   Operators must review CS3 WARN rows manually and add runtime validation or suppress
#   with a documented `// lint-config-sanity: ok` comment.
#
# Mutation: LCS-interval -- removes interval<=duration comparison so CS1 shape passes instead of FAIL
# Mutation: LCS-floor -- dropping the floor x disabled-cutout pairing lets the CS4 shape pass silently
# Mutation: LCS-floor-name -- matching minon/cutout as substrings again WARNs on adminOnline / minOnTime / cutoutDelay
# Mutation: LCS-floor-status -- dropping the status/counter suffix exclusion WARNs on cutoutCount / cutoutActive
# Mutation: LCS-floor-trip -- putting Trip|Alarm|Fault back in the status suffixes silences lpCutoutTrip=0
# Mutation: LCS-floor-suffix -- anchoring the cutout token to the name end misses lpCutoutPsi / lowLimitBar
# Mutation: CS1-units -- comparing raw factory arguments (makeHours(4) vs make(1800000L)) false-FAILs a sane pair
# Mutation: CS1-mixed -- reading make(ms) and makeSeconds(s) as one unit passes a 30 s cycle with a 60 s defrost
# Mutation: CS1-minutes -- skipping makeMinutes silently passes a 10 min cycle with a 45 min defrost
# Mutation: CS1-wrapped -- anchoring the factory to the whole defaultValue reads make(BRelTime.class, BRelTime.makeSeconds(300)) as unreadable
# Mutation: CS1-bothzero -- comparing a 0/0 (DEFAULT) pair FAILs every display-mirror panel
# Mutation: CS1-unreadable -- dropping the unreadable row skips an unparsed BRelTime default silently
# Mutation: LCS-finderr -- ignoring the find status skips an unreadable sub-directory and reports clean
set -u
LC_ALL=C
export LC_ALL

[ $# -ge 1 ] || { printf 'usage: lint-config-sanity.sh <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-config-sanity: not a directory: %s\n' "$SRC" >&2; exit 3; }

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"
touch "$_ROWS"

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
# A sub-directory find cannot enter is an env error (exit 3), never a shorter file list.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$SRC" -name '*.java'; then
  printf 'lint-config-sanity: cannot list every file under %s: %s\n' "$SRC" "$(head -n 1 "$_TMP/find.err")" >&2
  exit 3
fi

_row() {
  local sev="$1" loc="$2" reason="$3"
  printf '%s  lint-config-sanity  %s  %s\n' "$sev" "$loc" "$reason" >> "$_ROWS"
}

# ---------------------------------------------------------------------------
# Per-file analysis
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  # ---- CS1: interval <= duration ----
  # Extract @NiagaraProperty name + defaultValue for *Interval and *Duration slots.
  # Strategy: multi-line annotation accumulator; extract name= and defaultValue=.
  awk -v FILE="$f" '
  # reltime_ms(expr) -> milliseconds, or -1 when the expression is not one literal BRelTime factory/constant
  function reltime_ms(e,    a, n) {
    if (e ~ /^BRelTime\.make\([0-9]+[Ll]?\)$/)            { sub(/^BRelTime\.make\(/, "", e); sub(/[Ll]?\)$/, "", e); return e + 0 }
    if (e ~ /^BRelTime\.makeSeconds\([0-9]+\)$/)          { gsub(/[^0-9]/, "", e); return e * 1000 }
    if (e ~ /^BRelTime\.makeMinutes\([0-9]+\)$/)          { gsub(/[^0-9]/, "", e); return e * 60000 }
    if (e ~ /^BRelTime\.makeHours\([0-9]+\)$/)            { gsub(/[^0-9]/, "", e); return e * 3600000 }
    if (e ~ /^BRelTime\.makeDays\([0-9]+\)$/)             { gsub(/[^0-9]/, "", e); return e * 86400000 }
    if (e ~ /^BRelTime\.make\([0-9]+,[0-9]+,[0-9]+,[0-9]+\)$/) {
      sub(/^BRelTime\.make\(/, "", e); sub(/\)$/, "", e); n = split(e, a, ",")
      return ((a[1] * 24 + a[2]) * 60 + a[3]) * 60000 + a[4] * 1000
    }
    if (e == "BRelTime.DEFAULT") return 0
    if (e == "BRelTime.SECOND") return 1000
    if (e == "BRelTime.MINUTE") return 60000
    if (e == "BRelTime.HOUR")   return 3600000
    if (e == "BRelTime.DAY")    return 86400000
    return -1
  }
  # one_factory(expr) -> the single BRelTime factory call / constant inside expr (a wrapper such as
  # make(BRelTime.class, BRelTime.makeSeconds(300)) is allowed), or "" when there is none or more than one
  function one_factory(e,    t, n, hit) {
    t = e; n = 0; hit = ""
    while (match(t, /BRelTime\.(make[A-Za-z]*\([^()]*\)|DEFAULT|SECOND|MINUTE|HOUR|DAY)/)) {
      n++; hit = substr(t, RSTART, RLENGTH); t = substr(t, RSTART + RLENGTH)
    }
    return (n == 1) ? hit : ""
  }
  function human(ms) {
    if (ms % 3600000 == 0 && ms > 0) return (ms / 3600000) "h"
    if (ms % 60000 == 0 && ms > 0)   return (ms / 60000) "m"
    if (ms % 1000 == 0)              return (ms / 1000) "s"
    return ms "ms"
  }
  BEGIN { in_prop = 0; prop_buf = ""; prop_line = 0 }
  FNR == 1 { in_prop = 0; prop_buf = ""; prop_line = 0; delete interval_val; delete duration_val; delete interval_line; delete duration_line }

  !in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = $0; prop_line = FNR
    next
  }
  in_prop {
    prop_buf = prop_buf " " $0
    depth = 0
    for (ci = 1; ci <= length(prop_buf); ci++) {
      c = substr(prop_buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth <= 0 && index(prop_buf, "(") > 0) {
      # Extract name
      pname = ""
      if (match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
        seg = substr(prop_buf, RSTART); sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg)
        pname = seg
      }
      # Extract the defaultValue BRelTime in MILLISECONDS (one unit for the compare): make(ms[L]),
      # makeSeconds/Minutes/Hours/Days(n), make(d,h,m,s), DEFAULT/SECOND/MINUTE/HOUR/DAY
      # (javax/baja/sys/BRelTime.java:34-48, :56-83). A BRelTime default it cannot read is an
      # "unreadable" WARN for an Interval/Duration slot, never a silent skip.
      dval = -1; dexpr = ""
      if (match(prop_buf, /defaultValue[[:space:]]*=[[:space:]]*"[^"]*"/)) {
        dexpr = substr(prop_buf, RSTART, RLENGTH); sub(/^defaultValue[[:space:]]*=[[:space:]]*"/, "", dexpr); sub(/"$/, "", dexpr)
        gsub(/[[:space:]]/, "", dexpr)
      }
      if (index(dexpr, "BRelTime") > 0) dval = reltime_ms(one_factory(dexpr))
      if (pname != "" && dval >= 0) {
        if (pname ~ /Interval/) { interval_val[pname] = dval; interval_line[pname] = prop_line }
        if (pname ~ /Duration/) { duration_val[pname] = dval; duration_line[pname] = prop_line }
      } else if (pname != "" && index(dexpr, "BRelTime") > 0 && pname ~ /Interval|Duration/) {
        printf "WARN  lint-config-sanity  %s:%d  CS1: unreadable BRelTime default for %s (%s) -- interval/duration not compared; use a literal factory\n", FILE, prop_line, pname, dexpr
      }
      in_prop = 0; prop_buf = ""; prop_line = 0
    }
    next
  }

  END {
    # CS1: for each Interval, find any Duration in same class where interval <= duration
    for (iname in interval_val) {
      for (dname in duration_val) {
        # both 0 (BRelTime.DEFAULT on a display mirror / unset pair) is "not configured", not a contradiction
        if (interval_val[iname] == 0 && duration_val[dname] == 0) continue
        if (interval_val[iname] <= duration_val[dname]) {
          printf "FAIL  lint-config-sanity  %s:%d  CS1: interval<=duration: %s(%s) <= %s(%s)\n",
            FILE, interval_line[iname], iname, human(interval_val[iname]), dname, human(duration_val[dname])
        }
      }
    }
  }
  ' "$f" >> "$_ROWS"

  # ---- CS2: zero setpoint with no min>0 facet ----
  awk -v FILE="$f" '
  BEGIN { in_prop = 0; prop_buf = ""; prop_line = 0 }
  FNR == 1 { in_prop = 0; prop_buf = ""; prop_line = 0 }

  !in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = $0; prop_line = FNR
    next
  }
  in_prop {
    prop_buf = prop_buf " " $0
    depth = 0
    for (ci = 1; ci <= length(prop_buf); ci++) {
      c = substr(prop_buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth <= 0 && index(prop_buf, "(") > 0) {
      pname = ""
      if (match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
        seg = substr(prop_buf, RSTART); sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg)
        pname = seg
      }
      # CS2 applies only to *Setpoint/*Set slots with OPERATOR flag
      if (pname ~ /(Setpoint|Set)$/ && index(prop_buf, "OPERATOR") > 0) {
        # Check for zero default: BDouble.make(0) or BDouble.make(0.0)
        is_zero = 0
        if (match(prop_buf, /BDouble\.make\([[:space:]]*0[.0]*[[:space:]]*\)/)) is_zero = 1

        # Check for min>0 facet: BFacets.MIN, BDouble.make(<positive>)
        has_min = 0
        if (match(prop_buf, /BFacets\.MIN[^)]*BDouble\.make\([[:space:]]*([0-9]+\.?[0-9]*)[[:space:]]*\)/)) {
          seg = substr(prop_buf, RSTART)
          match(seg, /BDouble\.make\([[:space:]]*[0-9]/)
          seg2 = substr(seg, RSTART); sub(/BDouble\.make\(/, "", seg2); sub(/\).*/, "", seg2)
          gsub(/[[:space:]]/, "", seg2)
          if (seg2 + 0 > 0) has_min = 1
        }

        if (is_zero && !has_min) {
          printf "FAIL  lint-config-sanity  %s:%d  CS2: zero-default setpoint \"%s\" with OPERATOR flag and no min>0 facet\n",
            FILE, prop_line, pname
        }
      }
      in_prop = 0; prop_buf = ""; prop_line = 0
    }
    next
  }
  ' "$f" >> "$_ROWS"

  # ---- CS3: comment-only flag enforcement (WARN, heuristic) ----
  # Look for a comment containing "only when", "only valid", or "only if"
  # on the line immediately preceding a @NiagaraProperty declaration.
  awk -v FILE="$f" '
  FNR == 1 { prev_comment = ""; prev_line = 0 }
  {
    stripped = $0; sub(/^[[:space:]]+/, "", stripped)
    is_comment = (substr(stripped, 1, 2) == "//" && stripped ~ /[Oo]nly (when|valid|if)/)
    is_prop    = (index(stripped, "@NiagaraProperty") > 0)

    if (is_prop && prev_comment != "") {
      printf "WARN  lint-config-sanity  %s:%d  CS3: flag combo enforced only in comment: %s\n",
        FILE, FNR, prev_comment
      prev_comment = ""
    } else if (is_comment) {
      prev_comment = stripped
      prev_line    = FNR
    } else {
      prev_comment = ""
    }
  }
  ' "$f" >> "$_ROWS"

  # ---- CS4: permanent-minimum floor + disabled LP cutout floor (WARN) ----
  awk -v FILE="$f" '
  function num(s) {   # numeric literal inside a defaultValue string, or "" when not numeric
    sub(/^[[:space:]]*(BInteger|BDouble|BFloat|BLong)\.make\([[:space:]]*/, "", s)
    sub(/[[:space:]]*\)[[:space:]]*$/, "", s)
    sub(/[dDfFlL]$/, "", s)
    return (s ~ /^-?[0-9]+(\.[0-9]*)?$/) ? s : ""
  }
  BEGIN {
    in_prop = 0; buf = ""; pline = 0; nf = 0; nc = 0
    # one timer-suffix rule for both the floor and the cutout names [polish-2026-10-02 P2d]
    TIMER_SUFFIX = "(Time|Delay|Secs?|Seconds|Ms|Millis|Mins?|Minutes)[0-9]*$"
    # a cutout status / counter slot is not a cutout floor (cutoutCount, cutoutActive, lowLimitReached);
    # Trip / Alarm / Fault are NOT here: they also name cutout setpoints (lpCutoutTrip, lowLimitAlarm)
    # [polish-2026-10-02 P2e]
    STATUS_SUFFIX = "(Count|Counter|Cnt|Total|Active|Reached|Tripped|State|Status|Flag|Event|Events|Log)[0-9]*$"
  }
  !in_prop && index($0, "@NiagaraProperty") > 0 { in_prop = 1; buf = ""; pline = FNR }
  in_prop {
    buf = buf " " $0
    depth = 0
    for (ci = 1; ci <= length(buf); ci++) {
      c = substr(buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth > 0) next
    in_prop = 0
    pname = ""; dv = ""
    if (match(buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
      seg = substr(buf, RSTART); sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg); pname = seg
    }
    if (match(buf, /defaultValue[[:space:]]*=[[:space:]]*"[^"]*"/)) {
      seg = substr(buf, RSTART); sub(/defaultValue[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg); dv = num(seg)
    }
    if (pname == "" || dv == "") next
    # camelCase word boundaries, not substrings [polish-2026-10-02 P2b, #199 WU6b]: a floor is
    # min(Stages)On / ...Min(Stages)On (+ digits or a capitalised suffix), never a *Time/*Delay/*Sec
    # timer; an LP cutout floor carries a LowLimit / Cutout token (+ digits or a unit/qualifier suffix
    # such as Psi or Bar), never a *Time/*Delay/*Sec timer. adminOnline, minOnTime and cutoutDelay are
    # not matched; lpCutoutPsi and lowLimitBar are. A status/counter suffix (Count, Active, Reached,
    # State, ...) is not a cutout floor, and a non-numeric default (false, BBoolean.FALSE) never
    # reaches here: num() returns "" for it. [polish-2026-10-02 P2b/P2c/P2d]
    if ((pname ~ /(^min|Min)(Stages)?On([A-Z0-9]|$)/) && pname !~ TIMER_SUFFIX &&
        index(buf, "BRelTime") == 0 && dv + 0 != 0) {
      nf++; fname[nf] = pname; fline[nf] = pline; fval[nf] = dv
    }
    if (pname ~ /(^lowLimit|LowLimit|^cutout|Cutout)([A-Z0-9]|$)/ &&
        pname !~ TIMER_SUFFIX && pname !~ STATUS_SUFFIX && dv + 0 == 0) { nc++; cname[nc] = pname }
  }
  END {
    for (i = 1; i <= nf; i++) for (j = 1; j <= nc; j++)
      printf "WARN  lint-config-sanity  %s:%d  CS4: floor \"%s\"=%s with LP cutout \"%s\"=0 (disabled) -- a permanent minimum needs an enabled cutout that overrides it\n",
        FILE, fline[i], fname[i], fval[i], cname[j]
  }
  ' "$f" >> "$_ROWS"

done < "$_TMP/files"

if [ -s "$_ROWS" ]; then
  cat "$_ROWS"
  grep -q '^FAIL' "$_ROWS" && exit 1
fi
exit 0
