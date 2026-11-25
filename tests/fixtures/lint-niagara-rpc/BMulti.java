package com.example.rpc;

public interface BMulti
{
  @javax.baja.rpc.NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  String read(String slot, Context cx);

  @javax.baja.rpc.NiagaraRpc(
    permissions = "unrestricted",
    transports = { @Transport(type = TransportType.web) })
  String open(String slot, Context cx);

  @NiagaraRpc(permissions = "w", transports = { @Transport(type = TransportType.web) })
  default void write(String slot, double v, Context cx)
  {
    owner().set(prop, BDouble.make(v), cx);
  }
}
