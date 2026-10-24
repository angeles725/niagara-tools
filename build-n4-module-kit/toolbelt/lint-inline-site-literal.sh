#!/usr/bin/env bash
# lint-inline-site-literal.sh — flags site or station data (an ORD, an IPv4 address, a site name) inline outside the site table (G42).
#
# Room counts, ORDs, IPs, point names and per-site defaults sit in one named table (a constants class, a config
# slot set or a JSON/CSV resource) that control and render code read; a literal copied into logic makes the module
# unusable for a second site. Contract: the deferred G42 spec in ledger.d/2026-10-22-modular-line-efficient-quality.md.
# Doctrine: METHODOLOGY.md § Domain correctness "Site and station data live in one data table"; facade side
# types/dashboard.md § Facade contract. [ev: retro modular-line-efficient-quality Δ2] [ev: retro go-lint-inline-site-literal Δ1]
# Advisory WARN: exit 0, 1 under --strict.
#
# Files: every *.java under a -rt or -ux part's src/ and every *.js under an rc/ directory of the module root;
# srcTest/ (test sources), build/, node_modules/, dot and *-worktrees directories are not entered. Java literals
# come from the shared scanner (go/internal/javasrc: comments out; "..." strings, not chars or text blocks),
# JavaScript literals from go/internal/jssrc (strings and template literals; comments and regex literals are not).
# Site table: every file --site-table names (repeatable; a missing file is exit 3) and every file whose first line
# starts with `// site-table`; its literals are never rows.
# A literal is a row (the first one of its line) when it is
#   ord   a slot ORD below the station root (`slot:/<path>`, alone or as one part: `station:|slot:/Rooms/Room1`)
#         or a handle ORD (`h:2e4f`, `station:|h:2e4f`); a query ORD naming no slot below the root
#         (`station:|alarm:|bql:...`, `station:|slot:/|bql:...`) is not site data, nor is `h:mm`;
#   ip    it holds an IPv4 address (four 0-255 octets not inside a longer dotted number);
#   site  it matches a --site-pattern <regex> (repeatable, Go RE2 syntax; a bad regex is exit 3), e.g. 'Room[0-9]+'.
# Room counts (bare numbers) are not decidable and stay a human review item. The literal is shown up to 72 bytes.
# Declared differences from the spec: the check column is the lint ID (kit lint-<name> convention), the row adds
# `: "<literal>"` after the spec's text, and the query-ORD exemption above (retro go-lint-inline-site-literal Δ1).
# Not in n4lint --all (a module root and options that exist only alone: Spec.ModuleRoot).
#
# Implementation: rule package go/internal/lint/rules/inlinesiteliteral on the shared engine go/internal/lint
# (walker, Mode Valued parser, row formatter; go/README.md). go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-inline-site-literal <args>`. Needs
# `go` on PATH to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  lint-inline-site-literal.sh [--strict] [--site-table <path>]... [--site-pattern <regex>]... <module-root>
#   Row:  WARN  lint-inline-site-literal  <file>:<line>  <ord|ip|site> literal outside the site table: "<literal>"
#   Exit: 0  no row, or WARN rows without --strict · 1  a WARN row under --strict · 3  usage/env, an empty root, a root with no part src/ or rc/ file, a missing --site-table, a bad --site-pattern or a sub-directory find cannot enter
# Empty root: a root with nothing to scan is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-inline-site-literal  <root>  no *.java under a -rt or -ux part's src/ and no *.js under an rc/ (wrong path: pass the module root)`.
# Mutation: ISL1 -- dropping the ORD match passes BOrd.make("station:|slot:/Rooms/Room1") in a logic class clean
# Mutation: ISL2 -- dropping the site-table check flags the same ORD in the --site-table class
# Mutation: ISL3 -- dropping the IPv4 match passes "192.168.1.20" in a servlet clean
# Mutation: ISL4 -- dropping --site-pattern passes "Room3" in rc/app.js clean
# Mutation: ISL-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
# Mutation: ISL-K20 -- dropping the no-part ErrEmptyRoot reports a wrong-path root as a clean exit 0
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-inline-site-literal: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-inline-site-literal "$@"
