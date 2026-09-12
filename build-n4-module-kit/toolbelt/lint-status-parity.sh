#!/usr/bin/env bash
# lint-status-parity.sh — Per-instance config/status slot ratio gate (Wave 3, LR2, B861).
#
# Detects facade classes where the count of per-instance config slots
# (*Interval*, *Duration*, *Setpoint*; N>1) exceeds the count of scalar status
# slots (*Status, *Active, *Start, *Since; M). When N>1 and M<N this is an
# asymmetric facade: operators cannot tell which of the N configured instances
# is currently active/running.
#
# Default:     WARN (exit 0) — name-heuristic false positives are real for
#              legitimately aggregated rollup facades; accept WARN as advisory.
# With --strict: FAIL (exit 1).
#
# Usage:  lint-status-parity.sh [--strict] <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits WARN rows (or FAIL under --strict); exits 0 WARN-only, 1 under --strict, 3 usage/env.
#
# Row format:  WARN|FAIL  lint-status-parity  <class>  N=<n> per-instance config slots but <m> status slots
# Exits:       0 clean or WARN-only · 1 any WARN under --strict · 3 usage/env (K20)
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Documented limitation (name-heuristic):
#   Legitimately aggregated rollup facades may trigger a WARN when they intentionally
#   have more config slots than status slots (e.g. a master facade aggregating N rooms
#   with one global active flag). Use --strict only after auditing the WARN list.
#   Source of defaults: @NiagaraProperty name= attribute parsed by grep-level awk;
#   module-include.xml facets are NOT consulted for this check.
#
# Mutation: LSP-asymmetric -- removes N>M asymmetry detection so all facades pass regardless of parity
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'usage: lint-status-parity.sh [--strict] <java-src-dir>\n' >&2; exit 3 ;;
    *) break ;;
  esac
done

[ $# -ge 1 ] || { printf 'usage: lint-status-parity.sh [--strict] <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-status-parity: not a directory: %s\n' "$SRC" >&2; exit 3; }

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"
touch "$_ROWS"

_row() {
  local sev="$1" cls="$2" reason="$3"
  printf '%s  lint-status-parity  %s  %s\n' "$sev" "$cls" "$reason" >> "$_ROWS"
}

# Per-file analysis via awk.
# Collects @NiagaraProperty name= attributes and classifies them.
while IFS= read -r f; do
  # Derive class name from filename
  cls=$(basename "$f" .java)

  # Extract all @NiagaraProperty name= values from this file (multi-line annotation aware).
  # Strategy: collect lines between @NiagaraProperty and the next non-annotation line;
  # extract name="..." from the accumulated buffer.
  awk -v CLASS="$cls" '
  BEGIN { in_prop = 0; prop_buf = ""; n_config = 0; n_status = 0 }
  FNR == 1 { in_prop = 0; prop_buf = ""; n_config = 0; n_status = 0 }

  !in_prop && index($0, "@NiagaraProperty") > 0 {
    in_prop = 1; prop_buf = $0
    next
  }
  in_prop {
    prop_buf = prop_buf " " $0
    # Check paren balance to know when the annotation is complete
    depth = 0
    for (ci = 1; ci <= length(prop_buf); ci++) {
      c = substr(prop_buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth <= 0 && index(prop_buf, "(") > 0) {
      # Annotation complete — extract name
      pname = ""
      if (match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
        seg = substr(prop_buf, RSTART)
        sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
        sub(/".*/, "", seg)
        pname = seg
      }
      if (pname != "") {
        # Config: name contains Interval, Duration, or Setpoint
        if (pname ~ /(Interval|Duration|Setpoint)/) n_config++
        # Status: name ends with Status, Active, Start, or Since
        if (pname ~ /(Status|Active|Start|Since)$/) n_status++
      }
      in_prop = 0; prop_buf = ""
    }
    next
  }

  END {
    if (n_config > 1 && n_status < n_config) {
      printf "%s %d %d\n", CLASS, n_config, n_status
    }
  }
  ' "$f"
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort) | \
while IFS=" " read -r cls n_config n_status; do
  sev="WARN"
  [ "$STRICT" -eq 1 ] && sev="FAIL"
  _row "$sev" "$cls" "N=${n_config} per-instance config slots but ${n_status} status slots"
done

if [ -s "$_ROWS" ]; then
  cat "$_ROWS"
  if [ "$STRICT" -eq 1 ]; then
    grep -q '^FAIL' "$_ROWS" && exit 1
  fi
fi
exit 0
