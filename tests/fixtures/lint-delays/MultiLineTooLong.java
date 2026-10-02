package demo;
import javax.baja.sys.*;
public final class MultiLineTooLong extends BComponent {
  private Clock.Ticket t;
  public void arm() {
    t = Clock.schedule(
      this
      // a long hand-wrapped argument list
      //
      //
      //
      //
      //
      //
      //
      //
      //
      //
      //
      ,
      BRelTime.makeSeconds(5), null, null);
  }
}
