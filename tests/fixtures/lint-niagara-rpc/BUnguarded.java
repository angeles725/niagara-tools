package com.example.rpc;

public class BUnguarded
{
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void setpoint(String slot, double v, Context cx)
  {
    BComponent c = resolve(slot);
    c.set(c.getProperty(slot), BDouble.make(v), cx);
  }

  @NiagaraRpc(transports = { @Transport(type = TransportType.box) })
  public void run(Context cx)
  {
    BComponent c = resolve("x");
    c.invoke(c.getAction("reset"), null, cx);
  }

  @NiagaraRpc(permissions = "i", transports = { @Transport(type = TransportType.fox) })
  public void reset(Context cx)
  {
    target.doReset(null, cx);
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void typed(String slot, double v, Context cx)
  {
    BComponent c = resolve(slot);
    c.setDouble(c.getProperty(slot),
                v, cx);
  }
}
