// A finished device: reachability-only ping (course exam P3), discovery job submitted with the caller's Context.
package com.example.reach;

public class BReachDevice extends BNDevice
{
  public void doPing()
  {
    try (Socket s = new Socket())
    {
      s.connect(new InetSocketAddress(getHost(), getPort()), 2000);   // bounds the connect, not DNS
      pingOk();
    }
    catch (IOException e)
    {
      pingFail(e.getLocalizedMessage());
    }
  }

  public BOrd doSubmitDiscoveryJob(BReachDiscoveryPreferences prefs, Context cx)
  {
    return new BReachLearnJob(this, prefs).submit(cx);
  }
}
