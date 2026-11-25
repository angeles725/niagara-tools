package com.example.rpc;

public class BGuardAfter
{
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void late(String slot, double v, Context cx)
  {
    c.set(prop, BDouble.make(v), cx);
    checkCanWrite(cx);
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void early(String slot, double v, Context cx)
  {
    checkCanWrite(cx);
    c.set(prop, BDouble.make(v), cx);
  }
}
