package demo;
import javax.baja.sys.*;
// SetpointZero: temperatureSetpoint default=0.0, OPERATOR flags, no min>0 facet -> CS2 FAIL.
@NiagaraType
public final class SetpointZero extends BComponent {
  @NiagaraProperty(
    name = "temperatureSetpoint",
    flags = Flags.OPERATOR,
    defaultValue = "BDouble.make(0.0)"
  )
  private BDouble temperatureSetpoint;
}
