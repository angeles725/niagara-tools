#!/usr/bin/env bats
# Guards the kit's logic-series helper (toolbelt/java/n4kit/series) through its public interface:
# run-pure-test.sh on the worked-example module part (tests/fixtures/logic-series/thermo-rt), whose
# ThermostatSeriesTest steps a toy pure model through srcTest/series/thermostat.csv.
# Each test copies the fixture and rewrites the series, so a case is one CSV edit.
# SC2030/SC2031: `run` sets $status/$output per @test (a subshell to shellcheck); the bad() helper reads them in
# the same test, so they are not real subshell-loss bugs.
# shellcheck disable=SC2030,SC2031

setup_file() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
  . "$ROOT/go/go-tool.sh"
  go_tool_bin "$ROOT/go" run-pure-test >/dev/null
}

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  RUN="$ROOT/build-n4-module-kit/toolbelt/run-pure-test.sh"
  if ! find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | grep -q .; then
    if [ -n "${CI:-}" ]; then
      echo "CI: junit-4.13.2.jar not in ~/.gradle — ci.yml pre-fetch step is required" >&2
      false
    else
      skip "junit-4.13.2 not in ~/.gradle cache (run one gradle build to fetch it)"
    fi
  fi
  RT="$BATS_TEST_TMPDIR/thermo-rt"
  cp -R "$ROOT/tests/fixtures/logic-series/thermo-rt" "$RT"
  SERIES="$RT/srcTest/series/thermostat.csv"
  HEAD='time_ms, in:temp, in:setpoint, in:enable, out:cooling, out:demand~0.5'
}

# series <line>... : replaces the series with the given lines (the header is not added).
series() { printf '%s\n' "$@" > "$SERIES"; }

run_series() { run "$RUN" "$RT" com.example.thermo.ThermostatSeriesTest; }

@test "LS1: the worked example passes: exit 0, every expected value checked" {
  run_series
  [ "$status" -eq 0 ]
  [[ "$output" == *"series checked 15 values"* ]]
  [[ "$output" == *"OK (1 test)"* ]]
}

@test "LS2: a mismatch fails with exit 1 and names the FIRST one only (line, step, time, field, expected, actual)" {
  series "$HEAD" \
    '0,    4.0, 4, true, false, 0' \
    '1000, 5.5, 4, true, false, 37.5' \
    '2000, 9.0, 4, true, false, 0'
  run_series
  [ "$status" -eq 1 ]
  [[ "$output" == *"srcTest/series/thermostat.csv:3: step 2 (t=1000 ms) out:cooling expected 0.0, actual 1.0"* ]]
  [[ "$output" != *"step 3"* ]]
}

@test "LS3: tolerance: a value within ~tol passes, one beyond it fails and the message shows +/-tol in ASCII" {
  series "$HEAD" '0, 5.5, 4, true, true, 37.9'
  run_series
  [ "$status" -eq 0 ]
  series "$HEAD" '0, 5.5, 4, true, true, 38.1'
  run_series
  [ "$status" -eq 1 ]
  [[ "$output" == *"out:demand expected 38.1 +/-0.5, actual 37.5"* ]]
}

@test "LS4: an exact column (no ~tol) refuses any difference" {
  series 'time_ms, in:temp, in:setpoint, in:enable, out:demand' '0, 5.5, 4, true, 37.4'
  run_series
  [ "$status" -eq 1 ]
  [[ "$output" == *"out:demand expected 37.4, actual 37.5"* ]]
}

@test "LS5: a '-' cell is not checked at that step and is not counted" {
  series "$HEAD" '0, 5.5, 4, true, -, -' '1000, 5.5, 4, true, true, 37.5'
  run_series
  [ "$status" -eq 0 ]
  [[ "$output" == *"series checked 2 values"* ]]
}

# bad <expected message fragment> <line>... : the series is refused before any step, with exit 1.
bad() {
  local want="$1"; shift
  series "$@"
  run_series
  [ "$status" -eq 1 ]
  [[ "$output" == *"bad series srcTest/series/thermostat.csv:$want"* ]]
}

@test "LS6: bad series refused: first column not time_ms" {
  bad "1: first column must be time_ms, got 't'" 't, in:temp, out:cooling' '0, 1, false'
}

@test "LS7: bad series refused: a column without in:/out:" {
  bad "1: column 'temp' must start with in: or out:" 'time_ms, temp, out:cooling' '0, 1, false'
}

@test "LS8: bad series refused: no out: column" {
  bad "1: no out: column" 'time_ms, in:temp' '0, 1'
}

@test "LS9: bad series refused: a row with the wrong cell count" {
  bad "3: 5 cells, header has 6" "$HEAD" '0, 4, 4, true, false, 0' '1000, 4, 4, true, false'
}

@test "LS10: bad series refused: time goes back" {
  bad "3: time 500 goes back (previous 1000)" "$HEAD" '1000, 4, 4, true, false, 0' '500, 4, 4, true, false, 0'
}

@test "LS11: bad series refused: a non-numeric cell, a negative tolerance, a duplicate column, no rows" {
  bad "2: cell 'warm' is not a number, true or false" "$HEAD" '0, warm, 4, true, false, 0'
  bad "1: column 'out:demand~-1': tolerance must be a number >= 0" 'time_ms, in:temp, out:demand~-1' '0, 1, 0'
  bad "1: duplicate column 'in:temp'" 'time_ms, in:temp, in:temp, out:cooling' '0, 1, 1, false'
  bad "1: no data rows after the header" "$HEAD"
}

@test "LS12: the adapter's unknown output name is an error, never a silent pass" {
  series 'time_ms, in:temp, in:setpoint, in:enable, out:coolng' '0, 4, 4, true, false'
  run_series
  [ "$status" -eq 1 ]
  [[ "$output" == *"Thermostat has no output coolng"* ]]
}

@test "LS13: the series path resolves relative to the module part from any caller directory" {
  mkdir -p "$BATS_TEST_TMPDIR/elsewhere"
  cd "$BATS_TEST_TMPDIR/elsewhere"
  run_series
  [ "$status" -eq 0 ]
}

@test "LS14: a UTF-8 byte order mark before the header is ignored (a series saved by a Windows editor)" {
  series "$(printf '\xef\xbb\xbf')$HEAD" '0, 5.5, 4, true, true, 37.5'
  run_series
  [ "$status" -eq 0 ]
  [[ "$output" == *"series checked 2 values"* ]]
}

@test "LS15: bad series refused: a time beyond the long range names its line, never a raw NumberFormatException" {
  bad "2: time '99999999999999999999' is out of range (ms)" "$HEAD" '99999999999999999999, 4, 4, true, false, 0'
  [[ "$output" != *"NumberFormatException"* ]]
}
