#!/usr/bin/env bats
# RED-FIRST pins for bog-audit.sh (campaign 8 PR10, shard c8-audit/bog.md). Audits a station config.bog
# against the deployed module source. Row `<CHECK_ID>  PASS|FAIL|WARN  <path>  <detail>`; exit 0 clean /
# 1 any FAIL / 3 parse error, prefix not found, or python3 missing.
#
# Grammar (aligned to the REAL PANCCADIA/MX60 bogs): a COMPONENT is `<p n='Name' h='handle' t='PFX:Type'>`
# (single-quoted, carries a handle, optional m='pfx=module' on first use of a prefix); a VALUE property is
# `<p n="slot" t="b:type" v="..."/>` (double-quoted); an ACTION is `<a n="name" f="flags"/>` present only
# when flags differ from the class default; a LINK is `<p n='Link' t='b:Link'>` nesting
# `<p n="sourceOrd" v="h:handle"/>`, `<p n="sourceSlotName" v="..."/>`, `<p n="targetSlotName" v="..."/>`.
# Flags h/o/r/s/t (L = link-locked, ignored). Depth by one-element-per-line counting.
#
# Bound checks: CHECK1 inventory INFO · CHECK2 flag-drift WARN (--strict FAIL) · CHECK3 out-of-facet WARN ·
# CHECK4 transient-persisted WARN · CHECK5 schema-drift bog-extra FAIL · CHECK6 src-missing WARN ·
# CHECK7 link-dangling FAIL · CHECK8 hoa-leftover WARN · CHECK9 orphan-handle FAIL · CHECK10 duplicate-handle
# FAIL · CHECK11 proxy-link-safety (B810/B816) · CHECK12 dashboard-write-to-LINK_TARGET WARN (B816).
# CHECK2/3/4/5/6/7 need --source-dir → SKIP without it; CHECK1/8/9/10/11/12 run from the bog alone.
#
# RED today: bog-audit.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATIONS per check are noted inline.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  BA="$KIT/toolbelt/bog-audit.sh"
  T="$BATS_TEST_TMPDIR"
  # --- synthetic bog (REAL grammar): CHECK5 orphan (ghostSlot), CHECK3 (differentialUp=-1.5),
  #     CHECK7 dangling (targetSlotName ghostTarget), CHECK2 (intervalExpired f='o'),
  #     CHECK11 (own output -> BooleanWritable r1 with NO fallback = FAIL; r2 WITH fallback = clean). ---
  cat > "$T/synthetic.xml" <<'XML'
<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
 <p n='ColdRoom_3' h='c1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n="differentialUp" t="b:double" v="-1.5"/>
  <p n="ghostSlot" t="b:double" v="0"/>
  <p n='DefrostController' h='44b84' t='CRP:DefrostController'>
   <a n="intervalExpired" f="o"/>
   <p n="interval" t="b:RelTime" v="1800000"/>
   <p n="resistanceOut" t="b:StatusBoolean" v="false"/>
  </p>
  <p n='Evap1' h='e1' t='CRP:EvaporatorUnit'>
   <p n="evapOut" t="b:StatusBoolean" v="false"/>
   <p n='DangLink' t='b:Link'>
    <p n="sourceOrd" v="h:44b84"/>
    <p n="sourceSlotName" v="evapOut"/>
    <p n="targetSlotName" v="ghostTarget"/>
   </p>
  </p>
 </p>
 <p n='NrioNetwork' h='n1' m='nrio=nrio' t='nrio:NrioNetwork'>
  <p n='io' h='n2' t='nrio:Nrio34Module'>
   <p n='ro1' h='r1' t='c:BooleanWritable'>
    <p n='InLink' t='b:Link'>
     <p n="sourceOrd" v="h:44b84"/>
     <p n="sourceSlotName" v="resistanceOut"/>
     <p n="targetSlotName" v="in10"/>
    </p>
   </p>
   <p n='ro2' h='r2' t='c:BooleanWritable'>
    <p n='fallback' t='b:StatusBoolean'><p n="value" v="false"/></p>
    <p n='InLink' t='b:Link'>
     <p n="sourceOrd" v="h:e1"/>
     <p n="sourceSlotName" v="evapOut"/>
     <p n="targetSlotName" v="in10"/>
    </p>
   </p>
  </p>
 </p>
