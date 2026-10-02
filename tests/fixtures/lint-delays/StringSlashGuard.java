package demo;
import javax.baja.sys.*;
public final class StringSlashGuard extends BComponent {
  private Clock.Ticket t;
  public void arm(long delayMs, String url) {
    if (!url.startsWith("http://") && delayMs > 0) {   // the "//" inside the literal is not a comment
      t = Clock.schedule(this, BRelTime.make(delayMs), null, null);
    }
  }
}
