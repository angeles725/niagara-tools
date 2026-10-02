#!/usr/bin/env bats
# RED-FIRST pins for lint-link-target-flags.sh (fold-2026-10-02-pending-retros WU6b).
#   LTF1 FAIL: a link-in target slot (comment convention "link-in" / "Linked from" /
#              "Commissioning link" / "written by BLink", or a Table 2 row of --wiring-map)
#              that carries Flags.READONLY — Workbench LinkCheck refuses a READONLY target,
#              so the mirror silently never updates. [ev: retro panccadia-commissioning-lessons Δ1]
#   LTF2 FAIL: an OPERATOR *Mode / *Hoa slot that carries Flags.TRANSIENT — the operator
#              choice reverts to its default on restart and a facade->control link re-pushes it.
#              [ev: retro panccadia-persistent-config-hoa Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LTF="$KIT/toolbelt/lint-link-target-flags.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags"
}

@test "LTF-comment: READONLY slot whose comment names a Commissioning link / BLink -> LTF1 FAIL exit 1" {
  run "$LTF" "$FX/comment/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-link-target-flags"* ]]
  [[ "$output" == *"LTF1"* ]]
  [[ "$output" == *"evap1FreezeActive"* ]]
  [[ "$output" != *"comp1State"* ]]        # SUMMARY-only link-in slot is linkable -> not a subject
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 1 ]
}

@test "LTF-map: READONLY slot listed in --wiring-map Table 2 (control -> facade) -> LTF1 FAIL exit 1" {
  run "$LTF" --wiring-map "$FX/map/docs/wiring-map.md" "$FX/map/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF1"* ]]
  [[ "$output" == *"evap2InDrip"* ]]
  [[ "$output" != *"temperatureSetpoint"* ]]   # Table 1 rows are facade SOURCES, never targets here
}

@test "LTF-map-neg: the SAME READONLY slot without --wiring-map and without a link comment is not a known target -> exit 0" {
  run "$LTF" "$FX/map/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LTF-transient: OPERATOR *Mode / *Hoa slots with TRANSIENT -> LTF2 FAIL exit 1 (one row each)" {
  run "$LTF" "$FX/transient/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF2"* ]]
  [[ "$output" == *"comp3Mode"* ]]
  [[ "$output" == *"fanHoa"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 2 ]
}

@test "LTF-clean: SUMMARY-only link-in, self-set READONLY anchor, persisted OPERATOR mode, TRANSIENT computed status mode -> exit 0; dot-dir pruned" {
  run "$LTF" "$FX/clean/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"oldFlag"* ]] && [[ "$output" != *"oldMode"* ]]
}

@test "LTF-usage: no argument, a non-directory, a missing --wiring-map file, or no Java sources -> exit 3" {
  run "$LTF"
  [ "$status" -eq 3 ]
  run "$LTF" "/nonexistent/__ltf__"
  [ "$status" -eq 3 ]
  run "$LTF" --wiring-map "/nonexistent/__map__.md" "$FX/map/src"
  [ "$status" -eq 3 ]
  E="$BATS_TEST_TMPDIR/empty"; mkdir -p "$E"
  run "$LTF" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]]
}

# polish-2026-10-02 P2b (#199 WU6b): generate-wiring-map.sh scaffolds Table 2 from every SUMMARY/READONLY
# slot, including slots the component sets itself (timer anchors, computed status) with a `_(fill)_`
# Source RT slot. Only a row that names a source slot is a link-in target; an unfilled scaffold row or a
# Source cell marked self / n/a / — is not. Named mutation LTF-selfset (harvest every first-column slot
# again) -> LTF-selfset flips.
@test "LTF-selfset: Table 2 rows with an unfilled or self Source RT slot are not link-in targets -> exit 0" {
  local m="$BATS_TEST_TMPDIR/selfset.md"
  cat > "$m" <<'MD'
## Table 2 — SUMMARY display slots (control → facade)

| Facade slot | Workbench display name | Source RT slot | Full ord | Physical instance / crossing notes |
|-------------|------------------------|----------------|----------|-------------------------------------|
| `evap2InDrip` | Evaporadora 2 en goteo | _(fill)_ | _(fill)_ | |
MD
  run "$LTF" --wiring-map "$m" "$FX/map/src"
  [ "$status" -eq 0 ]
  local v; for v in 'self' 'n/a' '—' '-' '(self-set)' ''; do
    sed -i "s#^| \`evap2InDrip\` | Evaporadora 2 en goteo | [^|]* |#| \`evap2InDrip\` | Evaporadora 2 en goteo | $v |#" "$m"
    run "$LTF" --wiring-map "$m" "$FX/map/src"
    [ "$status" -eq 0 ] || { echo "source cell '$v' -> $output"; return 1; }
  done
  sed -i "s#^| \`evap2InDrip\` | Evaporadora 2 en goteo | [^|]* |#| \`evap2InDrip\` | Evaporadora 2 en goteo | \`inDrip\` |#" "$m"
  run "$LTF" --wiring-map "$m" "$FX/map/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF1: link-in target \"evap2InDrip\" (wiring-map Table 2)"* ]]
}

