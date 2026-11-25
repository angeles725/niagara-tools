#!/usr/bin/env bash
# lint-confirm-required.sh — FAILs a @NiagaraAction whose name says it is destructive and whose flags lack
# Flags.CONFIRM_REQUIRED.
#
# Workbench's invoke command and the bajaux invokeAction show a confirmation dialog for an action that carries
# CONFIRM_REQUIRED; without it a fat-finger invoke of a reset, a clear or a wipe runs at once. An action is
# destructive when a whole camelCase word of its name is reset, clear, delete, remove, purge, wipe, factory,
# restart, reboot or overwrite (types/actions.md § 3). The flag is a client prompt, not authorization: who may
# run the action is still the permission rules of types/security.md. Fix: add `flags = Flags.CONFIRM_REQUIRED`
# (`| Flags.OPERATOR` when operators may run it), or waive a reviewed action with
# `// lint-confirm-required: waived <reason>` on the @NiagaraAction line or on a // comment line right above it;
# the reason is mandatory, a bare marker is a FAIL of its own.
# Source is read as the sibling lints read it (go/internal/javasrc, comments stripped, literals one token): an
# annotation split over several lines is one annotation, and text in a comment, a string or a text block is not
# one. [ev: code baja javax/baja/sys/Flags.java:190,445] [ev: code bajaui-wb javax/baja/ui/commands/InvokeActionCommand.java:42-51]
#
# Implementation: rule package go/internal/lint/rules/confirmrequired on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-confirm-required <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-confirm-required.sh [--strict] <src-root>...
#   Row:  FAIL  lint-confirm-required  <file>:<line>  <action>: destructive action without Flags.CONFIRM_REQUIRED -- ...
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no *.java or a sub-directory find cannot enter
# Mutation: CR1 -- not reading the flags attribute (or the name list) lets a destructive action without the flag pass
# Mutation: CR2 -- reading only the first flag of a `|` list FAILs a flagged action that lists CONFIRM_REQUIRED later
# Mutation: CR3 -- reading one line of the annotation FAILs or misses an annotation split over several lines
# Mutation: CR4 -- ignoring the waiver marker FAILs a waived action
# Mutation: CR5 -- accepting a bare marker without a reason lets an unexplained waiver pass
# Mutation: CR6 -- matching a name substring instead of a whole word FAILs presetValues and cleared
# Mutation: CR7 -- reading comment or string text as code FAILs a quoted @NiagaraAction
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-confirm-required: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-confirm-required "$@"
