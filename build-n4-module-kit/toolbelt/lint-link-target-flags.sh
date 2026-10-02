#!/usr/bin/env bash
# lint-link-target-flags.sh — flags on linked slots: READONLY on a link-in target (LTF1),
# TRANSIENT on an operator mode/HOA slot (LTF2).
#
#   LTF1 FAIL: a @NiagaraProperty that is a link-in TARGET carries Flags.READONLY. Workbench
#              LinkCheck refuses a READONLY target (linkcheck.propReadonly), so the mirror can never
#              be linked and the dashboard shows the type default forever. A slot is a known link-in
#              target when (a) the comment block heading its run of consecutive annotations says
#              "link-in", "linked from", "commissioning link" or "written by (a) BLink"
#              (case-insensitive; the block ends at a blank line, code or another annotation),
#              or (b) it is a backticked first-column slot of Table 2 (control -> facade) in the
#              --wiring-map file (generate-wiring-map.sh layout).
#              READONLY stays legal for a slot the owning component/reader sets itself and no
#              external Link targets (e.g. a timer anchor). [ev: retro panccadia-commissioning-lessons Δ1]
#              [ev: retro panccadia-persistent-config-hoa Δ3]
#   LTF2 FAIL: an OPERATOR slot named *Mode / *Hoa (optionally digit-suffixed) or hoa* carries
#              Flags.TRANSIENT. A TRANSIENT operator choice is never encoded into config.bog, reverts
#              to its default (AUTO) on restart, and a facade->control link then re-pushes that
#              default onto the persisted control slot. Lives here (not lint-config-sanity) because the
#              failure is a flag on a link endpoint, the same class of defect as LTF1.
#              A non-OPERATOR computed status mode may stay TRANSIENT. [ev: retro panccadia-persistent-config-hoa Δ1]
#
# Heuristic limits (documented, grep-level): a link-in target with neither a comment nor a wiring-map
# row is invisible to LTF1 — keep the module comment convention or pass --wiring-map. Flags are
# matched as the READONLY / TRANSIENT / OPERATOR tokens anywhere in the annotation.
#
# Mutation: LTF-comment -- dropping the link-comment detection lets a commented READONLY link target pass
# Mutation: LTF-map -- dropping the --wiring-map Table 2 harvest lets a mapped READONLY target pass
# Mutation: LTF-transient -- dropping the TRANSIENT operator-mode check lets a non-persisted HOA pass
#
# Row:    FAIL  lint-link-target-flags  <file>:<line>  LTF<n>: <reason>
# Usage:  lint-link-target-flags.sh [--wiring-map <docs/wiring-map.md>] <java-src-dir>
# Exits:  0 clean · 1 any FAIL · 3 usage/env (K20; no Java sources is exit 3 + ERROR row)
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2). LC_ALL=C.
set -u
LC_ALL=C
export LC_ALL

usage_exit() {
  printf 'usage: lint-link-target-flags.sh [--wiring-map <docs/wiring-map.md>] <java-src-dir>\n' >&2
  exit 3
}

MAP=""
SRC=""
while [ $# -gt 0 ]; do
  case "$1" in
    --wiring-map)
      [ $# -ge 2 ] || usage_exit
      MAP="$2"; shift 2 ;;
    -*) usage_exit ;;
    *) [ -z "$SRC" ] || usage_exit; SRC="$1"; shift ;;
  esac
done
[ -n "$SRC" ] || usage_exit
[ -d "$SRC" ] || { printf 'lint-link-target-flags: not a directory: %s\n' "$SRC" >&2; exit 3; }
if [ -n "$MAP" ] && [ ! -f "$MAP" ]; then
  printf 'lint-link-target-flags: --wiring-map file not found: %s\n' "$MAP" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort > "$_TMP/files.txt"
if [ ! -s "$_TMP/files.txt" ]; then
  printf 'ERROR  lint-link-target-flags  %s  no Java sources found (K20: wrong path or empty scaffold)\n' "$SRC"
  exit 3
fi

# Table 2 (control -> facade) first-column slots of the wiring map: the facade link-in targets.
MAP_TARGETS=" "
if [ -n "$MAP" ]; then
  MAP_TARGETS=" $(awk '
    /^##[[:space:]]/ { in2 = ($0 ~ /Table 2/) ; next }
    in2 && /^\|/ {
      cell = $0; sub(/^\|[[:space:]]*/, "", cell); sub(/[[:space:]]*\|.*$/, "", cell)
      if (cell ~ /^`[A-Za-z_][A-Za-z0-9_]*`$/) { gsub(/`/, "", cell); print cell }
    }
  ' "$MAP" | sort -u | tr '\n' ' ') "
fi

: > "$_TMP/rows.txt"
while IFS= read -r f; do
  awk -v FILE="$f" -v MAP_TARGETS="$MAP_TARGETS" '
  function is_comment(s) { return (s ~ /^[[:space:]]*(\/\/|\/\*|\*)/) }
  BEGIN { in_prop = 0; buf = ""; pline = 0; cmt = ""; prev_cmt = 0 }
  {
    line = $0
    if (!in_prop) {
      if (index(line, "@NiagaraProperty") > 0) {
        in_prop = 1; buf = line; pline = FNR
      } else if (is_comment(line)) {
        cmt = (prev_cmt ? cmt " " : "") line   # a comment after a property starts a new block
        prev_cmt = 1
        next
      } else {   # blank line, code or another annotation (@NiagaraType): the block ends
        cmt = ""; prev_cmt = 0
        next
      }
    } else {
      buf = buf " " line
    }
    depth = 0
    for (ci = 1; ci <= length(buf); ci++) {
      c = substr(buf, ci, 1)
      if (c == "(") depth++
      else if (c == ")") depth--
    }
    if (depth > 0 || index(buf, "(") == 0) next

    pname = ""
    if (match(buf, /name[[:space:]]*=[[:space:]]*"[^"]+"/)) {
      seg = substr(buf, RSTART); sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg)
      pname = seg
    }
    if (pname != "") {
      lc = tolower(cmt)
      linked = (lc ~ /link-in|linked from|commissioning link|written by (a )?blink/)
      mapped = (index(MAP_TARGETS, " " pname " ") > 0)
      if (index(buf, "READONLY") > 0 && (linked || mapped)) {
        why = linked ? "link comment" : "wiring-map Table 2"   # "(wiring-map Table 2)" is matched by report-module.sh and commissioning-verify.sh
        printf "FAIL  lint-link-target-flags  %s:%d  LTF1: link-in target \"%s\" (%s) carries READONLY -- LinkCheck refuses a READONLY target; declare it SUMMARY only and guard writes server-side\n", FILE, pline, pname, why
      }
      if (index(buf, "TRANSIENT") > 0 && index(buf, "OPERATOR") > 0 &&
          (pname ~ /(Mode|Hoa|HOA)[0-9]*$/ || pname ~ /^hoa/)) {
        printf "FAIL  lint-link-target-flags  %s:%d  LTF2: operator mode/HOA slot \"%s\" is TRANSIENT -- it reverts to its default on restart and a link re-pushes it; persist it\n", FILE, pline, pname
      }
    }
    in_prop = 0; buf = ""; pline = 0; prev_cmt = 0   # cmt kept: a block heads its run of properties
  }
  ' "$f" >> "$_TMP/rows.txt"
done < "$_TMP/files.txt"

if [ -s "$_TMP/rows.txt" ]; then
  cat "$_TMP/rows.txt"
  grep -q '^FAIL' "$_TMP/rows.txt" && exit 1
fi
exit 0
