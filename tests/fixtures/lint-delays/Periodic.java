package demo;
import javax.baja.sys.*;
public final class Periodic extends BComponent {
  private Clock.Ticket t;
  public void arm(long p) { t = Clock.schedulePeriodically(this, BRelTime.make(Math.max(p, 0L)), null, null); } // FAIL: period no >0 floor
}
