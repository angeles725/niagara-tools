#!/usr/bin/env bash
# lint-license-isoperational-gate.sh — flags a LICENSED class (getLicenseFeature() returns a feature) whose
# changed() / action (timer) / servlet-write callback acts without an isOperational() / isFault() gate.
#
# The framework's license check is advisory: BAbstractService.checkLicense() sets a fatal fault and logs SEVERE on an
# unlicensed feature but does not throw, and ServiceManager runs serviceStarted() right after with no fault check, so
# a feature.check() in serviceStarted() only guards STARTUP. Later callbacks keep firing under a fault icon unless each
# one gates on isOperational() (= !isFatalFault() && !isDisabled() && !isFault()), the way the shipped services do by
# convention. See types/security.md §5.1. [ev: retro secure-authoring-isoperational-gate Δ1]
# [ev: corpus B1143] [ev: corpus B1145] [ev: corpus B1146]
# [ev: code javax/baja/sys/BAbstractService.java:87-90,148-150,152-190] [ev: code com/tridium/sys/service/ServiceManager.java:297-298]
#
# Structural rules (read on code: comments blanked and literal contents removed by the shared scanner
# go/internal/javasrc, so nothing quoted in a string counts; methods from javasrc Methods, the port of mb_parse):
#   Licensed  — the file declares `Feature getLicenseFeature()` and its body is not just `return null;` (the
#               BAbstractService default). A file with no declaration is out of scope.
#   Callback  — a method named `changed` (the property callback), `do<Upper>…` (an action: Clock.schedule timers and
#               operator actions both land in one) or a servlet write handler doPost/doPut/doDelete/doPatch; the
#               read handlers doGet/doHead/doOptions/doTrace are excluded.
#   Acts      — the callback body has a statement other than a `super.<name>(…);` call or a bare `return;`.
#   Gated     — the callback's FIRST statement (after any super.<name>(…); call) calls isOperational(, isFault( or
#               isFatalFault( unqualified or on `this.`: a check after the acting statements, or another object's
#               x.isOperational(), does not gate this class. A gate in a helper the callback calls is NOT followed
#               (one row the reviewer clears by moving the gate up — WARN, never FAIL).
#   Overlap   — none with lint-status-parity (config/status slot ratio) or lint-silent-protection (a protection trip
#               with no operator surface): neither reads getLicenseFeature() or the operational gate.
#
# Implementation: rule package go/internal/lint/rules/licensegate on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-license-isoperational-gate <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-license-isoperational-gate.sh [--strict] <src-root>
#   Row:  WARN  lint-license-isoperational-gate  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-license-isoperational-gate  <root>  no files to scan (K20: wrong path or empty scaffold)`.
# Declared difference from the bash lint (truth fix 2026-10-24): the bash read mb_strip's text, literal contents
# kept, so a quoted "isOperational()" gated a callback, a quoted getLicenseFeature() declaration licensed a class
# and a ";" inside a string cut the first statement before its gate. No real-tree row changed (Cliente 0,
# sources/decompiled 31).
# VCS-free by design (kit-links L2).
# Mutation: LIG-literal -- reading literal text as code passes a callback whose only "gate" is log("isOperational()")
# Mutation: LIG1 -- dropping the gate check WARNs a callback that returns early on !isOperational()
# Mutation: LIG2 -- treating `return null;` as a license feature WARNs an unlicensed service
# Mutation: LIG3 -- counting a super.changed() call as acting WARNs a callback that only delegates
# Mutation: LIG8 -- accepting a gate anywhere in the body passes a callback that acts before it checks
# Mutation: LIG9 -- accepting a qualified call passes child.isOperational() as this service's gate
# Mutation: LIG-awkfail -- ignoring an unreadable file (internal/lint Run) reports it as clean
# Mutation: LIG-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-license-isoperational-gate: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-license-isoperational-gate "$@"
