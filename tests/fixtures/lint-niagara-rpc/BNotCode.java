package com.example.rpc;

/**
 * Do not write @NiagaraRpc(permissions = "unrestricted", transports = {}) in a module.
 */
public class BNotCode
{
  // @NiagaraRpc(permissions = "unrestricted", transports = {})
  static final String SAMPLE = "@NiagaraRpc(permissions = \"unrestricted\")";

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void guardOnlyInTextIsNoGuard(String slot, double v, Context cx)
  {
    // checkCanWrite(cx);
    String note = "OPERATOR_WRITE checkCanWrite(cx) requireTargetWrite(c, p, cx)";
    c.set(prop, BDouble.make(v), cx);
  }
}
