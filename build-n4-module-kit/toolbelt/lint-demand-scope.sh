#!/usr/bin/env bash
# lint-demand-scope.sh — demand-in-scope lint (Campaign 9 PR2, B820). [ev: retro campaign9-demand-scope]
# Usage: lint-demand-scope.sh [--strict] <java-src-dir>
#
# Within a control-decision method (one that writes an output/cmd/target/setBool driven by a
# numeric input), checks whether the method's PARAMETER list OR the enclosing class's FIELDS
# contain a demand-shaped input {demand*, *call*, enable, loopEnable, *count, BStatusBoolean in*}.
# If the method reads a PROCESS VARIABLE (suction/pressure/discharge/temp/cv/coil/head) in a
# comparison but has ZERO demand-shaped input in scope -> WARN ("pressure without demand").
#
# Row:  WARN  demand-in-scope  <file>:<line>  <method> reads <pv> with no demand-shaped input in scope
# Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2).
# §820.3: WARN only (statically-decidable absence), never a hard FAIL — a pure-modulator block
# driven by an upstream demand gate would false-positive on a hard FAIL.
# Advisory: bare *count (e.g. onCount) is the weakest demand signal; kept per B820 but the
# matched token is named in the row so a reviewer sees which term cleared it.
set -u
LC_ALL=C
export LC_ALL

