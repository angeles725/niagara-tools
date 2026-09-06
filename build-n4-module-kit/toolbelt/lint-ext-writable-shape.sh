#!/usr/bin/env bash
# lint-ext-writable-shape.sh — ext-writable-shape lint (Campaign 9 PR10, B823/B826). [ev: retro campaign9-ext-writable-shape]
# Usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>
#
# A COMPLEX property (BStatusNumeric / BStatusBoolean / BStatusEnum) with Flags.OPERATOR and
# NO @NiagaraAction on its declaring class is a hazardous external-write target:
#   - a conformant oBIX PUT is rejected ("Cannot translate" — the slot is not BIObixWritable)
#   - a hand-crafted wrapped-<obj> PUT silently writes DEFAULT (0.0 for numeric) on a 200 OK
#     when the "value" child is omitted (the live silent-zero hazard, B823 §823.2)
# The safe external-write paths are:
#   (a) write the oBIX child-leaf bare <real> at …/<slot>/value (B826-G2, LIVE-CONFIRMED)
#   (b) add an OPERATOR @NiagaraAction that writes the slot (B822 additive-action doctrine)
#
# Exemption rule: a class that already exposes ANY @NiagaraAction provides at least one
# intentional external-action surface; the developer has thought about the action model.
# Accepted writing-action name patterns (EW3 is the only positive pin; B822):
#   set<Slot>  |  apply<Slot>  |  <slot>Cmd  (case-insensitive on the first letter)
# A class that exposes NO @NiagaraAction at all is the WARN case.
#
# Plain types (double, boolean, BRelTime, enum ordinal, String) are advertised writable="true"
# by the oBIX encoder — they accept a bare <real>/<bool> and need no child-leaf workaround.
# Only BStatus(Numeric|Boolean|Enum) (complex BStruct subtypes) are in scope here.
#
# Row:  WARN  ext-writable-shape  <file>:<line>  <slot>: OPERATOR <type> with no writing action
#             — external oBIX write must use the child leaf …/<slot>/value (bare <real>, B826)
#             or add an OPERATOR action (B822)
# Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2).
# [ev: corpus B823 §823.2]  [ev: corpus B826]  [ev: kit types/logic-authoring.md:62-70]
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>\n' >&2; exit 3 ;;
    *) break ;;
  esac
done

