package com.example.rpc;

public class BNoBox
{
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web), @Transport(type = TransportType.fox) })
  public static String webAndFox(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = @Transport(type = TransportType.web))
  public static String singleWeb(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = {})
  public static String none(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web), @Transport(type = TransportType.box) })
  public static String webAndBox(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = box) })
  public static String staticImport(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = TRANSPORTS)
  public static String constant(String name, Context cx)
  {
    return name;
  }

  // lint-niagara-rpc: reviewed this one is called from a fox view only
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.fox) })
  public static String waived(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(
    permissions = "r",
    transports = {
      @Transport(type = TransportType.web)
    })
  public static String multiLine(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(transports = @Transport(type = TransportType.fox), permissions = "r", isSecure = true)
  public static String attributesAfter(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web), EXTRA })
  public static String mixed(String name, Context cx)
  {
    return name;
  }
}
