package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.make(1800000)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(1))"))   // MIN >= 1 s -> floored in facets
public final class SlotGetterMinPos extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // WARN: floor lives in facets, advisory
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
