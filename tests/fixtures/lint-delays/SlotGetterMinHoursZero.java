package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.makeHours(1)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeHours(0))"))   // MIN = 0 h -> can schedule 0
public final class SlotGetterMinHoursZero extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // FAIL facet-min-zero
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
