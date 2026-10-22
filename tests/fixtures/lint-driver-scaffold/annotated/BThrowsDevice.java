// The header continues on a throws line and the brace sits on its own line.
package com.example.annotated;

public class BThrowsDevice extends BNDevice
{
  @Override
  public void doPing()
      throws Exception
  {
    pingOk();
  }
}
