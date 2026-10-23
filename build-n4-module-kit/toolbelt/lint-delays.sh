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
exec "$BIN" lint-delays "$@"
