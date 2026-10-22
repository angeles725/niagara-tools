package demo;
import javax.baja.sys.*;
public final class Owner extends BComponent {
  private Clock.Ticket ticket;
  public void arm() { ticket = Clock.schedule(this, BRelTime.makeSeconds(5), armed, null); }
  // no stopped() override: the ticket leaks on station stop
}
