#!/usr/bin/env bash
# lint-config-sanity.sh — Unsafe @NiagaraProperty default patterns gate (Wave 3, LR3, B862).
#
# Detects four unsafe default patterns in @NiagaraProperty annotations:
#
#   CS1 FAIL: *Interval default <= *Duration default in the same class.
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
#             the same class with a *LowLimit / *Cutout (name ending) default of 0 (= protection disabled). Either default alone is fine; together one
#             unit is held on with no LP cutout ("pulling with every solenoid closed").
#             [ev: retro panccadia-commissioning-lessons Δ3]
#
# Usage:  lint-config-sanity.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits FAIL (CS1/CS2) or WARN (CS3/CS4) rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#
# Row format:  FAIL|WARN  lint-config-sanity  <file>:<line>  CS<n>: <reason>
# Exits:       0 no FAIL (WARN-only is 0) · 1 any FAIL · 3 usage/env (K20)
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Source of defaults (documented precedence):
#   1. Java @NiagaraProperty defaultValue= literal (primary source).
#   2. module-include.xml facets (fallback when Java literal is absent or unreadable).
#   When neither source yields a readable value the check is skipped (never false-FAIL).
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
      # Extract defaultValue= seconds literal from BRelTime.makeSeconds(N) or BRelTime.make(N)
      dval = -1
      if (match(prop_buf, /defaultValue[[:space:]]*=[[:space:]]*"[^"]*BRelTime\.makeSeconds\([0-9]+\)/)) {
        seg = substr(prop_buf, RSTART)
        match(seg, /BRelTime\.makeSeconds\([0-9]+\)/)
        seg2 = substr(seg, RSTART); sub(/BRelTime\.makeSeconds\(/, "", seg2); sub(/\).*/, "", seg2)
        dval = seg2 + 0
      } else if (match(prop_buf, /defaultValue[[:space:]]*=[[:space:]]*"[^"]*BRelTime\.make\([0-9]+\)/)) {
        seg = substr(prop_buf, RSTART)
        match(seg, /BRelTime\.make\([0-9]+\)/)
        seg2 = substr(seg, RSTART); sub(/BRelTime\.make\(/, "", seg2); sub(/\).*/, "", seg2)
        dval = seg2 + 0
      }
      if (pname != "" && dval >= 0) {
        if (pname ~ /Interval/) { interval_val[pname] = dval; interval_line[pname] = prop_line }
        if (pname ~ /Duration/) { duration_val[pname] = dval; duration_line[pname] = prop_line }
      }
      in_prop = 0; prop_buf = ""; prop_line = 0
    }
    next
  }

  END {
    # CS1: for each Interval, find any Duration in same class where interval <= duration
    for (iname in interval_val) {
      for (dname in duration_val) {
        if (interval_val[iname] <= duration_val[dname]) {
          printf "FAIL  lint-config-sanity  %s:%d  CS1: interval<=duration: %s(%ds) <= %s(%ds)\n",
            FILE, interval_line[iname], iname, interval_val[iname], dname, duration_val[dname]
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
  BEGIN { in_prop = 0; buf = ""; pline = 0; nf = 0; nc = 0 }
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
    # timer; an LP cutout floor ENDS in LowLimit or Cutout (+ digits). adminOnline, minOnTime and
    # cutoutDelay are not matched.
    if ((pname ~ /(^min|Min)(Stages)?On([A-Z0-9]|$)/) && pname !~ /(Time|Delay|Secs?|Seconds|Ms|Millis|Mins?|Minutes)[0-9]*$/ &&
        index(buf, "BRelTime") == 0 && dv + 0 != 0) {
      nf++; fname[nf] = pname; fline[nf] = pline; fval[nf] = dv
    }
    if (pname ~ /(^lowLimit|LowLimit|^cutout|Cutout)[0-9]*$/ && dv + 0 == 0) { nc++; cname[nc] = pname }
  }
  END {
    for (i = 1; i <= nf; i++) for (j = 1; j <= nc; j++)
      printf "WARN  lint-config-sanity  %s:%d  CS4: floor \"%s\"=%s with LP cutout \"%s\"=0 (disabled) -- a permanent minimum needs an enabled cutout that overrides it\n",
        FILE, fline[i], fname[i], fval[i], cname[j]
  }
  ' "$f" >> "$_ROWS"

done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

if [ -s "$_ROWS" ]; then
  cat "$_ROWS"
  grep -q '^FAIL' "$_ROWS" && exit 1
fi
exit 0
