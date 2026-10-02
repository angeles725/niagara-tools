package demo;
import javax.baja.sys.*;
// TimerNotFloor: minOnTime is a BRelTime short-cycle guard, not a permanent-minimum floor -> no CS4
// even though pumpDownCutout is 0 (disabled).
@NiagaraType
@NiagaraProperty(name = "minOnTime", type = "BRelTime", defaultValue = "BRelTime.makeMinutes(3)", flags = Flags.OPERATOR)
@NiagaraProperty(name = "pumpDownCutout", type = "double", defaultValue = "0", flags = Flags.OPERATOR)
public final class TimerNotFloor extends BComponent {
}
