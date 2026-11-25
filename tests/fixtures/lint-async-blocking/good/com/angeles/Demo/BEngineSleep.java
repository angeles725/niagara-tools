package com.angeles.Demo;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Action;
import javax.baja.sys.BComponent;
import javax.baja.sys.BValue;
import javax.baja.sys.Context;
import javax.baja.sys.Flags;
import javax.baja.util.IFuture;
import javax.baja.util.Invocation;

// The sanctioned offload: ASYNC action, post() override handing a new Invocation to the module's own executor.
@NiagaraType
@NiagaraAction(name = "execute", flags = Flags.ASYNC)
public class BEngineSleep extends BComponent
{
  public IFuture post(Action action, BValue arg, Context cx)
  {
    pool.execute(new Invocation(this, action, arg, cx));
    return null;
  }

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
