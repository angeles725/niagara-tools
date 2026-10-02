package com.x;
import javax.baja.sys.*;
@NiagaraType
// Operator HOA for compressor 3 (0=AUTO, 1=OFF, 2=HAND), linked facade->control.
@NiagaraProperty(
  name = "comp3Mode",
  type = "int",
  defaultValue = "0",
  flags = Flags.SUMMARY | Flags.OPERATOR | Flags.TRANSIENT
)
@NiagaraProperty(name = "fanHoa", type = "int", defaultValue = "0", flags = Flags.OPERATOR | Flags.TRANSIENT)
public final class BCompressorPanel extends BComponent {
}