# polish-2026-10-02 P2c (#199, P2b review advisories): the self-set sentinels are whole-cell values
# (`_(fill)_`, self, self-set, n/a, none, a dash, empty), not substrings, so a real source whose name
# contains "fill" or "self" is still a link-in target; and only the table's header row picks the Source
# column, so a data row whose display name starts with "Source" stays a target.
# Named mutations: LTF-srcname (substring sentinels again) -> LTF-srcname flips; LTF-header (detect the
# header on any row) -> LTF-header flips.
# shellcheck disable=SC2016  # literal markdown backticks in the generated map, not expansions
@test "LTF-srcname: a Source RT slot named fillLevel / selfTestOk / refillDone is a real source -> LTF1 FAIL" {
  local m="$BATS_TEST_TMPDIR/srcname.md" v
  for v in '`fillLevel`' '`selfTestOk`' 'refillDone'; do
    printf '## Table 2 — SUMMARY display slots (control → facade)\n\n| Facade slot | Workbench display name | Source RT slot | Full ord | Notes |\n|---|---|---|---|---|\n| `evap2InDrip` | Drip | %s | | |\n' "$v" > "$m"
    run "$LTF" --wiring-map "$m" "$FX/map/src"
    [ "$status" -eq 1 ] || { echo "source $v -> $output"; return 1; }
    [[ "$output" == *"LTF1: link-in target \"evap2InDrip\" (wiring-map Table 2"* ]] || { echo "source $v -> $output"; return 1; }
  done
}

# polish-2026-10-02 P2d (#199, P2c review advisories): markdown backticks are stripped from the Source
# cell before the whole-cell sentinel match, so `self`, `n/a` or `(self-set)` written in code formatting
# (as the generator guidance prints it) is still self-set. Named mutation LTF-backtick (match the raw
# cell again) -> LTF-backtick flips.
# shellcheck disable=SC2016  # literal markdown backticks in the generated map, not expansions
@test "LTF-backtick: a backticked self / n/a / (self-set) Source RT cell is self-set -> exit 0" {
  local m="$BATS_TEST_TMPDIR/bt.md" v
  for v in '`self`' '`n/a`' '`(self-set)`' '`none`' '`_(fill)_`'; do
    printf '## Table 2 — SUMMARY display slots (control → facade)\n\n| Facade slot | Workbench display name | Source RT slot | Full ord | Notes |\n|---|---|---|---|---|\n| `evap2InDrip` | Drip | %s | | |\n' "$v" > "$m"
    run "$LTF" --wiring-map "$m" "$FX/map/src"
    [ "$status" -eq 0 ] || { echo "source $v -> $output"; return 1; }
  done
}
# shellcheck disable=SC2016  # literal markdown backticks in the generated map, not expansions
@test "LTF-header: a data row whose display name starts with Source is still a target (no column shift)" {
  local m="$BATS_TEST_TMPDIR/hdr.md"
  printf '## Table 2 — SUMMARY display slots (control → facade)\n\n| Facade slot | Workbench display name | Source RT slot | Full ord | Notes |\n|---|---|---|---|---|\n| `evap2InDrip` | Source pressure drip | `inDrip` | | |\n' > "$m"
  run "$LTF" --wiring-map "$m" "$FX/map/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF1: link-in target \"evap2InDrip\" (wiring-map Table 2)"* ]]
}