</bajaObjectGraph>
XML
  ( cd "$T" && cp synthetic.xml file.xml && zip -q synthetic.bog file.xml && rm -f file.xml )
  # --- clean bog: differentialUp in range, action hidden, no orphan/dangling, writable has a fallback ---
  cat > "$T/clean.xml" <<'XML'
<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
 <p n='ColdRoom_3' h='c1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n="differentialUp" t="b:double" v="1.5"/>
  <p n='DefrostController' h='44b84' t='CRP:DefrostController'>
   <p n="interval" t="b:RelTime" v="1800000"/>
  </p>
 </p>
</bajaObjectGraph>
XML
  ( cd "$T" && cp clean.xml file.xml && zip -q clean.bog file.xml && rm -f file.xml )
  # --- source-dir: declares differentialUp (MIN 0), interval, intervalExpired hidden, evapOut/resistanceOut;
  #     NOT ghostSlot, NOT ghostTarget ---
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
@NiagaraProperty(name="resistanceOut", type="BStatusBoolean", defaultValue="new BStatusBoolean()")
@NiagaraAction(name="intervalExpired", flags=Flags.HIDDEN)
public class BDefrostController {}
JAVA
  cat > "$T/src/com/angeles/ColdRoomPan/BEvaporatorUnit.java" <<'JAVA'
package com.angeles.ColdRoomPan;
@NiagaraProperty(name="evapOut", type="BStatusBoolean", defaultValue="new BStatusBoolean()")
public class BEvaporatorUnit {}
JAVA
  # real bogs (local-only smokes): C8_PANCCADIA_BOG overrides; else the extracted copy in the shard scratchpad
  BOGX="/tmp/claude-1000/-home-cristian-niagara-research/dd08d593-6527-4dc7-885c-938ea923b73f/scratchpad/bogx"
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

@test "BA3: CHECK7 — a link targetSlotName absent from source (ghostTarget) FAILs link-dangling" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK7"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "BA4: CHECK2 — un-hidden intervalExpired (f='o') WARNs; --strict makes it FAIL" {
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

@test "BA6: without --source-dir, CHECK2/3/4/5/6/7 SKIP; CHECK1/8/9/10/11/12 still run" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"CHECK5"* ]]
  [[ "$output" == *"CHECK1"* ]]
}

@test "BA7: python3 missing -> exit 3 (command -v python3 guard)" {
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

# ---- CHECK11 proxy-link-safety (B810/B816) --------------------------------
# A link whose source is an own-module output and whose target is a c:BooleanWritable/c:NumericWritable
# under a driver network: the target without an explicit <fallback> holds its last commanded state on a
# component stop/reload -> FAIL (a relay stuck ON is a safety issue). r1 (own output -> BooleanWritable,
# NO fallback) FAILs; r2 (same, WITH a fallback) does not. NAMED MUTATION: stop resolving the writable's
# fallback slot -> r2 also FAILs (the check no longer distinguishes).
@test "BA9: CHECK11 — an own output linked to a BooleanWritable with NO fallback FAILs proxy-link-safety" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK11"* ]] && [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"ro1"* ]]
}

@test "BA10: CHECK11 — a writable WITH an explicit fallback does NOT FAIL (ro2 is clean)" {
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan
  [[ "$output" == *"CHECK11"* ]]            # the check ran (ro1 above) ...
  [[ "$output" != *"CHECK11  FAIL  ro2"* ]] # ... but ro2, which has a fallback, is not a CHECK11 FAIL
}

