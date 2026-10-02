package demo;
import javax.baja.sys.*;
// FloorCamelNames: a per-unit floor (comp2MinOn=1) beside a disabled LP cutout (lpCutout=0) -> CS4 WARN.
@NiagaraType
@NiagaraProperty(name = "comp2MinOn", type = "int", defaultValue = "1", flags = Flags.OPERATOR)
@NiagaraProperty(name = "lpCutout", type = "double", defaultValue = "0.0", flags = Flags.OPERATOR)
public final class FloorCamelNames extends BComponent {
}
