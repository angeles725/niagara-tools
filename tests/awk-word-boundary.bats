#!/usr/bin/env bats
# AWB: no awk regex in the kit uses `\b`. In gawk (and POSIX awk) `\b` inside a regex is a BACKSPACE
# (0x08); the word boundary is `\y`. /\bclass\b/ therefore never matches Java source: a silent false
# negative (found in lint-demand-scope, fixed by its Go port). This ratchet scans every tracked shell /
# awk / hook script for `\b` inside a slash-delimited awk regex: after `~` / `!~`, as the regex argument of
# sub/gsub/gensub/match/split, or as a pattern rule at the start of a line. grep -E, sed and Python `\b`
# (real word boundaries) and a "\b" string escape are not flagged. No allow-list.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  FX="$BATS_TEST_DIRNAME/fixtures/awk-word-boundary"
}

# awk_wordb_hits FILE... — print `file:line:text` for each awk regex that contains \b.
awk_wordb_hits() {
  grep -nHP '(?:!?~|\b(?:g?sub|gensub|match|split)\s*\((?:[^()]*,)?)\s*/(?:[^/\\\n]|\\.)*\\b|^\s*!?/(?:[^/\\\n]|\\.)*\\b(?:[^/\\\n]|\\.)*/\s*\{' "$@"
}

@test "AWB1: no tracked kit script uses \\b inside an awk regex" {
  # Mutation: AWB1 -- dropping the ~/!~ operator form lets ln !~ /\bclass\b/ back in unflagged
  cd "$REPO"
  mapfile -t files < <(git ls-files -- '*.sh' '*.awk' '*.bash' '.githooks/*' 'scripts/*' | grep -v '^tests/fixtures/awk-word-boundary/')
  [ "${#files[@]}" -gt 50 ]
  run awk_wordb_hits "${files[@]}"
  [ "$status" -eq 1 ] || { printf '%s\n' 'awk regex with \b (use \y for a word boundary):' "$output"; false; }
}

@test "AWB2: the scanner flags every awk-regex \\b form (teeth)" {
  run awk_wordb_hits "$FX/bad.sh"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 5 ]
  [[ "$output" == *'bad.sh:3:'* && "$output" == *'bad.sh:4:'* && "$output" == *'bad.sh:5:'* ]]
  [[ "$output" == *'bad.sh:6:'* && "$output" == *'bad.sh:8:'* ]]
}

@test "AWB3: \\y, grep -E, sed, Python and a string escape are not flagged" {
  run awk_wordb_hits "$FX/good.sh"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "AWB4: gawk really reads \\b in a regex as a backspace, and \\y as a word boundary" {
  run gawk '/\bclass\b/ { print "b" } /\yclass\y/ { print "y" }' <<<'public class Foo'
  [ "$output" = "y" ]
}
