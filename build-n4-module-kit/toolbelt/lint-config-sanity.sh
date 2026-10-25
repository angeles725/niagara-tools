#!/usr/bin/env bash
# lint-config-sanity.sh — Unsafe @NiagaraProperty default patterns gate (Wave 3, LR3, B862).
#
# Detects four unsafe default patterns in @NiagaraProperty annotations:
#
#   CS1 FAIL: *Interval default <= *Duration default in the same file, both read in MILLISECONDS from
#             any literal BRelTime factory (make(ms), makeSeconds/Minutes/Hours/Days, make(d,h,m,s)) or
#             constant (BRelTime.java:34-48, :56-83). A BRelTime default it cannot read (a variable, an
#             expression) is a CS1 WARN "unreadable" row, never a silent skip. A pair where BOTH are 0
#             (BRelTime.DEFAULT on a display mirror) is unset, not a contradiction. [ev: audit-2026-10-03 A1]
#             Physical impossibility: the cycle interval must exceed the defrost/heat duration.
#             A too-small interval means "interval already elapsed" fires on the very first execute.
#
#   CS2 FAIL: *Setpoint or *Set slot with Flags.OPERATOR and default 0 (BDouble.make(0.0) or
#             BDouble.make(0)) and no min>0 facet (BFacets.MIN, BDouble.make(<positive>)).
#             A zero setpoint on a cooling unit runs at ambient temperature, silently.
#
#   CS3 WARN: A line holding @NiagaraProperty (any slot type) immediately preceded by a // comment line
#             containing "only when", "only valid" or "only if" (O or o) — a comment-only enforcement
#             rule that has no runtime gate. See documented limitation below.
#
#   CS4 WARN: a nonzero permanent-minimum floor default (...MinStagesOn / ...MinOn name on a
#             camelCase boundary, NOT a BRelTime or *Time/*Delay/*Sec short-cycle timer) coexists in
#             the same file with a *LowLimit* / *Cutout* (camelCase token, unit suffix allowed, timer and
#             status/counter suffix excluded) numeric default of 0 (= protection disabled). Either default alone is fine; together one
#             unit is held on with no LP cutout ("pulling with every solenoid closed").
#             [ev: retro panccadia-commissioning-lessons Δ3]
#
# Usage:  lint-config-sanity.sh <java-src-dir>
#
#   Scans all *.java recursively under <java-src-dir> (only the first argument is read; no flags).
#   Emits FAIL (CS1/CS2) or WARN (CS1 "unreadable", CS3/CS4) rows; exits 1 on any FAIL, 0 clean, 3 usage/env.
#   The checks run in Go: n4lint lint-config-sanity (go/internal/lint/rules/configsanity).
#
# Row format:  FAIL|WARN  lint-config-sanity  <file>:<line>  CS<n>: <reason>
# Exits:       0 no FAIL (WARN-only is 0) · 1 any FAIL · 3 usage/env, incl. a sub-directory find cannot enter (K20)
#              or a source file it cannot read ("cannot scan"; the other files' rows still print)
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-config-sanity  <root>  no *.java sources found (wrong path or empty source tree)` (go/internal/lint
# Spec.EmptyError; declared difference 2026-10-23: the bash lint exited 0 with no row).
#
# Dot-directories and *-worktrees directories excluded (D9b). VCS-free by design. kit-links.bats L2 enforces
# the no-version-control rule on all toolbelt scripts.
# [ev: retro live-commissioning-verification-gaps]
#
# Source of defaults: only the Java @NiagaraProperty annotation, read from the raw source (comments are
#   not stripped) by CS1/CS4, which read its defaultValue= literal. CS2 reads it with comments blanked
#   (literals kept): OPERATOR and a BDouble.make(0) in a comment inside the annotation ("never
#   OPERATOR-writable") count for nothing (issue #459). When
#   that is absent or unreadable the check is skipped (never false-FAIL) — except a BRelTime default on
#   an *Interval/*Duration slot, which is a CS1 "unreadable" WARN row.
#
# Annotation reader: from the @NiagaraProperty line, lines are joined until the parentheses balance (CS1/CS2
#   also wait for the first "(", so "@NiagaraProperty" alone on a line takes the next one). A one-line
#   annotation closes on its own line, so the annotation on the line right after it is read too. An
#   annotation still open at end of file is not read.
#
# Documented limitation — CS3 (comment-only flag enforcement):
#   The heuristic matches a comment containing "only when", "only valid", or "only if"
#   on the line immediately preceding a @NiagaraProperty declaration. This is intentionally
#   conservative (WARN, not FAIL) because:
#   (a) The pattern is grep-level; false positives are possible on unrelated comments.
#   (b) No runtime AST is available to confirm the enforcement gap.
#   Operators must review CS3 WARN rows manually and add runtime validation. There is no
#   suppression marker: a CS3 row stays until the comment or the runtime gate changes.
#
# Mutation: LCS-interval -- removes interval<=duration comparison so CS1 shape passes instead of FAIL
# Mutation: LCS-floor -- dropping the floor x disabled-cutout pairing lets the CS4 shape pass silently
# Mutation: LCS-floor-name -- matching minon/cutout as substrings again WARNs on adminOnline / minOnTime / cutoutDelay
# Mutation: LCS-floor-status -- dropping the status/counter suffix exclusion WARNs on cutoutCount / cutoutActive
# Mutation: LCS-floor-trip -- putting Trip|Alarm|Fault back in the status suffixes silences lpCutoutTrip=0
# Mutation: LCS-floor-suffix -- anchoring the cutout token to the name end misses lpCutoutPsi / lowLimitBar
# Mutation: LCS-cs2-comment -- reading CS2 from the raw lines again FAILs the READONLY heatSetpoint whose comment says OPERATOR
# Mutation: CS1-units -- comparing raw factory arguments (makeHours(4) vs make(1800000L)) false-FAILs a sane pair
# Mutation: CS1-mixed -- reading make(ms) and makeSeconds(s) as one unit passes a 30 s cycle with a 60 s defrost
# Mutation: CS1-minutes -- skipping makeMinutes silently passes a 10 min cycle with a 45 min defrost
# Mutation: CS1-wrapped -- anchoring the factory to the whole defaultValue reads make(BRelTime.class, BRelTime.makeSeconds(300)) as unreadable
# Mutation: CS1-bothzero -- comparing a 0/0 (DEFAULT) pair FAILs every display-mirror panel
# Mutation: CS1-unreadable -- dropping the unreadable row skips an unparsed BRelTime default silently
# Mutation: LCS-finderr -- ignoring the find status skips an unreadable sub-directory and reports clean
# Mutation: LCS-awkfail -- skipping a file the scan cannot read exits by the other files' rows (0 or 1), never 3
# Mutation: LCS-oneline -- taking the next line into a one-line annotation hides the annotation on it from CS1/CS2
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-config-sanity: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-config-sanity "$@"
