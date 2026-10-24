#!/usr/bin/env bash
# lint-uberjar-api-conflict.sh — a library must not be BOTH uberjar()'d and api()'d.
#
# Embedding a library with uberjar("group:artifact:ver") AND also declaring it as a compile
# dep with api("group:artifact:ver") puts two copies of the same classes in different
# classloader tiers — a conflict. Pick one: bundle it (uberjar) OR depend on a module that
# provides it (api). See types/module-wiring.md §1. Advisory (WARN).
# [ev: retro 2026-09-19-module-wiring]
#
# Implementation: go/internal/lint/rules/uberjarapi on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-uberjar-api-conflict <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3
# otherwise). An unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as
# awk_lint_list did); an unreadable *.gradle.kts is "cannot scan <file>: <reason>", exit 3 (the bash grep
# skipped it in silence). A root with no *.gradle.kts stays a clean exit 0 (report-module runs it on every
# module root; tests/lint-empty-root.bats ER5).
#
# Usage:  lint-uberjar-api-conflict.sh <src-root>     (finds *.gradle.kts under it)
#   Row:  WARN  lint-uberjar-api-conflict  <file>  <group:artifact> is both uberjar()d and api()d
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: UAC2 -- skip the intersection and UAC2 stops warning on a lib both uberjar'd and api'd
# Mutation: UAC-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: UAC3 -- matching raw text makes a commented or string api() a conflict and misses a split call
# Mutation: UAC4 -- not resolving a val argument misses a conflict whose coordinate is held in a variable
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-uberjar-api-conflict: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-uberjar-api-conflict "$@"
