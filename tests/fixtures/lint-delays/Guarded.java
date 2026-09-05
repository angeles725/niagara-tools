package demo;
import javax.baja.sys.*;
// D2c: each Clock.schedule below is protected by a SAME-METHOD positivity guard on the SAME
// delay expression, so lint-delays must PASS it "guarded at :N", not FAIL zero-floor/facet-min-zero.
// The four shapes are the real BEvaporatorUnit false positives on the fixed tree (c66e412).
@NiagaraType
@NiagaraProperty(name = "startDelay", type = "BRelTime", defaultValue = "BRelTime.make(0)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(0))"))
@NiagaraProperty(name = "stopDelay", type = "BRelTime", defaultValue = "BRelTime.make(0)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(0))"))
@NiagaraProperty(name = "fanOffDelay", type = "BRelTime", defaultValue = "BRelTime.make(0)",
  facets = @Facet("BFacets.make(BFacets.MIN, BRelTime.makeSeconds(0))"))
public final class Guarded extends BComponent {
  private Clock.Ticket t;

  // Shape 1 (BEvaporatorUnit:818/825): computed long guarded by `delayMs > 0L`.
  void powerOn() {
    long delayMs = computeDelay();
    if (delayMs > 0L)
      t = Clock.schedule(this, BRelTime.make(delayMs), null, null);
  }

  // Shape 2 (BEvaporatorUnit:920/923): the `== 0L` zero-branch does NOT reach the schedule (else does).
  void rising(boolean continuous) {
    if (continuous || getStartDelay().getMillis() == 0L)
      setBool(true);
    else
      t = Clock.schedule(this, getStartDelay(), null, null);
  }

  // Shape 3 (BEvaporatorUnit:935/936): guarded by `.getMillis() > 0L` in the condition.
  void falling() {
    if (getStopDelay().getMillis() > 0L)
      t = Clock.schedule(this, getStopDelay(), null, null);
  }

  // Shape 4 (BEvaporatorUnit:1113/1123): `== 0L` zero-branch handled; the else schedules.
  void defrost() {
    if (getFanOffDelay().getMillis() == 0L) { setBool(false); }
    else
      t = Clock.schedule(this, getFanOffDelay(), null, null);
  }

  long computeDelay() { return 0L; }
  void setBool(boolean b) {}
  public BRelTime getStartDelay()  { return (BRelTime) get("startDelay"); }
  public BRelTime getStopDelay()   { return (BRelTime) get("stopDelay"); }
  public BRelTime getFanOffDelay() { return (BRelTime) get("fanOffDelay"); }
}
