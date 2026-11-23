#!/usr/bin/env bats
# lab-station.bats — the launcher of lab-station (issue #848). The unit cases (pipe, waits, refusals) live in
# go/internal/labstation; here a fake install whose bin/station is a console stub proves the launcher builds the
# binary and runs a whole start, console, stop cycle from a script.

setup() {
  T="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/lab-station.sh"
  export NIAGARA_HOME="$BATS_TEST_TMPDIR/niagara" LAB_STATION_STATE="$BATS_TEST_TMPDIR/state"
  mkdir -p "$NIAGARA_HOME/bin"
  # shellcheck disable=SC2016  # the stub is a script body: its $ expands when IT runs
  printf '%s\n' '#!/usr/bin/env bash' 'echo "*** Station Started (1ms) ***"; printf "niagara>"' \
    'while IFS= read -r l; do case ${l%$'"'"'\r'"'"'} in' \
    '  heap) printf "usedMemory: 1kb\n\nniagara>" ;;' \
    '  quit) echo "*** Station shutdown ***"; exit 0 ;;' 'esac; done' > "$NIAGARA_HOME/bin/station"
  chmod +x "$NIAGARA_HOME/bin/station"
}

@test "LS1: start, console heap and stop run from a script" {
  run "$T" start LAB
  [ "$status" -eq 0 ]
  [[ "$output" == "lab-station: LAB started (pid "*"): *** Station Started (1ms) ***" ]]
  run "$T" console LAB heap
  [ "$status" -eq 0 ]
  [ "$output" = "usedMemory: 1kb" ]
  run "$T" stop LAB
  [ "$status" -eq 0 ]
  [ "$output" = "lab-station: LAB stopped: *** Station shutdown ***" ]
}

@test "LS2: --dry-run prints the plan and creates no state" {
  run "$T" start --dry-run LAB
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "lab-station: dry run, nothing done" ]
  [ ! -e "$LAB_STATION_STATE" ]
}

@test "LS3: console kill is refused, exit 3" {
  run "$T" console LAB kill
  [ "$status" -eq 3 ]
  [[ "$output" == *"console verb kill is refused"* ]]
}
