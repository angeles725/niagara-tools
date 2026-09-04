#!/usr/bin/env bash
# sweep-build-state.sh — CONTENT-ONLY validator for BUILD-STATE.md + retros/INDEX.md.
#
# This script is VCS-free by design. kit-links.bats L2 forbids a toolbelt/*.sh from invoking
# version control (these scripts run inside worktrees and on stations); the diff half of the
# retro gate lives in .githooks/pre-push, which calls this script for the content half.
#
# usage: sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>
# exit:  0 clean · 1 named integrity violation · 3 usage/env
#
# checks:
#   - build-state.v1 markers are balanced, anchored at column 0 (a marker quoted in prose,
#     inside backticks or indented, is NOT an envelope);
#   - each envelope carries module + retro_required + retro_pending, both booleans;
#   - the open_issues: multi-line list is tolerated (list items are not fields);
#   - INDEX integrity: every retro file has a row, every row points at a real retro, and every
#     row's review-status is pending or folded.
set -u

fail() { echo "sweep: $*" >&2; exit 1; }

[ "$#" -eq 3 ] || { echo "usage: sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>" >&2; exit 3; }
state=$1; retrodir=$2; index=$3
[ -f "$state" ]    || { echo "sweep: no BUILD-STATE file: $state" >&2; exit 3; }
[ -d "$retrodir" ] || { echo "sweep: no retros dir: $retrodir"    >&2; exit 3; }
[ -f "$index" ]    || { echo "sweep: no INDEX file: $index"       >&2; exit 3; }

# ---- 1. envelope well-formedness (column-0 anchored markers) ----------------
opens=$(grep -cE '^<!-- build-state\.v1 -->$' "$state")
closes=$(grep -cE '^<!-- /build-state\.v1 -->$' "$state")
[ "$opens" -eq "$closes" ] || fail "unbalanced build-state.v1 markers ($opens open / $closes close)"

inside=0 module="" req="" pend="" have_mod=0 have_req=0 have_pend=0
while IFS= read -r line; do
  if [ "$line" = '<!-- build-state.v1 -->' ]; then
    inside=1; module=""; req=""; pend=""; have_mod=0; have_req=0; have_pend=0; continue
  fi
  if [ "$line" = '<!-- /build-state.v1 -->' ]; then
    [ "$have_mod" -eq 1 ]  || fail "envelope missing required field: module"
    [ "$have_req" -eq 1 ]  || fail "envelope ($module) missing required field: retro_required"
    [ "$have_pend" -eq 1 ] || fail "envelope ($module) missing required field: retro_pending"
    case "$req"  in true|false) ;; *) fail "envelope ($module) retro_required not boolean: '$req'"  ;; esac
    case "$pend" in true|false) ;; *) fail "envelope ($module) retro_pending not boolean: '$pend'" ;; esac
    inside=0; continue
  fi
  [ "$inside" -eq 1 ] || continue
  case "$line" in
    *[![:space:]]*) : ;;                 # non-blank
    *) continue ;;                        # blank line inside envelope
  esac
  # tolerate the open_issues list: lines like "  - <text>" are not fields
  case "$line" in
    [[:space:]]*-[[:space:]]*) continue ;;
  esac
  key=${line%%:*}
  val=${line#*:}
  val=${val%%#*}                          # strip inline comment
  # trim
  val="${val#"${val%%[![:space:]]*}"}"
  val="${val%"${val##*[![:space:]]}"}"
  case "$key" in
    module)         module=$val; [ -n "$val" ] && have_mod=1 ;;
    retro_required) req=$val;  have_req=1 ;;
    retro_pending)  pend=$val; have_pend=1 ;;
  esac
done < "$state"

# ---- 2. INDEX integrity ------------------------------------------------------
declare -A rowseen
while IFS= read -r line; do
  case "$line" in \|*) : ;; *) continue ;; esac
  # data row iff it carries a dated retro filename (skips header + |---| separator)
  echo "$line" | grep -qE '[0-9]{4}-[0-9]{2}-[0-9]{2}[^|]*\.md' || continue
  fname=$(echo "$line" | grep -oE '[A-Za-z0-9._-]+\.md' | grep -vx 'INDEX.md' | head -1)
  [ -n "$fname" ] || continue
  rstatus=$(echo "$line" | awk -F'|' '{for(i=1;i<=NF;i++){gsub(/^ +| +$/,"",$i); if($i=="pending"||$i=="folded"){print $i; exit}}}')
  [ -n "$rstatus" ] || fail "INDEX row for $fname has no valid review-status (pending|folded)"
  [ -f "$retrodir/$fname" ] || fail "INDEX row points at a missing retro: $fname"
  rowseen["$fname"]=1
done < "$index"

shopt -s nullglob
for f in "$retrodir"/*.md; do
  b=$(basename "$f")
  [ "$b" = "INDEX.md" ] && continue
  [ -n "${rowseen[$b]:-}" ] || fail "retro has no INDEX row: $b"
done

exit 0
