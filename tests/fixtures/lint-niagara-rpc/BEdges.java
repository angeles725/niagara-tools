package com.example.rpc;

public class BEdges
{
  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  @SuppressWarnings({ "unchecked", "rawtypes" })
  public <T> T generic(String slot, Context cx) throws Exception
  {
    return (T) lookup(slot);
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public Object returned(String slot, Context cx)
  {
    return doReset(slot, cx);
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public String declarations(String slot, Context cx)
  {
    Runnable r = new Runnable()
    {
      public void doStep() { counter++; }
    };
    String t = """
        c.set(prop, value, cx);
        """;
    return t + slot;
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public void lambda(String slot, double v, Context cx)
  {
    Runnable r = () -> c.set(prop, BDouble.make(v), cx);
    r.run();
  }

  @NiagaraRpc(permissions = "r", transports = { @Transport(type = TransportType.web) })
  public String privileged(String slot, Context cx)
  {
    return AccessController.doPrivileged((PrivilegedAction<String>) () -> lookup(slot));
  }
}
