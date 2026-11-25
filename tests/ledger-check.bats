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
  sed -e 's/^| <Module> | <x.y.z> | <4.x.y.z> | <YYYY-MM-DD> | built | <one line> |/| Pan | 1.0.0 | 4.14.0.162 | 2026-10-01 | installed | first |/' \
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

@test "LCB5: the optional Niagara column: a version passes, a malformed one FAILs LC10, a ledger without it passes" {
  # Mutation: LCB5 -- accepting any text in the column leaves a typo that preflight ledger-niagara can never compare
  sed -e 's/^| <Module> | <x.y.z> | <4.x.y.z> | <YYYY-MM-DD> | built | <one line> |/| Pan | 1.0.0 | 4.14.0.162 | 2026-10-01 | built | first |/' \
    "$KIT/templates/VERSION-LEDGER.md" > "$BATS_TEST_TMPDIR/L.md"
  run "$LC" "$BATS_TEST_TMPDIR/L.md"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  sed -i 's/| 4.14.0.162 |/| 4.14 |/' "$BATS_TEST_TMPDIR/L.md"
  run "$LC" "$BATS_TEST_TMPDIR/L.md"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  ledger-check  $BATS_TEST_TMPDIR/L.md:"*"  LC10: niagara version 4.14 is not X.Y.Z[.W] (the niagara_home version the build used; — when unknown)" ]]
  # a ledger of the old shape (no Niagara column anywhere) is still clean
  sed -e 's/ | Niagara | / | /' -e 's/^|---|---|---|---|---|---|---|---|---|---|---|$/|---|---|---|---|---|---|---|---|---|---|/' \
    -e 's/^| Pan | 1.0.0 | 4.14 |/| Pan | 1.0.0 |/' "$BATS_TEST_TMPDIR/L.md" > "$BATS_TEST_TMPDIR/old.md"
  run "$LC" "$BATS_TEST_TMPDIR/old.md"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
