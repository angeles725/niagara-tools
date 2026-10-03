#!/usr/bin/env bash
# lint-timers.sh — timer-ticket conformance lint for Niagara N4 Java modules (Campaign 6 PR8).
#
# Usage: lint-timers.sh <java-root>
#   Scans all *.java files recursively under <java-root> (dot-dirs pruned).
#   Prints FAIL|PASS rows per owning class.
#   A class with no timers emits no FAIL (silently skipped).
#
# Row format: FAIL|PASS|WARN  <check>  <file>: <detail>   (WARN never changes the exit status)
# Exit: 0 no FAIL · 1 any FAIL · 2 usage · 3 env (incl. a sub-directory find cannot enter)
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# Evidence of the checks below (also cited at each check): [ev: corpus B816] [ev: retro continuous-fan-post-defrost-delay Δ2] [ev: retro panccadia-restart-seq-comp-lockout-hours Δ8]
# Mutation: S21-neg -- removes method-scope exclusion, causing method-local boolean + schedule to false-FAIL
# Mutation: S21-misparse -- drops max_d>=2 guard, making @NiagaraProperty(defaultValue=new Foo()) false-parse as a method
# Mutation: TT-comment -- matching the raw line (comments kept) lets a commented 'cancel' pass
# Mutation: TT-partial -- one cancel call for any field passes the whole class
# Mutation: TT-string -- not blanking string literals lets log("a.cancel()") pass
# Mutation: TT-helper -- without the callee bodies the stopped() -> cancelAll() -> cancelX() shape false-FAILs
# Mutation: TT-overaccept -- accepting any call with the field as an argument lets log(a) + b.cancel() pass a
# Mutation: TT-boundary -- an unanchored Ticket match reads 'MyTicket note' as a ticket field
# Mutation: TT-query -- a name merely CONTAINING cancel accepts isCancelled(a) as a cancel
# Mutation: TT-safecancel -- requiring the name to START with cancel false-FAILs safeCancel(a)/doCancel(b)
# Mutation: LC-orphan -- dropping the orphan-flag pass lets a cancelled ticket leave its companion flag stuck true
# Mutation: LC-orphan-ok -- ignoring the callee scope false-WARNs a disable path that clears the flag in a helper
# Mutation: LC-orphan-callers -- ignoring the callers reports a cancel helper whose callers own the flag
# Mutation: LC-gate -- dropping the parity pass lets an exit handler re-assert gated outputs ungated
# Mutation: LC-gate-ok -- not looking for the gate call in the release point false-WARNs the fixed shape
# Mutation: TT-finderr -- ignoring the find status skips an unreadable sub-directory and reports clean
#
# Detects timer lifecycle defects in module Java source files:
#
#   timer-ticket      A class that owns a Clock.Ticket (field declaration or
#                     Clock.schedule*() call) but its stopped() override does not
#                     cancel EVERY ticket field — the timer leaks on station stop.
#                     Checked per ticket field on comment- and string-blanked code:
#                     `f.cancel(` / `f[i].cancel(` / a call taking `f` to a method whose
#                     name contains `cancel`/`Cancel` and is not a query (cancelTicket(f),
#                     safeCancel(f) count; log(f), isCancelled(f) do not), in stopped() or in any same-file method it reaches
#                     through unqualified calls. A `cancel` token in a comment or string, or a cancel of
#                     another ticket, does not count. No named field (only
#                     Clock.schedule* calls) → any `.cancel(` in that scope counts.
#                     [ev: corpus B787] [ev: audit-2026-10-03 A1]
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
#   orphan-flag       WARN (advisory). T's companion flag F is set true where
#                     `T = Clock.schedule*(recv, delay, act, …)` is armed and cleared in
#                     T's expiry handler do<Act>(). A method other than stopped()/
#                     started()/the handler whose own body cancels T, and that never
#                     assigns F (itself or in a same-file callee), leaves F stuck true —
#                     the handler no longer runs. Not reported: a method that re-arms T
#                     (a re-arm), a helper whose every same-file caller assigns F (or
#                     re-arms T, or is a lifecycle callback), and a class with a self-heal
#                     `T == null && F … F = false`. [ev: corpus B801] [ev: corpus B812] [ev: audit-2026-10-03 A8]
#
#   release-gate-parity WARN (advisory). started()/atSteadyState() calls a begin*()
#                     gate and re-applies outputs through apply*() methods; a release
#                     point (exit*/end*/finish*/leave*/on*Exit|Expired|End) that calls
#                     one of those apply*() methods without calling a begin*() gate
#                     re-asserts outputs ungated (types/logic.md § release-point gate,
#                     rule 2). [ev: retro continuous-fan-post-defrost-delay Δ2]
#                     [ev: retro panccadia-restart-seq-comp-lockout-hours Δ8]
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
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

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
# One fail-closed walk feeds every check: a sub-directory find cannot enter is exit 3, never a shorter list.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$JAVA_ROOT" -name '*.java'; then
  printf 'lint-timers: cannot list every file under %s: %s\n' "$JAVA_ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  exit 3
