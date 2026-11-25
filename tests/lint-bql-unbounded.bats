#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# lint-bql-unbounded.sh — a bql: query literal whose select reads every row WARNs (FAIL under --strict): no top <n>,
# a history: query with no timestamp window, an alarm: query with no index path, an order by with no top. Every test
# builds its own source root from tests/fixtures/lint-bql-unbounded (Java and JavaScript files that are read, never
# compiled) and asserts the exact rows and exit code. Evidence for each check: the header of the wrapper
# (SelectQuery.java, BTopTable.java, BHistoryDbTable.java:232-262, AlarmDbConnection.java:52-110; types/queries.md §3, §6).
# [ev: corpus API survey 2026-11-24 media-cache-db §5.3-5.4] [ev: user-approved improvement 2026-11-24]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$ROOT/build-n4-module-kit"
  L="$KIT/toolbelt/lint-bql-unbounded.sh"
  FIX="$ROOT/tests/fixtures/lint-bql-unbounded"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC"
  W="WARN  lint-bql-unbounded  $SRC"
  NT=' has no top <n>: it returns every row of its extent'
  HW=' has no top <n> and no timestamp window (?period=, start=, end= or a timestamp predicate): it scans the whole history'
  AL=' has no top <n> and no index path (from openAlarms, from ackPendingAlarms, uuid =, source = or a timestamp range): it scans the whole alarm database'
  OB=' has an order by and no top <n>: order by loads every row before any cap'
}

# stage <File>... : copy fixtures into the source root under test
stage() { stage_in "$SRC" "$@"; }
stage_in() { local d="$1" f; shift; mkdir -p "$d"; for f in "$@"; do cp "$FIX/$f" "$d/"; done; }

# summary <checked> <flagged> <waived> <skipped> : the SUMMARY row
summary() { printf 'SUMMARY  lint-bql-unbounded  checked=%s flagged=%s waived=%s non-literal-skipped=%s' "$1" "$2" "$3" "$4"; }

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "BQLU-usage: no argument exits 3 with the usage on stderr, nothing on stdout" {
  run --separate-stderr "$L"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [[ "$stderr" == "usage: lint-bql-unbounded.sh [--help] [--strict] <src-root>..." ]]
}

@test "BQLU-help: --help prints the usage and the checks, exit 0" {
  run "$L" --help
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "usage: lint-bql-unbounded.sh [--help] [--strict] <src-root>..." ]]
  [[ "$output" == *"history-no-window"* && "$output" == *"alarm-unindexed"* && "$output" == *"lint-bql-unbounded: <reason>"* ]]
}

@test "BQLU-badroot: an unknown option or a missing directory exits 3" {
  run "$L" --nope "$SRC"
  [ "$status" -eq 3 ]
  run "$L" "$BATS_TEST_TMPDIR/absent"
  [ "$status" -eq 3 ]
}

@test "BQLU-empty: a root with no *.java or *.js is an ERROR row and exit 3, never a clean pass" {
  printf 'notes\n' > "$SRC/README.txt"
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-bql-unbounded  $SRC  no *.java or *.js sources found (wrong path or empty source tree)" ]
}

@test "BQLU-top: a top <n> (a number or a %d placeholder) bounds a query; only the unbounded ones are rows" {
  # Mutation: BQLU-top -- ignoring `top <n>` WARNs the bounded queries of BClean
  stage BClean.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$(summary 3 0 0 0)" ]
  local d="$BATS_TEST_TMPDIR/notop" n
  n="WARN  lint-bql-unbounded  $d"; stage_in "$d" BNoTop.java
  run "$L" "$d"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 5 ]
  [ "${lines[0]}" = "$n/BNoTop.java:5  no-top: \"select * from baja:Component\"$NT (types/queries.md §3)" ]
  [ "${lines[1]}" = "$n/BNoTop.java:6  no-top: \"select name, out from control:NumericWritable where out.v...\"$NT (types/queries.md §3)" ]
  [ "${lines[2]}" = "$n/BNoTop.java:7  no-top: \"select from control:NumericPoint\"$NT (types/queries.md §3)" ]
  [ "${lines[3]}" = "$n/BNoTop.java:10  no-top: \"select * where name = '%s'\"$NT (types/queries.md §3)" ]
  [ "${lines[4]}" = "$(summary 6 4 0 0)" ]
}

