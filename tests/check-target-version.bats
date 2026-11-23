#!/usr/bin/env bats
# check-target-version.sh — a module jar goes into the Niagara installation it was compiled for: the highest
# Tridium dependency floor of the jar's module.xml against the vendorVersion of the installation's modules/baja.jar,
# major and minor equal. The logic is pinned by go/internal/niagaraversion; this file runs the launcher end to end.
# [ev: retro deploy-target-version Δ1]

setup() {
  CHK="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/check-target-version.sh"
  W="$BATS_TEST_TMPDIR"
}

# jar_with <jar> <module.xml text>: a jar whose only entry is META-INF/module.xml
jar_with() {
  mkdir -p "$W/zip/META-INF"
  printf '%s\n' "$2" > "$W/zip/META-INF/module.xml"
  (cd "$W/zip" && zip -q -r "$1" META-INF)
}

# install_of <version>: an installation directory whose baja.jar carries that vendorVersion
install_of() {
  jar_with "$W/baja.jar.tmp" "<module name=\"baja\" vendor=\"Tridium\" vendorVersion=\"$1\"/>"
  mkdir -p "$W/Niagara-$1/modules"
  mv "$W/baja.jar.tmp" "$W/Niagara-$1/modules/baja.jar"
}

@test "CTV-pass: a jar compiled for 4.14 passes against a 4.14.0.162 installation, by root or modules directory" {
  jar_with "$W/Demo-rt.jar" '<module name="Demo-rt" vendorVersion="2.4.0"><dependencies><dependency name="baja" vendor="Tridium" vendorVersion="4.14"/></dependencies></module>'
  install_of 4.14.0.162
  run "$CHK" "$W/Demo-rt.jar" "$W/Niagara-4.14.0.162"
  [ "$status" -eq 0 ]
  [[ "$output" == "PASS  target-version  "*"was compiled against Niagara 4.14;"*"is Niagara 4.14.0.162" ]]
  run "$CHK" "$W/Demo-rt.jar" "$W/Niagara-4.14.0.162/modules"
  [ "$status" -eq 0 ]
}

@test "CTV-fail: the same jar against a 4.15.6.20 installation is exit 1 and names both versions" {
  jar_with "$W/Demo-rt.jar" '<module name="Demo-rt" vendorVersion="2.4.0"><dependencies><dependency name="baja" vendor="Tridium" vendorVersion="4.14"/></dependencies></module>'
  install_of 4.15.6.20
  run "$CHK" "$W/Demo-rt.jar" "$W/Niagara-4.15.6.20"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  target-version  "*"compiled against Niagara 4.14 but "*"is Niagara 4.15.6.20"* ]]
}

@test "CTV-usage: a missing argument, and an installation with no baja.jar, are exit 2 with no row" {
  run "$CHK"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: check-target-version"* ]]
  jar_with "$W/Demo-rt.jar" '<module name="Demo-rt"><dependencies><dependency name="baja" vendor="Tridium" vendorVersion="4.14"/></dependencies></module>'
  mkdir -p "$W/empty"
  run "$CHK" "$W/Demo-rt.jar" "$W/empty"
  [ "$status" -eq 2 ]
  [[ "$output" != *"PASS"* && "$output" != *"FAIL"* ]]
}
