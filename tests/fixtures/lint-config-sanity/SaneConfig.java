package demo;
import javax.baja.sys.*;
// SaneConfig: interval(90s) > duration(60s), setpoint non-zero, no comment-only flag combos -> clean.
@NiagaraType
public final class SaneConfig extends BComponent {
  @NiagaraProperty(
    name = "defrostInterval",
    defaultValue = "BRelTime.makeSeconds(90)"
  )
  private BRelTime defrostInterval;

  @NiagaraProperty(
    name = "defrostDuration",
    defaultValue = "BRelTime.makeSeconds(60)"
  )
  private BRelTime defrostDuration;

  @NiagaraProperty(
    name = "temperatureSetpoint",
    flags = Flags.OPERATOR,
    defaultValue = "BDouble.make(-20.0)",
    facets = "BFacets.make(BFacets.MIN, BDouble.make(-40.0))"
  )
  private BDouble temperatureSetpoint;
}
