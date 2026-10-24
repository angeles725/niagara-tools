#!/usr/bin/env bash
# lint-inert-coordination.sh — flags a coordinator whose unit list is STRUCTURALLY singleton
# (e.g. `units()` returns `Collections.singletonList(getParent())` or a one-element
# `Arrays.asList(...)`) while the SAME class still declares queue/token/stagger coordination
# state -- the machinery cross-unit-serializes over exactly one unit and is dead code.
#
# Shape (Δ5, PANCCADIA 2026-09-23): PR1c moved `BDefrostController` under each
# `BEvaporatorUnit`. `units()` now returns a single-element list, so the token/queue/
# `staggerDelay` machinery never engages -- a room with two electric resistances can defrost
# both at once, exactly the coordination the class exists to prevent. The class's own comment
# admits it ("staggerDelay is inert: with one unit in the list the waitingQueue is always
# empty") but nothing flags it as a defect. See types/issues-and-gotchas.md §J1 and
# types/logic.md "RT control logic" for the fix (coordinate at the parent that actually
# aggregates multiple units, or drop the dead machinery).
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ5]
# [ev: corpus BDefrostController.units() ~990-1001; staggerDelay ~309-330; waitingQueue ~1060]
#
# Usage:  lint-inert-coordination.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned). Run on any -rt profile with Java sources.
#   Code only: comments blanked and literal contents removed (go/internal/javasrc Strip + BlankLiterals; truth fix
#   T1 2026-10-24, the bash cut each line at its first "//" and read block comments and strings as code). Per file:
#     1. locate a `<ret-type> units()` method declaration and brace-extract its body forward;
#     2. the body must return a STRUCTURALLY singleton list: `singletonList(` anywhere in the
#        body, OR `Arrays.asList(` with no comma between the call's opening paren and the end
#        of its line (single-argument heuristic -- see limitations);
#     3. AND the class (whole file) must declare at least one coordination-state identifier:
#        a `Deque`/`Queue` typed field, or an identifier that starts with `token`/`stagger`
#        or carries `Token`/`Stagger` after its first word (Java camelCase: `waitingQueue`,
#        `acquireDefrostToken`, `staggerDelay`).
#   All true -> WARN at the units() declaration line.
#   Row:  WARN  lint-inert-coordination  <file>:<line>  units() returns a structurally singleton
#         list while <ids> (queue/token/stagger state) are still declared -- the coordination
#         machinery is dead code after the per-unit move; coordinate at the parent that
#         aggregates multiple units, or drop the unused fields
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
#
# Known limitations (advisory heuristic, documented per kit style):
#   - the singleton-return check only looks for `singletonList(`/`Arrays.asList(` code ANYWHERE
#     in the body, not that it is the actual `return` expression -- an unrelated local variable
#     using the same call shape is a false-positive risk, cheap to accept per
#     kit style (mirrors lint-wb-external-ord-value's same-method text-presence checks).
#   - the `Arrays.asList(` single-argument check is a same-line no-comma-to-end-of-line
#     heuristic (no real paren-depth tracking): a multi-line call, or a two-argument call whose
#     first argument itself contains no comma but a later unrelated comma appears further on
#     the SAME line, can mis-classify -- rare in this shape (a coordinator's argument is always
#     `getParent()`/a single component reference).
#   - `Token`/`Stagger` matching follows camelCase (a leading `token`/`stagger` word or a later
#     `Token`/`Stagger` word); an ALL-CAPS constant (`STAGGER_MS`) is a false-negative risk.
#   - a `Deque`/`Queue` field used for something UNRELATED to unit coordination (rare in a
#     control component) would false-positive; only fires when a singleton units() ALSO exists,
#     which narrows the risk considerably.
# Empty root: a root with no *.java is `ERROR  lint-inert-coordination  <root>  no files to scan (K20: wrong path
# or empty scaffold)` on stdout, exit 3.
#
# Implementation: rule package go/internal/lint/rules/inertcoordination on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-inert-coordination <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: ICO2 -- drop the coordination-state (queue/token/stagger) requirement so any singleton units() WARNs even with no dead machinery
# Mutation: ICO-lower -- requiring a capitalized Stagger/Token misses the lowerCamel staggerDelay field
# Mutation: ICO-call -- taking `return units()...;` for the declaration checks the wrong body
# Mutation: ICO-comment -- reading block comments as code WARNs a units() whose singletonList is commented out
# Mutation: ICO-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: ICO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-inert-coordination: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-inert-coordination "$@"