[ $# -ge 1 ] || { printf 'usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-ext-writable-shape: not a directory: %s\n' "$SRC" >&2; exit 3; }

# K20 / EW11: a source dir with no *.java files is a configuration error -> exit 3.
# Never emit a silent 0. Row contains ERROR + ext-writable-shape.
_nj=$(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | grep -c '\.java$' || true)
if [ "$_nj" -eq 0 ]; then
  printf 'ERROR  ext-writable-shape  %s  no Java sources found (K20: wrong path or empty scaffold)\n' "$SRC"
  exit 3
fi

WARNED=0

# shellcheck disable=SC2016  # awk program in single quotes — no shell expansion intended
while IFS= read -r f; do
  out=$(awk -v FILE="$f" '
  { lines[NR] = $0 }

  END {
    if (NR == 0) exit

    # -------------------------------------------------------------------------
    # Pass 1: collect @NiagaraAction names (paren-balanced join, same technique
    # as lint-silent-protection.sh pass0 and bog-audit.sh annotation scanner).
    # We only need to know IF the file has any action (class-level action check).
    # Presence of ANY @NiagaraAction — including HIDDEN callbacks — means the
    # developer has thought about the action surface for this class; the complex
    # OPERATOR slots are considered intentionally handled (B822 additive doctrine).
    # -------------------------------------------------------------------------
    has_action = 0
    in_ann = 0; buf = ""; depth = 0; prop_first = 0

    for (i = 1; i <= NR; i++) {
      ln = lines[i]

      if (!in_ann) {
        if (index(ln, "@NiagaraAction") > 0) {
          in_ann = 1; buf = ln; depth = 0; prop_first = 1
          # count parens on this first line
          tmp = ln
          for (ci = 1; ci <= length(tmp); ci++) {
            c = substr(tmp, ci, 1)
            if (c == "(") depth++
            else if (c == ")") { depth--; if (depth <= 0 && index(buf, "(") > 0) { in_ann = 0; has_action = 1; break } }
          }
          if (depth <= 0 && in_ann) { in_ann = 0; has_action = 1 }
        }
      } else {
        buf = buf " " ln
        for (ci = 1; ci <= length(ln); ci++) {
          c = substr(ln, ci, 1)
          if (c == "(") depth++
          else if (c == ")") { depth--; if (depth <= 0) { in_ann = 0; has_action = 1; break } }
        }
      }
      if (has_action) break   # one action is enough
    }

    # -------------------------------------------------------------------------
    # Pass 2: collect @NiagaraProperty annotations (paren-balanced multi-line
    # join as in C8 D9b / bog-audit; handles nested @Facet(...) parens).
    # For each: extract name, type, flags; emit WARN when:
    #   type ~ BStatus(Numeric|Boolean|Enum) (complex)  AND
    #   flags contain OPERATOR                          AND
    #   has_action == 0 (class exposes no @NiagaraAction)
    # -------------------------------------------------------------------------
    in_prop = 0; buf = ""; depth = 0; prop_first = 0; prop_line = 0

    for (i = 1; i <= NR; i++) {
      ln = lines[i]

      if (!in_prop) {
        if (index(ln, "@NiagaraProperty") > 0 && index(ln, "(") > 0) {
          in_prop = 1; buf = ln; depth = 0; prop_first = 1; prop_line = i
          for (ci = 1; ci <= length(ln); ci++) {
            c = substr(ln, ci, 1)
            if (c == "(") depth++
            else if (c == ")") depth--
          }
          if (depth <= 0) {
            # single-line annotation — process immediately
            _check_prop(buf, prop_line)
            in_prop = 0; buf = ""
          }
        }
      } else {
        buf = buf " " ln
        for (ci = 1; ci <= length(ln); ci++) {
          c = substr(ln, ci, 1)
          if (c == "(") depth++
          else if (c == ")") depth--
        }
        if (depth <= 0) {
          _check_prop(buf, prop_line)
          in_prop = 0; buf = ""
        }
      }
    }
  }

  function _check_prop(buf, lineno,    pname, ptype, pflags) {
    # Extract name
    pname = ""; ptype = ""; pflags = ""
    if (match(buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
      seg = substr(buf, RSTART)
      sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
      sub(/".*/, "", seg)
      pname = seg
    }
    if (pname == "") return

    # Extract type
    if (match(buf, /type[[:space:]]*=[[:space:]]*"[^"]*"/)) {
      seg = substr(buf, RSTART)
      sub(/type[[:space:]]*=[[:space:]]*"/, "", seg)
      sub(/".*/, "", seg)
      ptype = seg
    }
    if (ptype == "") return

    # Complex check: BStatusNumeric, BStatusBoolean, BStatusEnum
    # Accept with or without "B" prefix and with or without "baja:" namespace prefix.
    if (ptype !~ /([Bb]aja:)?(B)?[Ss]tatus[Nn]umeric|([Bb]aja:)?(B)?[Ss]tatus[Bb]oolean|([Bb]aja:)?(B)?[Ss]tatus[Ee]num/) return

    # Extract flags
    if (match(buf, /flags[[:space:]]*=[[:space:]]*/)) {
      seg = substr(buf, RSTART + RLENGTH)
      # grab until first "," or ")" that closes the flags value
      if (match(seg, /^[^,)]*/))
        pflags = substr(seg, 1, RLENGTH)
    }
    if (pflags !~ /OPERATOR/ && pflags !~ /"o"/) return

    # If class has any action -> exempt
    if (has_action) return

    printf "WARN  ext-writable-shape  %s:%d  %s: OPERATOR %s with no writing action" \
           " -- external oBIX write must use the child leaf .../%s/value (bare <real>, B826)" \
           " or add an OPERATOR action (B822)\n",
           FILE, lineno, pname, ptype, pname
  }
  ' "$f")
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    WARNED=1
  fi
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

[ "$STRICT" -eq 1 ] && [ "$WARNED" -eq 1 ] && exit 1
exit 0
