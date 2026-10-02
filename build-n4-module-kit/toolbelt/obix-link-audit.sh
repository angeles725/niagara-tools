#!/usr/bin/env bash
# obix-link-audit.sh — declared wiring map vs the links actually present on a live component.
#
# A link from the WRONG-but-valid source (e.g. a status slot fed by a raw relay command instead
# of the current-proven running flag) is structurally fine, so bog-audit.sh CHECK7/9/11 cannot
# see it. This tool diffs the source column of docs/wiring-map.md (generate-wiring-map.sh)
# against the baja:Link children of ONE component, read over oBIX (read-only GET) or from a
# saved oBIX response.
#
# Usage:
#   obix-link-audit.sh --map <wiring-map.md> --xml <component.xml> [--table 1|2]
#   obix-link-audit.sh --map <wiring-map.md> --obix <base-url> --component <path>
#                      [--table 1|2] [--insecure]
#
#   --table 2 (default)  "SUMMARY display slots (control → facade)": the audited component is
#                        the FACADE; each facade slot must be linked from its declared RT slot.
#   --table 1            "OPERATOR config slots (facade → control)": the audited component is
#                        the CONTROL component; each RT slot must be linked from its facade slot.
#   --obix               base like https://<station>/obix; GETs <base>/config/<path>/ once.
#                        Credentials come from OBIX_USER / OBIX_PASS and reach curl on stdin
#                        (curl -K -), never on the command line. --insecure accepts a
#                        self-signed station certificate.
#
# Declared names are compared on their last segment (Rack/comp1Running, Rack.comp1Running and
# comp1Running are the same slot). A row whose declared column is empty or _(fill)_ is SKIP.
#
# Rows:    STATUS  obix-link-audit  <slot>  <detail>   (MATCH | MISMATCH | MISSING | SKIP)
# Summary: obix-link-audit: M MATCH · X MISMATCH · Y MISSING · S SKIP  ->  CLEAN|ISSUES
# Exit:    0 no MISMATCH/MISSING · 1 any MISMATCH/MISSING · 3 usage/env
# Read-only: no station write; VCS-free (kit-links.bats L2).
# [ev: retro panccadia-commissioning-lessons Δ4]

set -u
LC_ALL=C; export LC_ALL

MAP=""; XML=""; BASE=""; COMP=""; TABLE=2; INSECURE=0

usage_exit() {
  printf 'usage: obix-link-audit.sh --map <wiring-map.md> (--xml <component.xml> | --obix <base-url> --component <path>) [--table 1|2] [--insecure]\n' >&2
  exit 3
}

while [ $# -gt 0 ]; do
  case "$1" in
    --map)       [ $# -ge 2 ] || usage_exit; MAP="$2"; shift 2 ;;
    --xml)       [ $# -ge 2 ] || usage_exit; XML="$2"; shift 2 ;;
    --obix)      [ $# -ge 2 ] || usage_exit; BASE="$2"; shift 2 ;;
    --component) [ $# -ge 2 ] || usage_exit; COMP="$2"; shift 2 ;;
    --table)     [ $# -ge 2 ] || usage_exit; TABLE="$2"; shift 2 ;;
    --insecure)  INSECURE=1; shift ;;
    *) usage_exit ;;
  esac
done

[ -n "$MAP" ] || usage_exit
[ -f "$MAP" ] || { printf 'obix-link-audit: map not found: %s\n' "$MAP" >&2; exit 3; }
case "$TABLE" in 1|2) ;; *) usage_exit ;; esac
if [ -n "$XML" ]; then
  [ -z "$BASE" ] || usage_exit
  [ -z "$COMP" ] || usage_exit
  [ -f "$XML" ] || { printf 'obix-link-audit: xml not found: %s\n' "$XML" >&2; exit 3; }
else
  [ -n "$BASE" ] || usage_exit
  [ -n "$COMP" ] || usage_exit
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

if [ -z "$XML" ]; then
  if [ -z "${OBIX_USER:-}" ] || [ -z "${OBIX_PASS:-}" ]; then
    printf 'obix-link-audit: live mode needs OBIX_USER and OBIX_PASS (a read-only oBIX user)\n' >&2
    exit 3
  fi
  command -v curl >/dev/null 2>&1 || { printf 'obix-link-audit: curl not found\n' >&2; exit 3; }
  _comp="${COMP#/}"; _comp="${_comp%/}"
  URL="${BASE%/}/config/${_comp}/"
  XML="$_TMP/component.xml"
  curl_args=(-fsS --max-time 20 -K - -o "$XML")
  [ "$INSECURE" -eq 1 ] && curl_args+=(-k)
  if ! printf 'user = "%s:%s"\n' "$OBIX_USER" "$OBIX_PASS" | curl "${curl_args[@]}" "$URL"; then
    printf 'obix-link-audit: GET failed: %s\n' "$URL" >&2
    exit 3
  fi