# ---- CHECK12 dashboard-write-to-LINK_TARGET (B816, advisory) --------------
# A slot written by the module's own servlet/dashboard that is ALSO a link target in the bog: the
# servlet write is ephemeral (loses to the link) -> WARN. Synthetic only (per the lead) unless a real
# case is found. Here in10 is a link target AND (marked in the source) a servlet-written slot.
@test "BA11: CHECK12 — a servlet-written slot that is also a link target WARNs (advisory)" {
  # source-dir gives the servlet-writes annotation; synthetic.bog has in10 as a link target.
  mkdir -p "$T/src/com/angeles/ColdRoomPan"
  cat > "$T/src/com/angeles/ColdRoomPan/DashboardServlet.java" <<'JAVA'
package com.angeles.ColdRoomPan;
public class DashboardServlet {
  void write(WebOp op) { resolve(op).set("in10", op.getRequest().getParameter("v")); } // dashboard writes in10
}
JAVA
  run "$BA" "$T/synthetic.bog" --module ColdRoomPan --source-dir "$T/src"
  [[ "$output" == *"CHECK12"* ]] && [[ "$output" == *"WARN"* ]] && [[ "$output" == *"in10"* ]]
}

# ---- Real-bog local smokes (SKIP-gated; local bless evidence, never a fake PASS) -------------------
# to_bog <path>: echo a .bog zip path, wrapping an extracted file.xml when needed.
to_bog() { case "$1" in *.xml) ( cd "$(dirname "$1")" && cp "$(basename "$1")" "$T/file.xml" ) && ( cd "$T" && zip -q real.bog file.xml && rm -f file.xml ); echo "$T/real.bog" ;; *) echo "$1" ;; esac; }

@test "BA-smoke-check11: PANCCADIA -> exactly 17 CHECK11 FAIL (own-linked writables with no explicit fallback), 0 CHECK9/10 (bog-alone)" {
  RB="${C8_PANCCADIA_BOG:-$BOGX/panccadia/file.xml}"
  [ -f "$RB" ] || skip "PANCCADIA bog not on this machine (set C8_PANCCADIA_BOG)"
  BF="$(to_bog "$RB")"
  # CHECK11 is bog-alone (no --source-dir). Count fixed by the corrected parser: 22 own-linked targets,
  # 5 with an explicit non-null fallback, 17 without -> 17 FAIL. The class-default fallback is NULL and
  # does NOT count as safe (the relay holds last state on a null command); writeOnUp is a separate path.
  run "$BA" "$BF" --module ColdRoomPan --module CompPan --module DashboardPan
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK11  FAIL')" -eq 17 ]
  [[ "$output" != *"CHECK9  FAIL"* ]] && [[ "$output" != *"CHECK10  FAIL"* ]]
}

@test "BA-smoke-check2: PANCCADIA + real source -> exactly one CHECK2 WARN, 0 CHECK5/7 (SKIP without a real source-dir)" {
  RB="${C8_PANCCADIA_BOG:-$BOGX/panccadia/file.xml}"
  SRC="${C8_PANCCADIA_SRC:-}"
  [ -f "$RB" ] || skip "PANCCADIA bog not on this machine"
  [ -n "$SRC" ] && [ -d "$SRC" ] || skip "no real PANCCADIA source-dir (set C8_PANCCADIA_SRC; synthetic src would false-flag CHECK5)"
  BF="$(to_bog "$RB")"
  run "$BA" "$BF" --module ColdRoomPan --module CompPan --module DashboardPan --source-dir "$SRC"
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK2  WARN')" -eq 1 ]
  [[ "$output" != *"CHECK5  FAIL"* ]] && [[ "$output" != *"CHECK7  FAIL"* ]]
}

