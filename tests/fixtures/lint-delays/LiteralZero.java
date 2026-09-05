package demo;
import javax.baja.sys.*;
public final class LiteralZero extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, BRelTime.make(0), null, null); }   // FAIL: literal 0
}
