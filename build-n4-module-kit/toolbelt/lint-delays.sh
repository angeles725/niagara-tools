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
# The delay (second argument) is classified in this order:
#   literal       BRelTime.make(N) / make<Unit>(N): FAIL literal-zero when 0, else clean.
#   Math.max      BRelTime.make(Math.max(x, k)): FAIL zero-floor when k <= 0 or k is not a literal.
#   helper (D2b)  BRelTime.make([Class.]fn(...)): clean when fn is a `static [final] long fn(` method, in any
#                 file under the root, whose body holds Math.max(.., N>=1), `>= 1`, `< 1 ? 1` or `return 1;`.
#   variable      BRelTime.make(v): the binding of v up to 50 lines back (Math.max floor, BRelTime literal or
#                 a helper call), else FAIL unfloored.
#   getter        getXxx(): the slot it reads (get("slot") or get(slot) within 3 lines) and that slot's
#                 BFacets.MIN: WARN facet-floor when >= 1 ms, else FAIL facet-min-zero.
#   constant      a `static final BRelTime NAME = <literal>` of the same file: FAIL literal-zero when 0.
#   anything else FAIL unfloored.
# A same-method guard (D2c) up to 80 lines back on the helper, variable or getter — `x > 0`, `x >= 1`,
# `x == 0`, or the same on `x().getMillis()` — prints a PASS "guarded at :N" row instead.
#
# Implementation: rule package go/internal/lint/rules/delays on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-delays <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-delays.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir> (only the first argument is read; no flag is parsed).
#
# Row format:  FAIL|WARN|PASS  lint-delays     <file>:<line>  <reason>   (<line> = the Clock.schedule*( line)
# Exits:       0 no FAIL · 1 any FAIL · 3 usage/env, an unscannable source file or a sub-directory find
#              cannot enter
#
# Dot-directories (e.g. .deploy-baseline/) and *-worktrees sub-directories are excluded from the file walk
# so baseline snapshots under an artifact dir are never double-counted (D9b).
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# Mutation: LD1 -- removes delay-floor check, allowing Math.max(x,0L) pattern to pass instead of FAIL
# Mutation: LD13 -- drop Minutes from the facet_min unit alternation -> makeMinutes MIN reads as unreadable, FAIL
# Mutation: LD16 -- stop accumulating continuation lines -> a multi-line Clock.schedule( FAILs unfloored
#
# Units: BRelTime.make(ms) and makeSeconds/makeMinutes/makeHours/makeDays(N) literals are read in
# both delay arguments and BFacets.MIN facets; a Clock.schedule*( call split over several lines is
# accumulated to its closing paren (up to 12 lines; a call still open past that window is a FAIL
# unclosed-call row, never a silent truncation). The facet floor is carried and printed in ms for every
# factory, and the // comment cut (go/internal/javasrc CutComments) skips string/char literals
# ("http://..." is not a comment). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ1]
# Mutation: LD21 -- facet_min back to seconds for the make<Unit> family -> LD21 detail is wrong
# Mutation: LD22 -- drop the unclosed-call report -> LD22 gets a blank "cannot verify floor:" row
# Mutation: LD23 -- naive // cut that ignores string literals -> LD23 loses its guard and FAILs
# Mutation: LD24 -- no block-comment skip in CutComments -> an apostrophe in /* don't */ keeps the trailing // text as a guard
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-delays: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
# shellcheck disable=SC2093  # the old bash body below no longer runs; it is removed in the next commits
exec "$BIN" lint-delays "$@"
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/scan-files.sh"

FAILED=0
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

[ $# -ge 1 ] || { printf 'usage: lint-delays.sh <java-src-dir>\n' >&2; exit 3; }
SRC="$1"
[ -d "$SRC" ] || { printf 'lint-delays: not a directory: %s\n' "$SRC" >&2; exit 3; }
# One fail-closed walk (lib/scan-files.sh): a sub-directory find cannot enter is exit 3, never a shorter list.
scan_files_or_exit lint-delays 3 "$_TMP/files" "$SRC" -name '*.java'

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
  ' 2>/dev/null < "$_TMP/files" | sort -u > "$_PH_FILE"
POSITIVE_HELPERS=$(tr '\n' ' ' < "$_PH_FILE")

# ---------------------------------------------------------------------------
# Process each Java file — all analysis in one awk pass (store-then-END).
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  result=$(awk -v FILE="$f" -v POSITIVE_HELPERS="$POSITIVE_HELPERS" '
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
            emit("WARN", lnum, "facet-floor  slot " slotname " facet MIN=" minval "ms (advisory)")
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
          v = facet_min(ln)
          if (v >= 0) prop_min[pname] = v
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
            v = facet_min(prop_buf)   # -1 = MIN present but unreadable value
            if (v != -2) prop_min[prop_name] = v
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
    MAX_CONT = 12   # continuation-line window of one Clock.schedule*( call
    for (i = 1; i <= NR; i++) {
      ln = lines[i]
      if (index(ln, "Clock.schedule") == 0) continue
      if (!match(ln, /Clock\.schedule(Periodically)?\(/)) continue

      call_rest = substr(ln, RSTART + RLENGTH)
      call_rest = strip_comment(call_rest)
      # A call split over lines: accumulate continuation lines (comments stripped) up to the
      # closing paren of the call, the same paren-balance technique as the @NiagaraProperty pass.
      # [ev: retro panccadia-restart-seq-comp-lockout-hours Δ1]
      for (j = i + 1; !call_closed(call_rest) && j <= NR && j <= i + MAX_CONT; j++) {
        nxt = strip_comment(lines[j])
        call_rest = call_rest " " nxt
      }
      # The window is bounded; a call still open past it is reported, never classified on a
      # truncated argument list. [polish-2026-10-02 P2a]
      if (!call_closed(call_rest)) {
        emit("FAIL", i, "unclosed-call  Clock.schedule( not closed within " MAX_CONT " lines -- cannot read the delay; wrap the call tighter")
        continue
      }
      arg2 = skip_arg(call_rest)
      classify(i, arg2)
    }
  }
  ' "$f")

  if [ -n "$result" ]; then
    printf '%s\n' "$result"
    printf '%s\n' "$result" | grep -q '^FAIL' && FAILED=1
  fi
done < "$_TMP/files"

[ "$FAILED" -eq 0 ] || exit 1
exit 0
