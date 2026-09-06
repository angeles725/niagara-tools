#!/usr/bin/env bash
# triage-console.sh — own-module exception triage over station console logs (Campaign 8 PR2).
#
# Three attribution channels are required because two is not enough.
# Channel 1 and 2 together miss the worst class: a load-time station failure
# (BRelTime/BComplex slot drift in the saved .bog) produces a com.tridium-only
# stack — ZERO own frames, ZERO own logger lines — and the outage is invisible.
# Channel 3 catches it by shape alone (package-independent). [ev: corpus B800]
#
# Real defect (CERT-live): PANCCADIA BDefrostController.armTrigger "time <= 0"
# appeared 5x on 2026-09-02/03; the subsequent reload exposed slot drift and
# the station refused to start — surfaced ONLY via C3 ([sys]/[sys.xml]).
# [ev: corpus B800] [ev: retro campaign8-triage-console]
#
# Channels (DUAL/TRIPLE attribution — B800 §800.5):
#   C1  own frame   — a stack line "at <package>." under an exception
#   C2  own logger  — WARNING/SEVERE with non-denylist [tag], stack-less ok
#   C3  load-fail   — SEVERE [sys] with following exception (C3a) and/or
#                     [sys.xml] "Cannot set property / Missing frozen property /
#                     Cannot decode slot" bog-drift rows (C3b); shape-based
#
# Level normalization (ASCII prefix match, bytes never decoded — B800 §800.5):
#   GRAVE → SEVERE   ADVERTENCIA → WARNING   INFORMACI* → INFO
#
# Usage:
#   triage-console.sh [--package PKG] [--tag t1,t2] [--console-dir DIR]
#                     <console.txt|dir>…
#
#   --package PKG   own package prefix, e.g. com.angeles (C1 disabled if absent)
#   --tag t1,t2     restrict C2 to these logger tags (comma-separated)
#   --console-dir   scan console*.txt in DIR (non-recursive; dot-dirs excluded)
#
# Row:  FAIL  triage-console  <basename>  <count>x <first-ts> -> <last-ts>  <LEVEL> <msg> @ <frame>
# Exit: 0 no rows · 1 any row · 3 usage/env
#
# All reading is LC_ALL=C with grep -a / awk; bytes are never re-encoded.
# This script is VCS-free by design.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

FAILED=0
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

PKG=""
TAG_FILTER=""
CONSOLE_DIR=""
FILES=()

while [ $# -gt 0 ]; do
  case "$1" in
    --package)
      shift
      [ $# -gt 0 ] || { printf 'triage-console: --package requires an argument\n' >&2; exit 3; }
      PKG="$1"; shift
      ;;
    --tag)
      shift
      [ $# -gt 0 ] || { printf 'triage-console: --tag requires an argument\n' >&2; exit 3; }
      TAG_FILTER="$1"; shift
      ;;
    --console-dir)
      shift
      [ $# -gt 0 ] || { printf 'triage-console: --console-dir requires an argument\n' >&2; exit 3; }
      CONSOLE_DIR="$1"; shift
      ;;
    --)
      shift; break
      ;;
    -*)
      printf 'triage-console: unknown option: %s\n' "$1" >&2; exit 3
      ;;
    *)
      FILES+=("$1"); shift
      ;;
  esac
done
# Collect remaining positional args after --
for arg; do FILES+=("$arg"); done

# Expand --console-dir
if [ -n "$CONSOLE_DIR" ]; then
  [ -d "$CONSOLE_DIR" ] || { printf 'triage-console: not a directory: %s\n' "$CONSOLE_DIR" >&2; exit 3; }
  while IFS= read -r f; do FILES+=("$f"); done \
    < <(find "$CONSOLE_DIR" -maxdepth 1 -name 'console*.txt' | LC_ALL=C sort)
fi

