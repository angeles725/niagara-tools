package com.example.rpc;

public class BNoContext
{
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String lastIsString(String key, String name)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String noParams()
  {
    return "x";
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String contextFirst(Context cx, String name)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String subtype(String name, SecurableContext cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String multiLine(
    String name,
    List<String> more)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String asArray(String name, Context[] cxs)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String plain(String name, Context cx)
  {
    return name;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String qualified(Map<String, List<String>> args, javax.baja.sys.Context cx)
  {
    return "x";
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String modifiers(String name, @SuppressWarnings("unused") final Context cx)
  {
    return name;
  }

  // lint-niagara-rpc: reviewed legacy two-argument shape kept for the old workbench caller
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.box) })
  public static String waived(String name)
  {
    return name;
  }

  public static String notAnRpc(String name)
  {
    return name;
  }
}
