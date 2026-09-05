package demo;
import javax.baja.sys.*;
public final class LiteralPos extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, BRelTime.makeSeconds(5), null, null); }   // PASS: literal > 0
}
