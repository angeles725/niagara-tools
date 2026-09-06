#!/usr/bin/env bash
# lint-ext-writable-shape.sh — ext-writable-shape lint (Campaign 9 PR10, B823/B826). [ev: retro campaign9-ext-writable-shape]
# Usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>
#
# A COMPLEX property (BStatusNumeric / BStatusBoolean / BStatusEnum) with Flags.OPERATOR and
# NO writing action on its declaring class is a hazardous external-write target:
#   - a conformant oBIX PUT is rejected ("Cannot translate" — the slot is not BIObixWritable)
#   - a hand-crafted wrapped-<obj> PUT silently writes DEFAULT (0.0 for numeric) on a 200 OK
#     when the "value" child is omitted (the live silent-zero hazard, B823 §823.2)
# The safe external-write paths are:
#   (a) write the oBIX child-leaf bare <real> at …/<slot>/value (B826-G2, LIVE-CONFIRMED)
#   (b) add an OPERATOR @NiagaraAction that writes the slot (B822 additive-action doctrine)
#
# Exemption rule (S22 / C10 D2b — per-slot writing-action check):
#   A slot X is exempt when:
#     (1) Some @NiagaraAction's do<Action>() body writes X via setX(, getX().setValue(, or
#         .set(X, — scoped only to do<Action> bodies (brace_depth>=2 guard); execute(),
#         changed(), and slotomatic-generated setters are excluded by construction.
#     (2) An @NiagaraAction name directly encodes the write: name matches set<X>/apply<X>/<x>Cmd
#         (B822 naming convention; EW3 positive pin).
#   Replacement for the C9 class-level has_action: an @NiagaraAction that does not write X
#   does NOT exempt X. [ev: corpus B831 §831.2; retro campaign10-ext-writable-per-slot]
#
# Row:  WARN  ext-writable-shape  <file>:<line>  <slot>: OPERATOR <type> with no writing action
#             — external oBIX write must use the child leaf …/<slot>/value (bare <real>, B826)
#             or add an OPERATOR action (B822)
# Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2).
# [ev: corpus B823 §823.2]  [ev: corpus B826]  [ev: kit types/logic-authoring.md:62-70]
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
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
  out=$(awk -v FILE="$f" "$MB_AWK"'
  { lines[NR] = $0 }

  END {
    if (NR == 0) exit

    # -------------------------------------------------------------------------
    # Comment strip (D6): blank // and /* */ content; line numbers preserved.
    # Runs over lines[] before every pass. A commented @NiagaraAction or
    # @NiagaraProperty must NOT trigger any pattern. [ev: retro campaign9-close lesson 21]
    # -------------------------------------------------------------------------
    in_bc = 0
    for (i = 1; i <= NR; i++) {
      ln = lines[i]; out = ""; j = 1
      while (j <= length(ln)) {
        if (in_bc) {
          if (substr(ln, j, 2) == "*/") { in_bc = 0; j += 2 }
          else j++
        } else {
          c2 = substr(ln, j, 2)
          if (c2 == "/*") { in_bc = 1; j += 2 }
          else if (c2 == "//") break
          else { out = out substr(ln, j, 1); j++ }
        }
      }
      slines[i] = out
    }

    # -------------------------------------------------------------------------
    # Pass 1: harvest @NiagaraAction names (paren-balanced join, ALL actions).
    # Replaces the C9 class-level has_action flag.
    # action_names{x}           — raw annotation name, for name-pattern exemption (EW3).
    # do_methods{"do"+cap1(x)}  — handler method names, for body-scan scope filter.
    # Handles single-line and multi-line annotation forms. [ev: D2b step 1]
    # [ev: client BCompressorControl.java:435-444 @ ff1b659]
    # -------------------------------------------------------------------------
    in_ann = 0; buf = ""; depth = 0
    for (i = 1; i <= NR; i++) {
      ln = slines[i]
      if (!in_ann) {
        if (index(ln, "@NiagaraAction") > 0) {
          in_ann = 1; buf = ln; depth = 0
          for (ci = 1; ci <= length(ln); ci++) {
            c = substr(ln, ci, 1)
            if (c == "(") depth++
            else if (c == ")") {
              depth--
              if (depth <= 0 && index(buf, "(") > 0) {
                _harvest_action(buf); in_ann = 0; buf = ""; break
              }
            }
          }
          if (in_ann && depth <= 0) { _harvest_action(buf); in_ann = 0; buf = "" }
        }
      } else {
        buf = buf " " ln
        for (ci = 1; ci <= length(ln); ci++) {
          c = substr(ln, ci, 1)
          if (c == "(") depth++
          else if (c == ")") {
            depth--
            if (depth <= 0) { _harvest_action(buf); in_ann = 0; buf = ""; break }
          }
        }
      }
    }

    # -------------------------------------------------------------------------
    # Pass 2: shared fragment (PEAK depth) — do_method bodies via mb_parse (D1f).
    # execute(), changed(), and slotomatic-generated setters fall out by construction
    # — none is named do<Action>. [ev: D1f; lib/method-boundary.sh mb_parse]
    # -------------------------------------------------------------------------
    if (length(do_methods) > 0) {
      _n = mb_parse(slines, NR, ms, me, mn)
      for (k = 0; k < _n; k++) {
        if (!(mn[k] in do_methods)) continue
        body = ""
        for (bi = ms[k]; bi <= me[k]; bi++) body = body " " slines[bi]
        _scan_writes(body)
      }
    }

    # -------------------------------------------------------------------------
    # Pass 3: collect @NiagaraProperty annotations (paren-balanced multi-line
    # join as in C8 D9b / bog-audit; handles nested @Facet(...) parens).
    # For each: extract name, type, flags; emit WARN when:
    #   type ~ BStatus(Numeric|Boolean|Enum) (complex)  AND
    #   flags contain OPERATOR                           AND
    #   slot NOT exempt (body-check + name-pattern check)
    # [ev: D2b step 4; D2c; D2d; D2e]
    # -------------------------------------------------------------------------
    in_prop = 0; buf = ""; depth = 0; prop_first = 0; prop_line = 0

    for (i = 1; i <= NR; i++) {
      ln = slines[i]

      if (!in_prop) {
        if (index(ln, "@NiagaraProperty") > 0 && index(ln, "(") > 0) {
          in_prop = 1; buf = ln; depth = 0; prop_first = 1; prop_line = i
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

  # Harvest one @NiagaraAction buffer: extract name="x", store in action_names{x}
  # and compute do_methods{"do"+cap1(x)}. [ev: D2b step 1+2; B831-G1]
  function _harvest_action(buf,    aname, seg) {
    aname = ""
    if (match(buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
      seg = substr(buf, RSTART)
      sub(/name[[:space:]]*=[[:space:]]*"/, "", seg)
      sub(/".*/, "", seg)
      aname = seg
    }
    if (aname == "") return
    action_names[aname] = 1
    # Resolve: action x -> "do" + toupper(first letter) + rest. [ev: D2b step 2; B831-G1]
    do_methods["do" toupper(substr(aname, 1, 1)) substr(aname, 2)] = 1
  }

  # Scan a do<Action> body for writes; add found slot names to exempt_slot{}.
  # Pattern 1: setX(           — standard setter (X = slot with capitalized first letter)
  # Pattern 2: getX().setValue( — BSimple/BComplex setValue path
  # Pattern 3: .set(prop,      — property-constant form (slotomatic generated region)
  # [ev: D2b step 4]
  function _scan_writes(body,    tmp, frag, slot) {
    tmp = body
    while (match(tmp, /set[A-Z][A-Za-z0-9_]*\(/)) {
      frag = substr(tmp, RSTART, RLENGTH)
      sub(/^set/, "", frag); sub(/\($/, "", frag)
      slot = tolower(substr(frag, 1, 1)) substr(frag, 2)
      exempt_slot[slot] = 1
      tmp = substr(tmp, RSTART + RLENGTH)
    }
    tmp = body
    while (match(tmp, /get[A-Z][A-Za-z0-9_]*\(\)\.setValue\(/)) {
      frag = substr(tmp, RSTART, RLENGTH)
      sub(/^get/, "", frag); sub(/\(\)\.setValue\($/, "", frag)
      slot = tolower(substr(frag, 1, 1)) substr(frag, 2)
      exempt_slot[slot] = 1
      tmp = substr(tmp, RSTART + RLENGTH)
    }
    tmp = body
    while (match(tmp, /\.set\([a-zA-Z_][a-zA-Z0-9_]*,/)) {
      frag = substr(tmp, RSTART, RLENGTH)
      sub(/^\.set\(/, "", frag); sub(/,$/, "", frag)
      exempt_slot[frag] = 1
      tmp = substr(tmp, RSTART + RLENGTH)
    }
  }

  function _check_prop(buf, lineno,    pname, ptype, pflags, seg, uc, an) {
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

    # Per-slot body-check exemption (D2b step 4; D2c):
    # slot X is exempt when some do<Action> body was found to write X.
    if (pname in exempt_slot) return

    # Name-pattern exemption (B822 / EW3 compatibility):
    # An @NiagaraAction whose name encodes the write convention — set<X>/apply<X>/<x>Cmd
    # — is itself a writing action without requiring the method body to be present.
    # [ev: B822 additive-action doctrine]
    uc = toupper(substr(pname, 1, 1)) substr(pname, 2)
    for (an in action_names) {
      if (an == "set" uc || an == "apply" uc || an == pname "Cmd") return
    }

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
