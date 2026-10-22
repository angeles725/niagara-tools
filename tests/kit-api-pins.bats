#!/usr/bin/env bats
# Pins kit API cards to the Niagara source they transcribe, so a copy error cannot come back silently.
# KP1 regression: types/logic.md's BStatus table was shifted by one bit from STALE on and had no ALARM row
#     (register C04, retro devguide415-runtime-api-corrections Δ4). Expected values are the constants of
#     javax.baja.status.BStatus (docSource 4.15 BStatus.java:702-716; 4.15.6.20 BStatus.java:758-786, corpus B1314 §1314.2).
# KP2 regression: types/logic.md taught `getPropagateFlags().and(...)`; BStatus has no and() method
#     (register C05, retro devguide415-runtime-api-corrections Δ5; kitControl BLogic masks with getBits()).

setup() { KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"; }

# the `| `FLAG` | 0xNN |` rows of the "8-bit BStatus reference" table, as "FLAG 0xNN" lines
bstatus_rows() {
  awk '/8-bit BStatus reference/ { on = 1; next }
       on && /^\|/ { in_table = 1
                     if (match($0, /`[A-Z_]+` *\| *0x[0-9A-Fa-f]+/)) {
                       row = substr($0, RSTART, RLENGTH); gsub(/[`|]/, "", row); gsub(/ +/, " ", row); print row }
                     next }
       in_table { exit }' "$KIT/types/logic.md"
}

@test "KP1: types/logic.md BStatus table equals the BStatus bit constants (STALE shift, missing ALARM)" {
  expected='DISABLED 0x01
FAULT 0x02
DOWN 0x04
ALARM 0x08
STALE 0x10
OVERRIDDEN 0x20
NULL 0x40
UNACKED_ALARM 0x80'
  run bstatus_rows
  [ "$status" -eq 0 ]
  if [ "$output" != "$expected" ]; then
    printf 'BStatus table drift:\n--- doc\n%s\n--- source constants\n%s\n' "$output" "$expected" >&2
    return 1
  fi
}

@test "KP2: no kit doc teaches BStatus.and() / getPropagateFlags().and( (method does not exist)" {
  run grep -rnE 'getPropagateFlags\(\)\.and\(|BStatus\.and\(' "$KIT"/*.md "$KIT"/types/*.md
  [ "$status" -eq 1 ]     # grep: no match
}
