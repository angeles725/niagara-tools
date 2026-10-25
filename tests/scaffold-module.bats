#!/usr/bin/env bats
# RED-FIRST pins for scaffold-module.sh (campaign 7 PR4, issue #45, from B794 companero MinimalPan build).
# Emits the SMALLEST correct N4 module source tree (B790/B793 recipe) so a new module starts gate-green.
#
# SURFACE (provisional — the design owns the fixture location + exact flag set; rebase if it moves, kept in one file):
#   scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version 4.14] [--plugin-version 7.6.17]
#   exit: 0 ok · 2 usage OR invalid name (name must be a Java identifier starting uppercase) · 3 env
#         (out-dir not creatable, or already contains the module)
#
# RED today: build-n4-module-kit/toolbelt/scaffold-module.sh does NOT exist and the golden fixture
# build-n4-module-kit/fixtures/MinimalPan/ is not bundled yet, so TC1/TC2/TC3/TC-K8 fail for the right
# reason (tool absent). TC4 SKIPs without a niagara_home/JDK 8 — it is local-only round-trip bless
# evidence, never a fake PASS. Green once PR4 lands the emitter + the fixture.
#
# NAMED MUTATIONS (run post-green):
#   - drop the <MOD>-rt.gradle.kts emission        -> TC3 diff fails (tree differs from the fixture).
#   - emit module.xml instead of module-include.xml -> TC3 fails (B793 correction #3).
#   - drop the stopped() cancel in the emitted component -> TC4 lint-timers FAILs locally.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SC="$KIT/toolbelt/scaffold-module.sh"
  FIXTURE="$KIT/fixtures/MinimalPan"
  FIXTURE_DASH="$KIT/fixtures/MinimalDash"
  FIXTURE_DRV="$KIT/fixtures/MinimalDriver"
  # logic + dashboard also get the shared station-test overlay (templates/station-test), which the
  # fixture trees do not carry; TC-ST* pin it against tests/fixtures/scaffold-module/station-test.
  ST_GOLDEN="$BATS_TEST_DIRNAME/fixtures/scaffold-module/station-test"
  ST_EXCLUDE="--exclude=moduleTest-include.xml --exclude=B*StationTestBase.java --exclude=B*LinkTest.java"
  OUT="$BATS_TEST_TMPDIR/out"
  mkdir -p "$OUT"
}

@test "TC1: missing args exits 2 with usage on stderr" {
  run "$SC"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage"* ]]
}

@test "TC2: an invalid module name (1bad — not an uppercase Java identifier) exits 2" {
  run "$SC" 1bad "$OUT"
  [ "$status" -eq 2 ]
}

@test "TC3: the emitted MinimalPan tree byte-equals the bundled fixture (diff -r, minus build/.gradle)" {
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE "$OUT/MinimalPan" "$FIXTURE"
  [ "$status" -eq 0 ]
}

@test "TC-K8: the emitted tree is identical under HOME=/nonexistent (no \$HOME coupling)" {
  HOME=/nonexistent run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE "$OUT/MinimalPan" "$FIXTURE"
  [ "$status" -eq 0 ]
}

@test "TC-DASH1: --type dashboard emits MinimalDash tree byte-equals the bundled fixture (diff -r, minus build/.gradle)" {
  run "$SC" MinimalDash "$OUT" --type dashboard
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE "$OUT/MinimalDash" "$FIXTURE_DASH"
  [ "$status" -eq 0 ]
}

@test "TC-DASH2: invalid --type exits 2" {
  run "$SC" MyModule "$OUT" --type unknown
  [ "$status" -eq 2 ]
}

