#!/usr/bin/env bats
# report-module.sh --ready (retro row P4): the ready-to-load report of a built jar, through the toolbelt launcher.
# Synthetic jars only (zip), a stub verify-module and schema-risk, no Niagara home, no station.
# Go unit tests: go/internal/reportmodule/ready*_test.go.

load helpers/stub-toolbelt

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  export KIT  # read by stub_toolbelt
  if ! command -v zip >/dev/null 2>&1; then skip "zip not available"; fi
  W="$BATS_TEST_TMPDIR/m"; A="$W/Mod-rt"
  mkdir -p "$A/build/libs" "$A/.deploy-baseline" "$BATS_TEST_TMPDIR/j/META-INF"
  printf '<module name="Mod-rt" vendor="V" vendorVersion="2.8.6" moduleName="Mod" runtimeProfile="rt">\n <dependency name="baja" vendor="Tridium" vendorVersion="4.14"/>\n</module>\n' \
    > "$BATS_TEST_TMPDIR/j/META-INF/module.xml"
  printf 'sf\n' > "$BATS_TEST_TMPDIR/j/META-INF/ACMESIGN.SF"
  printf 'rsa\n' > "$BATS_TEST_TMPDIR/j/META-INF/ACMESIGN.RSA"
  (cd "$BATS_TEST_TMPDIR/j" && zip -q -r "$A/build/libs/Mod-rt.jar" META-INF)
  stub_toolbelt "$BATS_TEST_TMPDIR/tb" verify-module.sh 'printf "PASS  bytecode  x  ok\n"; exit 0'
  rm "$BATS_TEST_TMPDIR/tb/schema-risk.sh"
  printf '#!/usr/bin/env bash\nexit 0\n' > "$BATS_TEST_TMPDIR/tb/schema-risk.sh"
  chmod +x "$BATS_TEST_TMPDIR/tb/schema-risk.sh"
}

@test "RMR1: --ready with the expected signer and target prints READY TO LOAD and the read-back, exit 0" {
  # Mutation: RMR1 -- a READY line printed while a row FAILs, or no read-back after it, fails this pin.
  run "$BATS_TEST_TMPDIR/tb/report-module.sh" "$W" --ready --expect-niagara 4.15 --expect-signer acmesigner
  [ "$status" -eq 0 ]
  [[ "$output" == *"Mod-rt  PASS  signer  ACMESIGN (expected acmesigner)"* ]]
  [[ "$output" == *"READY TO LOAD: Mod-rt 2.8.6 for Niagara 4.15 (sha256 "* ]]
  [[ "$output" == *"n4_station_runtime"* ]]
}

@test "RMR2: --ready with another expected signer is NOT READY, exit 1" {
  run "$BATS_TEST_TMPDIR/tb/report-module.sh" "$W" --ready --expect-signer othersigner
  [ "$status" -eq 1 ]
  [[ "$output" == *"NOT READY: Mod-rt 2.8.6"* ]]
  if [[ "$output" == *"READY TO LOAD"* ]]; then return 1; fi
}

@test "RMR3: --ready --json is one JSON object with the verdict" {
  run "$BATS_TEST_TMPDIR/tb/report-module.sh" "$W" --ready --json --expect-niagara 4.15 --expect-signer acmesigner
  [ "$status" -eq 0 ]
  [[ "$output" == *'"verdict": "READY"'* ]]
  [[ "$output" == *'"check": "sha256"'* ]]
}

@test "RMR4: --json without --ready is a usage error (exit 2)" {
  run "$BATS_TEST_TMPDIR/tb/report-module.sh" "$W" --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"--ready"* ]]
}
