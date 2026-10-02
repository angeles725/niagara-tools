#!/usr/bin/env bats
# Pins for obix-link-audit.sh — declared wiring map vs the links actually present on a live
# component (read-only oBIX). A link from the WRONG-but-valid source is structurally fine, so
# bog-audit cannot see it; only a diff against the declared source closes that gap.
# Rows: MATCH | MISMATCH | MISSING | SKIP  obix-link-audit  <slot>  <detail>
# Exit: 0 no MISMATCH/MISSING · 1 any · 3 usage/env.
# [ev: retro panccadia-commissioning-lessons Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  OLA="$KIT/toolbelt/obix-link-audit.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/obix-link-audit"
}

@test "OLA1: Table 2 vs a saved component dump -> MATCH / MISMATCH (actual source named) / MISSING / SKIP, exit 1" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State  declared comp1Running, linked from relayOut1 (slot:/Plant/IO)"* ]]
  [[ "$output" == *"MATCH  obix-link-audit  comp2State  "* ]]
  [[ "$output" == *"MATCH  obix-link-audit  comp3State  "* ]]
  [[ "$output" == *"MISSING  obix-link-audit  inDrip  declared dripActive, no link into inDrip"* ]]
  [[ "$output" == *"SKIP  obix-link-audit  freezeActive  "* ]]
  [[ "$output" == *"obix-link-audit: 2 MATCH · 1 MISMATCH · 1 MISSING · 1 SKIP  ->  ISSUES"* ]]
}

@test "OLA2: every declared link-in present from its declared source (arrow forms + Direct) -> exit 0, CLEAN" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade-all-match.xml"
  [ "$status" -eq 0 ]
  [[ "$output" == *"4 MATCH · 0 MISMATCH · 0 MISSING · 1 SKIP  ->  CLEAN"* ]]
}

@test "OLA3: --table 1 audits facade->control rows against the control component" {
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/control.xml" --table 1
  [ "$status" -eq 1 ]
  [[ "$output" == *"MATCH  obix-link-audit  suctionSetpoint  "* ]]
  [[ "$output" == *"MISSING  obix-link-audit  suctionDiffUp  declared diffUp, no link into suctionDiffUp"* ]]
  [[ "$output" == *"SKIP  obix-link-audit  diffDown  "* ]]
}

@test "OLA4: usage faults -> exit 3 (no args, missing map, both --xml and --obix, bad --table)" {
  run "$OLA"
  [ "$status" -eq 3 ]
  run "$OLA" --map "$BATS_TEST_TMPDIR/absent.md" --xml "$FX/facade.xml"
  [ "$status" -eq 3 ]
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml" --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 3 ]
  run "$OLA" --map "$FX/wiring-map.md" --xml "$FX/facade.xml" --table 3
  [ "$status" -eq 3 ]
}

@test "OLA5: live mode -> ONE read-only GET of <base>/config/<component>/, credentials via stdin config (never argv)" {
  bin="$BATS_TEST_TMPDIR/bin"; mkdir -p "$bin"
  cat > "$bin/curl" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$@" > "$BATS_TEST_TMPDIR/curl.args"
cat > "$BATS_TEST_TMPDIR/curl.stdin"
out=""
while [ \$# -gt 0 ]; do [ "\$1" = "-o" ] && out="\$2"; shift; done
cp "$FX/facade.xml" "\$out"
STUB
  chmod +x "$bin/curl"
  OBIX_USER=reader OBIX_PASS=secret PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISMATCH  obix-link-audit  comp1State"* ]]
  grep -qx 'https://station.example/obix/config/Plant/Facade/' "$BATS_TEST_TMPDIR/curl.args"
  [ "$(grep -cE '^(-X|--request|-d|--data.*|-T|--upload-file)$' "$BATS_TEST_TMPDIR/curl.args")" -eq 0 ]
  [ "$(grep -c 'secret' "$BATS_TEST_TMPDIR/curl.args")" -eq 0 ]
  grep -q 'reader:secret' "$BATS_TEST_TMPDIR/curl.stdin"
}

@test "OLA6: live mode without OBIX_USER/OBIX_PASS -> exit 3, no request made" {
  bin="$BATS_TEST_TMPDIR/bin"; mkdir -p "$bin"
  printf '#!/usr/bin/env bash\ntouch "%s/called"\n' "$BATS_TEST_TMPDIR" > "$bin/curl"; chmod +x "$bin/curl"
  OBIX_USER='' OBIX_PASS='' PATH="$bin:$PATH" run "$OLA" --map "$FX/wiring-map.md" \
    --obix https://station.example/obix --component /Plant/Facade
  [ "$status" -eq 3 ]
  [ ! -e "$BATS_TEST_TMPDIR/called" ]
}

@test "OLA7: a component with NO links -> every declared row MISSING (empty link set never reads the map as links)" {
  printf '<obj href="https://station.example/obix/config/Plant/Facade/"/>\n' > "$BATS_TEST_TMPDIR/empty.xml"
  run "$OLA" --map "$FX/wiring-map.md" --xml "$BATS_TEST_TMPDIR/empty.xml"
  [ "$status" -eq 1 ]
  [[ "$output" == *"0 MATCH · 0 MISMATCH · 4 MISSING · 1 SKIP  ->  ISSUES"* ]]
}
