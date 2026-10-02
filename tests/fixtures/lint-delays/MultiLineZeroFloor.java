package demo;
import javax.baja.sys.*;
public final class MultiLineZeroFloor extends BComponent {
  private Clock.Ticket t;
  public void arm(long delayMs) {
    t = Clock.schedule(this,
        BRelTime.make(Math.max(delayMs, 0L)),
        null, null);   // FAIL zero-floor (the real bug shape split over lines)
  }
}
