package com.angeles.MinimalPan;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;

/**
 * BMinimalPan — real-shape fixture (spaced = and multi-line annotations).
 * Used by SR10 to verify parse_slots handles the production annotation form.
 */
@NiagaraType
@NiagaraProperty(
  name = "setpoint",
  type = "double",
  defaultValue = "0d"
)
@NiagaraProperty(
  name = "interval",
  type = "baja:Double",
  defaultValue = "30d"
)
@NiagaraAction(name = "tickExpired")
public class BMinimalPan extends BComponent
{
  public BMinimalPan() {}
}
