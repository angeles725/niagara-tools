#!/usr/bin/env bats
# Pins for commissioning-verify.sh (issue #114, §6.b commissioning punch-list auditor).
# Orchestrates lint-config-sanity.sh, lint-status-parity.sh, lint-recovery-path.sh (per -rt/src),
# and bog-audit.sh CHECK11/CHECK13-19/CHECK20 (when --bog is given) into a single §6.b punch-list.
#
# Row format: STATUS  commissioning  <check>  <detail>
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 3 usage/env.
#
# RED today: commissioning-verify.sh does not exist -> every pin fails for the right reason.

load helpers/stub-toolbelt   # stub_toolbelt <dir> <member> <body> (polish P1c)

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CV="$KIT/toolbelt/commissioning-verify.sh"
  MINIMAL="$KIT/fixtures/MinimalPan"
  BOG_DIR="$BATS_TEST_DIRNAME/fixtures/bog"
  CV_FX="$BATS_TEST_DIRNAME/fixtures/commissioning"
}

@test "CV-usage: no argument -> exit 3 (usage)" {
  run "$CV"
  [ "$status" -eq 3 ]
}

@test "CV-clean: MinimalPan with no --bog -> exit 0, emits config-sanity/parity/recovery rows + SKIP-bog note + MANUAL footer" {
  run "$CV" "$MINIMAL"
  [ "$status" -eq 0 ]
  # Source check rows present
  [[ "$output" == *"config-sanity"* ]]
  [[ "$output" == *"status-parity"* ]]
  [[ "$output" == *"recovery-path"* ]]
  # Bog SKIP rows present (no --bog given)
  [[ "$output" == *"SKIP"*"proxy-link-safety"* ]]
  [[ "$output" == *"SKIP"*"station-logic"* ]]
  # Manual footer rows present
  [[ "$output" == *"MANUAL"*"hot-reload-console"* ]]
  [[ "$output" == *"MANUAL"*"plant-control"* ]]
  [[ "$output" == *"MANUAL"*"per-instance-values"* ]]
  # Summary present
  [[ "$output" == *"commissioning-verify:"* ]]
  [[ "$output" == *"CLEAN"* ]]
  # No FAIL status rows (summary may contain "0 FAIL" — match the row prefix instead)
  [[ "$output" != *"FAIL  commissioning"* ]]
}

