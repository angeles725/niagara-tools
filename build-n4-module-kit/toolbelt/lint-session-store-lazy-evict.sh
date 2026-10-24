#!/usr/bin/env bash
# lint-session-store-lazy-evict.sh — flags a STATIC session/token store whose eviction is
# only lazy (on query) or explicit-logout, with no sweep in the inserting method and no
# scheduled purge (a Clock.schedule* action whose do<Action>()/<action>() body removes from it).
#
# Shape (UXS7, B1160): a `static Map<String,*>` field (B1160: a `-ux` class) inserts on login
# (`.put(`) and removes ONLY via an explicit logout method or a query-time expiry check --
# never a sweep in the SAME method that inserts, and never a Clock.schedule-driven purge.
# Being STATIC, the map survives servlet re-mount, so a reused container id (JSESSIONID) can
# inherit an authenticated write session across a stop/restart cycle -- a session-fixation
# shape, not (primarily) an unbounded-growth leak (a session that never logs out is
# negligible heap). Rule: sweep-on-insert (or a scheduled/size-bounded eviction); question
# `static`; bind auth to user+issue-time, not the bare container id; audit the write. See
# types/issues-and-gotchas.md §F2. [ev: retro live-diagnosis-hardening-deltas Δ5]
# [ev: corpus B1160; DashboardConfigSession.java:31-126]
#
# Implementation: go/internal/lint/rules/sessionstore on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-session-store-lazy-evict <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3
# otherwise). An unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as
# awk_lint_run did).
#
# Usage:  lint-session-store-lazy-evict.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot, *-worktrees and srcTest sub-directories pruned) on ANY
#   module profile, not only -ux: B1160's store lives in a -ux class, but a static store behind an
#   -rt servlet has the same shape. srcTest is pruned because the rule guards production session
#   stores and test sources never ship in the module jar; a root that IS a srcTest dir is scanned.
#   Candidate: a class-level `static` field of a (java.util.) Map type -- Map, HashMap,
#   ConcurrentHashMap, LinkedHashMap, TreeMap, ... with generics nested up to three levels --
#   whose name is followed by `=` or `;` on the same line; a method parameter, a return type, a
#   local and a declaration in a comment are not candidates, and each name counts once. Every
#   match reads comment- and literal-blanked code (a put, sweep or schedule in a comment or a
#   string is not one; a // inside a string cuts nothing). A field
#   with a `.put(` only inside `static { }` is a constant table (no row). Every method that calls
#   `<field>.put(` (the name as a whole identifier) is checked and WARNs at its first put unless
#   EITHER (a) that same method sweeps the map -- `<field>.remove(`, a `removeIf(`/`remove(` on
#   its keySet()/values()/entrySet(), or an `iterator()` over it with a no-argument `.remove()`
#   (sweep-on-insert) -- or (b) a `Clock.schedule*` call names an action (one hop: the method
#   declared as `do<Action>()` or `<action>()`) whose body sweeps it (scheduled purge). A sweep
#   reachable only from an unrelated method (explicit logout, or a query-time expiry check) does
#   NOT count as a guard -- that is exactly the shape this lint flags.
#   Row:  WARN  lint-session-store-lazy-evict  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: SSL2 -- drop the `static` requirement so an instance-scope map (e.g. ConfigSession)
# false-WARNs
# Mutation: SSL5 -- file-wide co-occurrence (any Clock.schedule + any remove) hides the logout-only shape
# Mutation: SSL7 -- a probe without the return-type anchor takes `if (sweep(id)) {` for the declaration and its remove() for a purge
# Mutation: SSL-param -- taking any line with `static` and a Map<...> name as a field flags a static method's Map parameter
# Mutation: SSL-everyput -- checking only the file's first put (a substring match) reports OLD_SESSIONS's put as SESSIONS and misses relogin
# Mutation: SSL-literal -- matching the raw line counts a sweep spelled in a logged string as an eviction
# Mutation: SSL-srctest -- scanning srcTest reports a test fixture's static map as a production session store
# Mutation: SSL-awkfail -- an unreadable source file read as empty (internal/lint Run) reports it clean
# Mutation: SSL-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-session-store-lazy-evict: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-session-store-lazy-evict "$@"
