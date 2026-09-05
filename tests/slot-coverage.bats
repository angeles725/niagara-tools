#!/usr/bin/env bats
# RED-FIRST pins for MM2 exposed-set coverage (campaign 6, PR5b · toolbelt/slot-coverage.sh).
# Contract: openspec/changes/build-n4-module-campaign6/{math-models-mm2-mm3.md, design.md §D6/§5}.
#
# One set-difference metric behind three silent-deploy footguns: empty module.palette (B5),
# missing module.lexicon key (T8), dangling module-include.xml <type> (B12).
#
#   set_coverage(declared, required) -> (pct, missing, extra)
#     present = required & declared
#     pct     = round(100 * |present| / |required|, 1)   if |required| > 0   (integer-tenths in bash)
#             = "N/A"                                      if |required| == 0  (never 100)
#     missing = required - declared      (the coverage gap — the footgun)
#     extra   = declared - required      (dangling lint — reported, NOT scored)
#
# SURFACE (design §5, adopted verbatim):
#   slot-coverage.sh set-coverage <declared-csv> <required-csv>      ("" = empty set)
#   stdout = exactly THREE lines, each prefixed (so none is blank and bats $lines keeps all three):
#       pct=<n.n|N/A>
#       missing=<sorted,deduped,comma-joined | empty after '='>
#       extra=<sorted,deduped,comma-joined | empty after '='>
#   argc != 2 -> exit 2 + `usage: slot-coverage.sh set-coverage …` on stderr (keyed on message).
#   (A second parse mode `[--strict] <module-include.xml> <module.lexicon>` exists; NOT pinned here —
#    the apply worker's bats fixtures cover the CompPan-T8 empty-lexicon WARN, which I verify.)
#
# RED today: toolbelt/slot-coverage.sh does not exist yet -> run returns non-zero with no
# `pct=` line, so every pin FAILS for the right reason. Green once the impl lands the pure
# set_coverage over the CSV args (parse half is separate, fixture-tested).
#
# MUTATIONS each pin must catch (post-green, revert -> a pinned value moves):
#   - denominator |declared| not |required|:  A,B / A,B,C,D -> pct=100.0 != 50.0        (SC2)
#   - extra folded into numerator:            A,B,C,D,X / A,B,C,D -> pct=125.0 != 100.0 (SC3)
#   - 0/0 -> 100 instead of N/A:              "" / "" -> pct=N/A                          (SC5)
#   - missing computed as declared-required:  swaps missing/extra on SC2 and SC3

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SC="$KIT/toolbelt/slot-coverage.sh"
}

@test "SC1: full coverage A,B,C,D / A,B,C,D -> 100.0, no missing, no extra" {
  run "$SC" set-coverage "A,B,C,D" "A,B,C,D"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "pct=100.0" ]
  [ "${lines[1]}" = "missing=" ]
  [ "${lines[2]}" = "extra=" ]
}

@test "SC2: partial A,B / A,B,C,D -> 50.0, missing C,D (bites denominator + missing/extra swap)" {
  run "$SC" set-coverage "A,B" "A,B,C,D"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "pct=50.0" ]
  [ "${lines[1]}" = "missing=C,D" ]
  [ "${lines[2]}" = "extra=" ]
}

@test "SC3: dangling extra A,B,C,D,X / A,B,C,D -> 100.0, extra X NOT scored (bites extra-in-numerator)" {
  run "$SC" set-coverage "A,B,C,D,X" "A,B,C,D"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "pct=100.0" ]
  [ "${lines[1]}" = "missing=" ]
  [ "${lines[2]}" = "extra=X" ]
}

@test "SC4: empty palette, types exist '' / A -> 0.0, missing A (the B5 footgun as 0%)" {
  run "$SC" set-coverage "" "A"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "pct=0.0" ]
  [ "${lines[1]}" = "missing=A" ]
  [ "${lines[2]}" = "extra=" ]
}

@test "SC5: scaffold, zero exposed types '' / '' -> N/A (NOT 100, anti-false-confidence)" {
  run "$SC" set-coverage "" ""
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "pct=N/A" ]
  [ "${lines[1]}" = "missing=" ]
  [ "${lines[2]}" = "extra=" ]
}

@test "SC6: bad arity (1 arg) -> exit 2 + set-coverage usage on stderr" {
  run "$SC" set-coverage "A,B"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: slot-coverage.sh set-coverage"* ]]
}

# ---------------------------------------------------------------------------
# Parse subcommand tests (CompPan T8 fixture: empty lexicon + declared types)
# ---------------------------------------------------------------------------
# Fixture: tests/fixtures/slot-coverage/comppan-t8/
#   module-include.xml  — 3 types (CompPanStatus, CompressorControl, CompressorPan)
#   module.lexicon      — empty file (T8 footgun: slots render raw camelCase)
#
# Named mutation for SC6-parse: return N/A (or 100) when lexicon empty -> SC6-parse flips
# (pct must be 0.0 to prove denominator is |required|, not 0)

setup_parse_fixtures() {
  FIXDIR="$(cd "$BATS_TEST_DIRNAME" && pwd)/fixtures/slot-coverage"
}