@test "CV-bog-clean: station-logic-CHECK13-clean.bog + --module ColdRoomPan -> exit 0, no FAIL row" {
  run "$CV" "$MINIMAL" --bog "$BOG_DIR/station-logic-CHECK13-clean.bog" --module ColdRoomPan
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL  commissioning"* ]]
  [[ "$output" == *"commissioning-verify:"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "CV-bog-fail: station-logic-CHECK13.bog (CHECK11+CHECK13 FAIL) + --module ColdRoomPan -> exit 1, FAIL row" {
  run "$CV" "$MINIMAL" --bog "$BOG_DIR/station-logic-CHECK13.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"commissioning"* ]]
  [[ "$output" == *"ISSUES"* ]]
}

@test "CV-configfail: src fixture with CS1 violation (interval<=duration) -> exit 1, config-sanity FAIL row" {
  run "$CV" "$CV_FX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"commissioning"*"config-sanity"* ]]
  [[ "$output" == *"CS1"* ]]
  [[ "$output" == *"ISSUES"* ]]
}

# ---------------------------------------------------------------------------
# Values owed by the field (BUILD-LOOP §6.b table) — one MANUAL row per slot whose value the field
# still owes, so a safety-adjacent "disabled, no value yet" default cannot silently age out once the
# feature doc is archived. [ev: retro panccadia-commissioning-lessons Δ11]
# [ev: retro panccadia-version-defect-ledger Δ4]
# ---------------------------------------------------------------------------

@test "CV-owed1: --values-owed table -> one MANUAL values-owed row per owed slot; filled slots are not listed; exit unchanged" {
  run "$CV" "$MINIMAL" --values-owed "$CV_FX/values-owed/values-owed.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"MANUAL  commissioning  values-owed  overCurrentLimit: owed by field technician (nameplate RLA), unit A, still at safe default 0 (disabled)"* ]]
  [[ "$output" == *"MANUAL  commissioning  values-owed  lowPressureCutout: owed by refrigeration technician, unit psig, still at safe default 0 (disabled)"* ]]
  [[ "$output" != *"values-owed  stageDelay"* ]]
}

@test "CV-owed2: <module-root>/docs/values-owed.md is read by default" {
  root="$BATS_TEST_TMPDIR/mod"
  cp -r "$MINIMAL" "$root"
  mkdir -p "$root/docs"
  cp "$CV_FX/values-owed/values-owed.md" "$root/docs/values-owed.md"
  run "$CV" "$root"
  [[ "$output" == *"MANUAL  commissioning  values-owed  overCurrentLimit:"* ]]
}

@test "CV-owed3: every value provided -> one PASS values-owed row, no MANUAL values-owed row" {
  run "$CV" "$MINIMAL" --values-owed "$CV_FX/values-owed/values-owed-none.md"
  [[ "$output" == *"PASS  commissioning  values-owed  "* ]]
  [[ "$output" != *"MANUAL  commissioning  values-owed"* ]]
}

@test "CV-owed4: no values-owed table -> MANUAL row asking to declare one (or state none)" {
  run "$CV" "$MINIMAL"
  [[ "$output" == *"MANUAL  commissioning  values-owed  no docs/values-owed.md"* ]]
}

@test "CV-owed5: --values-owed naming a missing file -> exit 3" {
  run "$CV" "$MINIMAL" --values-owed "$BATS_TEST_TMPDIR/absent.md"
  [ "$status" -eq 3 ]
}

# polish-2026-10-02 P3 (#199 WU9): the values-owed parser fails closed. A table row with fewer than the
# five cells (Slot | Owed by | Unit | Safe default | Status) is a MANUAL "malformed" row, never silently
# skipped into a PASS; a markdown alignment row (`:---`, `:---:`, `---:`) is not an owed slot.
# Named mutations: CV-owed-short (skip short rows again) -> CV-owed-short flips; CV-owed-align (treat
# only plain dashes as the separator) -> CV-owed-align flips.
# shellcheck disable=SC2016  # literal markdown backticks in the table, not expansions
@test "CV-owed-short: a row missing its Status cell -> MANUAL malformed row, no PASS" {
  local f="$BATS_TEST_TMPDIR/owed.md"
  printf '| Slot | Owed by | Unit | Safe default | Status |\n|---|---|---|---|---|\n| `stageDelay` | engineer | s | 30 | filled |\n| `lowPressureCutout` | technician | psig | 0 |\n' > "$f"
  run "$CV" "$MINIMAL" --values-owed "$f"
  [[ "$output" == *"MANUAL  commissioning  values-owed  malformed values-owed row (needs Slot | Owed by | Unit | Safe default | Status): | \`lowPressureCutout\` | technician | psig | 0 |"* ]]
  [[ "$output" != *"PASS  commissioning  values-owed"* ]]
}

# shellcheck disable=SC2016  # literal markdown backticks in the table, not expansions
@test "CV-owed-align: alignment rows (:---, :---:, ---:) are not owed slots -> PASS when every value is filled" {
  local f="$BATS_TEST_TMPDIR/owed.md"
  printf '| Slot | Owed by | Unit | Safe default | Status |\n|:---|:---:|---:|:--|---|\n| `stageDelay` | engineer | s | 30 | filled |\n' > "$f"
  run "$CV" "$MINIMAL" --values-owed "$f"
  [[ "$output" == *"PASS  commissioning  values-owed  "* ]]
  [[ "$output" != *"MANUAL  commissioning  values-owed"* ]]
}

# polish-2026-10-02 P3b (#199, P3 review regression): only the table whose header starts with Slot is the
# values-owed table, so a second, unrelated table in the doc adds no MANUAL row; a doc with pipe rows but no
# Slot header fails closed (MANUAL). Named mutation CV-owed-scope (parse every pipe row again) ->
# CV-owed-scope flips.
# shellcheck disable=SC2016  # literal markdown backticks in the table, not expansions
@test "CV-owed-scope: a second unrelated table is ignored; no Slot-header table -> MANUAL, no PASS" {
  local f="$BATS_TEST_TMPDIR/owed.md"
  printf '| Slot | Owed by | Unit | Safe default | Status |\n|---|---|---|---|---|\n| `stageDelay` | engineer | s | 30 | filled |\n\n## Revisions\n\n| Date | Note |\n|---|---|\n| 2026-10-01 | first draft |\n' > "$f"
  run "$CV" "$MINIMAL" --values-owed "$f"
  [[ "$output" == *"PASS  commissioning  values-owed  "* ]]
  [[ "$output" != *"MANUAL  commissioning  values-owed"* ]] || { echo "$output" | grep values-owed; return 1; }
  printf '| Date | Note |\n|---|---|\n| 2026-10-01 | first draft |\n' > "$f"
  run "$CV" "$MINIMAL" --values-owed "$f"
  [[ "$output" == *"MANUAL  commissioning  values-owed  no values-owed table (header Slot | Owed by | Unit | Safe default | Status) in $f"* ]]
  [[ "$output" != *"PASS  commissioning  values-owed"* ]]
}

# polish-2026-10-02 P3c (#199, P3b review fail-open): a pipe block is a foreign table only when its second
# row is an alignment row; an owed table split by a blank line or a comment keeps reporting the rows after
# the break (a one-row or a multi-row headerless block), never a silent PASS.
# Named mutation CV-owed-split (every block after a break is a foreign table) -> CV-owed-split flips.
# shellcheck disable=SC2016  # literal markdown backticks in the table, not expansions
@test "CV-owed-split: owed rows after a blank line / comment inside the Slot table stay MANUAL rows" {
  local f="$BATS_TEST_TMPDIR/owed.md"
  printf '| Slot | Owed by | Unit | Safe default | Status |\n|---|---|---|---|---|\n| `stageDelay` | engineer | s | 30 | filled |\n\n| `lowPressureCutout` | technician | psig | 0 | owed |\n<!-- more -->\n| `overCurrentLimit` | technician | A | 0 | |\n| `highLimit` | technician | psig | 400 | owed |\n' > "$f"
  run "$CV" "$MINIMAL" --values-owed "$f"
  [[ "$output" == *"MANUAL  commissioning  values-owed  lowPressureCutout: owed by technician, unit psig, still at safe default 0"* ]] || { echo "$output" | grep values-owed; return 1; }
  [[ "$output" == *"MANUAL  commissioning  values-owed  overCurrentLimit: owed by technician, unit A, still at safe default 0"* ]]
  [[ "$output" == *"MANUAL  commissioning  values-owed  highLimit: owed by technician, unit psig, still at safe default 400"* ]]
  [[ "$output" != *"PASS  commissioning  values-owed"* ]]
}

@test "CV-manual2: the MANUAL footer carries the persisted-state, alarm-routing, consumer-impact and link-source rows" {
  run "$CV" "$MINIMAL"
  [[ "$output" == *"MANUAL  commissioning  persisted-state-restart"* ]]
  [[ "$output" == *"MANUAL  commissioning  alarm-routing"* ]]
  [[ "$output" == *"MANUAL  commissioning  consumer-impact"* ]]
  [[ "$output" == *"MANUAL  commissioning  link-source-audit"*"obix-link-audit.sh"* ]]
}

# ---------------------------------------------------------------------------
# polish-2026-10-02 P1 — link-target-flags per -rt/src, with the wiring map (WU9 gap).
# [ev: retro panccadia-commissioning-lessons Δ1]
# Named mutation CV-map: drop the --wiring-map pass-through -> CV-ltf2 flips.
# ---------------------------------------------------------------------------
_cv_map_tree() {  # $1 = module root; one Map-rt artifact holding the Table 2 READONLY mirror
  mkdir -p "$1/Map/Map-rt/src/com/x"
  cp "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/src/com/x/BRoomPanel.java" "$1/Map/Map-rt/src/com/x/"
}

@test "CV-ltf1: MinimalPan -> one PASS link-target-flags row per -rt/src" {
  run "$CV" "$MINIMAL"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  commissioning  link-target-flags:MinimalPan-rt"* ]]
}

@test "CV-ltf2: <module-root>/docs/wiring-map.md is passed as --wiring-map -> Table 2 READONLY FAILs" {
  local r="$BATS_TEST_TMPDIR/cvltf2"; _cv_map_tree "$r"
  run "$CV" "$r"
  [[ "$output" == *"PASS  commissioning  link-target-flags:Map-rt"* ]]   # no map: not a known target
  mkdir -p "$r/docs"; cp "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md" "$r/docs/"
  run "$CV" "$r"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  commissioning  link-target-flags:Map-rt"*"wiring-map Table 2"* ]]
}

@test "CV-ltf3: --wiring-map <file> explicit is forwarded; a missing file -> exit 3" {
  local r="$BATS_TEST_TMPDIR/cvltf3"; _cv_map_tree "$r"
  run "$CV" "$r" --wiring-map "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"link-target-flags:Map-rt"*"wiring-map Table 2"* ]]
  run "$CV" "$r" --wiring-map "$BATS_TEST_TMPDIR/nope.md"
  [ "$status" -eq 3 ]
}

# ---------------------------------------------------------------------------
# polish-2026-10-02 P1b — P1 review advisories (#199, lineage review-6d90aed0e4c74ba3).
#   CV-ltf4  fail-closed: lint exit non-zero (not 3) with no FAIL row -> FAIL row naming the exit.
#   CV-ltf5  lint exit 3 -> SKIP row (this script's env convention for every source lint), exit 0.
#   CV-ltf6  the chosen wiring map is named in the PASS detail and the Table 2 FAIL reason.
# Named mutations (observed): CV-failopen -> CV-ltf4 flips; CV-mapname -> CV-ltf6 flips.
# ---------------------------------------------------------------------------
@test "CV-ltf4: lint exit 127 with no FAIL row -> FAIL link-target-flags row, never PASS, exit 1" {
  local tb="$BATS_TEST_TMPDIR/cvltf4tb"; stub_toolbelt "$tb" lint-link-target-flags.sh 'exit 127'
  run "$tb/commissioning-verify.sh" "$MINIMAL"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  commissioning  link-target-flags:MinimalPan-rt  lint-link-target-flags.sh exited 127"* ]]
  [[ "$output" != *"PASS  commissioning  link-target-flags"* ]]
}

@test "CV-ltf5: lint exit 3 -> SKIP link-target-flags row, exit 0" {
  local tb="$BATS_TEST_TMPDIR/cvltf5tb"; stub_toolbelt "$tb" lint-link-target-flags.sh 'exit 3'
  run "$tb/commissioning-verify.sh" "$MINIMAL"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP  commissioning  link-target-flags:MinimalPan-rt  env fault (exit 3)"* ]]
}

@test "CV-ltf6: the chosen wiring map is named in the PASS detail and the Table 2 FAIL reason" {
  local r="$BATS_TEST_TMPDIR/cvltf6"; _cv_map_tree "$r"
  run "$CV" "$r"
  [[ "$output" == *"PASS  commissioning  link-target-flags:Map-rt  LTF1/LTF2 clean (no wiring map)"* ]]
  run "$CV" "$r" --wiring-map "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md"
  [[ "$output" == *"(wiring-map Table 2: $BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md)"* ]]
  mkdir -p "$r/docs"; cp "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md" "$r/docs/"
  run "$CV" "$r"
  [ "$status" -eq 1 ]
  [[ "$output" == *"(wiring-map Table 2: $r/docs/wiring-map.md)"* ]]
  run "$CV" "$MINIMAL" --wiring-map "$r/docs/wiring-map.md"
  [[ "$output" == *"PASS  commissioning  link-target-flags:MinimalPan-rt  LTF1/LTF2 clean (wiring map: $r/docs/wiring-map.md)"* ]]
}

# polish-2026-10-02 P1c: a map path holding & is named verbatim (bash 5.2 patsub_replacement).
# Named mutation CV-amp (unquote the replacement) -> CV-ltf7 flips.
@test "CV-ltf7: a wiring-map path holding & is named verbatim in the Table 2 FAIL reason" {
  local r="$BATS_TEST_TMPDIR/c&v"; _cv_map_tree "$r"
  mkdir -p "$r/docs"; cp "$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags/map/docs/wiring-map.md" "$r/docs/"
  run "$CV" "$r"
  [ "$status" -eq 1 ]
  [[ "$output" == *"(wiring-map Table 2: $r/docs/wiring-map.md)"* ]]
}
