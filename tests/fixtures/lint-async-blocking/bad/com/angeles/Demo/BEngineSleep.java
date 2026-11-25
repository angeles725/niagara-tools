package com.angeles.Demo;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;
import javax.baja.sys.Flags;

// Lab 9 shape: an ASYNC action whose handler sleeps 10 s on the shared engine thread, no post() override.
@NiagaraType
@NiagaraAction(name = "execute", flags = Flags.ASYNC)
public class BEngineSleep extends BComponent
{
  public void doExecute()
  {
    try
    {
      Thread.sleep(10000L);
    }
    catch (InterruptedException ex)
    {
      Thread.currentThread().interrupt();
    }
  }
}