@test "SC6-parse: empty lexicon + >=1 declared type -> pct=0.0 + FAIL, exit 1 (CompPan-T8 fixture)" {
  # CAMPAIGN 8 PR5 AMENDMENT (was exit 0/WARN): an empty lexicon with >=1 declared type is not a
  # warning — every operator slot renders raw camelCase (the T8 footgun is a ship-blocker), so it is
  # now a FAIL/exit 1. RED until PR5 lands the promotion; PR5 carries the CHANGELOG line. pct stays
  # 0.0 (denominator is |required|, never 0). Type-level output otherwise unchanged.
  setup_parse_fixtures
  XML="$FIXDIR/comppan-t8/module-include.xml"
  LEX="$FIXDIR/comppan-t8/module.lexicon"
  run "$SC" "$XML" "$LEX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"empty lexicon"* ]]
  [[ "$output" == *"pct=0.0"* ]]
}

# ---------------------------------------------------------------------------
# Dup-keys detection (operationalizes B759/B780 — duplicate bare keys in lexicon)
# ---------------------------------------------------------------------------
# Fixture: tests/fixtures/slot-coverage/dup-keys/
#   module-include.xml  — 1 type (FanMode)
#   module.lexicon      — has 'fan' twice (duplicate bare key)
#
# Named mutation: drop dup-keys detection block -> WARN disappears -> test flips

@test "dup-keys: duplicate bare key in lexicon emits WARN (exit 0; --strict -> exit 1)" {
  setup_parse_fixtures
  XML="$FIXDIR/dup-keys/module-include.xml"
  LEX="$FIXDIR/dup-keys/module.lexicon"
  run "$SC" "$XML" "$LEX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"slot-coverage: WARN dup-keys:"* ]]

  run "$SC" --strict "$XML" "$LEX"
  [ "$status" -eq 1 ]
}

# ---------------------------------------------------------------------------
# CAMPAIGN 8 PR5 — per-slot mode (@NiagaraProperty OPERATOR slots vs lexicon keys).
# Type-level (parse) output is unchanged; this is a NEW mode. Real shapes: ColdRoomPan 19 missing
# operator keys, DashboardPan-rt stale `differential`, chihuahua empty lexicon (SC6-parse amendment).
#
# SURFACE (design D6, confirmed):
#   slot-coverage.sh per-slot <module-include.xml> <module.lexicon> <src-dir>  (design D6, design.md:136-139)
#     prints `pct=<n.n> (per-slot)` plus:
#     MISSING <slot>  an OPERATOR @NiagaraProperty slot with no lexicon key (raw camelCase in UI)
#     STALE   <key>   a lexicon key that matches no declared slot (dead translation)
#   Non-operator slots are NOT required to have a key (the OPERATOR gate).
#
# RED today: no per-slot mode -> the MISSING/STALE rows never appear -> each pin fails for the right
# reason (mode absent). NAMED MUTATION (post-green): drop the stale-key diff -> STALE rows vanish (SP2).
FX_PS="$(cd "$BATS_TEST_DIRNAME" && pwd)/fixtures/slot-coverage/per-slot"

@test "SP1: per-slot flags an OPERATOR slot with no lexicon key as MISSING (naming the slot)" {
  run "$SC" per-slot "$FX_PS/module-include.xml" "$FX_PS/module.lexicon" "$FX_PS/src"
  [[ "$output" == *"MISSING"* ]] && [[ "$output" == *"setpoint"* ]]
}

@test "SP2: per-slot flags a lexicon key matching no slot as STALE (differential) [mutation target]" {
  run "$SC" per-slot "$FX_PS/module-include.xml" "$FX_PS/module.lexicon" "$FX_PS/src"
  [[ "$output" == *"STALE"* ]] && [[ "$output" == *"differential"* ]]
}

@test "SP3: a covered operator slot AND a non-operator slot are NOT flagged (coverage + OPERATOR gate)" {
  run "$SC" per-slot "$FX_PS/module-include.xml" "$FX_PS/module.lexicon" "$FX_PS/src"
  [[ "$output" == *"setpoint"* ]]          # anchor: per-slot analysis ran (setpoint is MISSING)
  [[ "$output" != *"fanMode"* ]]           # fanMode has slot+key -> covered, not reported
  [[ "$output" != *"internalState"* ]]     # READONLY (non-operator) -> not required to have a key
}

@test "SP4: an OPERATOR slot under src/.deploy-baseline/ is NOT counted MISSING (D9b dot-dir prune)" {
  # Anchored on setpoint (the real MISSING slot in the live src) so this is RED today (mode absent),
  # not a trivial pass. RED until per-slot lands + prunes dot-dirs. NAMED MUTATION (post-green):
  # remove the dot-dir prune -> staleKnob (no lexicon key) is scanned and reported MISSING -> SP4 flips.
  run "$SC" per-slot "$FX_PS/module-include.xml" "$FX_PS/module.lexicon" "$FX_PS/src"
  [[ "$output" == *"setpoint"* ]]        # anchor: per-slot analysis ran on the live src
  [[ "$output" != *"staleKnob"* ]]       # the .deploy-baseline slot is pruned, never counted
}
