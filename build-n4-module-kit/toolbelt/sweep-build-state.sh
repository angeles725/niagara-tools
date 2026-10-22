#!/usr/bin/env bash
# sweep-build-state.sh — CONTENT-ONLY validator for BUILD-STATE.md + retros/INDEX.md.
#
# This script is VCS-free by design. kit-links.bats L2 forbids a toolbelt/*.sh from invoking
# version control (these scripts run inside worktrees and on stations); the diff half of the
# retro gate lives in .githooks/pre-push, which calls this script for the content half.
#
# usage (content check):
#   sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>
#   exit:  0 clean · 1 named integrity violation · 3 usage/env
#
# usage (retro-debt aging — E5 contract):
#   sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age <N>] <retros-dir> <INDEX.md>
#   exit:  0 always (non-strict); use --strict (not implemented) to exit 1 on escalation
#   stdout (three lines):
#     total_pending=<n>
#     escalated_count=<n>
#     oldest_age=<n|N/A>
#   Only pending rows age; folded rows and 0-delta pending rows whose INDEX line carries a
#   `<!-- disposition: tracked-by-issue #<n> … -->` or `<!-- disposition: worklist-complete … -->`
#   comment (a parked non-retro doc) are excluded; any other disposition still ages.  Age is derived from the retro's FILENAME
#   date prefix (YYYY-MM-DD-…), never from the file body.  Today is INJECTED via --today.
#
# content checks:
#   - build-state.v1 markers are balanced, anchored at column 0 (a marker quoted in prose,
#     inside backticks or indented, is NOT an envelope);
#   - each envelope carries module + retro_required + retro_pending, both booleans;
#   - the open_issues: multi-line list is tolerated (list items are not fields);
#   - INDEX integrity: every retro file has a row, every row points at a real retro, and every
#     row's review-status is pending or folded;
#   - freshness: when INDEX.md is the GENERATED index (line 1 names toolbelt/gen-retro-index.sh),
#     it must equal what `gen-retro-index.sh --check <kit-root>` regenerates from the retro headers
#     (kit root = the retros dir's parent); a hand-written index (test fixtures) skips this check;
#   - ledger freshness: when a ledger.d/ directory sits beside <BUILD-STATE.md>, the kit block's
#     generated parts (last_change, open_issues items, session log) must equal what
#     `gen-build-state.sh --check <kit-root>` regenerates from ledger.d/; no ledger.d/ skips it.
set -u

fail() { echo "sweep: $*" >&2; exit 1; }

# marker_of <retro-file> — print the first whitespace-delimited word after "review-status:"
# from a column-0 anchored "<!-- review-status: …" comment within the first 5 lines.
# Returns the empty string when no such line exists (absent marker is tolerated).
# This script is VCS-free; the word is extracted without invoking version control.
marker_of() {
  sed -n '1,5p' "$1" \
    | grep -m1 -E '^<!--[[:space:]]*review-status:' \
    | sed -E 's/^<!--[[:space:]]*review-status:[[:space:]]*//' \
    | awk '{print $1}'
}

