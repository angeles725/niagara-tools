#!/usr/bin/env bash
# lint-timers.sh — timer-ticket conformance lint for Niagara N4 Java modules (Campaign 6 PR8).
#
# Detects timer lifecycle defects in module Java source files:
#
#   timer-ticket      A class that owns a Clock.Ticket (field declaration or
#                     Clock.schedule*() call) but its stopped() override does not
#                     cancel the ticket — the timer leaks on station stop.
#                     [ev: corpus B787]
#
#   discarded-ticket  A Clock.schedule*() call whose return value is not captured
#                     — the Clock.Ticket is immediately lost, no way to cancel it.
#
#   companion-flag    A boolean/int flag assigned true beside a Clock.schedule*
#                     call that is not assigned false inside stopped() or started();
#                     a clear only in the expiry handler does not count — a
#                     stop/restart cycle keeps the object alive and the flag stuck.
#                     Real shape: CompPan BCompressorControl :1760/:1764 startingUp.
#                     [ev: corpus B801] [ev: corpus B812]
#
#   jdk-thread        A class extending a B* Niagara component/service that uses JDK
#                     concurrency (ScheduledExecutorService, Executors.*, new Thread)
#                     instead of Clock.schedule — JDK pools ignore station lifecycle
#                     and the station SecurityManager denies modifyThread to module code.
#                     Real shape: chihuahua BChiDashboardService :229/:305/:314.
#                     [ev: corpus B800 §800.3] [ev: corpus B806]
#
#   changed-sched     A Clock.schedule* call reachable from changed() or started()
#                     (directly or via one private callee, one level deep) without an
#                     isRunning() or Sys.atSteadyState() guard IN THE SCHEDULING BODY
#                     — a guard only in the caller does not protect the callee body.
#                     Real shape: ColdRoomPan BEvaporatorUnit changed()->applyRunCmd()->
#                     Clock.schedule (pre-fix); fix adds if(!Sys.atSteadyState())return
#                     inside applyRunCmd(). NotRunningException x6 on PANCCADIA logs.
#                     [ev: corpus B816]
#
# Usage: lint-timers.sh <java-root>
#   Scans all *.java files recursively under <java-root> (dot-dirs pruned).
#   Prints FAIL|PASS rows per owning class.
#   A class with no timers emits no FAIL (silently skipped).
#   Exit 0 (all PASS) or 1 (any FAIL).
#
# Row format: FAIL|PASS  <check>  <file>: <detail>
# Exit: 0 no FAIL · 1 any FAIL · 2 usage · 3 env
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u
# shellcheck source=lib/method-boundary.sh
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"

FAILED=0

row() {
  printf '%s  %s  %s\n' "$1" "$2" "$3"
  case "$1" in FAIL) FAILED=1 ;; esac
}

usage_exit() {
  printf 'usage: lint-timers.sh <java-root>\n' >&2
  exit 2
}

