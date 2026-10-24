#!/usr/bin/env bats
# fail-open-idioms.bats — structural guard: no kit shell check derives pass/fail from text a producer may
# never print. Two shapes passed a broken PR (#409) through a lander:
#   GREP-GATE   `( build && vet && test 2>&1 | grep -E '^FAIL' ) && { echo FAIL; exit 1; }` — the group's
#               status is grep's: when build or vet fails, test never runs, grep reads nothing, the gate passes.
#   FMT-STDERR  `[ -z "$(gofmt -l go)" ]` — gofmt prints a syntax error on stderr, the list stays empty.
# Fix idioms: `( build && vet && test >log 2>&1 ) || fail`; `fmt=$(gofmt -l go 2>&1) && [ -z "$fmt" ] || fail`.
# The scanner reads scripts/, .githooks/ and build-n4-module-kit/toolbelt/**/*.sh (node_modules pruned), with
# quoted text blanked for GREP-GATE and comments cut for both; fixtures/fail-open-idioms/baseline.txt is a
# two-way ratchet (a new hit fails, a fixed one must shrink it).
# [ev: retro fail-open-pipeline-checks Δ1]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  FIX="$ROOT/tests/fixtures/fail-open-idioms"
  write_scanner > "$BATS_TEST_TMPDIR/scan.awk"
}

# write_scanner: prints the awk scanner. Rows: `GREP-GATE <file>:<line>` and `FMT-STDERR <file>:<line>`.
write_scanner() {
  cat <<'AWK'
# lex(s): CODE = the line with every quoted character blanked, TXT = the line with quotes kept; a trailing
# comment is cut from both. Quote state carries across lines.
function lex(s,    i, c, n) {
  CODE = ""; TXT = ""; n = length(s)
  for (i = 1; i <= n; i++) {
    c = substr(s, i, 1)
    if (sq) { if (c == "'") sq = 0; CODE = CODE " "; TXT = TXT c; continue }
    if (c == "\\") { CODE = CODE "  "; TXT = TXT c substr(s, i + 1, 1); i++; continue }
    if (dq) { if (c == "\"") dq = 0; CODE = CODE (dq ? " " : c); TXT = TXT c; continue }
    if (c == "'") { sq = 1; CODE = CODE " "; TXT = TXT c; continue }
    if (c == "\"") { dq = 1; CODE = CODE c; TXT = TXT c; continue }
    if (c == "#" && (i == 1 || substr(s, i - 1, 1) ~ /[[:space:];]/)) break
    CODE = CODE c; TXT = TXT c
  }
}
FNR == 1 { sq = 0; dq = 0 }
{
  lex($0)
  # GREP-GATE: a parenthesized group ending in `| grep …`, its status used as the failure signal
  if (CODE ~ /[(][^()]*[|][[:space:]]*grep([[:space:]][^()]*)?[)][[:space:]]*&&[[:space:]]*([{][^}]*)?(exit|fail|return)([[:space:];]|$)/)
    printf "GREP-GATE %s:%d\n", FILENAME, FNR
  # FMT-STDERR: a formatter's file list captured without its stderr
  t = TXT
  while (match(t, /[$][(][[:space:]]*(gofmt|shfmt)[[:space:]]+-l[^)]*[)]/)) {
    if (substr(t, RSTART, RLENGTH) !~ /2>/) { printf "FMT-STDERR %s:%d\n", FILENAME, FNR; break }
    t = substr(t, RSTART + RLENGTH)
  }
}
AWK
}

# scanned_files: every file the guard reads, repo-relative, sorted.
scanned_files() {
  (cd "$ROOT" && { find scripts .githooks -type f
    find build-n4-module-kit/toolbelt -name node_modules -prune -o -type f -name '*.sh' -print; } | LC_ALL=C sort)
}

@test "FOI1: the seeded fixture's four idioms are caught and the fixed forms are not" {
  # Mutation: FOI1 -- dropping the 2> exemption or the quote blanking reports fixed.sh; dropping a rule misses planted.sh
  run awk -f "$BATS_TEST_TMPDIR/scan.awk" "$FIX/planted.sh" "$FIX/fixed.sh"
  [ "$status" -eq 0 ]
  [ "$output" = "GREP-GATE $FIX/planted.sh:4
GREP-GATE $FIX/planted.sh:5
FMT-STDERR $FIX/planted.sh:6
FMT-STDERR $FIX/planted.sh:7" ]
}

@test "FOI2: the kit's scripts hold exactly the baselined fail-open idioms" {
  # Mutation: FOI2 -- planting `[ -z \"\$(gofmt -l go)\" ] || exit 1` in a toolbelt script fails this test
  local got want
  got=$(cd "$ROOT" && scanned_files | tr '\n' '\0' | xargs -0 awk -f "$BATS_TEST_TMPDIR/scan.awk")
  want=$(grep -v '^#' "$FIX/baseline.txt" || true)
  [ "$got" = "$want" ] || { printf 'tree:\n%s\nbaseline:\n%s\n' "$got" "$want"; return 1; }
}

@test "FOI3: the scan is not vacuous (it reads the toolbelt, the scripts and the pre-push hook)" {
  run scanned_files
  [ "${#lines[@]}" -ge 60 ]
  [[ "$output" == *".githooks/pre-push"* ]] && [[ "$output" == *"scripts/ng-deploy.sh"* ]]
  [[ "$output" == *"build-n4-module-kit/toolbelt/report-module.sh"* ]]
}
