package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.make(1800000)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.make(500))"))   // MIN = 500 ms (make(N) is ms)
public final class SlotGetterMinMs extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // WARN facet-floor, MIN is 500 ms
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
