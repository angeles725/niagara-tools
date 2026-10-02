package demo;
import javax.baja.sys.*;
public final class BlockCommentApostrophe extends BComponent {
  private Clock.Ticket t;
  public void arm(long delayMs) {
    /* don't reschedule twice */ int armed = 1; // delayMs > 0 is NOT checked here, this is comment text
    t = Clock.schedule(this, BRelTime.make(delayMs), null, null);
  }
}
