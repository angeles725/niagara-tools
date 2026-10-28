#!/usr/bin/env bats
# ledger-check.bats — toolbelt/ledger-check.sh (go/internal/ledgercheck, whose Go tests pin each LC check) end to
# end: the wrapper finds the kit's template, a fresh scaffold of every type carries a client ledger that passes,
# and a ledger row with a free-text status FAILs. (the 2026-09-26 version-defect-ledger retro, Δ1)

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LC="$KIT/toolbelt/ledger-check.sh"
  SC="$KIT/toolbelt/scaffold-module.sh"
}

@test "LCB1: a row whose status is not in the template's vocabulary FAILs through the wrapper" {
  # Mutation: LCB1 -- dropping the status check passes a row whose status is free text
  sed -e 's/^| <Module> | <x.y.z> | <YYYY-MM-DD> | built | <one line> |/| Pan | 1.0.0 | 2026-10-01 | installed | first |/' \
    "$KIT/templates/VERSION-LEDGER.md" > "$BATS_TEST_TMPDIR/L.md"
  run "$LC" "$BATS_TEST_TMPDIR/L.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  ledger-check  $BATS_TEST_TMPDIR/L.md:"*"  LC5: status installed is not one of built, packaged, deployed, superseded, do-not-install"* ]]
}

@test "LCB2: the kit template itself passes with only its sample row's placeholder WARN" {
  run "$LC" "$KIT/templates/VERSION-LEDGER.md"
  [ "$status" -eq 0 ]
  [ "$(grep -c 'LC9:' <<<"$output")" -eq 1 ]
  [ "$(grep -c . <<<"$output")" -eq 1 ]
}

@test "LCB3: every scaffold type emits odd/VERSION-LEDGER.md at the gradle root, and it is clean" {
  # Mutation: LCB3 -- a scaffold that skips the ledger leaves the client with nothing for BUILD-LOOP §0 to read
  for type in logic dashboard driver; do
    run "$SC" ModuleA "$BATS_TEST_TMPDIR/$type" --type "$type" --vendor Example
    [ "$status" -eq 0 ]
    L="$BATS_TEST_TMPDIR/$type/ModuleA/odd/VERSION-LEDGER.md"
    [ -f "$L" ] || { echo "$type: no $L"; false; }
    run "$LC" "$L"
    [ "$status" -eq 0 ] || { echo "$type: $output"; false; }
    [ -z "$output" ] || { echo "$type: $output"; false; }
  done
}

@test "LCB4: usage and a missing ledger are exit 3" {
  run "$LC"
  [ "$status" -eq 3 ]
  [[ "$output" == "usage: ledger-check.sh"* ]]
  run "$LC" "$BATS_TEST_TMPDIR/none.md"
  [ "$status" -eq 3 ]
  [[ "$output" == *"cannot read"* ]]
}
