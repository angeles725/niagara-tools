package com.x;
import javax.baja.sys.*;
/**
 * BDash: one config slot (`setpoint`) and one display slot (`status`) that the
 * integrator links from a control module's output point. The anchor `defrostStart`
 * is set by this component itself.
 */
@NiagaraType
@SuppressWarnings("unused")
@NiagaraProperty(
  name = "setpoint",
  type = "double",
  defaultValue = "0d",
  flags = Flags.SUMMARY | Flags.READONLY
)
@NiagaraProperty(
  name = "defrostStart",
  type = "BAbsTime",
  defaultValue = "BAbsTime.NULL",
  flags = Flags.TRANSIENT | Flags.SUMMARY | Flags.READONLY
)
@NiagaraProperty(
  name = "status",
  type = "double",
  defaultValue = "0d",
  flags = Flags.SUMMARY | Flags.READONLY
)
public class BDash extends BComponent {
}
