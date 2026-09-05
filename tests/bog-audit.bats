#!/usr/bin/env bats
# RED-FIRST pins for bog-audit.sh (campaign 8 PR10, shard scratchpad/c8-audit/bog.md). Audits a
# station config.bog against the deployed module source. Row `<CHECK_ID>  PASS|FAIL|WARN  <path>  <detail>`;
# exit 0 clean / 1 any FAIL / 3 parse error, prefix not found, or python3 missing.
#
# Bound checks (D10 + lead decision): CHECK1 inventory INFO · CHECK2 flag-drift WARN (--strict FAIL) ·
# CHECK3 out-of-facet WARN · CHECK4 transient-persisted WARN · CHECK5 schema-drift bog-extra FAIL ·
# CHECK6 src-missing WARN · CHECK7 link-dangling FAIL · CHECK8 hoa-leftover WARN · CHECK9 orphan-handle
# FAIL · CHECK10 duplicate-handle FAIL. CHECK2/3/4/5/6/7 need --source-dir (annotations) → SKIP without it;
# CHECK1/8/9/10 run from the bog alone.
#
# RED today: bog-audit.sh does not exist -> every pin fails for the right reason (tool absent). The
# synthetic bog follows the shard grammar (m='PREFIX=MODULE'; <p n h t f v>; <a n f>; links inside the
# target via sourceOrd h:<handle>/sourceSlotName/targetSlotName; flags h/o/r/s/t, L=link-locked). No real
# config.bog is on this machine, so the exact bytes are the parser's to confirm at apply time.
#
# NAMED MUTATIONS (post-green, per check): skip handle resolution -> CHECK7 dangling stops FAILing;
# accept an un-hidden action as default -> CHECK2 stops WARNing; ignore facet MIN -> CHECK3 stops WARNing;
# treat a bog-extra prop as known -> CHECK5 stops FAILing.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  BA="$KIT/toolbelt/bog-audit.sh"
  T="$BATS_TEST_TMPDIR"
  # --- synthetic bog: orphan prop (CHECK5), differentialUp=-1.5 (CHECK3), dangling link (CHECK7),
  #     intervalExpired f='o' (CHECK2); unique handles so CHECK9/10 stay clean ---
  cat > "$T/synthetic.xml" <<'XML'
<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
  <p m='CRP=ColdRoomPan'/>
  <CRP:BColdRoom n='ColdRoom_3' h='c1'>
    <p n='differentialUp' h='c1a' t='b:double' f='' v='-1.5'/>
    <p n='ghostSlot' h='c1b' t='b:double' f='' v='0'/>
    <CRP:BDefrostController n='DefrostController' h='44b84'>
      <a n='intervalExpired' f='o'/>
      <p n='interval' h='44b85' t='b:RelTime' f='' v='30min'/>
    </CRP:BDefrostController>
    <CRP:BEvaporatorUnit n='Evap1' h='e1'>
      <p n='runCmd' h='e1a' t='b:StatusBoolean' f=''>
        <sourceOrd h:c1a/>
        <sourceSlotName>evapOut</sourceSlotName>
        <targetSlotName>ghostTarget</targetSlotName>
      </p>
    </CRP:BEvaporatorUnit>
  </CRP:BColdRoom>
</bajaObjectGraph>
XML
  ( cd "$T" && cp synthetic.xml file.xml && zip -q synthetic.bog file.xml && rm -f file.xml )
  # --- clean bog: same shapes, all valid (no orphan, no dangling, differentialUp in range, action hidden) ---
  cat > "$T/clean.xml" <<'XML'
<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
  <p m='CRP=ColdRoomPan'/>
  <CRP:BColdRoom n='ColdRoom_3' h='c1'>
    <p n='differentialUp' h='c1a' t='b:double' f='' v='1.5'/>
    <CRP:BDefrostController n='DefrostController' h='44b84'>
      <p n='interval' h='44b85' t='b:RelTime' f='' v='30min'/>
    </CRP:BDefrostController>
  </CRP:BColdRoom>
</bajaObjectGraph>
XML
  ( cd "$T" && cp clean.xml file.xml && zip -q clean.bog file.xml && rm -f file.xml )
  # --- source-dir (annotations): declares differentialUp (MIN 0), interval, intervalExpired hidden,
  #     runCmd/evapOut — but NOT ghostSlot and NOT ghostTarget ---
  mkdir -p "$T/src/com/angeles/ColdRoomPan"
  cat > "$T/src/com/angeles/ColdRoomPan/BColdRoom.java" <<'JAVA'
package com.angeles.ColdRoomPan;
@NiagaraProperty(name="differentialUp", type="double", defaultValue="1.5",
  facets=@Facet("BFacets.make(BFacets.MIN, BDouble.make(0d))"))
public class BColdRoom {}
JAVA
  cat > "$T/src/com/angeles/ColdRoomPan/BDefrostController.java" <<'JAVA'
package com.angeles.ColdRoomPan;
@NiagaraProperty(name="interval", type="BRelTime", defaultValue="BRelTime.make(1800000)")
@NiagaraAction(name="intervalExpired", flags=Flags.HIDDEN)
public class BDefrostController {}
JAVA
  cat > "$T/src/com/angeles/ColdRoomPan/BEvaporatorUnit.java" <<'JAVA'
package com.angeles.ColdRoomPan;
@NiagaraProperty(name="runCmd", type="BStatusBoolean", defaultValue="new BStatusBoolean()")
@NiagaraProperty(name="evapOut", type="BStatusBoolean", defaultValue="new BStatusBoolean()")
public class BEvaporatorUnit {}
JAVA
}

@test "BA1: CHECK5 — a bog property with no matching source slot (ghostSlot) FAILs, exit 1" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK5"* ]] && [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"ghostSlot"* ]]
}

@test "BA2: CHECK3 — differentialUp=-1.5 below the source MIN 0 WARNs out-of-facet" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [[ "$output" == *"CHECK3"* ]] && [[ "$output" == *"WARN"* ]] && [[ "$output" == *"differentialUp"* ]]
}

@test "BA3: CHECK7 — a link to a slot absent from source (ghostTarget) FAILs link-dangling" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK7"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "BA4: CHECK2 — un-hidden intervalExpired (f='o') WARNs flag-drift; --strict makes it FAIL" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [[ "$output" == *"CHECK2"* ]] && [[ "$output" == *"WARN"* ]] && [[ "$output" == *"intervalExpired"* ]]
  run "$BA" --strict "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK2"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "BA5: a clean bog -> exit 0 (no FAIL)" {
  run "$BA" "$T/clean.bog" --module ColdRoomPan --source-dir "$T/src"
  [ "$status" -eq 0 ]
}

@test "BA6: without --source-dir, CHECK2/3/4/5/6/7 emit SKIP rows (CHECK1/8/9/10 still run)" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"CHECK5"* ]]   # source-dependent check skipped
  [[ "$output" == *"CHECK1"* ]]                                   # inventory still runs from the bog
}

@test "BA7: python3 missing -> exit 3 with a message (command -v python3 guard)" {
  NOPY="$T/nopy"; mkdir -p "$NOPY"
  for t in bash sh env grep sed awk unzip dirname basename cat sort head printf; do
    p="$(command -v "$t" 2>/dev/null)"; [ -n "$p" ] && ln -sf "$p" "$NOPY/"
  done
  PATH="$NOPY" run bash "$BA" "$T/synthetic.bog" --module ColdRoomPan
  [ "$status" -eq 3 ]
}

@test "BA8: no config.bog argument -> exit 3 (usage)" {
  run "$BA"
  [ "$status" -eq 3 ]
}