# Expand directory positional args (D9b: prune dot-dirs)
EXPANDED=()
for f in "${FILES[@]+"${FILES[@]}"}"; do
  if [ -d "$f" ]; then
    while IFS= read -r ff; do EXPANDED+=("$ff"); done \
      < <(find "$f" -maxdepth 1 -type f -name 'console*.txt' 2>/dev/null | LC_ALL=C sort)
  else
    EXPANDED+=("$f")
  fi
done
FILES=("${EXPANDED[@]+"${EXPANDED[@]}"}")

[ "${#FILES[@]}" -gt 0 ] || {
  printf 'usage: triage-console.sh [--package PKG] [--tag t1,t2] [--console-dir DIR] <console.txt>...\n' >&2
  exit 3
}

# ---- awk program — write to temp file for shellcheck transparency ----
cat > "$_TMP/triage.awk" << 'ENDAWK'
BEGIN {
  # Month name → zero-padded 2-digit number (EN + ES)
  mo["Jan"]=1;  mo["Feb"]=2;  mo["Mar"]=3;  mo["Apr"]=4
  mo["May"]=5;  mo["Jun"]=6;  mo["Jul"]=7;  mo["Aug"]=8
  mo["Sep"]=9;  mo["Oct"]=10; mo["Nov"]=11; mo["Dec"]=12
  # Spanish (those that differ from EN abbreviations)
  mo["Ene"]=1;  mo["Abr"]=4;  mo["Ago"]=8;  mo["Set"]=9;  mo["Dic"]=12

  # C2 denylist: framework tags excluded from own-logger detection (web* handled inline)
  dn["sys"]=1; dn["sys.xml"]=1; dn["fox"]=1; dn["box"]=1
  dn["driver"]=1; dn["station"]=1; dn["alarm"]=1
  dn["history"]=1; dn["jetty"]=1

  # Tag filter (--tag option)
  n_tf = (TAG_FILTER != "") ? split(TAG_FILTER, tf, ",") : 0

  # Per-block state
  has_blk = 0
  blk_level = ""; blk_tag = ""; blk_ts = ""; blk_msg = ""
  blk_ex_class = ""; blk_ex_msg = ""; blk_own_frame = ""
}

# ---- helpers ----

function is_denied(tag) {
  if (tag in dn) return 1
  if (length(tag) >= 3 && substr(tag, 1, 3) == "web") return 1
  return 0
}

# C2: tag is "own" (not denied, or explicitly in --tag list)
function tag_ok(tag,    i) {
  if (n_tf == 0) return !is_denied(tag)
  for (i = 1; i <= n_tf; i++) if (tag == tf[i]) return 1
  return 0
}

# Normalize Niagara level keywords (ASCII prefix match, byte-safe)
function norm_level(lv) {
  if (lv == "GRAVE")                 return "SEVERE"
  if (lv == "ADVERTENCIA")           return "WARNING"
  if (substr(lv, 1, 8) == "INFORMACI") return "INFO"
  return lv
}

# Convert "HH:MM:SS DD-Mon-YY" to sortable YYMMDDHHMMS string
function ts_sort(ts,    p, n, dmy, hms, ms, yy) {
  n = split(ts, p, " ")
  if (n < 2) return "99999999999999"
  hms = p[1]; gsub(/:/, "", hms)
  n = split(p[2], dmy, "-")
  if (n < 3) return "99999999999999"
  ms = (dmy[2] in mo) ? sprintf("%02d", mo[dmy[2]]) : "00"
  yy = dmy[3]
  return yy ms dmy[1] hms
}

# Digit-normalize a string for grouping (replaces runs of digits with N)
function norm(s,    r) { r = s; gsub(/[0-9]+/, "N", r); return r }

# Extract simple class name from a fully-qualified name
function simple_cls(fqn,    n, p) {
  n = split(fqn, p, ".")
  return (n > 0) ? p[n] : fqn
}

