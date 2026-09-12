package demo;
import javax.baja.sys.*;
// IntervalBad: defrostInterval default (30s) <= defrostDuration default (60s) -> CS1 FAIL.
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
