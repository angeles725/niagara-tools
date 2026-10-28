#!/usr/bin/env bash
# lint-lexicon-apostrophe.sh — WARNs a lexicon value whose single apostrophe MessageFormat would drop.
#
# The lint-lexicon-ascii.sh companion. types/lexicon.md § 3: getText(key, args...) formats the value with
# java.text.MessageFormat, which reads a single ' as a quote: "doesn't {0}" renders "doesnt ...", and a second lone '
# swallows the text between. A literal apostrophe in an entry read with arguments is written ''.
# Row LX1: a value (Java .properties: # and ! comments, \ continuation lines, key ends at the first unescaped =, :
# or blank) with a MessageFormat argument ({ then a digit) and a ' that is neither '' nor a deliberate quote of a
# blank-free text holding a brace ('{'). A value without arguments is not judged: read with getText(key) it is never
# formatted. Every *.lexicon under <module-root> (dot and *-worktrees directories pruned); a root without one is
# clean. A smoke on the Tridium organized tree gives 409 rows (French l', "doesn't"), real by the rule.
# [ev: corpus B1242 §1242.7] [ev: corpus B1236 §1236.4] [ev: retro course-coverage Δ26]
#
# Implementation: rule package go/internal/lint/rules/lexiconapostrophe on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-lexicon-apostrophe <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-lexicon-apostrophe.sh [--strict] <module-root>
#   Row:  WARN  lint-lexicon-apostrophe  <file>:<line>  LX1 <key> has a MessageFormat argument ... (types/lexicon.md § 3)
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env
# Mutation: LX1 -- treating a doubled '' as a lone apostrophe WARNs "can''t {0}"
# Mutation: LX1-noargs -- dropping the argument test WARNs a value with no {n} read by getText(key)
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-lexicon-apostrophe: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-lexicon-apostrophe "$@"