# ---- CHECK5 inherited frozen slot disambiguation (BA12) ---------------------
# A frozen bog slot (no t= attribute) on a class that extends a framework superclass
# outside the source tree is WARN "possibly inherited", not FAIL.
# A frozen bog slot on a class extending BComponent (or an own-module class) is FAIL.
# NAMED MUTATION: change the extends clause from BWebServlet to BComponent -> WARN becomes FAIL.
@test "BA12: CHECK5 — frozen slot on a framework-extending class is WARN not FAIL; BComponent-extending ghost stays FAIL" {
  local T12="$T/ba12"
  mkdir -p "$T12/src/com/company/MyMod"

  # BFoo extends BWebServlet (framework class, not in our source tree)
  cat > "$T12/src/com/company/MyMod/BFoo.java" <<'JAVA'
package com.company.MyMod;
@NiagaraProperty(name="myProp", type="BString", defaultValue="BString.make(\"\")")
public class BFoo extends BWebServlet {}
JAVA

  # BBar extends BComponent — a ghost frozen slot here must still be FAIL
  cat > "$T12/src/com/company/MyMod/BBar.java" <<'JAVA'
package com.company.MyMod;
@NiagaraProperty(name="realProp", type="BString", defaultValue="BString.make(\"\")")
public class BBar extends BComponent {}
JAVA

  # bog: BFoo has 'servletName' (frozen, no t=) + BBar has 'ghostFrozen' (frozen, no t=)
  local BOG12="$T12/test12.bog"
  local XML12="$T12/file.xml"
  cat > "$XML12" <<'XML'
<?xml version='1.0' encoding='UTF-8'?>
<bajaObjectGraph version='4.0'>
 <p n='Foo1' h='f1' m='MM=MyMod' t='MM:Foo'>
  <p n="myProp" t="b:String" v="hello"/>
  <p n="servletName" v="myfoo"/>
 </p>
 <p n='Bar1' h='b1' m='MM=MyMod' t='MM:Bar'>
  <p n="realProp" t="b:String" v="world"/>
  <p n="ghostFrozen" v="ghost"/>
 </p>
</bajaObjectGraph>
XML
  ( cd "$T12" && cp file.xml file.xml.bak && zip -q test12.bog file.xml && rm -f file.xml )

  run "$BA" "$BOG12" --module MyMod --source-dir "$T12/src"
  # servletName (frozen, BFoo extends BWebServlet) -> WARN on its own line, no FAIL line for it
  grep -qE '^CHECK5[[:space:]]+WARN.*servletName' <<< "$output"
  ! grep -qE '^CHECK5[[:space:]]+FAIL.*servletName' <<< "$output"
  # ghostFrozen (frozen, BBar extends BComponent) -> FAIL on its own line
  grep -qE '^CHECK5[[:space:]]+FAIL.*ghostFrozen' <<< "$output"
  # Exit must be 1 (at least one FAIL)
  [ "$status" -eq 1 ]
}