@test "TC4: round-trip preflight->build.sh->verify-module ALL PASS->lint-timers PASS (SKIP without niagara_home/JDK8)" {
  [ -n "${NIAGARA_HOME:-}" ] || skip "no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)"
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  # D2 layout: emitted root = <out-dir>/<ModuleName>, and inside it the <Module>/<Module>-rt findProjects
  # nesting → the -rt artifact is at $OUT/MinimalPan/MinimalPan/MinimalPan-rt.
  ROOT="$OUT/MinimalPan"; RT="$ROOT/MinimalPan/MinimalPan-rt"
  # scaffold -> the emitted module builds signed (build.sh runs slotomatic first, D3) + passes the gate + the timer lint.
  # build.sh usage: <module-root> <MOD> [niagara_home]; it runs slotomatic + jar + the verify gate itself.
  # build.sh refuses a real niagara_home (no .niagara-mirror): build into a throwaway mirror, never the install.
  run bash "$KIT/toolbelt/mirror-niagara-home.sh" "$NIAGARA_HOME" "$BATS_TEST_TMPDIR/nh-mirror" MinimalPan-rt.jar
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/build.sh" "$ROOT" MinimalPan "$BATS_TEST_TMPDIR/nh-mirror"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ALL PASS"* ]]
  run bash "$KIT/toolbelt/lint-timers.sh" "$RT/src"
  [ "$status" -eq 0 ]
}

@test "TC-SIGN: both scaffold types emit the root signing block, so lint-structure L15 stays silent (issue #266)" {
  # Named mutation: drop the niagaraSigning { aliases; signingProfileFile } part of the block from a fixture
  # root (keep allowDefaultProfile) -> the lint exits 0 with an L15 WARN on this root, and this test fails.
  for t in logic dashboard driver; do
    run "$SC" "Sign${t^}" "$OUT/$t" --type "$t"
    [ "$status" -eq 0 ]
    [[ "$output" == *"niagaraSigningAlias"* ]]
    root="$OUT/$t/Sign${t^}"
    mkdir -p "$root/.git"                      # the L10/L14 upward walk stops here, not in the test tmpdir
    grep -q 'allowDefaultProfile.set(false)' "$root/build.gradle.kts"
    run "$KIT/toolbelt/lint-structure.sh" "$root/Sign${t^}"
    [ "$status" -eq 0 ]                        # not 3 (usage/env: wrong path) and not 1 (FAIL)
    [[ "$output" != *"L15"* ]]
    # positive control: the same root without its aliases line WARNs, so the lint reached this exact root
    sed -i '/aliases\.set(/d' "$root/build.gradle.kts"
    run "$KIT/toolbelt/lint-structure.sh" "$root/Sign${t^}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"WARN  lint-structure  $root/build.gradle.kts  L15: "*"aliases"* ]]
  done
}

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
           AcmeLink-rt/src/com/example/AcmeLink/AcmeLinkCodec.java \
           AcmeLink-rt/srcTest/test/com/example/AcmeLink/AcmeLinkCodecTest.java \
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
  run grep -En '^import (javax\.baja|com\.tridium)' "$S/AcmeLinkCodec.java"
  [ "$status" -eq 1 ]
}

@test "TC-DRV5: the emitted codec srcTest passes as pure JUnit (SKIP without javac or the cached junit jar)" {
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  run "$KIT/toolbelt/run-pure-test.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-rt" com.example.AcmeLink.AcmeLinkCodecTest
  [ "$status" -eq 0 ]
  [[ "$output" == *"OK ("* ]]
}

