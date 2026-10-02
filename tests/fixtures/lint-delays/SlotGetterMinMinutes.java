package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(name = "lockout", type = "BRelTime",
  defaultValue = "BRelTime.makeMinutes(5)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeMinutes(1))"))   // MIN = 1 min -> floored in facets
public final class SlotGetterMinMinutes extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, getLockout(), null, null); } // WARN facet-floor, not FAIL
  public BRelTime getLockout() { return (BRelTime) get("lockout"); }
}
