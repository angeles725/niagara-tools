#!/usr/bin/env bash
# lint-bql-unbounded.sh — WARNs a `bql:` query literal whose select reads every row (FAIL under --strict).
#
# BQL has `top <n>` and no skip/offset, and the station evaluates a select as extent -> where -> aggregate -> distinct
# -> order -> top -> projection, so only a top (or a narrower source) bounds the read; an order by sorts everything
# before top cuts it. Module Java and JavaScript only (*.java, *.js; *.min.js, build/ and node_modules/ are skipped).
# Checks (WARN; each row names one):
#   no-top             no `top <n>` on an ORD that is not history: or alarm: (a tree walk, a table, `history:|bql:`).
#   history-no-window  a history:<path> query with no top and no timestamp window: no ?period=/start=/end= on the ORD
#                      and no timestamp predicate (timestamp >, <, >=, <=, =, in) in the where clause.
#   alarm-unindexed    an alarm: query with no top and none of: `from openAlarms`, `from ackPendingAlarms` (exact case),
#                      `uuid =`, `source =`, a timestamp range.
#   order-by-no-top    a query bounded by a window or an index path that has an `order by` and no top (an unbounded
#                      query with an order by is reported by its own check, and the row says so).
#   waiver-no-reason   FAIL: a `// lint-bql-unbounded:` marker with nothing after the colon; the query is not waived.
# `top` takes a number or a %d, %s, ? or {n} placeholder. A literal added to anything else with + or +=, and a
# JavaScript template with ${}, is built at run time: it is counted in the SUMMARY row (non-literal-skipped), never
# guessed. Comments and Javadoc are no code. Waive one query with `// lint-bql-unbounded: <reason>` on its line or
# the comment-only line above (the reason is mandatory). Aggregates (count(*)) and an `or` in the where clause are
# false positives: waive them. The reading is the Go package go/internal/lint/rules/bqlunbounded (doc comment, goldens).
#
# Evidence ([CERT] unless marked; paths under organized/<module>/<artifact>/vineflower/ unless stated; types/queries.md
# §3 cost table, §4 history recipes, §6 alarm recipes; API survey 2026-11-24 media-cache-db §1.4-5, §5.1, §5.3-5.5):
#   no-top             bql-rt com/tridium/bql/SelectQuery.java:62-87 (the pipeline; only BTopTable stops it) and
#                      com/tridium/bql/collection/BTopTable.java:57-63 (row counter cuts the cursor); tree BQL walks
#                      every component below the base, where filters after: com/tridium/bql/cursor/BogCursor.java:34-48.
#   history-no-window  history-rt com/tridium/history/db/BHistoryDbTable.java:232-262: no predicate = scan (:249-250),
#                      a timestamp range = timeQuery(min, max) (:252-261), nothing caps the rows; ORD parameters period,
#                      start, end, delta: niagara-help/bajadoc-clean/javax/baja/history/HistoryQuery.txt:7-19 [CERT-doc].
#   alarm-unindexed    niagara-help/source/alarm-rt/javax/baja/alarm/AlarmDbConnection.java:52-110 [CERT-doc]
#                      (doBqlQuery: openAlarms and ackPendingAlarms extents compared with equals, no predicate = scan(),
#                      uuid range, one source, timestamp range, else scan()); types/queries.md §6 says the same.
#   order-by-no-top    SelectQuery.java:75-81: Ordering.order(result) materializes the table, then BTopTable cuts it.
# Implementation: that rule package on the shared engine go/internal/lint (go/README.md). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this
# script execs `n4lint lint-bql-unbounded <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-bql-unbounded.sh [--help] [--strict] <src-root>...
#   Rows: WARN  lint-bql-unbounded  <file>:<line>  <check>: "<select>" <detail> (types/queries.md §n)   (FAIL under --strict)
#         FAIL  lint-bql-unbounded  <file>:<line>  waiver-no-reason: lint-bql-unbounded: needs a reason after the colon
#         SUMMARY  lint-bql-unbounded  checked=<n> flagged=<n> waived=<n> non-literal-skipped=<n>   (one per run)
#   Exits:  0  clean or WARN only · 1  any FAIL, or any WARN under --strict · 3  usage/env: no root, an unknown option,
#           a root that is not a directory, a root with no *.java or *.js (stdout gets `ERROR  lint-bql-unbounded  <root>
#           no *.java or *.js sources found ...`), a sub-directory the walk cannot enter or an unreadable file
# --help prints the usage and the rules and exits 0. VCS-free by design (kit-links L2).
# Mutation: BQLU-top -- ignoring `top <n>` WARNs the bounded queries of BClean
# Mutation: BQLU-window -- ignoring ?period= and the timestamp predicate WARNs the windowed history queries
# Mutation: BQLU-index -- matching openAlarms case-insensitively passes `from openalarms`, which the station scans
# Mutation: BQLU-dynamic -- judging a built string reports a query whose top or window is added at run time
# Mutation: BQLU-waiver -- ignoring the marker WARNs the two waived queries of BWaived
# Mutation: BQLU-waiver-bare -- accepting a bare marker waives the alarm query without a stated reason
# Mutation: BQLU-comment -- matching raw text WARNs the queries quoted in a comment or a Javadoc
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
case "${1:-}" in
  --help|-h)
    printf '%s\n' 'usage: lint-bql-unbounded.sh [--help] [--strict] <src-root>...' '' \
      'WARN a bql: query literal (module Java and JavaScript) whose select reads every row; FAIL under --strict.' \
      '  no-top             no top <n> (tree walk, table, history: listing).' \
      '  history-no-window  history:<path> with no top <n> and no ?period=/start=/end= or timestamp predicate.' \
      '  alarm-unindexed    alarm: with no top <n> and no from openAlarms/ackPendingAlarms, uuid =, source =' \
      '                     or timestamp range.' \
      '  order-by-no-top    an order by with no top <n>: it loads every row before any cap.' \
      '  Built strings (+, +=, a template substitution) are counted in the SUMMARY row, never judged.' \
      '  Waive one query: // lint-bql-unbounded: <reason>   (its line, or the comment-only line above;' \
      '  the reason is mandatory, a bare marker is a FAIL).' \
      'Rows: WARN|FAIL  lint-bql-unbounded  <file>:<line>  <detail>, then one SUMMARY row.' \
      'Exit 0 clean or WARN only, 1 any FAIL (or any WARN under --strict), 3 usage or env. See types/queries.md §3, §6.'
    exit 0 ;;
esac
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bql-unbounded: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-bql-unbounded "$@"
