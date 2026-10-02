package demo;
import javax.baja.sys.*;
// FloorNoCutout: a permanent-minimum floor (minStagesOn=1) ships while the LP cutout floor
// (suctionLowLimit) is 0 = disabled -> one compressor always pulls with no LP protection (CS4 WARN).
@NiagaraType
@NiagaraProperty(name = "minStagesOn", type = "int", defaultValue = "1", flags = Flags.OPERATOR)
@NiagaraProperty(
  name = "suctionLowLimit",
  type = "double",
  defaultValue = "0.0",
  flags = Flags.OPERATOR,
  facets = "BFacets.make(BFacets.MIN, BDouble.make(0.0))"
)
public final class FloorNoCutout extends BComponent {
}
