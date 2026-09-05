package demo;
import javax.baja.sys.*;
// The guard checks a DIFFERENT expression (getOther) than the scheduled delay (getStartDelay),
// so it does NOT protect the delay -> lint-delays must STILL FAIL facet-min-zero (D2c is
// expression-specific, not "any positivity check anywhere in the method").
@NiagaraType
@NiagaraProperty(name = "startDelay", type = "BRelTime", defaultValue = "BRelTime.make(0)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(0))"))
@NiagaraProperty(name = "other", type = "BRelTime", defaultValue = "BRelTime.make(0)")
public final class MisGuarded extends BComponent {
  private Clock.Ticket t;
  void rising() {
    if (getOther().getMillis() > 0L)
      t = Clock.schedule(this, getStartDelay(), null, null);   // startDelay MIN=0, guard is on `other`
  }
  public BRelTime getStartDelay() { return (BRelTime) get("startDelay"); }
  public BRelTime getOther()      { return (BRelTime) get("other"); }
}
