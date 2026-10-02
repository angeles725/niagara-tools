package demo;
import javax.baja.sys.*;
public final class LiteralHours extends BComponent {
  private static final BRelTime HOUR = BRelTime.makeHours(1);
  private Clock.Ticket a, b;
  public void arm() {
    a = Clock.schedule(this, BRelTime.makeMinutes(2), null, null);   // positive literal minutes -> clean
    b = Clock.schedulePeriodically(this, HOUR, null, null);          // positive constant hours -> clean
  }
}
