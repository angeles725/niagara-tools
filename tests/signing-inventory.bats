#!/usr/bin/env bats
# Pins for toolbelt/signing-inventory.sh (issue #1160): the read-only inventory of signing profiles, keystores
# and aliases, and with --module what the module records about its signer.
#
# Rows:  PASS|WARN|FAIL|INFO  <check id>  <detail>
# Exit:  0 (WARN rows allowed) · 1 a FAIL row · 2 usage or an unreadable input · 3 no Go toolchain
#
# The stores are built at test time with the keytool of $JAVA_HOME, else the one on PATH (the CI bats job installs
# Java 8, so a missing keytool is a broken setup and fails here, it does not skip). Every run names its profile
# with --profile: the user's own ~/.tridium is never read.

bats_require_minimum_version 1.5.0

CANARY="Pw-Canary-7Zq3"

setup_file() {
  KT="${JAVA_HOME:+$JAVA_HOME/bin/}keytool"
  command -v "$KT" >/dev/null 2>&1 || [ -x "$KT" ] || KT=keytool
  STORE="$BATS_FILE_TMPDIR/lab.jceks"
  "$KT" -genkeypair -alias good -keyalg RSA -keysize 2048 -validity 365 -dname "CN=good signer" -ext eku=codeSigning \
    -keystore "$STORE" -storetype JCEKS -storepass "$CANARY" -keypass "$CANARY" >/dev/null 2>&1
  export STORE
}

setup() {
  SI="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/signing-inventory.sh"
  [ -f "$STORE" ] || { echo "keytool could not build the fixture store (is a JDK's keytool on PATH or in JAVA_HOME?)" >&2; return 1; }
  PROFILE="$BATS_TEST_TMPDIR/niagara.signing.xml"
  _profile "$CANARY" >"$PROFILE"
  MOD="$BATS_TEST_TMPDIR/Mod"
  mkdir -p "$MOD"
}

# _profile <store password> prints a signing profile over the fixture store ("" leaves the password out)
_profile() {
  printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">\n<properties>\n'
  printf '<entry key="niagara.signing.storepath">%s</entry>\n<entry key="niagara.signing.storetype">JCEKS</entry>\n' "$STORE"
  [ -z "$1" ] || printf '<entry key="niagara.signing.storepass">%s</entry>\n' "$1"
  printf '<entry key="niagara.signing.keypass.good">%s</entry>\n</properties>\n' "$CANARY"
}

# _jar <signer> writes <MOD>/Mod-rt/build/libs/Mod-rt.jar signed by <signer> (META-INF/<signer>.SF + .RSA), compiled for 4.14
_jar() {
  mkdir -p "$MOD/Mod-rt/build/libs"
  python3 -I -c '
import sys, zipfile
z = zipfile.ZipFile(sys.argv[1], "w")
z.writestr("META-INF/module.xml", "<module name=\"Mod-rt\"><dependencies><dependency name=\"baja\" vendor=\"Tridium\" vendorVersion=\"4.14\"/></dependencies></module>")
for ext in ("SF", "RSA"):
    z.writestr("META-INF/%s.%s" % (sys.argv[2], ext), "x")
z.close()' "$MOD/Mod-rt/build/libs/Mod-rt.jar" "$1"
}

# _state <alias> writes the module's build-state.v1 envelope recording expected_signer
_state() {
  printf '<!-- build-state.v1 -->\nmodule: Mod\nexpected_signer: %s\n<!-- /build-state.v1 -->\n' "$1" >"$BATS_TEST_TMPDIR/state.md"
}

@test "SI1: a bad option, a stray argument or --module-name without --module is usage exit 2 on stderr" {
  for args in "--nope" "extra" "--module-name X" "--profile"; do
    # shellcheck disable=SC2086  # the words of one case are the arguments
    run --separate-stderr "$SI" $args
    [ "$status" -eq 2 ]
    [ -z "$output" ]
    [[ "$stderr" == *"usage: signing-inventory"* || "$stderr" == *"signing-inventory:"* ]]
  done
}

@test "SI2: a missing --profile file is exit 2 naming it" {
  run --separate-stderr "$SI" --profile "$BATS_TEST_TMPDIR/none.xml"
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"none.xml"* ]]
}

@test "SI3: a clean run lists the alias with subject, issuer, days left, code-signing and the file; exit 0" {
  run --separate-stderr "$SI" --profile "$PROFILE"
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  [[ "$output" == *"INFO  profile  $PROFILE"* ]]
  [[ "$output" == *"INFO  alias  good  signing key in $STORE"*"subject good signer"*"issuer good signer"* ]]
  [[ "$output" == *"PASS  validity  good: valid until "*" days left)"* ]]
  [[ "$output" == *"PASS  usage  good: code signing"* ]]
  [[ "$output" != *"FAIL"* ]]
}

@test "SI4: --module prints the recorded signer, the jar's signer and compiled-for version; PASS when they agree" {
  _jar GOOD
  _state good
  run "$SI" --profile "$PROFILE" --module "$MOD" --build-state "$BATS_TEST_TMPDIR/state.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"INFO  expected-signer  Mod: good (recorded in "* ]]
  [[ "$output" == *"PASS  expected-signer-known  good is in $STORE and in date"* ]]
  [[ "$output" == *"INFO  jar  Mod-rt/build/libs/Mod-rt.jar  signed by GOOD; compiled for Niagara 4.14"* ]]
}

@test "SI5: --module FAILs (exit 1) when the recorded alias is in no listed store" {
  _jar GOOD
  _state ghost
  run "$SI" --profile "$PROFILE" --module "$MOD" --build-state "$BATS_TEST_TMPDIR/state.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  expected-signer-known  ghost is in none of the 1 store(s) listed"* ]]
}

@test "SI6: --module FAILs (exit 1) when a built jar is signed by another signer than the recorded one" {
  _jar OTHER
  _state good
  run "$SI" --profile "$PROFILE" --module "$MOD" --build-state "$BATS_TEST_TMPDIR/state.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  jar-signer  Mod-rt/build/libs/Mod-rt.jar: signed by OTHER, expected good (META-INF/GOOD.SF)"* ]]
}

@test "SI7: a store the profile holds no password for is one locked WARN row with its path only; exit 0" {
  _profile "" >"$PROFILE"
  run --separate-stderr "$SI" --profile "$PROFILE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  store-locked  $STORE: "* ]]
  [[ "$output" != *"alias"* ]]
}

@test "SI8: no password, key or keystore message reaches stdout or stderr, on success and on every error path" {
  _jar GOOD
  _state good
  _profile wrong-Canary-1 >"$BATS_TEST_TMPDIR/wrong.xml"
  printf '<properties><entry key="niagara.signing.storepass">%s</nope></entry>' "$CANARY" >"$BATS_TEST_TMPDIR/broken.xml"
  for args in "--profile $PROFILE --module $MOD --build-state $BATS_TEST_TMPDIR/state.md" "--profile $BATS_TEST_TMPDIR/wrong.xml" \
    "--profile $BATS_TEST_TMPDIR/broken.xml" "--profile $PROFILE --nope" "--profile $PROFILE --module $MOD --build-state /none"; do
    # shellcheck disable=SC2086  # the words of one case are the arguments (no path here holds a space)
    run --separate-stderr "$SI" $args
    [[ "$output$stderr" != *"$CANARY"* ]]
    [[ "$output$stderr" != *"wrong-Canary-1"* ]]
    [[ "$output$stderr" != *"PrivateKeyEntry"* ]]
  done
}
