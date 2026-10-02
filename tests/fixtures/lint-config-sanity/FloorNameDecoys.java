package demo;
import javax.baja.sys.*;
// FloorNameDecoys: names that only CONTAIN "minon" / "cutout" are not a permanent-minimum floor or an
// LP cutout floor: adminOnline (a flag count), minOnTime (an int-seconds short-cycle timer, not a floor),
// cutoutDelay (a delay of 0 is not a disabled cutout). No CS4.
@NiagaraType
@NiagaraProperty(name = "adminOnline", type = "int", defaultValue = "1", flags = Flags.OPERATOR)
@NiagaraProperty(name = "minOnTime", type = "int", defaultValue = "300", flags = Flags.OPERATOR)
@NiagaraProperty(name = "cutoutDelay", type = "int", defaultValue = "0", flags = Flags.OPERATOR)
public final class FloorNameDecoys extends BComponent {
}
