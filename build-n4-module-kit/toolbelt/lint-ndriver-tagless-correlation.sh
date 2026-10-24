#!/usr/bin/env bash
# lint-ndriver-tagless-correlation.sh — WARNs an NDriver module whose messages have no getTag() while more than one request can be outstanding.
#
# NComm matches a response to its request by NMessage.getTag(). With no override every message carries the same
# tag, so when two requests are outstanding (BCommConfig.getMaxOutstandingTransactions() defaults to 32) a response
# can complete the wrong request. Fix: override getTag() with the protocol's transaction id, or make the comm
# config's getMaxOutstandingTransactions() return 1 (the driver wizard leaves that line commented out;
# types/driver-authoring.md § 1.7). Facts, per module root, from every *.java under a -rt part's src/ (srcTest/,
# test/ and build/ are not read; superclass chains followed through the root's classes): the NMessage subclasses,
# any getTag() declared in one, and a getMaxOutstandingTransactions() whose body is `return 1;` in a subclass of
# BCommConfig, BTcpCommConfig, BUdpCommConfig, BSerialCommConfig or BHttpCommConfig. One row per module, on the
# first NMessage subclass, when messages exist and neither fix is there. Java is read as tokens
# (go/internal/javasrc), so a commented-out override is not one. [ev: retro n4-course-drivers-bql-ords-files Δ3]
#
# Implementation: rule package go/internal/lint/rules/ndrivertaglesscorrelation on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-ndriver-tagless-correlation <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-ndriver-tagless-correlation.sh [--strict] <module-root>...
#   Row:  WARN  lint-ndriver-tagless-correlation  <file>:<line>  no getTag() override and getMaxOutstandingTransactions() is not 1 -- a response can complete the wrong request
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no -rt src/ *.java
# Mutation: NTC1 -- reading the commented-out `return 1;` hint as code passes the fixture module
# Mutation: NTC2 -- accepting any return value (not only 1) passes a comm config returning 4
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ndriver-tagless-correlation: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ndriver-tagless-correlation "$@"