@test "BQLU-strict: --strict prints the WARN rows as FAIL and exits 1; a clean tree stays exit 0" {
  stage BNoTop.java
  run "$L" --strict "$SRC"
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL  lint-bql-unbounded  ')" -eq 4 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
  stage_in "$BATS_TEST_TMPDIR/clean" BClean.java
  run "$L" --strict "$BATS_TEST_TMPDIR/clean"
  [ "$status" -eq 0 ]
  [ "$output" = "$(summary 3 0 0 0)" ]
}

@test "BQLU-window: a history: query needs a top, a ?period=/start=/end= or a timestamp predicate (not 'or', not value)" {
  # Mutation: BQLU-window -- ignoring ?period= and the timestamp predicate WARNs the windowed history queries
  stage BHistory.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 6 ]
  [ "${lines[0]}" = "$W/BHistory.java:5  history-no-window: \"select timestamp, value\"$HW (types/queries.md §3, §4)" ]
  [ "${lines[1]}" = "$W/BHistory.java:6  history-no-window: \"select *\"$HW (types/queries.md §3, §4)" ]
  [ "${lines[2]}" = "$W/BHistory.java:12  history-no-window: \"select * where timestamp > '2026-01-01' or value > 5\"$HW (types/queries.md §3, §4)" ]
  [ "${lines[3]}" = "$W/BHistory.java:13  history-no-window: \"select * where value > 75\"$HW (types/queries.md §3, §4)" ]
  [ "${lines[4]}" = "$W/BHistory.java:14  no-top: \"select *\"$NT (types/queries.md §3)" ]
  [ "${lines[5]}" = "$(summary 10 5 0 0)" ]
}

@test "BQLU-index: an alarm: query needs a top, openAlarms/ackPendingAlarms (exact case), uuid =, source = or a timestamp range" {
  # Mutation: BQLU-index -- matching openAlarms case-insensitively passes `from openalarms`, which the station scans
  stage BAlarm.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 6 ]
  [ "${lines[0]}" = "$W/BAlarm.java:5  alarm-unindexed: \"select *\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[1]}" = "$W/BAlarm.java:6  alarm-unindexed: \"select * where sourceState = 'offnormal'\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[2]}" = "$W/BAlarm.java:12  alarm-unindexed: \"select *\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[3]}" = "$W/BAlarm.java:13  order-by-no-top: \"select * from openAlarms order by timestamp desc\"$OB (types/queries.md §3)" ]
  [ "${lines[4]}" = "$W/BAlarm.java:15  alarm-unindexed: \"select * from openalarms\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[5]}" = "$(summary 11 5 0 0)" ]
}

@test "BQLU-orderby: an order by with no top is a row even when the source is windowed; an unbounded one says so" {
  stage BOrderBy.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[0]}" = "$W/BOrderBy.java:5  no-top: \"select name from baja:Component order by name\"$NT; order by sorts every row first (types/queries.md §3)" ]
  [ "${lines[1]}" = "$W/BOrderBy.java:6  order-by-no-top: \"select * order by timestamp desc\"$OB (types/queries.md §3)" ]
  [ "${lines[2]}" = "$(summary 3 2 0 0)" ]
}

@test "BQLU-dynamic: a built string is counted in the SUMMARY row, never judged; literals added to each other are one literal" {
  # Mutation: BQLU-dynamic -- judging a built string reports a query whose top or window is added at run time
  stage BDynamic.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "$W/BDynamic.java:7  no-top: \"select * from baja:Component\"$NT (types/queries.md §3)" ]
  [ "${lines[1]}" = "$(summary 1 1 0 3)" ]
}