STRICT=0
[ "${1:-}" = "--strict" ] && { STRICT=1; shift; }
[ $# -ge 1 ] || { printf 'usage: lint-demand-scope.sh [--strict] <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-demand-scope: not a directory: %s\n' "$SRC" >&2; exit 3; }

WARNED=0

# K20 / R2.5: a source dir with no *.java files is a configuration error -> exit 3.
# Never emit a silent 0 (C8 lesson; WP9b shape). Row contains ERROR + demand-in-scope.
_nj=$(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | wc -l)
if [ "$_nj" -eq 0 ]; then
  printf 'ERROR  demand-in-scope  %s  no *.java sources found\n' "$SRC"
  exit 3
fi

# shellcheck disable=SC2016  # awk program in single quotes — no shell expansion intended
while IFS= read -r f; do
  out=$(awk -v FILE="$f" '
  { lines[NR] = $0 }

  END {
    if (NR == 0) exit

    # ---------------------------------------------------------------------------
    # Pass 1: collect class-level demand-shaped field names/types.
    # A field is at class scope when brace depth == 1 (inside class body, outside
    # any method), has no "(" (not a method/constructor), and ends with ";".
    # Same-method-body discipline (D4/lint-timers-ext): we only scan class-level
    # lines here — method-local variables are scoped to the method params below.
    # ---------------------------------------------------------------------------
    depth = 0
    class_has_demand = 0
    for (i = 1; i <= NR; i++) {
      ln = lines[i]
      gsub(/\/\/.*$/, "", ln)          # strip line comments
      before = depth
      o = ln; gsub(/[^{]/, "", o); depth += length(o)
      c = ln; gsub(/[^}]/, "", c); depth -= length(c)

      # Field at class body (before==1): no method call "(" in the line, has ";"
      if (before == 1 && index(ln, "(") == 0 && index(ln, ";") > 0 &&
          ln !~ /\bclass\b/ && ln !~ /\binterface\b/) {
        lo = tolower(ln)
        # demand* > *call* > enable/loopEnable > *count  (B820 §820.2 order)
        if (lo ~ /demand/ || lo ~ /call/ || lo ~ /enable/ || lo ~ /count/) {
          class_has_demand = 1
        }
      }
    }

    # ---------------------------------------------------------------------------
    # Pass 2: find methods and apply rules 1-4 (B820 §820.2).
    # Only methods at class scope (brace depth 1 before the opening "{") are
    # examined; inner-class methods sit at depth > 1 and have their own context.
    # ---------------------------------------------------------------------------
    PV = "suction pressure discharge temp cv coil head"
    n_pv = split(PV, pvs)

    depth = 0
    i = 1
    while (i <= NR) {
      ln = lines[i]
      gsub(/\/\/.*$/, "", ln)
      before = depth
      o = ln; gsub(/[^{]/, "", o); opens = length(o)
      c = ln; gsub(/[^}]/, "", c); closes = length(c)
      depth += opens - closes

      # Method signature at class scope:
      #   before==1  — we are at class-body level (not inside a method)
      #   opens>0    — this line opens a block (the method body)
      #   has "(" and ")"  — method parameter list
      #   not a class/interface/enum/static-initializer declaration
      if (before == 1 && opens > 0 &&
          index(ln, "(") > 0 && index(ln, ")") > 0 &&
          ln !~ /\bclass\b/ && ln !~ /\binterface\b/ && ln !~ /\benum\b/ &&
          ln !~ /\bstatic[[:space:]]*\{/ && ln !~ /^[[:space:]]*\/\//) {

        # --- Extract method name (last identifier before the first "(") ---
        tmp = ln; sub(/\(.*/, "", tmp)
        meth_name = "unknown"
        if (match(tmp, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*$/)) {
          meth_name = substr(tmp, RSTART, RLENGTH)
          gsub(/[[:space:]]/, "", meth_name)
        }

        # --- Extract parameter list (from first "(" to first ")") ---
        p1 = index(ln, "(")
        after_p = substr(ln, p1 + 1)
        p2 = index(after_p, ")")
        params = (p2 > 0) ? substr(after_p, 1, p2 - 1) : after_p

        # --- Collect method body: from this sig line until depth returns to 1 ---
        body = ln
        cur = depth
        j = i + 1
        while (j <= NR && cur > 1) {
          bln = lines[j]
          gsub(/\/\/.*$/, "", bln)
          body = body "\n" bln
          bo = bln; gsub(/[^{]/, "", bo); cur += length(bo)
          bc = bln; gsub(/[^}]/, "", bc); cur -= length(bc)
          j++
        }

        # --- Rule 1: control-decision method? ---
        # Writes an output slot / target / cmd or returns a computed decision.
        is_ctrl = 0
        if (body ~ /target[[:space:]]*[+\-]?=/ || body ~ /target[+\-][+\-]/ ||
            body ~ /cmd\[/                       || body ~ /setBool\(/ ||
            body ~ /\.setValue\(/               || body ~ /set[A-Z][A-Za-z0-9]*\(/ ||
            body ~ /\breturn[^;]*[<>]/) {
          is_ctrl = 1
        }

        if (is_ctrl) {
          # --- Rule 2: reads a process variable in a comparison? ---
          # PV identifier (optionally suffixed with word chars) before "<" or ">"
          # with at least one space, OR "<"/">" with optional "=" then space before PV.
          has_pv = 0; pv_id = ""
          lo_body = tolower(body)
          for (p = 1; p <= n_pv; p++) {
            pv = pvs[p]
            if (lo_body ~ pv "[a-z0-9_.]*[[:space:]][<>]" ||
                lo_body ~ "[<>][=]?[[:space:]][a-z0-9_.]*" pv) {
              has_pv = 1; pv_id = pv; break
            }
          }

          if (has_pv) {
            # --- Rule 3: demand-shaped input in scope? ---
            # Scope = method parameters UNION enclosing-class fields (DS3).
            # Order: demand* > *call* > enable/loopEnable > *count (B820 §820.2).
            has_demand = class_has_demand
            if (!has_demand) {
              lo_params = tolower(params)
              if (lo_params ~ /demand/ || lo_params ~ /call/ ||
                  lo_params ~ /enable/ || lo_params ~ /count/) {
                has_demand = 1
              }
              # BStatusBoolean parameter named in[A-Z]... (B820 §820.2)
              if (!has_demand && params ~ /BStatusBoolean[[:space:]]+in[A-Z]/) {
                has_demand = 1
              }
            }

            # --- Rule 4: no demand in scope -> WARN ---
            if (!has_demand) {
              printf "WARN  demand-in-scope  %s:%d  %s reads %s with no demand-shaped input in scope\n",
                FILE, i, meth_name, pv_id
            }
          }
        }
      }

      i++
    }
  }
  ' "$f")
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    WARNED=1
  fi
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

[ "$STRICT" -eq 1 ] && [ "$WARNED" -eq 1 ] && exit 1
exit 0
