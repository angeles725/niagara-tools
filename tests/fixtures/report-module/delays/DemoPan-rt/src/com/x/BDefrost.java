package com.x;
import javax.baja.sys.*;
public final class BDefrost extends BComponent {
  private Clock.Ticket ticket;
  public void arm(long d) {
    // lint-delays defect: Math.max(d, 0L) floors at 0 -> Clock.schedule rejects delay <= 0.
    ticket = Clock.schedule(this, BRelTime.make(Math.max(d, 0L)), expired, null);
  }
  public void stopped() throws Exception {           // cancels in stopped() -> lint-timers PASSES,
    super.stopped();                                 // so only the lint-delays check can carry a FAIL
    if (ticket != null) { ticket.cancel(); ticket = null; }
  }
}
