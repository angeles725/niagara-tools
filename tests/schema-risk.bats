#!/usr/bin/env bats
# RED-FIRST pins for schema-risk.sh (campaign 7 PR5, issue #46, = MM3 classifier + B799 fixtures companero).
# Classifies a slot diff between two module snapshots into SAFE / LOSSY / OUTAGE per the B795 §795.4 survival
# matrix, so a saved-data-breaking change is caught BEFORE deploy (a .bog binds to the class by name, ungated).
#
# SURFACE (design will confirm; pins kept in one file): schema-risk.sh <before-dir> <after-dir>
#   -> one row per changed slot `<verdict>  <change_kind>  <slot>: <detail (B795 row)>` + final `verdict=<V>` line.
#   exit: 0 SAFE · 1 LOSSY · 2 OUTAGE · 3 usage/env. Vocabulary strictly B795 §795.4.
#   Module verdict = MAX severity (worst cell); an UNKNOWN change_kind is fail-safe OUTAGE (never SAFE).
#
# Fixtures (tests/fixtures/schema-risk/, scratch-authored B799, copied in so pins are self-contained):
#   add_slot SAFE(r1) · remove_slot LOSSY(r7) · retype_simple OUTAGE(r11/B739) · reorder SAFE(r2) ·
#   rename_slot LOSSY(r9) · unknown_kind OUTAGE(fail-safe) · mixed OUTAGE(worst-cell: add SAFE + retype OUTAGE).
#
# RED today: schema-risk.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATIONS (post-green):
#   - worst-cell -> first-cell: mixed reads SAFE (its first change is add_slot=SAFE) != OUTAGE.
#   - unknown_kind -> SAFE (drops the fail-safe): unknown_kind flips OUTAGE -> SAFE.
#   - retype detection off: retype_simple reads SAFE != OUTAGE.
#   - rename detection off (rename seen as remove+add): verdict stays LOSSY, so the change_kind LABEL pin
#     (SR_rename asserts "rename") is what catches it.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SR="$KIT/toolbelt/schema-risk.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/schema-risk"
}
pair() { run "$SR" "$FX/$1/before" "$FX/$1/after"; }

@test "SR1 add_slot -> SAFE (exit 0, verdict=SAFE) [B795 r1]" {
  pair add_slot; [ "$status" -eq 0 ]; [[ "$output" == *"verdict=SAFE"* ]]
}
@test "SR2 remove_slot -> LOSSY (exit 1, verdict=LOSSY) [B795 r7]" {
  pair remove_slot; [ "$status" -eq 1 ]; [[ "$output" == *"verdict=LOSSY"* ]]
}
@test "SR3 retype_simple -> OUTAGE (exit 2, verdict=OUTAGE) [B795 r11/B739]" {
  pair retype_simple; [ "$status" -eq 2 ]; [[ "$output" == *"verdict=OUTAGE"* ]]
}
@test "SR4 reorder -> SAFE (exit 0, verdict=SAFE) [B795 r2 byName index-free]" {
  pair reorder; [ "$status" -eq 0 ]; [[ "$output" == *"verdict=SAFE"* ]]
}
@test "SR5 rename_slot -> LOSSY AND the change_kind reads 'rename' not remove+add [B795 r9]" {
  pair rename_slot; [ "$status" -eq 1 ]; [[ "$output" == *"verdict=LOSSY"* ]]; [[ "$output" == *"rename"* ]]
}
@test "SR6 unknown_kind -> OUTAGE fail-safe (exit 2, names UNKNOWN, never SAFE)" {
  pair unknown_kind; [ "$status" -eq 2 ]; [[ "$output" == *"verdict=OUTAGE"* ]]; [[ "$output" == *"UNKNOWN"* ]]
}
@test "SR7 mixed (add SAFE + retype OUTAGE) -> OUTAGE by worst-cell (exit 2)" {
  pair mixed; [ "$status" -eq 2 ]; [[ "$output" == *"verdict=OUTAGE"* ]]
}
@test "SR8 missing args -> exit 3 (usage/env)" {
  run "$SR"; [ "$status" -eq 3 ]
}