# Classify the current block and record it in the groups arrays
function flush_blk(    key, dmsg, dframe, dlv, ts_s) {
  if (!has_blk) return
  has_blk = 0

  # ---- Priority: C1 > C3a > C3b > C2 ----
  if (blk_ex_class != "" && blk_own_frame != "") {
    # C1: exception whose stack passes through the own package
    key    = "C1" SUBSEP blk_level SUBSEP blk_ex_class SUBSEP norm(blk_ex_msg) SUBSEP blk_own_frame
    dmsg   = simple_cls(blk_ex_class) (blk_ex_msg != "" ? ": " blk_ex_msg : "")
    dframe = blk_own_frame
    dlv    = blk_level

  } else if (blk_tag == "sys" && blk_level == "SEVERE" && blk_ex_class != "") {
    # C3a: SEVERE [sys] load-time fatal (no own frame — tridium-only stack)
    key    = "C3a" SUBSEP blk_level SUBSEP blk_ex_class SUBSEP norm(blk_ex_msg)
    dmsg   = simple_cls(blk_ex_class) (blk_ex_msg != "" ? ": " blk_ex_msg : "")
    dframe = "[sys]"
    dlv    = blk_level

  } else if (blk_tag == "sys.xml" && blk_msg ~ \
    /Cannot set property|Missing frozen property|Cannot decode slot|No se puede asignar|Propiedad congelada ausente|No se puede decodificar/) {
    # C3b: [sys.xml] saved-bog schema-drift row (bog-load symptom)
    key    = "C3b" SUBSEP "WARNING" SUBSEP norm(blk_msg)
    dmsg   = blk_msg
    dframe = "[sys.xml]"
    dlv    = "WARNING"

  } else if ((blk_level == "WARNING" || blk_level == "SEVERE") && tag_ok(blk_tag)) {
    # C2: own logger — WARNING/SEVERE with non-denylist [tag]; stack-less ok
    key    = "C2" SUBSEP blk_level SUBSEP blk_tag SUBSEP norm(blk_msg)
    dmsg   = blk_msg
    dframe = "[" blk_tag "]"
    dlv    = blk_level

  } else {
    return
  }

  # Record group (first occurrence sets display strings)
  g_cnt[key]++
  if (!(key in g_dmsg)) {
    g_dmsg[key] = dmsg; g_dframe[key] = dframe; g_dlv[key] = dlv
  }
  ts_s = ts_sort(blk_ts)
  if (!(key in g_fs) || ts_s < g_fs[key]) { g_fs[key] = ts_s; g_fh[key] = blk_ts }
  if (!(key in g_ls) || ts_s > g_ls[key]) { g_ls[key] = ts_s; g_lh[key] = blk_ts }
}

