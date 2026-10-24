#!/usr/bin/env bash
# lint-silent-protection.sh — Silent-protection-trip lint (Campaign 9 S18, B824).
#
# Flags a PROTECTION TRIP that forces an output OFF / sheds a stage with NO
# operator surface: no SUMMARY/OPERATOR status-or-reason slot written on its
# path, no BAlarmSourceExt, no readable reason field surfaced.
#
# Usage:  lint-silent-protection.sh [--strict] <java-src-dir>
#
# Row:  WARN  lint-silent-protection  <file>:<line>  <method> forces <output>/sheds stage on condition
#       -- no status/reason/alarm surface in scope; add a *Alarm/*Reason SUMMARY slot or a BAlarmSourceExt
#
# Surface-name allowlist (SUMMARY/OPERATOR slots whose names signal a surface):
#   *Alarm *Fault *Skip* *Reason *Status *Mismatch *Stuck *Available *Fallback
# Effect-slot exemption: writing the trip own forced output slot is NOT a surface.
# Private-field exemption: a private boolean field is NEVER a surface.
# Exactly ONE WARN per trip site (dedupe on <file>:<line>).
# On condition: a trip action is guarded by an if( on its line or up to 10 lines above it in the method only
# when the action lies inside that if statement (its condition, then or else branch, braced or not); a cap
# below an if block that already closed is unconditional and no trip [ev: issue #459]. The surfaced-field
# index is tree-wide, so a B-facade writing its *Alarm SUMMARY slot from the pure class's field
# (getDischargeHighAlarm().setValue(ctl.dischargeHigh)) surfaces the pure class's if (dischargeHigh) trip.
#
# Console-only advisory: a trip whose ONLY surface is a console alarm (Pattern A BAlarmSourceExt /
# BAlarmRecord, or Pattern B BIAlarmSource adapter) with no local status/reason slot write is not
# silent, but has no surface while the alarm console is unattended. It gets one row
#   ADVISORY  lint-silent-protection  <file>:<line>  console-only: <method> forces <output>/sheds stage
#         -- surfaced only by a console alarm; add a local *Status/*Reason SUMMARY slot (advisory, unattended console)
# Its own ADVISORY severity (not WARN) keeps the WARN contract "a trip with NO surface" intact for
# existing consumers and never sets the --strict exit 1. [ev: retro alarm-console-design Δ3]
#
# Exits: 0 no WARN (or WARN without --strict) | 1 any WARN under --strict | 3 usage/env or a sub-directory find cannot enter
# Env errors (3) also cover an unreadable .java file and a root with no .java file; the latter prints the stdout row
#   ERROR  lint-silent-protection  <dir>  no Java sources found (K20: wrong path or empty scaffold)
# Only the first root is checked; later arguments are ignored. Dot-dirs (D9b) and *-worktrees dirs excluded.
# VCS-free by design.
# [ev: corpus B824]  [ev: retro campaign9-silent-protection]
# Mutation: S23-pos -- removes Pattern-B alarm-adapter exemption, causing false-WARN on alarmed trips
# Mutation: SP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: S23-and -- removes the BIAlarmSource check, allowing bare newOffnormalAlarm to suppress WARN
# Mutation: SP10 -- dropping the if-statement scope lets an if block closed above an unconditional cap guard it
# Mutation: SP-console -- drops the console-only row, so an alarm-only trip loses its local-surface advisory
#
# Implementation: rule package go/internal/lint/rules/silentprotection (a lint.TreeRule: every file's check
# reads the surface-slot, effect-slot, surfaced-field and alarm-class indexes built from the whole tree first)
# on the shared engine go/internal/lint (walker, parser, row formatter; go/README.md). Comments are blanked by
# the shared scanner go/internal/javasrc Strip, which keeps string literals (retro mb-strip-string-literals Δ1).
# This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing
# or older than a Go source, then this script execs `n4lint lint-silent-protection <args>`. Needs `go` on PATH
# to build (exit 3 otherwise).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-silent-protection: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-silent-protection "$@"