@test "BQLU-comment: comments, Javadoc and prose are no query; a text block is read" {
  # Mutation: BQLU-comment -- matching raw text WARNs the queries quoted in a comment or a Javadoc
  stage BNotCode.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "$W/BNotCode.java:13  alarm-unindexed: \"select *\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[1]}" = "$(summary 3 1 0 0)" ]
}

@test "BQLU-waiver: a marker with a reason waives the query on its line or on the comment-only line above" {
  # Mutation: BQLU-waiver -- ignoring the marker WARNs the two waived queries of BWaived
  stage BWaived.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "$W/BWaived.java:8  alarm-unindexed: \"select *\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[1]}" = "$(summary 3 1 2 0)" ]
}

@test "BQLU-waiver-bare: a marker with no reason is its own FAIL (exit 1) and waives nothing" {
  # Mutation: BQLU-waiver-bare -- accepting a bare marker waives the alarm query without a stated reason
  stage BWaiverBare.java
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[0]}" = "FAIL  lint-bql-unbounded  $SRC/BWaiverBare.java:5  waiver-no-reason: lint-bql-unbounded: needs a reason after the colon" ]
  [ "${lines[1]}" = "$W/BWaiverBare.java:6  alarm-unindexed: \"select *\"$AL (types/queries.md §3, §6)" ]
  [ "${lines[2]}" = "$(summary 1 1 0 0)" ]
}

@test "BQLU-js: a JavaScript string, template and waiver are read; *.min.js is not scanned" {
  stage ux.js lib.min.js
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[0]}" = "$W/ux.js:5  no-top: \"select * from baja:Component\"$NT (types/queries.md §3)" ]
  [ "${lines[1]}" = "$W/ux.js:7  history-no-window: \"select timestamp, value\"$HW (types/queries.md §3, §4)" ]
  [ "${lines[2]}" = "$(summary 5 2 1 2)" ]
}

@test "BQLU-edges: upper case, a |view: suffix, a | inside quotes, += and a ? placeholder after top" {
  stage BEdges.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[0]}" = "$W/BEdges.java:5  no-top: \"SELECT * FROM baja:Component\"$NT (types/queries.md §3)" ]
  [ "${lines[1]}" = "$W/BEdges.java:6  no-top: \"select name from baja:Component\"$NT (types/queries.md §3)" ]
  [ "${lines[2]}" = "$(summary 4 2 0 1)" ]
}

@test "BQLU-tree: the whole fixture tree gives the expected totals, in file order, over several roots" {
  run "$L" "$FIX"
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN  lint-bql-unbounded  ')" -eq 24 ]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL  lint-bql-unbounded  .*waiver-no-reason:')" -eq 1 ]
  [ "${lines[$((${#lines[@]} - 1))]}" = "$(summary 50 24 3 6)" ]
  mkdir -p "$BATS_TEST_TMPDIR/r2"; cp "$FIX/BClean.java" "$BATS_TEST_TMPDIR/r2/"; stage BOrderBy.java
  run "$L" "$BATS_TEST_TMPDIR/r2" "$SRC"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[2]}" = "$(summary 6 2 0 0)" ]
}

@test "BQLU-dotdir: a dot-directory, a *-worktrees directory, build/ and node_modules/ are pruned" {
  mkdir -p "$SRC/.hidden" "$SRC/m-worktrees" "$SRC/build" "$SRC/node_modules"
  local d; for d in .hidden m-worktrees build node_modules; do cp "$FIX/BNoTop.java" "$SRC/$d/"; done
  stage BClean.java
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "$(summary 3 0 0 0)" ]
}

@test "BQLU-finderr: an unreadable sub-directory is an env error (exit 3, named), never a clean pass" {
  mkdir -p "$SRC/locked"; cp "$FIX/BNoTop.java" "$SRC/locked/"; stage BClean.java
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$SRC"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}