[ $# -eq 1 ] || usage_exit
JAVA_ROOT="$1"
[ -d "$JAVA_ROOT" ] || { printf 'lint-timers: not a directory: %s\n' "$JAVA_ROOT" >&2; exit 3; }

# ---------------------------------------------------------------------------
# Process each Java file
# ---------------------------------------------------------------------------
while IFS= read -r f; do

  # ---- check: discarded-ticket ------------------------------------------
  # A Clock.schedule*( call whose result is not assigned (= before) or returned.
  while IFS= read -r line; do
    # Does this line have a Clock.schedule*( call?
    printf '%s' "$line" | grep -qE 'Clock\.schedule(Periodically)?\(' || continue
    # Is the result assigned? (= somewhere before Clock.schedule on this line)
    printf '%s' "$line" | grep -qE '=.*Clock\.schedule(Periodically)?\(' && continue
    # Is the result returned?
    printf '%s' "$line" | grep -qE 'return[[:space:]]+Clock\.schedule(Periodically)?\(' && continue
    # Result is discarded.
    row FAIL "discarded-ticket" "$f: Clock.schedule result is discarded — assign it to a Clock.Ticket field"
  done < "$f"

  # ---- detect timer ownership -------------------------------------------
  # A file owns a timer if it declares a Clock.Ticket field OR calls Clock.schedule*().
  has_timer=0
  grep -qE 'Clock\.Ticket' "$f" 2>/dev/null && has_timer=1
  grep -qE 'Clock\.schedule(Periodically)?\(' "$f" 2>/dev/null && has_timer=1

  # A timerless class is safe — no timer-ticket check needed.
  [ "$has_timer" -eq 0 ] && continue

  # ---- check: timer-ticket ----------------------------------------------
  # Find the stopped() method body (brace-counted) and look for any cancel call.
  # Accepts:  .cancel(   directly on a ticket field
  #           cancel*(   any helper method whose name contains "cancel"
  found_cancel=$(awk '
    BEGIN { in_stopped=0; depth=0; found=0 }
    /void[[:space:]]+stopped[[:space:]]*\(/ { in_stopped=1 }
    in_stopped {
      if (/cancel/) found=1
      for (i = 1; i <= length($0); i++) {
        c = substr($0, i, 1)
        if (c == "{") depth++
        if (c == "}") {
          depth--
          if (depth == 0) { in_stopped=0; break }
        }
      }
    }
    END { print found }
  ' "$f")

  if [ "$found_cancel" = "1" ]; then
    row PASS "timer-ticket" "$f: timer cancelled in stopped()"
  else
    row FAIL "timer-ticket" "$f: schedules a Clock ticket but stopped() does not cancel it"
  fi

done < <(find "$JAVA_ROOT" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# ---------------------------------------------------------------------------
# Check: companion-flag
# A boolean/int CLASS FIELD assigned true in the SAME METHOD BODY as a
# Clock.schedule* call must be assigned false inside stopped() OR started();
# a clear only in the expiry handler does not protect a stop/restart cycle —
# the same object is reused and the flag stays stuck.
# Only CLASS-SCOPE FIELDs (declared at brace depth 1, outside any method body)
# are candidates — a method-local boolean is re-initialised every call and
# cannot stay stuck across a stop/restart cycle.
# Real shape: CompPan BCompressorControl :1760 startingUp=true beside
# :1764 powerOnTicket=Clock.schedule; stopped() :1799-1805 cancels only.
# Root cause of pre-fix FP (anyNoHardware): Pass 1's candidate regex matched
# @NiagaraProperty( as a method signature and forward-walked the class body,
# pulling in Clock.schedule calls from unrelated methods.
# Fix: port the section-D method-boundary parser from lint-silent-protection.sh
# (net-brace-open detection + brace_depth >= 2 guard so the class body at
# depth 1 is never named as a method) + a class-scope FIELD pass.
# [ev: D1a; D1b; D1c; D1d; corpus B831 §S21]
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  _cf=$(awk "$MB_AWK"'
    BEGIN { n = 0 }
    { lines[++n] = $0 }
    END {
      mb_strip(lines, n, slines)

      # --- Phase 1: collect class-scope FIELD declarations (brace_depth == 1) ---
      # A boolean/int declared while the surrounding brace depth is exactly 1
      # (inside the class body, before any method opens) is a FIELD.
      # The same shape at depth >= 2 is a LOCAL and is not a candidate.
      n_fields = 0; brace_depth = 0
      for (i = 1; i <= n; i++) {
        ln = slines[i]; prev_depth = brace_depth
        for (ci = 1; ci <= length(ln); ci++) {
          c = substr(ln, ci, 1)
          if (c == "{") brace_depth++
          else if (c == "}") brace_depth--
        }
        if (prev_depth == 1 && \
            ln ~ /[[:space:]](boolean|int|long)[[:space:]][a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*[;=]/) {
          tmp = ln; gsub(/^[[:space:]]+/, "", tmp)
          while (tmp ~ /^(private|protected|public|static|final|volatile|transient)[[:space:]]/)
            sub(/^[a-z]+[[:space:]]+/, "", tmp)
          sub(/^(boolean|int|long)[[:space:]]+/, "", tmp)
          if (match(tmp, /^[a-zA-Z_][a-zA-Z0-9_]*/)) {
            fields[substr(tmp, 1, RLENGTH)] = 1; n_fields++
          }
        }
      }
      if (n_fields == 0) exit 0

      # --- Phase 2: method-boundary parser (shared fragment, PEAK depth) ---
      n_meth = mb_parse(slines, n, meth_start, meth_end, meth_name)

      # --- Phase 3: same-method binding (D1d) ---
      # FAIL only when a Clock.schedule* lies within [meth_start[m], meth_end[m]]
      # of the same method m that also contains the X = true assignment, and X is
      # a class FIELD. Binding is by line-range containment, not by proximity.
      flag_name = ""
      for (mi = 0; mi < n_meth && flag_name == ""; mi++) {
        ms = meth_start[mi]; me = meth_end[mi]
        body = ""
        for (i = ms; i <= me; i++) body = body " " slines[i]
        if (body !~ /Clock\.schedule/) continue
        tmp = body
        while (match(tmp, /[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*=[[:space:]]*true[[:space:]]*;/)) {
          frag = substr(tmp, RSTART, RLENGTH)
          ident = frag; sub(/[[:space:]]*=.*/, "", ident); gsub(/[[:space:]]/, "", ident)
          tmp = substr(tmp, RSTART + RLENGTH)
          if (ident != "" && ident in fields) { flag_name = ident; break }
        }
      }
      if (flag_name == "") exit 0

      # --- Pass 2 (unchanged): check stopped()/started() bodies for flag_name = false ---
      in_lc = 0; depth = 0; cleared = 0
      for (i = 1; i <= n; i++) {
        line = slines[i]
        if (!in_lc && line ~ /void[[:space:]]+(stopped|started)[[:space:]]*\(/) {
          in_lc = 1; depth = 0
        }
        if (in_lc) {
          if (line ~ (flag_name "[[:space:]]*=[[:space:]]*false")) cleared = 1
          for (c = 1; c <= length(line); c++) {
            ch = substr(line, c, 1)
            if (ch == "{") depth++
            if (ch == "}" && depth > 0) { depth--; if (depth == 0) { in_lc = 0; break } }
          }
        }
      }
      if (!cleared) print flag_name
    }
  ' "$f")
  [ -n "$_cf" ] && row FAIL "companion-flag" "$f: flag '${_cf}' set beside Clock.schedule* not cleared in stopped()/started()"
done < <(find "$JAVA_ROOT" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# ---------------------------------------------------------------------------
# Check: jdk-thread
# A B* Niagara component/service that uses JDK concurrency primitives instead
# of Clock.schedule — JDK pools are not bound to station lifecycle, and the
# station SecurityManager denies modifyThread to module code.
# Real shape: chihuahua BChiDashboardService :229/:305/:314.
# [ev: corpus B800 §800.3] [ev: corpus B806]
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  grep -qE 'class[[:space:]]+B[A-Za-z0-9_]+[[:space:]]+(extends|implements)[[:space:]]+B[A-Za-z0-9_]' "$f" || continue
  grep -qE 'ScheduledExecutorService|Executors\.|new[[:space:]]+Thread\(' "$f" || continue
  _jdk_class=$(grep -m1 -oE 'class[[:space:]]+B[A-Za-z0-9_]+' "$f" | awk '{print $2}')
  row FAIL "jdk-thread" "$f: ${_jdk_class:-BUnknown} uses JDK concurrency — use Clock.schedule instead (SecurityManager denies modifyThread)"
done < <(find "$JAVA_ROOT" -type d -name '.*' -prune -o -name '*.java' -print | sort)

# ---------------------------------------------------------------------------
# Check: changed-sched
# A Clock.schedule* call reachable from changed() or started() (directly or
# via one private callee, one level deep) must have an isRunning() or
# Sys.atSteadyState() guard IN THE SCHEDULING BODY — a guard only in the
# caller does not protect the callee body.
# Real shape: ColdRoomPan BEvaporatorUnit changed()->applyRunCmd()->
# Clock.schedule (pre-fix); fix adds if(!Sys.atSteadyState())return in
# applyRunCmd(). NotRunningException x6 on PANCCADIA console logs.
# [ev: corpus B816]
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  _cs=$(awk '
    BEGIN { n = 0 }
    { lines[++n] = $0 }
    END {
      n_kw = split("if for while switch catch try finally return new assert synchronized throw else do instanceof super this void boolean int long double float String abstract final static public private protected class interface enum Clock Sys BRelTime BComponent BAbstractService", KW_STR, " ")
      for (k = 1; k <= n_kw; k++) kw[KW_STR[k]] = 1

      i = 1
      while (i <= n) {
        line = lines[i]
        if (match(line, /[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\(/)) {
          cand = substr(line, RSTART, RLENGTH)
          sub(/[[:space:]]*\($/, "", cand)
          if (cand != "" && !(cand in kw)) {
            j = i; in_b = 0; depth = 0; body = ""; done_j = 0
            while (j <= n && !done_j) {
              ln = lines[j]
              for (c = 1; c <= length(ln); c++) {
                ch = substr(ln, c, 1)
                if (!in_b) {
                  if (ch == "{") { in_b = 1; depth = 1 }
                  else if (ch == ";") { done_j = 1; break }
                } else {
                  if (ch == "{") depth++
                  else if (ch == "}") {
                    depth--
                    if (depth == 0) {
                      if (!(cand in meth)) meth[cand] = body
                      i = j; done_j = 1; break
                    }
                  }
                  body = body ch
                }
              }
              if (!done_j) { body = body " "; j++ }
            }
          }
        }
        i++
      }

      for (lc in meth) {
        if (lc != "changed" && lc != "started") continue
        body = meth[lc]
        if (body ~ /Clock\.schedule/ && body !~ /isRunning|atSteadyState/) {
          print "direct"; exit
        }
        tmp = body
        while (match(tmp, /[a-z][a-zA-Z0-9_]*[[:space:]]*\(/)) {
          callee = substr(tmp, RSTART, RLENGTH)
          sub(/[[:space:]]*\($/, "", callee)
          tmp = substr(tmp, RSTART + RLENGTH)
          if (callee in kw || callee ~ /^[A-Z]/) continue
          if (callee in meth) {
            cb = meth[callee]
            if (cb ~ /Clock\.schedule/ && cb !~ /isRunning|atSteadyState/) {
              print "callee:" callee; exit
            }
          }
        }
      }
    }
  ' "$f")
  [ -n "$_cs" ] && row FAIL "changed-sched" "$f: Clock.schedule reachable from changed()/started() without isRunning()/atSteadyState() guard in scheduling body"
done < <(find "$JAVA_ROOT" -type d -name '.*' -prune -o -name '*.java' -print | sort)

[ "$FAILED" -eq 1 ] && exit 1
exit 0
