#!/usr/bin/env bash
# triage-console.sh — own-module exception triage over station console logs (Campaign 8 PR2).
# Three attribution channels:
#   C1  own frame   — stack line at <PKG>. under an exception
#   C2  own logger  — WARNING/SEVERE with a tag in the own-tag WHITELIST (stack-less ok);
#                     also: message contains the own-package path (e.g. [loader] cert-chain)
#   C3  bog-drift   — [sys] fatal exception, [sys.xml] slot-drift, [sys.registry] Missing class
# Usage: triage-console.sh [--package PKG] [--tag TAG_LIST] [--module-prefix PFX_LIST]
#                          [--console-dir DIR] <console.txt|dir>...
# Exit: 0 no rows · 1 any row · 3 usage/env
# [ev: corpus B800]  [ev: retro campaign8-triage-console]
set -u

FAILED=0
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

PKG="com.angeles"
OWN_TAGS="coldRoomPan,dashboardPan,dashboardpan,compPan,chihuahua"
MOD_PREFIX="ColdRoomPan,DashboardPan,CompPan,chihuahua"
CONSOLE_DIR=""
FILES=()
_DIR_ARG=""

while [ $# -gt 0 ]; do
  case "$1" in
    --package)       shift; PKG="$1";        shift ;;
    --tag)           shift; OWN_TAGS="$1";   shift ;;
    --module-prefix) shift; MOD_PREFIX="$1"; shift ;;
    --console-dir)   shift; CONSOLE_DIR="$1"; shift ;;
    --) shift; break ;;
    -*) printf 'triage-console: unknown option: %s\n' "$1" >&2; exit 3 ;;
    *)  FILES+=("$1"); shift ;;
  esac
done

[ -n "$CONSOLE_DIR" ] && FILES+=("$CONSOLE_DIR")

# Expand directories (D9b: prune dot-dirs); collect plain files
EXPANDED=()
for arg in "${FILES[@]+"${FILES[@]}"}"; do
  if [ -d "$arg" ]; then
    _DIR_ARG="$arg"
    while IFS= read -r -d '' f; do
      EXPANDED+=("$f")
    done < <(find "$arg" -maxdepth 1 -not -path '*/.*' -type f -name '*.txt' -print0 | sort -z)
  elif [ -f "$arg" ]; then
    EXPANDED+=("$arg")
  fi
done
FILES=("${EXPANDED[@]+"${EXPANDED[@]}"}")

[ "${#FILES[@]}" -gt 0 ] || { printf 'usage: triage-console.sh [...] <console.txt>...\n' >&2; exit 3; }

# Subject: dir arg or multiple files -> dir basename; single file -> file basename
if [ -n "$_DIR_ARG" ]; then
  SUBJECT=$(basename "$_DIR_ARG")
elif [ "${#FILES[@]}" -eq 1 ]; then
  SUBJECT=$(basename "${FILES[0]}")
else
  SUBJECT=$(basename "$(dirname "${FILES[0]}")")
fi

cat > "$_TMP/triage.awk" << 'ENDAWK'
BEGIN {
  mo["Jan"]=1;  mo["Feb"]=2;  mo["Mar"]=3;  mo["Apr"]=4
  mo["May"]=5;  mo["Jun"]=6;  mo["Jul"]=7;  mo["Aug"]=8
  mo["Sep"]=9;  mo["Oct"]=10; mo["Nov"]=11; mo["Dec"]=12
  mo["Ene"]=1;  mo["Abr"]=4;  mo["Ago"]=8;  mo["Set"]=9; mo["Dic"]=12
  mo["jan"]=1;  mo["feb"]=2;  mo["mar"]=3;  mo["apr"]=4
  mo["may"]=5;  mo["jun"]=6;  mo["jul"]=7;  mo["aug"]=8
  mo["sep"]=9;  mo["oct"]=10; mo["nov"]=11; mo["dec"]=12
  mo["ene"]=1;  mo["abr"]=4;  mo["ago"]=8;  mo["set"]=9; mo["dic"]=12

  n_ot = split(OWN_TAGS, ot_arr, ",")
  for (i = 1; i <= n_ot; i++) own_tag[ot_arr[i]] = 1

  n_mp = split(MOD_PREFIX, mp_arr, ",")
  for (i = 1; i <= n_mp; i++) mod_pfx[mp_arr[i]] = 1

  # own-package path form for C2b (e.g. "com.angeles" -> "com/angeles")
  own_path = PKG; gsub(/\./, "/", own_path)

  has_blk = 0
}

# Flush pending block on file boundary (enables cross-file grouping)
FNR == 1 && has_blk { flush_blk() }

function is_level(s) {
  if (s == "SEVERE" || s == "WARNING" || s == "INFO" ||
      s == "CONFIG" || s == "FINE"    || s == "FINER" || s == "FINEST" ||
      s == "GRAVE"  || s == "ADVERTENCIA") return 1
  if (substr(s, 1, 8) == "INFORMACI") return 1
  return 0
}

