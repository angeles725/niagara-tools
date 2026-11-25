package com.angeles.Demo;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;
import javax.baja.sys.Context;

// RED fixture: the handler of the frozen action "reset" is misspelled (doReeset). Slot-o-Matic and the jar
// task succeed; the station refuses the type at load with "Missing action do method".
@NiagaraType
@NiagaraAction(name = "reset")
public class BMisspelled extends BComponent
{
  public void doReeset(Context cx)
  {
  }
}
