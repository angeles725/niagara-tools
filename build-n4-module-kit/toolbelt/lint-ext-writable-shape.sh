#!/usr/bin/env bash
# lint-ext-writable-shape.sh — ext-writable-shape lint (Campaign 9 PR10, B823/B826). [ev: retro campaign9-ext-writable-shape]
# Usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>
#
# A COMPLEX property (a type naming StatusNumeric / StatusBoolean / StatusEnum, with or without the B or
# baja: prefix) whose flags value (up to its first "," or ")") holds OPERATOR or "o", and that no
# writing action of its own .java file exempts, is a hazardous external-write target:
#   - a conformant oBIX PUT is rejected ("Cannot translate" — the slot is not BIObixWritable)
#   - a hand-crafted wrapped-<obj> PUT silently writes DEFAULT (0.0 for numeric) on a 200 OK
#     when the "value" child is omitted (the live silent-zero hazard, B823 §823.2)
# The safe external-write paths are:
#   (a) write the oBIX child-leaf bare <real> at …/<slot>/value (B826-G2, LIVE-CONFIRMED)
#   (b) add an OPERATOR @NiagaraAction that writes the slot (B822 additive-action doctrine)
#
# Exemption rule (S22 / C10 D2b — per-slot writing-action check):
#   A slot X is exempt when:
#     (1) The do<Action>() method of some @NiagaraAction <action> in the file writes X by a call
#         named exactly setX( or getX().setValue(, or by .set(x, — only do<Action> bodies count
#         (any class of the file); execute(), changed() and slotomatic setters are not one.
#     (2) An @NiagaraAction name directly encodes the write: name is set<X>/apply<X>/<x>Cmd
#         (B822 naming convention; EW3 positive pin).
#   Replacement for the C9 class-level has_action: an @NiagaraAction that does not write X
#   does NOT exempt X. [ev: corpus B831 §831.2; retro campaign10-ext-writable-per-slot]
#   Comments and literal contents are never read as code. An annotation is read from the line
#   holding its "@" and "(" to the line where its parentheses balance: a second annotation on that
#   line, or one whose "(" is on a later line, is not read.
#
# Row:  WARN  ext-writable-shape  <file>:<line>  <slot>: OPERATOR <type> with no writing action
#             -- external oBIX write must use the child leaf .../<slot>/value (bare <real>, B826)
#             or add an OPERATOR action (B822)
# Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
# Env errors (3) cover a sub-directory find cannot enter (no rows), an unreadable .java file (stderr
# "lint-ext-writable-shape: cannot scan <file>: ...", the other files still scanned) and a root with no
# .java file, which prints the stdout row  ERROR  ext-writable-shape  <dir>  no Java sources found (K20: ...)
# Flags (--strict, --) come first; the first other argument is the root, later ones are ignored.
#
# Dot-directories (D9b) and *-worktrees directories pruned. VCS-free by design (kit-links L2).
# [ev: corpus B823 §823.2]  [ev: corpus B826]  [ev: kit types/logic-authoring.md:62-70]
# Mutation: EW-s22-neg2 -- removes per-slot do-body check, allowing a do<X> writing a different slot to exempt another
# Mutation: EW-s22-nondo -- widens scope past do<Action> bodies, allowing execute() to exempt OPERATOR slots
#
# Implementation: rule package go/internal/lint/rules/extwritable (a per-file lint.Rule) on the shared
# engine go/internal/lint (walker, argument parser, row formatter; go/README.md). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source,
# then this script execs `n4lint lint-ext-writable-shape <args>`. Needs `go` on PATH to build (exit 3 otherwise).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ext-writable-shape: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ext-writable-shape "$@"
