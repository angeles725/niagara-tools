#!/usr/bin/env bash
# lint-ws-placement.sh — advisory: a method that adds a freshly created component to a parent and never gives it
# a wire-sheet position (`wsAnnotation`).
#
# A component added from code with no `wsAnnotation` slot gets Workbench's in-memory diagonal stagger (+4, wrap to
# 2, width 8), so every sibling a sync action or a seeding loop creates lands on the same diagonal and overlaps
# (WsController.java:398-405). Workbench itself adds the slot only when absent, with flags 1 = Flags.READONLY,
# and otherwise sets it (WsController.java:189-200). Fix: a grid position from
# templates/ws-placement/src/com/angeles/wsplacement/WsGrid.java, added if absent with Flags.READONLY, never
# re-placed. Doctrine: types/logic-authoring.md §Wire-sheet placement, METHODOLOGY.md § rt (components).
# [ev: corpus B1172] [ev: retro ws-placement]
#
# Structural rules (comments blanked by go/internal/javasrc Strip, the mb_strip port; scope = one method from javasrc
# Methods, the mb_parse port):
#   WSP1 an `add(` call (bare `add(` or `<recv>.add(`; `addAll(`, `addLink(` … never match) with AT LEAST TWO
#        top-level arguments whose first argument is not a numeric literal (List.add(int, E) is not a slot add),
#        and whose second argument is a new component:
#          - `new B<Upper>…(` — except the value types that never get a glyph (BStatus*, BWsAnnotation, BFacets,
#            BOrd, BLink, BConversionLink, BAbsTime, BRelTime, BString, BDouble, BFloat, BInteger, BLong,
#            BBoolean, BEnumRange, BFormat, BIcon, BNameMap);
#          - `[<recv>.]create(` / `newInstance(` / `newCopy(`;
#          - a local assigned from one of the above anywhere in the same method (`BFoo f = new BFoo();`, a cast
#            `= (BComponent) create(t)` counts).
#        A leading cast on the argument is ignored. The arguments may wrap over up to 5 lines of the method.
#   Guard: the method mentions `wsAnnotation`, `BWsAnnotation` or `WsGrid` on a code line (a string literal
#        counts, a comment does not). The guard is per method: a sibling method that places nodes does not
#        silence this one.
#   One row per `add(` line. False negatives, never false WARNs on these shapes: a component carried through a
#   field, a return value or a helper method; `getInstance()`; a signature wrapped before `{` (mb_parse does not
#   see that method); a placement delegated to a helper whose name is none of the three guard words (the row
#   then shows and a reviewer confirms — it is WARN).
#
# Usage:  lint-ws-placement.sh [--strict] <src-root>...
#   Row:  WARN  lint-ws-placement  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source
#         file or a sub-directory find cannot enter
# Empty root: a root with no *.java is `ERROR  lint-ws-placement  <root>  no files to scan (K20: wrong path or empty
# scaffold)` on stdout, exit 3.
#
# Implementation: rule package go/internal/lint/rules/wsplacement on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-ws-placement <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# Declared difference from the bash lint (port 2026-10-24): an empty root among several stops the scan at its
# K20 row (the engine contract); the bash lint went on to the later roots. The exit code (3) is the same.
# VCS-free by design (kit-links L2).
# Mutation: WSP1 -- dropping the creation rule passes parent.add(name, create(type)) clean
# Mutation: WSP1-local -- dropping the local tracking passes `BFoo f = new BFoo(); add("f", f);` clean
# Mutation: WSP-guard -- dropping the wsAnnotation guard WARNs a method that already places its children
# Mutation: WSP-scope -- a file-wide guard lets a sibling method's wsAnnotation silence an unplaced add
# Mutation: WSP-list -- reading the last argument of a one-argument add WARNs list.add(new BFoo())
# Mutation: WSP-value -- dropping the value-type exclusion WARNs add("st", new BStatusNumeric())
# Mutation: WSP-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: WSP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ws-placement: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ws-placement "$@"
