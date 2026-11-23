#!/usr/bin/env bash
# lint-gradle-userhome.sh — WARNs a gradle.properties that sets the build's user.home through a setting a running daemon never reads.
#
# An already-running gradle daemon keeps the JVM it was forked with, so `org.gradle.jvmargs=-Duser.home=…` and
# `systemProp.user.home=…` do not move the home of the build (the niagara-signing plugin still reads the real
# ~/.tridium keystore). `toolbelt/build.sh` mirror mode isolates it the working way: JAVA_TOOL_OPTIONS plus --no-daemon.
# Rows: GU1 an uncommented org.gradle.jvmargs whose value holds -Duser.home=; GU2 an uncommented systemProp.user.home
# key. Read as java.util.Properties reads a file: # and ! comments, backslash continuations, `=`, `:` or blank between
# key and value. Only gradle.properties files are read; a doc or script that mentions the flag is not judged.
# [ev: retro r8-mirror-userhome]
#
# Implementation: rule package go/internal/lint/rules/gradleuserhome on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-gradle-userhome <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-gradle-userhome.sh [--strict] <root>
#   Row:  WARN  lint-gradle-userhome  <file>:<line>  GU<n> <what> ...
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no gradle.properties
# Mutation: GU1 -- continuing a comment line that ends in a backslash hides a live jvmargs WARN on the next line
# Mutation: GU2 -- matching -Duser.home= in any property's value WARNs a jvmargs-free key that merely mentions it
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-gradle-userhome: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-gradle-userhome "$@"
