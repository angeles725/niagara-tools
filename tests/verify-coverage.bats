#!/usr/bin/env bats
# RED-FIRST pins for the verify-module COVERAGE% metric (campaign 6, investigador1's math seam).
#
# Frozen contract (investigador1, 2026-09-05):
#   surface: verify-module.sh coverage <npass> <nfail> <nwarn> <nskip>   (pure path, echoes result, exit 0)
#   applicable = npass + nfail + nwarn        # SKIP EXCLUDED — structurally not-applicable
#   covered    = npass                        # ONLY clean passes are covered
#   pct        = round(100 * covered / applicable, 1)   if applicable > 0
#              = "N/A"  (string sentinel, NOT 100)       if applicable == 0
#   WARN is applicable-but-NOT-covered (in denominator, out of numerator) so an active warning
#   drags the score below 100 and stays visible — the anti-false-confidence rule.
#
# RED today: verify-module.sh has no `coverage` subcommand; the arg parser treats "coverage" as a
# non-.jar arg and exits 2 ("not a jar"). So every value pin FAILS now (status 2 + wrong output),
# and turns GREEN when the impl adds the pure coverage path over the row() counters (line 47),
# kept extractable (not buried in the summary print).
#
# MUTATIONS these pins MUST catch (run post-green, revert each -> a pinned number moves):
#   - SKIP into the denominator:      3 1 0 2 -> 50.0 != 75.0
#   - WARN into the numerator:        4 0 1 0 -> 100.0 != 80.0
#   - covered = npass+nfail:          1 2 0 0 -> 100.0 != 33.3   (also 3 1 0 2 -> 100.0 != 75.0)
#   - integer truncation / no round:  1 2 0 0 -> 33.0 != 33.3
#   - 0/0 -> 100 instead of N/A:      0 0 0 5 -> "100.0" != "N/A"

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  VM="$KIT/toolbelt/verify-module.sh"
}

@test "MM1: 3P/1F/0W/2S -> 75.0 (SKIP excluded from the denominator)" {
  run "$VM" coverage 3 1 0 2
  [ "$status" -eq 0 ]
  [ "$output" = "75.0" ]
}

@test "MM2: 4P/0F/0W/2S -> 100.0 (all applicable checks are clean passes)" {
  run "$VM" coverage 4 0 0 2
  [ "$status" -eq 0 ]
  [ "$output" = "100.0" ]
}

@test "MM3: 0P/0F/0W/5S -> N/A (applicable 0 -> string sentinel, NOT 100)" {
  run "$VM" coverage 0 0 0 5
  [ "$status" -eq 0 ]
  [ "$output" = "N/A" ]
}

@test "MM4: 3P/1F/1W/1S -> 60.0 (WARN is applicable-but-not-covered)" {
  run "$VM" coverage 3 1 1 1
  [ "$status" -eq 0 ]
  [ "$output" = "60.0" ]
}

@test "MM5: 4P/0F/1W/0S -> 80.0 (a lone WARN drops 100 -> 80, stays visible)" {
  run "$VM" coverage 4 0 1 0
  [ "$status" -eq 0 ]
  [ "$output" = "80.0" ]
}

@test "MM6: 1P/2F/0W/0S -> 33.3 (rounds to 1dp; bites integer truncation and covered=P+F)" {
  run "$VM" coverage 1 2 0 0
  [ "$status" -eq 0 ]
  [ "$output" = "33.3" ]
}

# Arity/type validation (investigador1: argc!=4 or any non-integer arg -> exit 2 + a coverage
# usage line). These pins key on the USAGE MESSAGE, not on exit 2 alone: today `coverage ...`
# already exits 2 as "not a jar: coverage", so a bare exit-2 assert would pass for the WRONG
# reason (low-bite). Asserting the `usage: verify-module.sh coverage` string makes them RED now
# (that string is absent) and green only once the coverage subcommand validates its own args.

@test "MM7: wrong argc (3 args) -> exit 2 + coverage usage on stderr" {
  run "$VM" coverage 1 2 3
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: verify-module.sh coverage"* ]]
}

@test "MM8: a non-integer arg -> exit 2 + coverage usage on stderr" {
  run "$VM" coverage 1 2 a 4
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: verify-module.sh coverage"* ]]
}
