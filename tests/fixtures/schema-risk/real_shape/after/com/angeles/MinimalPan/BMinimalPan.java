package com.angeles.MinimalPan;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;

/**
 * BMinimalPan — real-shape fixture after-snapshot.
 * Changes vs before:
 *   setpoint: double -> boolean  (retype_simple -> OUTAGE)
 *   temperature: added (add_slot -> SAFE)
 *   interval and tickExpired: unchanged
 */
@NiagaraType
@NiagaraProperty(
  name = "setpoint",
  type = "boolean",
  defaultValue = "false"
)
@NiagaraProperty(
  name = "interval",
  type = "baja:Double",
  defaultValue = "30d"
)
@NiagaraProperty(
  name = "temperature",
  type = "double",
  defaultValue = "0d"
)
@NiagaraAction(name = "tickExpired")
public class BMinimalPan extends BComponent
{
  public BMinimalPan() {}
}
