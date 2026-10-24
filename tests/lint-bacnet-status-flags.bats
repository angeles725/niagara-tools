#!/usr/bin/env bats
# lint-bacnet-status-flags.bats — G36: BACnet Present_Value proxy points whose facets lack statusFlags=true read
# {ok} through a field alarm, fault or override; one WARN row per device. The rule's golden cases live in
# go/internal/lint/rules/bacnetstatusflags. [ev: retro devguide415-r2-drivers-protocols Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-bacnet-status-flags.sh"
  ST="$BATS_TEST_TMPDIR/acmeStation"
  cp -r "$BATS_TEST_DIRNAME/fixtures/driver-lints/acmeStation" "$ST"
  BOG="$ST/config.bog"
}

ROW='/Drivers/BacnetNetwork/AcmeAhu1: 1 Present_Value proxy point(s) without facets statusFlags=true (first: /Drivers/BacnetNetwork/AcmeAhu1/points/SupplyTemp) -- a field alarm, fault or override reads {ok}'

@test "BSF-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-bacnet-status-flags.sh [--strict] <station-dir>..." ]
}

@test "BSF1: the fixture station WARNs its one Present_Value point without the facet, not the one with it" {
  # Mutation: BSF1 -- ignoring the statusFlags facet makes the count 2
  run "$L" "$ST"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-bacnet-status-flags  $BOG: $ROW" ]
  run "$L" --strict "$ST"
  [ "$status" -eq 1 ]
}

@test "BSF2: the bog's module alias is resolved; a zipped config.bog is read like a plain one" {
  # Mutation: BSF2 -- matching the literal bacnet: prefix turns this case green (exit 0, no row)
  (cd "$ST" && mkdir z && cp config.bog z/file.xml && cd z && zip -q ../saved.zip file.xml) && rm -r "$ST/z" "$BOG"
  mv "$ST/saved.zip" "$BOG"
  run "$L" "$ST"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-bacnet-status-flags  $BOG: $ROW" ]
}

@test "BSF3: a non-Present_Value proxy is clean; the facet set on every Present_Value point is clean" {
  # Mutation: BSF3 -- treating every propertyId as Present_Value makes the count 2
  sed -i 's#v="units=u:celsius;&\#xb0;C;(K);+273.15;"#v="statusFlags=b:true"#' "$BOG"
  run "$L" --strict "$ST"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "BSF-K20: a root with no *.bog is the ERROR row; a malformed bog exits 3" {
  rm "$BOG"
  run "$L" "$ST"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-bacnet-status-flags  $ST  no *.bog under the root (wrong path: pass a station directory)" ]
  printf '<bajaObjectGraph><p></bajaObjectGraph>\n' > "$BOG"
  run "$L" "$ST"
  [ "$status" -eq 3 ]
  [[ "$output" == "lint-bacnet-status-flags: $BOG:1:"* ]]
}
