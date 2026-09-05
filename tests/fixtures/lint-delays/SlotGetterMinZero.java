package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.make(1800000)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(0))"))   // MIN = 0 s -> a 0 delay is allowed
public final class SlotGetterMinZero extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // FAIL: slot floor is 0
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
