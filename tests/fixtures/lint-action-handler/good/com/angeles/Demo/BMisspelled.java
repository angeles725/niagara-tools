package com.angeles.Demo;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;
import javax.baja.sys.Context;

// GREEN fixture: the RED fixture with the handler spelled right.
@NiagaraType
@NiagaraAction(name = "reset")
public class BMisspelled extends BComponent
{
  public void doReset(Context cx)
  {
  }
}
