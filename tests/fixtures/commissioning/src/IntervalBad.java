package demo;
import javax.baja.sys.*;
// CV-configfail fixture: defrostInterval default (30s) <= defrostDuration default (60s) -> CS1 FAIL.
// Used by commissioning-verify.bats CV-configfail: ensures config-sanity trips and exits 1.
@NiagaraType
public final class IntervalBad extends BComponent {
  @NiagaraProperty(
    name = "defrostInterval",
    defaultValue = "BRelTime.makeSeconds(30)"
  )
  private BRelTime defrostInterval;

  @NiagaraProperty(
    name = "defrostDuration",
    defaultValue = "BRelTime.makeSeconds(60)"
  )
  private BRelTime defrostDuration;
}
