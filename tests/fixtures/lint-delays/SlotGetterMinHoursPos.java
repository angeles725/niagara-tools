package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "interval", type = "BRelTime",
  defaultValue = "BRelTime.makeHours(1)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeHours(2))"))   // MIN = 2 h
public final class SlotGetterMinHoursPos extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getInterval(), null, null); } // WARN facet-floor, MIN is 7200000 ms
  public BRelTime getInterval() { return (BRelTime) get("interval"); }
}
