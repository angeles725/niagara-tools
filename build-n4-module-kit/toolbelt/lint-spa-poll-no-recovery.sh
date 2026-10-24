#!/usr/bin/env bash
# lint-spa-poll-no-recovery.sh — flags an rc/ SPA poll loop (setInterval/setTimeout) whose catch
# path has no recovery keyed on time since the last SUCCESS: either nothing ever reloads after the
# underlying session dies (a station restart kills the web session; the kiosk polls the dead one
# forever), or the reload is driven by a failure COUNT / a bare reload-on-error, which is blind to a
# hung request that neither fails nor succeeds.
#
# Shape 1 (no recovery, 2.4.2): `catch` only marks data stale and repaints "error"; a fixed HMI
# panel stays on dead values until power-cycled. [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1]
# [ev: corpus DashboardPan-ux index.html poll() 2.4.2 -> 2.4.3]
# Shape 2 (failure-count watchdog, 2.4.3..2.8.0): reload after N consecutive failures; recovered ten
# restarts and froze on the eleventh because a fetch without timeout hung and the counter never
# advanced. Doctrine: types/dashboard.md § Poll loop reliability (watchdog on `lastOkAt`, set only by
# a validated success; probe via fetchT; reload on the app marker; hard ceiling).
# [ev: retro dashboard-frontend-reliability-rules Δ3]
#
# Usage:  lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>
#   Scans *.html and *.js under <ux-src-root> (dot-dirs pruned). `//` comments line-stripped. Per file:
#     1. the FIRST `setInterval(<name>, ...)` / `setTimeout(<name>, ...)` call names the poll fn;
#     2. its declaration body -- `function <name>(`, or `const|let|var <name> = [async] function(`
#        / an arrow function -- is brace-extracted; it must contain `catch (`/
#        `.catch(` — no catch, no shape to flag;
#     3. recovery = `location.reload(` or `location.href =` ANYWHERE in the file;
#     4. success-time key = a `lastOk*` identifier assigned from a clock (`Date.now()`,
#        `performance.now()`, `new Date`) INSIDE the poll fn body (the success path; a timestamp
#        set only at load is not a success time) AND read in a subtraction/comparison
#        (`- lastOkAt`, `lastOkAt <`/`>`) ANYWHERE in the file.
#   Rows (WARN at the setInterval/setTimeout call line):
#     no 3        -> "... no location.reload(/location.href recovery anywhere in the file ..."
#     3 but no 4  -> "... recovery is not keyed on time since the last success ..."
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
#
# Known limitations (advisory heuristic): file-wide co-occurrence (a reload button elsewhere plus a
# lastOk* timestamp suppresses the WARN — false-NEGATIVE bias); only the FIRST interval/timeout call
# per file is checked; a success timestamp not named lastOk* (e.g. `lastSuccessAt`) is a false
# positive — rename it or accept the WARN; a success timestamp set in a helper the poll calls
# (`.then(onOk)`) is not seen either — set it in the poll body; naive `//` stripping can cut a `//`
# inside a string/URL.
# Empty root: a root with no *.html/*.js is `ERROR  lint-spa-poll-no-recovery  <root>  no files to scan (K20: wrong
# path or empty scaffold)` on stdout, exit 3 (under `n4lint --all` such a root is clean for this -ux rule).
#
# Implementation: rule package go/internal/lint/rules/spapoll on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-spa-poll-no-recovery <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: SPR2 -- drop the lastOk success-time requirement so a failure-count-only watchdog passes clean
# Mutation: SPR8 -- drop the location.reload(/location.href recovery check so a lastOk timestamp that never reloads passes clean
# Mutation: SPR9 -- count a clock assignment anywhere in the file so a lastOk set only at load passes clean
# Mutation: SPR10 -- matching only `function <name>(` loses an arrow-function poll (silent pass)
# Mutation: SPR11 -- brace-extracting a concise-body arrow poll runs past its statement and reads a later lastOk* as its own
# Mutation: SPR-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: SPR-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-spa-poll-no-recovery: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-spa-poll-no-recovery "$@"
