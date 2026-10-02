package demo;
import javax.baja.sys.*;
public final class LiteralMinutesZero extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, BRelTime.makeMinutes(0), null, null); } // FAIL literal-zero
}
