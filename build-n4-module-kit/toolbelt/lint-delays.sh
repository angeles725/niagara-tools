#!/usr/bin/env bash
# lint-delays.sh — Clock.schedule/schedulePeriodically delay-floor lint (Campaign 8 PR1).
#
# Detects call sites where the delay or period argument has no strictly-positive (>0)
# floor, which causes Niagara's EngineManager to throw "time <= 0" at runtime and
# silently drop the schedule.
#
# Real defect: ColdRoomPan BDefrostController.armTrigger used Math.max(delayMs, 0L) —
# when overdue, delayMs=0 -> d=0 -> "time <= 0" swallowed; defrost died silently 5x on
# PANCCADIA 2026-09-02. lint-timers.sh cannot catch this class of bug.
# [ev: corpus B801]
#
# Usage:  lint-delays.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits FAIL/WARN rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#
# Row format:  FAIL|WARN  lint-delays  <file>:<line>  <reason>
# Exits:       0 no FAIL · 1 any FAIL · 3 usage/env
#
# Dot-directories (e.g. .deploy-baseline/) are excluded from the file walk so
# baseline snapshots under an artifact dir are never double-counted (D9b).
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

FAILED=0
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

[ $# -ge 1 ] || { printf 'usage: lint-delays.sh <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-delays: not a directory: %s\n' "$SRC" >&2; exit 3; }

# ---------------------------------------------------------------------------
# D2b — Cross-file positive-floor helper registry.
# Scan every .java under SRC (dot-dirs excluded) for static long <fn>( methods.
# A method is a proven strictly-positive floor when its body contains any of:
#   Math.max(…, N) with N≥1 · >=1 comparison · <1?1:x shape · return 1L branch.
# Result: POSITIVE_HELPERS — space-separated function names (no class prefix).
# ---------------------------------------------------------------------------
_PH_FILE="$_TMP/ph.txt"
# Multi-file awk: FNR==1 resets state at each new file so nothing bleeds across.
# shellcheck disable=SC2016  # awk program intentionally in single quotes (no shell expansion needed)
find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort | \
  xargs awk '
  BEGIN { fn = ""; body = ""; depth = 0; in_meth = 0 }
  FNR == 1 { fn = ""; body = ""; depth = 0; in_meth = 0 }
  function adjust(line,   o, c) {
    o = line; gsub(/[^{]/, "", o); depth += length(o)
    c = line; gsub(/[^}]/, "", c); depth -= length(c)
  }
  function is_floor(b) {
    if (b ~ /Math\.max\([^)]+,[[:space:]]*[1-9]/) return 1
    if (b ~ />=[[:space:]]*1[^0-9]/) return 1
    if (b ~ /<[[:space:]]*1[L]?[[:space:]]*\?[[:space:]]*1/) return 1
    if (b ~ /return[[:space:]]+1[L]?[[:space:]]*;/) return 1
    return 0
  }
  !in_meth && /static[[:space:]]+(final[[:space:]]+)?long[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/ {
    match($0, /static[[:space:]]+(final[[:space:]]+)?long[[:space:]]+/)
    rest = substr($0, RSTART + RLENGTH)
    match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)
    fn = substr(rest, 1, RLENGTH)
    in_meth = 1; body = $0; depth = 0; adjust($0)
    if (depth == 0 && index($0, "{") > 0) {
      if (is_floor(body)) print fn
      fn = ""; body = ""; in_meth = 0
    }
    next
  }
  in_meth {
    body = body "\n" $0; adjust($0)
    if (depth <= 0) {
      if (is_floor(body)) print fn
      fn = ""; body = ""; in_meth = 0; depth = 0
    }
  }
  ' 2>/dev/null | sort -u > "$_PH_FILE"
POSITIVE_HELPERS=$(tr '\n' ' ' < "$_PH_FILE")

# ---------------------------------------------------------------------------
# Process each Java file — all analysis in one awk pass (store-then-END).
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  result=$(awk -v FILE="$f" -v POSITIVE_HELPERS="$POSITIVE_HELPERS" '
  { lines[NR] = $0 }

  # Return ms value for a BRelTime literal expression; -1 if not a simple literal.
  function literal_ms(text,    seg) {
    if (match(text, /BRelTime\.makeSeconds\([0-9]+\)/)) {
      seg = substr(text, RSTART)
      sub(/BRelTime\.makeSeconds\(/, "", seg); sub(/\).*/, "", seg)
      return seg * 1000
    }
    if (match(text, /BRelTime\.make\([0-9]+\)/)) {
      seg = substr(text, RSTART)
      sub(/BRelTime\.make\(/, "", seg); sub(/\).*/, "", seg)
      return seg + 0
    }
    return -1
  }

  # Extract second argument from a Clock.schedule*( argument string.
  # Input: everything after the opening "(" of the call.
  # Skips the first top-level argument ("this"), returns the second.
  function skip_arg(s,    depth, i, c, out, past_first) {
    depth = 0; past_first = 0; out = ""
    for (i = 1; i <= length(s); i++) {
      c = substr(s, i, 1)
      if      (c == "(") depth++
      else if (c == ")") { if (depth == 0) break; depth-- }
      else if (c == "," && depth == 0) {
        if (!past_first) { past_first = 1; continue }
        else break
      }
      if (past_first) out = out c
    }
    return out
  }

  # Emit a result row.
  function emit(level, lnum, detail) {
    printf "%-4s  %-14s  %s:%d  %s\n", level, "lint-delays", FILE, lnum, detail
  }

  # D2c — same-method positivity guard detection.
  # Scans backward from lnum for a guard on expr (Java identifier or getter name)
  # within the same brace scope (up to 80 lines back).
  # Accepted guard patterns:
  #   expr > 0  |  expr >= 1  |  expr == 0  (zero-branch takes other path)
  #   expr.getMillis() > 0  |  expr.getMillis() >= 1  |  expr.getMillis() == 0
  # Returns the guard line number if found, 0 otherwise.
  function find_guard(lnum, expr,    j, ln2, re_gt, re_ge, re_eq, re_gt_ms, re_ge_ms, re_eq_ms) {
    if (expr == "") return 0
    # Bare-variable patterns: `expr > 0`, `expr >= 1`, `expr == 0`
    re_gt  = expr "[[:space:]]*>[[:space:]]*0"
    re_ge  = expr "[[:space:]]*>=[[:space:]]*1"
    re_eq  = expr "[[:space:]]*==[[:space:]]*0"
    # Getter patterns: `expr().getMillis() > 0`, `>= 1`, `== 0`
    # The `()` suffix of the getter call must appear before `.getMillis()`.
    re_gt_ms = expr "\\(\\)\\.getMillis\\(\\)[[:space:]]*>[[:space:]]*0"
    re_ge_ms = expr "\\(\\)\\.getMillis\\(\\)[[:space:]]*>=[[:space:]]*1"
    re_eq_ms = expr "\\(\\)\\.getMillis\\(\\)[[:space:]]*==[[:space:]]*0"
    for (j = lnum - 1; j >= 1 && j >= lnum - 80; j--) {
      ln2 = lines[j]
      if (index(ln2, expr) == 0) continue
      # Strip single-line comments before pattern matching to avoid false matches
      # on comment text that mentions the expression but is not a real guard.
      sub(/[[:space:]]*\/\/.*$/, "", ln2)
      if (index(ln2, expr) == 0) continue
      if (ln2 ~ re_gt || ln2 ~ re_ge || ln2 ~ re_eq ||
          ln2 ~ re_gt_ms || ln2 ~ re_ge_ms || ln2 ~ re_eq_ms) return j
    }
    return 0
  }

  # Classify the BRelTime argument of a Clock.schedule* call.
  function classify(lnum, arg,    v, k, seg, rest, i, c, depth, found,
                    kpart, varname, slotname, gname, minval, j, ln2,
                    inner, fn_h, seg_h, parts_h, n_h, pos, after_var, rhs,
                    guard_line) {
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", arg)

    # 1. Inline BRelTime literal: BRelTime.make(N) or BRelTime.makeSeconds(N)
    v = literal_ms(arg)
    if (v >= 0) {
      if (v == 0) emit("FAIL", lnum, "literal-zero  BRelTime(0) — Niagara rejects delay/period <= 0")
      return
    }

    # 2. Inline Math.max inside BRelTime.make: BRelTime.make(Math.max(x, kL))
    if (match(arg, /BRelTime\.make\(Math\.max\(/)) {
      seg = substr(arg, RSTART + RLENGTH)
      # skip first arg of Math.max, find top-level comma
      depth = 0; found = 0
      for (i = 1; i <= length(seg); i++) {
        c = substr(seg, i, 1)
        if      (c == "(") depth++
        else if (c == ")") { if (depth == 0) break; depth-- }
        else if (c == "," && depth == 0) { found = i; break }
      }
      if (found) {
        kpart = substr(seg, found + 1)
        gsub(/^[[:space:]]+/, "", kpart)
        if (match(kpart, /^[0-9]+/)) {
          k = substr(kpart, 1, RLENGTH) + 0
          if (k <= 0) emit("FAIL", lnum, "zero-floor  Math.max floor=" k " (<=0) — use Math.max(x,1L)")
          return
        }
      }
      emit("FAIL", lnum, "zero-floor  cannot determine Math.max floor in: " arg)
      return
    }

    # 2.5 D2b — BRelTime.make(<method-call>): resolve helper one level across files.
    # Handles: BRelTime.make(Class.fn(...)) and BRelTime.make(fn(...)).
    if (match(arg, /^BRelTime\.make\(/)) {
      inner = substr(arg, RSTART + RLENGTH)
      gsub(/^[[:space:]]+/, "", inner)
      fn_h = ""
      # Qualified call: Class.fn(
      if (match(inner, /^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
        seg_h = substr(inner, 1, RLENGTH); sub(/[[:space:]]*\($/, "", seg_h)
        n_h = split(seg_h, parts_h, /\./)
        fn_h = parts_h[n_h]
      # Unqualified call: fn(  (but not a bare identifier, so must be followed by paren)
      } else if (match(inner, /^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
        fn_h = substr(inner, 1, RLENGTH); sub(/[[:space:]]*\($/, "", fn_h)
      }
      if (fn_h != "") {
        if (index(" " POSITIVE_HELPERS " ", " " fn_h " ") > 0) return  # proven floor
        # D2c: check for same-method positivity guard on the helper call
        guard_line = find_guard(lnum, fn_h)
        if (guard_line > 0) { emit("PASS", lnum, "guarded at :" guard_line); return }
        emit("FAIL", lnum, "zero-floor  helper " fn_h " has no visible strictly-positive floor")
        return
      }
      # inner is a plain identifier or expression — fall through to case #3
    }

    # 3. BRelTime.make(VAR) — look up local variable binding (backwards, same method)
    if (match(arg, /BRelTime\.make\([^)]+\)/)) {
      seg = substr(arg, RSTART)
      sub(/BRelTime\.make\(/, "", seg); sub(/\).*/, "", seg)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", seg)
      varname = seg
      # D2c: check for same-method positivity guard before backward scan
      guard_line = find_guard(lnum, varname)
      if (guard_line > 0) { emit("PASS", lnum, "guarded at :" guard_line); return }
      for (j = lnum - 1; j >= 1 && j >= lnum - 50; j--) {
        ln2 = lines[j]
        if (index(ln2, varname) > 0) {
          # Math.max binding: long varname = Math.max(x, kL)
          if (match(ln2, /Math\.max\([^,]+,[[:space:]]*/)) {
            rest = substr(ln2, RSTART + RLENGTH)
            if (match(rest, /^[0-9]+/)) {
              k = substr(rest, 1, RLENGTH) + 0
              if (k <= 0) emit("FAIL", lnum, "zero-floor  local " varname "=Math.max(.,k) k=" k " (<=0)")
              return
            }
          }
          # Literal BRelTime assignment
          v = literal_ms(ln2)
          if (v >= 0) {
            if (v == 0) emit("FAIL", lnum, "literal-zero  local " varname "=BRelTime(0)")
            return
          }
          # D2b: helper call assignment — varname = Class.fn(...) or varname = fn(...)
          pos = index(ln2, varname)
          if (pos > 0) {
            after_var = substr(ln2, pos + length(varname))
            if (match(after_var, /^[[:space:]]*=[^=]/)) {
              rhs = substr(after_var, RSTART + RLENGTH)
              gsub(/^[[:space:]]+/, "", rhs)
              fn_h = ""
              if (match(rhs, /^[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
                seg_h = substr(rhs, 1, RLENGTH); sub(/[[:space:]]*\($/, "", seg_h)
                n_h = split(seg_h, parts_h, /\./)
                fn_h = parts_h[n_h]
              } else if (match(rhs, /^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
                fn_h = substr(rhs, 1, RLENGTH); sub(/[[:space:]]*\($/, "", fn_h)
              }
              if (fn_h != "") {
                if (index(" " POSITIVE_HELPERS " ", " " fn_h " ") > 0) return
                emit("FAIL", lnum, "zero-floor  helper " fn_h " has no visible strictly-positive floor")
                return
              }
            }
          }
        }
      }
      emit("FAIL", lnum, "unfloored  cannot verify floor of variable: " varname)
      return
    }

    # 4. Slot getter: getXxx()
    if (match(arg, /^get[A-Za-z0-9_]+[[:space:]]*\([[:space:]]*\)/)) {
      gname = substr(arg, 1, RLENGTH)
      sub(/[[:space:]]*\(.*/, "", gname)
      # D2c: check for same-method positivity guard on the getter call
      guard_line = find_guard(lnum, gname)
      if (guard_line > 0) { emit("PASS", lnum, "guarded at :" guard_line); return }
      if (gname in getter_slot) {
        slotname = getter_slot[gname]
        if (slotname in prop_min) {
          minval = prop_min[slotname]
          if (minval >= 1) {
            emit("WARN", lnum, "facet-floor  slot " slotname " facet MIN=" minval "s (advisory)")
          } else {
            emit("FAIL", lnum, "facet-min-zero  slot " slotname " facet MIN=0 — schedule can receive 0")
          }
        } else {
          emit("FAIL", lnum, "facet-min-zero  slot " slotname " has no MIN facet — can schedule 0")
        }
      } else {
        emit("FAIL", lnum, "unfloored  unknown getter " gname " — cannot verify floor")
      }
      return
    }

    # 5. Known static final BRelTime constant
    if (arg in const_ms) {
      v = const_ms[arg]
      if (v <= 0) emit("FAIL", lnum, "literal-zero  constant " arg "=" v "ms (<=0)")
      return
    }

    # 6. Unresolved
    emit("FAIL", lnum, "unfloored  cannot verify floor: " arg)
  }

  END {
    # ---- Pass 1: collect constants, @NiagaraProperty facets, getter->slot ----
    in_prop = 0; prop_buf = ""; prop_name = ""

    for (i = 1; i <= NR; i++) {
      ln = lines[i]

      # static final BRelTime NAME = BRelTime.make*(N) on one line
      if (match(ln, /static[[:space:]]+final[[:space:]]+BRelTime[[:space:]]+/)) {
        rest = substr(ln, RSTART + RLENGTH)
        if (match(rest, /^[A-Za-z_][A-Za-z0-9_]*/)) {
          cname = substr(rest, 1, RLENGTH)
          v = literal_ms(ln)
          if (v >= 0) const_ms[cname] = v
        }
      }

      # slotomatic form: static final Property <slot> = newProperty(..., BFacets.make(BFacets.MIN, ...))
      if (match(ln, /static[[:space:]]+final[[:space:]]+Property[[:space:]]+/)) {
        rest = substr(ln, RSTART + RLENGTH)
        if (match(rest, /^[a-z][A-Za-z0-9_]*/)) {
          pname = substr(rest, 1, RLENGTH)
          if (match(ln, /BFacets\.MIN,[[:space:]]*BRelTime\.makeSeconds\([0-9]+\)/)) {
            seg = substr(ln, RSTART)
            sub(/BFacets\.MIN,[[:space:]]*BRelTime\.makeSeconds\(/, "", seg)
            sub(/\).*/, "", seg)
            prop_min[pname] = seg + 0
          } else if (match(ln, /BFacets\.MIN,[[:space:]]*BRelTime\.make\([0-9]+\)/)) {
            seg = substr(ln, RSTART)
            sub(/BFacets\.MIN,[[:space:]]*BRelTime\.make\(/, "", seg)
            sub(/\).*/, "", seg)
            prop_min[pname] = seg + 0
          }
        }
      }

      # @NiagaraProperty — multi-line state machine (paren-balance based)
      if (!in_prop) {
        if (index(ln, "@NiagaraProperty") > 0) {
          in_prop = 1; prop_buf = ln; prop_name = ""
        }
      } else {
        prop_buf = prop_buf " " ln
      }

      if (in_prop) {
        # Extract name if not yet found
        if (prop_name == "" && match(prop_buf, /name[[:space:]]*=[[:space:]]*"[^"]*"/)) {
          seg = substr(prop_buf, RSTART)
          sub(/name[[:space:]]*=[[:space:]]*"/, "", seg); sub(/".*/, "", seg)
          prop_name = seg
        }
        # Check paren balance
        depth = 0; tmp = prop_buf
        n = length(tmp)
        for (ci = 1; ci <= n; ci++) {
          c = substr(tmp, ci, 1)
          if      (c == "(") depth++
          else if (c == ")") depth--
        }
        if (depth <= 0 && index(prop_buf, "@NiagaraProperty") > 0) {
          # Property block complete — extract MIN facet value (in ms)
          if (prop_name != "") {
            if (match(prop_buf, /BFacets\.MIN,[[:space:]]*BRelTime\.makeSeconds\([0-9]+\)/)) {
              seg = substr(prop_buf, RSTART)
              sub(/BFacets\.MIN,[[:space:]]*BRelTime\.makeSeconds\(/, "", seg)
              sub(/\).*/, "", seg)
              prop_min[prop_name] = seg + 0
            } else if (match(prop_buf, /BFacets\.MIN,[[:space:]]*BRelTime\.make\([0-9]+\)/)) {
              seg = substr(prop_buf, RSTART)
              sub(/BFacets\.MIN,[[:space:]]*BRelTime\.make\(/, "", seg)
              sub(/\).*/, "", seg)
              prop_min[prop_name] = seg + 0
            } else if (index(prop_buf, "BFacets.MIN") > 0) {
              prop_min[prop_name] = -1  # MIN present but unreadable value
            }
          }
          in_prop = 0; prop_buf = ""; prop_name = ""
        }
      }

      # Getter -> slot: public BRelTime getXxx() { return (BRelTime) get("slot"); }
      if (!in_prop && match(ln, /public[[:space:]]+BRelTime[[:space:]]+get[A-Za-z0-9_]+[[:space:]]*\(/)) {
        seg = substr(ln, RSTART)
        sub(/public[[:space:]]+BRelTime[[:space:]]+/, "", seg)
        if (match(seg, /^get[A-Za-z0-9_]+/)) {
          gname = substr(seg, 1, RLENGTH)
          for (j = i; j <= i + 3 && j <= NR; j++) {
            if (match(lines[j], /get\("[^"]+"\)/)) {
              slotline = substr(lines[j], RSTART)
              sub(/get\("/, "", slotline); sub(/".*/, "", slotline)
              getter_slot[gname] = slotline; break
            } else if (match(lines[j], /get\([a-z][A-Za-z0-9_]*\)/)) {
              slotline = substr(lines[j], RSTART)
              sub(/get\(/, "", slotline); sub(/\).*/, "", slotline)
              getter_slot[gname] = slotline; break
            }
          }
        }
      }
    }

    # ---- Pass 2: find Clock.schedule* calls and classify delay argument ----
    for (i = 1; i <= NR; i++) {
      ln = lines[i]
      if (index(ln, "Clock.schedule") == 0) continue
      if (!match(ln, /Clock\.schedule(Periodically)?\(/)) continue

      call_rest = substr(ln, RSTART + RLENGTH)
      arg2 = skip_arg(call_rest)
      classify(i, arg2)
    }
  }
  ' "$f")

  if [ -n "$result" ]; then
    printf '%s\n' "$result"
    printf '%s\n' "$result" | grep -q '^FAIL' && FAILED=1
  fi
done < <(find "$SRC" -type d -name '.*' -prune -o -name '*.java' -print | sort)

[ "$FAILED" -eq 0 ] || exit 1
exit 0
