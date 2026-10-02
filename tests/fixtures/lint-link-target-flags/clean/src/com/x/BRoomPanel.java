package com.x;
import javax.baja.sys.*;
@NiagaraType
// Linked from the control module (link-in display slot) -> SUMMARY only, linkable.
@NiagaraProperty(
  name = "evap1FreezeActive",
  type = "boolean",
  defaultValue = "false",
  flags = Flags.SUMMARY
)
// Timer anchor set by this component itself (no external Link targets it).
@NiagaraProperty(
  name = "defrostStart",
  type = "BAbsTime",
  defaultValue = "BAbsTime.NULL",
  flags = Flags.TRANSIENT | Flags.SUMMARY | Flags.READONLY
)
// Operator HOA choice: persisted (never TRANSIENT).
@NiagaraProperty(
  name = "comp1Mode",
  type = "int",
  defaultValue = "0",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
// Computed status mode, not an operator choice: TRANSIENT is fine.
@NiagaraProperty(
  name = "defrostMode",
  type = "int",
  defaultValue = "0",
  flags = Flags.TRANSIENT | Flags.SUMMARY | Flags.READONLY
)
public final class BRoomPanel extends BComponent {
}