function norm_level(lv) {
  if (lv == "GRAVE")       return "SEVERE"
  if (lv == "ADVERTENCIA") return "WARNING"
  if (substr(lv, 1, 8) == "INFORMACI") return "INFO"
  return lv
}

function norm(s,    r) { r = s; gsub(/[0-9]+/, "N", r); return r }

function simple_cls(fqn,    n, a) { n = split(fqn, a, "."); return a[n] }

# "at pkg.Cls.mth(File.java:N)" -> "Cls.mth"  (grouping key — no line#, merges refactors)
function frame_key(at_line,    p, cls_mth, n, parts) {
  sub(/^at /, "", at_line)
  p = index(at_line, "(")
  cls_mth = (p > 0) ? substr(at_line, 1, p - 1) : at_line
  n = split(cls_mth, parts, ".")
  if (n >= 2) return parts[n-1] "." parts[n]
  return cls_mth
}

# "at pkg.Cls.mth(File.java:N)" -> "Cls.mth(File.java:N)"  (display — preserves site)
function frame_disp(at_line,    p, cls_mth, rest, n, parts) {
  sub(/^at /, "", at_line)
  p = index(at_line, "(")
  if (p == 0) return at_line
  cls_mth = substr(at_line, 1, p - 1)
  rest    = substr(at_line, p)
  n = split(cls_mth, parts, ".")
  if (n >= 2) return parts[n-1] "." parts[n] rest
  return cls_mth rest
}

function ts_key(ts,    yy, mon, dd, hh, mm, ss) {
  hh = substr(ts, 1, 2); mm = substr(ts, 4, 2); ss = substr(ts, 7, 2)
  dd = substr(ts, 10, 2); mon = substr(ts, 13, 3); yy = substr(ts, 17, 2)
  return yy sprintf("%02d", mo[mon]+0) dd hh mm ss
}

function record(key, ts, lvn, dmsg, frame,    k) {
  k = ts_key(ts)
  if (!(key in g_cnt)) {
    g_cnt[key] = 0; g_first[key] = ts; g_first_k[key] = k
    g_last[key] = ts; g_last_k[key] = k
    g_level[key] = lvn; g_msg[key] = dmsg; g_frame[key] = frame
  }
  g_cnt[key]++
  if (k < g_first_k[key]) { g_first_k[key] = k; g_first[key] = ts }
  if (k > g_last_k[key])  { g_last_k[key]  = k; g_last[key]  = ts }
}

function flush_blk(    lvn, dmsg, key) {
  if (!has_blk) return
  has_blk = 0
  lvn = blk_level

  # C1 — own frame present (highest priority); key uses Class.method, display adds (File.java:N)
  if (blk_own_fkey != "") {
    dmsg = (blk_ex_class != "") \
           ? simple_cls(blk_ex_class) (blk_ex_msg != "" ? ": " blk_ex_msg : "") \
           : blk_hdr_msg
    key = lvn SUBSEP blk_ex_class SUBSEP norm(blk_ex_msg) SUBSEP blk_own_fkey
    record(key, blk_ts, lvn, dmsg, blk_own_fdisp)
    return
  }

  # C3a — SEVERE [sys] with an exception class (shape-based; EN+ES agnostic)
  if (blk_tag == "sys" && lvn == "SEVERE" && blk_ex_class != "") {
    dmsg = simple_cls(blk_ex_class) (blk_ex_msg != "" ? ": " blk_ex_msg : "")
    key  = lvn SUBSEP blk_ex_class SUBSEP norm(blk_ex_msg) SUBSEP "[sys]"
    record(key, blk_ts, lvn, dmsg, "[sys]")
    return
  }

  # C2 — own logger tag whitelist
  if ((lvn == "WARNING" || lvn == "SEVERE") && (blk_tag in own_tag)) {
    dmsg = blk_hdr_msg
    key  = lvn SUBSEP norm(blk_hdr_msg) SUBSEP blk_tag
    record(key, blk_ts, lvn, dmsg, "[" blk_tag "]")
    return
  }

  # C2b — message contains own-package path (e.g. [loader] cert-chain for our class)
  if ((lvn == "WARNING" || lvn == "SEVERE") && own_path != "" && index(blk_hdr_msg, own_path) > 0) {
    dmsg = blk_hdr_msg
    key  = lvn SUBSEP norm(blk_hdr_msg) SUBSEP blk_tag
    record(key, blk_ts, lvn, dmsg, "[" blk_tag "]")
    return
  }
}

# C3b — [sys.xml] WARNING single-line bog-drift (immediate, no block)
function handle_sysxml(ts, msg) {
  record("WARNING" SUBSEP norm(msg) SUBSEP "[sys.xml]",
         ts, "WARNING", msg, "[sys.xml]")
}

