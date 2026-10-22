#!/usr/bin/env bats
# lint-fork-per-line.bats — structural guard: no toolbelt/lint-*.sh forks a process per input line.
# A `printf '%s' "$line" | grep` (or sed/awk/cut/tr/od/wc/head/tail, or `grep … <<< "$line"`) inside a
# `while read` loop costs a clone and an exec per record: 35-97 s on a client tree, against 0.13-1.25 s for
# one awk per file (toolbelt/lib/awk-lint.sh awk_lint_each). The scanner below reports a fork whose input is
# the read variable of any enclosing `while read` loop. Every `while read` record counts as a line, so a
# per-FILE loop that forks on its loop variable is reported too; rewrite it with parameter expansion.
# Known forks present when the guard landed are frozen in fixtures/lint-fork-per-line/baseline.txt (a
# two-way ratchet: a new fork fails, a fixed fork must shrink the baseline).
# [ev: retro r8-t22-retro-deltas Δ2] [ev: retro line-loop-lint-forks Δ1]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  FIX="$ROOT/tests/fixtures/lint-fork-per-line"
  TOOLBELT="$ROOT/build-n4-module-kit/toolbelt"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  write_scanner > "$TMPDIR_T/scan.awk"
}
teardown() { rm -rf "$TMPDIR_T"; }

