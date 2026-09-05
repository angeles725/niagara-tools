package demo;
import javax.baja.sys.*;
public final class Floored extends BComponent {
  private Clock.Ticket t;
  public void arm(long delayMs) {
    long d = Math.max(delayMs, 1L);                            // PASS: floored at 1L
    t = Clock.schedule(this, BRelTime.make(d), null, null);
  }
}
