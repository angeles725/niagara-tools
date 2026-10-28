#!/usr/bin/env bash
# lint-charset-default.sh — WARNs text bytes encoded or decoded with the platform default charset.
#
# METHODOLOGY.md § rt (components): text bytes name their charset. The platform default is the host's (cp1252 on a
# Windows station, UTF-8 on a JACE), so code that relies on it writes and reads different bytes per host.
# Rows: CD1 a no-argument getBytes(); CD2 a new String(...) with one or three arguments whose first argument is a
# byte array (a name declared byte[] or byte... in the same file, or a getBytes/toByteArray/readAllBytes call);
# CD3 a one-argument new InputStreamReader / new OutputStreamWriter; CD4 any new FileReader / new FileWriter (Java 8
# has no charset overload for them). Java is read as tokens (go/internal/javasrc): a comment or a string literal never
# matches, and a new String over a char[] or a String is not judged. [ev: corpus B1419 §1419.4 c]
# [ev: retro course-coverage Δ19]
#
# Implementation: rule package go/internal/lint/rules/charsetdefault on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-charset-default <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-charset-default.sh [--strict] <src-root>
#   Row:  WARN  lint-charset-default  <file>:<line>  CD<n> <what> ... (METHODOLOGY.md § rt (components))
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.java
# Mutation: CD1 -- dropping the no-argument check of getBytes WARNs getBytes(StandardCharsets.UTF_8)
# Mutation: CD2 -- judging new String without the byte-array test WARNs new String(char[])
# Mutation: CD3 -- counting every InputStreamReader argument list as charset-less WARNs new InputStreamReader(in, UTF_8)
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-charset-default: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-charset-default "$@"