<<<<<<< HEAD
@test "CHECK12-pin: CHECK12 (dashboard-write-link WARN) is present in bog-audit.sh (R19.5 idempotent guard — PR10 merged)" {
  # Task 19.4: since PR10 is already merged, assert CHECK12 is in bog-audit.sh; do NOT re-add it.
  grep -qF 'CHECK12' "$KIT/toolbelt/bog-audit.sh"
  grep -qF 'LINK_TARGET' "$KIT/toolbelt/bog-audit.sh"
=======
# ================================================================
# CHECK13-CHECK19 station-logic pins (campaign 8 PR20, wave3 R20/D17)
# Folded from tests/station-logic.bats (kept self-contained there).
# mkbog helper: creates a .bog zip from stdin XML in $T/<name>.bog.
# ================================================================
_mkbog() { cat > "$T/$1.xml"; ( cd "$T" && cp "$1.xml" file.xml && zip -q "$1.bog" file.xml && rm -f file.xml ); }

@test "SL13: CHECK13 — two distinct sources into the same relay slot FAIL (relay-double-source)" {
  _mkbog c13sl <<'XML'
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
  run "$BA" "$T/c13sl.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK13"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL14: CHECK14 — an OPERATOR output slot with no outgoing relay link WARNs (own-output-unlinked)" {
  _mkbog c14sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_5' h='cr5' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit2' h='e5' t='CRP:EvaporatorUnit'>
   <p n="evapOut" f="o" t="b:StatusBoolean" v="false"/>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c14sl.bog" --module ColdRoomPan
  [[ "$output" == *"CHECK14"* ]] && [[ "$output" == *"WARN"* ]] && [[ "$output" == *"evapOut"* ]]
}

@test "SL15: CHECK15 — C-room-labeled slot sourced from a different unit index WARNs (sensor-crossed-by-name)" {
  _mkbog c15sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='Station' h='st' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit_3' h='e3' t='CRP:EvaporatorUnit'>
   <p n='L1' t='b:Link'><p n="sourceOrd" v="h:e3"/><p n="sourceSlotName" v="C1_temperature"/><p n="targetSlotName" v="tempOut"/></p>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c15sl.bog" --module ColdRoomPan
  [[ "$output" == *"CHECK15"* ]] && [[ "$output" == *"WARN"* ]]
}

@test "SL16: CHECK16 — hasDefrost=true with no DefrostController sibling FAILs" {
  _mkbog c16sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_2' h='cr2' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='Evap' h='ev2' t='CRP:EvaporatorUnit'>
   <p n="hasDefrost" t="b:boolean" v="true"/>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c16sl.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK16"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL17: CHECK17 — ColdRoom_1 whose link names address evap3* (index mismatch) FAILs" {
  _mkbog c17sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_1' h='cr1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='hoaLink' t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="hoa"/><p n="targetSlotName" v="evap3Hoa"/></p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c17sl.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK17"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL18: CHECK18 — an evap unit whose HOA tile (evap3) != freeze tile (evap1) FAILs (tile-number)" {
  _mkbog c18sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='ColdRoom_1' h='cr1' m='CRP=ColdRoomPan' t='CRP:ColdRoom'>
  <p n='EvaporatorUnit_1' h='eu1' t='CRP:EvaporatorUnit'>
   <p n='hoaL'    t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="hoa"/><p n="targetSlotName" v="evap3Hoa"/></p>
   <p n='freezeL' t='b:Link'><p n="sourceOrd" v="h:cr1"/><p n="sourceSlotName" v="frz"/><p n="targetSlotName" v="evap1Freeze"/></p>
  </p>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c18sl.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"CHECK18"* ]] && [[ "$output" == *"FAIL"* ]]
}

@test "SL19: CHECK19 — control component writing to a setpoint slot WARNs (reverse link-direction)" {
  _mkbog c19sl <<'XML'
<?xml version='1.0'?><bajaObjectGraph version='4.0'>
 <p n='DashPanel_1' h='dp1' m='DPan=DashboardPan' t='DPan:DashUnit'>
  <p n='SpLink' t='b:Link'><p n="sourceOrd" v="h:crp1"/><p n="sourceSlotName" v="temperatureOut"/><p n="targetSlotName" v="temperatureSetpoint"/></p>
 </p>
 <p n='Ctrl_1' h='crp1' m='CRP=ColdRoomPan' t='CRP:EvaporatorUnit'>
  <p n="temperatureOut" f="o" t="b:StatusNumeric" v="0"/>
 </p>
</bajaObjectGraph>
XML
  run "$BA" "$T/c19sl.bog" --module ColdRoomPan --module DashboardPan
  [[ "$output" == *"CHECK19"* ]] && [[ "$output" == *"WARN"* ]]
}

@test "SL-smoke-panccadia: PANCCADIA -> CHECK18 FAIL x2 (ColdRoom_1 EvapUnit_1 and _3), CHECK14 WARN ColdRoom_5, others clean (SKIP if absent)" {
  RB="${C8_PANCCADIA_BOG:-/nonexistent}"
  [ -f "$RB" ] || skip "PANCCADIA bog not on this machine (set C8_PANCCADIA_BOG)"
  BF="$RB"
  case "$RB" in
    *.xml) ( cd "$(dirname "$RB")" && cp "$(basename "$RB")" "$T/file.xml" ) \
           && ( cd "$T" && zip -q pan_sl.bog file.xml && rm -f file.xml )
           BF="$T/pan_sl.bog" ;;
  esac
  run "$BA" "$BF" --module ColdRoomPan --module CompPan --module DashboardPan
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c 'CHECK18  FAIL')" -eq 2 ]
  [[ "$output" == *"CHECK14"* ]] && [[ "$output" == *"WARN"* ]]
  [[ "$output" != *"CHECK13  FAIL"* ]]
  [[ "$output" != *"CHECK16  FAIL"* ]]
  [[ "$output" != *"CHECK17  FAIL"* ]]
>>>>>>> 5f0f377 (feat(bog-audit): K19 routing + SL fixtures + bog-audit.bats extension (campaign 8 PR20))
}
