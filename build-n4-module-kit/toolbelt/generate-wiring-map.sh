#!/usr/bin/env bash
# generate-wiring-map.sh — Scaffold a wiring-map.md template for a per-instance rearchitect.
#
# Scans @NiagaraProperty annotations in Java files under <facade-src-dir> and emits
# a markdown wiring-map template with three sections:
#
#   Table 1 — OPERATOR config slots (facade → control):
#     Setpoints, HOA modes, defrost settings, differentials that the facade writes
#     into the corresponding rt control slots.
#
#   Table 2 — SUMMARY display slots (control → facade):
#     Temperature readings, statuses, timer anchors that the integrator links
#     FROM the rt control module INTO the facade.
#
#   Table 3 — Per-instance commissioning checklist (BUILD-LOOP §6.b).
#
# Each table row carries the internal slot name, the Workbench display name and a full-ord
# column: the operator searches the live tree by the DISPLAY name, so a link table that names
# only the internal slot has to be redone by hand against Workbench. The display name is filled
# from the module lexicon by the bare "<slot>" key, the only key the framework reads (NSlot.java:42;
# a "<Type>.<slot>" key is never shown); a slot with no key shows "_(no lexicon key)_" — a lexicon gap
# slot-coverage.sh per-slot also reports. [ev: retro panccadia-commissioning-lessons Δ9] [ev: retro devguide415-lexicons-ux-i18n Δ1]
#
# The developer fills in the RT-slot, Full-ord and "Physical instance / crossing notes"
# columns before handing off the module for commissioning. The operator must not hand-derive
# the crossing.
#
# Usage:  generate-wiring-map.sh <facade-src-dir> [--lexicon <module.lexicon>] [--output <file>]
#
#   <facade-src-dir>   Source tree of the facade (-rt) profile.
#   --lexicon <file>   Lexicon for the display-name column. Default: <facade-src-dir>/module.lexicon,
#                      else <facade-src-dir>/../module.lexicon; none found -> placeholder names.
#   --output <file>    Write to <file> instead of stdout.
#
# Exits: 0 ok · 3 usage/env (K20)
#   An unreadable *.java or lexicon is env (3, `generate-wiring-map: cannot read <file>: <reason>`, no map): the
#   bash body skipped it and printed a map short of its slots or display names, exit 0.
#   An --output file that cannot be written (or filled) is env too (3, `cannot write <file>: <reason>`); the bash
#   body printed the shell's redirection error, then "wrote <file>", exit 0.
#
# [ev: retro live-commissioning-verification-gaps Δ6]
# Dot-directories excluded (D9b). VCS-free by design.
#
# Implementation: go/internal/wiringmap (golden runs recorded from the bash body at fa1201e6 by
# go/internal/wiringmap/testdata/record/record.sh). go/go-tool.sh `go_tool_bin` builds go/bin/generate-wiring-map
# when it is missing or older than a Go source, then this script execs it; no Go toolchain or a failed build is exit 3.

set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'generate-wiring-map: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" generate-wiring-map)" || exit 3
exec "$BIN" "$@"
