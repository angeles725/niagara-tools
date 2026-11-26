#!/usr/bin/env bash
# lint-dashboard-write-save.sh — flags a -ux dashboard that writes station state with the request Context while
# nothing in its module requests a station save.
#
# A write that is never saved lives only in the running station: the next restart loses it, and a driver that
# re-sends its saved value (writeOnStart/writeOnUp) drives the equipment from the stale one (a dashboard shipped
# three releases without a save path; a restart switched a unit ON). Gate form of BUILD-LOOP §6.b item 6.
# Pattern and the escape: types/dashboard.md § Write path. [ev: retro lint-dashboard-write-save Δ1] (issue #821)
#
# Files: every *.java under a -rt, -ux, -wb or -se part's src/ of the module root (srcTest/ and build/ pruned);
# comments and string literal contents are not code (go/internal/javasrc).
#   write     in a -ux file: .set( / .add( / .remove( / .invoke( whose last argument is a Context or WebOp declared
#             in that file (the request's Context), or any addSpecialEvent(. A list add, a one-argument remove or a
#             null-Context write is no write here.
#   save      anywhere in the module: BStation.save (Clock.schedule(station, d, BStation.save, null)),
#             Sys.getStation().save(), station.saveAsync(, Station.save*(
#   declared  MODULE-MAP.md at the module root holds a non-empty `persistence: <where>` line (outside ``` fences):
#             the module saves on purpose elsewhere (the control module's ConfigSaveScheduler) or needs none.
# One FAIL row per write line when the module has writes, no save and no declaration. Two WARN rows (exit 0, issue
# #1118), each with its own check code, both silenced by the same `persistence:` declaration:
#   write-path-no-save  a method of a -ux file that writes with the request Context and has no path to a save
#                       (requestSave() or a station save, called directly or through methods of the module), in a
#                       module that saves and a file where another method does reach a save; a write helper that a
#                       saving method of the same file calls is covered by that caller. One row, on its first write.
#   stop-drops-save     a stopped() that cancels a save ticket (<x>Ticket.cancel()) and neither calls <x>.save*( nor
#                       a method that does: a write 5-60 s before a restart or servlet stop is never saved.
#
# Implementation: go/internal/lint/rules/dashboardsave on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-dashboard-write-save <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise).
#
# Usage:  lint-dashboard-write-save.sh [--strict] <module-root>
#   Row:  FAIL  lint-dashboard-write-save  <file>:<line>  <detail>
#         WARN  write-path-no-save|stop-drops-save  <file>:<line>  <detail>
#   Exit: 0  no row or WARN only · 1  any FAIL · 3  usage/env, a root with no *.java under a -ux part's src/, an unreadable MODULE-MAP.md
# VCS-free by design (kit-links L2).
# Mutation: LDWS1 -- matching any last argument instead of a declared Context FAILs a read-only servlet's list add
# Mutation: LDWS2 -- dropping the save search FAILs a module that schedules BStation.save after its writes
# Mutation: LDWS3 -- counting an empty persistence: line passes an undeclared module
# Mutation: LDWS4 -- dropping the no -ux ErrEmptyRoot reports a wrong-path root as a clean exit 0
# The rule's own mutation pins (comments and literals, multi-line arguments, a save in -rt) live in the Go source
# with its golden cases (go/internal/lint/rules/dashboardsave).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-dashboard-write-save: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-dashboard-write-save "$@"