@test "TC-DRV6: the codec test bites: a codec that skips the checksum check makes the srcTest fail (exit 1)" {
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  C="$OUT/AcmeLink/AcmeLink/AcmeLink-rt/src/com/example/AcmeLink/AcmeLinkCodec.java"
  grep -q 'if (sum != buf\[off + CHECKSUM\]) throw' "$C"
  sed -i '/if (sum != buf\[off + CHECKSUM\]) throw/d' "$C"
  run grep -c 'IllegalArgumentException("bad checksum")' "$C"
  [ "$output" = "0" ]
  run "$KIT/toolbelt/run-pure-test.sh" "$OUT/AcmeLink/AcmeLink/AcmeLink-rt" com.example.AcmeLink.AcmeLinkCodecTest
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

@test "TC-DRV7: the emitted driver ships its station loopback test, registered for the test jar only, and its manager agent" {
  # Named mutations: drop moduleTest-include.xml -> verify-module moduletest WARNs again and this fails;
  # drop an <on type> from the -wb module-include.xml -> lint-agent-on-include-sync FAILs (the manager never shows).
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  R="$OUT/AcmeLink/AcmeLink/AcmeLink-rt"
  T="$R/srcTest/test/com/example/AcmeLink"
  # registered in moduleTest-include.xml, with its source, and never in the production include
  grep -q 'class="test.com.example.AcmeLink.BAcmeLinkLoopbackTest"' "$R/moduleTest-include.xml"
  [ -f "$T/BAcmeLinkLoopbackTest.java" ]
  [ -f "$T/AcmeLinkLoopbackDevice.java" ]
  run grep -c 'LoopbackTest' "$R/module-include.xml"
  [ "$output" = "0" ]
  # a BTestNgStation test that waits by polling, never a fixed sleep, against a loopback port-0 mock
  grep -q 'extends BTestNgStation' "$T/BAcmeLinkLoopbackTest.java"
  grep -q 'TestHelper.assertWillBeTrue' "$T/BAcmeLinkLoopbackTest.java"
  run grep -l 'Thread.sleep' "$T/BAcmeLinkLoopbackTest.java" "$T/AcmeLinkLoopbackDevice.java"
  [ "$status" -eq 1 ]
  grep -q 'new DatagramSocket(0, InetAddress.getLoopbackAddress())' "$T/AcmeLinkLoopbackDevice.java"
  grep -q 'moduleTestImplementation(":test-wb")' "$R/AcmeLink-rt.gradle.kts"
  # the -wb manager's agent is in its include line, so the processor cannot drop it
  run "$KIT/toolbelt/lint-agent-on-include-sync.sh" "$OUT/AcmeLink/AcmeLink"
  [ "$status" -eq 0 ]
  [[ "$output" != *FAIL* ]]
}

@test "TC-DRV8: the loopback mock answers the driver codec on 127.0.0.1 and stays silent on demand (SKIP without javac)" {
  # Named mutation: answer with seq + 1 in the mock -> the round trip below sees a wrong sequence and fails.
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  R="$OUT/AcmeLink/AcmeLink/AcmeLink-rt"
  P="$BATS_TEST_TMPDIR/probe"
  mkdir -p "$P/test/com/example/AcmeLink" "$P/classes"
  cat > "$P/test/com/example/AcmeLink/Probe.java" <<'JAVA'
package test.com.example.AcmeLink;
import java.net.*;
import com.example.AcmeLink.AcmeLinkCodec;
import com.example.AcmeLink.AcmeLinkCodec.Frame;
public class Probe {
  static Frame ask(DatagramSocket s, int port, Frame req) throws Exception {
    byte[] out = AcmeLinkCodec.encode(req);
    s.send(new DatagramPacket(out, out.length, InetAddress.getLoopbackAddress(), port));
    byte[] in = new byte[AcmeLinkCodec.BUFFER_LEN];
    DatagramPacket p = new DatagramPacket(in, in.length);
    try { s.receive(p); } catch (SocketTimeoutException e) { return null; }
    return AcmeLinkCodec.decode(p.getData(), p.getOffset(), p.getLength());
  }
  public static void main(String[] a) throws Exception {
    try (AcmeLinkLoopbackDevice dev = AcmeLinkLoopbackDevice.start(4711);
         DatagramSocket s = new DatagramSocket(0, InetAddress.getLoopbackAddress())) {
      s.setSoTimeout(2000);
      Frame ping = AcmeLinkCodec.request(7, AcmeLinkCodec.CMD_PING, 0, 0);
      Frame r = ask(s, dev.port(), ping);
      if (r == null || !AcmeLinkCodec.answers(ping, r) || r.value != 4711) throw new AssertionError("ping: " + r);
      Frame w = AcmeLinkCodec.request(8, AcmeLinkCodec.CMD_WRITE, 2, 42);
      r = ask(s, dev.port(), w);
      if (r == null || !AcmeLinkCodec.answers(w, r) || !Integer.valueOf(42).equals(dev.register(2))) throw new AssertionError("write: " + r);
      Frame rd = AcmeLinkCodec.request(9, AcmeLinkCodec.CMD_READ, 2, 0);
      r = ask(s, dev.port(), rd);
      if (r == null || !AcmeLinkCodec.answers(rd, r) || r.value != 42) throw new AssertionError("read: " + r);
      dev.setAnswering(false);
      s.setSoTimeout(500);
      if (ask(s, dev.port(), AcmeLinkCodec.request(10, AcmeLinkCodec.CMD_PING, 0, 0)) != null) throw new AssertionError("silent mock answered");
      if (dev.received(AcmeLinkCodec.CMD_PING) != 2) throw new AssertionError("pings recorded: " + dev.received(AcmeLinkCodec.CMD_PING));
      System.out.println("PROBE OK");
    }
  }
}
JAVA
  run javac -source 8 -target 8 -nowarn -d "$P/classes" -sourcepath "$R/src:$R/srcTest:$P" \
    "$P/test/com/example/AcmeLink/Probe.java"
  [ "$status" -eq 0 ]
  run java -cp "$P/classes" test.com.example.AcmeLink.Probe
  [ "$status" -eq 0 ]
  [[ "$output" == *"PROBE OK"* ]]
}

# ---------------------------------------------------------------------------------------------------------
# Station-test overlay (round 10): logic + dashboard get a BTestNgStation skeleton from templates/station-test
# (Fox-off fixture base, a link-in sample test, moduleTest-include.xml). The driver keeps its own loopback
# test (TC-DRV7) and gets no overlay. Doctrine: types/moduleTest.md, types/driver-authoring.md § 12.1.
# ---------------------------------------------------------------------------------------------------------

# st_golden_diff <type> <Mod>: the emitted overlay files of <Mod>-rt equal the golden for <type>.
st_golden_diff() {
  local t="$1" mod="$2"
  local r="$OUT/$mod/$mod/$mod-rt" g="$ST_GOLDEN/$t"
  diff "$g/moduleTest-include.xml" "$r/moduleTest-include.xml" || return 1
  diff "$g/BModuleAStationTestBase.java" "$r/srcTest/test/com/example/$mod/B${mod}StationTestBase.java" || return 1
  diff "$g/BModuleALinkTest.java" "$r/srcTest/test/com/example/$mod/B${mod}LinkTest.java" || return 1
}

@test "TC-ST1: --type logic emits the station-test overlay equal to its golden, with both test dependencies" {
  # Named mutation: skip the overlay copy for logic in scaffold-module.sh -> the diff finds no files and fails.
  run "$SC" ModuleA "$OUT" --type logic --vendor Example
  [ "$status" -eq 0 ]
  run st_golden_diff logic ModuleA
  [ "$status" -eq 0 ]
  G="$OUT/ModuleA/ModuleA/ModuleA-rt/ModuleA-rt.gradle.kts"
  grep -q 'moduleTestImplementation(":test-wb")' "$G"
  grep -q 'moduleTestImplementation("junit:junit:4.13.2")' "$G"
}

@test "TC-ST2: --type dashboard emits the station-test overlay equal to its golden, with both test dependencies" {
  # Named mutation: drop "status" from the dashboard link-in list -> the golden diff fails.
  run "$SC" ModuleA "$OUT" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  run st_golden_diff dashboard ModuleA
  [ "$status" -eq 0 ]
  G="$OUT/ModuleA/ModuleA/ModuleA-rt/ModuleA-rt.gradle.kts"
  grep -q 'moduleTestImplementation(":test-wb")' "$G"
  grep -q 'moduleTestImplementation("junit:junit:4.13.2")' "$G"
  # the overlay lands in -rt only, never in -ux
  run find "$OUT/ModuleA/ModuleA/ModuleA-ux" -name moduleTest-include.xml
  [ -z "$output" ]
}

@test "TC-ST3: a renamed overlay leaves no template token and registers its tests for the test jar only" {
  # Named mutation: drop the com.example -> com.<vendor> substitution -> the token grep finds com.example.
  for t in logic dashboard; do
    run "$SC" ColdBox "$OUT/$t" --type "$t" --vendor Acme
    [ "$status" -eq 0 ]
    R="$OUT/$t/ColdBox/ColdBox/ColdBox-rt"
    T="$R/srcTest/test/com/acme/ColdBox"
    [ -f "$T/BColdBoxStationTestBase.java" ]
    [ -f "$T/BColdBoxLinkTest.java" ]
    run grep -rlE 'ModuleA|com\.example|@LINK_IN_SLOTS@|Example' "$R/moduleTest-include.xml" "$T"
    [ "$status" -eq 1 ]
    grep -q 'class="test.com.acme.ColdBox.BColdBoxLinkTest"' "$R/moduleTest-include.xml"
    grep -q 'import com.acme.ColdBox.BColdBox;' "$T/BColdBoxLinkTest.java"
    run grep -c 'LinkTest\|StationTestBase' "$R/module-include.xml"
    [ "$output" = "0" ]
  done
}

@test "TC-ST4: the driver gets no station-test overlay (its loopback test stays the only station test)" {
  run "$SC" AcmeLink "$OUT" --type driver --vendor Example
  [ "$status" -eq 0 ]
  run find "$OUT/AcmeLink" -name 'B*StationTestBase.java' -o -name 'B*LinkTest.java'
  [ -z "$output" ]
}

@test "TC-ST5: the overlay fixture turns Fox off and tolerates only the exact Fox-not-serving failure" {
  # Named mutation: catch every Exception in startStation (drop the message check) -> the grep for the
  # rethrow guard fails.
  run "$SC" ModuleA "$OUT" --type logic --vendor Example
  [ "$status" -eq 0 ]
  B="$OUT/ModuleA/ModuleA/ModuleA-rt/srcTest/test/com/example/ModuleA/BModuleAStationTestBase.java"
  for slot in foxEnabled foxsEnabled multicastEnabled enableAnnouncement; do
    grep -q "fox.set(\"$slot\", BBoolean.FALSE);" "$B"
  done
  grep -q 'static final String FOX_NOT_STARTED = "Fox service did not start in time";' "$B"
  grep -q 'if (!FOX_NOT_STARTED.equals(e.getMessage()) || !station.isRunning()) throw e;' "$B"
  grep -q '@AfterMethod(alwaysRun = true)' "$B"
}

# st_link_in_flags <rt-dir> <Mod> <vendor-lc>: one "<slot> <flags>" line per LINK_IN_SLOTS entry of the
# emitted link test, the flags read from that slot's @NiagaraProperty block in the emitted component
# ("MISSING" when the component declares no such property).
st_link_in_flags() {
  local r="$1" mod="$2" v="$3"
  local test="$r/srcTest/test/com/$v/$mod/B${mod}LinkTest.java" comp="$r/src/com/$v/$mod/B${mod}.java"
  local slots
  slots="$(sed -n 's/.*LINK_IN_SLOTS = { \(.*\) };.*/\1/p' "$test" | tr -d '" ' | tr ',' ' ')"
  [ -n "$slots" ] || return 1
  for s in $slots; do
    awk -v want="$s" '
      /@NiagaraProperty\(/ { inblk = 1; name = ""; flags = ""; next }
      inblk && /name *= *"/ { n = $0; sub(/.*name *= *"/, "", n); sub(/".*/, "", n); name = n }
      inblk && /flags *=/ { f = $0; sub(/.*flags *= */, "", f); flags = f }
      inblk && /^\)/ { if (name == want) { print want " " flags; found = 1 } inblk = 0 }
      END { if (!found) print want " MISSING" }
    ' "$comp"
  done
}

@test "TC-ST6: no link-in slot named by the emitted link test is READONLY in the emitted component (source-level guard)" {
  # The station test asserts the same at run time (linkInSlotsAreNotReadonly): LinkCheck refuses a
  # READONLY target (javax.baja.sys.LinkCheck.make, linkcheck.propReadonly), types/dashboard.md link-in rule.
  # Named mutation: add Flags.READONLY to the dashboard fixture's status slot -> this test fails.
  for t in logic dashboard; do
    run "$SC" ModuleA "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    run st_link_in_flags "$OUT/$t/ModuleA/ModuleA/ModuleA-rt" ModuleA example
    [ "$status" -eq 0 ]
    [[ "$output" != *MISSING* ]]
    [[ "$output" != *READONLY* ]]
  done
}
