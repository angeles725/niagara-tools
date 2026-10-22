// The device is a nested class: its methods sit one class level deeper than a top-level class's.
package com.example.annotated;

public class BOuter
{
  public static class BInnerDevice extends BNDevice
  {
    public void doPing()
    {
      pingOk();
    }
  }
}