# ---------------------------------------------------------------------------
# --age mode: retro-debt aging report (E5 contract)
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--age" ]; then
  shift
  TODAY="" MAX_AGE=30
  while [ $# -gt 0 ]; do
    case "$1" in
      --today)    [ $# -ge 2 ] || { printf 'usage: sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>\n' >&2; exit 3; }; TODAY="$2"; shift 2 ;;
      --max-age)  [ $# -ge 2 ] || { printf 'usage: sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>\n' >&2; exit 3; }; MAX_AGE="$2"; shift 2 ;;
      --) shift; break ;;
      -*) printf 'usage: sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>\n' >&2; exit 3 ;;
      *) break ;;
    esac
  done
  [ -n "$TODAY" ] || { printf 'sweep-build-state: --today <YYYY-MM-DD> is required\n' >&2; exit 3; }
  [ $# -eq 2 ] || { printf 'usage: sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>\n' >&2; exit 3; }
  retrodir="$1"; index="$2"
  [ -d "$retrodir" ] || { printf 'sweep-build-state: no retros dir: %s\n' "$retrodir" >&2; exit 3; }
  [ -f "$index" ]    || { printf 'sweep-build-state: no INDEX file: %s\n' "$index"   >&2; exit 3; }

  # disposition_skips <INDEX line>: 0 (skip) only for a 0-delta row whose disposition value is allowed.
  disposition_skips() {
    local row="$1" deltas
    case "$row" in *'<!-- disposition: tracked-by-issue #'[0-9]*|*'<!-- disposition: worklist-complete'*) : ;; *) return 1 ;; esac
    deltas=$(printf '%s\n' "$row" | awk -F'|' '{for(i=1;i<=NF;i++){gsub(/^ +| +$/,"",$i); if($i=="pending"||$i=="folded"){print $(i+1); exit}}}' | tr -d ' ')
    [ "$deltas" = "0" ]
  }

  today_epoch=$(date -d "$TODAY" +%s)
  total_pending=0 escalated_count=0 oldest_age=-1

  while IFS= read -r line; do
    case "$line" in \|*) : ;; *) continue ;; esac
    printf '%s\n' "$line" | grep -qE '[0-9]{4}-[0-9]{2}-[0-9]{2}[^|]*\.md' || continue
    fname=$(printf '%s\n' "$line" | grep -oE '[A-Za-z0-9._-]+\.md' | grep -vx 'INDEX.md' | head -1)
    [ -n "$fname" ] || continue
    rstatus=$(printf '%s\n' "$line" | awk -F'|' '{for(i=1;i<=NF;i++){gsub(/^ +| +$/,"",$i); if($i=="pending"||$i=="folded"){print $i; exit}}}')
    [ "$rstatus" = "pending" ] || continue
    # A non-retro doc parked in retros/ (a BUILD proposal tracked by an issue, a completed
    # worklist) carries `<!-- disposition: <why> -->` on its INDEX row: not retro debt, never ages.
    # The skip is bounded (fail closed): the row's deltas cell (the cell after the status) must be
    # 0 AND the disposition must start with an allowed value — `tracked-by-issue #<n>` or
    # `worklist-complete`. Any other disposition, or one on a row that enumerates deltas, still ages.
    # Mutation: D8 -- dropping the 0-delta check hides a real retro's debt behind a comment.
    # Mutation: D9 -- accepting any disposition value hides a row parked with no tracked reason.
    if disposition_skips "$line"; then continue; fi
    total_pending=$(( total_pending + 1 ))
    # Extract YYYY-MM-DD from the filename prefix (first 10 chars)
    file_date="${fname:0:10}"
    file_epoch=$(date -d "$file_date" +%s 2>/dev/null) || continue
    age_days=$(( (today_epoch - file_epoch) / 86400 ))
    if [ "$age_days" -gt "$oldest_age" ]; then oldest_age=$age_days; fi
    if [ "$age_days" -gt "$MAX_AGE" ]; then
      escalated_count=$(( escalated_count + 1 ))
    fi
  done < "$index"

  printf 'total_pending=%d\n' "$total_pending"
  printf 'escalated_count=%d\n' "$escalated_count"
  if [ "$total_pending" -eq 0 ]; then
    printf 'oldest_age=N/A\n'
  else
    printf 'oldest_age=%d\n' "$oldest_age"
  fi
  exit 0
fi

# ---------------------------------------------------------------------------
# Default mode: content-integrity check (BUILD-STATE.md + retros/INDEX.md)
# ---------------------------------------------------------------------------
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
  m=$(marker_of "$retrodir/$fname")
  if [ -n "$m" ]; then
    case "$m" in
      pending|folded) ;;
      *) fail "retro marker out of domain: $fname has review-status '$m' (pending|folded)" ;;
    esac
    [ "$m" = "$rstatus" ] || fail "retro marker disagrees with INDEX row: $fname marker='$m' INDEX='$rstatus'"
  fi
  rowseen["$fname"]=1
done < "$index"

shopt -s nullglob
for f in "$retrodir"/*.md; do
  b=$(basename "$f")
  [ "$b" = "INDEX.md" ] && continue
  [ -n "${rowseen[$b]:-}" ] || fail "retro has no INDEX row: $b"
done

# ---- 3. generated INDEX freshness -------------------------------------------
# Mutation: GRI-sweep -- skipping the freshness call lets a deltas/module header drift pass the gate
if head -1 "$index" | grep -q '^<!-- GENERATED by toolbelt/gen-retro-index\.sh'; then
  gen="$(dirname "${BASH_SOURCE[0]}")/gen-retro-index.sh"
  [ -f "$gen" ] || { echo "sweep: generated INDEX but no generator: $gen" >&2; exit 3; }
  [ "$(cd "$retrodir" && pwd)/INDEX.md" = "$(cd "$(dirname "$index")" && pwd)/INDEX.md" ] \
    || fail "generated INDEX must be <retros-dir>/INDEX.md: $index"
  bash "$gen" --check "$(cd "$retrodir/.." && pwd)" >/dev/null \
    || fail "retros/INDEX.md is stale vs the retro headers; run toolbelt/gen-retro-index.sh"
fi

# ---- 4. generated kit ledger freshness ---------------------------------------
# Mutation: GBS-sweep -- skipping this call lets a fragment land without its regenerated BUILD-STATE.md
stateroot="$(cd "$(dirname "$state")" && pwd)"
if [ -d "$stateroot/ledger.d" ]; then
  gen="$(dirname "${BASH_SOURCE[0]}")/gen-build-state.sh"
  [ -f "$gen" ] || { echo "sweep: ledger.d/ present but no generator: $gen" >&2; exit 3; }
  [ "$(basename "$state")" = "BUILD-STATE.md" ] || fail "a ledger.d/ kit needs the state file named BUILD-STATE.md: $state"
  bash "$gen" --check "$stateroot" >/dev/null \
    || fail "BUILD-STATE.md is stale vs ledger.d/ (or a fragment is malformed); run toolbelt/gen-build-state.sh"
fi

exit 0
