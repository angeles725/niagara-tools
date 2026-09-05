package demo;
import javax.baja.sys.*;
public final class Overdue extends BComponent {
  private Clock.Ticket t;
  // Real shape from ColdRoomPan BDefrostController.armTrigger: overdue -> delayMs 0, floored at 0L not 1L.
  public void arm(long intervalMs, long elapsed) {
    long delayMs = (elapsed >= intervalMs) ? 0L : (intervalMs - elapsed);
    long d = Math.max(delayMs, 0L);
    t = Clock.schedule(this, BRelTime.make(d), null, null);   // FAIL: d has no >0 floor -> "time <= 0"
  }
}
