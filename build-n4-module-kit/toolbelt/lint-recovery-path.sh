#!/usr/bin/env bash
# lint-recovery-path.sh — Protection-output recovery-path gate (Wave 3, LR1, B860).
#
# Detects the "stranded-heater" shape: a protection or heat output slot
# (*Out, *Resistance, *Heat, *Resistencia) is written OFF/safe ONLY inside a
# mode-gated method and has NO unconditional reset on the started()/atSteadyState()/
# enable()/start() path.
#
# Real commissioning defect (PANCCADIA 2026-09): resistanceOut was turned off inside
# execute() guarded by `if (inDefrost) return;` but started() never reset it
# unconditionally — when the mode cleared, the heater stayed off silently.
#
# Usage:  lint-recovery-path.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir>.
#   Emits FAIL rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#
# Row format:  FAIL  lint-recovery-path  <file>:<line>  LRP: <slot> guarded-only in <method> -- no reset in started/atSteadyState/enable
# Exits:       0 no FAIL · 1 any FAIL · 3 usage/env, an unscannable file, an unreadable sub-directory or no *.java (K20)
#
# Dot-directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Documented limitation (grep-level heuristic):
#   False negatives are possible when the unconditional reset occurs in a helper method
#   that is called from started() but not itself named started/atSteadyState/enable/start.
#   Cross-method call chains cannot be followed at this level of analysis. Default posture
#   is FAIL (hard-safety): an undetected reset produces a false FAIL; an undetected missing
#   reset is a silent production defect. Accept the false FAIL and add the unconditional
#   reset inline in started() or suppress with a documented comment.
#
# Source of classification:
#   UNCONDITIONAL: getX().setValue(false) or setX(false) inside a method named
#     started | atSteadyState | changed | enable | start
#   GUARDED:       same write inside any other method
#   FAIL when:     slot has GUARDED write AND no UNCONDITIONAL write in this file.
#
# Implementation: rule package go/internal/lint/rules/recoverypath on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-recovery-path <args>`. Needs `go` on PATH to build (exit 3 otherwise). No flag is parsed and
# only the first argument is read, as before (`--strict` is a root, not a directory).
# Declared difference from the bash lint (port 2026-10-24): a listed file the scan cannot read is
# `lint-recovery-path: cannot scan <path>: ...` on stderr and exit 3 (gawk printed its fatal and the lint exited
# by the other rows, 0 when they were clean).
#
# Mutation: LRP-noguard -- removes the guarded-only classification so mode-gated-only safe-offs pass instead of FAIL
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-recovery-path: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-recovery-path "$@"
