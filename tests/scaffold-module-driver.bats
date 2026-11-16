#!/usr/bin/env bats
# scaffold-module --type driver: the emitted tree, its lint, its doctrine, its codec test and its build.
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

# ---------------------------------------------------------------------------------------------------------
# --type driver (round 10 DR2): an NDriver module, -rt + -wb, from the MinimalDriver fixture.
# Doctrine: types/driver-authoring.md § 1.6-§ 1.11, § 5, § 12. [ev: retro r10-dr2-driver-scaffold Δ1]
# ---------------------------------------------------------------------------------------------------------

@test "TC-DRV1: --type driver emits the MinimalDriver tree byte-equal to the bundled fixture (diff -r, minus build/.gradle)" {
  run "$SC" MinimalDriver "$OUT" --type driver
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle "$OUT/MinimalDriver" "$FIXTURE_DRV"
  [ "$status" -eq 0 ]
}

@test "TC-DRV2: a renamed driver has the rt + wb shape, every module-include type has its source, and no fixture name is left" {
  # Named mutation: drop the -wb tree from the fixture -> the wb gradle/include checks below fail.
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  M="$OUT/AcmeLink/AcmeLink"
  for f in AcmeLink-rt/AcmeLink-rt.gradle.kts AcmeLink-rt/module-include.xml AcmeLink-rt/module.palette \
           AcmeLink-rt/module.lexicon AcmeLink-rt/module-permissions.xml \
           AcmeLink-rt/src/com/example/AcmeLink/model/AcmeLinkCodec.java \
           AcmeLink-rt/srcTest/test/com/example/AcmeLink/model/AcmeLinkCodecTest.java \
           AcmeLink-wb/AcmeLink-wb.gradle.kts AcmeLink-wb/module-include.xml AcmeLink-wb/module.lexicon \
           AcmeLink-wb/module-permissions.xml; do
    [ -f "$M/$f" ] || { echo "missing: $f"; false; }
  done
  # every <type class=...> in each module-include.xml resolves to a source file of that profile
  for p in rt wb; do
    n=0
    while IFS= read -r cls; do
      n=$((n+1))
      [ -f "$M/AcmeLink-$p/src/${cls//.//}.java" ] || { echo "no source for $cls ($p)"; false; }
    done < <(grep -o 'class="[^"]*"' "$M/AcmeLink-$p/module-include.xml" | sed 's/class="//; s/"$//')
    [ "$n" -gt 0 ]
  done
  # the wb profile depends on the rt profile and on ndriver-wb; rt depends on ndriver-rt
  grep -q 'api(":ndriver-rt")' "$M/AcmeLink-rt/AcmeLink-rt.gradle.kts"
  grep -q 'api(":ndriver-wb")' "$M/AcmeLink-wb/AcmeLink-wb.gradle.kts"
  grep -q 'api(project(":AcmeLink-rt"))' "$M/AcmeLink-wb/AcmeLink-wb.gradle.kts"
  # the station-side socket permission is requested (types/driver-authoring.md § 5)
  grep -q '<name>NETWORK_COMMUNICATION</name>' "$M/AcmeLink-rt/module-permissions.xml"
  # no fixture name or fixture vendor survives the rename
  run grep -rIl -e MinimalDriver -e angeles -e Angeles "$OUT/AcmeLink"
  [ "$status" -eq 1 ]
}

@test "TC-DRV3: lint-driver-scaffold passes the emitted driver with no row at all (no FAIL, no WARN)" {
  # Named mutation: replace the device's doPing body with the wizard's lone pingOk(); -> FAIL row, exit 1.
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  for p in rt wb; do
    run "$KIT/toolbelt/lint-driver-scaffold.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-$p/src"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
  done
  # positive control: the same tree with the wizard ping stub FAILs, so the lint reached these sources
  dev="$OUT/AcmeLink/AcmeLink/AcmeLink-rt/src/com/example/AcmeLink/BAcmeLinkDevice.java"
  [ -f "$dev" ]
  printf '%s\n' 'package com.example.AcmeLink;' 'public class BStubDevice extends BNDevice {' \
    '  public void doPing() { pingOk(); }' '}' > "${dev%/*}/BStubDevice.java"
  run "$KIT/toolbelt/lint-driver-scaffold.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BStubDevice.java:3  doPing() only calls pingOk()"* ]]
}

