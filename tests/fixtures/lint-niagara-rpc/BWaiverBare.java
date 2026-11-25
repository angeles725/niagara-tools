package com.example.rpc;

public class BWaiverBare
{
  // lint-niagara-rpc: reviewed
  @NiagaraRpc(permissions = "unrestricted", transports = { @Transport(type = TransportType.web) })
  public static String lexicon(String key, Context cx)
  {
    return text(key);
  }
}
