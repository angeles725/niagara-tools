#!/usr/bin/env bats
# Campaign-11 REDs — the section-D method-boundary parser must standardise on PEAK
# brace depth (max_d), not NET. At kit dab0807 (v0.21.0) the three parser copies
# disagree on ONE-LINER methods: lint-timers.sh and lint-silent-protection.sh track
# NET depth (method open only when `brace_depth > old_depth`), so a `void x(){ ...; }`
# one-liner (net brace change 0) is never captured as a method — its companion-flag /
# trip slips through. lint-ext-writable-shape.sh already tracks PEAK (max_d) and
# catches one-liners. Fix: port PEAK + a getter/setter/is accessor skip into the two
# NET copies (the shared fragment, C11 T1). [ev: investigador1 B832 593019540]
#
# RED-for-the-right-reason on dab0807: the one-liner fixtures are the multi-line
# controls (which the lints DO flag) reflowed to a single line — the ONLY difference
# is the reflow, proving the miss is the NET-vs-PEAK parser bug, not fixture cleanliness.
#
# B832-G2 NOTE (not a pin — no biting fixture found): the silent-protection Case-B
# backward scan (lint-silent-protection.sh:~315,342) strips `//` line comments only,
# not `/* */` block comments, from both the brace-count line and the scanned line — a
# latent comment-satisfiable risk (RP15/RP17 class). It did NOT flip any tried fixture:
# a `{` inside a `/* */` block comment is miscounted in BOTH the depth tally and the
# Case-B `index(lk,"{")` stop, and the trip stays within the captured [start,end] method
# range, so the WARN survives. Recorded as a note for the shared-fragment fix (C11 T1):
# the fragment should strip `/* */` (single-line) alongside `//` for robustness. Left
# unpinned rather than shipping a vacuous pin.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LT="$KIT/toolbelt/lint-timers.sh"
  LSP="$KIT/toolbelt/lint-silent-protection.sh"
}

@test "C11-tl-oneliner: a FIELD flag set true beside Clock.schedule* in a ONE-LINER method still FAILs companion-flag (NET parser misses the one-liner -> RED on dab0807; PEAK fix -> GREEN)" {
  D="$BATS_TEST_TMPDIR/tl-oneliner"; mkdir -p "$D"
  cat > "$D/BOneLinerArm.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BOneLinerArm extends BComponent {
  private boolean startingUp = false;         // class FIELD — CAN stay stuck across stop/restart
  private Clock.Ticket powerOnTicket;
  public void arm() { startingUp = true; powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null); }
  public void doPowerOnExpired() { startingUp = false; }   // cleared only in expiry — does NOT count
  public void stopped() throws Exception { super.stopped(); if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* ]] && [[ "$output" == *"startingUp"* ]]
}

@test "C11-sp-oneliner: a detected trip (Math.min inside if) in a ONE-LINER method with NO alarm surface still WARNs (NET parser misses the one-liner -> RED on dab0807; PEAK fix -> GREEN)" {
  D="$BATS_TEST_TMPDIR/sp-oneliner"; mkdir -p "$D/com/x"
  cat > "$D/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double suctionLowLimit, boolean suctionValid) { if (suctionValid && suction < suctionLowLimit) target = Math.min(target, onCount - 1); return target; }
}
JAVA
  run "$LSP" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
  [[ "$output" == *"lint-silent-protection"* ]] && [[ "$output" == *"CompressorControl.java"* ]]
}

@test "C11-g1-setter: a ONE-LINER accessor (setInhibited) with a guarded boolean write is NOT protection logic -> 0 WARN (B832-G1 getter/setter/is skip; GREEN on dab0807 and on the PEAK+skip fix, flips a naive PEAK-only fix that would false-WARN the setter)" {
  D="$BATS_TEST_TMPDIR/g1-setter"; mkdir -p "$D/com/x"
  cat > "$D/com/x/Widget.java" <<'JAVA'
package com.x;
public class Widget {
  private boolean inhibited;
  public void setInhibited(boolean b) { if (b) this.inhibited = true; }
}
JAVA
  run "$LSP" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
}
