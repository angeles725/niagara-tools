#!/usr/bin/env bash
# lint-timers.sh — timer-ticket conformance lint for Niagara N4 Java modules (Campaign 6 PR8).
#
# Detects two lifecycle defects in module Java source files:
#
#   timer-ticket      A class that owns a Clock.Ticket (field declaration or
#                     Clock.schedule*() call) but its stopped() override does not
#                     cancel the ticket — the timer leaks on station stop.
#                     [ev: corpus B787]
#
#   discarded-ticket  A Clock.schedule*() call whose return value is not captured
#                     — the Clock.Ticket is immediately lost, no way to cancel it.
#
# Usage: lint-timers.sh <java-root>
#   Scans all *.java files recursively under <java-root>.
#   Prints FAIL|PASS rows per owning class.
#   A class with no timers emits no FAIL (silently skipped).
#   Exit 0 (all PASS) or 1 (any FAIL).
#
# Row format: FAIL|PASS  <check>  <file>: <detail>
# Exit: 0 no FAIL · 1 any FAIL · 2 usage · 3 env
# This script is VCS-free by design. version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

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

done < <(find "$JAVA_ROOT" -name '*.java' | sort)

[ "$FAILED" -eq 1 ] && exit 1
exit 0
