package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.makeDays(1)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeDays(0))"))   // MIN = 0 d -> can schedule 0
public final class SlotGetterMinDaysZero extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // FAIL facet-min-zero
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
