package demo;
import javax.baja.sys.*;
// FloorWithCutout: same floor, LP cutout enabled (nonzero) -> no CS4. A BRelTime minOn timer next to
// a disabled cutout is a short-cycle guard, not a permanent-minimum floor -> no CS4 either.
@NiagaraType
@NiagaraProperty(name = "minStagesOn", type = "int", defaultValue = "1", flags = Flags.OPERATOR)
@NiagaraProperty(name = "suctionLowLimit", type = "double", defaultValue = "15.0", flags = Flags.OPERATOR)
public final class FloorWithCutout extends BComponent {
}
