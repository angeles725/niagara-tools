package demo;
import javax.baja.sys.*;
public final class MultiLinePos extends BComponent {
  private Clock.Ticket t;
  public void arm() {
    t = Clock.schedule(
        this,
        BRelTime.makeSeconds(5),   // positive literal on its own line -> clean
        null,
        null);
  }
}
