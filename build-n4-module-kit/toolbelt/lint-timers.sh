#!/usr/bin/env bash
# lint-timers.sh — timer-ticket conformance lint for Niagara N4 Java modules (Campaign 6 PR8).
#
# Usage: lint-timers.sh <java-root>
#   Scans all *.java files recursively under <java-root> (dot-dirs and sibling *-worktrees checkouts pruned).
#   Prints FAIL|PASS rows per owning class, and advisory WARN rows (job-cancel, jdk-thread,
#   atSteadyState-only-timer, orphan-flag, release-gate-parity).
#   A class with no timers emits no FAIL (silently skipped).
#
# Row format: FAIL|PASS|WARN  <check>  <file>:<line>  <detail>   (WARN never changes the exit status)
#   <line> is the line the row is about: discarded-ticket the call; timer-ticket the first Clock.Ticket /
#   Clock.schedule* line (a missed field: its declaration); companion-flag the `flag = true` line;
#   jdk-thread the modifying call (FAIL) or the first JDK concurrency use (WARN); job-cancel the loop;
#   changed-sched the changed()/started() line; atSteadyState-only-timer its declaration; orphan-flag
#   and release-gate-parity the opening brace of the method they name.
# Every check reads code: comments stripped and literal contents blanked by the shared javasrc helpers
# (Strip, BlankLine), so a call, a guard or a cancel inside a comment or a string is never code.
# Exit: 0 no FAIL · 1 any FAIL · 2 usage · 3 env (incl. a sub-directory find cannot enter)
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-timers  <root>  no *.java sources found (wrong path or empty source tree)` (go/internal/lint
# Spec.EmptyError; declared difference 2026-10-23: the bash lint exited 0 with no row).
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# Evidence of the checks below (also cited at each check): [ev: corpus B800 §800.3] [ev: corpus B806] [ev: corpus B816] [ev: retro continuous-fan-post-defrost-delay Δ2] [ev: retro panccadia-restart-seq-comp-lockout-hours Δ8] [ev: corpus B787] [ev: audit-2026-10-03 A1]
# Mutation: S21-neg -- removes method-scope exclusion, causing method-local boolean + schedule to false-FAIL
# Mutation: S21-misparse -- drops the javasrc.Methods depth>=2 guard, making @NiagaraProperty(defaultValue=new Foo()) false-parse as a method
# Mutation: TT-comment -- matching the raw line (comments kept) lets a commented 'cancel' pass
# Mutation: TT-partial -- one cancel call for any field passes the whole class
# Mutation: TT-string -- not blanking string literals lets log("a.cancel()") pass
# Mutation: TT-helper -- without the callee bodies the stopped() -> cancelAll() -> cancelX() shape false-FAILs
# Mutation: TT-overaccept -- accepting any call with the field as an argument lets log(a) + b.cancel() pass a
# Mutation: TT-boundary -- an unanchored Ticket match reads 'MyTicket note' as a ticket field
# Mutation: TT-query -- a name merely CONTAINING cancel accepts isCancelled(a) as a cancel
# Mutation: TT-safecancel -- requiring the name to START with cancel false-FAILs safeCancel(a)/doCancel(b)
# Mutation: TT-helper-body -- accepting a helper by its name again lets logCancelRequest(a) pass a
# Mutation: TT-helper-param -- any .cancel( in the helper body counts, so stopTimer(a) { other.cancel(); } passes a
# Mutation: TT-helper-this -- rejecting every qualified call false-FAILs this.cancelTicket(a)
# Mutation: TT-helper-sigline -- reading parameters only from the brace line false-FAILs an Allman-style helper
# Mutation: TT-helper-nested -- a separator class that admits ( binds the call to the enclosing if and misses it
# Mutation: TT-noname -- a schedule with no named ticket field rendered "does not cancel ticket (ticket)"
# Mutation: LC-orphan -- dropping the orphan-flag pass lets a cancelled ticket leave its companion flag stuck true
# Mutation: LC-orphan-ok -- ignoring the callee scope false-WARNs a disable path that clears the flag in a helper
# Mutation: LC-orphan-callers -- ignoring the callers reports a cancel helper whose callers own the flag
# Mutation: LC-gate -- dropping the parity pass lets an exit handler re-assert gated outputs ungated
# Mutation: LC-gate-ok -- not looking for the gate call in the release point false-WARNs the fixed shape
# Mutation: TT-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
# Mutation: TC-B -- ignoring the modifying call lets a shutdownNow() with no MANAGE_EXECUTION pass as a WARN
# Mutation: JT-perm -- ignoring the permission lookup FAILs an executor whose module requests MANAGE_EXECUTION
# Mutation: JT-scope -- accepting any group type grants the station from a workbench-only request
# Mutation: JT-header -- a single-line class-header match misses `class BX` / `extends BComponent` on two lines
# Mutation: JT-self -- counting Thread.currentThread().interrupt() FAILs a thread restoring its own flag
# Mutation: JT-comment -- matching comment and string text FAILs a shutdownNow() that is only mentioned
# Mutation: JC-loop -- dropping the job-cancel pass lets a BSimpleJob loop run with no isAlive() check
# Mutation: JC-comment -- matching comment text lets a commented-out isAlive() silence the WARN
# Mutation: TR1-code -- scanning the raw lines (comments and strings kept) FAILs a schedule that is only mentioned
# Mutation: TR1-shape -- leaving Line 0 prints the file-level "<file>: <msg>" row with no line
#
# Detects timer lifecycle defects in module Java source files:
#
#   timer-ticket      A class that owns a Clock.Ticket (field declaration or
#                     Clock.schedule*() call) but its stopped() override does not
#                     cancel EVERY ticket field — the timer leaks on station stop.
#                     Checked per ticket field on comment- and string-blanked code:
#                     `f.cancel(` / `f[i].cancel(` / a call taking `f` (bare or this.)
#                     to a SAME-FILE method whose body cancels its own parameter —
#                     decided by the helper's body, not its name (log(f),
#                     isCancelled(f), a helper cancelling another ticket and an
#                     other-file helper do not count), in stopped() or in any same-file method it reaches
#                     through unqualified calls. A `cancel` token in a comment or string, or a cancel of
#                     another ticket, does not count. No named field (only
#                     Clock.schedule* calls) → any `.cancel(` in that scope counts.
#                     [ev: corpus B787] [ev: audit-2026-10-03 A1]
#
#   discarded-ticket  A Clock.schedule*() call whose return value is not captured
#                     — the Clock.Ticket is immediately lost, no way to cancel it.
#                     Captured means its own statement (from the last ; { or } before
#                     the call, across lines) assigns (=, not == != <= >=) or returns it:
#                     `t =\n Clock.schedule(...)` is kept, `y = 1; Clock.schedule(...)` is not.
#
#   companion-flag    A boolean/int flag assigned true beside a Clock.schedule*
#                     call that is not assigned false inside stopped() or started();
#                     a clear only in the expiry handler does not count — a
#                     stop/restart cycle keeps the object alive and the flag stuck.
#                     Real shape: CompPan BCompressorControl :1760/:1764 startingUp.
#                     [ev: corpus B801] [ev: corpus B812]
#
#   jdk-thread        A B* class (its header may span lines up to the first `{`) that
#                     uses JDK concurrency (ExecutorService, Executors.*, ExecutorUtil.*,
#                     new Thread). Creating and starting a thread is ALLOWED (Tridium
#                     Lab 9), so that alone is a WARN recommending Clock / BWorker /
#                     BSimpleJob / a named ExecutorUtil executor. FAIL only when the
#                     class MODIFIES a thread — .shutdown( .shutdownNow( .interrupt(
#                     .setPriority( .setDaemon( (ExecutorUtil.shutdown included;
#                     Thread.currentThread().interrupt() excluded) on comment- and
#                     string-blanked code — and the nearest module-permissions.xml
#                     above the file requests no MANAGE_EXECUTION under
#                     type="station" or type="all": the modifyThread check denies the
#                     call at stop and the thread outlives the component.
#                     Real shape: chihuahua BChiDashboardService :305/:455.
#                     [ev: corpus B800 §800.3] [ev: corpus B806]
#                     [ev: retro n4-course-threading-jobs-doctrine Δ2]
#                     [ev: retro n4-course-handson-services-history-threading Δ3]
#
#   job-cancel        WARN (advisory). A class extending BSimpleJob whose run(...) body
#                     holds a for/while/do loop but no isAlive() call: cancel only
#                     interrupts the job thread, so the loop must check isAlive() and
#                     throw JobCancelException. [ev: retro n4-course-threading-jobs-doctrine Δ6]
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
#
# Implementation: rule package go/internal/lint/rules/timers (a lint.TreeRule: every check runs over every
# file in walk order, so rows come grouped by check as above) on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). It keeps the awk programs' gawk semantics: POSIX
# leftmost-longest matching, and gawk's `for (k in a)` hash order for the orphan-flag and
# release-gate-parity rows. This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-timers <args>`. Needs `go` on PATH to build (exit 3 otherwise). An unreadable
# module-permissions.xml is an env error too: exit 3, the run stops at that file.
#
# Declared differences from the bash lint (truth fixes 2026-10-23, R8-D14 — a port keeps no known lie):
#   - discarded-ticket, timer-ticket, companion-flag, changed-sched, atSteadyState-only-timer, orphan-flag
#     and release-gate-parity read code, not raw or comment-stripped lines: a Clock.schedule, a guard or
#     a flag assignment in a comment or a string no longer FAILs or passes a class.
#   - discarded-ticket judges the call's own statement (an `=` of an earlier statement on the line no
#     longer counts; a multi-line `t =\n Clock.schedule(` is kept); a call on an unterminated last line
#     is reported (bash `read` skipped that line).
#   - every row carries its line (above); job-cancel and jdk-thread moved the "line N" out of the text.
#   - jdk-thread names the permissions file by its module directory (`m-rt/module-permissions.xml`), not
#     a second absolute path, and reads the grant directly: a path with a space no longer FAILs a module
#     that requests MANAGE_EXECUTION (bash `read -r` split the path).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-timers: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-timers "$@"
