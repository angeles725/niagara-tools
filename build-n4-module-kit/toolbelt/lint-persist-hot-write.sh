#!/usr/bin/env bash
# lint-persist-hot-write.sh — flags a persisted (non-transient) property SETTER called from
# changed() or any of its one-hop callees with no cadence guard nearby the call site.
#
# Shape (PER8, B1159): a NON-transient @NiagaraProperty slot (declared without Flags.TRANSIENT
# in the raw slot-o-matic form `Property <name> = newProperty(<flags>, ...)`) is correctly
# non-transient (it MUST survive restart, e.g. for fair lead/lag rotation) but its setter is
# called EVERY CYCLE from changed()/execute(), so each call marks the station config dirty and
# fires every link off that slot. Pairs with lint-changed-hot-write.sh (RUN8): that lint flags
# the callback-rate SHAPE (fan-in + no rate guard); this lint flags the SETTER CALL SITE itself,
# independent of branch count, so a single always-executed persisted write still WARNs.
# Rule: integrate in memory; checkpoint the non-transient slot on a periodic tick + in
# stopped(); seed the value once on start (seedHours()-style guard). See
# types/issues-and-gotchas.md §I2. [ev: retro live-diagnosis-hardening-deltas Δ3]
# [ev: corpus B1159; BCompressorControl.java:326,1403,2019]
#
# Usage:  lint-persist-hot-write.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned). Only the raw slot-o-matic declaration
#   `Property <name> = newProperty(<flags>, ...)` is recognized (the @NiagaraProperty
#   annotation form is not parsed -- false-negative, not false-positive, documented
#   limitation). A setter CALL (not its method DECLARATION) inside changed()'s body or ANY
#   zero-arg method changed() calls (one hop, every callee) is a candidate. A cadence guard is a
#   `Clock.millis(` comparison or a `%` (modulo/counter) test within 5 lines above the call,
#   in the SAME method.
#   Row:  WARN  lint-persist-hot-write  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java is `ERROR  lint-persist-hot-write  <root>  no files to scan (K20: wrong path or
# empty scaffold)` on stdout, exit 3.
#
# Implementation: rule package go/internal/lint/rules/persisthotwrite on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md), sharing changed(), bodies, callees and properties with its sibling
# lint through go/internal/lint/rules/hotwrite. This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-persist-hot-write <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# Truth fix T1 (2026-10-24): code is read with comments blanked and literal contents removed (go/internal/javasrc
# Strip + BlankLiterals, shared with the sibling lint): the bash/awk cut each line at its first "//" (inside a
# string too) and read block comments, Javadoc and strings as code, so a Javadoc `{@code execute()}` was taken
# for execute()'s declaration and a "}" in a string ended a body early.
# VCS-free by design (kit-links L2).
# Mutation: PHW7 -- cutting each line at its first "//" hides a setter written after a URL string
# Mutation: PHW2 -- drop the cadence-guard lookback so a guarded setter call still false-WARNs
# Mutation: PHW6 -- keeping only the last callee drops a hot write in an earlier callee
# Mutation: PHW-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: PHW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-persist-hot-write: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-persist-hot-write "$@"
