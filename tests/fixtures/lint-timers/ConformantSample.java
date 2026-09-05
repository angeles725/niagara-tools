package fixtures;
import javax.baja.sys.*;
// Lint fixture: a class that owns a Clock.Ticket and properly cancels it in stopped().
// Represents the conformant pattern (BDefrostController, BCompressorControl shape).
public class ConformantSample extends BComponent {
  private Clock.Ticket ticketField;
  public void start() {
    ticketField = Clock.schedule(this, BRelTime.makeSeconds(5), onTick, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (ticketField != null) { ticketField.cancel(); ticketField = null; }
  }
}
