#!/usr/bin/env bats
# RED-FIRST pins for bog-audit.sh CHECK13-CHECK19 (campaign 8 PR20, wave3 R20/D17). Station-logic
# wiring checks over the handle/link graph already built by CHECK7-CHECK11 (post-processing, no new
# grammar). bog-alone (no --source-dir needed). Real shapes proven on issue #49 (PANCCADIA).
#   CHECK13 relay-double-source FAIL   CHECK14 own-output-unlinked WARN   CHECK15 sensor-crossed WARN
#   CHECK16 hasDefrost<->DefrostController sibling FAIL   CHECK17 roomN-index-mismatch FAIL
#   CHECK18 dashboard tile-number consistency FAIL   CHECK19 link-direction WARN
#
# The apply worker folds these into tests/bog-audit.bats (absent on main until PR10 lands); kept here as
# a self-contained file so they are RED for the right reason now. RED today: bog-audit.sh absent.
# NAMED MUTATIONS (R20.9): CHECK13 add second source -> FAIL; CHECK14 remove relay link -> WARN;
# CHECK16 set hasDefrost=true without sibling -> FAIL; CHECK18 swap freeze links to mismatch tile -> FAIL.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  BA="$KIT/toolbelt/bog-audit.sh"
  T="$BATS_TEST_TMPDIR"
  BOGX="/tmp/claude-1000/-home-cristian-niagara-research/dd08d593-6527-4dc7-885c-938ea923b73f/scratchpad/bogx"
}
# mkbog <name> : read an XML bog body from stdin, wrap and zip it -> $T/<name>.bog
mkbog() { cat > "$T/$1.xml"; ( cd "$T" && cp "$1.xml" file.xml && zip -q "$1.bog" file.xml && rm -f file.xml ); }

@test "SL13: CHECK13 — two distinct sources into the same relay slot FAIL (relay-double-source)" {
  # Named mutation (R20.9): two sources on the same relay slot -> FAIL.
  # SrcA/SrcB are non-self-closing so they register in handle_map (needed for CHECK1 to find the module).
  mkbog c13 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='NrioNet' h='n1' m='nrio=nrio' t='nrio:NrioNetwork'>
  <p n='ro1' h='r1' t='c:BooleanWritable'>
   <p n='L1' t='b:Link'><p n="sourceOrd" v="h:s1"/><p n="sourceSlotName" v="out"/><p n="targetSlotName" v="in10"/></p>
   <p n='L2' t='b:Link'><p n="sourceOrd" v="h:s2"/><p n="sourceSlotName" v="out"/><p n="targetSlotName" v="in10"/></p>
  </p>
 </p>
 <p n='SrcA' h='s1' m='CRP=ColdRoomPan' t='CRP:EvaporatorUnit'>
  <p n="evapOut" t="b:StatusBoolean" v="false"/>
 </p>
 <p n='SrcB' h='s2' t='CRP:EvaporatorUnit'>
  <p n="evapOut" t="b:StatusBoolean" v="false"/>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c13.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK13"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL14: CHECK14 — an OPERATOR output slot with no outgoing relay link WARNs (own-output-unlinked)" {
  mkbog c14 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_5' h='cr5' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit2' h='e5' t='CRP:EvaporatorUnit'>
   <p n="evapOut" f="o" t="b:StatusBoolean" v="false"/>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c14.bog" --module ColdRoomPan
  [[ "$output" == *"CHECK14"* ]] && [[ "$output" == *"WARN"* ]] && [[ "$output" == *"evapOut"* ]]
}

@test "SL16: CHECK16 — hasDefrost=true with no DefrostController sibling FAILs" {
  mkbog c16 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_2' h='cr2' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='Evap' h='ev2' t='CRP:EvaporatorUnit'>
   <p n="hasDefrost" t="b:boolean" v="true"/>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c16.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK16"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL17: CHECK17 — ColdRoom_1 whose link names address evap3* (index mismatch) FAILs" {
  mkbog c17 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_1' h='cr1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='hoaLink' t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="hoa"/><p n="targetSlotName" v="evap3Hoa"/></p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c17.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK17"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL18: CHECK18 — an evap unit whose HOA tile (evap3) != freeze tile (evap1) FAILs (tile-number)" {
  mkbog c18 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_1' h='cr1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit_1' h='eu1' t='CRP:EvaporatorUnit'>
   <p n='hoaL'    t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="hoa"/><p n="targetSlotName" v="evap3Hoa"/></p>
   <p n='freezeL' t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="frz"/><p n="targetSlotName" v="evap1Freeze"/></p>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c18.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK18"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL15: CHECK15 — C-room-labeled slot sourced from a component with a different unit index WARNs (sensor-crossed-by-name)" {
  # Lead pin (R20.4): slot name C1_temperature from EvaporatorUnit_3 (index 3 != C-room 1) -> WARN.
  # RED-proven: running without CHECK15 rule produces no CHECK15 in output.
  mkbog c15 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='Station' h='st' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit_3' h='e3' t='CRP:EvaporatorUnit'>
   <p n='L1' t='b:Link'><p n="sourceOrd" v="h:e3"/><p n="sourceSlotName" v="C1_temperature"/><p n="targetSlotName" v="tempOut"/></p>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c15.bog" --module ColdRoomPan
  [[ "$output" == *"CHECK15"* ]] && [[ "$output" == *"WARN"* ]]
}

@test "SL19: CHECK19 — control component writing to a setpoint slot WARNs (reverse link-direction)" {
  # Lead pin (R20.8): own-module source writes to temperatureSetpoint -> WARN (expected: panel->control).
  # RED-proven: running without CHECK19 rule produces no CHECK19 in output.
  mkbog c19 <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='DashPanel_1' h='dp1' m='DPan=DashboardPan' t='DPan:DashUnit'>
  <p n='SpLink' t='b:Link'><p n="sourceOrd" v="h:crp1"/><p n="sourceSlotName" v="temperatureOut"/><p n="targetSlotName" v="temperatureSetpoint"/></p>
 </p>
 <p n='Ctrl_1' h='crp1' m='CRP=ColdRoomPan' t='CRP:EvaporatorUnit'>
  <p n="temperatureOut" f="o" t="b:StatusNumeric" v="0"/>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c19.bog" --module ColdRoomPan --module DashboardPan
  [[ "$output" == *"CHECK19"* ]] && [[ "$output" == *"WARN"* ]]
}

@test "SL-smoke-panccadia: PANCCADIA -> CHECK18 FAIL on ColdRoom_1 EvaporatorUnit_1 and _3, CHECK14 WARN ColdRoom_5, else clean (SKIP if absent)" {
  RB="${C8_PANCCADIA_BOG:-$BOGX/panccadia/file.xml}"
  [ -f "$RB" ] || skip "PANCCADIA bog not on this machine"
  BF="$RB"; case "$RB" in *.xml) ( cd "$(dirname "$RB")" && cp "$(basename "$RB")" "$T/file.xml" ) && ( cd "$T" && zip -q pan.bog file.xml && rm -f file.xml ); BF="$T/pan.bog" ;; esac
  run "$BA" "$BF" --module ColdRoomPan --module CompPan --module DashboardPan
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK18  FAIL')" -eq 2 ]   # EvaporatorUnit_1 and _3
  [[ "$output" == *"CHECK14"* ]] && [[ "$output" == *"WARN"* ]]      # ColdRoom_5/EvaporatorUnit2 evapOut
  [[ "$output" != *"CHECK13  FAIL"* ]] && [[ "$output" != *"CHECK16  FAIL"* ]] && [[ "$output" != *"CHECK17  FAIL"* ]]
}
