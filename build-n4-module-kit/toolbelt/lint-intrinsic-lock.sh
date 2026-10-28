#!/usr/bin/env bash
# lint-intrinsic-lock.sh — WARNs locks on a Niagara object's own monitor, and nested locks.
#
# types/logic.md § Tridium rt idioms (mutual exclusion): the framework takes a component's own monitor (every property
# set runs synchronized (this.instance) in the slot map; BComponent has synchronized methods), so a module that holds
# it while waiting on another thread can deadlock. Lock a private final Object, prefer Atomic*, never nest locks.
# A Niagara class is a named class whose own or superclass name is B + an upper-case letter (Buffer is not).
# Rows: IL1 a synchronized instance method of a Niagara class; IL2 synchronized (this) or (<Class>.this) whose this is
# a Niagara class (an anonymous class's this is not judged); IL3 a synchronized block inside another synchronized
# block or method, in any class. A static synchronized method is not judged. Java is read as tokens
# (go/internal/javasrc): a comment or a string never matches. Advisory: Tridium's own source does all three (a smoke
# on the organized tree gives 1827 IL1, 253 IL2, 579 IL3 rows), so WARN, never FAIL. [ev: corpus B1242 §1242.7]
# [ev: corpus B1238 §1238.7] [ev: retro course-coverage Δ24]
#
# Implementation: rule package go/internal/lint/rules/intrinsiclock on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-intrinsic-lock <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-intrinsic-lock.sh [--strict] <src-root>
#   Row:  WARN  lint-intrinsic-lock  <file>:<line>  IL<n> <what> ... (types/logic.md § Tridium rt idioms)
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.java
# Mutation: IL1 -- dropping the Niagara-class test WARNs a synchronized method of a plain helper class
# Mutation: IL2 -- judging every synchronized block as this WARNs synchronized (lock) on a private final Object
# Mutation: IL3 -- dropping the method-body regions misses a block nested in a synchronized method
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-intrinsic-lock: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-intrinsic-lock "$@"
