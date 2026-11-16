#!/usr/bin/env bats
# scaffold-module --type driver: the station loopback test and its mock.
# Shared setup (the Go binary, KIT, SC, the fixture and golden paths, OUT): tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

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
import com.example.AcmeLink.model.AcmeLinkCodec;
import com.example.AcmeLink.model.AcmeLinkCodec.Frame;
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