# write_scanner: prints the awk scanner. Rows: `FORK <file>:<line> <var>` and, when a file ends with a loop
# or a quote still open, `UNBALANCED <file> loops=<n> sq=<0|1> dq=<0|1>` (the scanner lost the structure,
# so its silence on that file proves nothing).
write_scanner() {
  cat <<'AWK'
FNR == 1 { if (prev != "") endfile(prev); prev = FILENAME; hd = ""; sp = 0; pend = ""; sq = 0; dq = 0; split("", seen) }
END { if (prev != "") endfile(prev) }
function endfile(f) { if (sp != 0 || sq || dq) printf "UNBALANCED %s loops=%d sq=%d dq=%d\n", f, sp, sq, dq }
# lex(s): sets CODE (every quoted character blanked) and SQFREE (single-quoted text blanked, double quotes
# kept so "$line" stays visible); a trailing comment is cut from both. Quote state carries across lines.
function lex(s,    i, c, n, inq) {
  CODE = ""; SQFREE = ""; n = length(s)
  for (i = 1; i <= n; i++) {
    c = substr(s, i, 1)
    if (sq) { if (c == "'") sq = 0; CODE = CODE " "; SQFREE = SQFREE " "; continue }
    if (c == "\\") { CODE = CODE "  "; SQFREE = SQFREE c substr(s, i + 1, 1); i++; continue }
    if (dq) { if (c == "\"") { dq = 0; CODE = CODE c } else CODE = CODE " "; SQFREE = SQFREE c; continue }
    if (c == "'") { sq = 1; CODE = CODE " "; SQFREE = SQFREE " "; continue }
    if (c == "\"") { dq = 1; CODE = CODE c; SQFREE = SQFREE c; continue }
    if (c == "#" && (i == 1 || substr(s, i - 1, 1) ~ /[[:space:];]/)) break
    CODE = CODE c; SQFREE = SQFREE c
  }
}
# check(): a fork whose input is the read variable of an enclosing `while read` loop
function check(   k, m, vv, j, re1, re2) {
  for (k = 1; k <= sp; k++) {
    m = split(stk[k], vv, " ")
    for (j = 1; j <= m; j++) {
      re1 = "(printf|echo)[^|;]*[$][{]?" vv[j] "([^A-Za-z_0-9][^|;]*)?[|][[:space:]]*(grep|sed|awk|cut|tr|od|wc|head|tail)([[:space:]]|$)"
      re2 = "(grep|sed|awk|cut|tr|od|wc|head|tail)[[:space:]][^|;]*<<<[[:space:]]*\"?[$][{]?" vv[j] "([^A-Za-z_0-9]|$)"
      if ((SQFREE ~ re1 || SQFREE ~ re2) && !(FNR in seen)) { seen[FNR]; printf "FORK %s:%d %s\n", FILENAME, FNR, vv[j] }
    }
  }
}
hd != "" { if ($0 ~ ("^[[:space:]]*" hd "[[:space:]]*$")) hd = ""; next }
{
  inq = (sq || dq)
  lex($0)
  # a heredoc body (an awk program, a generated file) is not shell: skip it up to its delimiter
  if (!inq && match($0, /<<-?[[:space:]]*['"]?[A-Za-z_][A-Za-z_0-9]*['"]?/) && substr($0, RSTART, 3) != "<<<") {
    hd = substr($0, RSTART, RLENGTH); sub(/^<<-?[[:space:]]*['"]?/, "", hd); sub(/['"]$/, "", hd)
  }
  if (match(CODE, /(^|[;|&(`[:space:]])while[[:space:]]+(IFS=[^[:space:]]*[[:space:]]+)?read([[:space:]]+-[A-Za-z]+)*([[:space:]]+[A-Za-z_][A-Za-z_0-9]*)+/)) {
    v = substr(CODE, RSTART, RLENGTH); sub(/.*read/, "", v)
    n = split(v, w, /[[:space:]]+/); pend = ""
    for (i = 1; i <= n; i++) if (w[i] ~ /^[A-Za-z_][A-Za-z_0-9]*$/) pend = pend " " w[i]
  }
  # every `do` opens a loop frame (carrying the pending read variables), every `done` closes one
  rest = CODE
  while (match(rest, /(^|[;[:space:]])(do|done)([[:space:]<>;|)&]|$)/)) {
    t = substr(rest, RSTART, RLENGTH); rest = substr(rest, RSTART + RLENGTH - 1); if (rest == "") rest = " "
    if (t ~ /done/) { if (sp > 0) sp-- } else { stk[++sp] = pend; pend = ""; check() }
  }
  check()
}
AWK
}

scan() { awk -f "$TMPDIR_T/scan.awk" "$@"; }

@test "FPL1: every planted per-line fork is reported with its file:line and loop variable" {
  # Mutation: FPL1 -- dropping the fork match (or the read-variable stack) leaves the planted rows unreported.
  run scan "$FIX/planted.sh"
  [ "$status" -eq 0 ]
  expected="FORK $FIX/planted.sh:4 line
FORK $FIX/planted.sh:5 line
FORK $FIX/planted.sh:9 name
FORK $FIX/planted.sh:10 rest
FORK $FIX/planted.sh:11 f"
  [ "$output" = "$expected" ]
}

@test "FPL2: one awk per file, per-file forks, comments, awk programs and post-done pipes are not reported" {
  # Mutation: FPL2 -- matching the fork on the raw line (no quote/comment blanking) reports clean.sh rows.
  run scan "$FIX/clean.sh"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "FPL3: a file the scanner cannot balance is reported, so silence is never read as clean" {
  # Mutation: FPL3 -- removing the end-of-file balance check hides the unclosed loop.
  run scan "$FIX/unbalanced.sh"
  [ "$status" -eq 0 ]
  [ "$output" = "UNBALANCED $FIX/unbalanced.sh loops=1 sq=0 dq=0" ]
}

@test "FPL4: the real toolbelt lints match the frozen baseline exactly (no new fork, no stale row)" {
  # Mutation: FPL4 -- planting a printf|grep on the read variable of a real lint's line loop adds a row.
  scan "$TOOLBELT"/lint-*.sh > "$TMPDIR_T/rows"
  if grep '^UNBALANCED' "$TMPDIR_T/rows"; then return 1; fi
  awk '$1 == "FORK" { f = $2; sub(/:[0-9]+$/, "", f); sub(/.*\//, "", f); n[f " " $3]++ }
       END { for (k in n) print k, n[k] }' "$TMPDIR_T/rows" | sort > "$TMPDIR_T/actual"
  grep -v '^#' "$FIX/baseline.txt" | sort > "$TMPDIR_T/expected"
  if ! diff "$TMPDIR_T/expected" "$TMPDIR_T/actual"; then
    echo "per-line fork set changed: port a new fork to awk_lint_each / Go; delete the row of a fixed one" >&2
    grep '^FORK' "$TMPDIR_T/rows" >&2
    return 1
  fi
}