# C3c — [sys.registry] Missing class( for)? "?OwnPrefix:Type (B818: both forms)
function handle_registry(ts, msg,    p, rest, colon, prefix) {
  p = index(msg, "Missing class")
  if (p == 0) return
  rest = substr(msg, p + 13)
  while (substr(rest, 1, 1) ~ /[ \t]/) rest = substr(rest, 2)
  if (substr(rest, 1, 3) == "for")      rest = substr(rest, 4)
  while (substr(rest, 1, 1) ~ /[ \t]/) rest = substr(rest, 2)
  if (substr(rest, 1, 1) == "\"")      rest = substr(rest, 2)
  colon = index(rest, ":")
  if (colon == 0) return
  prefix = substr(rest, 1, colon - 1)
  if (!(prefix in mod_pfx)) return
  record("SEVERE" SUBSEP norm(msg) SUBSEP "[sys.registry]",
         ts, "SEVERE", msg, "[sys.registry]")
}

{
  f1       = $1
  f2_start = (NF >= 2) ? substr($2, 1, 1) : ""

  if (is_level(f1) && f2_start == "[") {
    flush_blk()

    lvn = norm_level(f1)
    # Skip "LEVEL " (length+1) and the "[" bracket (another +1) -> start at length+3
    rest = substr($0, length(f1) + 3)
    close_ts = index(rest, "]")
    if (close_ts == 0) { has_blk = 0; next }
    ts    = substr(rest, 1, close_ts - 1)
    rest2 = substr(rest, close_ts + 1)     # "[tag] message"
    if (substr(rest2, 1, 1) != "[") { has_blk = 0; next }
    close_tag = index(rest2, "]")
    if (close_tag == 0) { has_blk = 0; next }
    tag      = substr(rest2, 2, close_tag - 2)
    msg_rest = substr(rest2, close_tag + 2)

    # C3b — [sys.xml] single-line (immediate)
    if (tag == "sys.xml" && (lvn == "WARNING" || lvn == "SEVERE")) {
      handle_sysxml(ts, msg_rest)
      has_blk = 0; next
    }

    # C3c — [sys.registry] single-line (immediate)
    if (tag == "sys.registry" && lvn == "SEVERE") {
      handle_registry(ts, msg_rest)
      has_blk = 0; next
    }

    if (lvn == "WARNING" || lvn == "SEVERE") {
      has_blk = 1; blk_level = lvn; blk_ts = ts; blk_tag = tag
      blk_hdr_msg = msg_rest; blk_ex_class = ""; blk_ex_msg = ""
      blk_own_fkey = ""; blk_own_fdisp = ""
    } else {
      has_blk = 0
    }
    next
  }

  if (!has_blk) next
  cont = $0; gsub(/^[ \t]+/, "", cont)

  # Stack frame line
  if (substr(cont, 1, 3) == "at ") {
    fline = substr(cont, 4)
    if (blk_own_fkey == "" && substr(fline, 1, length(PKG)) == PKG) {
      blk_own_fkey  = frame_key(cont)
      blk_own_fdisp = frame_disp(cont)
    }
    next
  }

  # Exception class line (colon form or bare form)
  if (blk_ex_class == "") {
    if (index(cont, ":") > 0) {
      cp   = index(cont, ":")
      cand = substr(cont, 1, cp - 1)
      mpt  = substr(cont, cp + 2); gsub(/^[[:space:]]+/, "", mpt)
      if (index(cand, " ") == 0 && length(cand) > 0 &&
          (cand ~ /(Exception|Error|Throwable)$/ ||
           substr(cand, 1, 5) == "java."  ||
           substr(cand, 1, 6) == "javax.")) {
        blk_ex_class = cand; blk_ex_msg = mpt
      }
    } else if (index(cont, " ") == 0 && length(cont) > 0 &&
               (cont ~ /(Exception|Error|Throwable)$/ ||
                substr(cont, 1, 5) == "java."  ||
                substr(cont, 1, 6) == "javax.")) {
      blk_ex_class = cont; blk_ex_msg = ""
    }
  }
}

END {
  flush_blk()
  for (key in g_cnt) {
    printf "FAIL  triage-console  %s  %dx %s -> %s  %s %s @ %s\n",
      SUBJECT, g_cnt[key], g_first[key], g_last[key],
      g_level[key], g_msg[key], g_frame[key]
  }
}
ENDAWK

result=$(LC_ALL=C awk \
  -v PKG="$PKG" \
  -v OWN_TAGS="$OWN_TAGS" \
  -v MOD_PREFIX="$MOD_PREFIX" \
  -v SUBJECT="$SUBJECT" \
  -f "$_TMP/triage.awk" \
  "${FILES[@]+"${FILES[@]}"}")

if [ -n "$result" ]; then
  printf '%s\n' "$result"
  FAILED=1
fi

[ "$FAILED" -eq 0 ] || exit 1
exit 0
