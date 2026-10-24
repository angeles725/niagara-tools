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
@test "SR9 unreadable module-include.xml -> exit 4 (env)" {
  local tmp
  tmp=$(mktemp -d)
  mkdir -p "$tmp/before/com/angeles/MinimalPan" "$tmp/after/com/angeles/MinimalPan"
  cp "$FX/add_slot/before/com/angeles/MinimalPan/BMinimalPan.java" \
     "$tmp/before/com/angeles/MinimalPan/"
  cp "$FX/add_slot/before/module-include.xml" "$tmp/before/"
  cp "$FX/add_slot/after/com/angeles/MinimalPan/BMinimalPan.java" \
     "$tmp/after/com/angeles/MinimalPan/"
  touch "$tmp/after/module-include.xml"
  chmod 000 "$tmp/after/module-include.xml"
  run "$SR" "$tmp/before" "$tmp/after"
  chmod 644 "$tmp/after/module-include.xml"
  rm -rf "$tmp"
  [ "$status" -eq 4 ]
}
@test "SR10 real-shape (spaced = and multi-line annotations) -> OUTAGE; rows show add_slot + retype_simple" {
  pair real_shape
  [ "$status" -eq 2 ]
  [[ "$output" == *"verdict=OUTAGE"* ]]
  [[ "$output" == *"retype_simple"* ]]
  [[ "$output" == *"add_slot"* ]]
}

@test "SR-CSV the embedded Go classifier table byte-equals tests/fixtures/schema-risk/b795-795.4.csv" {
  # The table moved from the bash CSV_TABLE heredoc to go/internal/schemarisk/b795-795.4.csv (go:embed).
  cmp "$KIT/../go/internal/schemarisk/b795-795.4.csv" "$FX/b795-795.4.csv"
}

@test "SR-multi several rows of one category list in declaration order (after for add_slot, before for the rest)" {
  # Mutation: SR-multi -- listing a category by name (or map order) puts alpha before zeta and fails here
  pair multi_row
  [ "$status" -eq 2 ]
  [ "$output" = "$(cat "$BATS_TEST_DIRNAME/../go/internal/schemarisk/testdata/multi_row.golden")" ]
}

@test "SR-locked: an unreadable sub-directory is an env error (exit 4), never a clean pass" {
  # Mutation: SR-locked -- a raw find walk skips the locked directory and passes clean.
  mkdir -p "$BATS_TEST_TMPDIR/before" "$BATS_TEST_TMPDIR/after"
  for d in before after; do
    printf '<module><type name="A"/></module>\n' > "$BATS_TEST_TMPDIR/$d/module-include.xml"
    printf 'class A {}\n' > "$BATS_TEST_TMPDIR/$d/A.java"
  done
  mkdir -p "$BATS_TEST_TMPDIR/after/locked"
  chmod 000 "$BATS_TEST_TMPDIR/after/locked"
  if [ -r "$BATS_TEST_TMPDIR/after/locked" ]; then chmod 755 "$BATS_TEST_TMPDIR/after/locked"; skip "running as root"; fi
  run "$SR" "$BATS_TEST_TMPDIR/before" "$BATS_TEST_TMPDIR/after"
  chmod 755 "$BATS_TEST_TMPDIR/after/locked"
  [ "$status" -eq 4 ]
  [[ "$output" == *"error: cannot list every file under $BATS_TEST_TMPDIR/after: "* ]]
}

@test "SR-empty-before a before snapshot with no slot adds every after slot (SAFE), never removes them" {
  # Mutation: SR-empty-before -- reading the after slots as the before side (awk NR==FNR on an empty first file) prints remove_slot rows, LOSSY
  mkdir -p "$BATS_TEST_TMPDIR/before"
  cp "$FX/add_slot/before/module-include.xml" "$BATS_TEST_TMPDIR/before/"
  run "$SR" "$BATS_TEST_TMPDIR/before" "$FX/add_slot/after"
  [ "$status" -eq 0 ]
  [[ "$output" == *"add_slot  MinimalPan.setpoint"* ]]
  [[ "$output" != *"remove_"* ]]
  [[ "$output" == *"verdict=SAFE"* ]]
}

@test "SR-unreadable-java an unreadable *.java is an env error (exit 4, no verdict), never a partial-read verdict" {
  # Mutation: SR-unreadable-java -- skipping the unreadable file (the bash awk fatal) computes LOSSY removals from a partial read
  mkdir -p "$BATS_TEST_TMPDIR/after"
  cp -r "$FX/add_slot/after/." "$BATS_TEST_TMPDIR/after/"
  local f="$BATS_TEST_TMPDIR/after/com/angeles/MinimalPan/BMinimalPan.java"
  chmod 000 "$f"
  if [ -r "$f" ]; then chmod 644 "$f"; skip "running as root"; fi
  run "$SR" "$FX/add_slot/before" "$BATS_TEST_TMPDIR/after"
  chmod 644 "$f"
  [ "$status" -eq 4 ]
  [[ "$output" == *"error: cannot read $f: "* ]]
  [[ "$output" != *"verdict="* ]]
}

@test "SR-unbalanced an annotation whose parentheses never close is an UNKNOWN row (OUTAGE), never a silent skip (L3)" {
  # Mutation: SR-unbalanced -- joining the open annotation to the end of input swallows the slot after it and reads SAFE
  mkdir -p "$BATS_TEST_TMPDIR/after/p"
  cp "$FX/add_slot/before/module-include.xml" "$BATS_TEST_TMPDIR/after/"
  sed 's/@NiagaraProperty(name="setpoint", type="double", defaultValue="0.0", flags=Flags.SUMMARY|Flags.OPERATOR)/@NiagaraProperty(name="setpoint", type="double"/' \
    "$FX/add_slot/before/com/angeles/MinimalPan/BMinimalPan.java" > "$BATS_TEST_TMPDIR/after/p/BMinimalPan.java"
  run "$SR" "$FX/add_slot/before" "$BATS_TEST_TMPDIR/after"
  [ "$status" -eq 2 ]
  [[ "$output" == *"OUTAGE  UNKNOWN  MinimalPan.<unparsed>@$BATS_TEST_TMPDIR/after/p/BMinimalPan.java:"* ]]
  [[ "$output" == *"verdict=OUTAGE"* ]]
}
