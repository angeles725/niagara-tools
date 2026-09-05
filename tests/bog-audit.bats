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