fi

# Declared rows of the selected table: "<audited-slot>\t<declared-source>\t<label>"
# Table 2: audited = facade slot (col 1), declared source = RT slot (col 2).
# Table 1: audited = RT slot (col 2),     declared source = facade slot (col 1).
awk -F'|' -v T="$TABLE" '
  function last(s) { gsub(/`/, "", s); gsub(/^[ \t]+|[ \t]+$/, "", s); sub(/.*[\/.]/, "", s); return s }
  function trim(s) { gsub(/`/, "", s); gsub(/^[ \t]+|[ \t]+$/, "", s); return s }
  /^## / { sec = ($0 ~ ("^## Table " T " ")) ? 1 : 0; next }
  !sec || NF < 4 { next }
  {
    f = trim($2); r = trim($3)
    if (f == "" || f ~ /^-+$/ || f == "Facade slot" || substr(f, 1, 2) == "_(") next
    undeclared = (r == "" || substr(r, 1, 2) == "_(")
    if (T == 2) printf "%s\t%s\t%s\n", f, (undeclared ? "" : last(r)), f
    else        printf "%s\t%s\t%s\n", (undeclared ? "" : last(r)), f, f
  }' "$MAP" > "$_TMP/declared.tsv"

[ -s "$_TMP/declared.tsv" ] || { printf 'obix-link-audit: no Table %s rows in %s\n' "$TABLE" "$MAP" >&2; exit 3; }

# Actual links into the component: "<target-slot>\t<source-slot>\t<source-component>"
# from display="Indirect|Direct: <src>.<slot> → slot:/<tgt>.<slot>" (&#x2192;, → or ->).
grep -o 'display="\(Indirect\|Direct\): [^"]*"' "$XML" \
  | sed -e 's/^display="[A-Za-z]*: //' -e 's/"$//' -e 's/&#x2192;/->/g' -e 's/→/->/g' \
  | awk '{
      p = index($0, "->"); if (p == 0) next
      src = substr($0, 1, p - 1); tgt = substr($0, p + 2)
      gsub(/^[ \t]+|[ \t]+$/, "", src); gsub(/^[ \t]+|[ \t]+$/, "", tgt)
      ss = src; sub(/.*\./, "", ss); sc = src; sub(/\.[^.]*$/, "", sc)
      ts = tgt; sub(/.*\./, "", ts)
      printf "%s\t%s\t%s\n", ts, ss, sc
    }' > "$_TMP/links.tsv"

awk -F'\t' '
  FILENAME == ARGV[1] { n[$1]++; src[$1, n[$1]] = $2; comp[$1, n[$1]] = $3; next }
  {
    slot = $1; want = $2; label = $3
    if (want == "" || slot == "") {
      printf "SKIP  obix-link-audit  %s  source not declared in the wiring map (fill it before commissioning)\n", label
      ns++; next
    }
    if (!(slot in n)) {
      printf "MISSING  obix-link-audit  %s  declared %s, no link into %s\n", slot, want, slot
      nmiss++; next
    }
    ok = 0; got = ""
    for (i = 1; i <= n[slot]; i++) {
      if (src[slot, i] == want) ok = 1
      got = got (got == "" ? "" : ", ") src[slot, i] " (" comp[slot, i] ")"
    }
    if (ok) { printf "MATCH  obix-link-audit  %s  linked from declared %s\n", slot, want; nm++ }
    else    { printf "MISMATCH  obix-link-audit  %s  declared %s, linked from %s\n", slot, want, got; nx++ }
  }
  END {
    printf "obix-link-audit: %d MATCH · %d MISMATCH · %d MISSING · %d SKIP  ->  %s\n",
      nm, nx, nmiss, ns, ((nx + nmiss) > 0 ? "ISSUES" : "CLEAN")
    exit ((nx + nmiss) > 0 ? 1 : 0)
  }' "$_TMP/links.tsv" "$_TMP/declared.tsv"
