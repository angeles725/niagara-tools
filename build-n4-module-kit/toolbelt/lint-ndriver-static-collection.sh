#!/usr/bin/env bash
# lint-ndriver-static-collection.sh — WARNs a static collection field in a driver network, device or point-device-ext class.
#
# Every station component of a driver class shares one static field: a static Set of subscribed points in a device
# class (the course reference, BTrafficLightDriverDevice) mixes every device's subscriptions. Keep the collection
# per instance (types/driver-authoring.md § 1.7). A class qualifies when its superclass is BNDevice, BNNetwork,
# BDevice, BDeviceNetwork or BNPointDeviceExt, or a class of the same module root that qualifies. A row is a
# static field (final or not) of a qualifying class whose declared type is Set, Map, List, Collection, Queue,
# Deque or a java.util / java.util.concurrent implementation of one, unless its initializer starts with
# Collections.unmodifiable… or Collections.empty…. Scans every *.java under a -rt part's src/ of each module root
# (srcTest/, test/ and build/ are not read); Java is read as tokens (go/internal/javasrc), so a commented-out or
# quoted field is never one. [ev: retro n4-course-handson-ndriver Δ7] [ev: retro n4-course-drivers-bql-ords-files Δ3]
#
# Implementation: rule package go/internal/lint/rules/ndriverstaticcollection on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-ndriver-static-collection <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-ndriver-static-collection.sh [--strict] <module-root>...
#   Row:  WARN  lint-ndriver-static-collection  <file>:<line>  static <type> <name> is shared by every instance -- keep it per instance
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no -rt src/ *.java
# Mutation: NSC1 -- not following module-local superclasses passes a static List in a subclass of a local device base
# Mutation: NSC2 -- dropping the unmodifiable/empty initializer exemption WARNs a Collections.unmodifiableSet constant
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-ndriver-static-collection: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-ndriver-static-collection "$@"