# ---- main line processing ----
{
  line = $0
  gsub(/\r$/, "", line)   # strip CRLF

  # ---- LEVEL line detection ----
  # Match the level keyword prefix (byte-safe: INFORMACI matches INFORMACI\xD3N in LC_ALL=C)
  if (match(line, /^(SEVERE|WARNING|INFO|ERROR|DEBUG|FATAL|GRAVE|ADVERTENCIA|INFORMACI)/)) {
    raw_lv = substr(line, 1, RLENGTH)
    rest   = substr(line, RLENGTH + 1)

    # Require the [HH:MM:SS DD-Mon-YY TZ] timestamp bracket to confirm it is a log line
    if (match(rest, /\[[0-9][0-9]:[0-9][0-9]:[0-9][0-9] [0-9][0-9]-[A-Za-z][A-Za-z][A-Za-z]-[0-9][0-9] [A-Z]+\]/)) {
      ts_raw = substr(rest, RSTART, RLENGTH)  # "[HH:MM:SS DD-Mon-YY TZ]"
      after  = substr(rest, RSTART + RLENGTH)

      # Build display timestamp "HH:MM:SS DD-Mon-YY" (drop TZ and brackets)
      ts_inner = substr(ts_raw, 2, length(ts_raw) - 2)   # strip outer [ ]
      split(ts_inner, tsp, " ")
      ts_disp = tsp[1] " " tsp[2]   # "HH:MM:SS DD-Mon-YY"

      # Parse logger tag from the next [tag] bracket
      tag_s = ""; msg_s = after
      if (match(after, /^\[([A-Za-z][A-Za-z0-9._-]*)\]/)) {
        tag_s = substr(after, RSTART + 1, RLENGTH - 2)
        msg_s = substr(after, RSTART + RLENGTH)
        gsub(/^[[:space:]]+/, "", msg_s)
      }

      flush_blk()
      has_blk      = 1
      blk_level    = norm_level(raw_lv)
      blk_tag      = tag_s
      blk_ts       = ts_disp
      blk_msg      = msg_s
      blk_ex_class = ""
      blk_ex_msg   = ""
      blk_own_frame = ""
    }
    next
  }

  # ---- continuation line (indented exception / stack frame) ----
  if (!has_blk) next
  gsub(/^[[:space:]]+/, "", line)   # trim leading whitespace
  if (line == "") next              # blank line — skip

  # Frame line: "at <class>(<file>:<line>)"
  if (substr(line, 1, 3) == "at ") {
    frame = substr(line, 4)
    # C1 own-frame detection: frame must start with the own package prefix
    if (PKG != "" && blk_own_frame == "" && index(frame, PKG ".") == 1) {
      blk_own_frame = frame   # keep full frame for display; grouping uses it as-is
    }
    next
  }

  # Skip Caused by / Suppressed continuations
  if (substr(line, 1, 10) == "Caused by:" || substr(line, 1, 11) == "Suppressed:") next

  # Exception class line — two forms:
  #   "java.lang.FooException: message"  (colon + message)
  #   "javax.baja.sys.NotRunningException"  (bare class, no colon)
  # Must have no spaces in the candidate class name.
  if (blk_ex_class == "") {
    if (index(line, ":") > 0) {
      cp   = index(line, ":")
      cand = substr(line, 1, cp - 1)
      mpart = substr(line, cp + 1); gsub(/^[[:space:]]+/, "", mpart)
      if (index(cand, " ") == 0 && length(cand) > 0 && \
          (cand ~ /(Exception|Error|Throwable)$/ || \
           substr(cand, 1, 5) == "java." || substr(cand, 1, 6) == "javax.")) {
        blk_ex_class = cand
        blk_ex_msg   = mpart
      }
    } else if (index(line, " ") == 0 && length(line) > 0 && \
               (line ~ /(Exception|Error|Throwable)$/ || \
                substr(line, 1, 5) == "java." || substr(line, 1, 6) == "javax.")) {
      # Bare exception class with no message (e.g. javax.baja.sys.NotRunningException)
      blk_ex_class = line
      blk_ex_msg   = ""
    }
  }
}

END {
  flush_blk()
  for (key in g_cnt) {
    printf "FAIL  triage-console  %s  %dx %s -> %s  %s %s @ %s\n", \
      BNAME, g_cnt[key], g_fh[key], g_lh[key], g_dlv[key], g_dmsg[key], g_dframe[key]
  }
}
ENDAWK

# ---- Process each console file ----
for f in "${FILES[@]+"${FILES[@]}"}"; do
  [ -f "$f" ] || { printf 'triage-console: cannot read: %s\n' "$f" >&2; continue; }
  bname=$(basename "$f")
  result=$(LC_ALL=C awk \
    -v PKG="$PKG" \
    -v TAG_FILTER="$TAG_FILTER" \
    -v BNAME="$bname" \
    -f "$_TMP/triage.awk" \
    "$f")
  if [ -n "$result" ]; then
    printf '%s\n' "$result"
    FAILED=1
  fi
done

[ "$FAILED" -eq 0 ] || exit 1
exit 0
