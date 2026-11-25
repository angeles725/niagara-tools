package com.example.rpc;

public class BUnrestricted
{
  @NiagaraRpc(permissions = "unrestricted", transports = { @Transport(type = TransportType.web) })
  public static String readLabel(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "Unrestricted", transports = { @Transport(type = TransportType.box) })
  public static String readMixedCase(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "", transports = { @Transport(type = TransportType.box) })
  public static String readEmpty(String name, Context cx)
  {
    return name;
  }
}