@test "TC-DRV4: the emitted driver carries the DR1 doctrine, not wizard placeholders" {
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  S="$OUT/AcmeLink/AcmeLink/AcmeLink-rt/src/com/example/AcmeLink"
  # no placeholder marker anywhere in the generated module
  run grep -rIn -e 'TODO' -e 'FIXME' -e 'not implemented' "$OUT/AcmeLink/AcmeLink"
  [ "$status" -eq 1 ]
  # § 1.9: the comm config is a SUBCLASS with its own message and link-message factories; no bare base config slot
  grep -q 'extends BUdpCommConfig' "$S/BAcmeLinkUdpCommConfig.java"
  grep -q 'protected IMessageFactory makeMessageFactory()' "$S/BAcmeLinkUdpCommConfig.java"
  grep -q 'protected NLinkMessageFactory makeLinkMessageFactory()' "$S/BAcmeLinkUdpCommConfig.java"
  run grep -rEn 'type = "B(Udp|Tcp)CommConfig"' "$S"
  [ "$status" -eq 1 ]
  grep -q 'type = "BAcmeLinkUdpCommConfig"' "$S/BAcmeLinkNetwork.java"
  # § 1.10 / § 1.7: tag correlation on request and response; § 1.9: sender check in validateResponse
  grep -q 'public Object getTag()' "$S/AcmeLinkRequest.java"
  grep -q 'public Object getTag()' "$S/AcmeLinkResponse.java"
  grep -q 'sameDevice(' "$S/AcmeLinkRequest.java"
  grep -q 'FAILED_RESPONSE' "$S/AcmeLinkRequest.java"
  # § 1.6: discovery is submitted with the caller's Context, never null
  grep -q 'doSubmitDiscoveryJob(BNDiscoveryPreferences preferences, Context cx)' "$S/BAcmeLinkNetwork.java"
  grep -q 'job.submit(cx)' "$S/BAcmeLinkNetwork.java"
  run grep -rn 'submit(null)' "$S"
  [ "$status" -eq 1 ]
  # § 1.11: both discovery hosts stop when the job is no longer alive
  grep -q 'isAlive()' "$S/BAcmeLinkNetwork.java"
  grep -q 'isAlive()' "$S/BAcmeLinkPointDeviceExt.java"
  # § 5: a real ping (a request on the wire) that can fail, not the wizard's unconditional pingOk
  grep -q 'pingFail(' "$S/BAcmeLinkDevice.java"
  grep -q 'CMD_PING' "$S/BAcmeLinkDevice.java"
  # § 12: the codec is Baja-free, so its srcTest runs as pure JUnit
  run grep -En '^import (javax\.baja|com\.tridium)' "$S/model/AcmeLinkCodec.java"
  [ "$status" -eq 1 ]
}

@test "TC-DRV5: the emitted codec srcTest passes as pure JUnit (SKIP without javac or the cached junit jar)" {
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  run "$KIT/toolbelt/run-pure-test.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-rt" com.example.AcmeLink.model.AcmeLinkCodecTest
  [ "$status" -eq 0 ]
  [[ "$output" == *"OK ("* ]]
}

@test "TC-DRV6: the codec test bites: a codec that skips the checksum check makes the srcTest fail (exit 1)" {
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  C="$OUT/AcmeLink/AcmeLink/AcmeLink-rt/src/com/example/AcmeLink/model/AcmeLinkCodec.java"
  grep -q 'if (sum != buf\[off + CHECKSUM\]) throw' "$C"
  sed -i '/if (sum != buf\[off + CHECKSUM\]) throw/d' "$C"
  run grep -c 'IllegalArgumentException("bad checksum")' "$C"
  [ "$output" = "0" ]
  run "$KIT/toolbelt/run-pure-test.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-rt" com.example.AcmeLink.model.AcmeLinkCodecTest
  [ "$status" -eq 1 ]
}

@test "TC-DRV-BUILD: the emitted driver builds signed through build.sh into a mirror (SKIP without niagara_home + signing profile)" {
  [ -n "${NIAGARA_HOME:-}" ] || skip "no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)"
  [ -n "${KIT_SIGNING_PROFILE:-}" ] && [ -n "${KIT_SIGNING_ALIAS:-}" ] \
    || skip "set KIT_SIGNING_PROFILE + KIT_SIGNING_ALIAS: the root signing block refuses the default profile"
  run "$SC" MinimalDriver "$OUT" --type driver
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/mirror-niagara-home.sh" "$NIAGARA_HOME" "$BATS_TEST_TMPDIR/nh-mirror" MinimalDriver-rt.jar MinimalDriver-wb.jar
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/build.sh" --no-report --gradle-prop "niagaraSigningProfile=$KIT_SIGNING_PROFILE" \
    --gradle-prop "niagaraSigningAlias=$KIT_SIGNING_ALIAS" "$OUT/MinimalDriver" MinimalDriver "$BATS_TEST_TMPDIR/nh-mirror"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ALL PASS"* ]]
}
