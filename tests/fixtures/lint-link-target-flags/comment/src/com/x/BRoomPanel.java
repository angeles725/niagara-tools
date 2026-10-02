package com.x;
import javax.baja.sys.*;
// Facade room panel: link-in display slots + operator config.
@NiagaraType
// Commissioning link: written by BLink from the evaporator freeze-stat point.
@NiagaraProperty(
  name = "evap1FreezeActive",
  type = "boolean",
  defaultValue = "false",
  flags = Flags.SUMMARY | Flags.READONLY
)
// Linked from the control module compressor state point (link-in display slot).
@NiagaraProperty(
  name = "comp1State",
  type = "int",
  defaultValue = "0",
  flags = Flags.SUMMARY
)
public final class BRoomPanel extends BComponent {
}
