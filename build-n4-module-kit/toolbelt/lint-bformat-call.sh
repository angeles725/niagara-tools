#!/usr/bin/env bash
# lint-bformat-call.sh — flags BFormat special calls written in a form BFormat never parses: %time(<arg>)% and
# %user% fall through to reflection and print %err:...% instead of a time or a user name.
#
# BFormat matches time() and user() only as the exact calls "time()" and "user()" (S/util/BFormat.java:443-444):
# time() takes NO argument and returns a BAbsTime to chain (%time().toDateString%), and %user% without the
# parentheses is a reflective call that fails. See types/logic.md § BFormat pattern reference.
# [ev: retro devguide415-r2-runtime-contracts-formats Δ6] [ev: corpus B1317 §1317.3]
#
# Rules (Java: // and /* */ comments blanked first; other files read as-is):
#   BFC1 time-arg — `%time(` followed by anything but `)`
#   BFC2 user     — `%user%`
# Scope: *.java, *.lexicon, *.bog (plain-XML bog), *.palette — where display formats and facets live.
#
# Usage:  lint-bformat-call.sh [--strict] <src-root>
#   Row:  WARN  lint-bformat-call  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with none of those files is an env error (exit 3) with
# `ERROR  lint-bformat-call  <root>  no files to scan (K20: wrong path or empty scaffold)` on stdout.
# Implementation: rule package go/internal/lint/rules/bformatcall on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-bformat-call
# <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design (kit-links L2).
# Mutation: BF1 -- dropping the time-argument rule passes %time(HH:mm)% clean
# Mutation: BF2 -- dropping the bare-user rule passes %user% clean
# Mutation: BF-ok -- dropping the not-`)` guard WARNs the valid %time()% and %time().toDateString%
# Mutation: BF-comment -- not blanking Java comments reports a pattern quoted in a comment
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bformat-call: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-bformat-call "$@"
