package com.x;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(
  name = "evap2InDrip",
  type = "boolean",
  defaultValue = "false",
  flags = Flags.SUMMARY | Flags.READONLY
)
@NiagaraProperty(
  name = "temperatureSetpoint",
  type = "double",
  defaultValue = "-5.0",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
public final class BRoomPanel extends BComponent {
}