fi

# timer-ticket (per ticket field). Input: one Java file. Output lines: "OK" | "NOSTOP" | "MISS <field>".
# Comments are blanked by mb_strip, string/char literal contents here; methods come from mb_parse.
cat > "$_TMP/ticket.awk" <<'AWKEOF'
function blank_str(s,    out, j, c, q) {
  out = ""; q = ""
  for (j = 1; j <= length(s); j++) {
    c = substr(s, j, 1)
    if (q != "") { if (c == "\\") { j++; continue } if (c == q) { q = ""; out = out c } ; continue }
    if (c == "\"" || c == "'") q = c
    out = out c
  }
  return out
}
{ raw[++n] = $0 }
END {
  mb_strip(raw, n, st)
  for (i = 1; i <= n; i++) code[i] = blank_str(st[i])
  cnt = mb_parse(code, n, ms, me, mn)
  for (i = 1; i <= n; i++) inm[i] = 0
  for (k = 0; k < cnt; k++) { for (i = ms[k]; i <= me[k]; i++) inm[i] = 1; body[mn[k]] = body[mn[k]] "\n" lines_of(ms[k], me[k]) }
  # ticket FIELDS: Clock.Ticket / Ticket declarations outside every method body (arrays included)
  nf = 0
  for (i = 1; i <= n; i++) {
    if (inm[i]) continue
    t = code[i]
    while (match(t, /(^|[^A-Za-z0-9_])(Clock[[:space:]]*\.[[:space:]]*)?Ticket[[:space:]]*(\[[[:space:]]*\])?[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      d = substr(t, RSTART, RLENGTH); t = substr(t, RSTART + RLENGTH)
      sub(/.*[[:space:]]/, "", d)
      if (!(d in isf)) { isf[d] = 1; fld[++nf] = d }
    }
  }
  if (!("stopped" in body)) { print "NOSTOP"; exit }
  scope = body["stopped"]
  # same-file methods reachable from stopped() through unqualified calls (transitive: stopped -> cancelAll ->
  # cancelInterval); each method body is added once, so the walk ends
  hop["stopped"] = 1; qn = 1; queue[1] = "stopped"; qi = 0
  while (qi < qn) {
    t = body[queue[++qi]]
    while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
      id = substr(t, RSTART, RLENGTH); pre = (RSTART > 1) ? substr(t, RSTART - 1, 1) : ""
      t = substr(t, RSTART + RLENGTH); sub(/[[:space:]]*\($/, "", id)
      if (pre != "." && (id in body) && !(id in hop)) { hop[id] = 1; queue[++qn] = id; scope = scope "\n" body[id] }
    }
  }
  if (nf == 0) { print ((scope ~ /\.[[:space:]]*cancel[[:space:]]*\(/) ? "OK" : "MISS (ticket)"); exit }
  miss = 0
  for (q = 1; q <= nf; q++) {
    f = fld[q]
    if (cancels(scope, f)) continue
    print "MISS " f; miss = 1
  }
  if (!miss) print "OK"
}
function lines_of(a, b,    s, i) { s = ""; for (i = a; i <= b; i++) s = s code[i] "\n"; return s }
# f.cancel( · f[...].cancel( · this.f.cancel( · a call taking f as a whole argument to a method whose NAME
# contains "cancel"/"Cancel" and is not a query: cancel(f), cancelTicket(f), safeCancel(f), doCancel(f) count;
# log(f), isCancelled(f), wasCancelled(f), hasCancel(f) do not (a query prefix or a "Cancelled" past tense).
function cancels(s, f,    re1, re2, t, nm) {
  re1 = "(^|[^A-Za-z0-9_.])(this[[:space:]]*\\.[[:space:]]*)?" f "[[:space:]]*(\\[[^]]*\\][[:space:]]*)?\\.[[:space:]]*cancel[[:space:]]*\\("
  if (s ~ re1) return 1
  re2 = "[A-Za-z0-9_]*[Cc]ancel[A-Za-z0-9_]*[[:space:]]*\\(([^()]*[^A-Za-z0-9_.])?" f "[[:space:]]*[,)]"
  t = s
  while (match(t, re2)) {
    nm = substr(t, RSTART, RLENGTH); t = substr(t, RSTART + RLENGTH)
    sub(/[[:space:]]*\(.*/, "", nm)
    if (nm ~ /^(is|was|has|had|can|should|get|check|needs|did|will)[A-Z]/ || nm ~ /Cancelled|Canceled/) continue
    return 1
  }
  return 0
}
AWKEOF
printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"

# orphan-flag + release-gate-parity (WARN). Output lines: "ORPHAN <method> <ticket> <flag>" | "GATE <method> <apply> <gate>".
cat > "$_TMP/lifecycle.awk" <<'AWKEOF'
{ raw[++n] = $0 }
END {
  mb_strip(raw, n, code)
  all = ""; for (i = 1; i <= n; i++) all = all code[i] "\n"
  cnt = mb_parse(code, n, ms, me, mn)
  for (k = 0; k < cnt; k++) { b = ""; for (i = ms[k]; i <= me[k]; i++) b = b code[i] "\n"; body[mn[k]] = body[mn[k]] b; names[k] = mn[k] }
  # class-scope boolean fields (outside every method)
  for (i = 1; i <= n; i++) inm[i] = 0
  for (k = 0; k < cnt; k++) for (i = ms[k]; i <= me[k]; i++) inm[i] = 1
  for (i = 1; i <= n; i++) if (!inm[i] && match(code[i], /boolean[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*[;=]/)) {
    d = substr(code[i], RSTART, RLENGTH); sub(/^boolean[[:space:]]+/, "", d); sub(/[[:space:]]*[;=]$/, "", d); bfield[d] = 1
  }
  # ---- orphan-flag: pairs (T, F) — F set true where T is armed AND cleared in T's expiry handler ----
  # Handler of `T = Clock.schedule*(recv, delay, act, …)` = method do<Act>(); only a flag that handler clears
  # is T's companion (the companion-flag doctrine), so an unrelated `x = true` beside a schedule is not paired.
  np = 0
  for (m in body) {
    t = body[m]
    while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*Clock[[:space:]]*\.[[:space:]]*schedule[A-Za-z]*[[:space:]]*\(/)) {
      T = substr(t, RSTART, RLENGTH); sub(/[[:space:]]*=.*/, "", T); t = substr(t, RSTART + RLENGTH)
      act = third_arg(t)
      if (act !~ /^[A-Za-z_][A-Za-z0-9_]*$/) continue
      h = "do" toupper(substr(act, 1, 1)) substr(act, 2)
      if (!(h in body)) continue
      u = body[m]
      while (match(u, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*true[[:space:]]*;/)) {
        F = substr(u, RSTART, RLENGTH); sub(/[[:space:]]*=.*/, "", F); u = substr(u, RSTART + RLENGTH)
        if (!(F in bfield)) continue
        if (body[h] !~ ("(^|[^A-Za-z0-9_.])" F "[[:space:]]*=[[:space:]]*false")) continue
        key = T SUBSEP F
        if (!(key in paired)) { paired[key] = 1; np++; PT[np] = T; PF[np] = F; PH[np] = h }
      }
    }
  }
  for (m in body) {
    if (m == "stopped" || m == "started") continue
    sc = reach(m)
    for (p = 1; p <= np; p++) {
      if (m == PH[p]) continue
      # the method that cancels T DIRECTLY reports (its callers would only repeat the row)
      if (body[m] !~ ("(^|[^A-Za-z0-9_.])" PT[p] "[[:space:]]*\\.[[:space:]]*cancel[[:space:]]*\\(")) continue
      # a self-heal anywhere in the class (`if (T == null && F) F = false;`) releases the orphan
      if (all ~ (PT[p] "[[:space:]]*==[[:space:]]*null[^;]*" PF[p] "[[:space:]]*=[[:space:]]*false")) continue
      if (sc ~ ("(^|[^A-Za-z0-9_.])" PT[p] "[[:space:]]*=[[:space:]]*Clock[[:space:]]*\\.[[:space:]]*schedule")) continue
      if (sc ~ ("(^|[^A-Za-z0-9_.])" PF[p] "[[:space:]]*=[^=]")) continue
      # a helper whose every same-file caller assigns F (or re-arms T, or is a lifecycle callback / the
      # handler) is not an orphan: the caller owns the flag
      if (callers_handle(m, PT[p], PF[p], PH[p])) continue
      print "ORPHAN " m " " PT[p] " " PF[p]
    }
  }
  # ---- release-gate-parity ----
  boot = ""
  if ("started" in body) boot = boot reach("started")
  if ("atSteadyState" in body) boot = boot reach("atSteadyState")
  ng = 0; na = 0
  t = boot
  while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
    id = substr(t, RSTART, RLENGTH); pre = (RSTART > 1) ? substr(t, RSTART - 1, 1) : ""
    t = substr(t, RSTART + RLENGTH); sub(/[[:space:]]*\($/, "", id)
    if (pre == "." || !(id in body)) continue
    if (id ~ /^begin[A-Z]/ && !(id in isg)) { isg[id] = 1; G[++ng] = id }
    if (id ~ /^apply[A-Z]/ && !(id in isa)) { isa[id] = 1; A[++na] = id }
  }
  if (ng == 0 || na == 0) exit
  for (m in body) {
    if (m !~ /^(exit|end|finish|leave)[A-Z]/ && m !~ /^on[A-Za-z0-9_]*(Exit|Expired|Expiry|End)[A-Za-z0-9_]*$/) continue
    sc = reach(m); hit = ""
    for (a = 1; a <= na; a++) if (body[m] ~ ("(^|[^A-Za-z0-9_.])" A[a] "[[:space:]]*\\(")) { hit = A[a]; break }
    if (hit == "") continue
    gated = 0
    for (g = 1; g <= ng; g++) if (sc ~ ("(^|[^A-Za-z0-9_.])" G[g] "[[:space:]]*\\(")) gated = 1
    if (!gated) print "GATE " m " " hit " " G[1]
  }
}
# callers_handle(m, T, F, H): m has at least one same-file caller, and every caller assigns F or re-arms T
# (in its own reach) or is stopped()/started()/H
function callers_handle(m, T, F, H,    c, any, sc, re) {
  any = 0; re = "(^|[^A-Za-z0-9_.])" m "[[:space:]]*\\("
  for (c in body) {
    if (c == m || body[c] !~ re) continue
    any = 1
    if (c == "stopped" || c == "started" || c == H) continue
    sc = reach(c)
    if (sc ~ ("(^|[^A-Za-z0-9_.])" F "[[:space:]]*=[^=]")) continue
    if (sc ~ ("(^|[^A-Za-z0-9_.])" T "[[:space:]]*=[[:space:]]*Clock[[:space:]]*\\.[[:space:]]*schedule")) continue
    return 0
  }
  return any
}
# third_arg(s): the third depth-0 argument of the call whose "(" was just consumed (s starts after it)
function third_arg(s,    i, c, d, k, a) {
  d = 0; k = 1; a = ""
  for (i = 1; i <= length(s); i++) {
    c = substr(s, i, 1)
    if (c == "(") d++
    else if (c == ")") { if (d == 0) break; d-- }
    else if (c == "," && d == 0) { k++; continue }
    if (k == 3) a = a c
  }
  gsub(/[[:space:]]/, "", a)
  return a
}
# reach(m): m's body plus every same-file method it reaches through unqualified calls
function reach(m,    seen, q, qn, qi, out, t, id, pre) {
  qn = 1; q[1] = m; seen[m] = 1; qi = 0; out = ""
  while (qi < qn) {
    t = body[q[++qi]]; out = out "\n" t
    while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(/)) {
      id = substr(t, RSTART, RLENGTH); pre = (RSTART > 1) ? substr(t, RSTART - 1, 1) : ""
      t = substr(t, RSTART + RLENGTH); sub(/[[:space:]]*\($/, "", id)
      if (pre != "." && (id in body) && !(id in seen)) { seen[id] = 1; q[++qn] = id }
    }
  }
  return out
}
AWKEOF

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

  # ---- check: timer-ticket (per ticket field; see header) ------------------
  if ! tt=$(awk -f "$_TMP/method-boundary.awk" -f "$_TMP/ticket.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-timers: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
    exit 3
  fi
  case "$tt" in
    OK) row PASS "timer-ticket" "$f: timer cancelled in stopped()" ;;
    NOSTOP) row FAIL "timer-ticket" "$f: schedules a Clock ticket but has no stopped() override to cancel it" ;;
    *) while IFS= read -r m; do
         [ -n "$m" ] || continue
         row FAIL "timer-ticket" "$f: stopped() does not cancel ticket ${m#MISS } (a cancel in a comment, a string or of another ticket does not count)"
       done <<< "$tt" ;;
  esac

done < "$_TMP/files"

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
done < "$_TMP/files"

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
done < "$_TMP/files"

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
done < "$_TMP/files"

# ---------------------------------------------------------------------------
# Check: atSteadyState-only-timer
# A class that arms a timer (Clock.schedule/schedulePeriodically) inside
# atSteadyState() but never overrides started() will silently fail to arm
# the timer when mounted onto an already-running station (commissioning
# drag-drop, component enable, parent start after initial bootstrap).
# atSteadyState() fires ONCE during station bootstrap; a late-mounted
# component receives started() but NOT atSteadyState().
# Emit WARN (exit 0) — not FAIL: a BTimeTrigger subclass may legitimately
# rely on the parent's started() override.  Only flag when no started()
# override is present anywhere in the file.
# Live case: BDefrostController (ColdRoomPan-rt), PANCCADIA León.
# [ev: corpus B729 §729.4; retro 2026-09-18-insights-issues-catalog-deltas Δ2]
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  _atss=$(awk '
    BEGIN { n = 0 }
    { lines[++n] = $0 }
    END {
      in_atss = 0; depth = 0; atss_has_sched = 0; has_started = 0
      for (i = 1; i <= n; i++) {
        ln = lines[i]
        if (ln ~ /void[[:space:]]+started[[:space:]]*\(/) has_started = 1
        if (!in_atss && ln ~ /void[[:space:]]+atSteadyState[[:space:]]*\(/) {
          in_atss = 1; depth = 0
        }
        if (in_atss) {
          if (ln ~ /Clock\.schedule/) atss_has_sched = 1
          for (ci = 1; ci <= length(ln); ci++) {
            c = substr(ln, ci, 1)
            if (c == "{") depth++
            else if (c == "}") {
              depth--
              if (depth == 0) { in_atss = 0 }
            }
          }
        }
      }
      if (atss_has_sched && !has_started) print "warn"
    }
  ' "$f")
  [ -n "$_atss" ] && row WARN "atSteadyState-only-timer" "$f: timer armed only in atSteadyState() — add started() override with Sys.atSteadyState() guard or commissioning-time mounts will never arm the timer"
done < "$_TMP/files"

# ---------------------------------------------------------------------------
# Lifecycle WARNs (advisory, never change the exit): orphan-flag, release-gate-parity. See the header.
# ---------------------------------------------------------------------------
while IFS= read -r f; do
  if ! lc=$(awk -f "$_TMP/method-boundary.awk" -f "$_TMP/lifecycle.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-timers: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
    exit 3
  fi
  while read -r kind m x y; do
    case "$kind" in
      ORPHAN) row WARN "orphan-flag" "$f: ${m}() cancels ${x} but never clears its companion flag ${y} (set true beside the schedule) -- the expiry handler that clears it will not run" ;;
      GATE)   row WARN "release-gate-parity" "$f: ${m}() re-applies ${x}() without the ${y}() gate that started()/atSteadyState() uses (types/logic.md release-point gate rule 2)" ;;
    esac
  done <<< "$lc"
done < "$_TMP/files"

[ "$FAILED" -eq 1 ] && exit 1
exit 0
